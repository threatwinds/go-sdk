package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/threatwinds/go-sdk/store"
	"github.com/tidwall/gjson"
)

// sampleSize is how many matching records come back as evidence when a
// correlation fires. It is what the query builder defaulted to.
const sampleSize = 10

// operators maps the rule vocabulary, which is a contract with rule authors,
// onto store operators.
var operators = map[string]store.Op{
	"filter_term":    store.OpEq,
	"must_not_term":  store.OpNotEq,
	"filter_match":   store.OpContains,
	"must_not_match": store.OpNotContains,
}

// Normalize ensures that the newest fields are populated even if old aliases were used
// This is now a method on the generated Rule struct from plugins.pb.go
func (r *Rule) Normalize() {
	if len(r.Correlation) == 0 && len(r.AfterEvents) > 0 {
		r.Correlation = r.AfterEvents
	}
	if r.Impact == nil {
		r.Impact = &Impact{}
	}
}

// IndexPatternResolver turns the index pattern a v11 rule names into the
// dataType a store reads it by. dataType is empty for a pattern that covers
// every dataType; ok is false for one the store does not recognise.
type IndexPatternResolver func(dataset store.Dataset, pattern string) (dataType string, ok bool)

var indexPatternResolver atomic.Pointer[IndexPatternResolver]

// SetIndexPatternResolver installs the translation ExecuteStore uses for rules
// that name an indexPattern instead of a dataType. Without one such a rule is
// refused, since guessing a store's naming would read the wrong data.
func SetIndexPatternResolver(r IndexPatternResolver) {
	if r == nil {
		indexPatternResolver.Store(nil)
		return
	}
	indexPatternResolver.Store(&r)
}

// ExecuteStore reports whether at least Count records match within Within, and if so
// returns up to sampleSize of them as evidence. Counting and sampling are two
// calls so that records are read only when the threshold is actually met.
func (e *SearchRequest) ExecuteStore(ctx context.Context, st store.Store, scope store.Scope, previous *string) (bool, []json.RawMessage, error) {
	filters, err := e.filters(previous)
	if err != nil {
		return false, nil, err
	}

	var window time.Duration
	if e.Within != "" {
		// An unparseable window used to be ignored, which silently widened the
		// search to all of history.
		if window, err = time.ParseDuration(e.Within); err != nil || window <= 0 {
			return false, nil, fmt.Errorf("correlation: invalid 'within' %q", e.Within)
		}
	}

	// Correlating against another source is opt-in; by default a rule looks at
	// the same dataType that triggered it.
	switch {
	case e.DataType != "":
		scope.DataType = e.DataType
	case e.IndexPattern != "":
		r := indexPatternResolver.Load()
		if r == nil {
			return false, nil, fmt.Errorf("correlation: the rule reads index pattern %q but no resolver is installed", e.IndexPattern)
		}
		dt, ok := (*r)(scope.Dataset, e.IndexPattern)
		if !ok {
			return false, nil, fmt.Errorf("correlation: index pattern %q is not one this store reads", e.IndexPattern)
		}
		// A pattern over every dataType reads all of them, which is not the same
		// as leaving the triggering rule's dataType in place.
		scope.DataType = dt
	}

	// Pin the end of the range so the count and the sample cannot disagree.
	if scope.To.IsZero() {
		scope.To = time.Now()
	}

	var n int64
	if window > 0 {
		scope.From = scope.To.Add(-window)
		n, err = st.CountInWindow(ctx, scope, filters, window)
	} else {
		n, err = st.Count(ctx, scope, filters)
	}
	if err != nil {
		return false, nil, err
	}

	if uint64(n) >= e.Count {
		docs, err := st.FetchN(ctx, scope, filters, sampleSize)
		if err != nil {
			return false, nil, err
		}
		return true, docs, nil
	}

	var docs []json.RawMessage
	for _, or := range e.Or {
		matched, more, err := or.ExecuteStore(ctx, st, scope, previous)
		if err != nil {
			return false, nil, err
		}
		if matched {
			docs = append(docs, more...)
		}
	}
	return len(docs) > 0, docs, nil
}

func (e *SearchRequest) filters(previous *string) ([]store.Filter, error) {
	out := make([]store.Filter, 0, len(e.With))
	for _, expr := range e.With {
		if expr.Value == nil {
			return nil, fmt.Errorf("correlation: nil value for %q", expr.Field)
		}

		val := expr.Value.AsInterface()
		if s, ok := val.(string); ok && previous != nil &&
			strings.HasPrefix(s, "{{.") && strings.HasSuffix(s, "}}") {
			val = gjson.Get(*previous, strings.TrimSuffix(strings.TrimPrefix(s, "{{."), "}}")).Value()
		}
		if val == nil {
			return nil, fmt.Errorf("correlation: unresolved placeholder for %q", expr.Field)
		}

		op, ok := operators[expr.Operator]
		if !ok {
			return nil, fmt.Errorf("correlation: unknown operator %q", expr.Operator)
		}
		out = append(out, store.Filter{Field: expr.Field, Op: op, Value: val})
	}
	return out, nil
}
