package opensearch

import (
	"context"
	"fmt"

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
