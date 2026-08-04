// Package clickhouse implements store.Store over ClickHouse.
//
// One table per dataset, partitioned by tenant. The tenant predicate is added
// by this package on every operation rather than by callers, so a query that
// forgets it does not exist.
package clickhouse

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/threatwinds/go-sdk/store"
)

// Config describes where the data lives. Table names are supplied rather than
// derived: which datasets exist and what they are called is product knowledge,
// and this package holds none.
type Config struct {
	// Addr, Database and Auth build a connection when Conn is nil.
	Addr     []string
	Database string
	Username string
	Password string

	// Conn takes precedence, for callers that already hold one.
	Conn driver.Conn

	// Tables maps a dataset to its table. A dataset with no entry is an error
	// at call time rather than a silent read of nothing.
	Tables map[store.Dataset]string

	// TenantColumn carries the tenant. It has no default: a driver that
	// guessed it would silently return every tenant's rows when it guessed
	// wrong.
	TenantColumn string

	// TimeColumn bounds Scope.From/To. Defaults to "@timestamp".
	TimeColumn string

	// DataTypeColumn narrows Scope.DataType. Empty means the dataset has no
	// such dimension and a scoped DataType is an error.
	DataTypeColumn string

	// ColdVolume names the storage volume SetRetention moves data to. Defaults
	// to "cold"; it has to match the storage policy the table was created with.
	ColdVolume string

	// DialTimeout and MaxOpenConns apply only when this package builds the
	// connection.
	DialTimeout  time.Duration
	MaxOpenConns int
}

// boundLayout keeps the millisecond that DateTime64(3) stores.
const boundLayout = "2006-01-02 15:04:05.000"

type Driver struct {
	conn driver.Conn
	cfg  Config
}

var _ store.Store = (*Driver)(nil)

func New(cfg Config) (*Driver, error) {
	if cfg.TenantColumn == "" {
		return nil, errors.New("clickhouse: TenantColumn is required")
	}
	if len(cfg.Tables) == 0 {
		return nil, errors.New("clickhouse: at least one dataset table is required")
	}
	if cfg.TimeColumn == "" {
		cfg.TimeColumn = "@timestamp"
	}

	conn := cfg.Conn
	if conn == nil {
		if len(cfg.Addr) == 0 {
			return nil, errors.New("clickhouse: Addr or Conn is required")
		}
		opts := &clickhouse.Options{
			Addr: cfg.Addr,
			Auth: clickhouse.Auth{
				Database: cfg.Database,
				Username: cfg.Username,
				Password: cfg.Password,
			},
			DialTimeout:  cfg.DialTimeout,
			MaxOpenConns: cfg.MaxOpenConns,
			Compression:  &clickhouse.Compression{Method: clickhouse.CompressionLZ4},
		}
		if opts.DialTimeout == 0 {
			opts.DialTimeout = 10 * time.Second
		}
		c, err := clickhouse.Open(opts)
		if err != nil {
			return nil, fmt.Errorf("clickhouse: open: %w", err)
		}
		conn = c
	}

	return &Driver{conn: conn, cfg: cfg}, nil
}

func (d *Driver) Close() error { return d.conn.Close() }

// table resolves the dataset and refuses an unknown one. Returning an error
// rather than an empty name keeps a typo from reading nothing and looking like
// no data.
func (d *Driver) table(s store.Scope) (string, error) {
	name, ok := d.cfg.Tables[s.Dataset]
	if !ok {
		return "", fmt.Errorf("clickhouse: no table configured for dataset %q", s.Dataset)
	}
	if d.cfg.Database != "" {
		return quoteIdent(d.cfg.Database) + "." + quoteIdent(name), nil
	}
	return quoteIdent(name), nil
}

// where builds the predicate every operation shares: the tenant, the time
// bounds and the caller's filters. It is the only place a query is scoped, so
// there is no path that forgets to.
func (d *Driver) where(s store.Scope, filters []store.Filter) (string, []any, error) {
	var (
		clauses []string
		args    []any
	)

	switch s.Tenant {
	case "":
		return "", nil, store.ErrNoTenant
	case store.AllTenants:
		// Deliberately unscoped; the caller had to name it.
	default:
		clauses = append(clauses, quoteIdent(d.cfg.TenantColumn)+" = ?")
		args = append(args, s.Tenant)
	}

	if s.DataType != "" {
		if d.cfg.DataTypeColumn == "" {
			return "", nil, fmt.Errorf("clickhouse: scope has a data type but no DataTypeColumn is configured")
		}
		clauses = append(clauses, quoteIdent(d.cfg.DataTypeColumn)+" = ?")
		args = append(args, s.DataType)
	}

	if !s.From.IsZero() {
		clauses = append(clauses, quoteIdent(d.cfg.TimeColumn)+" >= toDateTime64(?, 3, 'UTC')")
		args = append(args, s.From.UTC().Format(boundLayout))
	}
	if !s.To.IsZero() {
		clauses = append(clauses, quoteIdent(d.cfg.TimeColumn)+" <= toDateTime64(?, 3, 'UTC')")
		args = append(args, s.To.UTC().Format(boundLayout))
	}

	for _, f := range filters {
		c, a, err := renderFilter(f)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, c)
		args = append(args, a...)
	}

	if len(clauses) == 0 {
		return "1", nil, nil
	}
	return strings.Join(clauses, " AND "), args, nil
}
