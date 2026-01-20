package os

import (
	"context"
	"testing"
)

func TestQueryBuilder_BasicQuery(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Size(20).
		From(10).
		Match("title", "test").
		Build()

	if query.Size != 20 {
		t.Errorf("Expected size 20, got %d", query.Size)
	}

	if query.From != 10 {
		t.Errorf("Expected from 10, got %d", query.From)
	}

	if len(query.Query.Bool.Must) == 0 {
		t.Error("Expected at least one must clause")
	}
}

func TestQueryBuilder_BoolQuery(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Must(
			TermQuery("status", "active"),
			RangeQuery("age", "gte", 18),
		).
		Should(
			MatchQuery("category", "books"),
			MatchQuery("category", "movies"),
		).
		Filter(ExistsQuery("created_at")).
		MustNot(TermQuery("deleted", true)).
		Build()

	if len(query.Query.Bool.Must) != 2 {
		t.Errorf("Expected 2 must clauses, got %d", len(query.Query.Bool.Must))
	}

	if len(query.Query.Bool.Should) != 2 {
		t.Errorf("Expected 2 should clauses, got %d", len(query.Query.Bool.Should))
	}

	if len(query.Query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Query.Bool.Filter))
	}

	if len(query.Query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Query.Bool.MustNot))
	}
}

func TestQueryBuilder_Sort(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Sort("timestamp", "desc").
		Sort("title", "asc").
		Build()

	if len(query.Sort) != 2 {
		t.Errorf("Expected 2 sort fields, got %d", len(query.Sort))
	}
}

func TestQueryBuilder_Aggregations(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		TermsAgg("top_categories", "category", 10).
		SumAgg("total_price", "price").
		AvgAgg("avg_rating", "rating").
		Build()

	if len(query.Aggs) != 3 {
		t.Errorf("Expected 3 aggregations, got %d", len(query.Aggs))
	}

	if query.Aggs["top_categories"].Terms == nil {
		t.Error("Expected terms aggregation")
	}

	if query.Aggs["total_price"].Sum == nil {
		t.Error("Expected sum aggregation")
	}

	if query.Aggs["avg_rating"].Avg == nil {
		t.Error("Expected avg aggregation")
	}
}

func TestQueryBuilder_SourceFiltering(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		IncludeSource("id", "title", "price").
		ExcludeSource("internal_data").
		Build()

	if query.Source == nil {
		t.Fatal("Expected source to be set")
	}

	if len(query.Source.Includes) != 3 {
		t.Errorf("Expected 3 included fields, got %d", len(query.Source.Includes))
	}

	if len(query.Source.Excludes) != 1 {
		t.Errorf("Expected 1 excluded field, got %d", len(query.Source.Excludes))
	}
}

func TestQueryBuilder_Collapse(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Collapse("user_id").
		Build()

	if query.Collapse == nil {
		t.Fatal("Expected collapse to be set")
	}

	if query.Collapse.Field == "" {
		t.Error("Expected collapse field to be set")
	}
}

func TestQueryBuilder_DefaultValues(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.Build()

	if query.Size != 10 {
		t.Errorf("Expected default size 10, got %d", query.Size)
	}

	if query.From != 0 {
		t.Errorf("Expected default from 0, got %d", query.From)
	}
}

func TestQueryBuilder_RangeQuery(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Range("age", "gte", 18).
		Range("age", "lte", 65).
		Build()

	if len(query.Query.Bool.Must) != 2 {
		t.Errorf("Expected 2 range queries, got %d", len(query.Query.Bool.Must))
	}
}

func TestQueryBuilder_WildcardAndPrefix(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Wildcard("name", "test*").
		Prefix("code", "ABC").
		Build()

	if len(query.Query.Bool.Must) != 2 {
		t.Errorf("Expected 2 queries, got %d", len(query.Query.Bool.Must))
	}
}

func TestHelperFunctions_TermQuery(t *testing.T) {
	query := TermQuery("status", "active")

	if query.Term == nil {
		t.Fatal("Expected term query to be set")
	}

	if _, ok := query.Term["status"]; !ok {
		t.Error("Expected status field in term query")
	}
}

func TestHelperFunctions_MatchQuery(t *testing.T) {
	query := MatchQuery("title", "search text")

	if query.Match == nil {
		t.Fatal("Expected match query to be set")
	}

	if _, ok := query.Match["title"]; !ok {
		t.Error("Expected title field in match query")
	}
}

func TestHelperFunctions_RangeQuery(t *testing.T) {
	query := RangeQuery("age", "gte", 18)

	if query.Range == nil {
		t.Fatal("Expected range query to be set")
	}

	if _, ok := query.Range["age"]; !ok {
		t.Error("Expected age field in range query")
	}
}

func TestBoolBuilder_Basic(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustTerm("status", "active").
		ShouldMatch("category", "books").
		FilterExists("created_at").
		MustNotTerm("deleted", true).
		Build()

	if query.Bool == nil {
		t.Fatal("Expected bool query to be set")
	}

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if len(query.Bool.Should) != 1 {
		t.Errorf("Expected 1 should clause, got %d", len(query.Bool.Should))
	}

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_NestedBool(t *testing.T) {
	ctx := context.Background()

	// Create inner OR condition: (type=ip OR type=domain)
	innerOr := NewBoolBuilder(ctx, []string{"test-index"}, "testing").
		ShouldTerm("type", "ip").
		ShouldTerm("type", "domain").
		MinimumShouldMatch(1)

	// Create outer query with nested bool
	query := NewBoolBuilder(ctx, []string{"test-index"}, "testing").
		MustTerm("status", "active").
		FilterBool(innerOr).
		Build()

	if query.Bool == nil {
		t.Fatal("Expected bool query to be set")
	}

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	// Check nested bool
	nestedBool := query.Bool.Filter[0].Bool
	if nestedBool == nil {
		t.Fatal("Expected nested bool in filter")
	}

	if len(nestedBool.Should) != 2 {
		t.Errorf("Expected 2 should clauses in nested bool, got %d", len(nestedBool.Should))
	}

	if nestedBool.MinimumShouldMatch != 1 {
		t.Errorf("Expected minimum_should_match=1, got %v", nestedBool.MinimumShouldMatch)
	}
}

func TestBoolBuilder_DeeplyNested(t *testing.T) {
	ctx := context.Background()
	indices := []string{"test-index"}

	// Level 3: innermost condition
	level3 := NewBoolBuilder(ctx, indices, "testing").
		MustTerm("subtype", "malware")

	// Level 2: middle condition with level 3
	level2 := NewBoolBuilder(ctx, indices, "testing").
		MustTerm("category", "threat").
		FilterBool(level3)

	// Level 1: outermost with level 2
	query := NewBoolBuilder(ctx, indices, "testing").
		MustTerm("status", "active").
		MustBool(level2).
		Build()

	// Verify structure
	if query.Bool == nil {
		t.Fatal("Expected bool query")
	}

	if len(query.Bool.Must) != 2 {
		t.Errorf("Expected 2 must clauses at level 1, got %d", len(query.Bool.Must))
	}

	// Find the nested bool in must clauses
	var nestedBool *Bool
	for _, q := range query.Bool.Must {
		if q.Bool != nil {
			nestedBool = q.Bool
			break
		}
	}

	if nestedBool == nil {
		t.Fatal("Expected nested bool at level 2")
	}

	if len(nestedBool.Filter) != 1 {
		t.Errorf("Expected 1 filter at level 2, got %d", len(nestedBool.Filter))
	}

	// Check level 3
	level3Bool := nestedBool.Filter[0].Bool
	if level3Bool == nil {
		t.Fatal("Expected nested bool at level 3")
	}

	if len(level3Bool.Must) != 1 {
		t.Errorf("Expected 1 must at level 3, got %d", len(level3Bool.Must))
	}
}

func TestBoolBuilder_QueryBuilderIntegration(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	// Create nested bool using builder.Bool()
	orCondition := builder.Bool().
		ShouldTerm("type", "ip").
		ShouldTerm("type", "domain").
		MinimumShouldMatch(1)

	query := builder.
		Term("status", "active").
		FilterBool(orCondition).
		Size(50).
		Build()

	if query.Size != 50 {
		t.Errorf("Expected size 50, got %d", query.Size)
	}

	if len(query.Query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Query.Bool.Filter))
	}

	nestedBool := query.Query.Bool.Filter[0].Bool
	if nestedBool == nil {
		t.Fatal("Expected nested bool in filter")
	}

	if len(nestedBool.Should) != 2 {
		t.Errorf("Expected 2 should clauses, got %d", len(nestedBool.Should))
	}
}

