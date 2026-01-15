package os

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/threatwinds/go-sdk/catcher"
)

// QueryBuilder provides a fluent API for building OpenSearch queries
type QueryBuilder struct {
	ctx     context.Context
	indices []string
	mapper  *FieldMapper
	request SearchRequest
	errors  []error
	// knnQuery stores k-NN query separately for proper OpenSearch 3.x handling.
	// In OpenSearch 3.x, k-NN queries must be at the top level, not nested in bool.must.
	knnQuery    *knnQueryConfig
	processName string
}

// knnQueryConfig stores k-NN query configuration for deferred building.
type knnQueryConfig struct {
	field string
	query *KNNQuery
}

var defaultMapper *FieldMapper

func init() {
	defaultMapper = NewFieldMapper()
}

// NewQueryBuilder creates a new QueryBuilder with default mapper
func NewQueryBuilder(ctx context.Context, indices []string, processName string) *QueryBuilder {
	return NewQueryBuilderWithMapper(ctx, indices, defaultMapper, processName)
}

// NewQueryBuilderWithMapper creates a new QueryBuilder with custom mapper
func NewQueryBuilderWithMapper(ctx context.Context, indices []string, mapper *FieldMapper, processName string) *QueryBuilder {
	return &QueryBuilder{
		ctx:     ctx,
		indices: indices,
		mapper:  mapper,
		request: SearchRequest{
			Size: 10, // default size
			From: 0,  // default offset
			Query: &Query{
				Bool: &Bool{
					Must:    []Query{},
					Filter:  []Query{},
					Should:  []Query{},
					MustNot: []Query{},
				},
			},
		},
		errors:      []error{},
		processName: processName,
	}
}

// Size sets the number of results to return
func (b *QueryBuilder) Size(size int64) *QueryBuilder {
	b.request.Size = size
	return b
}

// From sets the offset for pagination
func (b *QueryBuilder) From(from int64) *QueryBuilder {
	b.request.From = from
	return b
}

// SearchAfter sets cursor-based pagination
func (b *QueryBuilder) SearchAfter(values []int64) *QueryBuilder {
	b.request.SearchAfter = values
	return b
}

// Sort adds sorting to the query
func (b *QueryBuilder) Sort(field string, order string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeSort)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("sort field '%s': %w", field, err))
		return b
	}

	if b.request.Sort == nil {
		b.request.Sort = []map[string]map[string]interface{}{}
	}

	sortMap := map[string]map[string]interface{}{
		resolvedField: {
			"order": order,
		},
	}

	b.request.Sort = append(b.request.Sort, sortMap)
	return b
}

// IncludeSource specifies which fields to include in results
func (b *QueryBuilder) IncludeSource(fields ...string) *QueryBuilder {
	if b.request.Source == nil {
		b.request.Source = &Source{}
	}
	b.request.Source.Includes = append(b.request.Source.Includes, fields...)
	return b
}

// ExcludeSource specifies which fields to exclude from results
func (b *QueryBuilder) ExcludeSource(fields ...string) *QueryBuilder {
	if b.request.Source == nil {
		b.request.Source = &Source{}
	}
	b.request.Source.Excludes = append(b.request.Source.Excludes, fields...)
	return b
}

// Collapse deduplicates results by field
func (b *QueryBuilder) Collapse(field string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, catcher.Error("failed to resolve collapse field", err, map[string]any{
			"process": b.processName,
			"field":   field,
		}))
		return b
	}

	b.request.Collapse = &Collapse{
		Field: resolvedField,
	}
	return b
}

// Must adds queries that must match (AND logic)
func (b *QueryBuilder) Must(queries ...Query) *QueryBuilder {
	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, queries...)
	return b
}

// Should adds queries that should match (OR logic)
func (b *QueryBuilder) Should(queries ...Query) *QueryBuilder {
	b.request.Query.Bool.Should = append(b.request.Query.Bool.Should, queries...)
	return b
}

// Filter adds filter queries (no scoring, AND logic)
func (b *QueryBuilder) Filter(queries ...Query) *QueryBuilder {
	b.request.Query.Bool.Filter = append(b.request.Query.Bool.Filter, queries...)
	return b
}

// MustNot adds queries that must not match (NOT logic)
func (b *QueryBuilder) MustNot(queries ...Query) *QueryBuilder {
	b.request.Query.Bool.MustNot = append(b.request.Query.Bool.MustNot, queries...)
	return b
}

// MinimumShouldMatch sets the minimum number of should clauses that must match
func (b *QueryBuilder) MinimumShouldMatch(value interface{}) *QueryBuilder {
	b.request.Query.Bool.MinimumShouldMatch = value
	return b
}

// --- Request-Level Methods ---

// Version enables version tracking in results
func (b *QueryBuilder) Version(enabled bool) *QueryBuilder {
	b.request.Version = enabled
	return b
}

// StoredFields specifies which stored fields to return
func (b *QueryBuilder) StoredFields(fields ...string) *QueryBuilder {
	b.request.StoredFields = append(b.request.StoredFields, fields...)
	return b
}

// ScriptFields adds script fields to compute values
func (b *QueryBuilder) ScriptFields(scripts interface{}) *QueryBuilder {
	b.request.ScriptFields = scripts
	return b
}

