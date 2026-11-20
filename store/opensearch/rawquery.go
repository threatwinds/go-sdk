package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sdkos "github.com/threatwinds/go-sdk/os"
	"github.com/threatwinds/go-sdk/store"
)

const DialectSQL = "opensearch-sql"

var _ store.RawQuerier = (*Driver)(nil)

func (d *Driver) Dialect() string { return DialectSQL }

// RawQuery runs caller-supplied query text. It requires store.AllTenants
// because the driver cannot inject a tenant predicate into text it did not
// build — refusing is the only honest option, and it forces the caller to
// acknowledge that the query reads across tenants.
func (d *Driver) RawQuery(ctx context.Context, s store.Scope, query string) ([]map[string]any, error) {
	if s.Tenant == "" {
		return nil, store.ErrNoTenant
	}
	if s.Tenant != store.AllTenants {
		return nil, fmt.Errorf(
			"%w: RawQuery cannot be scoped to tenant %q; the query text must carry its own tenant predicate and be run with store.AllTenants",
			store.ErrUnsupported, s.Tenant)
	}
	return sdkos.SearchSQL(ctx, query)
}

// FetchQuery is RawQuery under the agnostic name
func (d *Driver) FetchQuery(ctx context.Context, s store.Scope, sql string) ([]json.RawMessage, error) {

	var query string = sql
	if s.Tenant != store.AllTenants {
		query = strings.TrimRight(strings.TrimSpace(sql), ";")
		query += fmt.Sprintf(" WHERE tenant_id = '%s';", s.Tenant)
	}

	rows, err := d.RawQuery(ctx, s, query)
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, 0, len(rows))
	for _, r := range rows {
		b, err := json.Marshal(r)
		if err != nil {
			return nil, fmt.Errorf("opensearch: fetch query: %w", err)
		}
		out = append(out, b)
	}
	return out, nil
}
