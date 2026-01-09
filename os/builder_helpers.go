package os

import "fmt"

// Standalone query constructors for use in boolean logic

// TermQuery creates a term query (exact match)
func TermQuery(field string, value interface{}) Query {
	return Query{
		Term: map[string]map[string]interface{}{
			field: {"value": value},
		},
	}
}

// TermsQuery creates a terms query (match any value)
func TermsQuery(field string, values []interface{}) Query {
	return Query{
		Terms: map[string][]interface{}{
			field: values,
		},
	}
}

// MatchQuery creates a full-text match query
func MatchQuery(field string, query string) Query {
	return Query{
		Match: map[string]Match{
			field: {Query: query},
		},
	}
}

// MatchPhraseQuery creates a match phrase query
func MatchPhraseQuery(field string, query string) Query {
	return Query{
		MatchPhrase: map[string]MatchPhrase{
			field: {Query: query},
		},
	}
}

// MatchPhrasePrefixQuery creates a match phrase prefix query
func MatchPhrasePrefixQuery(field string, query string) Query {
	return Query{
		MatchPhrasePrefix: map[string]MatchPhrasePrefix{
			field: {Query: query},
		},
	}
}

// MultiMatchQuery creates a multi-match query across multiple fields
func MultiMatchQuery(query string, fields []string) Query {
	return Query{
		MultiMatch: &MultiMatch{
			Query:  query,
			Fields: fields,
		},
	}
}

// RangeQuery creates a range query
// operator can be: "gte", "lte", "gt", "lt"
func RangeQuery(field string, operator string, value interface{}) Query {
	return Query{
		Range: map[string]map[string]interface{}{
			field: {operator: value},
		},
	}
}

// RangeQueryBetween creates a range query between two values
func RangeQueryBetween(field string, gte, lte interface{}) Query {
	return Query{
		Range: map[string]map[string]interface{}{
			field: {
				"gte": gte,
				"lte": lte,
			},
		},
	}
}

// ExistsQuery creates an exists query (field must be present)
func ExistsQuery(field string) Query {
	return Query{
		Exists: map[string]string{"field": field},
	}
}

// WildcardQuery creates a wildcard query
func WildcardQuery(field string, pattern string) Query {
	return Query{
		Wildcard: map[string]map[string]interface{}{
			field: {"value": pattern},
		},
	}
}

// PrefixQuery creates a prefix query
func PrefixQuery(field string, prefix string) Query {
	return Query{
		Prefix: map[string]string{field: prefix},
	}
}

// FuzzyQuery creates a fuzzy query
func FuzzyQuery(field string, value string, fuzziness ...string) Query {
	fuzzy := "AUTO"
	if len(fuzziness) > 0 {
		fuzzy = fuzziness[0]
	}

	return Query{
		Fuzzy: map[string]map[string]interface{}{
			field: {
				"value":     value,
				"fuzziness": fuzzy,
			},
		},
	}
}

// RegexpQuery creates a regexp query
func RegexpQuery(field string, regexp string) Query {
	return Query{
		Regexp: map[string]string{field: regexp},
	}
}

// IDsQuery creates an IDs query (match by document IDs)
func IDsQuery(ids []interface{}) Query {
	return Query{
		IDs: map[string][]interface{}{"values": ids},
	}
}

// CIDRQuery creates a term query for "ip" type fields using CIDR notation.
// This is for fields mapped as "ip" type (storing single IP addresses).
// OpenSearch natively supports CIDR notation in term queries on ip fields.
// Example: CIDRQuery("client_ip", "192.168.0.0/16") matches all IPs in 192.168.x.x range
// Note: For "ip_range" type fields, use IPRangeContainsQuery or IPRangeIntersectsQuery instead.
func CIDRQuery(field string, cidr string) Query {
	return Query{
		Term: map[string]map[string]interface{}{
			field: {"value": cidr},
		},
	}
}

// CIDRsQuery creates a terms query for "ip" type fields using multiple CIDR notations.
// This is for fields mapped as "ip" type (storing single IP addresses).
// Example: CIDRsQuery("client_ip", "192.168.0.0/24", "10.0.0.0/8")
// Note: For "ip_range" type fields, use IPRangeContainsQuery or IPRangeIntersectsQuery instead.
func CIDRsQuery(field string, cidrs ...string) Query {
	values := make([]interface{}, len(cidrs))
	for i, cidr := range cidrs {
		values[i] = cidr
	}
	return Query{
		Terms: map[string][]interface{}{
			field: values,
		},
	}
}

