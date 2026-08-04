package clickhouse

import (
	"fmt"
	"strings"

	"github.com/threatwinds/go-sdk/store"
)

// renderFilter turns one predicate into SQL. Values always become bound
// parameters; only the field name is interpolated, and quoteIdent is what makes
// that safe.
func renderFilter(f store.Filter) (string, []any, error) {
	if f.Field == "" {
		return "", nil, fmt.Errorf("clickhouse: filter has no field")
	}
	col, err := column(f.Field)
	if err != nil {
		return "", nil, err
	}

	switch f.Op {
	case store.OpEq:
		return col + " = ?", []any{f.Value}, nil
	case store.OpNotEq:
		return col + " != ?", []any{f.Value}, nil

	case store.OpGt:
		return col + " > ?", []any{f.Value}, nil
	case store.OpGte:
		return col + " >= ?", []any{f.Value}, nil
	case store.OpLt:
		return col + " < ?", []any{f.Value}, nil
	case store.OpLte:
		return col + " <= ?", []any{f.Value}, nil

	case store.OpIn, store.OpNotIn:
		vals, err := toSlice(f.Value)
		if err != nil {
			return "", nil, err
		}
		if len(vals) == 0 {
			// An empty IN matches nothing and an empty NOT IN matches
			// everything. Saying so explicitly beats emitting "IN ()", which
			// ClickHouse rejects.
			if f.Op == store.OpIn {
				return "0", nil, nil
			}
			return "1", nil, nil
		}
		op := "IN"
		if f.Op == store.OpNotIn {
			op = "NOT IN"
		}
		return fmt.Sprintf("%s %s (%s)", col, op, placeholders(len(vals))), vals, nil

	case store.OpBetween:
		lo, hi, err := toPair(f.Value)
		if err != nil {
			return "", nil, err
		}
		return col + " BETWEEN ? AND ?", []any{lo, hi}, nil

	case store.OpContains:
		return "positionCaseInsensitive(toString(" + col + "), ?) > 0", []any{f.Value}, nil
	case store.OpNotContains:
		return "positionCaseInsensitive(toString(" + col + "), ?) = 0", []any{f.Value}, nil

	case store.OpExists:
		// A JSON path that was never written reads as NULL; a plain column
		// reads as its zero value, so emptiness is the closest shared meaning.
		return col + " IS NOT NULL", nil, nil

	default:
		return "", nil, fmt.Errorf("clickhouse: unsupported operator %q", f.Op)
	}
}

// column renders a field reference. A dotted name addresses a JSON subcolumn —
// log.event_id becomes `log`.`event_id` — which is how the dynamic half of a
// record stays queryable without being declared.
func column(field string) (string, error) {
	parts := strings.Split(field, ".")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			return "", fmt.Errorf("clickhouse: malformed field %q", field)
		}
		out = append(out, quoteIdent(p))
	}
	return strings.Join(out, "."), nil
}

// quoteIdent backtick-quotes an identifier and doubles any backtick inside it,
// which is what keeps a field name from closing the quote and becoming SQL.
// Field names reach this from rule definitions and API callers, so it is the
// boundary that matters.
func quoteIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
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
	case nil:
		return nil, nil
	default:
		return []any{v}, nil
	}
}

func toPair(v any) (any, any, error) {
	switch t := v.(type) {
	case [2]any:
		return t[0], t[1], nil
	case []any:
		if len(t) != 2 {
			return nil, nil, fmt.Errorf("clickhouse: between needs two values, got %d", len(t))
		}
		return t[0], t[1], nil
	default:
		return nil, nil, fmt.Errorf("clickhouse: between needs a pair, got %T", v)
	}
}
