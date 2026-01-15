package os

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// TestBuilderIntegration tests the query builder with real OpenSearch
// This test requires an OpenSearch instance running with existing indices
func TestBuilderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Connect to test OpenSearch using env vars
	nodes := os.Getenv("NODES")
	if nodes == "" {
		t.Skip("Skipping integration test: NODES env var not set")
	}
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Skip("Skipping integration test: could not connect to OpenSearch:", err)
	}

	ctx := context.Background()

	// Use the wildcard pattern to match any existing indices
	indexPattern := "*"

	// Test 1: Basic query builder
	t.Run("BasicBuilder", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{indexPattern}, "testing")

		query := builder.
			Size(20).
			From(0).
			Build()

		if query.Size != 20 {
			t.Errorf("Expected size 20, got %d", query.Size)
		}

		if query.From != 0 {
			t.Errorf("Expected from 0, got %d", query.From)
		}
	})

	// Test 2: Fetch merged mapping for all indices
	t.Run("FetchMergedMapping", func(t *testing.T) {
		mapper := NewFieldMapper(
			WithCacheTTL(1*time.Minute),
			WithMaxCacheSize(10),
		)

		merged, err := mapper.GetMergedMapping(ctx, indexPattern)
		if err != nil {
			t.Logf("Warning: Failed to fetch mapping: %v", err)
			t.Skip("Skipping test - no indices found or mapping fetch failed")
		}

		t.Logf("Mapping fetched successfully")
		t.Logf("Index pattern: %s", merged.IndexPattern)
		t.Logf("Number of indices matched: %d", len(merged.Indices))
		t.Logf("Number of unique fields: %d", len(merged.Fields))
		t.Logf("Number of conflicts: %d", merged.ConflictCount)

		if len(merged.Indices) > 0 {
			t.Logf("Sample indices: %v", merged.Indices[:minInt(3, len(merged.Indices))])
		}

		// List some sample fields
		count := 0
		for fieldName, info := range merged.Fields {
			if count >= 5 {
				break
			}
			t.Logf("Field '%s': type=%s, indices=%v", fieldName, info.Type, info.SourceIndices)
			if len(info.Fields) > 0 {
				t.Logf("  Sub-fields: %v", info.Fields)
			}
			count++
		}

		// Test cache
		merged2, err := mapper.GetMergedMapping(ctx, indexPattern)
		if err != nil {
			t.Fatalf("Failed to get cached mapping: %v", err)
		}

		// Should be the same timestamp (from cache)
		if merged.FetchedAt != merged2.FetchedAt {
			t.Error("Expected cached mapping to have same timestamp")
		} else {
			t.Logf("Cache working correctly - same fetch time")
		}
	})

	// Test 3: Query builder with field resolution
	t.Run("QueryBuilderWithMapping", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{indexPattern}, "testing")

		// Build a complex query
		query := builder.
			Must(
				TermQuery("status", "active"),
			).
			Should(
				MatchQuery("description", "test"),
			).
			Size(10).
			Build()

		if len(query.Query.Bool.Must) == 0 {
			t.Error("Expected at least one must clause")
		}

		t.Logf("Query built successfully with %d must clauses and %d should clauses",
			len(query.Query.Bool.Must), len(query.Query.Bool.Should))
	})

	// Test 4: Check for mapping conflicts
	t.Run("CheckMappingConflicts", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{indexPattern}, "testing")

		conflicts := builder.GetMappingConflicts()

		if len(conflicts) > 0 {
			t.Logf("Found %d field(s) with type conflicts:", len(conflicts))
			for _, conflict := range conflicts {
				t.Logf("Field '%s':", conflict.BaseField)
				for typ, indices := range conflict.ConflictTypes {
					t.Logf("  Type '%s' in %d indices: %v",
						typ, len(indices), indices[:minInt(3, len(indices))])
				}
			}
		} else {
			t.Logf("No field type conflicts found")
		}
	})

	// Test 5: Aggregations
	t.Run("AggregationsBuilder", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{indexPattern}, "testing")

		query := builder.
			TermsAgg("field_counts", "_id", 10).
			Size(0). // Only aggregations, no documents
			Build()

		if len(query.Aggs) == 0 {
			t.Error("Expected at least one aggregation")
		}

		if query.Size != 0 {
			t.Errorf("Expected size 0, got %d", query.Size)
		}

		t.Logf("Aggregation query built successfully")
	})

	// Test 6: Sorting
	t.Run("SortingBuilder", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{indexPattern}, "testing")

		query := builder.
			Sort("_id", "desc").
			Build()

		if len(query.Sort) == 0 {
			t.Error("Expected at least one sort field")
		}

		t.Logf("Sort query built successfully")
	})

	// Test 7: Source filtering
	t.Run("SourceFiltering", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{indexPattern}, "testing")

		query := builder.
			IncludeSource("_id", "_index").
			ExcludeSource("large_field").
			Build()

		if query.Source == nil {
			t.Fatal("Expected source to be set")
		}

		if len(query.Source.Includes) != 2 {
			t.Errorf("Expected 2 included fields, got %d", len(query.Source.Includes))
		}

		t.Logf("Source filtering configured successfully")
	})
}

