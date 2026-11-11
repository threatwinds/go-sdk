package composite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/tidwall/gjson"

	"github.com/threatwinds/go-sdk/store"
)

const (
	dsLogs   store.Dataset = "logs"
	dsOther  store.Dataset = "statistics"
	legacyTn               = "default"
)

var cutover = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

// fake is an in-memory store holding records with an @timestamp, an id, a host
// and a number n. It records every scope it was asked about.
type fake struct {
	mu     sync.Mutex
	rows   []json.RawMessage
	scopes []store.Scope
	filter [][]store.Filter
	writes []string
	err    error
	fields []store.Field
	health store.Health
	byID   map[string]json.RawMessage
}

func row(id string, at time.Time, host string, n int) json.RawMessage {
	b, _ := json.Marshal(map[string]any{"@timestamp": at.Format(time.RFC3339), "id": id, "host": host, "n": n})
	return b
}

// run makes rows count rows of host, one an hour, starting at start and going
// back (newer first), numbered from first.
func run(prefix string, start time.Time, count int, host string) []json.RawMessage {
	out := make([]json.RawMessage, count)
	for i := 0; i < count; i++ {
		out[i] = row(fmt.Sprintf("%s%d", prefix, i), start.Add(-time.Duration(i)*time.Hour), host, i)
	}
	return out
}

func (f *fake) record(s store.Scope, filters []store.Filter) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scopes = append(f.scopes, s)
	f.filter = append(f.filter, filters)
}

func (f *fake) called() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.scopes) > 0
}

func (f *fake) match(s store.Scope, filters []store.Filter) []json.RawMessage {
	var out []json.RawMessage
	for _, r := range f.rows {
		at, _ := time.Parse(time.RFC3339, gjson.GetBytes(r, "@timestamp").String())
		if !s.From.IsZero() && at.Before(s.From) || !s.To.IsZero() && at.After(s.To) {
			continue
		}
		ok := true
		for _, fl := range filters {
			if fl.Op == store.OpEq && gjson.GetBytes(r, fl.Field).String() != fmt.Sprint(fl.Value) {
				ok = false
			}
		}
		if ok {
			out = append(out, r)
		}
	}
	return out
}

func (f *fake) Count(_ context.Context, s store.Scope, fl []store.Filter) (int64, error) {
	f.record(s, fl)
	if f.err != nil {
		return 0, f.err
	}
	return int64(len(f.match(s, fl))), nil
}

func (f *fake) CountInWindow(ctx context.Context, s store.Scope, fl []store.Filter, _ time.Duration) (int64, error) {
	f.mu.Lock()
	f.writes = append(f.writes, "CountInWindow")
	f.mu.Unlock()
	return f.Count(ctx, s, fl)
}

func (f *fake) FetchByID(_ context.Context, s store.Scope, id string) (json.RawMessage, error) {
	f.record(s, nil)
	if f.err != nil {
		return nil, f.err
	}
	if d, ok := f.byID[id]; ok {
		return d, nil
	}
	return nil, store.ErrNotFound
}

func (f *fake) FetchPage(_ context.Context, s store.Scope, fl []store.Filter, p store.Page) ([]json.RawMessage, int64, error) {
	f.record(s, fl)
	if f.err != nil {
		return nil, 0, f.err
	}
	rows := append([]json.RawMessage(nil), f.match(s, fl)...)
	by := p.SortBy
	if by == "" {
		by = "@timestamp"
	}
	sort.SliceStable(rows, func(i, j int) bool {
		c := compareResults(gjson.GetBytes(rows[i], by), gjson.GetBytes(rows[j], by))
		if p.Order == store.Asc {
			return c < 0
		}
		return c > 0
	})
	total := int64(len(rows))
	if p.Offset >= len(rows) {
		return nil, total, nil
	}
	end := p.Offset + p.Limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[p.Offset:end], total, nil
}

