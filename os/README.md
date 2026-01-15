# ThreatWinds OpenSearch Wrapper

A simplified Go client wrapper for OpenSearch integrated into the ThreatWinds SDK. It provides fluent query builders, automatic field mapping resolution, and group-based access control filtering.

## Features

- **Fluent Query Builder**: Type-safe query construction with automatic `.keyword` field resolution
- **Field Mapping Resolution**: Automatic detection of field types with LRU caching
- **Group-Based Access Control**: Built-in `visibleBy` filtering for multi-tenant applications
- **Bool Query Builder**: Nested boolean queries with field resolution at all levels
- **ML Inference**: Support for generating text embeddings using OpenSearch ML Commons
- **Bulk Operations**: High-performance bulk indexing queue with automatic flushing
- **Index Helpers**: Time-based index naming and pattern generation

## Installation

```bash
go get github.com/threatwinds/go-sdk/os
```

## Quick Start

### Connecting to OpenSearch

```go
package main

import (
    "context"
    "log"

    twos "github.com/threatwinds/go-sdk/os"
)

func main() {
    // Connect to OpenSearch (singleton - first call wins)
    err := twos.Connect(
        []string{"https://localhost:9200"},
        "admin",
        "admin",
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

### Basic Search with Query Builder

```go
ctx := context.Background()
indices := []string{"entities-*"}

// Build a query with automatic field resolution
builder := twos.NewQueryBuilder(ctx, indices, "my-process")
query := builder.
    Term("type", "ip").              // auto-resolves to type.keyword
    Range("reputation", "lte", -1).
    Match("description", "malware").
    Sort("@timestamp", "desc").
    Size(100).
    Build()

// SearchIn adds visibleBy filter automatically
results, err := query.SearchIn(ctx, indices, []string{"public", "org:acme"})
if err != nil {
    log.Fatal(err)
}

for _, hit := range results.Hits.Hits {
    log.Printf("ID: %s, Score: %v", hit.ID, hit.Score)
}
```

### Manual Query Construction

```go
req := twos.SearchRequest{
    From: 0,
    Size: 10,
    Query: &twos.Query{
        Bool: &twos.Bool{
            Filter: []twos.Query{
                {
                    Term: map[string]map[string]interface{}{
                        "type.keyword": {"value": "ip"},
                    },
                },
                {
                    Range: map[string]map[string]interface{}{
                        "reputation": {"lte": -1},
                    },
                },
            },
        },
    },
}

// SearchIn applies group-based access control
results, err := req.SearchIn(ctx, []string{"entities-*"}, []string{"public"})
```

## API Reference

### Connection

#### Connect

```go
func Connect(nodes []string, user, password string) error
```

Establishes a singleton connection to OpenSearch. Only the first call takes effect; subsequent calls return the existing connection.

### Search Methods

#### SearchIn

```go
func (q SearchRequest) SearchIn(ctx context.Context, index []string, groups []string) (SearchResult, error)
```

Executes a search with automatic `visibleBy.keyword` filtering based on provided groups. Use this for user-facing queries that require access control.

#### WideSearchIn

```go
func (q SearchRequest) WideSearchIn(ctx context.Context, index []string) (SearchResult, error)
```

Executes a search without access control filtering. Use this for admin or system operations.

### QueryBuilder

The `QueryBuilder` provides a fluent API for constructing OpenSearch queries with automatic field type resolution.

#### Creating a QueryBuilder

```go
// With default mapper (shared across builders)
builder := twos.NewQueryBuilder(ctx, []string{"index-*"}, "my-process")

// With custom mapper
mapper := twos.NewFieldMapper(
    twos.WithCacheTTL(10 * time.Minute),
    twos.WithMaxCacheSize(100),
    twos.WithConflictStrategy(twos.MostCommon),
)
builder := twos.NewQueryBuilderWithMapper(ctx, indices, mapper, "my-process")
```

#### Pagination Methods

```go
builder.Size(100)           // Number of results to return (default: 10)
builder.From(20)            // Offset for pagination
builder.SearchAfter([]int64{...})  // Cursor-based pagination
```

#### Sorting

```go
builder.Sort("@timestamp", "desc")
builder.Sort("score", "asc")
```

#### Source Filtering

```go
builder.IncludeSource("id", "name", "type")
builder.ExcludeSource("visibleBy", "internalField")
```

#### Query Clauses

```go
// Term query (exact match) - auto-resolves to .keyword for text fields
builder.Term("type", "ip")