// === IP Range Field Queries ===
// These are for fields mapped as "ip_range" type (storing IP address ranges).
// ip_range fields store ranges like {"gte": "10.0.0.0", "lte": "10.255.255.255"} or CIDR notation.

// IPRangeContainsQuery creates a range query that matches ip_range fields containing the given IP.
// Use this to find documents where the stored IP range contains the query IP address.
// The "relation" parameter is set to "contains" to match ranges that fully contain the query value.
// Example: IPRangeContainsQuery("allowed_ips", "192.168.1.50") matches ranges like 192.168.0.0/16
func IPRangeContainsQuery(field string, ip string) Query {
	return Query{
		Range: map[string]map[string]interface{}{
			field: {
				"gte":      ip,
				"lte":      ip,
				"relation": "contains",
			},
		},
	}
}

// IPRangeIntersectsQuery creates a range query that matches ip_range fields intersecting with the given range.
// Use this to find documents where the stored IP range overlaps with the query range.
// The "relation" parameter is set to "intersects" (default behavior).
// Example: IPRangeIntersectsQuery("blocked_ranges", "192.168.0.0", "192.168.255.255")
func IPRangeIntersectsQuery(field string, fromIP, toIP string) Query {
	return Query{
		Range: map[string]map[string]interface{}{
			field: {
				"gte":      fromIP,
				"lte":      toIP,
				"relation": "intersects",
			},
		},
	}
}

// IPRangeWithinQuery creates a range query that matches ip_range fields entirely within the given range.
// Use this to find documents where the stored IP range is completely contained within the query range.
// The "relation" parameter is set to "within".
// Example: IPRangeWithinQuery("subnet", "10.0.0.0", "10.255.255.255") matches 10.0.0.0/24, 10.1.0.0/16, etc.
func IPRangeWithinQuery(field string, fromIP, toIP string) Query {
	return Query{
		Range: map[string]map[string]interface{}{
			field: {
				"gte":      fromIP,
				"lte":      toIP,
				"relation": "within",
			},
		},
	}
}

// QueryStringQuery creates a query string query
func QueryStringQuery(query string, defaultField ...string) Query {
	qs := &QueryString{
		Query: query,
	}
	if len(defaultField) > 0 {
		qs.DefaultField = defaultField[0]
	}
	return Query{QueryString: qs}
}

// SimpleQueryStringQuery creates a simple query string query
func SimpleQueryStringQuery(query string, fields ...string) Query {
	return Query{
		SimpleQueryString: &SimpleQueryString{
			Query:  query,
			Fields: fields,
		},
	}
}

// Note: Smart helper functions are implemented in QueryBuilder
// The standalone query constructors below don't have access to mapping info

// Range helpers

// RangeGte creates a range query with gte (greater than or equal)
func RangeGte(field string, value interface{}) Query {
	return RangeQuery(field, "gte", value)
}

// RangeLte creates a range query with lte (less than or equal)
func RangeLte(field string, value interface{}) Query {
	return RangeQuery(field, "lte", value)
}

// RangeGt creates a range query with gt (greater than)
func RangeGt(field string, value interface{}) Query {
	return RangeQuery(field, "gt", value)
}

// RangeLt creates a range query with lt (less than)
func RangeLt(field string, value interface{}) Query {
	return RangeQuery(field, "lt", value)
}

// === KNN (k-Nearest Neighbor) Vector Search Queries ===
// These are for fields mapped as "knn_vector" type (storing dense vectors for similarity search).
// k-NN queries find the k most similar vectors to a given query vector.

// KNNQueryOptions provides optional parameters for k-NN queries.
type KNNQueryOptions struct {
	// Filter is an optional query to pre-filter documents before k-NN search.
	Filter *Query
	// MinScore specifies the minimum score threshold. Cannot be used with MaxDistance.
	MinScore *float64
	// MaxDistance specifies the maximum distance threshold. Cannot be used with MinScore.
	MaxDistance *float64
	// Boost multiplies the score of all results.
	Boost *float64
	// EfSearch controls the HNSW search list size (higher = more accurate, slower).
	EfSearch *int
	// Nprobe controls the number of IVF clusters to search.
	Nprobe *int
	// Rescore enables rescoring with exact distance calculations.
	Rescore *bool
	// OversampleFactor controls how many candidates to fetch for rescoring.
	OversampleFactor *float64
}

// NewKNNQuery creates a basic k-NN query to find the k nearest neighbors.
// field: the knn_vector field name
// vector: the query vector (must match the field's dimension)
// k: number of nearest neighbors to return
// Example: NewKNNQuery("embedding", []float32{0.1, 0.2, 0.3}, 10)
func NewKNNQuery(field string, vector []float32, k int) Query {
	return Query{
		KNN: map[string]*KNNQuery{
			field: {
				Vector: vector,
				K:      k,
			},
		},
	}
}

