package opensearch

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

const (
	dsLogs   store.Dataset = "logs"
	dsAlerts store.Dataset = "alerts"
	dsStats  store.Dataset = "statistics"
	dsReport store.Dataset = "compliance_report"
)

func testLayout() SpecLayout {
	return SpecLayout{
		dsLogs:   {Prefix: "v11-log", IncludeDataType: true, DateFormat: "2006-01-02"},
		dsAlerts: {Prefix: "v11-alert", DateFormat: "2006-01-02"},
		dsStats:  {Prefix: "v11-statistics", DateFormat: "2006.01"},
		dsReport: {Prefix: "v11-compliance-report"},
	}
}

func testDriver(t *testing.T) *Driver {
	t.Helper()
	d, err := New(Config{Layout: testLayout(), TenantField: "tenantId"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return d
}

func TestNewRequiresLayoutAndTenantField(t *testing.T) {
	if _, err := New(Config{TenantField: "tenantId"}); err == nil {
		t.Error("want an error without a Layout")
	}
	if _, err := New(Config{Layout: testLayout()}); err == nil {
		t.Error("want an error without a TenantField")
	}
}

func TestNewDefaultsTimestampField(t *testing.T) {
	d := testDriver(t)
	if d.cfg.TimestampField != defaultTimestampField {
		t.Errorf("TimestampField = %q, want %q", d.cfg.TimestampField, defaultTimestampField)
	}
}

func TestBuildQueryAlwaysScopesToTenant(t *testing.T) {
	q, err := testDriver(t).buildQuery(store.Scope{Tenant: "acme", Dataset: dsLogs}, nil)
	if err != nil {
		t.Fatalf("buildQuery: %v", err)
	}
	if !hasClause(t, q, `{"term":{"tenantId":"acme"}}`) {
		t.Fatalf("tenant term missing from %s", mustJSON(t, q))
	}
}

func TestBuildQueryUsesConfiguredTenantField(t *testing.T) {
	d, err := New(Config{Layout: testLayout(), TenantField: "org_id"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	q, err := d.buildQuery(store.Scope{Tenant: "acme", Dataset: dsLogs}, nil)
	if err != nil {
		t.Fatalf("buildQuery: %v", err)
	}
	if !hasClause(t, q, `{"term":{"org_id":"acme"}}`) {
		t.Fatalf("configured tenant field not used: %s", mustJSON(t, q))
	}
}

func TestBuildQueryRejectsEmptyTenant(t *testing.T) {
	_, err := testDriver(t).buildQuery(store.Scope{Dataset: dsLogs}, nil)
	if !errors.Is(err, store.ErrNoTenant) {
		t.Fatalf("want ErrNoTenant, got %v", err)
	}
}

func TestBuildQueryAllTenantsOmitsTerm(t *testing.T) {
	q, err := testDriver(t).buildQuery(store.Scope{Tenant: store.AllTenants, Dataset: dsLogs}, nil)
	if err != nil {
		t.Fatalf("buildQuery: %v", err)
	}
	if hasClause(t, q, `{"term":{"tenantId":"*"}}`) {
		t.Fatalf("AllTenants should not emit a tenant term: %s", mustJSON(t, q))
	}
}

func TestBuildQueryTimeRange(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	q, err := testDriver(t).buildQuery(store.Scope{
		Tenant: "acme", Dataset: dsLogs, From: from, To: from.Add(time.Hour),
	}, nil)
	if err != nil {
		t.Fatalf("buildQuery: %v", err)
	}
	want := `{"range":{"@timestamp":{"gte":"2026-08-01T00:00:00Z","lte":"2026-08-01T01:00:00Z"}}}`
	if !hasClause(t, q, want) {
		t.Fatalf("time range missing from %s", mustJSON(t, q))
	}
}

func TestBuildQueryPlacesNegationsInMustNot(t *testing.T) {
	q, err := testDriver(t).buildQuery(
		store.Scope{Tenant: "acme", Dataset: dsAlerts},
		[]store.Filter{{Field: "action", Op: store.OpNotEq, Value: "allow"}},
	)
	if err != nil {
		t.Fatalf("buildQuery: %v", err)
	}
	b := q["bool"].(map[string]any)
	if _, ok := b["must_not"]; !ok {
		t.Fatalf("must_not missing from %s", mustJSON(t, q))
	}
	for _, c := range b["filter"].([]map[string]any) {
		if term, ok := c["term"].(map[string]any); ok && term["action"] != nil {
			t.Fatalf("negated filter leaked into filter clause: %s", mustJSON(t, q))
		}
	}
}

func TestFilterClauses(t *testing.T) {
	cases := []struct {
		name   string
		filter store.Filter
		want   string
		negate bool
	}{
		{"eq", store.Filter{Field: "action", Op: store.OpEq, Value: "SendAs"},
			`{"term":{"action":"SendAs"}}`, false},
		{"not eq", store.Filter{Field: "action", Op: store.OpNotEq, Value: "SendAs"},
			`{"term":{"action":"SendAs"}}`, true},
		{"in", store.Filter{Field: "target.port", Op: store.OpIn, Value: []int{9001, 9030}},
			`{"terms":{"target.port":[9001,9030]}}`, false},
		{"not in", store.Filter{Field: "target.port", Op: store.OpNotIn, Value: []int{9001}},
			`{"terms":{"target.port":[9001]}}`, true},
		{"gte", store.Filter{Field: "log.bytes", Op: store.OpGte, Value: 1000},
			`{"range":{"log.bytes":{"gte":1000}}}`, false},
		{"between", store.Filter{Field: "log.bytes", Op: store.OpBetween, Value: []any{1, 9}},
			`{"range":{"log.bytes":{"gte":1,"lte":9}}}`, false},
		{"contains", store.Filter{Field: "log.message", Op: store.OpContains, Value: "denied"},
			`{"match":{"log.message":"denied"}}`, false},
		{"exists", store.Filter{Field: "origin.ip", Op: store.OpExists},
			`{"exists":{"field":"origin.ip"}}`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, negate, err := filterClause(tc.filter)
			if err != nil {
				t.Fatalf("filterClause: %v", err)
			}
			if negate != tc.negate {
				t.Errorf("negate = %v, want %v", negate, tc.negate)
			}
			if s := mustJSON(t, got); s != tc.want {
				t.Errorf("got %s, want %s", s, tc.want)
			}
		})
	}
}

func TestFilterClauseRejectsUnknownOperator(t *testing.T) {
	_, _, err := filterClause(store.Filter{Field: "x", Op: store.Op("regex")})
	if !errors.Is(err, store.ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func TestSpecLayoutReadIndices(t *testing.T) {
	l := testLayout()
	cases := []struct {
		scope store.Scope
		want  string
	}{
		{store.Scope{Dataset: dsLogs}, "v11-log-*"},
		{store.Scope{Dataset: dsLogs, DataType: "o365"}, "v11-log-o365-*"},
		{store.Scope{Dataset: dsAlerts}, "v11-alert-*"},
		{store.Scope{Dataset: dsStats}, "v11-statistics-*"},
		{store.Scope{Dataset: dsReport}, "v11-compliance-report"},
	}
	for _, tc := range cases {
		got, err := l.ReadIndices(tc.scope)
		if err != nil {
			t.Fatalf("ReadIndices(%v): %v", tc.scope, err)
		}
		if len(got) != 1 || got[0] != tc.want {
			t.Errorf("ReadIndices(%v) = %v, want [%s]", tc.scope, got, tc.want)
		}
	}
}

func TestSpecLayoutRejectsUnknownDataset(t *testing.T) {
	_, err := testLayout().ReadIndices(store.Scope{Dataset: "nope"})
	if !errors.Is(err, store.ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func TestSpecLayoutWriteIndex(t *testing.T) {
	l := testLayout()
	now := time.Date(2026, 8, 2, 15, 4, 5, 0, time.UTC)
	cases := []struct {
		scope store.Scope
		want  string
	}{
		{store.Scope{Dataset: dsLogs, DataType: "wineventlog"}, "v11-log-wineventlog-2026-08-02"},
		{store.Scope{Dataset: dsAlerts}, "v11-alert-2026-08-02"},
		{store.Scope{Dataset: dsStats}, "v11-statistics-2026.08"},
		{store.Scope{Dataset: dsReport}, "v11-compliance-report"},
	}
	for _, tc := range cases {
		got, err := l.WriteIndex(tc.scope, now)
		if err != nil {
			t.Fatalf("WriteIndex(%v): %v", tc.scope, err)
		}
		if got != tc.want {
			t.Errorf("WriteIndex(%v) = %q, want %q", tc.scope, got, tc.want)
		}
	}
}

func TestSpecLayoutWriteIndexRequiresDataType(t *testing.T) {
	if _, err := testLayout().WriteIndex(store.Scope{Dataset: dsLogs}, time.Now()); err == nil {
		t.Fatal("want an error when writing a log with no DataType")
	}
}

func TestSpecLayoutDataTypeOf(t *testing.T) {
	l := testLayout()
	cases := []struct {
		ds    store.Dataset
		index string
		want  string
	}{
		{dsLogs, "v11-log-o365-2026-07-23", "o365"},
		{dsLogs, "v11-log-antivirus-bitdefender-gz-2026-07-23", "antivirus-bitdefender-gz"},
		{dsAlerts, "v11-alert-2026-07-23", ""},
		{dsStats, "v11-statistics-2026.07", ""},
	}
	for _, tc := range cases {
		if got := l.DataTypeOf(tc.ds, tc.index); got != tc.want {
			t.Errorf("DataTypeOf(%s) = %q, want %q", tc.index, got, tc.want)
		}
	}
}

func TestIntervalSpec(t *testing.T) {
	cases := []struct {
		iv      store.Interval
		wantKey string
		wantVal string
	}{
		{store.Interval{Duration: 5 * time.Minute}, "fixed_interval", "5m"},
		{store.Interval{Duration: 2 * time.Hour}, "fixed_interval", "2h"},
		{store.Interval{Duration: 24 * time.Hour}, "fixed_interval", "1d"},
		{store.Interval{Duration: 30 * time.Second}, "fixed_interval", "30s"},
		{store.Interval{Calendar: store.CalendarMonth}, "calendar_interval", "month"},
	}
	for _, tc := range cases {
		k, v, err := intervalSpec(tc.iv)
		if err != nil {
			t.Fatalf("intervalSpec(%v): %v", tc.iv, err)
		}
		if k != tc.wantKey || v != tc.wantVal {
			t.Errorf("intervalSpec(%v) = %s/%s, want %s/%s", tc.iv, k, v, tc.wantKey, tc.wantVal)
		}
	}
	if _, _, err := intervalSpec(store.Interval{}); err == nil {
		t.Error("want an error for an empty interval")
	}
}

func TestParseIndexAge(t *testing.T) {
	cases := map[string]time.Duration{
		"30d":   30 * 24 * time.Hour,
		"24h":   24 * time.Hour,
		"15m":   15 * time.Minute,
		"30s":   30 * time.Second,
		"500ms": 500 * time.Millisecond,
		"":      0,
	}
	for in, want := range cases {
		got, err := parseIndexAge(in)
		if err != nil {
			t.Fatalf("parseIndexAge(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("parseIndexAge(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := parseIndexAge("nonsense"); err == nil {
		t.Error("want an error for unparseable input")
	}
}

func TestParseBytes(t *testing.T) {
	gb, mb := float64(1<<30), float64(1<<20)
	cases := map[string]int64{
		"2.9gb":   int64(2.9 * gb),
		"909.5mb": int64(909.5 * mb),
		"512b":    512,
		"":        0,
	}
	for in, want := range cases {
		if got := parseBytes(in); got != want {
			t.Errorf("parseBytes(%q) = %d, want %d", in, got, want)
		}
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func hasClause(t *testing.T, q map[string]any, want string) bool {
	t.Helper()
	b, ok := q["bool"].(map[string]any)
	if !ok {
		return false
	}
	clauses, ok := b["filter"].([]map[string]any)
	if !ok {
		return false
	}
	for _, c := range clauses {
		if mustJSON(t, c) == want {
			return true
		}
	}
	return false
}

func TestRawQueryRequiresAllTenants(t *testing.T) {
	d := testDriver(t)
	if _, err := d.RawQuery(context.Background(), store.Scope{Tenant: "acme"}, "SELECT 1"); !errors.Is(err, store.ErrUnsupported) {
		t.Fatalf("want ErrUnsupported for a tenant-scoped RawQuery, got %v", err)
	}
	if _, err := d.RawQuery(context.Background(), store.Scope{}, "SELECT 1"); !errors.Is(err, store.ErrNoTenant) {
		t.Fatalf("want ErrNoTenant, got %v", err)
	}
}

func TestFormatIndexAgeRoundTrips(t *testing.T) {
	for _, want := range []time.Duration{
		30 * 24 * time.Hour, 36 * time.Hour, 90 * time.Minute, 45 * time.Second,
	} {
		got, err := parseIndexAge(formatIndexAge(want))
		if err != nil {
			t.Fatalf("parseIndexAge(formatIndexAge(%v)): %v", want, err)
		}
		if got != want {
			t.Errorf("round trip of %v gave %v", want, got)
		}
	}
}

func TestPatchDeleteAge(t *testing.T) {
	policy := map[string]any{"states": []any{
		map[string]any{"name": "open", "transitions": []any{
			map[string]any{"state_name": "delete", "conditions": map[string]any{"min_index_age": "30d"}},
		}},
		map[string]any{"name": "ingest", "transitions": []any{
			map[string]any{"state_name": "open", "conditions": map[string]any{"min_index_age": "24h"}},
		}},
	}}

	if n := patchDeleteAge(policy, "90d"); n != 1 {
		t.Fatalf("patched %d transitions, want 1", n)
	}

	states := policy["states"].([]any)
	del := states[0].(map[string]any)["transitions"].([]any)[0].(map[string]any)
	if got := del["conditions"].(map[string]any)["min_index_age"]; got != "90d" {
		t.Errorf("delete transition = %v, want 90d", got)
	}
	// The unrelated ingest transition must be untouched.
	ing := states[1].(map[string]any)["transitions"].([]any)[0].(map[string]any)
	if got := ing["conditions"].(map[string]any)["min_index_age"]; got != "24h" {
		t.Errorf("ingest transition changed to %v, want 24h", got)
	}
}