// SetSource sets the full Source object (for advanced control)
func (b *QueryBuilder) SetSource(source *Source) *QueryBuilder {
	b.request.Source = source
	return b
}

// Term adds a term query (exact match)
func (b *QueryBuilder) Term(field string, value interface{}) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeTerm)
	if err != nil {
		// Check if this is a text field with no .keyword - try to convert to match
		if info, ok := b.getFieldInfo(field); ok && info.Type == "text" && len(info.Fields) == 0 {
			catcher.Info("attempting to convert Term query to Match query because the field is of text type without .keyword sub-field", map[string]any{
				"status":  http.StatusBadRequest,
				"field":   field,
				"process": b.processName,
			})
			return b.Match(field, fmt.Sprint(value))
		}
		b.errors = append(b.errors, fmt.Errorf("term field '%s': %w", field, err))
		return b
	}

	query := Query{
		Term: map[string]map[string]interface{}{
			resolvedField: {"value": value},
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// Terms add a terms query (match any of the values)
func (b *QueryBuilder) Terms(field string, values []interface{}) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeTerms)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("terms field '%s': %w", field, err))
		return b
	}

	query := Query{
		Terms: map[string][]interface{}{
			resolvedField: values,
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// CIDR adds a CIDR query for "ip" type fields (e.g., "192.168.0.0/16")
// This is for fields mapped as "ip" type (storing single IP addresses).
// OpenSearch natively supports CIDR notation in term queries on ip fields.
// Note: For "ip_range" type fields, use IPRangeContains, IPRangeIntersects, or IPRangeWithin instead.
func (b *QueryBuilder) CIDR(field string, cidr string) *QueryBuilder {
	// IP fields don't need .keyword resolution - use field as-is
	query := Query{
		Term: map[string]map[string]interface{}{
			field: {"value": cidr},
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// CIDRs add a terms query for "ip" type fields using multiple CIDR notations.
// This is for fields mapped as "ip" type (storing single IP addresses).
// Example: CIDRs("client_ip", "192.168.0.0/24", "10.0.0.0/8")
// Note: For "ip_range" type fields, use IPRangeContains, IPRangeIntersects, or IPRangeWithin instead.
func (b *QueryBuilder) CIDRs(field string, cidrs ...string) *QueryBuilder {
	values := make([]interface{}, len(cidrs))
	for i, cidr := range cidrs {
		values[i] = cidr
	}

	query := Query{
		Terms: map[string][]interface{}{
			field: values,
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// === IP Range Field Query Methods ===
// These are for fields mapped as "ip_range" type (storing IP address ranges).
// ip_range fields store ranges like {"gte": "10.0.0.0", "lte": "10.255.255.255"} or CIDR notation.

// IPRangeContains finds documents where the stored ip_range contains the given IP address.
// Example: IPRangeContains("allowed_ips", "192.168.1.50") matches ranges like 192.168.0.0/16
func (b *QueryBuilder) IPRangeContains(field string, ip string) *QueryBuilder {
	query := Query{
		Range: map[string]map[string]interface{}{
			field: {
				"gte":      ip,
				"lte":      ip,
				"relation": "contains",
			},
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// IPRangeIntersects finds documents where the stored ip_range overlaps with the given range.
// Example: IPRangeIntersects("blocked_ranges", "192.168.0.0", "192.168.255.255")
func (b *QueryBuilder) IPRangeIntersects(field string, fromIP, toIP string) *QueryBuilder {
	query := Query{
		Range: map[string]map[string]interface{}{
			field: {
				"gte":      fromIP,
				"lte":      toIP,
				"relation": "intersects",
			},
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// IPRangeWithin finds documents where the stored ip_range is entirely within the given range.
// Example: IPRangeWithin("subnet", "10.0.0.0", "10.255.255.255") matches 10.0.0.0/24, 10.1.0.0/16, etc.
func (b *QueryBuilder) IPRangeWithin(field string, fromIP, toIP string) *QueryBuilder {
	query := Query{
		Range: map[string]map[string]interface{}{
			field: {
				"gte":      fromIP,
				"lte":      toIP,
				"relation": "within",
			},
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// Match adds a full-text match query
func (b *QueryBuilder) Match(field string, value string) *QueryBuilder {
	// For match queries, we want the base field (not .keyword)
	info, ok := b.getFieldInfo(field)
	if ok {
		// If it's a keyword field, auto-convert to Term query
		if info.Type == "keyword" || (info.Type != "text" && info.AllowsTerm) {
			catcher.Info("converting Match to Term query because field is a keyword", map[string]any{
				"field":   field,
				"status":  http.StatusBadRequest,
				"process": b.processName,
			})
			return b.Term(field, value)
		}
	}

	query := Query{
		Match: map[string]Match{
			field: {Query: value},
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// MatchPhrase adds a match phrase query
func (b *QueryBuilder) MatchPhrase(field string, value string) *QueryBuilder {
	info, ok := b.getFieldInfo(field)
	if ok && info.Type != "text" {
		b.errors = append(b.errors, fmt.Errorf("match_phrase field '%s' is %s type, requires text field", field, info.Type))
		return b
	}

	query := Query{
		MatchPhrase: map[string]MatchPhrase{
			field: {Query: value},
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// Range adds a range query
func (b *QueryBuilder) Range(field string, operator string, value interface{}) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeRange)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("range field '%s': %w", field, err))
		return b
	}

	query := Query{
		Range: map[string]map[string]interface{}{
			resolvedField: {operator: value},
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// Exists adds an exists query (field must be present)
func (b *QueryBuilder) Exists(field string) *QueryBuilder {
	query := Query{
		Exists: map[string]string{"field": field},
	}

	b.request.Query.Bool.Filter = append(b.request.Query.Bool.Filter, query)
	return b
}

// Wildcard adds a wildcard query
func (b *QueryBuilder) Wildcard(field string, pattern string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeWildcard)
	if err != nil {
		// Wildcard works on both text and keyword, so try to resolve but don't fail
		resolvedField = field
	}

	query := Query{
		Wildcard: map[string]map[string]interface{}{
			resolvedField: {"value": pattern},
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// Prefix adds a prefix query (with field resolution)
func (b *QueryBuilder) Prefix(field string, prefix string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypePrefix)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("prefix field '%s': %w", field, err))
		return b
	}

	query := Query{
		Prefix: map[string]string{resolvedField: prefix},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// --- Additional Query Types ---

// IDs adds an IDs query to filter documents by ID (no field resolution needed)
func (b *QueryBuilder) IDs(ids ...interface{}) *QueryBuilder {
	query := Query{
		IDs: map[string][]interface{}{"values": ids},
	}

	b.request.Query.Bool.Filter = append(b.request.Query.Bool.Filter, query)
	return b
}

// QueryString adds a query_string query with Lucene syntax support
func (b *QueryBuilder) QueryString(query string, defaultOperator ...string) *QueryBuilder {
	qs := &QueryString{Query: query}
	if len(defaultOperator) > 0 {
		qs.DefaultOperator = defaultOperator[0]
	}

	q := Query{QueryString: qs}
	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, q)
	return b
}

// SimpleQueryString adds a simple_query_string query
func (b *QueryBuilder) SimpleQueryString(query string, fields ...string) *QueryBuilder {
	sqs := &SimpleQueryString{Query: query}
	if len(fields) > 0 {
		sqs.Fields = fields
	}

	q := Query{SimpleQueryString: sqs}
	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, q)
	return b
}

// MultiMatch adds a multi_match query across multiple fields
func (b *QueryBuilder) MultiMatch(query string, fields []string, matchType ...string) *QueryBuilder {
	mm := &MultiMatch{Query: query, Fields: fields}
	if len(matchType) > 0 {
		mm.Type = matchType[0]
	}

	q := Query{MultiMatch: mm}
	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, q)
	return b
}

// Fuzzy adds a fuzzy query for typo-tolerant matching (with field resolution)
func (b *QueryBuilder) Fuzzy(field string, value string, fuzziness ...string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeFuzzy)
	if err != nil {
		// Fuzzy works like Match - needs text fields
		// Check if this is a keyword field and convert to Term
		if info, ok := b.getFieldInfo(field); ok && info.Type == "keyword" {
			catcher.Info("fuzzy queries work better on text fields", map[string]any{
				"status":  http.StatusBadRequest,
				"field":   field,
				"process": b.processName,
			})
		}
		b.errors = append(b.errors, fmt.Errorf("fuzzy field '%s': %w", field, err))
		return b
	}

	fuzzyValue := map[string]interface{}{"value": value}
	if len(fuzziness) > 0 {
		fuzzyValue["fuzziness"] = fuzziness[0]
	}

	query := Query{
		Fuzzy: map[string]map[string]interface{}{
			resolvedField: fuzzyValue,
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// Regexp adds a regexp query (with field resolution - prefers keyword fields)
func (b *QueryBuilder) Regexp(field string, pattern string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeRegexp)
	if err != nil {
		// Regexp works on keyword/text, use as-is if resolution fails
		resolvedField = field
	}

	query := Query{
		Regexp: map[string]string{resolvedField: pattern},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// MatchPhrasePrefix adds a match_phrase_prefix query (with field resolution)
func (b *QueryBuilder) MatchPhrasePrefix(field string, value string) *QueryBuilder {
	// match_phrase_prefix works like match - needs text fields
	info, ok := b.getFieldInfo(field)
	if ok && info.Type != "text" {
		b.errors = append(b.errors, fmt.Errorf("match_phrase_prefix field '%s' is %s type, requires text field", field, info.Type))
		return b
	}

	query := Query{
		MatchPhrasePrefix: map[string]MatchPhrasePrefix{
			field: {Query: value},
		},
	}

	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, query)
	return b
}

// MatchAll adds a match_all query (matches everything)
func (b *QueryBuilder) MatchAll() *QueryBuilder {
	// Simply don't add any conditions - an empty bool query matches all
	return b
}

// === KNN (k-Nearest Neighbor) Vector Search Methods ===
// These methods are for fields mapped as "knn_vector" type (storing dense vectors for similarity search).

// KNN adds a basic k-NN query to find the k nearest neighbors.
// field: the knn_vector field name
// vector: the query vector (must match the field's dimension)
// k: number of nearest neighbors to return
// Example: builder.KNN("embedding", []float32{0.1, 0.2, 0.3}, 10)
//
// Note: In OpenSearch 3.x, k-NN queries must be at the top level of the query.
// This builder handles the proper placement automatically.
func (b *QueryBuilder) KNN(field string, vector []float32, k int) *QueryBuilder {
	b.knnQuery = &knnQueryConfig{
		field: field,
		query: &KNNQuery{
			Vector: vector,
			K:      k,
		},
	}
	return b
}

// KNNWithFilter adds a k-NN query with a pre-filter.
// The filter is applied before the k-NN search, so only matching documents are considered.
// field: the knn_vector field name
// vector: the query vector
// k: number of nearest neighbors to return
// filter: query to filter documents before k-NN search
// Example:
//
//	builder.KNNWithFilter("embedding", []float32{0.1, 0.2, 0.3}, 10,
//	    TermQuery("category.keyword", "tech"))
//
// Note: In OpenSearch 3.x, k-NN queries must be at the top level of the query.
// This builder handles the proper placement automatically.
func (b *QueryBuilder) KNNWithFilter(field string, vector []float32, k int, filter Query) *QueryBuilder {
	b.knnQuery = &knnQueryConfig{
		field: field,
		query: &KNNQuery{
			Vector: vector,
			K:      k,
			Filter: &filter,
		},
	}
	return b
}

// KNNWithMinScore adds a k-NN query with a minimum score threshold.
// Only results with similarity score >= minScore are returned.
// field: the knn_vector field name
// vector: the query vector
// k: maximum number of results to return (used as Size limit)
// minScore: minimum similarity score threshold (0.0 to 1.0 for cosine similarity)
// Example: builder.KNNWithMinScore("embedding", []float32{0.1, 0.2, 0.3}, 10, 0.8)
//
// Note: In OpenSearch 3.x, k-NN queries must be at the top level of the query
// and require exactly ONE of k, distance, or score. When min_score is set,
// this function sets the Size field to limit results instead of using K.
func (b *QueryBuilder) KNNWithMinScore(field string, vector []float32, k int, minScore float64) *QueryBuilder {
	// In OpenSearch 3.x, only one of k, min_score, or max_distance can be set.
	// When using min_score, we set Size to limit the number of results.
	b.request.Size = int64(k)
	b.knnQuery = &knnQueryConfig{
		field: field,
		query: &KNNQuery{
			Vector:   vector,
			MinScore: &minScore,
		},
	}
	return b
}

// KNNWithMaxDistance adds a k-NN query with a maximum distance threshold.
// Only results within maxDistance from the query vector are returned.
// The distance metric depends on the space_type configured in the index mapping.
// field: the knn_vector field name
// vector: the query vector
// k: maximum number of results to return (used as Size limit)
// maxDistance: maximum distance threshold (e.g., for L2 distance)
// Example: builder.KNNWithMaxDistance("embedding", []float32{0.1, 0.2, 0.3}, 10, 100.0)
//
// Note: In OpenSearch 3.x, k-NN queries must be at the top level of the query
// and require exactly ONE of k, distance, or score. When max_distance is set,
// this function sets the Size field to limit results instead of using K.
func (b *QueryBuilder) KNNWithMaxDistance(field string, vector []float32, k int, maxDistance float64) *QueryBuilder {
	// In OpenSearch 3.x, only one of k, min_score, or max_distance can be set.
	// When using max_distance, we set Size to limit the number of results.
	b.request.Size = int64(k)
	b.knnQuery = &knnQueryConfig{
		field: field,
		query: &KNNQuery{
			Vector:      vector,
			MaxDistance: &maxDistance,
		},
	}
	return b
}

// KNNWithOptions adds a k-NN query with full options.
// field: the knn_vector field name
// vector: the query vector
// k: number of nearest neighbors to return (or max results when using min_score/max_distance)
// opts: optional parameters like filter, min_score, max_distance, ef_search, etc.
// Example:
//
//	builder.KNNWithOptions("embedding", []float32{0.1, 0.2, 0.3}, 10, KNNQueryOptions{
//	    MinScore: Float64Ptr(0.8),
//	    EfSearch: IntPtr(100),
//	})
//
// Note: In OpenSearch 3.x, k-NN queries must be at the top level of the query
// and require exactly ONE of k, distance, or score. When min_score or max_distance
// is set, K is not used in the query (Size is set instead to limit results).
func (b *QueryBuilder) KNNWithOptions(field string, vector []float32, k int, opts KNNQueryOptions) *QueryBuilder {
	knn := &KNNQuery{
		Vector: vector,
		Filter: opts.Filter,
		Boost:  opts.Boost,
	}

	// In OpenSearch 3.x, only one of k, min_score, or max_distance can be set.
	// Determine which to use based on options provided.
	if opts.MinScore != nil {
		knn.MinScore = opts.MinScore
		b.request.Size = int64(k)
	} else if opts.MaxDistance != nil {
		knn.MaxDistance = opts.MaxDistance
		b.request.Size = int64(k)
	} else {
		knn.K = k
	}

	// Add method parameters if any are set
	if opts.EfSearch != nil || opts.Nprobe != nil {
		knn.MethodParameters = &KNNMethodParameters{
			EfSearch: opts.EfSearch,
			Nprobe:   opts.Nprobe,
		}
	}

	// Add rescore options if set
	if opts.Rescore != nil {
		knn.Rescore = opts.Rescore
		if opts.OversampleFactor != nil {
			knn.RescoreContext = &KNNRescoreContext{
				OversampleFactor: opts.OversampleFactor,
			}
		}
	}

	b.knnQuery = &knnQueryConfig{
		field: field,
		query: knn,
	}
	return b
}

// Aggregations

// TermsAgg adds a terms aggregation
func (b *QueryBuilder) TermsAgg(name string, field string, size int64) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("terms aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Terms: &Terms{
			Field: resolvedField,
			Size:  size,
		},
	}

	return b
}

// --- Metric Aggregations ---

// SumAgg adds a sum aggregation
func (b *QueryBuilder) SumAgg(name string, field string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("sum aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Sum: &Agg{Field: resolvedField},
	}

	return b
}

// AvgAgg adds an average aggregation
func (b *QueryBuilder) AvgAgg(name string, field string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("avg aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Avg: &Agg{Field: resolvedField},
	}

	return b
}

// MinAgg adds a min aggregation
func (b *QueryBuilder) MinAgg(name string, field string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("min aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Min: &Agg{Field: resolvedField},
	}

	return b
}

// MaxAgg adds a max aggregation
func (b *QueryBuilder) MaxAgg(name string, field string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("max aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Max: &Agg{Field: resolvedField},
	}

	return b
}

// CardinalityAgg adds a cardinality aggregation (unique count)
func (b *QueryBuilder) CardinalityAgg(name string, field string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("cardinality aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Cardinality: &Cardinality{Field: resolvedField},
	}

	return b
}

// ValueCountAgg adds a value_count aggregation
func (b *QueryBuilder) ValueCountAgg(name string, field string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("value_count aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		ValueCount: &Agg{Field: resolvedField},
	}

	return b
}

// StatsAgg adds a stats aggregation (min, max, avg, sum, count)
func (b *QueryBuilder) StatsAgg(name string, field string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("stats aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Stats: &Agg{Field: resolvedField},
	}

	return b
}

// ExtendedStatsAgg adds an extended_stats aggregation
func (b *QueryBuilder) ExtendedStatsAgg(name string, field string, sigma int64) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("extended_stats aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		ExtendedStats: &ExtendedStats{Field: resolvedField, Sigma: sigma},
	}

	return b
}

// PercentilesAgg adds a percentiles aggregation
func (b *QueryBuilder) PercentilesAgg(name string, field string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("percentiles aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Percentiles: &Agg{Field: resolvedField},
	}

	return b
}

// PercentileRanksAgg adds a percentile_ranks aggregation
func (b *QueryBuilder) PercentileRanksAgg(name string, field string, values []int64) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("percentile_ranks aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		PercentileRanks: &PercentileRanks{Field: resolvedField, Values: values},
	}

	return b
}

// MatrixStatsAgg adds a matrix_stats aggregation
func (b *QueryBuilder) MatrixStatsAgg(name string, fields []string) *QueryBuilder {
	// Resolve all fields
	resolvedFields := make([]string, 0, len(fields))
	for _, field := range fields {
		resolvedField, err := b.resolveField(field, QueryTypeAggregation)
		if err != nil {
			b.errors = append(b.errors, fmt.Errorf("matrix_stats aggregation field '%s': %w", field, err))
			return b
		}
		resolvedFields = append(resolvedFields, resolvedField)
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		MatrixStats: map[string][]string{"fields": resolvedFields},
	}

	return b
}

// TopHitsAgg adds a top_hits aggregation
func (b *QueryBuilder) TopHitsAgg(name string, size int64) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		TopHits: &TopHits{Size: size},
	}

	return b
}

// --- Bucket Aggregations ---

// DateHistogramAgg adds a date histogram aggregation
func (b *QueryBuilder) DateHistogramAgg(name string, field string, interval string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("date_histogram aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		DateHistogram: &Histogram{
			Field:    resolvedField,
			Interval: interval,
		},
	}

	return b
}

// HistogramAgg adds a numeric histogram aggregation
func (b *QueryBuilder) HistogramAgg(name string, field string, interval float64) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("histogram aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Histogram: &Histogram{
			Field:    resolvedField,
			Interval: interval,
		},
	}

	return b
}

// RangeAgg adds a range aggregation
func (b *QueryBuilder) RangeAgg(name string, field string, ranges []map[string]interface{}) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("range aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Range: &Range{Field: resolvedField, Ranges: ranges},
	}

	return b
}

// DateRangeAgg adds a date_range aggregation
func (b *QueryBuilder) DateRangeAgg(name string, field string, format string, ranges []map[string]interface{}) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("date_range aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		DateRange: &DateRange{
			Format: format,
			Range:  Range{Field: resolvedField, Ranges: ranges},
		},
	}

	return b
}

// IPRangeAgg adds an ip_range aggregation
func (b *QueryBuilder) IPRangeAgg(name string, field string, ranges []map[string]interface{}) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("ip_range aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		IPRange: &Range{Field: resolvedField, Ranges: ranges},
	}

	return b
}

// MultiTermsAgg adds a multi_terms aggregation
func (b *QueryBuilder) MultiTermsAgg(name string, terms []Agg, order map[string]string) *QueryBuilder {
	// Resolve fields in each term
	resolvedTerms := make([]Agg, 0, len(terms))
	for _, term := range terms {
		resolvedField, err := b.resolveField(term.Field, QueryTypeAggregation)
		if err != nil {
			b.errors = append(b.errors, fmt.Errorf("multi_terms aggregation field '%s': %w", term.Field, err))
			return b
		}
		resolvedTerms = append(resolvedTerms, Agg{Field: resolvedField, Size: term.Size})
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		MultiTerms: &MultiTerms{Terms: resolvedTerms, Order: order},
	}

	return b
}

// SignificantTermsAgg adds a significant_terms aggregation
func (b *QueryBuilder) SignificantTermsAgg(name string, field string) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("significant_terms aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		SignificantTerms: &Agg{Field: resolvedField},
	}

	return b
}

// SignificantTextAgg adds a significant_text aggregation
func (b *QueryBuilder) SignificantTextAgg(name string, field string, opts map[string]interface{}) *QueryBuilder {
	// significant_text works on text fields, but doesn't need .keyword
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	if opts == nil {
		opts = make(map[string]interface{})
	}
	opts["field"] = field

	b.request.Aggs[name] = Aggs{
		SignificantText: opts,
	}

	return b
}

// FilterAgg adds a filter aggregation
func (b *QueryBuilder) FilterAgg(name string, filter Query) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	filterMap := make(map[string]interface{})
	if filter.Bool != nil {
		filterMap["bool"] = filter.Bool
	} else if filter.Term != nil {
		filterMap["term"] = filter.Term
	} else if filter.Terms != nil {
		filterMap["terms"] = filter.Terms
	} else if filter.Range != nil {
		filterMap["range"] = filter.Range
	} else if filter.Exists != nil {
		filterMap["exists"] = filter.Exists
	} else if filter.Match != nil {
		filterMap["match"] = filter.Match
	}

	b.request.Aggs[name] = Aggs{
		Filter: filterMap,
	}

	return b
}

// FiltersAgg adds a filters aggregation
func (b *QueryBuilder) FiltersAgg(name string, filters map[string]interface{}) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Filters: filters,
	}

	return b
}

// GlobalAgg adds a global aggregation
func (b *QueryBuilder) GlobalAgg(name string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Global: struct{}{},
	}

	return b
}

// NestedAgg adds a nested aggregation
func (b *QueryBuilder) NestedAgg(name string, path string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Nested: map[string]string{"path": path},
	}

	return b
}

// ReverseNestedAgg adds a reverse_nested aggregation
func (b *QueryBuilder) ReverseNestedAgg(name string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		ReverseNested: struct{}{},
	}

	return b
}

// SamplerAgg adds a sampler aggregation
func (b *QueryBuilder) SamplerAgg(name string, shardSize int) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Sampler: map[string]interface{}{"shard_size": shardSize},
	}

	return b
}