// NewKNNQueryWithOptions creates a k-NN query with additional options.
// field: the knn_vector field name
// vector: the query vector (must match the field's dimension)
// k: number of nearest neighbors to return
// opts: optional parameters like filter, min_score, max_distance, etc.
// Example:
//
//	NewKNNQueryWithOptions("embedding", []float32{0.1, 0.2, 0.3}, 10, KNNQueryOptions{
//	    MinScore: Float64Ptr(0.8),
//	    Filter: &Query{Term: map[string]map[string]interface{}{"category": {"value": "tech"}}},
//	})
func NewKNNQueryWithOptions(field string, vector []float32, k int, opts KNNQueryOptions) Query {
	knn := &KNNQuery{
		Vector:      vector,
		K:           k,
		Filter:      opts.Filter,
		MinScore:    opts.MinScore,
		MaxDistance: opts.MaxDistance,
		Boost:       opts.Boost,
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

	return Query{
		KNN: map[string]*KNNQuery{
			field: knn,
		},
	}
}

// NewKNNQueryWithFilter creates a k-NN query with a pre-filter.
// The filter is applied before the k-NN search, so only matching documents are considered.
// field: the knn_vector field name
// vector: the query vector
// k: number of nearest neighbors to return
// filter: query to filter documents before k-NN search
// Example:
//
//	NewKNNQueryWithFilter("embedding", []float32{0.1, 0.2, 0.3}, 10,
//	    TermQuery("status.keyword", "active"))
func NewKNNQueryWithFilter(field string, vector []float32, k int, filter Query) Query {
	return Query{
		KNN: map[string]*KNNQuery{
			field: {
				Vector: vector,
				K:      k,
				Filter: &filter,
			},
		},
	}
}

// NewKNNQueryWithMinScore creates a k-NN query with a minimum score threshold.
// Only results with similarity score >= minScore are returned.
// field: the knn_vector field name
// vector: the query vector
// k: number of nearest neighbors to return
// minScore: minimum similarity score threshold (0.0 to 1.0 for cosine similarity)
// Example: NewKNNQueryWithMinScore("embedding", []float32{0.1, 0.2, 0.3}, 10, 0.8)
func NewKNNQueryWithMinScore(field string, vector []float32, k int, minScore float64) Query {
	return Query{
		KNN: map[string]*KNNQuery{
			field: {
				Vector:   vector,
				K:        k,
				MinScore: &minScore,
			},
		},
	}
}

// NewKNNQueryWithMaxDistance creates a k-NN query with a maximum distance threshold.
// Only results within maxDistance from the query vector are returned.
// The distance metric depends on the space_type configured in the index mapping.
// field: the knn_vector field name
// vector: the query vector
// k: number of nearest neighbors to return
// maxDistance: maximum distance threshold (e.g., for L2 distance)
// Example: NewKNNQueryWithMaxDistance("embedding", []float32{0.1, 0.2, 0.3}, 10, 100.0)
func NewKNNQueryWithMaxDistance(field string, vector []float32, k int, maxDistance float64) Query {
	return Query{
		KNN: map[string]*KNNQuery{
			field: {
				Vector:      vector,
				K:           k,
				MaxDistance: &maxDistance,
			},
		},
	}
}

// Float64Ptr is a helper to create a pointer to a float64 value.
// Useful for setting optional KNN query parameters.
func Float64Ptr(v float64) *float64 {
	return &v
}

// IntPtr is a helper to create a pointer to an int value.
// Useful for setting optional KNN query parameters.
func IntPtr(v int) *int {
	return &v
}

// BoolPtr is a helper to create a pointer to a bool value.
// Useful for setting optional KNN query parameters.
func BoolPtr(v bool) *bool {
	return &v
}

// Validation helpers

// ValidateQuery checks if a query is valid for the given field type
func ValidateQuery(fieldType string, queryType QueryType) error {
	switch queryType {
	case QueryTypeTerm, QueryTypeTerms, QueryTypeRange:
		if fieldType == "text" {
			return fmt.Errorf("field type '%s' requires .keyword sub-field for %s query", fieldType, queryTypeString(queryType))
		}
		return nil

	case QueryTypeMatch, QueryTypeMatchPhrase:
		if fieldType != "text" {
			return fmt.Errorf("field type '%s' requires text type for %s query", fieldType, queryTypeString(queryType))
		}
		return nil

	case QueryTypeExists, QueryTypeWildcard:
		// These work with any field type
		return nil

	default:
		return nil
	}
}
