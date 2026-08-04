package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

func (d *Driver) Count(ctx context.Context, s store.Scope, filters []store.Filter) (int64, error) {
	tbl, err := d.table(s)
	if err != nil {
		return 0, err
	}
	pred, args, err := d.where(s, filters)
	if err != nil {
		return 0, err
	}

	var n uint64
	q := fmt.Sprintf("SELECT count() FROM %s WHERE %s", tbl, pred)
	if err := d.conn.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("clickhouse: count: %w", err)
	}
	return int64(n), nil
}

// CountInWindow counts a trailing window ending at Scope.To, or at now when it
// is unset. The bounds are pinned onto the scope so a caller that counts and
// then fetches sees the same window for both.
func (d *Driver) CountInWindow(ctx context.Context, s store.Scope, filters []store.Filter, window time.Duration) (int64, error) {
	if window <= 0 {
		return 0, fmt.Errorf("clickhouse: window must be positive")
	}
	if s.To.IsZero() {
		s.To = time.Now().UTC()
	}
	s.From = s.To.Add(-window)
	return d.Count(ctx, s, filters)
}

func (d *Driver) FetchByID(ctx context.Context, s store.Scope, id string) (json.RawMessage, error) {
	rows, err := d.FetchN(ctx, s, []store.Filter{{Field: "id", Op: store.OpEq, Value: id}}, 1)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, store.ErrNotFound
	}
	return rows[0], nil
}

func (d *Driver) FetchN(ctx context.Context, s store.Scope, filters []store.Filter, n int) ([]json.RawMessage, error) {
	return d.fetch(ctx, s, filters, store.Page{Limit: n, SortBy: d.cfg.TimeColumn, Order: store.Desc})
}

func (d *Driver) FetchPage(ctx context.Context, s store.Scope, filters []store.Filter, page store.Page) ([]json.RawMessage, int64, error) {
	total, err := d.Count(ctx, s, filters)
	if err != nil {
		return nil, 0, err
	}
	rows, err := d.fetch(ctx, s, filters, page)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// fetch returns whole records as JSON. Selecting the row rather than named
// columns is what lets a record keep the dynamic half of its shape without this
// package knowing any of it.
func (d *Driver) fetch(ctx context.Context, s store.Scope, filters []store.Filter, page store.Page) ([]json.RawMessage, error) {
	tbl, err := d.table(s)
	if err != nil {
		return nil, err
	}
	pred, args, err := d.where(s, filters)
	if err != nil {
		return nil, err
	}

	limit := page.Limit
	if limit <= 0 {
		limit = 100
	}

	sortBy := page.SortBy
	if sortBy == "" {
		sortBy = d.cfg.TimeColumn
	}
	sortCol, err := column(sortBy)
	if err != nil {
		return nil, err
	}
	order := "DESC"
	if page.Order == store.Asc {
		order = "ASC"
	}

	// formatRow serialises the whole row, declared columns and discovered JSON
	// paths alike, so this package never has to know a record's shape.
	q := fmt.Sprintf(
		"SELECT formatRow('JSONEachRow', *) FROM %s WHERE %s ORDER BY %s %s LIMIT ? OFFSET ?",
		tbl, pred, sortCol, order,
	)
	args = append(args, limit, page.Offset)

	rows, err := d.conn.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: fetch: %w", err)
	}
	defer rows.Close()

	out := make([]json.RawMessage, 0, limit)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("clickhouse: scan: %w", err)
		}
		out = append(out, json.RawMessage(raw))
	}
	return out, rows.Err()
}
