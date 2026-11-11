package clickhouse_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/threatwinds/go-sdk/store"
	ch "github.com/threatwinds/go-sdk/store/clickhouse"
)

const (
	dsLogs       store.Dataset = "logs"
	dsAlerts     store.Dataset = "alerts"
	dsStatistics store.Dataset = "statistics"
)

// newLive connects to a ClickHouse the caller provides. Skipped when there is
// none, so the suite stays runnable without one.
func newLive(t *testing.T) *ch.Driver {
	t.Helper()
	addr := os.Getenv("CLICKHOUSE_ADDR")
	if addr == "" {
		t.Skip("set CLICKHOUSE_ADDR to run the live driver tests")
	}
	d, err := ch.New(ch.Config{
		Addr:     []string{addr},
		Database: "utmstack",
		Username: "default",
		Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		Tables: map[store.Dataset]string{
			dsLogs:       "logs",
			dsAlerts:     "alerts",
			dsStatistics: "statistics",
		},
		TenantColumn:   "tenantId",
		TimeColumn:     "@timestamp",
		DataTypeColumn: "dataType",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return d
}

func scope(tenant string) store.Scope {
	return store.Scope{Tenant: tenant, Dataset: dsLogs}
}

// seed writes the fixture a test needs rather than relying on whatever the
// table happens to hold, so the suite does not depend on the order it runs in
// or on data another test left behind.
func seed(t *testing.T, d *ch.Driver, tenant string, docs ...map[string]any) {
	t.Helper()
	ctx := context.Background()
	for _, doc := range docs {
		if _, ok := doc["@timestamp"]; !ok {
			doc["@timestamp"] = time.Now().UTC().Format("2006-01-02 15:04:05.000")
		}
		if err := d.Insert(ctx, scope(tenant), "", doc); err != nil {
			t.Fatalf("seeding %s: %v", tenant, err)
		}
	}
}

// The property the whole design rests on: a scope with no tenant is an error,
// never a read of everything.
func TestLiveRefusesAScopeWithoutATenant(t *testing.T) {
	d := newLive(t)
	if _, err := d.Count(context.Background(), scope(""), nil); err != store.ErrNoTenant {
		t.Fatalf("Count with no tenant = %v, want ErrNoTenant", err)
	}
}

func TestLiveCountIsScopedToTheTenant(t *testing.T) {
	d := newLive(t)
	ctx := context.Background()

	seed(t, d, "tenant-a", map[string]any{"dataType": "linux", "dataSource": "h1", "raw": "a"})
	seed(t, d, "tenant-b", map[string]any{"dataType": "linux", "dataSource": "h2", "raw": "b"})

	a, err := d.Count(ctx, scope("tenant-a"), nil)
	if err != nil {
		t.Fatalf("count a: %v", err)
	}
	b, err := d.Count(ctx, scope("tenant-b"), nil)
	if err != nil {
		t.Fatalf("count b: %v", err)
	}
	all, err := d.Count(ctx, scope(store.AllTenants), nil)
	if err != nil {
		t.Fatalf("count all: %v", err)
	}

	if a == 0 || b == 0 {
		t.Fatalf("expected rows for both tenants, got a=%d b=%d", a, b)
	}
	// The property is isolation, not arithmetic over a shared fixture: each
	// tenant sees only its own, so neither count can reach the total.
	if a >= all || b >= all {
		t.Fatalf("a tenant's count reaches the total: a=%d b=%d all=%d", a, b, all)
	}
	if a+b > all {
		t.Fatalf("tenant counts exceed the total: a=%d b=%d all=%d", a, b, all)
	}
}

// Filtering inside the dynamic JSON column is what replaces the field mappings
// the previous engine needed declared up front.
func TestLiveFiltersOnAJSONPath(t *testing.T) {
	d := newLive(t)

	seed(t, d, "tenant-json", map[string]any{
		"dataType": "wineventlog", "dataSource": "DC01",
		"log": map[string]any{"event_id": 4688, "process": map[string]any{"name": "powershell.exe"}},
	})

	rows, err := d.FetchN(context.Background(), scope(store.AllTenants), []store.Filter{
		{Field: "log.event_id", Op: store.OpEq, Value: 4688},
	}, 10)
	if err != nil {
		t.Fatalf("FetchN: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("the 4688 event was not found by its JSON path")
	}
}

func TestLiveWriteAndReadBack(t *testing.T) {
	d := newLive(t)
	ctx := context.Background()
	s := scope("tenant-live")
	s.From = time.Now().UTC().Add(-time.Hour)

	doc := map[string]any{
		"@timestamp": time.Now().UTC().Format("2006-01-02 15:04:05.000"),
		"dataType":   "linux",
		"dataSource": "test-host",
		"severity":   "low",
		"log":        map[string]any{"probe": true, "n": 7},
		"raw":        "written by the driver test",
	}
	if err := d.Insert(ctx, s, "live-1", doc); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// The insert is asynchronous, so give the server a moment to make it
	// visible rather than asserting on a race.
	deadline := time.Now().Add(10 * time.Second)
	for {
		n, err := d.Count(ctx, scope("tenant-live"), nil)
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the written row never became visible")
		}
		time.Sleep(300 * time.Millisecond)
	}

	// And it must not be visible to another tenant.
	if n, err := d.Count(ctx, scope("tenant-a"), []store.Filter{
		{Field: "id", Op: store.OpEq, Value: "live-1"},
	}); err != nil || n != 0 {
		t.Fatalf("another tenant sees the row: n=%d err=%v", n, err)
	}
}

func TestLiveTopValuesAndTimeline(t *testing.T) {
	d := newLive(t)
	ctx := context.Background()

	buckets, err := d.TopValues(ctx, scope(store.AllTenants), "dataType", nil, 5)
	if err != nil {
		t.Fatalf("TopValues: %v", err)
	}
	if len(buckets) == 0 {
		t.Fatal("no buckets")
	}

	pts, err := d.Timeline(ctx, scope(store.AllTenants), nil, store.Interval{Calendar: store.CalendarDay})
	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}
	if len(pts) == 0 {
		t.Fatal("no timeline points")
	}
}

// GroupBy takes a list of fields and Group carries Children, so a caller asking
// for two levels expects two. Grouping by only the first and returning no
// children reports a plausible answer built from half the request, which no
// error would reveal.
func TestLiveGroupByNestsEveryField(t *testing.T) {
	ctx := context.Background()
	const ds store.Dataset = "grouped"

	d := newLiveTable(t, ds, "logs_grouped")
	tenant := "t-groupby"
	sc := store.Scope{Tenant: tenant, Dataset: ds}

	// Two dataTypes, two sources under the first and one under the second.
	rows := []struct {
		dataType, dataSource string
		n                    int
	}{
		{"firewall", "fw-01", 5},
		{"firewall", "fw-02", 3},
		{"linux", "srv-01", 2},
	}
	for _, r := range rows {
		for i := 0; i < r.n; i++ {
			doc := fmt.Sprintf(`{"@timestamp":%q,"dataType":%q,"dataSource":%q,"x":1}`,
				time.Now().UTC().Format("2006-01-02 15:04:05.000"), r.dataType, r.dataSource)
			if err := d.Insert(ctx, sc, "", json.RawMessage(doc)); err != nil {
				t.Fatalf("Insert: %v", err)
			}
		}
	}

	groups, err := d.GroupBy(ctx, sc, []string{"dataType", "dataSource"}, nil, store.GroupOpts{Limit: 100})
	if err != nil {
		t.Fatalf("GroupBy: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("top level = %d groups, want 2", len(groups))
	}

	// Biggest first, and a parent totals its children.
	if groups[0].Key != "firewall" || groups[0].Count != 8 {
		t.Errorf("first group = %q/%d, want firewall/8", groups[0].Key, groups[0].Count)
	}
	if len(groups[0].Children) != 2 {
		t.Fatalf("firewall has %d children, want 2", len(groups[0].Children))
	}
	if groups[0].Children[0].Field != "dataSource" {
		t.Errorf("child field = %q, want dataSource", groups[0].Children[0].Field)
	}
	if groups[0].Children[0].Key != "fw-01" || groups[0].Children[0].Count != 5 {
		t.Errorf("first child = %q/%d, want fw-01/5", groups[0].Children[0].Key, groups[0].Children[0].Count)
	}
	if groups[1].Key != "linux" || groups[1].Count != 2 || len(groups[1].Children) != 1 {
		t.Errorf("second group = %q/%d with %d children", groups[1].Key, groups[1].Count, len(groups[1].Children))
	}
}

// DescribeFields joins the declared columns with the paths found inside the
// JSON, which is how a caller learns about fields nobody declared.
func TestLiveDescribeFindsJSONPaths(t *testing.T) {
	d := newLive(t)

	seed(t, d, "tenant-describe", map[string]any{
		"dataType": "wineventlog", "dataSource": "DC01",
		"log": map[string]any{"event_id": 4625},
	})

	fields, err := d.DescribeFields(context.Background(), scope(store.AllTenants))
	if err != nil {
		t.Fatalf("DescribeFields: %v", err)
	}

	var sawColumn, sawJSONPath bool
	for _, f := range fields {
		if f.Name == "tenantId" {
			sawColumn = true
		}
		if f.Name == "log.event_id" {
			sawJSONPath = true
		}
	}
	if !sawColumn {
		t.Error("declared columns missing")
	}
	if !sawJSONPath {
		t.Error("discovered JSON paths missing")
	}
}

// The failure this guards against is silent: MODIFY TTL replaces the whole
// expression, so asking for a flat retention on a tiered table would drop the
// move to cold storage and nothing would say so — the data would just stop
// leaving the hot disk until it filled.
func TestLiveSetRetentionRefusesToFlattenATier(t *testing.T) {
	ctx := context.Background()
	const ds store.Dataset = "tiered"

	// Its own tiered table. The product tables ship flat, so depending on one of
	// them being tiered would make this test pass or fail on a deployment
	// choice rather than on the behaviour it is about.
	tiered := newLiveTable(t, ds, "logs_tiered")
	if err := tiered.EnableTiering(ctx, ds, "hot_cold", store.Retention{
		Keep:      730 * 24 * time.Hour,
		ColdAfter: 90 * 24 * time.Hour,
	}); err != nil {
		t.Fatalf("EnableTiering: %v", err)
	}

	before, err := tiered.Retention(ctx, ds)
	if err != nil {
		t.Fatalf("Retention: %v", err)
	}
	if !before.Tiered() {
		t.Fatalf("the tiered table reports no cold tier: %+v", before)
	}

	err = tiered.SetRetention(ctx, ds, store.Retention{Keep: 30 * 24 * time.Hour})
	if err == nil {
		t.Fatal("a flat retention was accepted on a tiered table")
	}

	after, err := tiered.Retention(ctx, ds)
	if err != nil {
		t.Fatalf("Retention: %v", err)
	}
	if after.ColdAfter != before.ColdAfter || after.Keep != before.Keep {
		t.Fatalf("retention changed despite the refusal: before=%v after=%v", before, after)
	}
}

// newLiveTable creates a throwaway table following the partitioning standard
// and returns a driver pointed at it. Partitioning by month with the tenant in
// the sorting key rather than the partition key is what the product tables do;
// a fixture that diverged would be testing a layout nothing ships.
func newLiveTable(t *testing.T, ds store.Dataset, name string) *ch.Driver {
	t.Helper()
	ctx := context.Background()
	d := newLive(t)

	if err := d.Exec(ctx, `CREATE TABLE IF NOT EXISTS utmstack.`+name+` (
		tenantId LowCardinality(String),
		`+"`@timestamp`"+` DateTime64(3,'UTC'),
		dataType LowCardinality(String) DEFAULT '',
		dataSource String DEFAULT '',
		x UInt8
	) ENGINE = MergeTree
	PARTITION BY toYYYYMM(`+"`@timestamp`"+`)
	ORDER BY (tenantId, `+"`@timestamp`"+`)
	TTL toDateTime(`+"`@timestamp`"+`) + INTERVAL 730 DAY DELETE
	SETTINGS ttl_only_drop_parts = 1`); err != nil {
		t.Fatalf("creating %s: %v", name, err)
	}
	t.Cleanup(func() { _ = d.Exec(ctx, "DROP TABLE IF EXISTS utmstack."+name) })

	out, err := ch.New(ch.Config{
		Addr:         []string{os.Getenv("CLICKHOUSE_ADDR")},
		Database:     "utmstack",
		Username:     "default",
		Password:     os.Getenv("CLICKHOUSE_PASSWORD"),
		Tables:       map[store.Dataset]string{ds: name},
		TenantColumn: "tenantId",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return out
}

func TestLiveRetentionRoundTrip(t *testing.T) {
	d := newLive(t)
	ctx := context.Background()

	before, err := d.Retention(ctx, dsLogs)
	if err != nil {
		t.Fatalf("Retention: %v", err)
	}
	t.Cleanup(func() { _ = d.SetRetention(ctx, dsLogs, before) })

	want := store.Retention{Keep: 730 * 24 * time.Hour}
	if before.Tiered() {
		want.ColdAfter = 90 * 24 * time.Hour
	}
	if err := d.SetRetention(ctx, dsLogs, want); err != nil {
		t.Fatalf("SetRetention: %v", err)
	}

	got, err := d.Retention(ctx, dsLogs)
	if err != nil {
		t.Fatalf("Retention: %v", err)
	}
	if got.Keep != want.Keep || got.ColdAfter != want.ColdAfter {
		t.Fatalf("round trip = %v, want %v", got, want)
	}
}

func TestSetRetentionRejectsAColdTierLongerThanKeep(t *testing.T) {
	d := newLive(t)

	err := d.SetRetention(context.Background(), dsLogs, store.Retention{
		Keep:      30 * 24 * time.Hour,
		ColdAfter: 60 * 24 * time.Hour,
	})
	if err == nil {
		t.Fatal("accepted a cold tier that outlives the retention")
	}
}

// The three datasets are separate tables because their shapes, volumes and
// lifetimes differ. This checks the driver reaches each one and that the
// retention each was created with is what comes back.
//
// No cold tier is expected: the shipped schema keeps everything on local disk,
// and object storage is opt-in, added later by EnableTiering. A deployment that
// took it up is not what this asserts.
func TestLiveTheThreeDatasets(t *testing.T) {
	d := newLive(t)
	ctx := context.Background()

	for _, tc := range []struct {
		ds       store.Dataset
		keepDays int
		coldDays int
	}{
		{dsLogs, 730, 0},
		{dsAlerts, 730, 0},
		{dsStatistics, 1095, 0},
	} {
		t.Run(string(tc.ds), func(t *testing.T) {
			if _, err := d.Count(ctx, store.Scope{Tenant: store.AllTenants, Dataset: tc.ds}, nil); err != nil {
				t.Fatalf("Count: %v", err)
			}

			r, err := d.Retention(ctx, tc.ds)
			if err != nil {
				t.Fatalf("Retention: %v", err)
			}
			if got := int(r.Keep.Hours() / 24); got != tc.keepDays {
				t.Errorf("Keep = %d days, want %d", got, tc.keepDays)
			}
			if got := int(r.ColdAfter.Hours() / 24); got != tc.coldDays {
				t.Errorf("ColdAfter = %d days, want %d", got, tc.coldDays)
			}
		})
	}
}

// An alert whose logs have expired is a drill-down that errors, so the two must
// expire together. This is the assertion that keeps a well-meaning change to
// one of them from quietly breaking the other.
func TestLiveAlertsDoNotOutliveTheirLogs(t *testing.T) {
	d := newLive(t)
	ctx := context.Background()

	logs, err := d.Retention(ctx, dsLogs)
	if err != nil {
		t.Fatalf("logs retention: %v", err)
	}
	alerts, err := d.Retention(ctx, dsAlerts)
	if err != nil {
		t.Fatalf("alerts retention: %v", err)
	}

	if alerts.Keep > logs.Keep {
		t.Fatalf("alerts are kept %v and logs only %v: the drill-down would break for the difference",
			alerts.Keep, logs.Keep)
	}
}

func TestLiveWriteToEachDataset(t *testing.T) {
	d := newLive(t)
	ctx := context.Background()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000")

	docs := map[store.Dataset]map[string]any{
		dsLogs: {
			"@timestamp": now, "dataType": "linux", "dataSource": "h1",
			"log": map[string]any{"cmd": "id"}, "raw": "uid=0",
		},
		dsAlerts: {
			// severity is numeric in a stored alert: the plugin's AlertFields
			// declares its own int, which shadows the protobuf's "low"/"medium"/
			// "high". A string here is not what the table receives.
			"@timestamp": now, "name": "Suspicious logon", "dataType": "wineventlog",
			"dataSource": "DC01", "severity": 3, "severityLabel": "High",
			"technique": "T1078",
		},
		dsStatistics: {
			"@timestamp": now, "type": "enqueue_success", "dataType": "linux",
			"dataSource": "h1", "count": 42, "bytes": 8192,
		},
	}

	for ds, doc := range docs {
		s := store.Scope{Tenant: "tenant-write", Dataset: ds}
		if err := d.Insert(ctx, s, "", doc); err != nil {
			t.Fatalf("insert into %s: %v", ds, err)
		}
	}

	for ds := range docs {
		s := store.Scope{Tenant: "tenant-write", Dataset: ds}
		n, err := d.Count(ctx, s, nil)
		if err != nil {
			t.Fatalf("count %s: %v", ds, err)
		}
		if n == 0 {
			t.Errorf("%s: nothing was written", ds)
		}
	}
}

// The upgrade path an on-prem install takes: it starts with plain retention and
// no object storage, and adds tiering later without rebuilding the table or
// losing what it already holds.
func TestLiveEnableTieringOnAnExistingTable(t *testing.T) {
	ctx := context.Background()
	const ds store.Dataset = "simple"

	// Its own table, created plain and dropped afterwards, so the test does not
	// depend on one another test left behind.
	simple := newLiveTable(t, ds, "logs_simple")

	before, err := simple.Count(ctx, store.Scope{Tenant: store.AllTenants, Dataset: ds}, nil)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}

	err = simple.EnableTiering(ctx, ds, "hot_cold", store.Retention{
		Keep:      730 * 24 * time.Hour,
		ColdAfter: 90 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("EnableTiering: %v", err)
	}

	got, err := simple.Retention(ctx, ds)
	if err != nil {
		t.Fatalf("Retention: %v", err)
	}
	if !got.Tiered() {
		t.Fatal("the table still reports no cold tier")
	}

	after, err := simple.Count(ctx, store.Scope{Tenant: store.AllTenants, Dataset: ds}, nil)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if after != before {
		t.Fatalf("rows changed across the migration: %d -> %d", before, after)
	}
}
