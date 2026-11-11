// Package composite serves one logical store from two: a primary that takes
// every write and holds everything from Cutover on, and a read-only secondary
// store that holds what came before.
//
// It serves an installation whose older data sits in a different store than its
// new data, for as long as that older data is kept. The split is by time, so a
// record is in exactly one of them and the two never need reconciling; what the
// composite does is route a query to whichever holds the range asked about, and
// merge when the range spans Cutover.
//
// The invariant it relies on: the primary holds nothing older than Cutover and
// the secondary holds nothing newer. Cutover must be the instant the secondary
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

	// Secondary is only ever read. Nothing here writes to it.
	Secondary store.Store

	// Cutover is the instant the secondary stopped receiving writes.
	Cutover time.Time

	// SecondaryExpires is when the secondary store is switched off. From then on the
	// composite never reaches it, so a query that would have crossed Cutover is
	// answered from the primary alone instead of failing against a store that
	// is gone. The zero value means it never expires. Set it to the cutover plus
	// the secondary's retention.
	SecondaryExpires time.Time

	// SecondaryTenant is the one tenant the secondary data belongs to. A secondary store
	// built for a single-tenant installation has no tenant field, so it cannot
	// tell tenants apart: only this tenant (or store.AllTenants) may read it,
	// and any other tenant sees the primary alone.
	SecondaryTenant string

	// SecondaryDatasets lists the datasets the secondary store serves. Anything else
	// is answered from the primary only, so an unmapped dataset is not mistaken
	// for an empty one.
	SecondaryDatasets []store.Dataset

	// TimeField is the field both stores order by. Defaults to "@timestamp".
	TimeField string

	// TolerateSecondary lets a failing secondary store degrade a read to the
	// primary's answer instead of failing it. A partial answer can look like
	// "nothing happened", so it is off unless the caller says otherwise, and
	// OnSecondaryError is how it finds out.
	TolerateSecondary bool
	OnSecondaryError  func(op string, err error)

	// SecondaryField maps a field name as the composite's callers write it to the
	// name the secondary store knows it by, and SecondaryFieldBack does the reverse.
	// SecondaryDocument rewrites a secondary record into the shape the primary's
	// records have. All three are optional.
	SecondaryField     func(string) string
	SecondaryFieldBack func(string) string
	SecondaryDocument  func(json.RawMessage) json.RawMessage

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
	case cfg.Secondary == nil:
		return nil, errors.New("composite: Config.Secondary is required")
	case cfg.Cutover.IsZero():
		return nil, errors.New("composite: Config.Cutover is required")
	case cfg.SecondaryTenant == "":
		return nil, errors.New("composite: Config.SecondaryTenant is required, so that no other tenant can read the secondary data")
	case cfg.SecondaryTenant == store.AllTenants:
		return nil, errors.New("composite: Config.SecondaryTenant must name one tenant")
	case !cfg.SecondaryExpires.IsZero() && !cfg.SecondaryExpires.After(cfg.Cutover):
		return nil, errors.New("composite: Config.SecondaryExpires must be after Config.Cutover")
	}
	if cfg.TimeField == "" {
		cfg.TimeField = defaultTimeField
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	ds := make(map[store.Dataset]bool, len(cfg.SecondaryDatasets))
	for _, d := range cfg.SecondaryDatasets {
		ds[d] = true
	}
	return &Store{cfg: cfg, datasets: ds}, nil
}

// plan says which stores a scope reaches and with what scope each is asked.
type plan struct {
	primary   *store.Scope
	secondary *store.Scope
}

// SecondaryActive reports whether the secondary store can still be reached: it is
// until SecondaryExpires, if one is set.
func (c *Store) SecondaryActive() bool {
	return c.cfg.SecondaryExpires.IsZero() || c.cfg.Now().Before(c.cfg.SecondaryExpires)
}

func (c *Store) eligibleForSecondary(s store.Scope) bool {
	return c.SecondaryActive() && c.datasets[s.Dataset] && (s.Tenant == c.cfg.SecondaryTenant || s.Tenant == store.AllTenants)
}

func (c *Store) plan(s store.Scope) plan {
	var p plan

	if s.To.IsZero() || !s.To.Before(c.cfg.Cutover) {
		ps := s
		p.primary = &ps
	}

	if c.eligibleForSecondary(s) && (s.From.IsZero() || s.From.Before(c.cfg.Cutover)) {
		ss := s
		// The secondary has no tenant field to scope by, which is why only its own
		// tenant gets here.
		ss.Tenant = store.AllTenants
		if ss.To.IsZero() || ss.To.After(c.cfg.Cutover) {
			ss.To = c.cfg.Cutover
		}
		p.secondary = &ss
	}
	return p
}