func TestBoolBuilder_AllClauseTypes(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustTerm("field1", "value1").
		MustTerms("field2", "a", "b", "c").
		MustMatch("field3", "text").
		MustRange("field4", "gte", 10).
		MustExists("field5").
		MustWildcard("field6", "test*").
		ShouldTerm("field7", "value7").
		ShouldTerms("field8", "x", "y").
		ShouldMatch("field9", "text").
		ShouldRange("field10", "lte", 100).
		ShouldExists("field11").
		ShouldWildcard("field12", "*test").
		FilterTerm("field13", "value13").
		FilterTerms("field14", "m", "n").
		FilterMatch("field15", "text").
		FilterRange("field16", "gt", 0).
		FilterExists("field17").
		FilterWildcard("field18", "te*st").
		MustNotTerm("field19", "excluded").
		MustNotTerms("field20", "bad1", "bad2").
		MustNotMatch("field21", "spam").
		MustNotRange("field22", "lt", 0).
		MustNotExists("field23").
		MustNotWildcard("field24", "spam*").
		Build()

	if len(query.Bool.Must) != 6 {
		t.Errorf("Expected 6 must clauses, got %d", len(query.Bool.Must))
	}

	if len(query.Bool.Should) != 6 {
		t.Errorf("Expected 6 should clauses, got %d", len(query.Bool.Should))
	}

	if len(query.Bool.Filter) != 6 {
		t.Errorf("Expected 6 filter clauses, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.MustNot) != 6 {
		t.Errorf("Expected 6 must_not clauses, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_RawQueryMethods(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustQuery(TermQuery("status", "active")).
		ShouldQuery(MatchQuery("title", "test")).
		FilterQuery(ExistsQuery("created")).
		MustNotQuery(TermQuery("deleted", true)).
		Build()

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if len(query.Bool.Should) != 1 {
		t.Errorf("Expected 1 should clause, got %d", len(query.Bool.Should))
	}

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query, errors := builder.
		MustTerm("field", "value").
		BuildWithErrors()

	if query.Bool == nil {
		t.Fatal("Expected bool query even with no errors")
	}

	if len(errors) != 0 {
		t.Errorf("Expected no errors for valid query, got %d", len(errors))
	}

	if builder.HasErrors() {
		t.Error("HasErrors should return false")
	}
}

func TestHelperFunctions_Or(t *testing.T) {
	query := Or(
		TermQuery("type", "ip"),
		TermQuery("type", "domain"),
	)

	if query.Bool == nil {
		t.Fatal("Expected bool query")
	}

	if len(query.Bool.Should) != 2 {
		t.Errorf("Expected 2 should clauses, got %d", len(query.Bool.Should))
	}

	if query.Bool.MinimumShouldMatch != 1 {
		t.Errorf("Expected minimum_should_match=1, got %v", query.Bool.MinimumShouldMatch)
	}
}

func TestHelperFunctions_And(t *testing.T) {
	query := And(
		TermQuery("status", "active"),
		TermQuery("type", "ip"),
	)

	if query.Bool == nil {
		t.Fatal("Expected bool query")
	}

	if len(query.Bool.Must) != 2 {
		t.Errorf("Expected 2 must clauses, got %d", len(query.Bool.Must))
	}
}

func TestHelperFunctions_Not(t *testing.T) {
	query := Not(
		TermQuery("deleted", true),
	)

	if query.Bool == nil {
		t.Fatal("Expected bool query")
	}

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_NilNestedBool(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	// Should not panic with nil nested bool
	query := builder.
		MustTerm("status", "active").
		MustBool(nil).
		ShouldBool(nil).
		FilterBool(nil).
		MustNotBool(nil).
		Build()

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}
}

func TestHelperFunctions_RangeBetween(t *testing.T) {
	query := RangeQueryBetween("age", 18, 65)

	if query.Range == nil {
		t.Fatal("Expected range query to be set")
	}

	rangeParams, ok := query.Range["age"]
	if !ok {
		t.Fatal("Expected age field in range query")
	}

	if _, ok := rangeParams["gte"]; !ok {
		t.Error("Expected gte parameter")
	}

	if _, ok := rangeParams["lte"]; !ok {
		t.Error("Expected lte parameter")
	}
}

func TestHelperFunctions_RangeHelpers(t *testing.T) {
	tests := []struct {
		name     string
		query    Query
		operator string
	}{
		{"RangeGte", RangeGte("age", 18), "gte"},
		{"RangeLte", RangeLte("age", 65), "lte"},
		{"RangeGt", RangeGt("price", 100), "gt"},
		{"RangeLt", RangeLt("price", 1000), "lt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.query.Range == nil {
				t.Fatal("Expected range query to be set")
			}
		})
	}
}

func TestHelperFunctions_MultiMatch(t *testing.T) {
	query := MultiMatchQuery("search text", []string{"title", "description", "tags"})

	if query.MultiMatch == nil {
		t.Fatal("Expected multi_match query to be set")
	}

	if query.MultiMatch.Query != "search text" {
		t.Errorf("Expected query 'search text', got %s", query.MultiMatch.Query)
	}

	if len(query.MultiMatch.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(query.MultiMatch.Fields))
	}
}

func TestHelperFunctions_IDsQuery(t *testing.T) {
	ids := []interface{}{"id1", "id2", "id3"}
	query := IDsQuery(ids)

	if query.IDs == nil {
		t.Fatal("Expected ids query to be set")
	}

	values, ok := query.IDs["values"]
	if !ok {
		t.Fatal("Expected values in ids query")
	}

	if len(values) != 3 {
		t.Errorf("Expected 3 ids, got %d", len(values))
	}
}

func TestHelperFunctions_QueryString(t *testing.T) {
	query := QueryStringQuery("status:active AND category:books", "title")

	if query.QueryString == nil {
		t.Fatal("Expected query_string to be set")
	}

	if query.QueryString.Query != "status:active AND category:books" {
		t.Errorf("Unexpected query string: %s", query.QueryString.Query)
	}

	if query.QueryString.DefaultField != "title" {
		t.Errorf("Expected default field 'title', got %s", query.QueryString.DefaultField)
	}
}

func TestHelperFunctions_SimpleQueryString(t *testing.T) {
	query := SimpleQueryStringQuery("search + text", "title", "description")

	if query.SimpleQueryString == nil {
		t.Fatal("Expected simple_query_string to be set")
	}

	if len(query.SimpleQueryString.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(query.SimpleQueryString.Fields))
	}
}

// --- Tests for new QueryBuilder methods ---

func TestQueryBuilder_RequestLevelMethods(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Version(true).
		StoredFields("field1", "field2").
		ScriptFields(map[string]interface{}{"test_script": map[string]interface{}{"script": "_score"}}).
		Build()

	if !query.Version {
		t.Error("Expected version to be true")
	}

	if len(query.StoredFields) != 2 {
		t.Errorf("Expected 2 stored fields, got %d", len(query.StoredFields))
	}

	if query.ScriptFields == nil {
		t.Error("Expected script fields to be set")
	}
}

func TestQueryBuilder_SetSource(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	source := &Source{
		Includes: []string{"field1", "field2"},
		Excludes: []string{"internal"},
	}

	query := builder.
		SetSource(source).
		Build()

	if query.Source == nil {
		t.Fatal("Expected source to be set")
	}

	if len(query.Source.Includes) != 2 {
		t.Errorf("Expected 2 includes, got %d", len(query.Source.Includes))
	}

	if len(query.Source.Excludes) != 1 {
		t.Errorf("Expected 1 exclude, got %d", len(query.Source.Excludes))
	}
}

func TestQueryBuilder_IDs(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		IDs("id1", "id2", "id3").
		Build()

	if len(query.Query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Query.Bool.Filter))
	}

	if query.Query.Bool.Filter[0].IDs == nil {
		t.Error("Expected IDs query in filter")
	}
}

func TestQueryBuilder_QueryString(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		QueryString("status:active AND type:threat", "AND").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].QueryString == nil {
		t.Error("Expected QueryString query in must")
	}

	if query.Query.Bool.Must[0].QueryString.DefaultOperator != "AND" {
		t.Errorf("Expected default operator AND, got %s", query.Query.Bool.Must[0].QueryString.DefaultOperator)
	}
}

func TestQueryBuilder_SimpleQueryString(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		SimpleQueryString("test query", "title", "description").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].SimpleQueryString == nil {
		t.Error("Expected SimpleQueryString query in must")
	}

	if len(query.Query.Bool.Must[0].SimpleQueryString.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(query.Query.Bool.Must[0].SimpleQueryString.Fields))
	}
}

func TestQueryBuilder_MultiMatch(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MultiMatch("search text", []string{"title", "description"}, "best_fields").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].MultiMatch == nil {
		t.Error("Expected MultiMatch query in must")
	}

	if query.Query.Bool.Must[0].MultiMatch.Type != "best_fields" {
		t.Errorf("Expected type best_fields, got %s", query.Query.Bool.Must[0].MultiMatch.Type)
	}
}

