// Package composite serves one logical store from two: a primary that takes
// every write and holds everything from Cutover on, and a read-only legacy
// store that holds what came before.
//
// It exists for the period in which an installation has moved to ClickHouse
// but its history still sits in OpenSearch. The split is by time, so a record
// is in exactly one of them and the two never need reconciling; what the
// composite does is route a query to whichever holds the range asked about, and
// merge when the range spans Cutover.
//
// The invariant it relies on: the primary holds nothing older than Cutover and
// the legacy holds nothing newer. Cutover must be the instant the legacy
// stopped receiving writes.
package composite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

const defaultTimeField = "@timestamp"

type Config struct {
	// Primary takes every write and is authoritative for records from Cutover.
	Primary store.Store

	// Legacy is only ever read. Nothing here writes to it.
	Legacy store.Store

	// Cutover is the instant the legacy stopped receiving writes.
	Cutover time.Time

	// LegacyTenant is the one tenant the legacy data belongs to. A legacy store
	// built for a single-tenant installation has no tenant field, so it cannot
	// tell tenants apart: only this tenant (or store.AllTenants) may read it,
	// and any other tenant sees the primary alone.
	LegacyTenant string

	// LegacyDatasets lists the datasets the legacy store serves. Anything else
	// is answered from the primary only, so an unmapped dataset is not mistaken
	// for an empty one.
	LegacyDatasets []store.Dataset

	// TimeField is the field both stores order by. Defaults to "@timestamp".
	TimeField string

	// TolerateLegacy lets a failing legacy store degrade a read to the
	// primary's answer instead of failing it. A partial answer can look like
	// "nothing happened", so it is off unless the caller says otherwise, and
	// OnLegacyError is how it finds out.
	TolerateLegacy bool
	OnLegacyError  func(op string, err error)

	// LegacyField maps a field name as the composite's callers write it to the
	// name the legacy store knows it by, and LegacyFieldBack does the reverse.
	// LegacyDocument rewrites a legacy record into the shape the primary's
	// records have. All three are optional.
	LegacyField     func(string) string
	LegacyFieldBack func(string) string
	LegacyDocument  func(json.RawMessage) json.RawMessage

	// Now is the clock, replaceable in tests.
	Now func() time.Time
}

type Store struct {
	cfg      Config
	datasets map[store.Dataset]bool
}

var _ store.Store = (*Store)(nil)

func New(cfg Config) (*Store, error) {
	switch {
	case cfg.Primary == nil:
		return nil, errors.New("composite: Config.Primary is required")
	case cfg.Legacy == nil:
		return nil, errors.New("composite: Config.Legacy is required")
	case cfg.Cutover.IsZero():
		return nil, errors.New("composite: Config.Cutover is required")
	case cfg.LegacyTenant == "":
		return nil, errors.New("composite: Config.LegacyTenant is required, so that no other tenant can read the legacy data")
	case cfg.LegacyTenant == store.AllTenants:
		return nil, errors.New("composite: Config.LegacyTenant must name one tenant")
	}
	if cfg.TimeField == "" {
		cfg.TimeField = defaultTimeField
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	ds := make(map[store.Dataset]bool, len(cfg.LegacyDatasets))
	for _, d := range cfg.LegacyDatasets {
		ds[d] = true
	}
	return &Store{cfg: cfg, datasets: ds}, nil
}

// plan says which stores a scope reaches and with what scope each is asked.
type plan struct {
	primary *store.Scope
	legacy  *store.Scope
}

func (c *Store) eligibleForLegacy(s store.Scope) bool {
	return c.datasets[s.Dataset] && (s.Tenant == c.cfg.LegacyTenant || s.Tenant == store.AllTenants)
}

func (c *Store) plan(s store.Scope) plan {
	var p plan

	if s.To.IsZero() || !s.To.Before(c.cfg.Cutover) {
		ps := s
		p.primary = &ps
	}

	if c.eligibleForLegacy(s) && (s.From.IsZero() || s.From.Before(c.cfg.Cutover)) {
		ls := s
		// The legacy has no tenant field to scope by, which is why only its own
		// tenant gets here.
		ls.Tenant = store.AllTenants
		if ls.To.IsZero() || ls.To.After(c.cfg.Cutover) {
			ls.To = c.cfg.Cutover
		}
		p.legacy = &ls
	}
	return p
}

func (p plan) both() bool { return p.primary != nil && p.legacy != nil }

func (c *Store) legacyFilters(filters []store.Filter) []store.Filter {
	if c.cfg.LegacyField == nil || len(filters) == 0 {
		return filters
	}
	out := make([]store.Filter, len(filters))
	for i, f := range filters {
		if f.Field != "" {
			f.Field = c.cfg.LegacyField(f.Field)
		}
		out[i] = f
	}
	return out
}

func (c *Store) legacyName(field string) string {
	if c.cfg.LegacyField == nil || field == "" {
		return field
	}
	return c.cfg.LegacyField(field)
}

func (c *Store) legacyNames(fields []string) []string {
	if c.cfg.LegacyField == nil {
		return fields
	}
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = c.cfg.LegacyField(f)
	}
	return out
}