// DiversifiedSamplerAgg adds a diversified_sampler aggregation
func (b *QueryBuilder) DiversifiedSamplerAgg(name string, field string, shardSize int) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("diversified_sampler aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		DiversifiedSampler: map[string]interface{}{
			"field":      resolvedField,
			"shard_size": shardSize,
		},
	}

	return b
}

// AdjacencyMatrixAgg adds an adjacency_matrix aggregation
func (b *QueryBuilder) AdjacencyMatrixAgg(name string, filters map[string]interface{}) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		AdjacencyMatrix: filters,
	}

	return b
}

// --- Geo Aggregations ---

// GeoDistanceAgg adds a geo_distance aggregation
func (b *QueryBuilder) GeoDistanceAgg(name string, field string, origin interface{}, ranges []map[string]interface{}) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("geo_distance aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		GeoDistance: &GeoDistance{
			Origin: origin,
			Range:  Range{Field: resolvedField, Ranges: ranges},
		},
	}

	return b
}

// GeohashGridAgg adds a geohash_grid aggregation
func (b *QueryBuilder) GeohashGridAgg(name string, field string, precision int) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("geohash_grid aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		GeohashGrid: &Grid{Field: resolvedField, Precision: precision},
	}

	return b
}

// GeohexGridAgg adds a geohex_grid aggregation
func (b *QueryBuilder) GeohexGridAgg(name string, field string, precision int) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("geohex_grid aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		GeohexGrid: &Grid{Field: resolvedField, Precision: precision},
	}

	return b
}

