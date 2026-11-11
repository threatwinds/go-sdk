package factory

import (
	"strings"
	"testing"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

const dual = `
primary:
  engine: clickhouse
  clickhouse:
    addr: ["127.0.0.1:9000"]
    database: utmstack
    tenantColumn: tenantId
    tables: {logs: logs, alerts: alerts}
legacy:
  engine: opensearch
  opensearch:
    urls: ["https://127.0.0.1:9200"]
    username: admin
    password: secret
    tenantField: tenantId
    layout:
      logs: {prefix: v11-log, includeDataType: true, dateFormat: "2006.01.02"}
      alerts: {prefix: v11-alert, dateFormat: "2006.01.02"}
    textFields: {logs: raw}
  cutover: "2026-11-01T00:00:00Z"
  expires: "2026-12-01T00:00:00Z"
  tenant: default-tenant
  datasets: [logs, alerts]
  fieldPrefix: {current: log, legacy: event}
`

const single = `
primary:
  engine: clickhouse
  clickhouse:
    addr: ["127.0.0.1:9000"]
    tenantColumn: tenantId
    tables: {logs: logs}
`

func mustParse(t *testing.T, raw string) Config {
	t.Helper()
	cfg, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return cfg
}

func TestAPrimaryAloneBuildsThePrimaryItself(t *testing.T) {
	b, err := New(mustParse(t, single), Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })

	if b.Store == nil {
		t.Fatal("no store")
	}
	if b.Composite != nil {
		t.Error("without a legacy there is nothing to composite")
	}
	if b.ResolveIndexPattern != nil {
		t.Error("a ClickHouse primary has no index patterns to resolve")
	}
}

func TestADualConfigurationBuildsACompositeThatKnowsWhenTheLegacyEnds(t *testing.T) {
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	b, err := New(mustParse(t, dual), Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })

	if b.Composite == nil || b.Store != store.Store(b.Composite) {
		t.Fatal("a legacy must give a composite")
	}
	if !b.Composite.LegacyActive() {
		t.Error("the legacy is active before it expires")
	}

	now = time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC)
	if b.Composite.LegacyActive() {
		t.Error("the legacy must be inactive after the configured expiry")
	}
}

func TestTheLegacyLayoutResolvesTheIndexPatternsOfV11Rules(t *testing.T) {
	b, err := New(mustParse(t, dual), Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })

	if b.ResolveIndexPattern == nil {
		t.Fatal("no resolver")
	}
	if dt, ok := b.ResolveIndexPattern("logs", "v11-log-deceptive-bytes-*"); !ok || dt != "deceptive-bytes" {
		t.Errorf("got (%q, %v)", dt, ok)
	}
	if _, ok := b.ResolveIndexPattern("logs", "somebody-elses-*"); ok {
		t.Error("an unknown pattern must not resolve")
	}
}

func TestParseRejectsAnUnknownField(t *testing.T) {
	_, err := Parse([]byte("primary:\n  engine: clickhouse\n  clickhous: {}\n"))
	if err == nil {
		t.Fatal("a misspelt setting must be an error, not ignored")
	}
}

func TestParseAcceptsJSONToo(t *testing.T) {
	raw := `{"primary":{"engine":"clickhouse","clickhouse":{"addr":["h:9000"],"tenantColumn":"t","tables":{"logs":"logs"}}}}`
	if _, err := Parse([]byte(raw)); err != nil {
		t.Fatal(err)
	}
}

func TestNewRefusesAConfigurationThatCannotWork(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{"no engine", func(s string) string { return strings.Replace(s, "engine: clickhouse", "", 1) }, "primary.engine is required"},
		{"unknown engine", func(s string) string { return strings.Replace(s, "engine: clickhouse", "engine: mysql", 1) }, "is not one of"},
		{"engine without its section", func(s string) string {
			return "primary:\n  engine: opensearch\n"
		}, "primary.opensearch is missing"},
		{"legacy without cutover", func(s string) string { return strings.Replace(s, `  cutover: "2026-11-01T00:00:00Z"`, "", 1) }, "legacy.cutover"},
		{"cutover not a time", func(s string) string { return strings.Replace(s, "2026-11-01T00:00:00Z", "yesterday", 1) }, "legacy.cutover"},
		{"expiry before cutover", func(s string) string { return strings.Replace(s, "2026-12-01T00:00:00Z", "2026-10-01T00:00:00Z", 1) }, "after legacy.cutover"},
		{"legacy without tenant", func(s string) string { return strings.Replace(s, "  tenant: default-tenant", "", 1) }, "legacy.tenant is required"},
		{"field prefix without a legacy name", func(s string) string { return strings.Replace(s, "legacy: event", "legacy: log", 1) }, "fieldPrefix"},
		{"legacy without datasets", func(s string) string { return strings.Replace(s, "  datasets: [logs, alerts]", "", 1) }, "legacy.datasets"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Parse([]byte(tc.mutate(dual)))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			_, err = New(cfg, Options{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error containing %q", err, tc.want)
			}
		})
	}
}

func TestTwoOpenSearchStoresMustBeTheSameCluster(t *testing.T) {
	raw := `
primary:
  engine: opensearch
  opensearch:
    urls: ["https://a:9200"]
    tenantField: tenantId
    layout: {logs: {prefix: log, includeDataType: true}}
legacy:
  engine: opensearch
  opensearch:
    urls: ["https://b:9200"]
    tenantField: tenantId
    layout: {logs: {prefix: v11-log, includeDataType: true}}
  cutover: "2026-11-01T00:00:00Z"
  tenant: t
  datasets: [logs]
`
	_, err := New(mustParse(t, raw), Options{})
	if err == nil || !strings.Contains(err.Error(), "same one") {
		t.Fatalf("got %v", err)
	}
}
