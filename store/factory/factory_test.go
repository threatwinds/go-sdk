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
secondary:
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
  fieldPrefix: {primary: log, secondary: event}
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
		t.Error("without a secondary there is nothing to composite")
	}
	if b.ResolveIndexPattern != nil {
		t.Error("a ClickHouse primary has no index patterns to resolve")
	}
}

func TestADualConfigurationBuildsACompositeThatKnowsWhenTheSecondaryEnds(t *testing.T) {
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	b, err := New(mustParse(t, dual), Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })

	if b.Composite == nil || b.Store != store.Store(b.Composite) {
		t.Fatal("a secondary must give a composite")
	}
	if !b.Composite.SecondaryActive() {
		t.Error("the secondary is active before it expires")
	}

	now = time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC)
	if b.Composite.SecondaryActive() {
		t.Error("the secondary must be inactive after the configured expiry")
	}
}

func TestTheSecondaryLayoutResolvesTheIndexPatternsOfV11Rules(t *testing.T) {
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
		{"secondary without cutover", func(s string) string { return strings.Replace(s, `  cutover: "2026-11-01T00:00:00Z"`, "", 1) }, "secondary.cutover"},
		{"cutover not a time", func(s string) string { return strings.Replace(s, "2026-11-01T00:00:00Z", "yesterday", 1) }, "secondary.cutover"},
		{"expiry before cutover", func(s string) string { return strings.Replace(s, "2026-12-01T00:00:00Z", "2026-10-01T00:00:00Z", 1) }, "after secondary.cutover"},
		{"secondary without tenant", func(s string) string { return strings.Replace(s, "  tenant: default-tenant", "", 1) }, "secondary.tenant is required"},
		{"field prefix without a secondary name", func(s string) string { return strings.Replace(s, "secondary: event", "secondary: log", 1) }, "fieldPrefix"},
		{"secondary without datasets", func(s string) string { return strings.Replace(s, "  datasets: [logs, alerts]", "", 1) }, "secondary.datasets"},
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
secondary:
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

func TestADisabledSecondaryIsNeverBuiltAndTheRestOfTheSectionIsKept(t *testing.T) {
	enabled := mustParse(t, dual)
	disabled := mustParse(t, dual+"  disabled: true\n")

	if !disabled.Secondary.Disabled || disabled.Secondary.Tenant != enabled.Secondary.Tenant || disabled.Secondary.Cutover != enabled.Secondary.Cutover {
		t.Fatal("disabling must only flip the flag and leave the other settings in place")
	}

	b, err := New(disabled, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })

	if b.Composite != nil {
		t.Error("a disabled secondary must not produce a composite")
	}
	if !b.SecondaryDisabled {
		t.Error("SecondaryDisabled must say so")
	}
	if b.Store == nil {
		t.Fatal("the primary must still be served")
	}
	if _, isComposite := b.Store.(interface{ SecondaryActive() bool }); isComposite {
		t.Error("the store must be the primary alone")
	}
	if dt, ok := b.ResolveIndexPattern("logs", "v11-log-deceptive-bytes-*"); !ok || dt != "deceptive-bytes" {
		t.Errorf("v11 rules still name their indices, so the resolver must survive: (%q, %v)", dt, ok)
	}
}

func TestADisabledSecondaryNeverConnects(t *testing.T) {
	var calls [][]string
	prev := connectOpenSearch
	connectOpenSearch = func(urls []string, _, _ string) error {
		calls = append(calls, urls)
		return nil
	}
	t.Cleanup(func() { connectOpenSearch = prev })

	b, err := New(mustParse(t, dual+"  disabled: true\n"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	_ = b.Close()
	if len(calls) != 0 {
		t.Fatalf("a disabled secondary must not be connected, got %v", calls)
	}

	// The same configuration, enabled, does connect: the check above is not
	// passing because nothing ever would.
	b, err = New(mustParse(t, dual), Options{})
	if err != nil {
		t.Fatal(err)
	}
	_ = b.Close()
	if len(calls) != 1 || calls[0][0] != "https://127.0.0.1:9200" {
		t.Fatalf("an enabled secondary must connect once, got %v", calls)
	}
}

func TestADisabledSecondaryIsNotCheckedForTheSettingsItNoLongerNeeds(t *testing.T) {
	// The rest of a disabled section may be stale; it must not stop a process
	// from starting.
	stale := strings.Replace(dual, `  cutover: "2026-11-01T00:00:00Z"`, "", 1)
	stale = strings.Replace(stale, "  tenant: default-tenant", "", 1)

	if _, err := New(mustParse(t, stale), Options{}); err == nil {
		t.Fatal("while enabled, the missing settings are an error")
	}
	b, err := New(mustParse(t, stale+"  disabled: true\n"), Options{})
	if err != nil {
		t.Fatalf("once disabled they are not: %v", err)
	}
	_ = b.Close()
}

func TestTheIndexPatternResolverSurvivesWithoutASecondaryBlock(t *testing.T) {
	b, err := New(mustParse(t, single), Options{})
	if err != nil {
		t.Fatal(err)
	}
	_ = b.Close()
	if b.SecondaryDisabled {
		t.Error("no secondary block is not a disabled one")
	}
}