func (f *fake) FetchN(ctx context.Context, s store.Scope, fl []store.Filter, n int) ([]json.RawMessage, error) {
	rows, _, err := f.FetchPage(ctx, s, fl, store.Page{Limit: n, Order: store.Desc})
	return rows, err
}

func (f *fake) TopValues(_ context.Context, s store.Scope, field string, fl []store.Filter, n int) ([]store.Bucket, error) {
	f.record(s, fl)
	if f.err != nil {
		return nil, f.err
	}
	counts := map[string]int64{}
	for _, r := range f.match(s, fl) {
		counts[gjson.GetBytes(r, field).String()]++
	}
	var out []store.Bucket
	for k, c := range counts {
		out = append(out, store.Bucket{Key: k, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Count > out[j].Count || out[i].Count == out[j].Count && out[i].Key < out[j].Key
	})
	if len(out) > n {
		out = out[:n]
	}
	return out, nil
}

func (f *fake) Timeline(_ context.Context, s store.Scope, fl []store.Filter, _ store.Interval) ([]store.Point, error) {
	f.record(s, fl)
	if f.err != nil {
		return nil, f.err
	}
	by := map[time.Time]int64{}
	for _, r := range f.match(s, fl) {
		at, _ := time.Parse(time.RFC3339, gjson.GetBytes(r, "@timestamp").String())
		by[at.Truncate(24*time.Hour)]++
	}
	var out []store.Point
	for at, c := range by {
		out = append(out, store.Point{At: at, Count: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out, nil
}

func (f *fake) TimelineByField(ctx context.Context, s store.Scope, field string, fl []store.Filter, iv store.Interval, n int) ([]store.Series, error) {
	f.record(s, fl)
	if f.err != nil {
		return nil, f.err
	}
	var out []store.Series
	vals, _ := f.TopValues(ctx, s, field, fl, n)
	for _, v := range vals {
		pts, _ := f.Timeline(ctx, s, append(append([]store.Filter(nil), fl...), store.Filter{Field: field, Op: store.OpEq, Value: v.Key}), iv)
		out = append(out, store.Series{Key: v.Key, Points: pts})
	}
	return out, nil
}

func (f *fake) GroupBy(_ context.Context, s store.Scope, fields []string, fl []store.Filter, o store.GroupOpts) ([]store.Group, error) {
	f.record(s, fl)
	if f.err != nil {
		return nil, f.err
	}
	counts := map[string]int64{}
	for _, r := range f.match(s, fl) {
		counts[gjson.GetBytes(r, fields[0]).String()]++
	}
	var out []store.Group
	for k, c := range counts {
		out = append(out, store.Group{Field: fields[0], Key: k, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Count > out[j].Count || out[i].Count == out[j].Count && out[i].Key < out[j].Key
	})
	if o.Limit > 0 && len(out) > o.Limit {
		out = out[:o.Limit]
	}
	return out, nil
}

func (f *fake) DescribeFields(_ context.Context, s store.Scope) ([]store.Field, error) {
	f.record(s, nil)
	return append([]store.Field(nil), f.fields...), f.err
}

func (f *fake) Insert(_ context.Context, _ store.Scope, id string, _ any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes = append(f.writes, "Insert:"+id)
	return nil
}
func (f *fake) BulkWriter(store.Dataset) (store.BulkWriter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes = append(f.writes, "BulkWriter")
	return nil, nil
}
func (f *fake) UpdateWhere(context.Context, store.Scope, []store.Filter, map[string]any) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes = append(f.writes, "UpdateWhere")
	return 1, nil
}
func (f *fake) Flush(context.Context, store.Dataset) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes = append(f.writes, "Flush")
	return nil
}
func (f *fake) Usage(context.Context) ([]store.DatasetUsage, error) {
	return []store.DatasetUsage{{Dataset: dsLogs, Documents: int64(len(f.rows))}}, f.err
}
func (f *fake) Retention(context.Context, store.Dataset) (store.Retention, error) {
	return store.Retention{Keep: time.Hour}, nil
}
func (f *fake) SetRetention(context.Context, store.Dataset, store.Retention) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes = append(f.writes, "SetRetention")
	return nil
}
func (f *fake) Health(context.Context) (store.Health, error) {
	if f.health.Status == "" {
		return store.Health{Status: store.HealthOK}, f.err
	}
	return f.health, f.err
}
func (f *fake) Drop(context.Context, store.Scope) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes = append(f.writes, "Drop")
	return 0, nil
}

// build returns a composite over a primary with 5 records after the cutover and
// a legacy with 7 before it.
func build(t *testing.T, mutate func(*Config)) (*Store, *fake, *fake) {
	t.Helper()
	prim := &fake{rows: run("p", cutover.Add(5*time.Hour), 5, "new")}
	leg := &fake{rows: run("l", cutover.Add(-time.Hour), 7, "old")}
	cfg := Config{
		Primary: prim, Legacy: leg, Cutover: cutover,
		LegacyTenant: legacyTn, LegacyDatasets: []store.Dataset{dsLogs},
		Now: func() time.Time { return cutover.Add(6 * time.Hour) },
	}
	if mutate != nil {
		mutate(&cfg)
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c, prim, leg
}

func scope() store.Scope { return store.Scope{Tenant: legacyTn, Dataset: dsLogs} }

func ids(docs []json.RawMessage) []string {
	out := make([]string, len(docs))
	for i, d := range docs {
		out[i] = gjson.GetBytes(d, "id").String()
	}
	return out
}

func eq(t *testing.T, got, want []string) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestNewValidatesItsConfig(t *testing.T) {
	ok := Config{Primary: &fake{}, Legacy: &fake{}, Cutover: cutover, LegacyTenant: "t"}
	for name, mut := range map[string]func(*Config){
		"primary":     func(c *Config) { c.Primary = nil },
		"legacy":      func(c *Config) { c.Legacy = nil },
		"cutover":     func(c *Config) { c.Cutover = time.Time{} },
		"tenant":      func(c *Config) { c.LegacyTenant = "" },
		"all tenants": func(c *Config) { c.LegacyTenant = store.AllTenants },
	} {
		cfg := ok
		mut(&cfg)
		if _, err := New(cfg); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := New(ok); err != nil {
		t.Fatal(err)
	}
}

func TestAScopeWithoutATenantIsRefused(t *testing.T) {
	c, _, _ := build(t, nil)
	if _, err := c.Count(context.Background(), store.Scope{Dataset: dsLogs}, nil); !errors.Is(err, store.ErrNoTenant) {
		t.Fatalf("want ErrNoTenant, got %v", err)
	}
}

func TestRoutingFollowsTheTimeRange(t *testing.T) {
	cases := []struct {
		name         string
		from, to     time.Time
		wantP, wantL bool
	}{
		{"open range reaches both", time.Time{}, time.Time{}, true, true},
		{"after the cutover is the primary alone", cutover.Add(time.Hour), time.Time{}, true, false},
		{"before the cutover is the legacy alone", time.Time{}, cutover.Add(-time.Hour), false, true},
		{"spanning reaches both", cutover.Add(-time.Hour), cutover.Add(time.Hour), true, true},
		{"starting at the cutover is the primary alone", cutover, cutover.Add(time.Hour), true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, prim, leg := build(t, nil)
			s := scope()
			s.From, s.To = tc.from, tc.to
			if _, err := c.Count(context.Background(), s, nil); err != nil {
				t.Fatal(err)
			}
			if prim.called() != tc.wantP || leg.called() != tc.wantL {
				t.Errorf("primary=%v legacy=%v, want %v %v", prim.called(), leg.called(), tc.wantP, tc.wantL)
			}
		})
	}
}

func TestTheLegacyIsOnlyAskedAboutRecordsBeforeTheCutover(t *testing.T) {
	c, _, leg := build(t, nil)
	if _, err := c.Count(context.Background(), scope(), nil); err != nil {
		t.Fatal(err)
	}
	got := leg.scopes[0]
	if !got.To.Equal(cutover) {
		t.Errorf("legacy To = %v, want the cutover", got.To)
	}
	if got.Tenant != store.AllTenants {
		t.Errorf("legacy tenant = %q: it has no tenant field, so it must not be scoped by one", got.Tenant)
	}
}

func TestAnotherTenantNeverReadsTheLegacy(t *testing.T) {
	c, prim, leg := build(t, nil)
	s := store.Scope{Tenant: "someone-else", Dataset: dsLogs}

	n, err := c.Count(context.Background(), s, nil)
	if err != nil {
		t.Fatal(err)
	}
	if leg.called() {
		t.Fatal("the legacy holds one tenant's data and must not be read for another")
	}
	if n != 5 || !prim.called() {
		t.Errorf("count = %d, want the primary's 5", n)
	}
}

func TestADatasetTheLegacyDoesNotServeIsAnsweredFromThePrimary(t *testing.T) {
	c, _, leg := build(t, nil)
	if _, err := c.Count(context.Background(), store.Scope{Tenant: legacyTn, Dataset: dsOther}, nil); err != nil {
		t.Fatal(err)
	}
	if leg.called() {
		t.Fatal("an unmapped dataset must not reach the legacy")
	}
}

func TestAllTenantsMayReadTheLegacy(t *testing.T) {
	c, _, leg := build(t, nil)
	if _, err := c.Count(context.Background(), store.Scope{Tenant: store.AllTenants, Dataset: dsLogs}, nil); err != nil {
		t.Fatal(err)
	}
	if !leg.called() {
		t.Fatal("an all-tenants read covers the legacy's tenant")
	}
}

func TestCountSumsBothStores(t *testing.T) {
	c, _, _ := build(t, nil)
	n, err := c.Count(context.Background(), scope(), nil)
	if err != nil || n != 12 {
		t.Fatalf("count = %d, %v; want 12", n, err)
	}
}

func TestCountInWindowIsThePrimarysOwnWhenTheWindowIsAllNew(t *testing.T) {
	c, prim, leg := build(t, nil)
	s := scope()
	s.To = cutover.Add(6 * time.Hour)
	if _, err := c.CountInWindow(context.Background(), s, nil, 2*time.Hour); err != nil {
		t.Fatal(err)
	}
	if leg.called() {
		t.Error("a window after the cutover must not reach the legacy")
	}
	if len(prim.writes) == 0 || prim.writes[0] != "CountInWindow" {
		t.Error("the primary should have answered in its own terms")
	}
}

func TestCountInWindowSpanningTheCutoverSumsBoth(t *testing.T) {
	c, _, _ := build(t, nil)
	s := scope()
	s.To = cutover.Add(5 * time.Hour)
	n, err := c.CountInWindow(context.Background(), s, nil, 48*time.Hour)
	if err != nil || n != 12 {
		t.Fatalf("count = %d, %v; want 12", n, err)
	}
}

func TestPageNewestFirstReadsThePrimaryThenTheLegacy(t *testing.T) {
	cases := []struct {
		name          string
		offset, limit int
		want          []string
	}{
		{"inside the primary", 0, 3, []string{"p0", "p1", "p2"}},
		{"across the boundary", 3, 4, []string{"p3", "p4", "l0", "l1"}},
		{"inside the legacy", 6, 3, []string{"l1", "l2", "l3"}},
		{"to the end", 9, 10, []string{"l4", "l5", "l6"}},
		{"past the end", 12, 5, []string{}},
		{"exactly the primary", 0, 5, []string{"p0", "p1", "p2", "p3", "p4"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _, _ := build(t, nil)
			rows, total, err := c.FetchPage(context.Background(), scope(), nil, store.Page{Offset: tc.offset, Limit: tc.limit})
			if err != nil {
				t.Fatal(err)
			}
			if total != 12 {
				t.Errorf("total = %d, want 12", total)
			}
			eq(t, ids(rows), tc.want)
		})
	}
}

func TestPageOldestFirstReadsTheLegacyThenThePrimary(t *testing.T) {
	c, _, _ := build(t, nil)
	rows, total, err := c.FetchPage(context.Background(), scope(), nil, store.Page{Offset: 5, Limit: 4, Order: store.Asc})
	if err != nil {
		t.Fatal(err)
	}
	if total != 12 {
		t.Errorf("total = %d", total)
	}
	eq(t, ids(rows), []string{"l1", "l0", "p4", "p3"})
}

func TestPageSortedByAnotherFieldMergesBothStores(t *testing.T) {
	c, _, _ := build(t, nil)
	// n runs 0..4 in the primary and 0..6 in the legacy.
	rows, total, err := c.FetchPage(context.Background(), scope(), nil, store.Page{Offset: 0, Limit: 5, SortBy: "n", Order: store.Desc})
	if err != nil {
		t.Fatal(err)
	}
	if total != 12 {
		t.Errorf("total = %d", total)
	}
	var ns []string
	for _, r := range rows {
		ns = append(ns, gjson.GetBytes(r, "n").String())
	}
	eq(t, ns, []string{"6", "5", "4", "4", "3"})

	rows, _, _ = c.FetchPage(context.Background(), scope(), nil, store.Page{Offset: 10, Limit: 5, SortBy: "n", Order: store.Desc})
	if len(rows) != 2 {
		t.Errorf("the last page should hold 2 rows, got %d", len(rows))
	}
}

func TestAPageTooDeepToMergeIsRefusedRatherThanReadInFull(t *testing.T) {
	c, _, _ := build(t, nil)
	_, _, err := c.FetchPage(context.Background(), scope(), nil, store.Page{Offset: maxMergeWindow, Limit: 10, SortBy: "n"})
	if !errors.Is(err, store.ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func TestFetchNTakesTheNewestAcrossBoth(t *testing.T) {
	c, _, leg := build(t, nil)

	rows, err := c.FetchN(context.Background(), scope(), nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, ids(rows), []string{"p0", "p1", "p2"})
	if leg.called() {
		t.Error("the legacy was read although the primary filled the request")
	}

	rows, _ = c.FetchN(context.Background(), scope(), nil, 7)
	eq(t, ids(rows), []string{"p0", "p1", "p2", "p3", "p4", "l0", "l1"})
}

func TestFetchByIDFallsBackToTheLegacyOnlyWhenTheRecordIsNotInThePrimary(t *testing.T) {
	c, prim, leg := build(t, nil)
	leg.byID = map[string]json.RawMessage{"old": json.RawMessage(`{"id":"old"}`)}
	prim.byID = map[string]json.RawMessage{"new": json.RawMessage(`{"id":"new"}`)}

	if d, err := c.FetchByID(context.Background(), scope(), "new"); err != nil || gjson.GetBytes(d, "id").String() != "new" {
		t.Fatalf("new: %s %v", d, err)
	}
	if leg.called() {
		t.Error("a hit in the primary must not touch the legacy")
	}
	if d, err := c.FetchByID(context.Background(), scope(), "old"); err != nil || gjson.GetBytes(d, "id").String() != "old" {
		t.Fatalf("old: %s %v", d, err)
	}
	if _, err := c.FetchByID(context.Background(), scope(), "nowhere"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	other := store.Scope{Tenant: "x", Dataset: dsLogs}
	before := len(leg.scopes)
	_, _ = c.FetchByID(context.Background(), other, "old")
	if len(leg.scopes) != before {
		t.Error("another tenant must not fall back to the legacy")
	}
}

func TestTopValuesSumsTheSameKeyAcrossStores(t *testing.T) {
	c, prim, leg := build(t, nil)
	prim.rows = append(prim.rows, run("x", cutover.Add(time.Hour), 4, "shared")...)
	leg.rows = append(leg.rows, run("y", cutover.Add(-time.Hour), 3, "shared")...)

	got, err := c.TopValues(context.Background(), scope(), "host", nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []store.Bucket{{Key: "old", Count: 7}, {Key: "shared", Count: 7}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTimelineAddsTheBucketThatHoldsTheCutover(t *testing.T) {
	c, prim, leg := build(t, nil)
	// Both stores have records on the cutover's own day.
	prim.rows = []json.RawMessage{row("a", cutover.Add(time.Hour), "h", 1), row("b", cutover.Add(26*time.Hour), "h", 1)}
	leg.rows = []json.RawMessage{row("c", cutover.Add(-time.Hour), "h", 1), row("d", cutover.Add(-30*time.Hour), "h", 1)}

	pts, err := c.Timeline(context.Background(), scope(), nil, store.Interval{Calendar: store.CalendarDay})
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 4 {
		t.Fatalf("got %d buckets, want 4: %v", len(pts), pts)
	}
	for i := 1; i < len(pts); i++ {
		if !pts[i-1].At.Before(pts[i].At) {
			t.Errorf("buckets are not in time order: %v", pts)
		}
	}
}

func TestGroupByMergesGroupsAndKeepsTheRequestedLimit(t *testing.T) {
	c, prim, leg := build(t, nil)
	prim.rows = append(prim.rows, run("x", cutover.Add(time.Hour), 2, "shared")...)
	leg.rows = append(leg.rows, run("y", cutover.Add(-time.Hour), 6, "shared")...)

	got, err := c.GroupBy(context.Background(), scope(), []string{"host"}, nil, store.GroupOpts{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Key != "shared" || got[0].Count != 8 || got[1].Key != "old" || got[1].Count != 7 {
		t.Errorf("got %+v", got)
	}
}

func TestEveryWriteGoesToThePrimary(t *testing.T) {
	c, prim, leg := build(t, nil)
	ctx := context.Background()
	s := scope()

	_ = c.Insert(ctx, s, "1", nil)
	_, _ = c.BulkWriter(dsLogs)
	_, _ = c.UpdateWhere(ctx, s, nil, nil)
	_ = c.Flush(ctx, dsLogs)
	_ = c.SetRetention(ctx, dsLogs, store.Retention{})
	_, _ = c.Drop(ctx, s)

	if len(prim.writes) != 6 {
		t.Errorf("primary saw %v", prim.writes)
	}
	if len(leg.writes) != 0 {
		t.Errorf("the legacy is read-only but saw %v", leg.writes)
	}
}

func TestALegacyFailureFailsTheReadByDefault(t *testing.T) {
	c, _, leg := build(t, nil)
	leg.err = errors.New("opensearch is down")

	if _, err := c.Count(context.Background(), scope(), nil); err == nil {
		t.Fatal("a half answer must not look like a whole one")
	}
	if _, _, err := c.FetchPage(context.Background(), scope(), nil, store.Page{Limit: 3, Offset: 5}); err == nil {
		t.Fatal("FetchPage must fail too")
	}
}

func TestALegacyFailureDegradesToThePrimaryWhenTolerated(t *testing.T) {
	var gotOp string
	c, _, leg := build(t, func(cfg *Config) {
		cfg.TolerateLegacy = true
		cfg.OnLegacyError = func(op string, _ error) { gotOp = op }
	})
	leg.err = errors.New("opensearch is down")

	n, err := c.Count(context.Background(), scope(), nil)
	if err != nil || n != 5 {
		t.Fatalf("count = %d, %v; want the primary's 5", n, err)
	}
	if gotOp != "Count" {
		t.Errorf("OnLegacyError got %q", gotOp)
	}

	rows, total, err := c.FetchPage(context.Background(), scope(), nil, store.Page{Offset: 3, Limit: 4})
	if err != nil {
		t.Fatal(err)
	}
	eq(t, ids(rows), []string{"p3", "p4"})
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
}

func TestAPrimaryFailureIsNeverHidden(t *testing.T) {
	c, prim, _ := build(t, func(cfg *Config) { cfg.TolerateLegacy = true })
	prim.err = errors.New("clickhouse is down")
	if _, err := c.Count(context.Background(), scope(), nil); err == nil {
		t.Fatal("the primary's failure must surface")
	}
}

func TestLegacyQueriesUseTheLegacyFieldNamesAndComeBackInTheCurrentShape(t *testing.T) {
	toLegacy, fromLegacy := PrefixMap("log", "event")
	c, prim, leg := build(t, func(cfg *Config) {
		cfg.LegacyField = toLegacy
		cfg.LegacyFieldBack = fromLegacy
		cfg.LegacyDocument = RenameDocKey("event", "log")
	})
	leg.rows = []json.RawMessage{json.RawMessage(`{"@timestamp":"2026-10-31T23:00:00Z","id":"l0","event":{"user":"root"}}`)}
	prim.rows = nil

	filters := []store.Filter{{Field: "log.user", Op: store.OpEq, Value: "root"}, {Op: store.OpSearch, Value: "x"}}
	rows, _, err := c.FetchPage(context.Background(), scope(), filters, store.Page{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if got := leg.filter[len(leg.filter)-1]; got[0].Field != "event.user" || got[1].Field != "" {
		t.Errorf("legacy filters = %+v", got)
	}
	if len(rows) != 1 || gjson.GetBytes(rows[0], "log.user").String() != "root" {
		t.Errorf("the legacy record was not put in the current shape: %s", rows)
	}
}

func TestDescribeFieldsUnitesBothAndFlagsADisagreement(t *testing.T) {
	toLegacy, fromLegacy := PrefixMap("log", "event")
	c, prim, leg := build(t, func(cfg *Config) { cfg.LegacyField = toLegacy; cfg.LegacyFieldBack = fromLegacy })
	prim.fields = []store.Field{{Name: "log.user", Type: "string", Filterable: true, Searchable: true}, {Name: "new", Type: "string", Filterable: true}}
	leg.fields = []store.Field{{Name: "event.user", Type: "keyword", Filterable: true}, {Name: "old", Type: "long", Filterable: true}}

	got, err := c.DescribeFields(context.Background(), scope())
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]store.Field{}
	for _, f := range got {
		byName[f.Name] = f
	}
	if len(got) != 3 {
		t.Fatalf("got %+v", got)
	}
	u := byName["log.user"]
	if !u.Conflict {
		t.Error("string in one store and keyword in the other is a disagreement")
	}
	if u.Searchable {
		t.Error("a field only one store can search is not searchable through both")
	}
}

func TestHealthIsDegradedNotDownWhenOnlyTheLegacyIsUnwell(t *testing.T) {
	c, _, leg := build(t, nil)
	leg.health = store.Health{Status: store.HealthUnavailable, Message: "red"}

	h, err := c.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != store.HealthDegraded {
		t.Errorf("status = %q, want degraded", h.Status)
	}

	leg.health = store.Health{}
	if h, _ := c.Health(context.Background()); h.Status != store.HealthOK {
		t.Errorf("status = %q, want ok", h.Status)
	}
}

func TestPrefixMapAndRenameDocKey(t *testing.T) {
	to, back := PrefixMap("log", "event")
	for in, want := range map[string]string{"log": "event", "log.user": "event.user", "logger.x": "logger.x", "other": "other"} {
		if got := to(in); got != want {
			t.Errorf("to(%q) = %q, want %q", in, got, want)
		}
	}
	if back("event.user.name") != "log.user.name" {
		t.Error("back must undo to")
	}

	rename := RenameDocKey("event", "log")
	if got := string(rename(json.RawMessage(`{"id":"1","event":{"a":1}}`))); !gjson.Valid(got) || gjson.Get(got, "log.a").Int() != 1 || gjson.Get(got, "event").Exists() {
		t.Errorf("rename = %s", got)
	}
	if got := string(rename(json.RawMessage(`{"event":{"a":1},"log":{"a":2}}`))); gjson.Get(got, "log.a").Int() != 2 {
		t.Errorf("an existing key must win: %s", got)
	}
	if got := string(rename(json.RawMessage(`not json`))); got != "not json" {
		t.Errorf("rename mangled %q", got)
	}
}
