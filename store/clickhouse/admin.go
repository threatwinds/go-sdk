package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

// DescribeFields lists what can be queried, and that is two things joined: the
// declared columns, and the paths ClickHouse discovered inside the JSON column.
// The second half is why a record can carry fields nobody declared and still be
// filterable.
func (d *Driver) DescribeFields(ctx context.Context, s store.Scope) ([]store.Field, error) {
	tbl, ok := d.cfg.Tables[s.Dataset]
	if !ok {
		return nil, fmt.Errorf("clickhouse: no table configured for dataset %q", s.Dataset)
	}

	db := d.cfg.Database
	if db == "" {
		db = "currentDatabase()"
	}

	rows, err := d.conn.Query(ctx,
		"SELECT name, type FROM system.columns WHERE database = ? AND table = ? ORDER BY position",
		db, tbl)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: describe: %w", err)
	}
	defer rows.Close()

	var out []store.Field
	var jsonCols []string
	for rows.Next() {
		var name, typ string
		if err := rows.Scan(&name, &typ); err != nil {
			return nil, err
		}
		if strings.HasPrefix(typ, "JSON") {
			jsonCols = append(jsonCols, name)
		}
		out = append(out, store.Field{
			Name:       name,
			Type:       typ,
			Filterable: true,
			Searchable: strings.Contains(typ, "String"),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, jc := range jsonCols {
		paths, err := d.jsonPaths(ctx, s, jc)
		if err != nil {
			return nil, err
		}
		out = append(out, paths...)
	}
	return out, nil
}

// jsonPaths asks the data what shape it has. A path that appears with more than
// one type is reported as a conflict rather than being hidden, because a filter
// on it will behave differently depending on the row.
func (d *Driver) jsonPaths(ctx context.Context, s store.Scope, col string) ([]store.Field, error) {
	tbl, err := d.table(s)
	if err != nil {
		return nil, err
	}
	pred, args, err := d.where(s, nil)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf(
		"SELECT p.1 AS path, groupUniqArray(p.2) AS types FROM %s ARRAY JOIN JSONAllPathsWithTypes(%s) AS p WHERE %s GROUP BY path ORDER BY path",
		tbl, quoteIdent(col), pred,
	)
	rows, err := d.conn.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: json paths: %w", err)
	}
	defer rows.Close()

	var out []store.Field
	for rows.Next() {
		var path string
		var types []string
		if err := rows.Scan(&path, &types); err != nil {
			return nil, err
		}
		out = append(out, store.Field{
			Name:       col + "." + path,
			Type:       strings.Join(types, "|"),
			Filterable: true,
			Searchable: len(types) == 1 && strings.Contains(types[0], "String"),
			Conflict:   len(types) > 1,
		})
	}
	return out, rows.Err()
}

func (d *Driver) Usage(ctx context.Context) ([]store.DatasetUsage, error) {
	var out []store.DatasetUsage
	for dataset, tbl := range d.cfg.Tables {
		var (
			docs  uint64
			bytes uint64
		)
		err := d.conn.QueryRow(ctx,
			"SELECT sum(rows), sum(bytes_on_disk) FROM system.parts WHERE table = ? AND active",
			tbl).Scan(&docs, &bytes)
		if err != nil {
			return nil, fmt.Errorf("clickhouse: usage: %w", err)
		}
		out = append(out, store.DatasetUsage{
			Dataset:   dataset,
			Documents: int64(docs),
			Bytes:     int64(bytes),
		})
	}
	return out, nil
}

// Retention reads the TTL off the table itself, which is where it lives: in
// ClickHouse retention is part of the schema rather than a policy attached from
// outside.
func (d *Driver) Retention(ctx context.Context, dataset store.Dataset) (store.Retention, error) {
	tbl, ok := d.cfg.Tables[dataset]
	if !ok {
		return store.Retention{}, fmt.Errorf("clickhouse: no table configured for dataset %q", dataset)
	}

	var expr string
	err := d.conn.QueryRow(ctx,
		"SELECT engine_full FROM system.tables WHERE name = ? AND database = currentDatabase()",
		tbl).Scan(&expr)
	if err != nil {
		return store.Retention{}, fmt.Errorf("clickhouse: retention: %w", err)
	}

	keep, cold := parseTTL(expr)
	return store.Retention{
		Keep:      time.Duration(keep) * 24 * time.Hour,
		ColdAfter: time.Duration(cold) * 24 * time.Hour,
	}, nil
}

// SetRetention rewrites the whole TTL, because that is the only thing MODIFY TTL
// can do: it replaces the expression rather than amending it.
//
// So a flat retention asked of a table that currently moves data to cold storage
// is refused. Accepting it would drop the move clause, and nothing would report
// that: the data would simply stop leaving the hot disk until it filled.
func (d *Driver) SetRetention(ctx context.Context, dataset store.Dataset, r store.Retention) error {
	tbl, err := d.table(store.Scope{Dataset: dataset})
	if err != nil {
		return err
	}
	if r.Keep <= 0 {
		return fmt.Errorf("clickhouse: retention must be positive")
	}
	if r.ColdAfter < 0 || r.ColdAfter >= r.Keep {
		return fmt.Errorf("clickhouse: ColdAfter must be shorter than Keep")
	}

	if !r.Tiered() {
		current, err := d.Retention(ctx, dataset)
		if err != nil {
			return err
		}
		if current.Tiered() {
			return fmt.Errorf(
				"clickhouse: refusing to replace a tiered retention with a flat one; set ColdAfter to keep the move to %q",
				d.coldVolume())
		}
	}

	col := quoteIdent(d.cfg.TimeColumn)
	clauses := []string{}
	if r.Tiered() {
		clauses = append(clauses, fmt.Sprintf("toDateTime(%s) + INTERVAL %d DAY TO VOLUME %s",
			col, days(r.ColdAfter), quoteString(d.coldVolume())))
	}
	clauses = append(clauses, fmt.Sprintf("toDateTime(%s) + INTERVAL %d DAY DELETE", col, days(r.Keep)))

	q := fmt.Sprintf("ALTER TABLE %s MODIFY TTL %s", tbl, strings.Join(clauses, ", "))
	if err := d.conn.Exec(ctx, q); err != nil {
		return fmt.Errorf("clickhouse: set retention: %w", err)
	}
	return nil
}

// EnableTiering points a dataset at a storage policy that has a cold volume,
// for an install that started without one and later added object storage.
//
// It exists because the two halves cannot be done in one statement: the policy
// has to be in place before a TTL may name its volume. Doing them in the wrong
// order fails with an error about a volume that does not exist, which reads
// like a typo rather than an ordering problem.
//
// The policy itself is server configuration and this package cannot create it.
// One requirement is worth knowing before you write it: ClickHouse only accepts
// a new policy that contains every VOLUME NAME of the old one, and a table with
// no policy is on "default", whose volume is also called "default". So the hot
// volume has to be named "default" — call it "hot" and the table can never be
// migrated without being rebuilt.
func (d *Driver) EnableTiering(ctx context.Context, dataset store.Dataset, policy string, r store.Retention) error {
	if policy == "" {
		return fmt.Errorf("clickhouse: EnableTiering needs a storage policy")
	}
	if !r.Tiered() {
		return fmt.Errorf("clickhouse: EnableTiering needs a ColdAfter")
	}
	tbl, err := d.table(store.Scope{Dataset: dataset})
	if err != nil {
		return err
	}

	q := fmt.Sprintf("ALTER TABLE %s MODIFY SETTING storage_policy = %s", tbl, quoteString(policy))
	if err := d.conn.Exec(ctx, q); err != nil {
		return fmt.Errorf("clickhouse: adopting storage policy %q: %w", policy, err)
	}
	return d.SetRetention(ctx, dataset, r)
}

func (d *Driver) coldVolume() string {
	if d.cfg.ColdVolume != "" {
		return d.cfg.ColdVolume
	}
	return "cold"
}

func days(d time.Duration) int64 { return int64(d.Hours() / 24) }

func (d *Driver) Health(ctx context.Context) (store.Health, error) {
	var free, total uint64
	err := d.conn.QueryRow(ctx,
		"SELECT sum(free_space), sum(total_space) FROM system.disks").Scan(&free, &total)
	if err != nil {
		return store.Health{Status: store.HealthUnavailable, Message: err.Error()}, nil
	}

	h := store.Health{Status: store.HealthOK}
	if total > 0 {
		h.DiskUsedPct = float64(total-free) / float64(total) * 100
	}
	if h.DiskUsedPct > 90 {
		h.Status = store.HealthDegraded
		h.Message = "disk above 90%"
	}
	return h, nil
}

// Drop deletes what the scope covers and nothing else. It is a mutation, so it
// is asynchronous and the count is not knowable up front.
func (d *Driver) Drop(ctx context.Context, s store.Scope) (int64, error) {
	tbl, err := d.table(s)
	if err != nil {
		return 0, err
	}
	pred, args, err := d.where(s, nil)
	if err != nil {
		return 0, err
	}
	if pred == "1" {
		return 0, fmt.Errorf("clickhouse: refusing an unbounded drop")
	}

	q := fmt.Sprintf("ALTER TABLE %s DELETE WHERE %s", tbl, pred)
	if err := d.conn.Exec(ctx, q, args...); err != nil {
		return 0, fmt.Errorf("clickhouse: drop: %w", err)
	}
	return 0, nil
}

// parseTTL reads the two ages out of the engine definition. ClickHouse renders
// intervals as toIntervalDay(n), and the clauses are comma-separated, so each is
// read on its own rather than assuming an order.
func parseTTL(engineFull string) (keepDays, coldDays int) {
	i := strings.Index(engineFull, "TTL ")
	if i < 0 {
		return 0, 0
	}
	tail := engineFull[i+4:]
	if j := strings.Index(tail, "SETTINGS"); j >= 0 {
		tail = tail[:j]
	}

	for _, clause := range strings.Split(tail, ",") {
		n := intervalDays(clause)
		if n == 0 {
			continue
		}
		if strings.Contains(clause, "TO VOLUME") || strings.Contains(clause, "TO DISK") {
			coldDays = n
		} else {
			keepDays = n
		}
	}
	return keepDays, coldDays
}

func intervalDays(clause string) int {
	var n int
	if j := strings.Index(clause, "toIntervalDay("); j >= 0 {
		if _, err := fmt.Sscanf(clause[j:], "toIntervalDay(%d)", &n); err == nil {
			return n
		}
	}
	if j := strings.Index(clause, "INTERVAL "); j >= 0 {
		if _, err := fmt.Sscanf(clause[j:], "INTERVAL %d DAY", &n); err == nil {
			return n
		}
	}
	return 0
}

// quoteString renders a SQL string literal for the few places a value cannot be
// a bound parameter — a volume name in DDL is one of them.
func quoteString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
}