// Terms query (match any)
builder.Terms("status", []interface{}{"active", "pending"})

// Match query (full-text search)
builder.Match("description", "malware threat")

// Match phrase query
builder.MatchPhrase("title", "security incident")

// Range query
builder.Range("reputation", "lte", -1)
builder.Range("createdAt", "gte", "2024-01-01")

// Exists query
builder.Exists("email")

// Wildcard query
builder.Wildcard("hostname", "server-*")

// Prefix query
builder.Prefix("name", "test")
```

#### Boolean Logic

```go
// Must (AND)
builder.Must(query1, query2)

// Should (OR)
builder.Should(query1, query2)

// Filter (no scoring)
builder.Filter(query1, query2)

// Must Not (NOT)
builder.MustNot(query1)

// Minimum should match
builder.MinimumShouldMatch(1)
```

#### Aggregations

```go
builder.TermsAgg("types", "type", 100)      // Terms aggregation
builder.SumAgg("total_score", "score")       // Sum aggregation
builder.AvgAgg("avg_score", "score")         // Average aggregation
builder.MinAgg("min_score", "score")         // Min aggregation
builder.MaxAgg("max_score", "score")         // Max aggregation
builder.CardinalityAgg("unique_users", "userId")  // Unique count
builder.DateHistogramAgg("over_time", "@timestamp", "1d")  // Date histogram
```

#### Collapse (Deduplication)

```go
builder.Collapse("userId")  // Deduplicate by field
```

#### Building the Query

```go
// Build and get SearchRequest
query := builder.Build()

// Build with error checking
query, errors := builder.BuildWithErrors()
if len(errors) > 0 {
    log.Printf("Query builder errors: %v", errors)
}

// Check for mapping conflicts
conflicts := builder.GetMappingConflicts()
```

### BoolBuilder

The `BoolBuilder` provides fine-grained control over nested boolean queries with field resolution.

#### Creating a BoolBuilder

```go
// From QueryBuilder (inherits context and mapper)
boolBuilder := builder.Bool()

// Standalone
boolBuilder := twos.NewBoolBuilder(ctx, indices, "my-process")
```

#### Term Methods

```go
boolBuilder.MustTerm("type", "ip")
boolBuilder.ShouldTerm("type", "domain")
boolBuilder.FilterTerm("status", "active")
boolBuilder.MustNotTerm("blocked", true)
```

#### Terms Methods

```go
boolBuilder.MustTerms("type", "ip", "domain", "url")
boolBuilder.ShouldTerms("status", "active", "pending")
boolBuilder.FilterTerms("category", "threat", "malware")
boolBuilder.MustNotTerms("source", "internal", "test")
```

#### Match Methods

```go
boolBuilder.MustMatch("description", "malware")
boolBuilder.ShouldMatch("title", "security")
boolBuilder.FilterMatch("content", "threat")
boolBuilder.MustNotMatch("notes", "false positive")
```

#### Range Methods

```go
boolBuilder.MustRange("reputation", "lte", -1)
boolBuilder.ShouldRange("score", "gte", 80)
boolBuilder.FilterRange("createdAt", "gte", "2024-01-01")
boolBuilder.MustNotRange("age", "gt", 365)
```

#### Exists Methods

```go
boolBuilder.MustExists("email")
boolBuilder.ShouldExists("phone")
boolBuilder.FilterExists("address")
boolBuilder.MustNotExists("deletedAt")
```

#### Wildcard Methods

```go
boolBuilder.MustWildcard("hostname", "prod-*")
boolBuilder.ShouldWildcard("name", "*-server")
boolBuilder.FilterWildcard("path", "/api/*")
boolBuilder.MustNotWildcard("file", "*.tmp")
```

#### Nested Bool Queries

```go
// Simple OR condition
orCondition := builder.Bool().
    ShouldTerm("type", "ip").
    ShouldTerm("type", "domain").
    MinimumShouldMatch(1)

