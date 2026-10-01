package clickhouse

import (
	"strings"
	"testing"
	"time"

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

// Ordering is not string comparison — toString(...) would make "9" > "10",
// a wrong answer rather than an error, the kind nobody goes looking for. A
// JSON path still needs narrowing to one type, the same reason In/Eq do, so
// this reaches for dynamicElement instead: the filter value's own Go type
// says which one, and a row holding a different type there just does not
// match, rather than failing the query for every row being scanned.
func TestOrderingOperatorsAreNotCastButAreNarrowedOnAJSONPath(t *testing.T) {
	for _, op := range []store.Op{store.OpGt, store.OpGte, store.OpLt, store.OpLte} {
		sql, _, err := renderFilter(store.Filter{Field: "log.bytes", Op: op, Value: 100}, "raw")
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		if strings.Contains(sql, "toString") {
			t.Errorf("%s rendered %q: a numeric comparison became lexicographic", op, sql)
		}
		if !strings.Contains(sql, "dynamicElement(`log`.`bytes`, 'Int64')") {
			t.Errorf("%s rendered %q, want the subcolumn narrowed to Int64 before comparing", op, sql)
		}
	}
}

// A declared column is usually in the sort key; dynamicElement belongs to the
// JSON half only, same as toString does for Eq/In.
func TestOrderingOperatorsLeaveADeclaredColumnAlone(t *testing.T) {
	sql, _, err := renderFilter(store.Filter{Field: "impactScore", Op: store.OpGt, Value: 5}, "raw")
	if err != nil {
		t.Fatalf("renderFilter: %v", err)
	}
	if strings.Contains(sql, "dynamicElement") {
		t.Errorf("rendered %q: a declared column was narrowed, defeating its index", sql)
	}
}

// Between reads two Go values but ClickHouse compares against one type; the
// low end of the pair is what decides it.
func TestBetweenNarrowsAJSONSubcolumnToTheLowValuesType(t *testing.T) {
	sql, args, err := renderFilter(store.Filter{
		Field: "log.durationMs", Op: store.OpBetween, Value: []any{10, 500},
	}, "raw")
	if err != nil {
		t.Fatalf("renderFilter: %v", err)
	}
	if !strings.Contains(sql, "dynamicElement(`log`.`durationMs`, 'Int64') BETWEEN") {
		t.Errorf("rendered %q, want the subcolumn narrowed before BETWEEN", sql)
	}
	if len(args) != 2 {
		t.Fatalf("args = %v, want the pair bound", args)
	}
}

// A timestamp path doesn't get dynamicElement at all: which Dynamic type
// name ClickHouse picked for it (DateTime, DateTime64(3), DateTime64(9), ...)
// depends on the precision the source sent, and dynamicElement reads every
// row as NULL the moment the name it's given doesn't match that exactly —
// confirmed live against dev-18, where a real path came back DateTime and a
// hardcoded DateTime64(3) silently matched zero rows instead of the real 150.
// Reparsing the text form sidesteps having to guess the precision.
func TestOrderingOnATimestampJSONPathReparsesInsteadOfGuessingThePrecision(t *testing.T) {
	sql, args, err := renderFilter(store.Filter{
		Field: "log.lastSeen", Op: store.OpGt, Value: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}, "raw")
	if err != nil {
		t.Fatalf("renderFilter: %v", err)
	}
	if strings.Contains(sql, "dynamicElement") {
		t.Errorf("rendered %q: a timestamp path was narrowed by a guessed Dynamic type name", sql)
	}
	if !strings.Contains(sql, "parseDateTimeBestEffortOrNull(toString(`log`.`lastSeen`))") {
		t.Errorf("rendered %q, want the subcolumn reparsed from its text form", sql)
	}
	if len(args) != 1 {
		t.Fatalf("args = %v, want the timestamp bound", args)
	}
}

// The same NO_COMMON_TYPE / TYPE_MISMATCH that made In cast its column and
// set applies to a bare Eq/NotEq on a JSON path: this is that same fix,
// applied to the operator an API actually calls the most.
func TestEqAndNotEqCastAJSONSubcolumn(t *testing.T) {
	for op, want := range map[store.Op]string{store.OpEq: "=", store.OpNotEq: "!="} {
		sql, args, err := renderFilter(store.Filter{Field: "log.pepe", Op: op, Value: 3.14}, "raw")
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		if !strings.Contains(sql, "toString(`log`.`pepe`) "+want) {
			t.Errorf("%s rendered %q, want the subcolumn cast before %s", op, sql, want)
		}
		if len(args) != 1 {
			t.Fatalf("%s args = %v, want the value bound", op, args)
		}
		if _, ok := args[0].(string); !ok {
			t.Errorf("%s bound %v (%T), want it stringified to match the cast column", op, args[0], args[0])
		}
	}
}

// A declared column already has one type; casting it away is what In already
// avoids, and Eq/NotEq must avoid it the same way.
func TestEqAndNotEqLeaveADeclaredColumnAlone(t *testing.T) {
	sql, args, err := renderFilter(store.Filter{Field: "severity", Op: store.OpEq, Value: "high"}, "raw")
	if err != nil {
		t.Fatalf("renderFilter: %v", err)
	}
	if strings.Contains(sql, "toString") {
		t.Errorf("rendered %q: a declared column was cast, defeating its index", sql)
	}
	if len(args) != 1 || args[0] != "high" {
		t.Errorf("args = %v, want the value bound as given", args)
	}
}