// GeotileGridAgg adds a geotile_grid aggregation
func (b *QueryBuilder) GeotileGridAgg(name string, field string, precision int) *QueryBuilder {
	resolvedField, err := b.resolveField(field, QueryTypeAggregation)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("geotile_grid aggregation field '%s': %w", field, err))
		return b
	}

	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		GeotileGrid: &Grid{Field: resolvedField, Precision: precision},
	}

	return b
}

// --- Pipeline Aggregations ---

// SumBucketAgg adds a sum_bucket pipeline aggregation
func (b *QueryBuilder) SumBucketAgg(name string, bucketsPath string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		SumBucket: &PipelineAgg{BucketsPath: bucketsPath},
	}

	return b
}

// AvgBucketAgg adds an avg_bucket pipeline aggregation
func (b *QueryBuilder) AvgBucketAgg(name string, bucketsPath string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		AvgBucket: &PipelineAgg{BucketsPath: bucketsPath},
	}

	return b
}

// MinBucketAgg adds a min_bucket pipeline aggregation
func (b *QueryBuilder) MinBucketAgg(name string, bucketsPath string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		MinBucket: &PipelineAgg{BucketsPath: bucketsPath},
	}

	return b
}

// MaxBucketAgg adds a max_bucket pipeline aggregation
func (b *QueryBuilder) MaxBucketAgg(name string, bucketsPath string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		MaxBucket: &PipelineAgg{BucketsPath: bucketsPath},
	}

	return b
}