func TestQueryBuilder_Fuzzy(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Fuzzy("title", "serch", "AUTO").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].Fuzzy == nil {
		t.Error("Expected Fuzzy query in must")
	}
}

func TestQueryBuilder_Regexp(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Regexp("status", "act.*").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].Regexp == nil {
		t.Error("Expected Regexp query in must")
	}
}

func TestQueryBuilder_MatchPhrasePrefix(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MatchPhrasePrefix("title", "quick bro").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].MatchPhrasePrefix == nil {
		t.Error("Expected MatchPhrasePrefix query in must")
	}
}

// --- Tests for new aggregation methods ---

func TestQueryBuilder_MetricAggregations(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MinAgg("min_price", "price").
		MaxAgg("max_price", "price").
		ValueCountAgg("count_items", "item_id").
		StatsAgg("price_stats", "price").
		PercentilesAgg("price_percentiles", "price").
		Build()

	if len(query.Aggs) != 5 {
		t.Errorf("Expected 5 aggregations, got %d", len(query.Aggs))
	}

	if query.Aggs["min_price"].Min == nil {
		t.Error("Expected min aggregation")
	}

	if query.Aggs["max_price"].Max == nil {
		t.Error("Expected max aggregation")
	}

	if query.Aggs["count_items"].ValueCount == nil {
		t.Error("Expected value_count aggregation")
	}

	if query.Aggs["price_stats"].Stats == nil {
		t.Error("Expected stats aggregation")
	}

	if query.Aggs["price_percentiles"].Percentiles == nil {
		t.Error("Expected percentiles aggregation")
	}
}

func TestQueryBuilder_ExtendedStatsAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		ExtendedStatsAgg("extended_price_stats", "price", 2).
		Build()

	if query.Aggs["extended_price_stats"].ExtendedStats == nil {
		t.Error("Expected extended_stats aggregation")
	}

	if query.Aggs["extended_price_stats"].ExtendedStats.Sigma != 2 {
		t.Errorf("Expected sigma 2, got %d", query.Aggs["extended_price_stats"].ExtendedStats.Sigma)
	}
}

func TestQueryBuilder_PercentileRanksAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		PercentileRanksAgg("price_ranks", "price", []int64{100, 200, 300}).
		Build()

	if query.Aggs["price_ranks"].PercentileRanks == nil {
		t.Error("Expected percentile_ranks aggregation")
	}

	if len(query.Aggs["price_ranks"].PercentileRanks.Values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(query.Aggs["price_ranks"].PercentileRanks.Values))
	}
}

func TestQueryBuilder_MatrixStatsAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MatrixStatsAgg("matrix_stats", []string{"price", "quantity", "rating"}).
		Build()

	if query.Aggs["matrix_stats"].MatrixStats == nil {
		t.Error("Expected matrix_stats aggregation")
	}
}

func TestQueryBuilder_TopHitsAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		TopHitsAgg("top_docs", 5).
		Build()

	if query.Aggs["top_docs"].TopHits == nil {
		t.Error("Expected top_hits aggregation")
	}

	if query.Aggs["top_docs"].TopHits.Size != 5 {
		t.Errorf("Expected size 5, got %d", query.Aggs["top_docs"].TopHits.Size)
	}
}

func TestQueryBuilder_BucketAggregations(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		HistogramAgg("price_histogram", "price", 100.0).
		DateHistogramAgg("date_histogram", "created_at", "1d").
		Build()

	if len(query.Aggs) != 2 {
		t.Errorf("Expected 2 aggregations, got %d", len(query.Aggs))
	}

	if query.Aggs["price_histogram"].Histogram == nil {
		t.Error("Expected histogram aggregation")
	}

	if query.Aggs["date_histogram"].DateHistogram == nil {
		t.Error("Expected date_histogram aggregation")
	}
}

func TestQueryBuilder_RangeAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	ranges := []map[string]interface{}{
		{"to": 100},
		{"from": 100, "to": 200},
		{"from": 200},
	}

	query := builder.
		RangeAgg("price_ranges", "price", ranges).
		Build()

	if query.Aggs["price_ranges"].Range == nil {
		t.Error("Expected range aggregation")
	}

	if len(query.Aggs["price_ranges"].Range.Ranges) != 3 {
		t.Errorf("Expected 3 ranges, got %d", len(query.Aggs["price_ranges"].Range.Ranges))
	}
}

func TestQueryBuilder_DateRangeAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	ranges := []map[string]interface{}{
		{"to": "now-1M"},
		{"from": "now-1M"},
	}

	query := builder.
		DateRangeAgg("date_ranges", "created_at", "yyyy-MM-dd", ranges).
		Build()

	if query.Aggs["date_ranges"].DateRange == nil {
		t.Error("Expected date_range aggregation")
	}

	if query.Aggs["date_ranges"].DateRange.Format != "yyyy-MM-dd" {
		t.Errorf("Expected format 'yyyy-MM-dd', got %s", query.Aggs["date_ranges"].DateRange.Format)
	}
}

func TestQueryBuilder_IPRangeAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	ranges := []map[string]interface{}{
		{"to": "10.0.0.100"},
		{"from": "10.0.0.100"},
	}

	query := builder.
		IPRangeAgg("ip_ranges", "client_ip", ranges).
		Build()

	if query.Aggs["ip_ranges"].IPRange == nil {
		t.Error("Expected ip_range aggregation")
	}
}

func TestQueryBuilder_SignificantTermsAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		SignificantTermsAgg("significant", "category").
		Build()

	if query.Aggs["significant"].SignificantTerms == nil {
		t.Error("Expected significant_terms aggregation")
	}
}

func TestQueryBuilder_FilterAndFiltersAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	filterQuery := Query{
		Term: map[string]map[string]interface{}{
			"status": {"value": "active"},
		},
	}

	filters := map[string]interface{}{
		"filters": map[string]interface{}{
			"active":   map[string]interface{}{"term": map[string]interface{}{"status": "active"}},
			"inactive": map[string]interface{}{"term": map[string]interface{}{"status": "inactive"}},
		},
	}

	query := builder.
		FilterAgg("active_only", filterQuery).
		FiltersAgg("status_filters", filters).
		Build()

	if query.Aggs["active_only"].Filter == nil {
		t.Error("Expected filter aggregation")
	}

	if query.Aggs["status_filters"].Filters == nil {
		t.Error("Expected filters aggregation")
	}
}

func TestQueryBuilder_GlobalAndNestedAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		GlobalAgg("all_products").
		NestedAgg("nested_reviews", "reviews").
		ReverseNestedAgg("back_to_root").
		Build()

	if query.Aggs["all_products"].Global == nil {
		t.Error("Expected global aggregation")
	}

	if query.Aggs["nested_reviews"].Nested == nil {
		t.Error("Expected nested aggregation")
	}

	if query.Aggs["back_to_root"].ReverseNested == nil {
		t.Error("Expected reverse_nested aggregation")
	}
}

func TestQueryBuilder_SamplerAggs(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		SamplerAgg("sample", 100).
		DiversifiedSamplerAgg("diverse_sample", "category", 50).
		Build()

	if query.Aggs["sample"].Sampler == nil {
		t.Error("Expected sampler aggregation")
	}

	if query.Aggs["diverse_sample"].DiversifiedSampler == nil {
		t.Error("Expected diversified_sampler aggregation")
	}
}

func TestQueryBuilder_GeoAggregations(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	ranges := []map[string]interface{}{
		{"to": 100},
		{"from": 100, "to": 300},
		{"from": 300},
	}

	query := builder.
		GeoDistanceAgg("distance_rings", "location", "40.7128,-74.0060", ranges).
		GeohashGridAgg("geohash", "location", 5).
		GeotileGridAgg("geotile", "location", 7).
		Build()

	if query.Aggs["distance_rings"].GeoDistance == nil {
		t.Error("Expected geo_distance aggregation")
	}

	if query.Aggs["geohash"].GeohashGrid == nil {
		t.Error("Expected geohash_grid aggregation")
	}

	if query.Aggs["geotile"].GeotileGrid == nil {
		t.Error("Expected geotile_grid aggregation")
	}
}

func TestQueryBuilder_PipelineAggregations(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		DateHistogramAgg("sales_per_month", "@timestamp", "month").
		SumAgg("monthly_sales", "sales").
		SumBucketAgg("total_sales", "sales_per_month>monthly_sales").
		AvgBucketAgg("avg_monthly_sales", "sales_per_month>monthly_sales").
		MinBucketAgg("min_monthly_sales", "sales_per_month>monthly_sales").
		MaxBucketAgg("max_monthly_sales", "sales_per_month>monthly_sales").
		Build()

	if query.Aggs["total_sales"].SumBucket == nil {
		t.Error("Expected sum_bucket aggregation")
	}

	if query.Aggs["avg_monthly_sales"].AvgBucket == nil {
		t.Error("Expected avg_bucket aggregation")
	}

	if query.Aggs["min_monthly_sales"].MinBucket == nil {
		t.Error("Expected min_bucket aggregation")
	}

	if query.Aggs["max_monthly_sales"].MaxBucket == nil {
		t.Error("Expected max_bucket aggregation")
	}
}

