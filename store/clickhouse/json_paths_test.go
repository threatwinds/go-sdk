package clickhouse

import (
	"strings"
	"testing"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

func pathsDriver(cfg Config) *Driver {
	cfg.Tables = map[store.Dataset]string{"logs": "logs"}
	cfg.TenantColumn = "tenantId"
	cfg.TimeColumn = "@timestamp"
	return &Driver{cfg: cfg}
}

// Path discovery has to read a sample. Expanding every record of the table into
// one row per path is what ran a 38 MiB table past a 4 GiB memory limit, so the
// query must be bounded by time and by a row count, and must choose the rows
// before it touches the JSON column.
func TestJSONPathsQueryReadsABoundedSample(t *testing.T) {
	d := pathsDriver(Config{
		IDColumns:      map[store.Dataset]string{"logs": "id"},
		DataTypeColumn: "dataType",
	})
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

	q, args, err := d.jsonPathsQuery(store.Scope{Tenant: "t1", Dataset: "logs"}, "log", now)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"PREWHERE `id` IN (SELECT `id` FROM `logs` WHERE",
		"LIMIT ? BY `dataType`",
		"ARRAY JOIN JSONAllPathsWithTypes(`log`)",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("query is missing %q:\n%s", want, q)
		}
	}

	// tenant, window start, sample size, then the same tenant and window start
	// for the outer read.
	wantFrom := now.Add(-jsonPathsWindow).Format(boundLayout)
	if len(args) != 5 || args[0] != "t1" || args[1] != wantFrom || args[2] != jsonPathsSample ||
		args[3] != "t1" || args[4] != wantFrom {
		t.Errorf("args = %v", args)
	}
}

func TestJSONPathsQueryKeepsAWindowTheScopeSets(t *testing.T) {
	d := pathsDriver(Config{})
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	q, args, err := d.jsonPathsQuery(store.Scope{Tenant: "t1", Dataset: "logs", From: from},
		"log", time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(q, "PREWHERE") || !strings.HasSuffix(strings.Split(q, ") ARRAY JOIN")[0], "LIMIT ?") {
		t.Errorf("a dataset with no id column should be read by a plain bounded LIMIT:\n%s", q)
	}
	if len(args) != 3 || args[1] != from.Format(boundLayout) {
		t.Errorf("args = %v, want the scope's own start", args)
	}
}

func TestJSONPathsQueryRefusesAnUnscopedTenant(t *testing.T) {
	d := pathsDriver(Config{})
	if _, _, err := d.jsonPathsQuery(store.Scope{Dataset: "logs"}, "log", time.Now()); err == nil {
		t.Fatal("a query with no tenant must not be built")
	}
}
