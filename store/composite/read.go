package composite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

// maxMergeWindow bounds how deep a page sorted by something other than time can
// reach. Merging two stores by an arbitrary field means reading offset+limit
// rows from each, so an unbounded offset is an unbounded read.
const maxMergeWindow = 10000

const defaultPageSize = 100

func (c *Store) check(s store.Scope) error {
	if s.Tenant == "" {
		return store.ErrNoTenant
	}
	return nil
}

func (c *Store) Count(ctx context.Context, s store.Scope, filters []store.Filter) (int64, error) {
	if err := c.check(s); err != nil {
		return 0, err
	}
	p := c.plan(s)
	prim, leg, err := fanOut(c, p, "Count", func(st store.Store, sc store.Scope, legacy bool) (int64, error) {
		if legacy {
			return st.Count(ctx, sc, c.legacyFilters(filters))
		}
		return st.Count(ctx, sc, filters)
	})
	return prim + leg, err
}

func (c *Store) CountInWindow(ctx context.Context, s store.Scope, filters []store.Filter, window time.Duration) (int64, error) {
	if err := c.check(s); err != nil {
		return 0, err
	}

	end := s.To
	if end.IsZero() {
		end = c.cfg.Now()
	}
	w := s
	w.From = end.Add(-window)
	w.To = end

	p := c.plan(w)
	if p.legacy == nil {
		// Entirely within the primary: let it answer in its own terms.
		return c.cfg.Primary.CountInWindow(ctx, s, filters, window)
	}
	return c.Count(ctx, w, filters)
}

func (c *Store) FetchByID(ctx context.Context, s store.Scope, id string) (json.RawMessage, error) {
	if err := c.check(s); err != nil {
		return nil, err
	}
	doc, err := c.cfg.Primary.FetchByID(ctx, s, id)
	if err == nil || !errors.Is(err, store.ErrNotFound) {
		return doc, err
	}
	if !c.eligibleForLegacy(s) {
		return nil, err
	}

	ls := s
	ls.Tenant = store.AllTenants
	doc, lerr := c.cfg.Legacy.FetchByID(ctx, ls, id)
	switch {
	case lerr == nil:
		return c.legacyDoc(doc), nil
	case errors.Is(lerr, store.ErrNotFound):
		return nil, lerr
	case c.legacyFailed("FetchByID", lerr):
		return nil, store.ErrNotFound
	default:
		return nil, fmt.Errorf("composite: legacy FetchByID: %w", lerr)
	}
}

// FetchN returns the newest n records: the primary's first, and the legacy's
// only for what the primary could not fill.
func (c *Store) FetchN(ctx context.Context, s store.Scope, filters []store.Filter, n int) ([]json.RawMessage, error) {
	if err := c.check(s); err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, nil
	}

	p := c.plan(s)
	var out []json.RawMessage
	if p.primary != nil {
		rows, err := c.cfg.Primary.FetchN(ctx, *p.primary, filters, n)
		if err != nil {
			return nil, err
		}
		out = rows
	}
	if p.legacy == nil || len(out) >= n {
		return out, nil
	}

	rows, _, err := c.cfg.Legacy.FetchPage(ctx, *p.legacy, c.legacyFilters(filters), store.Page{
		Limit: n - len(out), SortBy: c.legacyName(c.cfg.TimeField), Order: store.Desc,
	})
	if err != nil {
		if c.legacyFailed("FetchN", err) {
			return out, nil
		}
		return nil, fmt.Errorf("composite: legacy FetchN: %w", err)
	}
	return append(out, c.legacyDocs(rows)...), nil
}

// FetchPage pages across both stores. Sorted by time, which is the default, the
// two are consecutive runs — the primary's records are all newer than the
// legacy's — so the page is read from one or both without sorting anything. Any
// other order has to read offset+limit rows from each and merge them.
func (c *Store) FetchPage(ctx context.Context, s store.Scope, filters []store.Filter, page store.Page) ([]json.RawMessage, int64, error) {
	if err := c.check(s); err != nil {
		return nil, 0, err
	}
	if page.Limit <= 0 {
		page.Limit = defaultPageSize
	}
	if page.Offset < 0 {
		page.Offset = 0
	}
	if page.SortBy == "" {
		page.SortBy = c.cfg.TimeField
	}
	if page.Order == "" {
		page.Order = store.Desc
	}

	p := c.plan(s)
	switch {
	case p.primary == nil && p.legacy == nil:
		return nil, 0, nil
	case p.legacy == nil:
		return c.cfg.Primary.FetchPage(ctx, *p.primary, filters, page)
	case p.primary == nil:
		rows, total, err := c.legacyPage(ctx, *p.legacy, filters, page)
		if err != nil && c.legacyFailed("FetchPage", err) {
			return nil, 0, nil
		}
		return rows, total, err
	}

	if page.SortBy == c.cfg.TimeField {
		return c.timePage(ctx, p, filters, page)
	}
	return c.mergedPage(ctx, p, filters, page)
}