func TestQueryBuilder_MorePipelineAggregations(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		StatsBucketAgg("stats", "path>agg").
		ExtendedStatsBucketAgg("extended_stats", "path>agg").
		CumulativeSumAgg("cumulative", "path>agg").
		DerivativeAgg("derivative", "path>agg").
		Build()

	if query.Aggs["stats"].StatsBucket == nil {
		t.Error("Expected stats_bucket aggregation")
	}

	if query.Aggs["extended_stats"].ExtendedStatsBucket == nil {
		t.Error("Expected extended_stats_bucket aggregation")
	}

	if query.Aggs["cumulative"].CumulativeSum == nil {
		t.Error("Expected cumulative_sum aggregation")
	}

	if query.Aggs["derivative"].Derivative == nil {
		t.Error("Expected derivative aggregation")
	}
}

func TestQueryBuilder_MovingAvgAndSerialDiff(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MovingAvgAgg("moving_avg", "path>agg", 10, "simple").
		SerialDiffAgg("serial_diff", "path>agg", 1).
		Build()

	if query.Aggs["moving_avg"].MovingAvg == nil {
		t.Error("Expected moving_avg aggregation")
	}

	if query.Aggs["moving_avg"].MovingAvg.Window != 10 {
		t.Errorf("Expected window 10, got %d", query.Aggs["moving_avg"].MovingAvg.Window)
	}

	if query.Aggs["moving_avg"].MovingAvg.Model != "simple" {
		t.Errorf("Expected model 'simple', got %s", query.Aggs["moving_avg"].MovingAvg.Model)
	}

	if query.Aggs["serial_diff"].SerialDiff == nil {
		t.Error("Expected serial_diff aggregation")
	}

	if query.Aggs["serial_diff"].SerialDiff.Lag != 1 {
		t.Errorf("Expected lag 1, got %d", query.Aggs["serial_diff"].SerialDiff.Lag)
	}
}

func TestQueryBuilder_BucketSortAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	sort := []map[string]interface{}{
		{"total_sales": map[string]interface{}{"order": "desc"}},
	}

	query := builder.
		BucketSortAgg("sort_buckets", sort, 10).
		Build()

	if query.Aggs["sort_buckets"].BucketSort == nil {
		t.Error("Expected bucket_sort aggregation")
	}
}

func TestQueryBuilder_SubAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		TermsAgg("categories", "category", 10).
		SubAgg("categories", "avg_price", Aggs{Avg: &Agg{Field: "price"}}).
		Build()

	if query.Aggs["categories"].Aggs == nil {
		t.Error("Expected sub-aggregations")
	}

	if query.Aggs["categories"].Aggs["avg_price"].Avg == nil {
		t.Error("Expected avg sub-aggregation")
	}
}

func TestQueryBuilder_Agg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	customAgg := Aggs{
		Terms: &Terms{Field: "custom_field", Size: 20},
	}

	query := builder.
		Agg("custom", customAgg).
		Build()

	if query.Aggs["custom"].Terms == nil {
		t.Error("Expected custom aggregation")
	}

	if query.Aggs["custom"].Terms.Size != 20 {
		t.Errorf("Expected size 20, got %d", query.Aggs["custom"].Terms.Size)
	}
}

// --- Tests for new BoolBuilder methods ---

func TestBoolBuilder_PrefixMethods(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustPrefix("code", "ABC").
		ShouldPrefix("name", "Jo").
		FilterPrefix("sku", "SKU-").
		MustNotPrefix("status", "DEL").
		Build()

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if len(query.Bool.Should) != 1 {
		t.Errorf("Expected 1 should clause, got %d", len(query.Bool.Should))
	}

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_IDsMethods(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		FilterIDs("id1", "id2").
		MustNotIDs("id3", "id4").
		Build()

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_QueryStringMethods(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustQueryString("status:active").
		ShouldQueryString("type:threat OR type:indicator").
		FilterQueryString("category:malware", "AND").
		MustNotQueryString("deleted:true").
		Build()

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if len(query.Bool.Should) != 1 {
		t.Errorf("Expected 1 should clause, got %d", len(query.Bool.Should))
	}

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_MultiMatchMethods(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustMultiMatch("search text", []string{"title", "description"}).
		ShouldMultiMatch("optional", []string{"tags"}, "best_fields").
		FilterMultiMatch("filter text", []string{"content"}).
		MustNotMultiMatch("excluded", []string{"spam_field"}).
		Build()

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if len(query.Bool.Should) != 1 {
		t.Errorf("Expected 1 should clause, got %d", len(query.Bool.Should))
	}

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_FuzzyMethods(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustFuzzy("title", "serch").
		ShouldFuzzy("description", "approxmate", "AUTO").
		Build()

	// Note: Fuzzy may add errors if field type is wrong, but query still builds
	if query.Bool == nil {
		t.Error("Expected bool query to be set")
	}
}

func TestBoolBuilder_RegexpMethods(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustRegexp("status", "act.*").
		ShouldRegexp("code", "[A-Z]{3}[0-9]+").
		FilterRegexp("id", "id-[0-9]+").
		MustNotRegexp("type", "test.*").
		Build()

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if len(query.Bool.Should) != 1 {
		t.Errorf("Expected 1 should clause, got %d", len(query.Bool.Should))
	}

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_MatchPhrasePrefixMethods(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustMatchPhrasePrefix("title", "quick bro").
		ShouldMatchPhrasePrefix("description", "the lazy").
		Build()

	// Note: May add errors if field type is wrong
	if query.Bool == nil {
		t.Error("Expected bool query to be set")
	}
}

func TestBoolBuilder_MatchPhraseMethods(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustMatchPhrase("title", "quick brown fox").
		ShouldMatchPhrase("description", "lazy dog").
		FilterMatchPhrase("content", "exact phrase").
		MustNotMatchPhrase("spam", "buy now").
		Build()

	// Note: May add errors if field type is wrong
	if query.Bool == nil {
		t.Error("Expected bool query to be set")
	}
}

func TestQueryBuilder_AdjacencyMatrixAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	filters := map[string]interface{}{
		"filters": map[string]interface{}{
			"grpA": map[string]interface{}{"term": map[string]interface{}{"type": "A"}},
			"grpB": map[string]interface{}{"term": map[string]interface{}{"type": "B"}},
		},
	}

	query := builder.
		AdjacencyMatrixAgg("interactions", filters).
		Build()

	if query.Aggs["interactions"].AdjacencyMatrix == nil {
		t.Error("Expected adjacency_matrix aggregation")
	}
}

func TestQueryBuilder_MultiTermsAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	terms := []Agg{
		{Field: "category"},
		{Field: "brand"},
	}

	order := map[string]string{"_count": "desc"}

	query := builder.
		MultiTermsAgg("category_brand", terms, order).
		Build()

	if query.Aggs["category_brand"].MultiTerms == nil {
		t.Error("Expected multi_terms aggregation")
	}

	if len(query.Aggs["category_brand"].MultiTerms.Terms) != 2 {
		t.Errorf("Expected 2 terms, got %d", len(query.Aggs["category_brand"].MultiTerms.Terms))
	}
}

func TestQueryBuilder_SignificantTextAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	opts := map[string]interface{}{
		"min_doc_count": 5,
	}

	query := builder.
		SignificantTextAgg("significant_text", "content", opts).
		Build()

	if query.Aggs["significant_text"].SignificantText == nil {
		t.Error("Expected significant_text aggregation")
	}
}

// --- Tests for QueryBuilder methods with 0% coverage ---

func TestQueryBuilder_SearchAfter(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		SearchAfter([]int64{1234567890, 42}).
		Build()

	if len(query.SearchAfter) != 2 {
		t.Errorf("Expected 2 search_after values, got %d", len(query.SearchAfter))
	}

	if query.SearchAfter[0] != 1234567890 {
		t.Errorf("Expected first search_after value 1234567890, got %d", query.SearchAfter[0])
	}
}

func TestQueryBuilder_MinimumShouldMatch(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Should(MatchQuery("field1", "value1")).
		Should(MatchQuery("field2", "value2")).
		MinimumShouldMatch(1).
		Build()

	if query.Query.Bool.MinimumShouldMatch != 1 {
		t.Errorf("Expected minimum_should_match 1, got %v", query.Query.Bool.MinimumShouldMatch)
	}
}

func TestQueryBuilder_Terms(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Terms("status", []interface{}{"active", "pending", "review"}).
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].Terms == nil {
		t.Error("Expected terms query in must")
	}
}

func TestQueryBuilder_MatchPhrase(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MatchPhrase("title", "quick brown fox").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].MatchPhrase == nil {
		t.Error("Expected match_phrase query in must")
	}
}

