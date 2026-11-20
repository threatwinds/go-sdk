package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/threatwinds/go-sdk/store"
)

// FetchQuery runs caller-supplied ClickHouse SQL. It requires store.AllTenants
// because the driver cannot inject a tenant predicate into text it did not
// build — an aggregate query returns rows with no tenant column left to
// filter — so the caller must carry the predicate and acknowledge it is
// reading across tenants.
func (d *Driver) FetchQuery(ctx context.Context, s store.Scope, sql string) ([]json.RawMessage, error) {

	roctx := clickhouse.Context(ctx, clickhouse.WithSettings(clickhouse.Settings{
		"readonly":           1,
		"max_execution_time": 30,
	}))

	if s.Tenant == "" {
		return nil, store.ErrNoTenant
	}
	if s.Tenant != store.AllTenants {
		return nil, fmt.Errorf(
			"%w: FetchQuery cannot be scoped to tenant %q; the query must carry its own tenant predicate and be run with store.AllTenants",
			store.ErrUnsupported, s.Tenant)
	}

	q := fmt.Sprintf("SELECT formatRow('JSONEachRow', *) FROM (%s) AS r",
		strings.TrimRight(strings.TrimSpace(sql), ";"))
	return d.scanRows(roctx, q, nil, 0)
}