// StatsBucketAgg adds a stats_bucket pipeline aggregation
func (b *QueryBuilder) StatsBucketAgg(name string, bucketsPath string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		StatsBucket: &PipelineAgg{BucketsPath: bucketsPath},
	}

	return b
}

// ExtendedStatsBucketAgg adds an extended_stats_bucket pipeline aggregation
func (b *QueryBuilder) ExtendedStatsBucketAgg(name string, bucketsPath string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		ExtendedStatsBucket: &PipelineAgg{BucketsPath: bucketsPath},
	}

	return b
}

// CumulativeSumAgg adds a cumulative_sum pipeline aggregation
func (b *QueryBuilder) CumulativeSumAgg(name string, bucketsPath string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		CumulativeSum: &PipelineAgg{BucketsPath: bucketsPath},
	}

	return b
}

// DerivativeAgg adds a derivative pipeline aggregation
func (b *QueryBuilder) DerivativeAgg(name string, bucketsPath string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		Derivative: &PipelineAgg{BucketsPath: bucketsPath},
	}

	return b
}

// MovingAvgAgg adds a moving_avg pipeline aggregation
func (b *QueryBuilder) MovingAvgAgg(name string, bucketsPath string, window int, model string) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		MovingAvg: &MovingAvg{
			Window:      window,
			Model:       model,
			PipelineAgg: PipelineAgg{BucketsPath: bucketsPath},
		},
	}

	return b
}

