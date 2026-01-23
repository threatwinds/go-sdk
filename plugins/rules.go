package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdkos "github.com/threatwinds/go-sdk/os"
	"github.com/tidwall/gjson"
)

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

// Execute performs the correlation search using the provided context and previous event data
func (e *SearchRequest) Execute(previous *string) (bool, []sdkos.Hit, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	builder := sdkos.NewQueryBuilder(ctx, []string{e.IndexPattern}, "plugin_analysis")

	// Add time range filter
	if e.Within != "" {
		duration, err := time.ParseDuration(e.Within)
		if err == nil {
			builder.Range("@timestamp", "gte", time.Now().Add(-duration).Format(time.RFC3339))
		}
	}

	// Add expressions
	for _, expr := range e.With {
		if expr.Value == nil {
			return false, nil, fmt.Errorf("expression value cannot be nil")
		}
		val := expr.Value.AsInterface()
		// Handle dynamic values
		if previous != nil {
			if s, ok := val.(string); ok && strings.HasPrefix(s, "{{.") && strings.HasSuffix(s, "}}") {
				field := strings.ReplaceAll(s, "{{.", "")
				field = strings.ReplaceAll(field, "}}", "")
				val = gjson.Get(*previous, field).Value()
			}
		}

		if val == nil {
			return false, nil, fmt.Errorf("expression value cannot be nil after placeholder resolution")
		}

		switch expr.Operator {
		case "filter_term":
			builder.Term(expr.Field, val)
		case "must_not_term":
			builder.MustNot(*sdkos.NewQueryBuilder(ctx, []string{e.IndexPattern}, "plugin_analysis").Term(expr.Field, val).Build().Query)
		case "filter_match":
			if s, ok := val.(string); ok {
				builder.Match(expr.Field, s)
			}
		case "must_not_match":
			if s, ok := val.(string); ok {
				builder.MustNot(*sdkos.NewQueryBuilder(ctx, []string{e.IndexPattern}, "plugin_analysis").Match(expr.Field, s).Build().Query)
			}
		}
	}

	sr := builder.Build()

	result, err := sr.WideSearchIn(ctx, []string{e.IndexPattern})
	if err != nil {
		return false, nil, err
	}

	if uint64(result.Hits.Total.Value) >= e.Count {
		return true, result.Hits.Hits, nil
	}

	var hits []sdkos.Hit
	for _, or := range e.Or {
		if alert, newHits, err := or.Execute(previous); alert {
			hits = append(hits, newHits...)
		} else if err != nil {
			return false, nil, err
		}
	}

	return len(e.Or) != 0 && len(hits) > 0, hits, nil
}