func (c *Store) legacyPage(ctx context.Context, s store.Scope, filters []store.Filter, page store.Page) ([]json.RawMessage, int64, error) {
	page.SortBy = c.legacyName(page.SortBy)
	rows, total, err := c.cfg.Legacy.FetchPage(ctx, s, c.legacyFilters(filters), page)
	if err != nil {
		return nil, 0, err
	}
	return c.legacyDocs(rows), total, nil
}

// timePage reads a page out of two stores that split the time axis between
// them. The run that comes first in the requested order is read first; the
// second is read only for what the first could not supply.
func (c *Store) timePage(ctx context.Context, p plan, filters []store.Filter, page store.Page) ([]json.RawMessage, int64, error) {
	fetch := func(legacy bool, pg store.Page) ([]json.RawMessage, int64, error) {
		if legacy {
			return c.legacyPage(ctx, *p.legacy, filters, pg)
		}
		return c.cfg.Primary.FetchPage(ctx, *p.primary, filters, pg)
	}

	// Newest first reads the primary, which is the newer, first.
	firstLegacy := page.Order == store.Asc

	rows1, t1, err := fetch(firstLegacy, page)
	if err != nil {
		if firstLegacy && c.legacyFailed("FetchPage", err) {
			return c.cfg.Primary.FetchPage(ctx, *p.primary, filters, page)
		}
		return nil, 0, c.wrap(firstLegacy, err)
	}

	var rows2 []json.RawMessage
	var t2 int64
	remaining := page.Limit - len(rows1)
	if remaining > 0 {
		next := page
		next.Limit = remaining
		next.Offset = page.Offset - int(t1)
		if next.Offset < 0 {
			next.Offset = 0
		}
		rows2, t2, err = fetch(!firstLegacy, next)
	} else {
		// The page is full from the first run; the second is only counted.
		t2, err = c.countOne(ctx, p, !firstLegacy, filters)
	}
	if err != nil {
		if !firstLegacy && c.legacyFailed("FetchPage", err) {
			return rows1, t1, nil
		}
		return nil, 0, c.wrap(!firstLegacy, err)
	}
	return append(rows1, rows2...), t1 + t2, nil
}

func (c *Store) countOne(ctx context.Context, p plan, legacy bool, filters []store.Filter) (int64, error) {
	if legacy {
		return c.cfg.Legacy.Count(ctx, *p.legacy, c.legacyFilters(filters))
	}
	return c.cfg.Primary.Count(ctx, *p.primary, filters)
}

func (c *Store) wrap(legacy bool, err error) error {
	if legacy {
		return fmt.Errorf("composite: legacy FetchPage: %w", err)
	}
	return err
}

// mergedPage serves a page sorted by a field the two stores do not split on, by
// reading the first offset+limit rows of each and merging them.
func (c *Store) mergedPage(ctx context.Context, p plan, filters []store.Filter, page store.Page) ([]json.RawMessage, int64, error) {
	window := page.Offset + page.Limit
	if window > maxMergeWindow {
		return nil, 0, fmt.Errorf("%w: a page sorted by %q reaches row %d, past the %d this store can merge; sort by time or narrow the query",
			store.ErrUnsupported, page.SortBy, window, maxMergeWindow)
	}

	type result struct {
		rows  []json.RawMessage
		total int64
	}
	head := store.Page{Offset: 0, Limit: window, SortBy: page.SortBy, Order: page.Order}
	prim, leg, err := fanOut(c, p, "FetchPage", func(st store.Store, sc store.Scope, legacy bool) (result, error) {
		var r result
		var err error
		if legacy {
			r.rows, r.total, err = c.legacyPage(ctx, sc, filters, head)
		} else {
			r.rows, r.total, err = st.FetchPage(ctx, sc, filters, head)
		}
		return r, err
	})
	if err != nil {
		return nil, 0, err
	}

	merged := mergeSorted(prim.rows, leg.rows, page.SortBy, page.Order)
	if page.Offset >= len(merged) {
		return nil, prim.total + leg.total, nil
	}
	end := page.Offset + page.Limit
	if end > len(merged) {
		end = len(merged)
	}
	return merged[page.Offset:end], prim.total + leg.total, nil
}