// SerialDiffAgg adds a serial_diff pipeline aggregation
func (b *QueryBuilder) SerialDiffAgg(name string, bucketsPath string, lag int) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		SerialDiff: &SerialDiff{
			Lag:         lag,
			PipelineAgg: PipelineAgg{BucketsPath: bucketsPath},
		},
	}

	return b
}

// BucketSortAgg adds a bucket_sort pipeline aggregation
func (b *QueryBuilder) BucketSortAgg(name string, sort []map[string]interface{}, size int) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}

	b.request.Aggs[name] = Aggs{
		BucketSort: map[string]interface{}{
			"sort": sort,
			"size": size,
		},
	}

	return b
}

// --- Aggregation Utilities ---

// SubAgg adds a sub-aggregation to an existing aggregation
func (b *QueryBuilder) SubAgg(parentName string, childName string, childAgg Aggs) *QueryBuilder {
	if b.request.Aggs == nil {
		b.errors = append(b.errors, fmt.Errorf("parent aggregation '%s' does not exist", parentName))
		return b
	}

	parent, exists := b.request.Aggs[parentName]
	if !exists {
		b.errors = append(b.errors, fmt.Errorf("parent aggregation '%s' does not exist", parentName))
		return b
	}

	if parent.Aggs == nil {
		parent.Aggs = make(map[string]Aggs)
	}
	parent.Aggs[childName] = childAgg
	b.request.Aggs[parentName] = parent
	return b
}

