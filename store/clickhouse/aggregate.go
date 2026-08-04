package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

func (d *Driver) TopValues(ctx context.Context, s store.Scope, field string, filters []store.Filter, n int) ([]store.Bucket, error) {
	tbl, err := d.table(s)
	if err != nil {
		return nil, err
	}
	pred, args, err := d.where(s, filters)
	if err != nil {
		return nil, err
	}
	col, err := column(field)
	if err != nil {
		return nil, err
	}
	if n <= 0 {
		n = 10
	}

	q := fmt.Sprintf(
		"SELECT toString(%s) AS k, count() AS c FROM %s WHERE %s GROUP BY k ORDER BY c DESC LIMIT ?",
		col, tbl, pred,
	)
	rows, err := d.conn.Query(ctx, q, append(args, n)...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: top values: %w", err)
	}
	defer rows.Close()

	var out []store.Bucket
	for rows.Next() {
		var b store.Bucket
		var c uint64
		if err := rows.Scan(&b.Key, &c); err != nil {
			return nil, err
		}
		b.Count = int64(c)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (d *Driver) Timeline(ctx context.Context, s store.Scope, filters []store.Filter, interval store.Interval) ([]store.Point, error) {
	tbl, err := d.table(s)
	if err != nil {
		return nil, err
	}
	pred, args, err := d.where(s, filters)
	if err != nil {
		return nil, err
	}
	bucket, err := d.bucketExpr(interval)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf(
		"SELECT %s AS b, count() AS c FROM %s WHERE %s GROUP BY b ORDER BY b",
		bucket, tbl, pred,
	)
	rows, err := d.conn.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: timeline: %w", err)
	}
	defer rows.Close()

	var out []store.Point
	for rows.Next() {
		var p store.Point
		var c uint64
		if err := rows.Scan(&p.At, &c); err != nil {
			return nil, err
		}
		p.Count = int64(c)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (d *Driver) TimelineByField(ctx context.Context, s store.Scope, field string, filters []store.Filter, interval store.Interval, n int) ([]store.Series, error) {
	top, err := d.TopValues(ctx, s, field, filters, n)
	if err != nil {
		return nil, err
	}

	// One timeline per top value rather than a single pivoted query: the set of
	// keys is not known until the first query returns, and a pivot would have
	// to name them as columns.
	out := make([]store.Series, 0, len(top))
	for _, b := range top {
		pts, err := d.Timeline(ctx, s, append(filters, store.Filter{
			Field: field, Op: store.OpEq, Value: b.Key,
		}), interval)
		if err != nil {
			return nil, err
		}
		out = append(out, store.Series{Key: b.Key, Points: pts})
	}
	return out, nil
}

func (d *Driver) GroupBy(ctx context.Context, s store.Scope, fields []string, filters []store.Filter, opts store.GroupOpts) ([]store.Group, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("clickhouse: group by needs at least one field")
	}
	tbl, err := d.table(s)
	if err != nil {
		return nil, err
	}
	pred, args, err := d.where(s, filters)
	if err != nil {
		return nil, err
	}

	col, err := column(fields[0])
	if err != nil {
		return nil, err
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	sel := fmt.Sprintf("toString(%s) AS k, count() AS c", col)
	if opts.TopHits > 0 {
		sel += fmt.Sprintf(", groupArray(%d)(formatRow('JSONEachRow', *)) AS hits", opts.TopHits)
	}

	q := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s GROUP BY k ORDER BY c DESC LIMIT ?",
		sel, tbl, pred,
	)
	rows, err := d.conn.Query(ctx, q, append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: group by: %w", err)
	}
	defer rows.Close()

	var out []store.Group
	for rows.Next() {
		g := store.Group{Field: fields[0]}
		var c uint64
		if opts.TopHits > 0 {
			var hits []string
			if err := rows.Scan(&g.Key, &c, &hits); err != nil {
				return nil, err
			}
			for _, h := range hits {
				g.Hits = append(g.Hits, json.RawMessage(h))
			}
		} else if err := rows.Scan(&g.Key, &c); err != nil {
			return nil, err
		}
		g.Count = int64(c)
		out = append(out, g)
	}
	return out, rows.Err()
}

// bucketExpr renders the timeline bucket. Calendar units go through ClickHouse's
// own functions so month and week boundaries follow the calendar rather than a
// fixed number of seconds.
func (d *Driver) bucketExpr(i store.Interval) (string, error) {
	col := quoteIdent(d.cfg.TimeColumn)
	switch i.Calendar {
	case store.CalendarHour:
		return "toStartOfHour(" + col + ")", nil
	case store.CalendarDay:
		return "toStartOfDay(" + col + ")", nil
	case store.CalendarWeek:
		return "toStartOfWeek(" + col + ")", nil
	case store.CalendarMonth:
		return "toStartOfMonth(" + col + ")", nil
	}
	if i.Duration <= 0 {
		return "", fmt.Errorf("clickhouse: interval needs a calendar unit or a positive duration")
	}
	return fmt.Sprintf("toStartOfInterval(%s, INTERVAL %d SECOND)", col, int64(i.Duration/time.Second)), nil
}