query := builder.
    Term("status", "active").
    FilterBool(orCondition).
    Build()

// Complex nested query
innerOr := builder.Bool().
    ShouldTerm("subtype", "ip").
    ShouldTerm("subtype", "domain").
    MinimumShouldMatch(1)

threatCondition := builder.Bool().
    MustTerm("type", "threat").
    MustRange("severity", "gte", 5)

indicatorCondition := builder.Bool().
    MustTerm("type", "indicator").
    MustBool(innerOr)

query := builder.
    FilterBool(
        builder.Bool().
            ShouldBool(threatCondition).
            ShouldBool(indicatorCondition).
            MinimumShouldMatch(1),
    ).
    Build()
```

#### Adding Raw Queries

```go
boolBuilder.MustQuery(query1, query2)
boolBuilder.ShouldQuery(query1, query2)
boolBuilder.FilterQuery(query1, query2)
boolBuilder.MustNotQuery(query1, query2)
```

### Query Helper Functions

Standalone query constructors for use without field resolution:

```go
// Term queries
twos.TermQuery("type.keyword", "ip")
twos.TermsQuery("status.keyword", []interface{}{"active", "pending"})

// Match queries
twos.MatchQuery("description", "malware")
twos.MatchPhraseQuery("title", "security incident")
twos.MatchPhrasePrefixQuery("name", "John")
twos.MultiMatchQuery("search text", []string{"title", "description"})

// Range queries
twos.RangeQuery("score", "gte", 80)
twos.RangeQueryBetween("age", 18, 65)
twos.RangeGte("score", 80)
twos.RangeLte("score", 100)
twos.RangeGt("age", 17)
twos.RangeLt("age", 66)

// Other queries
twos.ExistsQuery("email")
twos.WildcardQuery("hostname", "server-*")
twos.PrefixQuery("name", "test")
twos.FuzzyQuery("name", "john", "AUTO")
twos.RegexpQuery("email", ".*@example\\.com")
twos.IDsQuery([]interface{}{"id1", "id2", "id3"})
twos.QueryStringQuery("status:active AND type:ip")
twos.SimpleQueryStringQuery("malware threat", "title", "description")
```

#### Boolean Combinators

```go
// OR - combines queries with should and minimum_should_match=1
twos.Or(query1, query2, query3)

// AND - combines queries with must
twos.And(query1, query2, query3)

// NOT - creates must_not query
twos.Not(query1, query2)
```

### Field Mapping

The `FieldMapper` caches index mappings and handles field type resolution.

#### Creating a FieldMapper

```go
mapper := twos.NewFieldMapper(
    twos.WithCacheTTL(10 * time.Minute),      // Cache TTL (default: 5 min)
    twos.WithMaxCacheSize(100),                // Max cached patterns (default: 50)
    twos.WithConflictStrategy(twos.MostCommon), // Conflict resolution
    twos.WithStrictMode(false),                // Error on unknown fields
)
```

#### Conflict Strategies

- `MostCommon`: Uses the most common type across indices (default)
- `MostPermissive`: Uses the most permissive type (text > keyword)
- `Strict`: Returns error on any type conflict
- `MostRecent`: Uses type from most recent index

#### Cache Management

```go
mapper.Invalidate("entities-*")  // Remove pattern from cache
mapper.Clear()                   // Clear all cached mappings
```

### Index Helpers

#### Index Prefixes

```go
twos.EntityPrefix       // "entity"
twos.RelationPrefix     // "relation"
twos.CommentPrefix      // "comment"
twos.ConsolidatedPrefix // "consolidated"
twos.HistoryPrefix      // "history"
```

#### Building Index Names

```go
// Build pattern for searching
twos.BuildIndexPattern(twos.EntityPrefix)
// → "entity-*"

twos.BuildIndexPattern(twos.EntityPrefix, twos.ConsolidatedPrefix)
// → "entity-consolidated-*"