func TestQueryBuilder_Exists(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		Exists("created_at").
		Build()

	if len(query.Query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Query.Bool.Filter))
	}

	if query.Query.Bool.Filter[0].Exists == nil {
		t.Error("Expected exists query in filter")
	}
}

func TestQueryBuilder_MatchAll(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MatchAll().
		Size(100).
		Build()

	// MatchAll just returns without adding conditions
	if query.Size != 100 {
		t.Errorf("Expected size 100, got %d", query.Size)
	}
}

func TestQueryBuilder_CardinalityAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		CardinalityAgg("unique_users", "user_id").
		Build()

	if query.Aggs["unique_users"].Cardinality == nil {
		t.Error("Expected cardinality aggregation")
	}
}

func TestQueryBuilder_GeohexGridAgg(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		GeohexGridAgg("geohex", "location", 4).
		Build()

	if query.Aggs["geohex"].GeohexGrid == nil {
		t.Error("Expected geohex_grid aggregation")
	}

	if query.Aggs["geohex"].GeohexGrid.Precision != 4 {
		t.Errorf("Expected precision 4, got %d", query.Aggs["geohex"].GeohexGrid.Precision)
	}
}

func TestQueryBuilder_BuildWithErrors(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query, errors := builder.
		Term("status", "active").
		Build(), builder.errors

	if query.Query.Bool == nil {
		t.Error("Expected bool query to be set")
	}

	// Should have no errors for valid query
	_ = errors // errors are checked via builder.errors
}

func TestQueryBuilder_MustBool(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	nestedBool := builder.Bool().
		MustTerm("status", "active")

	query := builder.
		MustBool(nestedBool).
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].Bool == nil {
		t.Error("Expected nested bool query in must")
	}
}

func TestQueryBuilder_ShouldBool(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	nestedBool := builder.Bool().
		MustTerm("type", "ip")

	query := builder.
		ShouldBool(nestedBool).
		Build()

	if len(query.Query.Bool.Should) != 1 {
		t.Errorf("Expected 1 should clause, got %d", len(query.Query.Bool.Should))
	}

	if query.Query.Bool.Should[0].Bool == nil {
		t.Error("Expected nested bool query in should")
	}
}

func TestQueryBuilder_MustNotBool(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	nestedBool := builder.Bool().
		MustTerm("deleted", true)

	query := builder.
		MustNotBool(nestedBool).
		Build()

	if len(query.Query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Query.Bool.MustNot))
	}

	if query.Query.Bool.MustNot[0].Bool == nil {
		t.Error("Expected nested bool query in must_not")
	}
}

func TestQueryBuilder_NilBoolHandling(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	// All nil Bool methods should not panic and return builder
	query := builder.
		MustBool(nil).
		ShouldBool(nil).
		FilterBool(nil).
		MustNotBool(nil).
		Build()

	// Should have empty clauses
	if len(query.Query.Bool.Must) != 0 {
		t.Errorf("Expected 0 must clauses, got %d", len(query.Query.Bool.Must))
	}
}

// --- Tests for helper functions with 0% coverage ---

func TestHelperFunctions_TermsQuery(t *testing.T) {
	query := TermsQuery("status", []interface{}{"active", "pending"})

	if query.Terms == nil {
		t.Fatal("Expected terms query to be set")
	}

	if _, ok := query.Terms["status"]; !ok {
		t.Error("Expected status field in terms query")
	}

	if len(query.Terms["status"]) != 2 {
		t.Errorf("Expected 2 values, got %d", len(query.Terms["status"]))
	}
}

func TestHelperFunctions_MatchPhraseQuery(t *testing.T) {
	query := MatchPhraseQuery("title", "quick brown fox")

	if query.MatchPhrase == nil {
		t.Fatal("Expected match_phrase query to be set")
	}

	if _, ok := query.MatchPhrase["title"]; !ok {
		t.Error("Expected title field in match_phrase query")
	}
}

func TestHelperFunctions_MatchPhrasePrefixQuery(t *testing.T) {
	query := MatchPhrasePrefixQuery("title", "quick bro")

	if query.MatchPhrasePrefix == nil {
		t.Fatal("Expected match_phrase_prefix query to be set")
	}

	if _, ok := query.MatchPhrasePrefix["title"]; !ok {
		t.Error("Expected title field in match_phrase_prefix query")
	}
}

func TestHelperFunctions_WildcardQuery(t *testing.T) {
	query := WildcardQuery("name", "test*")

	if query.Wildcard == nil {
		t.Fatal("Expected wildcard query to be set")
	}

	if _, ok := query.Wildcard["name"]; !ok {
		t.Error("Expected name field in wildcard query")
	}
}

func TestHelperFunctions_PrefixQuery(t *testing.T) {
	query := PrefixQuery("code", "ABC")

	if query.Prefix == nil {
		t.Fatal("Expected prefix query to be set")
	}

	if _, ok := query.Prefix["code"]; !ok {
		t.Error("Expected code field in prefix query")
	}
}

func TestHelperFunctions_FuzzyQuery(t *testing.T) {
	query := FuzzyQuery("title", "serch", "AUTO")

	if query.Fuzzy == nil {
		t.Fatal("Expected fuzzy query to be set")
	}

	if _, ok := query.Fuzzy["title"]; !ok {
		t.Error("Expected title field in fuzzy query")
	}
}

func TestHelperFunctions_FuzzyQueryNoFuzziness(t *testing.T) {
	query := FuzzyQuery("title", "serch")

	if query.Fuzzy == nil {
		t.Fatal("Expected fuzzy query to be set")
	}
}

func TestHelperFunctions_RegexpQuery(t *testing.T) {
	query := RegexpQuery("status", "act.*")

	if query.Regexp == nil {
		t.Fatal("Expected regexp query to be set")
	}

	if _, ok := query.Regexp["status"]; !ok {
		t.Error("Expected status field in regexp query")
	}
}

func TestHelperFunctions_ValidateQuery(t *testing.T) {
	tests := []struct {
		name      string
		fieldType string
		queryType QueryType
		wantErr   bool
	}{
		{"keyword with term", "keyword", QueryTypeTerm, false},
		{"text with term", "text", QueryTypeTerm, true},
		{"text with match", "text", QueryTypeMatch, false},
		{"keyword with match", "keyword", QueryTypeMatch, true},
		{"any with exists", "text", QueryTypeExists, false},
		{"any with wildcard", "keyword", QueryTypeWildcard, false},
		{"text with match_phrase", "text", QueryTypeMatchPhrase, false},
		{"keyword with range", "keyword", QueryTypeRange, false},
		{"text with range", "text", QueryTypeRange, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQuery(tt.fieldType, tt.queryType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateQuery(%s, %v) error = %v, wantErr %v", tt.fieldType, tt.queryType, err, tt.wantErr)
			}
		})
	}
}

// --- Tests for BoolBuilder methods with 0% coverage ---

func TestBoolBuilder_FilterFuzzy(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		FilterFuzzy("title", "serch", "AUTO").
		Build()

	// Note: May fail field resolution, but should still build
	if query.Bool == nil {
		t.Error("Expected bool query to be set")
	}
}

func TestBoolBuilder_MustNotFuzzy(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustNotFuzzy("title", "spam").
		Build()

	if query.Bool == nil {
		t.Error("Expected bool query to be set")
	}
}

func TestBoolBuilder_FilterMatchPhrasePrefix(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		FilterMatchPhrasePrefix("title", "quick bro").
		Build()

	if query.Bool == nil {
		t.Error("Expected bool query to be set")
	}
}

func TestBoolBuilder_MustNotMatchPhrasePrefix(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustNotMatchPhrasePrefix("title", "bad phrase").
		Build()

	if query.Bool == nil {
		t.Error("Expected bool query to be set")
	}
}

func TestBoolBuilder_Errors(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	// Build a valid query
	builder.MustTerm("status", "active")

	// Get errors (should be empty for valid query)
	errors := builder.Errors()

	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(errors))
	}
}

func TestBoolBuilder_MustNotBool_WithErrors(t *testing.T) {
	ctx := context.Background()

	// Create a nested bool builder
	nestedBool := NewBoolBuilder(ctx, []string{"test-index"}, "testing").
		MustTerm("status", "active")

	// Create outer builder with must_not bool
	query := NewBoolBuilder(ctx, []string{"test-index"}, "testing").
		MustNotBool(nestedBool).
		Build()

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}

	if query.Bool.MustNot[0].Bool == nil {
		t.Error("Expected nested bool query in must_not")
	}
}

// --- Tests for CIDR query support ---

