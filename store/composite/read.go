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
	prim, sec, err := fanOut(c, p, "Count", func(st store.Store, sc store.Scope, secondary bool) (int64, error) {
		if secondary {
			return st.Count(ctx, sc, c.secondaryFilters(filters))
		}
		return st.Count(ctx, sc, filters)
	})
	return prim + sec, err
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
	if p.secondary == nil {
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
	if !c.eligibleForSecondary(s) {
		return nil, err
	}

	ss := s
	ss.Tenant = store.AllTenants
	doc, serr := c.cfg.Secondary.FetchByID(ctx, ss, id)
	switch {
	case serr == nil:
		return c.secondaryDoc(doc), nil
	case errors.Is(serr, store.ErrNotFound):
		return nil, serr
	case c.secondaryFailed("FetchByID", serr):
		return nil, store.ErrNotFound
	default:
		return nil, fmt.Errorf("composite: secondary FetchByID: %w", serr)
	}
}

// FetchN returns the newest n records: the primary's first, and the secondary's
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
	if p.secondary == nil || len(out) >= n {
		return out, nil
	}

	rows, _, err := c.cfg.Secondary.FetchPage(ctx, *p.secondary, c.secondaryFilters(filters), store.Page{
		Limit: n - len(out), SortBy: c.secondaryName(c.cfg.TimeField), Order: store.Desc,
	})
	if err != nil {
		if c.secondaryFailed("FetchN", err) {
			return out, nil
		}
		return nil, fmt.Errorf("composite: secondary FetchN: %w", err)
	}
	return append(out, c.secondaryDocs(rows)...), nil
}

// FetchPage pages across both stores. Sorted by time, which is the default, the
// two are consecutive runs — the primary's records are all newer than the
// secondary's — so the page is read from one or both without sorting anything. Any
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
	case p.primary == nil && p.secondary == nil:
		return nil, 0, nil
	case p.secondary == nil:
		return c.cfg.Primary.FetchPage(ctx, *p.primary, filters, page)
	case p.primary == nil:
		rows, total, err := c.secondaryPage(ctx, *p.secondary, filters, page)
		if err != nil && c.secondaryFailed("FetchPage", err) {
			return nil, 0, nil
		}
		return rows, total, err
	}

	if page.SortBy == c.cfg.TimeField {
		return c.timePage(ctx, p, filters, page)
	}
	return c.mergedPage(ctx, p, filters, page)
}

func (c *Store) secondaryPage(ctx context.Context, s store.Scope, filters []store.Filter, page store.Page) ([]json.RawMessage, int64, error) {
	page.SortBy = c.secondaryName(page.SortBy)
	rows, total, err := c.cfg.Secondary.FetchPage(ctx, s, c.secondaryFilters(filters), page)
	if err != nil {
		return nil, 0, err
	}
	return c.secondaryDocs(rows), total, nil
}

// timePage reads a page out of two stores that split the time axis between
// them. The run that comes first in the requested order is read first; the
// second is read only for what the first could not supply.
func (c *Store) timePage(ctx context.Context, p plan, filters []store.Filter, page store.Page) ([]json.RawMessage, int64, error) {
	fetch := func(secondary bool, pg store.Page) ([]json.RawMessage, int64, error) {
		if secondary {
			return c.secondaryPage(ctx, *p.secondary, filters, pg)
		}
		return c.cfg.Primary.FetchPage(ctx, *p.primary, filters, pg)
	}

	// Newest first reads the primary, which is the newer, first.
	firstSecondary := page.Order == store.Asc

	rows1, t1, err := fetch(firstSecondary, page)
	if err != nil {
		if firstSecondary && c.secondaryFailed("FetchPage", err) {
			return c.cfg.Primary.FetchPage(ctx, *p.primary, filters, page)
		}
		return nil, 0, c.wrap(firstSecondary, err)
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
		rows2, t2, err = fetch(!firstSecondary, next)
	} else {
		// The page is full from the first run; the second is only counted.
		t2, err = c.countOne(ctx, p, !firstSecondary, filters)
	}
	if err != nil {
		if !firstSecondary && c.secondaryFailed("FetchPage", err) {
			return rows1, t1, nil
		}
		return nil, 0, c.wrap(!firstSecondary, err)
	}
	return append(rows1, rows2...), t1 + t2, nil
}

func (c *Store) countOne(ctx context.Context, p plan, secondary bool, filters []store.Filter) (int64, error) {
	if secondary {
		return c.cfg.Secondary.Count(ctx, *p.secondary, c.secondaryFilters(filters))
	}
	return c.cfg.Primary.Count(ctx, *p.primary, filters)
}

