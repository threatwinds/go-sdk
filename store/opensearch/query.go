package opensearch

import (
	"fmt"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

// buildQuery is the only place the tenant term is added, so no caller can skip it.
func (d *Driver) buildQuery(s store.Scope, filters []store.Filter) (map[string]any, error) {
	if s.Tenant == "" {
		return nil, store.ErrNoTenant
	}

	must := make([]map[string]any, 0, len(filters)+2)
	mustNot := make([]map[string]any, 0, len(filters))

	if s.Tenant != store.AllTenants {
		must = append(must, map[string]any{"term": map[string]any{d.cfg.TenantField: s.Tenant}})
	}
	if r := d.timeRange(s.From, s.To); r != nil {
		must = append(must, r)
	}

	for _, f := range filters {
		clause, negate, err := filterClause(f)
		if err != nil {
			return nil, err
		}
		if negate {
			mustNot = append(mustNot, clause)
		} else {
			must = append(must, clause)
		}
	}

	b := map[string]any{"filter": must}
	if len(mustNot) > 0 {
		b["must_not"] = mustNot
	}
	return map[string]any{"bool": b}, nil
}

func (d *Driver) timeRange(from, to time.Time) map[string]any {
	bounds := map[string]any{}
	if !from.IsZero() {
		bounds["gte"] = from.UTC().Format(time.RFC3339)
	}
	if !to.IsZero() {
		bounds["lte"] = to.UTC().Format(time.RFC3339)
	}
	if len(bounds) == 0 {
		return nil
	}
	return map[string]any{"range": map[string]any{d.cfg.TimestampField: bounds}}
}

// filterClause reports whether the clause belongs in must_not rather than filter.
func filterClause(f store.Filter) (map[string]any, bool, error) {
	if f.Field == "" {
		return nil, false, fmt.Errorf("opensearch: filter with empty field")
	}
	switch f.Op {
	case store.OpEq:
		return map[string]any{"term": map[string]any{f.Field: f.Value}}, false, nil
	case store.OpNotEq:
		return map[string]any{"term": map[string]any{f.Field: f.Value}}, true, nil
	case store.OpIn, store.OpNotIn:
		vals, err := toSlice(f.Value)
		if err != nil {
			return nil, false, err
		}
		return map[string]any{"terms": map[string]any{f.Field: vals}}, f.Op == store.OpNotIn, nil
	case store.OpGt:
		return rangeClause(f.Field, "gt", f.Value), false, nil
	case store.OpGte:
		return rangeClause(f.Field, "gte", f.Value), false, nil
	case store.OpLt:
		return rangeClause(f.Field, "lt", f.Value), false, nil
	case store.OpLte:
		return rangeClause(f.Field, "lte", f.Value), false, nil
	case store.OpBetween:
		vals, err := toSlice(f.Value)
		if err != nil || len(vals) != 2 {
			return nil, false, fmt.Errorf("opensearch: OpBetween on %q needs exactly two bounds", f.Field)
		}
		return map[string]any{"range": map[string]any{
			f.Field: map[string]any{"gte": vals[0], "lte": vals[1]},
		}}, false, nil
	case store.OpContains:
		return map[string]any{"match": map[string]any{f.Field: f.Value}}, false, nil
	case store.OpNotContains:
		return map[string]any{"match": map[string]any{f.Field: f.Value}}, true, nil
	case store.OpExists:
		return map[string]any{"exists": map[string]any{"field": f.Field}}, false, nil
	default:
		return nil, false, fmt.Errorf("%w: operator %q", store.ErrUnsupported, f.Op)
	}
}

func rangeClause(field, op string, v any) map[string]any {
	return map[string]any{"range": map[string]any{field: map[string]any{op: v}}}
}

func toSlice(v any) ([]any, error) {
	switch t := v.(type) {
	case []any:
		return t, nil
	case []string:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out, nil
	case []int:
		out := make([]any, len(t))
		for i, n := range t {
			out[i] = n
		}
		return out, nil
	case []int64:
		out := make([]any, len(t))
		for i, n := range t {
			out[i] = n
		}
		return out, nil
	default:
		return nil, fmt.Errorf("opensearch: expected a slice, got %T", v)
	}
}

func intervalSpec(iv store.Interval) (string, string, error) {
	if iv.Calendar != store.CalendarNone {
		return "calendar_interval", string(iv.Calendar), nil
	}
	d := iv.Duration
	switch {
	case d <= 0:
		return "", "", fmt.Errorf("opensearch: interval needs a positive Duration or a Calendar unit")
	case d%(24*time.Hour) == 0:
		return "fixed_interval", fmt.Sprintf("%dd", int(d/(24*time.Hour))), nil
	case d%time.Hour == 0:
		return "fixed_interval", fmt.Sprintf("%dh", int(d/time.Hour)), nil
	case d%time.Minute == 0:
		return "fixed_interval", fmt.Sprintf("%dm", int(d/time.Minute)), nil
	default:
		return "fixed_interval", fmt.Sprintf("%ds", int(d/time.Second)), nil
	}
}

// aggBuckets treats a missing aggregation as no buckets: an empty result set
// legitimately produces one.
func aggBuckets(aggs map[string]any, name string) []map[string]any {
	node, ok := aggs[name].(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := node["buckets"].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, b := range raw {
		if m, ok := b.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func bucketKey(b map[string]any) string {
	if s, ok := b["key_as_string"].(string); ok {
		return s
	}
	switch k := b["key"].(type) {
	case string:
		return k
	case float64:
		return fmt.Sprintf("%.0f", k)
	default:
		return fmt.Sprint(k)
	}
}

func bucketCount(b map[string]any) int64 {
	if c, ok := b["doc_count"].(float64); ok {
		return int64(c)
	}
	return 0
}

func bucketTime(b map[string]any) time.Time {
	if ms, ok := b["key"].(float64); ok {
		return time.UnixMilli(int64(ms)).UTC()
	}
	return time.Time{}
}