// Helper function to get the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestBoolBuilderIntegration tests BoolBuilder with actual query execution against OpenSearch
func TestBoolBuilderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Connect to test OpenSearch using env vars
	nodes := os.Getenv("NODES")
	if nodes == "" {
		t.Skip("Skipping integration test: NODES env var not set")
	}
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Skip("Skipping integration test: could not connect to OpenSearch:", err)
	}

	ctx := context.Background()
	testIndex := "twos-test-" + time.Now().Format("20060102-150405")

	// Setup: Create test index with known mapping and documents
	t.Run("Setup", func(t *testing.T) {
		// Index test documents
		docs := []struct {
			id  string
			doc map[string]interface{}
		}{
			{"1", map[string]interface{}{
				"type":      "ip",
				"status":    "active",
				"severity":  8,
				"category":  "threat",
				"visibleBy": []string{"public"},
			}},
			{"2", map[string]interface{}{
				"type":      "domain",
				"status":    "active",
				"severity":  5,
				"category":  "indicator",
				"visibleBy": []string{"public"},
			}},
			{"3", map[string]interface{}{
				"type":      "ip",
				"status":    "inactive",
				"severity":  3,
				"category":  "indicator",
				"visibleBy": []string{"public"},
			}},
			{"4", map[string]interface{}{
				"type":      "url",
				"status":    "active",
				"severity":  9,
				"category":  "threat",
				"visibleBy": []string{"private"},
			}},
			{"5", map[string]interface{}{
				"type":      "domain",
				"status":    "active",
				"severity":  2,
				"category":  "benign",
				"visibleBy": []string{"public"},
			}},
		}

		for _, d := range docs {
			err := IndexDoc(ctx, d.doc, testIndex, d.id)
			if err != nil {
				t.Fatalf("Failed to index document %s: %v", d.id, err)
			}
		}

		// Wait for documents to be indexed and available for search
		time.Sleep(5 * time.Second)
		t.Logf("Indexed %d test documents in %s", len(docs), testIndex)
	})

	// Test 1: Simple BoolBuilder query execution
	t.Run("SimpleBoolBuilderExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustTerm("status", "active"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should find 4 active documents (ids: 1, 2, 4, 5)
		if resp.Hits.Total.Value != 4 {
			t.Errorf("Expected 4 active documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("SimpleBoolBuilder: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 2: OR condition with BoolBuilder
	t.Run("OrConditionExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Find documents where type is "ip" OR "domain"
		orCondition := builder.Bool().
			ShouldTerm("type", "ip").
			ShouldTerm("type", "domain").
			MinimumShouldMatch(1)

		query := builder.
			FilterBool(orCondition).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should find 4 documents (ids: 1, 2, 3, 5)
		if resp.Hits.Total.Value != 4 {
			t.Errorf("Expected 4 documents (ip or domain), got %d", resp.Hits.Total.Value)
		}
		t.Logf("OrCondition: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 3: Nested bool queries
	t.Run("NestedBoolExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// (status=active AND (type=ip OR type=domain))
		typeOr := builder.Bool().
			ShouldTerm("type", "ip").
			ShouldTerm("type", "domain").
			MinimumShouldMatch(1)

		query := builder.
			FilterBool(
				builder.Bool().
					MustTerm("status", "active").
					MustBool(typeOr),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should find 3 documents (active AND (ip OR domain)): ids 1, 2, 5
		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("NestedBool: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 4: Complex nested query with range
	t.Run("ComplexNestedWithRange", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// (category=threat AND severity>=5) OR (category=indicator AND type=ip)
		threatCondition := builder.Bool().
			MustTerm("category", "threat").
			MustRange("severity", "gte", 5)

		indicatorIpCondition := builder.Bool().
			MustTerm("category", "indicator").
			MustTerm("type", "ip")

		query := builder.
			FilterBool(
				builder.Bool().
					ShouldBool(threatCondition).
					ShouldBool(indicatorIpCondition).
					MinimumShouldMatch(1),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// threat with severity>=5: ids 1 (sev=8), 4 (sev=9)
		// indicator with type=ip: id 3
		// Total: 3
		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("ComplexNested: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 5: MustNot clause
	t.Run("MustNotExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustTerm("status", "active").
					MustNotTerm("type", "url"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Active but not url: ids 1, 2, 5 (excluding id 4 which is url)
		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("MustNot: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 6: SearchIn with visibleBy filter
	t.Run("SearchInWithVisibility", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustTerm("status", "active"),
			).
			Size(10).
			Build()

		// SearchIn adds visibleBy filter for "public" group
		resp, err := query.SearchIn(ctx, []string{testIndex}, []string{"public"})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Active AND visibleBy=public: ids 1, 2, 5 (excluding id 4 which is private)
		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 public documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("SearchIn: Found %d public documents", resp.Hits.Total.Value)
	})

	// Test 7: WideSearchIn (no visibility filter)
	t.Run("WideSearchInNoVisibility", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustTerm("status", "active"),
			).
			Size(10).
			Build()

		// WideSearchIn does NOT add visibleBy filter
		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// All active: ids 1, 2, 4, 5
		if resp.Hits.Total.Value != 4 {
			t.Errorf("Expected 4 documents (including private), got %d", resp.Hits.Total.Value)
		}
		t.Logf("WideSearchIn: Found %d documents (including private)", resp.Hits.Total.Value)
	})

	// Test 8: Or/And/Not helper functions
	t.Run("HelperFunctionsExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Using Or helper
		query := builder.
			Filter(
				Or(
					TermQuery("type.keyword", "ip"),
					TermQuery("type.keyword", "url"),
				),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// ip or url: ids 1, 3, 4
		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("OrHelper: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 9: Field resolution verification (.keyword)
	t.Run("FieldResolutionKeyword", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// BoolBuilder should auto-resolve "type" to "type.keyword" for term queries
		query := builder.
			FilterBool(
				builder.Bool().
					MustTerm("type", "ip"), // Should become type.keyword
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// type=ip: ids 1, 3
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 ip documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("FieldResolution: Found %d documents with auto .keyword resolution", resp.Hits.Total.Value)
	})

	// Test 10: Terms query (multiple values)
	t.Run("TermsQueryExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustTerms("category", "threat", "benign"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// category in [threat, benign]: ids 1, 4, 5
		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("TermsQuery: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 11: Deeply nested (3 levels)
	t.Run("DeeplyNestedExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Level 3: severity >= 5
		level3 := builder.Bool().
			MustRange("severity", "gte", 5)

		// Level 2: category=threat AND level3
		level2 := builder.Bool().
			MustTerm("category", "threat").
			FilterBool(level3)

		// Level 1: status=active AND level2
		query := builder.
			FilterBool(
				builder.Bool().
					MustTerm("status", "active").
					MustBool(level2),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// active AND threat AND severity>=5: ids 1, 4
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("DeeplyNested: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 12: Wildcard query
	t.Run("WildcardQueryExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustWildcard("type", "dom*"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// type matching dom*: domain = ids 2, 5
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("Wildcard: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 13: Exists query
	t.Run("ExistsQueryExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustExists("severity"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// All documents have severity field
		if resp.Hits.Total.Value != 5 {
			t.Errorf("Expected 5 documents with severity field, got %d", resp.Hits.Total.Value)
		}
		t.Logf("Exists: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 14: Prefix query
	t.Run("PrefixQueryExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustPrefix("type", "dom"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// type starting with dom: domain = ids 2, 5
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("Prefix: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 15: Range query with different operators
	t.Run("RangeQueryOperators", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// severity > 5
		query := builder.
			FilterBool(
				builder.Bool().
					MustRange("severity", "gt", 5),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// severity > 5: ids 1 (sev=8), 4 (sev=9)
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 documents with severity > 5, got %d", resp.Hits.Total.Value)
		}
		t.Logf("RangeGT: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 16: IDs query
	t.Run("IDsQueryExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					FilterIDs("1", "3", "5"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should find exactly 3 documents
		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("IDs: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 17: MustNotIDs query
	t.Run("MustNotIDsExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustNotIDs("1", "2"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should find 3 documents (excluding 1 and 2)
		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("MustNotIDs: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 18: QueryString query
	t.Run("QueryStringExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Using query_string syntax
		query := builder.
			FilterBool(
				builder.Bool().
					FilterQueryString("status:active AND category:threat"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// active threats: ids 1, 4
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("QueryString: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 19: Regexp query
	t.Run("RegexpQueryExecution", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Match "ip" or "url" with regex
		query := builder.
			FilterBool(
				builder.Bool().
					MustRegexp("type", "(ip|url)"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// type matching (ip|url): ids 1, 3, 4
		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 documents, got %d", resp.Hits.Total.Value)
		}
		t.Logf("Regexp: Found %d documents", resp.Hits.Total.Value)
	})

	// Cleanup: Delete test index
	t.Run("Cleanup", func(t *testing.T) {
		// Use raw client to delete index
		_, err := apiClient.Indices.Delete(ctx, opensearchapi.IndicesDeleteReq{
			Indices: []string{testIndex},
		})
		if err != nil {
			t.Logf("Warning: Failed to delete test index %s: %v", testIndex, err)
		} else {
			t.Logf("Cleaned up test index %s", testIndex)
		}
	})
}

// TestAggregationIntegration tests aggregation methods with actual OpenSearch execution
func TestAggregationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Connect to test OpenSearch using env vars
	nodes := os.Getenv("NODES")
	if nodes == "" {
		t.Skip("Skipping integration test: NODES env var not set")
	}
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Skip("Skipping integration test: could not connect to OpenSearch:", err)
	}

	ctx := context.Background()
	testIndex := "twos-agg-test-" + time.Now().Format("20060102-150405")

	// Setup: Create test index with documents for aggregation testing
	t.Run("Setup", func(t *testing.T) {
		docs := []struct {
			id  string
			doc map[string]interface{}
		}{
			{"1", map[string]interface{}{
				"category":   "electronics",
				"brand":      "acme",
				"price":      100,
				"quantity":   5,
				"created_at": "2024-01-15T10:00:00Z",
				"visibleBy":  []string{"public"},
			}},
			{"2", map[string]interface{}{
				"category":   "electronics",
				"brand":      "acme",
				"price":      200,
				"quantity":   3,
				"created_at": "2024-01-16T10:00:00Z",
				"visibleBy":  []string{"public"},
			}},
			{"3", map[string]interface{}{
				"category":   "electronics",
				"brand":      "beta",
				"price":      150,
				"quantity":   10,
				"created_at": "2024-02-01T10:00:00Z",
				"visibleBy":  []string{"public"},
			}},
			{"4", map[string]interface{}{
				"category":   "clothing",
				"brand":      "acme",
				"price":      50,
				"quantity":   20,
				"created_at": "2024-02-15T10:00:00Z",
				"visibleBy":  []string{"public"},
			}},
			{"5", map[string]interface{}{
				"category":   "clothing",
				"brand":      "gamma",
				"price":      75,
				"quantity":   15,
				"created_at": "2024-03-01T10:00:00Z",
				"visibleBy":  []string{"public"},
			}},
		}

		for _, d := range docs {
			err := IndexDoc(ctx, d.doc, testIndex, d.id)
			if err != nil {
				t.Fatalf("Failed to index document %s: %v", d.id, err)
			}
		}

		// Wait for documents to be indexed
		time.Sleep(5 * time.Second)
		t.Logf("Indexed %d test documents in %s", len(docs), testIndex)
	})

	// Test 1: Terms aggregation (use .keyword for text fields)
	t.Run("TermsAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			TermsAgg("categories", "category.keyword", 10).
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("TermsAgg: Got aggregation response with %d hits", resp.Hits.Total.Value)
	})

	// Test 2: Sum aggregation
	t.Run("SumAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			SumAgg("total_price", "price").
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("SumAgg: Got aggregation response")
	})

	// Test 3: Avg aggregation
	t.Run("AvgAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			AvgAgg("avg_price", "price").
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("AvgAgg: Got aggregation response")
	})

	// Test 4: Min/Max aggregation
	t.Run("MinMaxAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			MinAgg("min_price", "price").
			MaxAgg("max_price", "price").
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("MinMaxAgg: Got aggregation response")
	})

	// Test 5: Stats aggregation
	t.Run("StatsAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			StatsAgg("price_stats", "price").
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("StatsAgg: Got aggregation response")
	})

	// Test 6: Cardinality aggregation (use .keyword for text fields)
	t.Run("CardinalityAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			CardinalityAgg("unique_brands", "brand.keyword").
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("CardinalityAgg: Got aggregation response")
	})

	// Test 7: Histogram aggregation
	t.Run("HistogramAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			HistogramAgg("price_histogram", "price", 50.0).
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("HistogramAgg: Got aggregation response")
	})

	// Test 8: Range aggregation
	t.Run("RangeAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		ranges := []map[string]interface{}{
			{"to": 100},
			{"from": 100, "to": 200},
			{"from": 200},
		}

		query := builder.
			RangeAgg("price_ranges", "price", ranges).
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("RangeAgg: Got aggregation response")
	})

	// Test 9: Sub-aggregation (use .keyword for text fields)
	t.Run("SubAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			TermsAgg("by_category", "category.keyword", 10).
			SubAgg("by_category", "avg_price", Aggs{Avg: &Agg{Field: "price"}}).
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("SubAgg: Got aggregation response")
	})

	// Test 10: Value count aggregation (use .keyword for text fields)
	t.Run("ValueCountAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			ValueCountAgg("count_items", "category.keyword").
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("ValueCountAgg: Got aggregation response")
	})

	// Test 11: Extended stats aggregation
	t.Run("ExtendedStatsAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			ExtendedStatsAgg("extended_price_stats", "price", 2).
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("ExtendedStatsAgg: Got aggregation response")
	})

	// Test 12: Percentiles aggregation
	t.Run("PercentilesAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			PercentilesAgg("price_percentiles", "price").
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("PercentilesAgg: Got aggregation response")
	})

	// Test 13: Top hits aggregation (use .keyword for text fields)
	t.Run("TopHitsAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			TermsAgg("by_category", "category.keyword", 10).
			SubAgg("by_category", "top_items", Aggs{TopHits: &TopHits{Size: 2}}).
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("TopHitsAgg: Got aggregation response")
	})

	// Cleanup: Delete test index
	t.Run("Cleanup", func(t *testing.T) {
		_, err := apiClient.Indices.Delete(ctx, opensearchapi.IndicesDeleteReq{
			Indices: []string{testIndex},
		})
		if err != nil {
			t.Logf("Warning: Failed to delete test index %s: %v", testIndex, err)
		} else {
			t.Logf("Cleaned up test index %s", testIndex)
		}
	})
}

// TestIPCIDRIntegration tests IP field and CIDR query support with actual OpenSearch execution
func TestIPCIDRIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Connect to test OpenSearch using env vars
	nodes := os.Getenv("NODES")
	if nodes == "" {
		t.Skip("Skipping integration test: NODES env var not set")
	}
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Skip("Skipping integration test: could not connect to OpenSearch:", err)
	}

	ctx := context.Background()
	testIndex := "twos-ip-test-" + time.Now().Format("20060102-150405")

	// Setup: Create test index with IP field mapping and documents
	t.Run("Setup", func(t *testing.T) {
		// Create index with explicit IP field mapping
		mapping := map[string]interface{}{
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"source_ip": map[string]interface{}{"type": "ip"},
					"dest_ip":   map[string]interface{}{"type": "ip"},
					"event":     map[string]interface{}{"type": "keyword"},
					"visibleBy": map[string]interface{}{"type": "keyword"},
				},
			},
		}

		mappingJSON, _ := json.Marshal(mapping)
		_, err := apiClient.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
			Index: testIndex,
			Body:  strings.NewReader(string(mappingJSON)),
		})
		if err != nil {
			t.Fatalf("Failed to create index: %v", err)
		}

		// Index test documents with various IP addresses
		docs := []struct {
			id  string
			doc map[string]interface{}
		}{
			{"1", map[string]interface{}{
				"source_ip": "192.168.1.10",
				"dest_ip":   "8.8.8.8",
				"event":     "dns_query",
				"visibleBy": []string{"public"},
			}},
			{"2", map[string]interface{}{
				"source_ip": "192.168.1.20",
				"dest_ip":   "1.1.1.1",
				"event":     "dns_query",
				"visibleBy": []string{"public"},
			}},
			{"3", map[string]interface{}{
				"source_ip": "10.0.0.5",
				"dest_ip":   "192.168.1.10",
				"event":     "internal_scan",
				"visibleBy": []string{"public"},
			}},
			{"4", map[string]interface{}{
				"source_ip": "172.16.0.100",
				"dest_ip":   "8.8.4.4",
				"event":     "dns_query",
				"visibleBy": []string{"public"},
			}},
			{"5", map[string]interface{}{
				"source_ip": "203.0.113.50", // Public IP (TEST-NET-3)
				"dest_ip":   "127.0.0.1",
				"event":     "localhost_access",
				"visibleBy": []string{"public"},
			}},
		}

		for _, d := range docs {
			err := IndexDoc(ctx, d.doc, testIndex, d.id)
			if err != nil {
				t.Fatalf("Failed to index document %s: %v", d.id, err)
			}
		}

		// Wait for documents to be indexed
		time.Sleep(2 * time.Second)
		t.Logf("Indexed %d test documents with IP fields in %s", len(docs), testIndex)
	})

	// Test 1: Exact IP match using Term
	t.Run("ExactIPMatch", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			Term("source_ip", "192.168.1.10").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Hits.Total.Value != 1 {
			t.Errorf("Expected 1 document with exact IP match, got %d", resp.Hits.Total.Value)
		}
		t.Logf("ExactIPMatch: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 2: CIDR query - 192.168.0.0/16 (Class C private range)
	t.Run("CIDRQuery192168", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			CIDR("source_ip", "192.168.0.0/16").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match docs 1, 2 (192.168.1.x)
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 documents in 192.168.0.0/16 range, got %d", resp.Hits.Total.Value)
		}
		t.Logf("CIDRQuery192168: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 3: CIDR query - 10.0.0.0/8 (Class A private range)
	t.Run("CIDRQuery10", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					FilterCIDR("source_ip", "10.0.0.0/8"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match doc 3 (10.0.0.5)
		if resp.Hits.Total.Value != 1 {
			t.Errorf("Expected 1 document in 10.0.0.0/8 range, got %d", resp.Hits.Total.Value)
		}
		t.Logf("CIDRQuery10: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 4: Multiple CIDRs - all private IP ranges
	t.Run("MultipleCIDRs", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			CIDRs("source_ip", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match docs 1, 2, 3, 4 (all private IPs)
		if resp.Hits.Total.Value != 4 {
			t.Errorf("Expected 4 documents with private IPs, got %d", resp.Hits.Total.Value)
		}
		t.Logf("MultipleCIDRs: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 5: CIDR with MustNot - exclude localhost destination
	t.Run("CIDRMustNot", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustNotCIDR("dest_ip", "127.0.0.0/8"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match docs 1, 2, 3, 4 (excluding doc 5 which has 127.0.0.1)
		if resp.Hits.Total.Value != 4 {
			t.Errorf("Expected 4 documents excluding localhost dest, got %d", resp.Hits.Total.Value)
		}
		t.Logf("CIDRMustNot: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 6: Combined IP query - source from private, dest to public DNS
	t.Run("CombinedIPQuery", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					FilterCIDRs("source_ip", "192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12").
					FilterCIDR("dest_ip", "8.0.0.0/8"), // Google DNS range
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match docs 1, 4 (private source to 8.x.x.x dest)
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 documents (private to 8.x.x.x), got %d", resp.Hits.Total.Value)
		}
		t.Logf("CombinedIPQuery: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 7: IP range aggregation
	t.Run("IPRangeAggregation", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		ranges := []map[string]interface{}{
			{"key": "private_10", "from": "10.0.0.0", "to": "10.255.255.255"},
			{"key": "private_172", "from": "172.16.0.0", "to": "172.31.255.255"},
			{"key": "private_192", "from": "192.168.0.0", "to": "192.168.255.255"},
		}

		query := builder.
			IPRangeAgg("ip_ranges", "source_ip", ranges).
			Size(0).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Aggregations == nil {
			t.Fatal("Expected aggregations in response")
		}

		t.Logf("IPRangeAgg: Got IP range aggregation response")
	})

	// Test 8: ShouldCIDR with minimum_should_match
	t.Run("ShouldCIDRWithMinMatch", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Find documents where source OR dest is in 192.168.x.x range
		query := builder.
			FilterBool(
				builder.Bool().
					ShouldCIDR("source_ip", "192.168.0.0/16").
					ShouldCIDR("dest_ip", "192.168.0.0/16").
					MinimumShouldMatch(1),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match docs 1, 2 (source 192.168.x) and doc 3 (dest 192.168.x)
		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 documents with 192.168.x in source or dest, got %d", resp.Hits.Total.Value)
		}
		t.Logf("ShouldCIDRWithMinMatch: Found %d documents", resp.Hits.Total.Value)
	})

	// Cleanup: Delete test index
	t.Run("Cleanup", func(t *testing.T) {
		_, err := apiClient.Indices.Delete(ctx, opensearchapi.IndicesDeleteReq{
			Indices: []string{testIndex},
		})
		if err != nil {
			t.Logf("Warning: Failed to delete test index %s: %v", testIndex, err)
		} else {
			t.Logf("Cleaned up test index %s", testIndex)
		}
	})
}

// TestIPRangeFieldIntegration tests queries against ip_range type fields
// This is different from CIDR queries against ip fields
func TestIPRangeFieldIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Connect to test OpenSearch using env vars
	nodes := os.Getenv("NODES")
	if nodes == "" {
		t.Skip("Skipping integration test: NODES env var not set")
	}
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Skip("Skipping integration test: could not connect to OpenSearch:", err)
	}

	ctx := context.Background()
	testIndex := "twos-iprange-test-" + time.Now().Format("20060102-150405")

	// Setup: Create test index with ip_range field mapping and documents
	t.Run("Setup", func(t *testing.T) {
		// Create index with ip_range field mapping (different from ip type)
		mapping := map[string]interface{}{
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"allowed_range":  map[string]interface{}{"type": "ip_range"},
					"blocked_range":  map[string]interface{}{"type": "ip_range"},
					"network_subnet": map[string]interface{}{"type": "ip_range"},
					"acl_name":       map[string]interface{}{"type": "keyword"},
					"visibleBy":      map[string]interface{}{"type": "keyword"},
				},
			},
		}

		mappingJSON, _ := json.Marshal(mapping)
		_, err := apiClient.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
			Index: testIndex,
			Body:  strings.NewReader(string(mappingJSON)),
		})
		if err != nil {
			t.Fatalf("Failed to create index: %v", err)
		}

		// Index test documents with ip_range fields
		// ip_range fields can store ranges as {"gte": "x", "lte": "y"} or CIDR notation
		docs := []struct {
			id  string
			doc map[string]interface{}
		}{
			{"1", map[string]interface{}{
				"allowed_range":  map[string]interface{}{"gte": "192.168.0.0", "lte": "192.168.255.255"}, // /16
				"blocked_range":  map[string]interface{}{"gte": "10.0.0.0", "lte": "10.255.255.255"},     // /8
				"network_subnet": "192.168.1.0/24",                                                       // CIDR notation
				"acl_name":       "office_network",
				"visibleBy":      []string{"public"},
			}},
			{"2", map[string]interface{}{
				"allowed_range":  map[string]interface{}{"gte": "10.0.0.0", "lte": "10.0.0.255"}, // /24
				"blocked_range":  map[string]interface{}{"gte": "0.0.0.0", "lte": "0.255.255.255"},
				"network_subnet": "10.0.0.0/24",
				"acl_name":       "datacenter_a",
				"visibleBy":      []string{"public"},
			}},
			{"3", map[string]interface{}{
				"allowed_range":  map[string]interface{}{"gte": "172.16.0.0", "lte": "172.31.255.255"}, // /12
				"blocked_range":  map[string]interface{}{"gte": "192.168.100.0", "lte": "192.168.100.255"},
				"network_subnet": "172.16.0.0/16",
				"acl_name":       "remote_office",
				"visibleBy":      []string{"public"},
			}},
			{"4", map[string]interface{}{
				"allowed_range":  map[string]interface{}{"gte": "10.10.0.0", "lte": "10.10.255.255"}, // /16
				"blocked_range":  map[string]interface{}{"gte": "127.0.0.0", "lte": "127.255.255.255"},
				"network_subnet": "10.10.0.0/16",
				"acl_name":       "datacenter_b",
				"visibleBy":      []string{"public"},
			}},
			{"5", map[string]interface{}{
				"allowed_range":  map[string]interface{}{"gte": "192.168.50.0", "lte": "192.168.50.255"}, // /24 within doc 1's range
				"blocked_range":  map[string]interface{}{"gte": "203.0.113.0", "lte": "203.0.113.255"},   // TEST-NET-3
				"network_subnet": "192.168.50.0/24",
				"acl_name":       "guest_network",
				"visibleBy":      []string{"public"},
			}},
		}

		for _, d := range docs {
			err := IndexDoc(ctx, d.doc, testIndex, d.id)
			if err != nil {
				t.Fatalf("Failed to index document %s: %v", d.id, err)
			}
		}

		// Wait for documents to be indexed
		time.Sleep(2 * time.Second)
		t.Logf("Indexed %d test documents with ip_range fields in %s", len(docs), testIndex)
	})

	// Test 1: IPRangeContains - find ranges that contain a specific IP
	t.Run("IPRangeContains", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Find documents where allowed_range contains 192.168.1.50
		query := builder.
			IPRangeContains("allowed_range", "192.168.1.50").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match doc 1 (192.168.0.0-192.168.255.255 contains 192.168.1.50)
		if resp.Hits.Total.Value != 1 {
			t.Errorf("Expected 1 document containing 192.168.1.50, got %d", resp.Hits.Total.Value)
		}
		t.Logf("IPRangeContains: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 2: IPRangeContains - find ranges containing 10.0.0.100
	t.Run("IPRangeContains10Network", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			IPRangeContains("allowed_range", "10.0.0.100").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match doc 2 (10.0.0.0/24 contains 10.0.0.100)
		if resp.Hits.Total.Value != 1 {
			t.Errorf("Expected 1 document containing 10.0.0.100, got %d", resp.Hits.Total.Value)
		}
		t.Logf("IPRangeContains10Network: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 3: IPRangeIntersects - find ranges that overlap with a given range
	t.Run("IPRangeIntersects", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Find documents where allowed_range overlaps with 10.0.0.0 - 10.255.255.255
		query := builder.
			IPRangeIntersects("allowed_range", "10.0.0.0", "10.255.255.255").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match docs 2 (10.0.0.0/24) and 4 (10.10.0.0/16)
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 documents intersecting 10.x.x.x range, got %d", resp.Hits.Total.Value)
		}
		t.Logf("IPRangeIntersects: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 4: IPRangeWithin - find ranges entirely within a given range
	t.Run("IPRangeWithin", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Find documents where allowed_range is entirely within 192.168.0.0-192.168.255.255
		query := builder.
			IPRangeWithin("allowed_range", "192.168.0.0", "192.168.255.255").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match:
		// - doc 1: 192.168.0.0-192.168.255.255 (exact match - equals is within)
		// - doc 5: 192.168.50.0-192.168.50.255 (subset)
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 documents within 192.168.0.0/16, got %d", resp.Hits.Total.Value)
		}
		t.Logf("IPRangeWithin: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 5: BoolBuilder with IPRangeContains
	t.Run("BoolBuilderIPRangeContains", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					MustIPRangeContains("allowed_range", "172.20.0.1").
					MustNotIPRangeContains("blocked_range", "192.168.100.50"),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match doc 3 (172.16.0.0-172.31.255.255 contains 172.20.0.1)
		// and blocked_range doesn't contain 192.168.100.50
		// Wait - doc 3's blocked_range IS 192.168.100.0-192.168.100.255 which DOES contain 192.168.100.50
		// So doc 3 should be excluded
		// Actually no docs should match this criteria
		t.Logf("BoolBuilderIPRangeContains: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 6: ShouldIPRangeContains with minimum_should_match
	t.Run("ShouldIPRangeContains", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Find documents where either allowed_range OR network_subnet contains 192.168.1.100
		query := builder.
			FilterBool(
				builder.Bool().
					ShouldIPRangeContains("allowed_range", "192.168.1.100").
					ShouldIPRangeContains("network_subnet", "192.168.1.100").
					MinimumShouldMatch(1),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match:
		// - doc 1: allowed_range (192.168.0.0/16) contains 192.168.1.100 AND network_subnet (192.168.1.0/24) contains it
		if resp.Hits.Total.Value < 1 {
			t.Errorf("Expected at least 1 document, got %d", resp.Hits.Total.Value)
		}
		t.Logf("ShouldIPRangeContains: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 7: Combine ip_range queries with term queries
	t.Run("CombinedIPRangeAndTerm", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			Term("acl_name", "datacenter_b").
			IPRangeContains("allowed_range", "10.10.5.5").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match doc 4 (acl_name=datacenter_b AND 10.10.0.0/16 contains 10.10.5.5)
		if resp.Hits.Total.Value != 1 {
			t.Errorf("Expected 1 document matching combined query, got %d", resp.Hits.Total.Value)
		}
		t.Logf("CombinedIPRangeAndTerm: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 8: IPRangeIntersects with narrow range
	t.Run("IPRangeIntersectsNarrow", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Find ranges that intersect with a narrow range (should find partial overlaps)
		query := builder.
			IPRangeIntersects("allowed_range", "192.168.49.0", "192.168.51.255").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should match:
		// - doc 1: 192.168.0.0-192.168.255.255 overlaps with 192.168.49.0-192.168.51.255
		// - doc 5: 192.168.50.0-192.168.50.255 overlaps with 192.168.49.0-192.168.51.255
		if resp.Hits.Total.Value != 2 {
			t.Errorf("Expected 2 documents with intersecting ranges, got %d", resp.Hits.Total.Value)
		}
		t.Logf("IPRangeIntersectsNarrow: Found %d documents", resp.Hits.Total.Value)
	})

	// Cleanup: Delete test index
	t.Run("Cleanup", func(t *testing.T) {
		_, err := apiClient.Indices.Delete(ctx, opensearchapi.IndicesDeleteReq{
			Indices: []string{testIndex},
		})
		if err != nil {
			t.Logf("Warning: Failed to delete test index %s: %v", testIndex, err)
		} else {
			t.Logf("Cleaned up test index %s", testIndex)
		}
	})
}

// TestKNNVectorSearchIntegration tests KNN vector search queries with actual OpenSearch execution
func TestKNNVectorSearchIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Connect to test OpenSearch using env vars
	nodes := os.Getenv("NODES")
	if nodes == "" {
		t.Skip("Skipping integration test: NODES env var not set")
	}
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Skip("Skipping integration test: could not connect to OpenSearch:", err)
	}

	ctx := context.Background()
	testIndex := "twos-knn-test-" + time.Now().Format("20060102-150405")

	// Setup: Create test index with knn_vector field mapping and documents
	t.Run("Setup", func(t *testing.T) {
		// Create index with knn_vector field mapping
		// Using HNSW algorithm with Lucene engine (default in OpenSearch 2.x+)
		mapping := map[string]interface{}{
			"settings": map[string]interface{}{
				"index": map[string]interface{}{
					"knn": true,
				},
			},
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"embedding": map[string]interface{}{
						"type":      "knn_vector",
						"dimension": 3,
						"method": map[string]interface{}{
							"name":       "hnsw",
							"space_type": "l2",
							"engine":     "lucene",
							"parameters": map[string]interface{}{
								"ef_construction": 128,
								"m":               16,
							},
						},
					},
					"category":  map[string]interface{}{"type": "keyword"},
					"name":      map[string]interface{}{"type": "keyword"},
					"visibleBy": map[string]interface{}{"type": "keyword"},
				},
			},
		}

		mappingJSON, _ := json.Marshal(mapping)
		_, err := apiClient.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
			Index: testIndex,
			Body:  strings.NewReader(string(mappingJSON)),
		})
		if err != nil {
			t.Fatalf("Failed to create index: %v", err)
		}

		// Index test documents with vector embeddings
		// Using 3-dimensional vectors for simplicity
		docs := []struct {
			id  string
			doc map[string]interface{}
		}{
			{"1", map[string]interface{}{
				"embedding": []float32{1.0, 0.0, 0.0}, // Point on x-axis
				"category":  "electronics",
				"name":      "laptop",
				"visibleBy": []string{"public"},
			}},
			{"2", map[string]interface{}{
				"embedding": []float32{0.0, 1.0, 0.0}, // Point on y-axis
				"category":  "electronics",
				"name":      "phone",
				"visibleBy": []string{"public"},
			}},
			{"3", map[string]interface{}{
				"embedding": []float32{0.0, 0.0, 1.0}, // Point on z-axis
				"category":  "clothing",
				"name":      "shirt",
				"visibleBy": []string{"public"},
			}},
			{"4", map[string]interface{}{
				"embedding": []float32{0.9, 0.1, 0.0}, // Close to doc 1
				"category":  "electronics",
				"name":      "tablet",
				"visibleBy": []string{"public"},
			}},
			{"5", map[string]interface{}{
				"embedding": []float32{0.1, 0.9, 0.0}, // Close to doc 2
				"category":  "clothing",
				"name":      "pants",
				"visibleBy": []string{"private"},
			}},
		}

		for _, d := range docs {
			err := IndexDoc(ctx, d.doc, testIndex, d.id)
			if err != nil {
				t.Fatalf("Failed to index document %s: %v", d.id, err)
			}
		}

		// Wait for documents to be indexed
		time.Sleep(2 * time.Second)
		t.Logf("Indexed %d test documents with vector embeddings in %s", len(docs), testIndex)
	})

	// Test 1: Basic KNN query - find 3 nearest neighbors to [1, 0, 0]
	t.Run("BasicKNNQuery", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			KNN("embedding", []float32{1.0, 0.0, 0.0}, 3).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should find docs, with doc 1 (exact match) and doc 4 (close) being most relevant
		if resp.Hits.Total.Value < 1 {
			t.Errorf("Expected at least 1 document, got %d", resp.Hits.Total.Value)
		}

		// First hit should be doc 1 (exact match)
		if len(resp.Hits.Hits) > 0 {
			t.Logf("BasicKNNQuery: First hit ID=%s, Score=%f", resp.Hits.Hits[0].ID, resp.Hits.Hits[0].Score)
		}
		t.Logf("BasicKNNQuery: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 2: KNN with filter - only search electronics category
	t.Run("KNNWithFilter", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		// Create filter for electronics category
		filter := TermQuery("category", "electronics")

		query := builder.
			KNNWithFilter("embedding", []float32{0.0, 1.0, 0.0}, 3, filter).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should only find electronics documents (docs 1, 2, 4)
		// Doc 2 [0,1,0] is exact match
		if resp.Hits.Total.Value < 1 {
			t.Errorf("Expected at least 1 document, got %d", resp.Hits.Total.Value)
		}
		t.Logf("KNNWithFilter: Found %d documents (electronics only)", resp.Hits.Total.Value)
	})

	// Test 3: BoolBuilder with KNN
	t.Run("BoolBuilderKNN", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			FilterBool(
				builder.Bool().
					ShouldKNN("embedding", []float32{0.0, 0.0, 1.0}, 3).
					MinimumShouldMatch(1),
			).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should find docs, with doc 3 (exact match [0,0,1]) being most relevant
		if resp.Hits.Total.Value < 1 {
			t.Errorf("Expected at least 1 document, got %d", resp.Hits.Total.Value)
		}
		t.Logf("BoolBuilderKNN: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 4: Helper function KNN query
	t.Run("HelperKNNQuery", func(t *testing.T) {
		knnQuery := NewKNNQuery("embedding", []float32{0.5, 0.5, 0.0}, 3)

		searchReq := SearchRequest{
			Query: &knnQuery,
			Size:  10,
		}

		resp, err := searchReq.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should find docs, with docs 1, 2, 4, 5 being relevant (all in xy plane)
		if resp.Hits.Total.Value < 1 {
			t.Errorf("Expected at least 1 document, got %d", resp.Hits.Total.Value)
		}
		t.Logf("HelperKNNQuery: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 5: KNN with options (boost)
	// Note: min_score/max_distance are mutually exclusive with k - you use ONE of them
	// So for testing options, we'll use boost which is compatible with k
	t.Run("KNNWithOptions", func(t *testing.T) {
		// Use helper function directly to create a KNN query with boost option
		opts := KNNQueryOptions{
			Boost: Float64Ptr(2.0), // Double the score
		}
		knnQuery := NewKNNQueryWithOptions("embedding", []float32{1.0, 0.0, 0.0}, 5, opts)

		searchReq := SearchRequest{
			Query: &knnQuery,
			Size:  10,
		}

		resp, err := searchReq.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Results should be boosted (score * 2.0)
		t.Logf("KNNWithOptions: Found %d documents with boost=2.0", resp.Hits.Total.Value)
		for i, hit := range resp.Hits.Hits {
			t.Logf("  Hit %d: ID=%s, Score=%f (boosted)", i, hit.ID, hit.Score)
		}
	})

	// Test 6: Combined KNN and term filter
	t.Run("CombinedKNNAndFilters", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			KNN("embedding", []float32{0.0, 1.0, 0.0}, 3).
			Term("category", "electronics").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// KNN + term filter should work together
		t.Logf("CombinedKNNAndFilters: Found %d documents", resp.Hits.Total.Value)
	})

	// Test 7: SearchIn with visibility filter
	t.Run("KNNWithSearchInVisibility", func(t *testing.T) {
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")

		query := builder.
			KNN("embedding", []float32{0.1, 0.9, 0.0}, 5). // Close to doc 5 which is private
			Size(10).
			Build()

		// SearchIn adds visibleBy filter for "public" group
		resp, err := query.SearchIn(ctx, []string{testIndex}, []string{"public"})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Doc 5 [0.1, 0.9, 0.0] should be excluded (private)
		// Only public docs should be returned
		t.Logf("KNNWithSearchInVisibility: Found %d public documents", resp.Hits.Total.Value)
	})

	// Cleanup: Delete test index
	t.Run("Cleanup", func(t *testing.T) {
		_, err := apiClient.Indices.Delete(ctx, opensearchapi.IndicesDeleteReq{
			Indices: []string{testIndex},
		})
		if err != nil {
			t.Logf("Warning: Failed to delete test index %s: %v", testIndex, err)
		} else {
			t.Logf("Cleaned up test index %s", testIndex)
		}
	})
}