func (c *Store) legacyDoc(d json.RawMessage) json.RawMessage {
	if c.cfg.LegacyDocument == nil {
		return d
	}
	return c.cfg.LegacyDocument(d)
}

func (c *Store) legacyDocs(docs []json.RawMessage) []json.RawMessage {
	if c.cfg.LegacyDocument == nil {
		return docs
	}
	out := make([]json.RawMessage, len(docs))
	for i, d := range docs {
		out[i] = c.cfg.LegacyDocument(d)
	}
	return out
}

// legacyFailed applies the tolerance policy. It reports whether the failure
// should be swallowed.
func (c *Store) legacyFailed(op string, err error) bool {
	if !c.cfg.TolerateLegacy {
		return false
	}
	if c.cfg.OnLegacyError != nil {
		c.cfg.OnLegacyError(op, err)
	}
	return true
}

// fanOut runs the primary and legacy halves of a plan together. A legacy
// failure is an error unless tolerated, in which case that half is empty.
func fanOut[T any](c *Store, p plan, op string, run func(st store.Store, s store.Scope, legacy bool) (T, error)) (T, T, error) {
	var (
		prim, leg  T
		pErr, lErr error
		wg         sync.WaitGroup
	)
	if p.primary != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			prim, pErr = run(c.cfg.Primary, *p.primary, false)
		}()
	}
	if p.legacy != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			leg, lErr = run(c.cfg.Legacy, *p.legacy, true)
		}()
	}
	wg.Wait()

	if pErr != nil {
		var zero T
		return zero, zero, pErr
	}
	if lErr != nil {
		if !c.legacyFailed(op, lErr) {
			var zero T
			return zero, zero, fmt.Errorf("composite: legacy %s: %w", op, lErr)
		}
		var zero T
		leg = zero
	}
	return prim, leg, nil
}

// ---- Writer -----------------------------------------------------------------

// Every write goes to the primary. The legacy is read-only, so a record there
// is never changed: UpdateWhere reaches only primary records, and an alert that
// lives in the legacy cannot be updated.

func (c *Store) Insert(ctx context.Context, s store.Scope, id string, doc any) error {
	return c.cfg.Primary.Insert(ctx, s, id, doc)
}

func (c *Store) BulkWriter(dataset store.Dataset) (store.BulkWriter, error) {
	return c.cfg.Primary.BulkWriter(dataset)
}

func (c *Store) UpdateWhere(ctx context.Context, s store.Scope, filters []store.Filter, patch map[string]any) (int64, error) {
	return c.cfg.Primary.UpdateWhere(ctx, s, filters, patch)
}

func (c *Store) Flush(ctx context.Context, dataset store.Dataset) error {
	return c.cfg.Primary.Flush(ctx, dataset)
}

// ---- Admin ------------------------------------------------------------------

func (c *Store) Retention(ctx context.Context, dataset store.Dataset) (store.Retention, error) {
	return c.cfg.Primary.Retention(ctx, dataset)
}

func (c *Store) SetRetention(ctx context.Context, dataset store.Dataset, r store.Retention) error {
	return c.cfg.Primary.SetRetention(ctx, dataset, r)
}

// Drop removes records from the primary only; the legacy is read-only and its
// own retention ends it.
func (c *Store) Drop(ctx context.Context, s store.Scope) (int64, error) {
	return c.cfg.Primary.Drop(ctx, s)
}

// Usage lists both stores' datasets, the legacy's after the primary's.
func (c *Store) Usage(ctx context.Context) ([]store.DatasetUsage, error) {
	prim, err := c.cfg.Primary.Usage(ctx)
	if err != nil {
		return nil, err
	}
	leg, err := c.cfg.Legacy.Usage(ctx)
	if err != nil {
		if !c.legacyFailed("Usage", err) {
			return nil, fmt.Errorf("composite: legacy Usage: %w", err)
		}
		return prim, nil
	}
	return append(prim, leg...), nil
}

// Health is the primary's, degraded further when the legacy is not well. A
// legacy that is down must not make the installation look down.
func (c *Store) Health(ctx context.Context) (store.Health, error) {
	h, err := c.cfg.Primary.Health(ctx)
	if err != nil {
		return h, err
	}

	lh, lerr := c.cfg.Legacy.Health(ctx)
	switch {
	case lerr != nil:
		lh = store.Health{Status: store.HealthUnavailable, Message: lerr.Error()}
	case lh.Status == store.HealthOK || lh.Status == "":
		return h, nil
	}

	if h.Status == store.HealthOK {
		h.Status = store.HealthDegraded
	}
	msg := "legacy store: " + string(lh.Status)
	if lh.Message != "" {
		msg += ": " + lh.Message
	}
	if h.Message != "" {
		msg = h.Message + "; " + msg
	}
	h.Message = msg
	if lh.DiskUsedPct > h.DiskUsedPct {
		h.DiskUsedPct = lh.DiskUsedPct
	}
	return h, nil
}