// Agg adds a raw Aggs object (for advanced/custom aggregations)
func (b *QueryBuilder) Agg(name string, agg Aggs) *QueryBuilder {
	if b.request.Aggs == nil {
		b.request.Aggs = make(map[string]Aggs)
	}
	b.request.Aggs[name] = agg
	return b
}

// --- BoolBuilder Integration ---

// Bool creates a new BoolBuilder with inherited context from QueryBuilder
func (b *QueryBuilder) Bool() *BoolBuilder {
	return NewBoolBuilderWithMapper(b.ctx, b.indices, b.mapper, b.processName)
}

// MustBool adds a nested bool query to must clause
func (b *QueryBuilder) MustBool(nested *BoolBuilder) *QueryBuilder {
	if nested == nil {
		return b
	}
	b.errors = append(b.errors, nested.errors...)
	b.request.Query.Bool.Must = append(b.request.Query.Bool.Must, Query{Bool: &nested.query})
	return b
}

// ShouldBool adds a nested bool query to should clause
func (b *QueryBuilder) ShouldBool(nested *BoolBuilder) *QueryBuilder {
	if nested == nil {
		return b
	}
	b.errors = append(b.errors, nested.errors...)
	b.request.Query.Bool.Should = append(b.request.Query.Bool.Should, Query{Bool: &nested.query})
	return b
}