func (p plan) both() bool { return p.primary != nil && p.secondary != nil }

func (c *Store) secondaryFilters(filters []store.Filter) []store.Filter {
	if c.cfg.SecondaryField == nil || len(filters) == 0 {
		return filters
	}
	out := make([]store.Filter, len(filters))
	for i, f := range filters {
		if f.Field != "" {
			f.Field = c.cfg.SecondaryField(f.Field)
		}
		out[i] = f
	}
	return out
}

func (c *Store) secondaryName(field string) string {
	if c.cfg.SecondaryField == nil || field == "" {
		return field
	}
	return c.cfg.SecondaryField(field)
}

func (c *Store) secondaryNames(fields []string) []string {
	if c.cfg.SecondaryField == nil {
		return fields
	}
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = c.cfg.SecondaryField(f)
	}
	return out
}

func (c *Store) secondaryDoc(d json.RawMessage) json.RawMessage {
	if c.cfg.SecondaryDocument == nil {
		return d
	}
	return c.cfg.SecondaryDocument(d)
}

func (c *Store) secondaryDocs(docs []json.RawMessage) []json.RawMessage {
	if c.cfg.SecondaryDocument == nil {
		return docs
	}
	out := make([]json.RawMessage, len(docs))
	for i, d := range docs {
		out[i] = c.cfg.SecondaryDocument(d)
	}
	return out
}

// secondaryFailed applies the tolerance policy. It reports whether the failure
// should be swallowed.
func (c *Store) secondaryFailed(op string, err error) bool {
	if !c.cfg.TolerateSecondary {
		return false
	}
	if c.cfg.OnSecondaryError != nil {
		c.cfg.OnSecondaryError(op, err)
	}
	return true
}

// fanOut runs the primary and secondary halves of a plan together. A secondary
// failure is an error unless tolerated, in which case that half is empty.
func fanOut[T any](c *Store, p plan, op string, run func(st store.Store, s store.Scope, secondary bool) (T, error)) (T, T, error) {
	var (
		prim, sec  T
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
	if p.secondary != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sec, lErr = run(c.cfg.Secondary, *p.secondary, true)
		}()
	}
	wg.Wait()

	if pErr != nil {
		var zero T
		return zero, zero, pErr
	}
	if lErr != nil {
		if !c.secondaryFailed(op, lErr) {
			var zero T
			return zero, zero, fmt.Errorf("composite: secondary %s: %w", op, lErr)
		}
		var zero T
		sec = zero
	}
	return prim, sec, nil
}

// ---- Writer -----------------------------------------------------------------

// Every write goes to the primary. The secondary is read-only, so a record there
// is never changed: UpdateWhere reaches only primary records, and an alert that
// lives in the secondary cannot be updated.

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

// Drop removes records from the primary only; the secondary is read-only and its
// own retention ends it.
func (c *Store) Drop(ctx context.Context, s store.Scope) (int64, error) {
	return c.cfg.Primary.Drop(ctx, s)
}

// Usage lists both stores' datasets, the secondary's after the primary's.
func (c *Store) Usage(ctx context.Context) ([]store.DatasetUsage, error) {
	prim, err := c.cfg.Primary.Usage(ctx)
	if err != nil {
		return nil, err
	}
	if !c.SecondaryActive() {
		return prim, nil
	}
	sec, err := c.cfg.Secondary.Usage(ctx)
	if err != nil {
		if !c.secondaryFailed("Usage", err) {
			return nil, fmt.Errorf("composite: secondary Usage: %w", err)
		}
		return prim, nil
	}
	return append(prim, sec...), nil
}

// Health is the primary's, degraded further when the secondary is not well. A
// secondary that is down must not make the installation look down.
func (c *Store) Health(ctx context.Context) (store.Health, error) {
	h, err := c.cfg.Primary.Health(ctx)
	if err != nil || !c.SecondaryActive() {
		return h, err
	}

	sh, serr := c.cfg.Secondary.Health(ctx)
	switch {
	case serr != nil:
		sh = store.Health{Status: store.HealthUnavailable, Message: serr.Error()}
	case sh.Status == store.HealthOK || sh.Status == "":
		return h, nil
	}

	if h.Status == store.HealthOK {
		h.Status = store.HealthDegraded
	}
	msg := "secondary store: " + string(sh.Status)
	if sh.Message != "" {
		msg += ": " + sh.Message
	}
	if h.Message != "" {
		msg = h.Message + "; " + msg
	}
	h.Message = msg
	if sh.DiskUsedPct > h.DiskUsedPct {
		h.DiskUsedPct = sh.DiskUsedPct
	}
	return h, nil
}