// Build current month's index
twos.BuildCurrentIndex(twos.CommentPrefix)
// → "comment-2024-01" (current month)

// Build index for specific date
date := time.Date(2023, 10, 21, 0, 0, 0, 0, time.UTC)
twos.BuildIndex(date, twos.RelationPrefix, twos.HistoryPrefix)
// → "relation-history-2023-10"
```

### ML Inference

Support for OpenSearch ML Commons text embedding models.

#### Generate Embeddings

```go
// Single text
embedding, err := twos.MLPredictSingle(ctx, "model_id", "text to embed")

// Batch of texts
embeddings, err := twos.MLPredict(ctx, "model_id", []string{"text1", "text2"})

// Large batch with automatic chunking
embeddings, err := twos.MLPredictBatch(ctx, "model_id", manyTexts, 10)
```

### Bulk Operations

The `BulkQueue` provides a thread-safe way to accumulate and process documents in batches.

#### Using BulkQueue

```go
// Create a new queue
config := twos.BulkQueueConfig{
    FlushThreshold: 100,
    FlushInterval:  5 * time.Second,
}
queue := twos.NewBulkQueue(config)

// Add documents
queue.Add("index-name", document)
queue.AddWithID("index-name", "doc-id", document)

// Force flush
err := queue.Flush()

// Stop the queue (gracefully flushes remaining items)
queue.Stop()
```

### Document Operations

#### Indexing Documents

```go
doc := map[string]interface{}{
    "type":      "ip",
    "value":     "192.168.1.1",
    "visibleBy": []string{"public"},
}

index := twos.BuildCurrentIndex(twos.EntityPrefix)
err := twos.IndexDoc(ctx, doc, index, "document-id")
```

#### Updating Documents

```go
// After searching, update and save a hit
for _, hit := range results.Hits.Hits {
    hit.Source["status"] = "processed"
    err := hit.Save(ctx)
    if err != nil {
        log.Printf("Failed to update: %v", err)
    }
}
```

#### Deleting Documents

```go
for _, hit := range results.Hits.Hits {
    err := hit.Delete(ctx)
    if err != nil {
        log.Printf("Failed to delete: %v", err)
    }
}
```

### Working with Hit Sources

#### Parsing Source to Struct

```go
type Entity struct {
    Type  string `json:"type"`
    Value string `json:"value"`
}

for _, hit := range results.Hits.Hits {
    var entity Entity
    err := hit.Source.ParseSource(&entity)
    if err != nil {
        log.Printf("Parse error: %v", err)
        continue
    }
    log.Printf("Entity: %+v", entity)
}
```

#### Setting Source from Struct

```go
entity := Entity{Type: "ip", Value: "10.0.0.1"}
err := hit.Source.SetSource(entity)
if err != nil {
    log.Printf("Set source error: %v", err)
}
```

## Common Patterns

### SearchIn vs WideSearchIn

```go
// SearchIn: Adds visibleBy.keyword filter automatically
// Use for user-facing queries with access control
query.SearchIn(ctx, []string{"entities"}, []string{"public", "org:acme"})

// WideSearchIn: No access control filtering
// Use for admin/system operations
query.WideSearchIn(ctx, []string{"entities"})
```

### Keyword Fields for Exact Match

```go
// WRONG - text field won't match exactly
{Term: {"type": {"value": "ip"}}}

// CORRECT - use .keyword for term queries
{Term: {"type.keyword": {"value": "ip"}}}

// OR use QueryBuilder (auto-resolves)
builder.Term("type", "ip")  // → type.keyword
```

## Testing

```bash
# Run all tests (requires OpenSearch instance)
NODES=https://localhost:9200 USER=admin PASSWORD=admin go test -v ./...

# Run specific test
go test -v -run TestSearchIn ./...
```

## Dependencies

- [opensearch-go/v4](https://github.com/opensearch-project/opensearch-go) - OpenSearch client
- [golang-lru/v2](https://github.com/hashicorp/golang-lru) - LRU cache for mappings
- [threatwinds/go-sdk](https://github.com/threatwinds/go-sdk) - Error handling, entities

## License

MIT License