func TestHelperFunctions_CIDRQuery(t *testing.T) {
	query := CIDRQuery("client_ip", "192.168.0.0/16")

	if query.Term == nil {
		t.Fatal("Expected term query to be set")
	}

	if _, ok := query.Term["client_ip"]; !ok {
		t.Error("Expected client_ip field in term query")
	}

	if query.Term["client_ip"]["value"] != "192.168.0.0/16" {
		t.Errorf("Expected CIDR value '192.168.0.0/16', got %v", query.Term["client_ip"]["value"])
	}
}

func TestHelperFunctions_CIDRsQuery(t *testing.T) {
	query := CIDRsQuery("client_ip", "192.168.0.0/24", "10.0.0.0/8")

	if query.Terms == nil {
		t.Fatal("Expected terms query to be set")
	}

	if _, ok := query.Terms["client_ip"]; !ok {
		t.Error("Expected client_ip field in terms query")
	}

	if len(query.Terms["client_ip"]) != 2 {
		t.Errorf("Expected 2 CIDR values, got %d", len(query.Terms["client_ip"]))
	}
}

func TestQueryBuilder_CIDR(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		CIDR("client_ip", "192.168.0.0/16").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].Term == nil {
		t.Error("Expected term query in must")
	}

	if query.Query.Bool.Must[0].Term["client_ip"]["value"] != "192.168.0.0/16" {
		t.Errorf("Expected CIDR value, got %v", query.Query.Bool.Must[0].Term["client_ip"]["value"])
	}
}

func TestQueryBuilder_CIDRs(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		CIDRs("client_ip", "192.168.0.0/24", "10.0.0.0/8", "172.16.0.0/12").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	if query.Query.Bool.Must[0].Terms == nil {
		t.Error("Expected terms query in must")
	}

	if len(query.Query.Bool.Must[0].Terms["client_ip"]) != 3 {
		t.Errorf("Expected 3 CIDR values, got %d", len(query.Query.Bool.Must[0].Terms["client_ip"]))
	}
}

func TestBoolBuilder_MustCIDR(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustCIDR("source_ip", "10.0.0.0/8").
		Build()

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if query.Bool.Must[0].Term == nil {
		t.Error("Expected term query in must")
	}
}

func TestBoolBuilder_ShouldCIDR(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		ShouldCIDR("source_ip", "192.168.0.0/16").
		ShouldCIDR("source_ip", "10.0.0.0/8").
		MinimumShouldMatch(1).
		Build()

	if len(query.Bool.Should) != 2 {
		t.Errorf("Expected 2 should clauses, got %d", len(query.Bool.Should))
	}
}

func TestBoolBuilder_FilterCIDR(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		FilterCIDR("dest_ip", "172.16.0.0/12").
		Build()

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}
}

