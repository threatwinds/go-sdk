package clickhouse

import (
	"fmt"
	"strings"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

// renderFilter turns one predicate into SQL. Values always become bound
// parameters; only the field name is interpolated, and quoteIdent is what makes
// that safe.
func renderFilter(f store.Filter, textCol string) (string, []any, error) {
	// A search reads the whole record, so it is the one filter that does not
	// name a field.
	switch f.Op {
	case store.OpSearch, store.OpNotSearch:
		if textCol == "" {
			return "", nil, fmt.Errorf("clickhouse: this dataset has no text to search")
		}
		col := quoteIdent(textCol)
		if f.Op == store.OpNotSearch {
			return "positionCaseInsensitive(" + col + ", ?) = 0", []any{f.Value}, nil
		}
		return "positionCaseInsensitive(" + col + ", ?) > 0", []any{f.Value}, nil
	}

	if f.Field == "" {
		return "", nil, fmt.Errorf("clickhouse: filter has no field")
	}
	col, err := column(f.Field)
	if err != nil {
		return "", nil, err
	}

	switch f.Op {
	// A JSON path can hold a different type on a different row — ClickHouse
	// stores it as Dynamic rather than forcing one type on insert — and
	// comparing a Dynamic column to a literal errors (NO_COMMON_TYPE /
	// TYPE_MISMATCH) the moment any row being scanned holds a different type
	// than the one being compared against, not just the rows that do. Eq and
	// NotEq sidestep it exactly like In/NotIn already do below: both sides
	// become text, so every row's actual type reads as a value rather than
	// failing the query some other row is in.
	case store.OpEq, store.OpNotEq:
		op := "="
		if f.Op == store.OpNotEq {
			op = "!="
		}
		if isJSONPath(f.Field) {
			return "toString(" + col + ") " + op + " ?", []any{toStrings([]any{f.Value})[0]}, nil
		}
		return col + " " + op + " ?", []any{f.Value}, nil

	// Ordering can't take the toString shortcut — "10" < "9" alphabetically
	// is not what a numeric or date range means — so this narrows a JSON path
	// to the Dynamic type the filter's own value is (dynJSONType), before
	// comparing. A row where the path is some other type reads as NULL from
	// dynamicElement and so never matches the range, which is what an
	// operator filtering by range wants — not every row that isn't a number
	// failing the whole query for everyone.
	case store.OpGt:
		return rangeColumn(f.Field, col, f.Value) + " > ?", []any{f.Value}, nil
	case store.OpGte:
		return rangeColumn(f.Field, col, f.Value) + " >= ?", []any{f.Value}, nil
	case store.OpLt:
		return rangeColumn(f.Field, col, f.Value) + " < ?", []any{f.Value}, nil
	case store.OpLte:
		return rangeColumn(f.Field, col, f.Value) + " <= ?", []any{f.Value}, nil

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

		if isJSONPath(f.Field) {
			return fmt.Sprintf("toString(%s) %s (%s)", col, op, placeholders(len(vals))), toStrings(vals), nil
		}
		return fmt.Sprintf("%s %s (%s)", col, op, placeholders(len(vals))), vals, nil

	case store.OpBetween, store.OpNotBetween:
		lo, hi, err := toPair(f.Value)
		if err != nil {
			return "", nil, err
		}
		rc := rangeColumn(f.Field, col, lo)
		if f.Op == store.OpNotBetween {
			return rc + " NOT BETWEEN ? AND ?", []any{lo, hi}, nil
		}
		return rc + " BETWEEN ? AND ?", []any{lo, hi}, nil

	case store.OpContains:
		return "positionCaseInsensitive(toString(" + col + "), ?) > 0", []any{f.Value}, nil
	case store.OpNotContains:
		return "positionCaseInsensitive(toString(" + col + "), ?) = 0", []any{f.Value}, nil

	// Anchored matching is case-insensitive like Contains is, so a filter built
	// in a UI behaves the same whichever of the two the user picked.
	case store.OpStartsWith:
		return "startsWith(lower(toString(" + col + ")), lower(?))", []any{f.Value}, nil
	case store.OpNotStartsWith:
		return "NOT startsWith(lower(toString(" + col + ")), lower(?))", []any{f.Value}, nil
	case store.OpEndsWith:
		return "endsWith(lower(toString(" + col + ")), lower(?))", []any{f.Value}, nil
	case store.OpNotEndsWith:
		return "NOT endsWith(lower(toString(" + col + ")), lower(?))", []any{f.Value}, nil

	case store.OpContainsAny, store.OpNotContainsAny:
		vals, err := toSlice(f.Value)
		if err != nil {
			return "", nil, err
		}
		if len(vals) == 0 {
			if f.Op == store.OpContainsAny {
				return "0", nil, nil
			}
			return "1", nil, nil
		}
		expr := "multiSearchAnyCaseInsensitive(toString(" + col + "), ?)"
		if f.Op == store.OpNotContainsAny {
			expr = "NOT " + expr
		}
		return expr, []any{vals}, nil

	case store.OpExists:
		// A JSON path that was never written reads as NULL; a plain column
		// reads as its zero value, so emptiness is the closest shared meaning.
		return col + " IS NOT NULL", nil, nil
	case store.OpNotExists:
		return col + " IS NULL", nil, nil

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

func isJSONPath(field string) bool { return strings.Contains(field, ".") }

// rangeColumn is what an ordered comparison (Gt/Gte/Lt/Lte/Between) reads. A
// plain column already only ever holds one type, so it is its own value. A
// JSON path is narrowed to the Dynamic type sample is, so ClickHouse compares
// like against like instead of raising NO_COMMON_TYPE the moment the scanned
// rows hold more than one type for that path — a row where the path is some
// other type reads as NULL and just does not match, which is what a range
// filter means by a row not qualifying. sample is unrecognized for a handful
// of Go types (see dynJSONType); the column is left bare for those, same as
// before this existed.
func rangeColumn(field, col string, sample any) string {
	if !isJSONPath(field) {
		return col
	}
	typ, ok := dynJSONType(sample)
	if !ok {
		return col
	}
	return "dynamicElement(" + col + ", '" + typ + "')"
}

// dynJSONType names the ClickHouse Dynamic type that holds v, for the Go
// types a filter's Value realistically arrives as: JSON-decoded (float64,
// string, bool), Go-native integers, and time.Time for date ranges.
func dynJSONType(v any) (string, bool) {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "Int64", true
	case float32, float64:
		return "Float64", true
	case bool:
		return "Bool", true
	case time.Time:
		return "DateTime64(3)", true
	case string:
		return "String", true
	default:
		return "", false
	}
}

func toStrings(vals []any) []any {
	out := make([]any, len(vals))
	for i, v := range vals {
		if s, ok := v.(string); ok {
			out[i] = s
			continue
		}
		out[i] = fmt.Sprint(v)
	}
	return out
}