// FilterBool adds a nested bool query to filter clause
func (b *QueryBuilder) FilterBool(nested *BoolBuilder) *QueryBuilder {
	if nested == nil {
		return b
	}
	b.errors = append(b.errors, nested.errors...)
	b.request.Query.Bool.Filter = append(b.request.Query.Bool.Filter, Query{Bool: &nested.query})
	return b
}

// MustNotBool adds a nested bool query to must_not clause
func (b *QueryBuilder) MustNotBool(nested *BoolBuilder) *QueryBuilder {
	if nested == nil {
		return b
	}
	b.errors = append(b.errors, nested.errors...)
	b.request.Query.Bool.MustNot = append(b.request.Query.Bool.MustNot, Query{Bool: &nested.query})
	return b
}

// Build returns the constructed SearchRequest
func (b *QueryBuilder) Build() SearchRequest {
	if len(b.errors) > 0 {
		_ = catcher.Error("QueryBuilder has errors", errors.New("see the errors list in the arguments"), map[string]any{
			"errors":  b.errors,
			"process": b.processName,
		})
	}

	// Handle k-NN query placement for OpenSearch 3.x compatibility.
	// k-NN queries must be at the top level of the query, not nested in bool.must.
	if b.knnQuery != nil {
		// Check if there are any bool clauses that need to be combined as filters
		boolHasClauses := b.request.Query != nil && b.request.Query.Bool != nil &&
			(len(b.request.Query.Bool.Must) > 0 ||
				len(b.request.Query.Bool.Filter) > 0 ||
				len(b.request.Query.Bool.Should) > 0 ||
				len(b.request.Query.Bool.MustNot) > 0)

		if boolHasClauses {
			// Combine existing bool clauses as a filter inside the k-NN query
			if b.knnQuery.query.Filter == nil {
				b.knnQuery.query.Filter = &Query{Bool: b.request.Query.Bool}
			} else {
				// Merge existing k-NN filter with bool clauses
				if b.knnQuery.query.Filter.Bool == nil {
					b.knnQuery.query.Filter.Bool = &Bool{}
				}
				b.knnQuery.query.Filter.Bool.Must = append(b.knnQuery.query.Filter.Bool.Must, b.request.Query.Bool.Must...)
				b.knnQuery.query.Filter.Bool.Filter = append(b.knnQuery.query.Filter.Bool.Filter, b.request.Query.Bool.Filter...)
				b.knnQuery.query.Filter.Bool.Should = append(b.knnQuery.query.Filter.Bool.Should, b.request.Query.Bool.Should...)
				b.knnQuery.query.Filter.Bool.MustNot = append(b.knnQuery.query.Filter.Bool.MustNot, b.request.Query.Bool.MustNot...)
			}
		}

		// Set k-NN query at the top level
		b.request.Query = &Query{
			KNN: map[string]*KNNQuery{
				b.knnQuery.field: b.knnQuery.query,
			},
		}
	}

	return b.request
}

// BuildWithErrors returns the SearchRequest and any errors encountered
func (b *QueryBuilder) BuildWithErrors() (SearchRequest, []error) {
	return b.Build(), b.errors
}

// GetMappingConflicts returns fields with type conflicts across indices
func (b *QueryBuilder) GetMappingConflicts() []FieldInfo {
	conflicts := []FieldInfo{}

	for _, indexPattern := range b.indices {
		merged, err := b.mapper.GetMergedMapping(b.ctx, indexPattern)
		if err != nil {
			continue
		}
		conflicts = append(conflicts, merged.GetConflicts()...)
	}

	return conflicts
}

// Helper methods

func (b *QueryBuilder) resolveField(field string, queryType QueryType) (string, error) {
	// Try to get merged mapping for all index patterns
	for _, indexPattern := range b.indices {
		merged, err := b.mapper.GetMergedMapping(b.ctx, indexPattern)
		if err != nil {
			// If mapping fetch fails, use field as-is
			_ = catcher.Error("failed to fetch mapping", err, map[string]any{
				"pattern": indexPattern,
				"process": b.processName,
			})
			continue
		}

		// Try to resolve field
		resolvedField, err := merged.ResolveFieldName(field, queryType)
		if err == nil {
			return resolvedField, nil
		}

		_ = catcher.Error("failed to resolve field name", err, map[string]any{
			"field":   field,
			"process": b.processName,
		})
	}

	// If no mapping found or resolution failed, use field as-is
	return field, nil
}

func (b *QueryBuilder) getFieldInfo(field string) (FieldInfo, bool) {
	for _, indexPattern := range b.indices {
		merged, err := b.mapper.GetMergedMapping(b.ctx, indexPattern)
		if err != nil {
			_ = catcher.Error("failed to get merged mapping", err, map[string]any{
				"pattern": indexPattern,
				"process": b.processName,
				"field":   field,
			})
			continue
		}

		info, ok := merged.GetFieldInfo(field)
		if ok {
			return info, true
		}
	}

	return FieldInfo{}, false
}