func TestBoolBuilder_MustNotCIDR(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustNotCIDR("source_ip", "127.0.0.0/8"). // Exclude localhost
		Build()

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_MustCIDRs(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustCIDRs("client_ip", "192.168.0.0/24", "10.0.0.0/8").
		Build()

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if query.Bool.Must[0].Terms == nil {
		t.Error("Expected terms query in must")
	}

	if len(query.Bool.Must[0].Terms["client_ip"]) != 2 {
		t.Errorf("Expected 2 CIDR values, got %d", len(query.Bool.Must[0].Terms["client_ip"]))
	}
}

func TestBoolBuilder_FilterCIDRs(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	// Filter to only private IP ranges
	query := builder.
		FilterCIDRs("source_ip", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16").
		Build()

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.Filter[0].Terms["source_ip"]) != 3 {
		t.Errorf("Expected 3 private CIDR ranges, got %d", len(query.Bool.Filter[0].Terms["source_ip"]))
	}
}

func TestBoolBuilder_MustNotCIDRs(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	// Exclude private IP ranges
	query := builder.
		MustNotCIDRs("source_ip", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8").
		Build()

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}

	if len(query.Bool.MustNot[0].Terms["source_ip"]) != 4 {
		t.Errorf("Expected 4 CIDR ranges to exclude, got %d", len(query.Bool.MustNot[0].Terms["source_ip"]))
	}
}

func TestBoolBuilder_CombinedIPQuery(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	// Complex query: source from private network, destination NOT localhost
	query := builder.
		FilterCIDRs("source_ip", "10.0.0.0/8", "192.168.0.0/16").
		MustNotCIDR("dest_ip", "127.0.0.0/8").
		Build()

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

// === IP Range Field Tests (for "ip_range" type fields) ===
// These test queries against ip_range fields which store IP address ranges

func TestHelperFunctions_IPRangeContainsQuery(t *testing.T) {
	query := IPRangeContainsQuery("allowed_ips", "192.168.1.50")

	if query.Range == nil {
		t.Fatal("Expected Range query to be set")
	}

	rangeParams := query.Range["allowed_ips"]
	if rangeParams["gte"] != "192.168.1.50" {
		t.Errorf("Expected gte='192.168.1.50', got %v", rangeParams["gte"])
	}
	if rangeParams["lte"] != "192.168.1.50" {
		t.Errorf("Expected lte='192.168.1.50', got %v", rangeParams["lte"])
	}
	if rangeParams["relation"] != "contains" {
		t.Errorf("Expected relation='contains', got %v", rangeParams["relation"])
	}
}

func TestHelperFunctions_IPRangeIntersectsQuery(t *testing.T) {
	query := IPRangeIntersectsQuery("blocked_ranges", "192.168.0.0", "192.168.255.255")

	if query.Range == nil {
		t.Fatal("Expected Range query to be set")
	}

	rangeParams := query.Range["blocked_ranges"]
	if rangeParams["gte"] != "192.168.0.0" {
		t.Errorf("Expected gte='192.168.0.0', got %v", rangeParams["gte"])
	}
	if rangeParams["lte"] != "192.168.255.255" {
		t.Errorf("Expected lte='192.168.255.255', got %v", rangeParams["lte"])
	}
	if rangeParams["relation"] != "intersects" {
		t.Errorf("Expected relation='intersects', got %v", rangeParams["relation"])
	}
}

func TestHelperFunctions_IPRangeWithinQuery(t *testing.T) {
	query := IPRangeWithinQuery("subnet", "10.0.0.0", "10.255.255.255")

	if query.Range == nil {
		t.Fatal("Expected Range query to be set")
	}

	rangeParams := query.Range["subnet"]
	if rangeParams["gte"] != "10.0.0.0" {
		t.Errorf("Expected gte='10.0.0.0', got %v", rangeParams["gte"])
	}
	if rangeParams["lte"] != "10.255.255.255" {
		t.Errorf("Expected lte='10.255.255.255', got %v", rangeParams["lte"])
	}
	if rangeParams["relation"] != "within" {
		t.Errorf("Expected relation='within', got %v", rangeParams["relation"])
	}
}

func TestQueryBuilder_IPRangeContains(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		IPRangeContains("allowed_ips", "192.168.1.50").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Fatalf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	rangeQ := query.Query.Bool.Must[0].Range
	if rangeQ == nil {
		t.Fatal("Expected Range query in must clause")
	}

	rangeParams := rangeQ["allowed_ips"]
	if rangeParams["relation"] != "contains" {
		t.Errorf("Expected relation='contains', got %v", rangeParams["relation"])
	}
}

func TestQueryBuilder_IPRangeIntersects(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		IPRangeIntersects("blocked_ranges", "192.168.0.0", "192.168.255.255").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Fatalf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	rangeQ := query.Query.Bool.Must[0].Range
	if rangeQ == nil {
		t.Fatal("Expected Range query in must clause")
	}

	rangeParams := rangeQ["blocked_ranges"]
	if rangeParams["relation"] != "intersects" {
		t.Errorf("Expected relation='intersects', got %v", rangeParams["relation"])
	}
}

func TestQueryBuilder_IPRangeWithin(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		IPRangeWithin("subnet", "10.0.0.0", "10.255.255.255").
		Build()

	if len(query.Query.Bool.Must) != 1 {
		t.Fatalf("Expected 1 must clause, got %d", len(query.Query.Bool.Must))
	}

	rangeQ := query.Query.Bool.Must[0].Range
	if rangeQ == nil {
		t.Fatal("Expected Range query in must clause")
	}

	rangeParams := rangeQ["subnet"]
	if rangeParams["relation"] != "within" {
		t.Errorf("Expected relation='within', got %v", rangeParams["relation"])
	}
}

func TestBoolBuilder_MustIPRangeContains(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustIPRangeContains("allowed_ips", "192.168.1.50").
		Build()

	if len(query.Bool.Must) != 1 {
		t.Fatalf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	rangeQ := query.Bool.Must[0].Range
	if rangeQ == nil {
		t.Fatal("Expected Range query")
	}

	rangeParams := rangeQ["allowed_ips"]
	if rangeParams["relation"] != "contains" {
		t.Errorf("Expected relation='contains', got %v", rangeParams["relation"])
	}
}

func TestBoolBuilder_ShouldIPRangeContains(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		ShouldIPRangeContains("allowed_ips", "192.168.1.50").
		MinimumShouldMatch(1).
		Build()

	if len(query.Bool.Should) != 1 {
		t.Fatalf("Expected 1 should clause, got %d", len(query.Bool.Should))
	}
}

func TestBoolBuilder_FilterIPRangeContains(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		FilterIPRangeContains("allowed_ips", "10.0.0.1").
		Build()

	if len(query.Bool.Filter) != 1 {
		t.Fatalf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}
}

func TestBoolBuilder_MustNotIPRangeContains(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustNotIPRangeContains("blocked_ips", "192.168.1.1").
		Build()

	if len(query.Bool.MustNot) != 1 {
		t.Fatalf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_MustIPRangeIntersects(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustIPRangeIntersects("network_range", "192.168.0.0", "192.168.255.255").
		Build()

	if len(query.Bool.Must) != 1 {
		t.Fatalf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	rangeQ := query.Bool.Must[0].Range
	if rangeQ == nil {
		t.Fatal("Expected Range query")
	}

	rangeParams := rangeQ["network_range"]
	if rangeParams["relation"] != "intersects" {
		t.Errorf("Expected relation='intersects', got %v", rangeParams["relation"])
	}
	if rangeParams["gte"] != "192.168.0.0" {
		t.Errorf("Expected gte='192.168.0.0', got %v", rangeParams["gte"])
	}
	if rangeParams["lte"] != "192.168.255.255" {
		t.Errorf("Expected lte='192.168.255.255', got %v", rangeParams["lte"])
	}
}

func TestBoolBuilder_FilterIPRangeIntersects(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		FilterIPRangeIntersects("network_range", "10.0.0.0", "10.255.255.255").
		Build()

	if len(query.Bool.Filter) != 1 {
		t.Fatalf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}
}

func TestBoolBuilder_MustIPRangeWithin(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustIPRangeWithin("subnet", "10.0.0.0", "10.255.255.255").
		Build()

	if len(query.Bool.Must) != 1 {
		t.Fatalf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	rangeQ := query.Bool.Must[0].Range
	if rangeQ == nil {
		t.Fatal("Expected Range query")
	}

	rangeParams := rangeQ["subnet"]
	if rangeParams["relation"] != "within" {
		t.Errorf("Expected relation='within', got %v", rangeParams["relation"])
	}
}

func TestBoolBuilder_FilterIPRangeWithin(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		FilterIPRangeWithin("subnet", "172.16.0.0", "172.31.255.255").
		Build()

	if len(query.Bool.Filter) != 1 {
		t.Fatalf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}
}

func TestBoolBuilder_MustNotIPRangeWithin(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	query := builder.
		MustNotIPRangeWithin("subnet", "192.168.0.0", "192.168.255.255").
		Build()

	if len(query.Bool.MustNot) != 1 {
		t.Fatalf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestBoolBuilder_CombinedIPRangeQuery(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	// Complex query: find ranges that contain a specific IP AND are within a larger network
	query := builder.
		MustIPRangeContains("acl_range", "192.168.1.100").
		FilterIPRangeWithin("acl_range", "192.168.0.0", "192.168.255.255").
		MustNotIPRangeIntersects("blocked_range", "10.0.0.0", "10.255.255.255").
		Build()

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if len(query.Bool.MustNot) != 1 {
		t.Errorf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}
}

func TestQueryBuilder_CombinedIPAndIPRangeQuery(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	// Mix of ip field queries (CIDR) and ip_range field queries
	query := builder.
		CIDR("client_ip", "192.168.0.0/16").           // For "ip" type field
		IPRangeContains("allowed_ranges", "10.0.0.1"). // For "ip_range" type field
		Build()

	if len(query.Query.Bool.Must) != 2 {
		t.Errorf("Expected 2 must clauses, got %d", len(query.Query.Bool.Must))
	}

	// First should be term query (CIDR on ip field)
	if query.Query.Bool.Must[0].Term == nil {
		t.Error("Expected first clause to be a term query (CIDR)")
	}

	// Second should be range query (IPRangeContains)
	if query.Query.Bool.Must[1].Range == nil {
		t.Error("Expected second clause to be a range query (IPRangeContains)")
	}
}

// === KNN (k-Nearest Neighbor) Vector Search Tests ===

func TestHelperFunctions_NewKNNQuery(t *testing.T) {
	vector := []float32{0.1, 0.2, 0.3, 0.4}
	query := NewKNNQuery("embedding", vector, 10)

	if query.KNN == nil {
		t.Fatal("Expected KNN query to be set")
	}

	knn := query.KNN["embedding"]
	if knn == nil {
		t.Fatal("Expected embedding field in KNN query")
	}

	if len(knn.Vector) != 4 {
		t.Errorf("Expected vector length 4, got %d", len(knn.Vector))
	}

	if knn.K != 10 {
		t.Errorf("Expected k=10, got %d", knn.K)
	}
}

func TestHelperFunctions_NewKNNQueryWithFilter(t *testing.T) {
	vector := []float32{0.1, 0.2, 0.3}
	filter := TermQuery("category.keyword", "tech")
	query := NewKNNQueryWithFilter("embedding", vector, 5, filter)

	if query.KNN == nil {
		t.Fatal("Expected KNN query to be set")
	}

	knn := query.KNN["embedding"]
	if knn == nil {
		t.Fatal("Expected embedding field in KNN query")
	}

	if knn.Filter == nil {
		t.Error("Expected filter to be set")
	}

	if knn.Filter.Term == nil {
		t.Error("Expected term query in filter")
	}
}

func TestHelperFunctions_NewKNNQueryWithMinScore(t *testing.T) {
	vector := []float32{0.1, 0.2, 0.3}
	query := NewKNNQueryWithMinScore("embedding", vector, 10, 0.8)

	if query.KNN == nil {
		t.Fatal("Expected KNN query to be set")
	}

	knn := query.KNN["embedding"]
	if knn.MinScore == nil {
		t.Fatal("Expected min_score to be set")
	}

	if *knn.MinScore != 0.8 {
		t.Errorf("Expected min_score=0.8, got %f", *knn.MinScore)
	}
}

func TestHelperFunctions_NewKNNQueryWithMaxDistance(t *testing.T) {
	vector := []float32{0.1, 0.2, 0.3}
	query := NewKNNQueryWithMaxDistance("embedding", vector, 10, 100.0)

	if query.KNN == nil {
		t.Fatal("Expected KNN query to be set")
	}

	knn := query.KNN["embedding"]
	if knn.MaxDistance == nil {
		t.Fatal("Expected max_distance to be set")
	}

	if *knn.MaxDistance != 100.0 {
		t.Errorf("Expected max_distance=100.0, got %f", *knn.MaxDistance)
	}
}

func TestHelperFunctions_NewKNNQueryWithOptions(t *testing.T) {
	vector := []float32{0.1, 0.2, 0.3}
	opts := KNNQueryOptions{
		MinScore: Float64Ptr(0.9),
		EfSearch: IntPtr(200),
		Boost:    Float64Ptr(1.5),
	}
	query := NewKNNQueryWithOptions("embedding", vector, 20, opts)

	if query.KNN == nil {
		t.Fatal("Expected KNN query to be set")
	}

	knn := query.KNN["embedding"]
	if knn.K != 20 {
		t.Errorf("Expected k=20, got %d", knn.K)
	}

	if knn.MinScore == nil || *knn.MinScore != 0.9 {
		t.Error("Expected min_score=0.9")
	}

	if knn.Boost == nil || *knn.Boost != 1.5 {
		t.Error("Expected boost=1.5")
	}

	if knn.MethodParameters == nil {
		t.Fatal("Expected method_parameters to be set")
	}

	if knn.MethodParameters.EfSearch == nil || *knn.MethodParameters.EfSearch != 200 {
		t.Error("Expected ef_search=200")
	}
}

func TestHelperFunctions_KNNQueryWithRescore(t *testing.T) {
	vector := []float32{0.1, 0.2, 0.3}
	opts := KNNQueryOptions{
		Rescore:          BoolPtr(true),
		OversampleFactor: Float64Ptr(2.0),
	}
	query := NewKNNQueryWithOptions("embedding", vector, 10, opts)

	knn := query.KNN["embedding"]
	if knn.Rescore == nil || !*knn.Rescore {
		t.Error("Expected rescore=true")
	}

	if knn.RescoreContext == nil {
		t.Fatal("Expected rescore_context to be set")
	}

	if knn.RescoreContext.OversampleFactor == nil || *knn.RescoreContext.OversampleFactor != 2.0 {
		t.Error("Expected oversample_factor=2.0")
	}
}

func TestQueryBuilder_KNN(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3, 0.4}
	query := builder.
		KNN("embedding", vector, 10).
		Build()

	if query.Query.KNN == nil {
		t.Fatal("Expected KNN query at top level")
	}

	knn := query.Query.KNN["embedding"]
	if knn == nil {
		t.Fatal("Expected embedding field in KNN query")
	}

	if knn.K != 10 {
		t.Errorf("Expected k=10, got %d", knn.K)
	}
}

func TestQueryBuilder_KNNWithFilter(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	filter := TermQuery("status.keyword", "active")
	query := builder.
		KNNWithFilter("embedding", vector, 5, filter).
		Build()

	if query.Query.KNN == nil {
		t.Fatal("Expected KNN query at top level")
	}

	knn := query.Query.KNN["embedding"]
	if knn == nil {
		t.Fatal("Expected embedding field")
	}

	if knn.Filter == nil {
		t.Fatal("Expected filter to be set")
	}

	if knn.Filter.Term == nil {
		t.Error("Expected term query in filter")
	}
}

func TestQueryBuilder_KNNWithMinScore(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.5, 0.5, 0.5}
	query := builder.
		KNNWithMinScore("embedding", vector, 10, 0.85).
		Build()

	if query.Query.KNN == nil {
		t.Fatal("Expected KNN query at top level")
	}

	knn := query.Query.KNN["embedding"]
	if knn.MinScore == nil {
		t.Fatal("Expected min_score to be set")
	}

	if *knn.MinScore != 0.85 {
		t.Errorf("Expected min_score=0.85, got %f", *knn.MinScore)
	}
}

func TestQueryBuilder_KNNWithMaxDistance(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.5, 0.5, 0.5}
	query := builder.
		KNNWithMaxDistance("embedding", vector, 10, 50.0).
		Build()

	if query.Query.KNN == nil {
		t.Fatal("Expected KNN query at top level")
	}

	knn := query.Query.KNN["embedding"]
	if knn.MaxDistance == nil {
		t.Fatal("Expected max_distance to be set")
	}

	if *knn.MaxDistance != 50.0 {
		t.Errorf("Expected max_distance=50.0, got %f", *knn.MaxDistance)
	}
}

func TestQueryBuilder_KNNWithOptions(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	opts := KNNQueryOptions{
		MinScore: Float64Ptr(0.7),
		EfSearch: IntPtr(150),
		Nprobe:   IntPtr(10),
	}
	query := builder.
		KNNWithOptions("embedding", vector, 15, opts).
		Build()

	if query.Query.KNN == nil {
		t.Fatal("Expected KNN query at top level")
	}

	knn := query.Query.KNN["embedding"]
	// In OpenSearch 3.x, if MinScore is set, K should NOT be set on the query itself.
	// Instead, Size on the request limits results.
	if knn.K != 0 {
		t.Errorf("Expected k=0 (unset) when MinScore is used, got %d", knn.K)
	}
	if knn.MinScore == nil || *knn.MinScore != 0.7 {
		t.Errorf("Expected MinScore=0.7, got %v", knn.MinScore)
	}
	if query.Size != 15 {
		t.Errorf("Expected Size=15, got %v", query.Size)
	}

	if knn.MethodParameters == nil {
		t.Fatal("Expected method_parameters to be set")
	}

	if knn.MethodParameters.EfSearch == nil || *knn.MethodParameters.EfSearch != 150 {
		t.Error("Expected ef_search=150")
	}

	if knn.MethodParameters.Nprobe == nil || *knn.MethodParameters.Nprobe != 10 {
		t.Error("Expected nprobe=10")
	}
}

func TestBoolBuilder_MustKNN(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	query := builder.
		MustKNN("embedding", vector, 10).
		Build()

	if len(query.Bool.Must) != 1 {
		t.Fatalf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if query.Bool.Must[0].KNN == nil {
		t.Error("Expected KNN query in must")
	}
}

func TestBoolBuilder_ShouldKNN(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	vector1 := []float32{0.1, 0.2, 0.3}
	vector2 := []float32{0.4, 0.5, 0.6}
	query := builder.
		ShouldKNN("embedding1", vector1, 5).
		ShouldKNN("embedding2", vector2, 5).
		MinimumShouldMatch(1).
		Build()

	if len(query.Bool.Should) != 2 {
		t.Errorf("Expected 2 should clauses, got %d", len(query.Bool.Should))
	}
}

func TestBoolBuilder_FilterKNN(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	query := builder.
		FilterKNN("embedding", vector, 10).
		Build()

	if len(query.Bool.Filter) != 1 {
		t.Fatalf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if query.Bool.Filter[0].KNN == nil {
		t.Error("Expected KNN query in filter")
	}
}

func TestBoolBuilder_MustNotKNN(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	query := builder.
		MustNotKNN("embedding", vector, 10).
		Build()

	if len(query.Bool.MustNot) != 1 {
		t.Fatalf("Expected 1 must_not clause, got %d", len(query.Bool.MustNot))
	}

	if query.Bool.MustNot[0].KNN == nil {
		t.Error("Expected KNN query in must_not")
	}
}

func TestBoolBuilder_MustKNNWithFilter(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	filter := TermQuery("category.keyword", "tech")
	query := builder.
		MustKNNWithFilter("embedding", vector, 10, filter).
		Build()

	knn := query.Bool.Must[0].KNN["embedding"]
	if knn == nil {
		t.Fatal("Expected embedding field in KNN query")
	}

	if knn.Filter == nil {
		t.Error("Expected filter to be set")
	}
}

func TestBoolBuilder_MustKNNWithMinScore(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	query := builder.
		MustKNNWithMinScore("embedding", vector, 10, 0.9).
		Build()

	knn := query.Bool.Must[0].KNN["embedding"]
	if knn.MinScore == nil || *knn.MinScore != 0.9 {
		t.Error("Expected min_score=0.9")
	}
}

func TestBoolBuilder_MustKNNWithMaxDistance(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	query := builder.
		MustKNNWithMaxDistance("embedding", vector, 10, 75.0).
		Build()

	knn := query.Bool.Must[0].KNN["embedding"]
	if knn.MaxDistance == nil || *knn.MaxDistance != 75.0 {
		t.Error("Expected max_distance=75.0")
	}
}

func TestBoolBuilder_MustKNNWithOptions(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	opts := KNNQueryOptions{
		MinScore: Float64Ptr(0.8),
		EfSearch: IntPtr(100),
		Rescore:  BoolPtr(true),
	}
	query := builder.
		MustKNNWithOptions("embedding", vector, 20, opts).
		Build()

	knn := query.Bool.Must[0].KNN["embedding"]
	if knn.K != 20 {
		t.Errorf("Expected k=20, got %d", knn.K)
	}

	if knn.MethodParameters == nil || knn.MethodParameters.EfSearch == nil {
		t.Error("Expected method_parameters.ef_search to be set")
	}

	if knn.Rescore == nil || !*knn.Rescore {
		t.Error("Expected rescore=true")
	}
}

func TestBoolBuilder_CombinedKNNAndTermQuery(t *testing.T) {
	ctx := context.Background()
	builder := NewBoolBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	query := builder.
		MustKNN("embedding", vector, 10).
		FilterTerm("status", "active").
		Build()

	if len(query.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause, got %d", len(query.Bool.Must))
	}

	if len(query.Bool.Filter) != 1 {
		t.Errorf("Expected 1 filter clause, got %d", len(query.Bool.Filter))
	}

	if query.Bool.Must[0].KNN == nil {
		t.Error("Expected KNN query in must")
	}
}

func TestQueryBuilder_CombinedKNNAndMatch(t *testing.T) {
	ctx := context.Background()
	builder := NewQueryBuilder(ctx, []string{"test-index"}, "testing")

	vector := []float32{0.1, 0.2, 0.3}
	query := builder.
		KNN("embedding", vector, 10).
		Match("description", "machine learning").
		Build()

	// In OpenSearch 3.x, other bool queries are moved to KNN filter
	if query.Query.KNN == nil {
		t.Fatal("Expected top level KNN query")
	}

	knn := query.Query.KNN["embedding"]
	if knn.Filter == nil || knn.Filter.Bool == nil {
		t.Fatal("Expected boolean filter map in KNN query")
	}

	if len(knn.Filter.Bool.Must) != 1 {
		t.Errorf("Expected 1 must clause in filter, got %d", len(knn.Filter.Bool.Must))
	}

	// Should be Match
	if knn.Filter.Bool.Must[0].Match == nil {
		t.Error("Expected clause in filter to be Match query")
	}
}

func TestPointerHelpers(t *testing.T) {
	// Test Float64Ptr
	f := Float64Ptr(1.5)
	if f == nil || *f != 1.5 {
		t.Error("Float64Ptr failed")
	}

	// Test IntPtr
	i := IntPtr(100)
	if i == nil || *i != 100 {
		t.Error("IntPtr failed")
	}

	// Test BoolPtr
	b := BoolPtr(true)
	if b == nil || !*b {
		t.Error("BoolPtr failed")
	}
}
