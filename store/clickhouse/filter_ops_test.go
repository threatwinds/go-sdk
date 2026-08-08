package clickhouse

import (
	"strings"
	"testing"

	"github.com/threatwinds/go-sdk/store"
)

// A prefix or a suffix is not a "contains". Rendering either as the other
// returns rows the caller did not ask for, and the response says nothing.
func TestAnchoredMatchingIsAnchored(t *testing.T) {
	cases := map[store.Op]string{
		store.OpStartsWith:    "startsWith",
		store.OpNotStartsWith: "NOT startsWith",
		store.OpEndsWith:      "endsWith",
		store.OpNotEndsWith:   "NOT endsWith",
	}

	for op, want := range cases {
		sql, args, err := renderFilter(store.Filter{Field: "host", Op: op, Value: "web-"}, "raw")
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		if !strings.Contains(sql, want) {
			t.Errorf("%s rendered %q, want it to use %s", op, sql, want)
		}
		if strings.Contains(sql, "position") {
			t.Errorf("%s rendered as a substring match: %q", op, sql)
		}
		if len(args) != 1 {
			t.Errorf("%s bound %d args, want the value bound", op, len(args))
		}
	}
}

func TestTheNegationsNegate(t *testing.T) {
	notExists, _, err := renderFilter(store.Filter{Field: "log.user", Op: store.OpNotExists}, "raw")
	if err != nil {
		t.Fatalf("not_exists: %v", err)
	}
	if !strings.Contains(notExists, "IS NULL") {
		t.Errorf("not_exists rendered %q", notExists)
	}

	notBetween, args, err := renderFilter(store.Filter{
		Field: "severity", Op: store.OpNotBetween, Value: []any{1, 3},
	}, "raw")
	if err != nil {
		t.Fatalf("not_between: %v", err)
	}
	if !strings.Contains(notBetween, "NOT BETWEEN") {
		t.Errorf("not_between rendered %q", notBetween)
	}
	if len(args) != 2 {
		t.Errorf("not_between bound %d args, want 2", len(args))
	}
}

// A search reads the record, not a field, so it is the one filter with nothing
// to name. A dataset with no text to search says so rather than quietly
// matching nothing.
func TestSearchReadsTheRecord(t *testing.T) {
	sql, args, err := renderFilter(store.Filter{Op: store.OpSearch, Value: "denied"}, "raw")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(sql, "`raw`") || len(args) != 1 {
		t.Errorf("search rendered %q with %d args", sql, len(args))
	}

	if _, _, err := renderFilter(store.Filter{Op: store.OpSearch, Value: "x"}, ""); err == nil {
		t.Error("a dataset with no text column accepted a search")
	}
}

func TestContainsAnyTakesAList(t *testing.T) {
	sql, args, err := renderFilter(store.Filter{
		Field: "message", Op: store.OpContainsAny, Value: []any{"denied", "refused"},
	}, "raw")
	if err != nil {
		t.Fatalf("contains_any: %v", err)
	}
	if !strings.Contains(sql, "multiSearchAnyCaseInsensitive") {
		t.Errorf("rendered %q", sql)
	}
	if len(args) != 1 {
		t.Errorf("bound %d args, want the list bound as one", len(args))
	}

	// An empty list matches nothing, which beats emitting a call with no needles.
	sql, _, err = renderFilter(store.Filter{Field: "m", Op: store.OpContainsAny, Value: []any{}}, "raw")
	if err != nil || sql != "0" {
		t.Errorf("empty list rendered %q (%v), want a false predicate", sql, err)
	}
}

// The field name is the only part interpolated, so it stays the boundary that
// matters however an operator renders.
func TestTheFieldIsStillQuoted(t *testing.T) {
	sql, _, err := renderFilter(store.Filter{Field: "we`ird", Op: store.OpStartsWith, Value: "x"}, "raw")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(sql, "we``ird") {
		t.Errorf("field was not quoted: %q", sql)
	}
}

// IN is the one comparison ClickHouse refuses on a JSON subcolumn: the column
// reads as Dynamic and the set needs a concrete type. Without the cast every
// "field is one of these" filter fails at query time — the second most common
// shape the API speaks.
func TestInCastsAJSONSubcolumn(t *testing.T) {
	sql, args, err := renderFilter(store.Filter{
		Field: "log.eventID", Op: store.OpIn, Value: []any{"4624", "4672"},
	}, "raw")
	if err != nil {
		t.Fatalf("renderFilter: %v", err)
	}
	if !strings.Contains(sql, "toString(`log`.`eventID`) IN") {
		t.Errorf("rendered %q, want the subcolumn cast before IN", sql)
	}
	if len(args) != 2 {
		t.Fatalf("args = %v, want the two set members", args)
	}
}

// A declared column is usually in the sort key, and wrapping one in a function
// is how an index stops being read. The cast belongs to the JSON half only.
func TestInLeavesADeclaredColumnAlone(t *testing.T) {
	sql, _, err := renderFilter(store.Filter{
		Field: "dataType", Op: store.OpIn, Value: []any{"wineventlog"},
	}, "raw")
	if err != nil {
		t.Fatalf("renderFilter: %v", err)
	}
	if strings.Contains(sql, "toString") {
		t.Errorf("rendered %q: a declared column was cast, defeating its index", sql)
	}
}

// Both sides have to be String or the set has no common supertype, so a filter
// written with numbers must still render against the cast column.
func TestInStringifiesItsSetForAJSONSubcolumn(t *testing.T) {
	_, args, err := renderFilter(store.Filter{
		Field: "log.eventID", Op: store.OpIn, Value: []any{4624, 4672},
	}, "raw")
	if err != nil {
		t.Fatalf("renderFilter: %v", err)
	}
	for _, a := range args {
		if _, ok := a.(string); !ok {
			t.Errorf("arg %v is %T, want a string to match the cast column", a, a)
		}
	}
}

// Ordering is not string comparison. Casting these would make "9" > "10", which
// is a wrong answer rather than an error — the kind nobody goes looking for.
func TestOrderingOperatorsAreNotCast(t *testing.T) {
	for _, op := range []store.Op{store.OpGt, store.OpGte, store.OpLt, store.OpLte} {
		sql, _, err := renderFilter(store.Filter{Field: "log.bytes", Op: op, Value: 100}, "raw")
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		if strings.Contains(sql, "toString") {
			t.Errorf("%s rendered %q: a numeric comparison became lexicographic", op, sql)
		}
	}
}