func (c *Store) wrap(secondary bool, err error) error {
	if secondary {
		return fmt.Errorf("composite: secondary FetchPage: %w", err)
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
	prim, sec, err := fanOut(c, p, "FetchPage", func(st store.Store, sc store.Scope, secondary bool) (result, error) {
		var r result
		var err error
		if secondary {
			r.rows, r.total, err = c.secondaryPage(ctx, sc, filters, head)
		} else {
			r.rows, r.total, err = st.FetchPage(ctx, sc, filters, head)
		}
		return r, err
	})
	if err != nil {
		return nil, 0, err
	}

	merged := mergeSorted(prim.rows, sec.rows, page.SortBy, page.Order)
	if page.Offset >= len(merged) {
		return nil, prim.total + sec.total, nil
	}
	end := page.Offset + page.Limit
	if end > len(merged) {
		end = len(merged)
	}
	return merged[page.Offset:end], prim.total + sec.total, nil
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
	prim, sec, err := fanOut(c, p, "TopValues", func(st store.Store, sc store.Scope, secondary bool) ([]store.Bucket, error) {
		if secondary {
			return st.TopValues(ctx, sc, c.secondaryName(field), c.secondaryFilters(filters), ask)
		}
		return st.TopValues(ctx, sc, field, filters, ask)
	})
	if err != nil {
		return nil, err
	}
	return mergeBuckets(prim, sec, n), nil
}

// Timeline sums buckets with the same start. The bucket that holds Cutover is
// in both stores and is added up; both are expected to align their buckets the
// same way for a given interval.
func (c *Store) Timeline(ctx context.Context, s store.Scope, filters []store.Filter, interval store.Interval) ([]store.Point, error) {
	if err := c.check(s); err != nil {
		return nil, err
	}
	prim, sec, err := fanOut(c, c.plan(s), "Timeline", func(st store.Store, sc store.Scope, secondary bool) ([]store.Point, error) {
		if secondary {
			return st.Timeline(ctx, sc, c.secondaryFilters(filters), interval)
		}
		return st.Timeline(ctx, sc, filters, interval)
	})
	if err != nil {
		return nil, err
	}
	return mergePoints(prim, sec), nil
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
	prim, sec, err := fanOut(c, p, "TimelineByField", func(st store.Store, sc store.Scope, secondary bool) ([]store.Series, error) {
		if secondary {
			return st.TimelineByField(ctx, sc, c.secondaryName(field), c.secondaryFilters(filters), interval, ask)
		}
		return st.TimelineByField(ctx, sc, field, filters, interval, ask)
	})
	if err != nil {
		return nil, err
	}
	return mergeSeries(prim, sec, n), nil
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
	prim, sec, err := fanOut(c, p, "GroupBy", func(st store.Store, sc store.Scope, secondary bool) ([]store.Group, error) {
		if secondary {
			gs, err := st.GroupBy(ctx, sc, c.secondaryNames(fields), c.secondaryFilters(filters), ask)
			return c.fromSecondaryGroups(gs, fields), err
		}
		return st.GroupBy(ctx, sc, fields, filters, ask)
	})
	if err != nil {
		return nil, err
	}
	if !p.both() {
		return append(prim, sec...), nil
	}
	return mergeGroups(prim, sec, opts), nil
}

// fromSecondaryGroups puts a secondary answer back in the caller's field names and
// record shape. Each level is named by the first of the fields that remain.
func (c *Store) fromSecondaryGroups(gs []store.Group, fields []string) []store.Group {
	for i := range gs {
		g := &gs[i]
		if len(fields) > 0 && g.Field != "" {
			g.Field = fields[0]
		}
		g.Hits = c.secondaryDocs(g.Hits)
		if len(g.Children) > 0 && len(fields) > 1 {
			g.Children = c.fromSecondaryGroups(g.Children, fields[1:])
		}
	}
	return gs
}

func (c *Store) DescribeFields(ctx context.Context, s store.Scope) ([]store.Field, error) {
	if err := c.check(s); err != nil {
		return nil, err
	}
	// Fields are a property of the dataset, not of a time range, so both stores
	// describe it whenever the secondary serves the dataset at all.
	p := plan{}
	ps := s
	p.primary = &ps
	if c.eligibleForSecondary(s) {
		ss := s
		ss.Tenant = store.AllTenants
		p.secondary = &ss
	}

	prim, sec, err := fanOut(c, p, "DescribeFields", func(st store.Store, sc store.Scope, secondary bool) ([]store.Field, error) {
		fs, err := st.DescribeFields(ctx, sc)
		if secondary && c.cfg.SecondaryFieldBack != nil {
			for i := range fs {
				fs[i].Name = c.cfg.SecondaryFieldBack(fs[i].Name)
			}
		}
		return fs, err
	})
	if err != nil {
		return nil, err
	}
	return mergeFields(prim, sec), nil
}