// TopValues is exact when one store answers. When both do, each is asked for
// more than n values and the counts are summed, so a value that ranks high only
// by adding the two together can still be missed or undercounted.
func (c *Store) TopValues(ctx context.Context, s store.Scope, field string, filters []store.Filter, n int) ([]store.Bucket, error) {
	if err := c.check(s); err != nil {
		return nil, err
	}
	p := c.plan(s)
	ask := n
	if p.both() {
		ask = widen(n)
	}
	prim, leg, err := fanOut(c, p, "TopValues", func(st store.Store, sc store.Scope, legacy bool) ([]store.Bucket, error) {
		if legacy {
			return st.TopValues(ctx, sc, c.legacyName(field), c.legacyFilters(filters), ask)
		}
		return st.TopValues(ctx, sc, field, filters, ask)
	})
	if err != nil {
		return nil, err
	}
	return mergeBuckets(prim, leg, n), nil
}

// Timeline sums buckets with the same start. The bucket that holds Cutover is
// in both stores and is added up; both are expected to align their buckets the
// same way for a given interval.
func (c *Store) Timeline(ctx context.Context, s store.Scope, filters []store.Filter, interval store.Interval) ([]store.Point, error) {
	if err := c.check(s); err != nil {
		return nil, err
	}
	prim, leg, err := fanOut(c, c.plan(s), "Timeline", func(st store.Store, sc store.Scope, legacy bool) ([]store.Point, error) {
		if legacy {
			return st.Timeline(ctx, sc, c.legacyFilters(filters), interval)
		}
		return st.Timeline(ctx, sc, filters, interval)
	})
	if err != nil {
		return nil, err
	}
	return mergePoints(prim, leg), nil
}

func (c *Store) TimelineByField(ctx context.Context, s store.Scope, field string, filters []store.Filter, interval store.Interval, n int) ([]store.Series, error) {
	if err := c.check(s); err != nil {
		return nil, err
	}
	p := c.plan(s)
	ask := n
	if p.both() {
		ask = widen(n)
	}
	prim, leg, err := fanOut(c, p, "TimelineByField", func(st store.Store, sc store.Scope, legacy bool) ([]store.Series, error) {
		if legacy {
			return st.TimelineByField(ctx, sc, c.legacyName(field), c.legacyFilters(filters), interval, ask)
		}
		return st.TimelineByField(ctx, sc, field, filters, interval, ask)
	})
	if err != nil {
		return nil, err
	}
	return mergeSeries(prim, leg, n), nil
}

func (c *Store) GroupBy(ctx context.Context, s store.Scope, fields []string, filters []store.Filter, opts store.GroupOpts) ([]store.Group, error) {
	if err := c.check(s); err != nil {
		return nil, err
	}
	p := c.plan(s)
	ask := opts
	if p.both() && opts.Limit > 0 {
		ask.Limit = widen(opts.Limit)
	}
	prim, leg, err := fanOut(c, p, "GroupBy", func(st store.Store, sc store.Scope, legacy bool) ([]store.Group, error) {
		if legacy {
			gs, err := st.GroupBy(ctx, sc, c.legacyNames(fields), c.legacyFilters(filters), ask)
			return c.fromLegacyGroups(gs, fields), err
		}
		return st.GroupBy(ctx, sc, fields, filters, ask)
	})
	if err != nil {
		return nil, err
	}
	if !p.both() {
		return append(prim, leg...), nil
	}
	return mergeGroups(prim, leg, opts), nil
}

// fromLegacyGroups puts a legacy answer back in the caller's field names and
// record shape. Each level is named by the first of the fields that remain.
func (c *Store) fromLegacyGroups(gs []store.Group, fields []string) []store.Group {
	for i := range gs {
		g := &gs[i]
		if len(fields) > 0 && g.Field != "" {
			g.Field = fields[0]
		}
		g.Hits = c.legacyDocs(g.Hits)
		if len(g.Children) > 0 && len(fields) > 1 {
			g.Children = c.fromLegacyGroups(g.Children, fields[1:])
		}
	}
	return gs
}

func (c *Store) DescribeFields(ctx context.Context, s store.Scope) ([]store.Field, error) {
	if err := c.check(s); err != nil {
		return nil, err
	}
	// Fields are a property of the dataset, not of a time range, so both stores
	// describe it whenever the legacy serves the dataset at all.
	p := plan{}
	ps := s
	p.primary = &ps
	if c.eligibleForLegacy(s) {
		ls := s
		ls.Tenant = store.AllTenants
		p.legacy = &ls
	}

	prim, leg, err := fanOut(c, p, "DescribeFields", func(st store.Store, sc store.Scope, legacy bool) ([]store.Field, error) {
		fs, err := st.DescribeFields(ctx, sc)
		if legacy && c.cfg.LegacyFieldBack != nil {
			for i := range fs {
				fs[i].Name = c.cfg.LegacyFieldBack(fs[i].Name)
			}
		}
		return fs, err
	})
	if err != nil {
		return nil, err
	}
	return mergeFields(prim, leg), nil
}
