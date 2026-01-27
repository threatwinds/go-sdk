package os

// SearchResult represents the response from an OpenSearch search query.
type SearchResult struct {
	Took         int64                  `json:"took"`
	TimedOut     bool                   `json:"timed_out"`
	Shards       Shards                 `json:"_shards"`
	Hits         Hits                   `json:"hits"`
	Aggregations map[string]interface{} `json:"aggregations"`
}

// Hits contain the search hits and metadata.
type Hits struct {
	Total    Total       `json:"total"`
	MaxScore interface{} `json:"max_score"`
	Hits     []Hit       `json:"hits"`
}

// HitSource represents the _source field of a search hit as a map.
type HitSource map[string]interface{}

// Hit represents a single search result document.
type Hit struct {
	Index   string                 `json:"_index"`
	ID      string                 `json:"_id"`
	Version int64                  `json:"_version"`
	Score   interface{}            `json:"_score"`
	Source  HitSource              `json:"_source"`
	Fields  map[string]interface{} `json:"fields"`
	Sort    []any                  `json:"sort"`
	Found   bool                   `json:"found,omitempty"`
}

// Total contains the total number of matching documents.
type Total struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

// Shards contain information about the shards involved in the search.
type Shards struct {
	Total      int64 `json:"total"`
	Successful int64 `json:"successful"`
	Skipped    int64 `json:"skipped"`
	Failed     int64 `json:"failed"`
}

// SearchRequest represents an OpenSearch search request body.
type SearchRequest struct {
	Version      bool                                `json:"version,omitempty"`
	From         int64                               `json:"from,omitempty"`
	Size         int64                               `json:"size"`
	Sort         []map[string]map[string]interface{} `json:"sort,omitempty"`
	StoredFields []string                            `json:"stored_fields,omitempty"`
	Source       *Source                             `json:"_source,omitempty"`
	Query        *Query                              `json:"query,omitempty"`
	Collapse     *Collapse                           `json:"collapse,omitempty"`
	Aggs         map[string]Aggs                     `json:"aggs,omitempty"`
	SearchAfter  []any                               `json:"search_after,omitempty"`
	ScriptFields interface{}                         `json:"script_fields,omitempty"`
}

// Collapse specifies field collapsing (deduplication) settings.
type Collapse struct {
	Field string `json:"field,omitempty"`
}

// Aggs represents aggregation configurations for OpenSearch queries.
type Aggs struct {
	Aggs                map[string]Aggs        `json:"aggs,omitempty"`
	Avg                 *Agg                   `json:"avg,omitempty"`
	Sum                 *Agg                   `json:"sum,omitempty"`
	Min                 *Agg                   `json:"min,omitempty"`
	Max                 *Agg                   `json:"max,omitempty"`
	Cardinality         *Cardinality           `json:"cardinality,omitempty"`
	ValueCount          *Agg                   `json:"value_count,omitempty"`
	Stats               *Agg                   `json:"stats,omitempty"`
	ExtendedStats       *ExtendedStats         `json:"extended_stats,omitempty"`
	MatrixStats         map[string][]string    `json:"matrix_stats,omitempty"`
	Percentiles         *Agg                   `json:"percentiles,omitempty"`
	PercentileRanks     *PercentileRanks       `json:"percentile_ranks,omitempty"`
	TopHits             *TopHits               `json:"top_hits,omitempty"`
	Terms               *Terms                 `json:"terms,omitempty"`
	MultiTerms          *MultiTerms            `json:"multi_terms,omitempty"`
	Sampler             map[string]interface{} `json:"sampler,omitempty"`
	DiversifiedSampler  map[string]interface{} `json:"diversified_sampler,omitempty"`
	SignificantTerms    *Agg                   `json:"significant_terms,omitempty"`
	SignificantText     map[string]interface{} `json:"significant_text,omitempty"`
	Histogram           *Histogram             `json:"histogram,omitempty"`
	DateHistogram       *Histogram             `json:"date_histogram,omitempty"`
	Range               *Range                 `json:"range,omitempty"`
	DateRange           *DateRange             `json:"date_range,omitempty"`
	IPRange             *Range                 `json:"ip_range,omitempty"`
	Filter              map[string]interface{} `json:"filter,omitempty"`
	Filters             map[string]interface{} `json:"filters,omitempty"`
	Global              interface{}            `json:"global,omitempty"`
	Nested              map[string]string      `json:"nested,omitempty"`
	ReverseNested       interface{}            `json:"reverse_nested,omitempty"`
	SumBucket           *PipelineAgg           `json:"sum_bucket,omitempty"`
	AvgBucket           *PipelineAgg           `json:"avg_bucket,omitempty"`
	MinBucket           *PipelineAgg           `json:"min_bucket,omitempty"`
	MaxBucket           *PipelineAgg           `json:"max_bucket,omitempty"`
	StatsBucket         *PipelineAgg           `json:"stats_bucket,omitempty"`
	ExtendedStatsBucket *PipelineAgg           `json:"extended_stats_bucket,omitempty"`
	BucketSort          map[string]interface{} `json:"bucket_sort,omitempty"`
	CumulativeSum       *PipelineAgg           `json:"cumulative_sum,omitempty"`
	Derivative          *PipelineAgg           `json:"derivative,omitempty"`
	MovingAvg           *MovingAvg             `json:"moving_avg,omitempty"`
	SerialDiff          *SerialDiff            `json:"serial_diff,omitempty"`
	GeoDistance         *GeoDistance           `json:"geo_distance,omitempty"`
	GeohashGrid         *Grid                  `json:"geohash_grid,omitempty"`
	GeohexGrid          *Grid                  `json:"geohex_grid,omitempty"`
	GeotileGrid         *Grid                  `json:"geotile_grid,omitempty"`
	AdjacencyMatrix     map[string]interface{} `json:"adjacency_matrix,omitempty"`
}

type Grid struct {
	Field     string      `json:"field,omitempty"`
	Precision int         `json:"precision,omitempty"`
	Bounds    interface{} `json:"bounds,omitempty"`
	Size      int         `json:"size,omitempty"`
	ShardSize int         `json:"shard_size,omitempty"`
}

type GeoDistance struct {
	Origin interface{} `json:"origin,omitempty"`
	Range
}

type SerialDiff struct {
	Lag int `json:"lag,omitempty"`
	PipelineAgg
}

type MovingAvg struct {
	Predict  int                    `json:"predict,omitempty"`
	Window   int                    `json:"window,omitempty"`
	Model    string                 `json:"model,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
	PipelineAgg
}

type PipelineAgg struct {
	BucketsPath string `json:"buckets_path,omitempty"`
}

type Range struct {
	Field  string                   `json:"field,omitempty"`
	Ranges []map[string]interface{} `json:"ranges,omitempty"`
}

type DateRange struct {
	Format string `json:"format,omitempty"`
	Range
}

type Terms struct {
	Field       string `json:"field,omitempty"`
	Size        int64  `json:"size,omitempty"`
	Missing     string `json:"missing,omitempty"`
	MinDocCount int64  `json:"min_doc_count,omitempty"`
}

type Histogram struct {
	Field    string      `json:"field,omitempty"`
	Interval interface{} `json:"interval,omitempty"`
}

type MultiTerms struct {
	Terms []Agg             `json:"terms,omitempty"`
	Order map[string]string `json:"order,omitempty"`
}

type TopHits struct {
	Size int64 `json:"size,omitempty"`
}

type PercentileRanks struct {
	Field  string  `json:"field,omitempty"`
	Values []int64 `json:"values,omitempty"`
}

type Agg struct {
	Field string `json:"field,omitempty"`
	Size  int64  `json:"size,omitempty"`
}

type ExtendedStats struct {
	Field string `json:"field,omitempty"`
	Sigma int64  `json:"sigma,omitempty"`
}

type Cardinality struct {
	Field              string `json:"field,omitempty"`
	PrecisionThreshold int64  `json:"precision_threshold,omitempty"`
}

// Query represents an OpenSearch query clause.
// Only one query type should be set per Query instance.
type Query struct {
	Bool              *Bool                             `json:"bool,omitempty"`
	Term              map[string]map[string]interface{} `json:"term,omitempty"`
	Terms             map[string][]interface{}          `json:"terms,omitempty"`
	IDs               map[string][]interface{}          `json:"ids,omitempty"`
	Range             map[string]map[string]interface{} `json:"range,omitempty"`
	Exists            map[string]string                 `json:"exists,omitempty"`
	Prefix            map[string]string                 `json:"prefix,omitempty"`
	Fuzzy             map[string]map[string]interface{} `json:"fuzzy,omitempty"`
	Wildcard          map[string]map[string]interface{} `json:"wildcard,omitempty"`
	Regexp            map[string]string                 `json:"regexp,omitempty"`
	Match             map[string]Match                  `json:"match,omitempty"`
	MultiMatch        *MultiMatch                       `json:"multi_match,omitempty"`
	MatchBoolPrefix   map[string]MatchBoolPrefix        `json:"match_bool_prefix,omitempty"`
	MatchPhrase       map[string]MatchPhrase            `json:"match_phrase,omitempty"`
	MatchPhrasePrefix map[string]MatchPhrasePrefix      `json:"match_phrase_prefix,omitempty"`
	QueryString       *QueryString                      `json:"query_string,omitempty"`
	SimpleQueryString *SimpleQueryString                `json:"simple_query_string,omitempty"`
	KNN               map[string]*KNNQuery              `json:"knn,omitempty"`
}

// KNNQuery represents a k-NN (k-nearest neighbor) vector search query.
// It searches for the k most similar vectors to the provided query vector.
type KNNQuery struct {
	// Vector is the query vector to search for similar vectors.
	// Must have the same dimension as the knn_vector field in the index.
	Vector []float32 `json:"vector"`

	// K is the number of nearest neighbors to return.
	// In OpenSearch 3.x, exactly ONE of k, min_score, or max_distance must be set.
	K int `json:"k,omitempty"`

	// Filter is an optional query to pre-filter documents before the k-NN search.
	// Only documents matching the filter will be considered for k-NN search.
	Filter *Query `json:"filter,omitempty"`

	// MinScore specifies the minimum score threshold for results.
	// Only results with a score >= min_score are returned.
	// Cannot be used with max_distance.
	MinScore *float64 `json:"min_score,omitempty"`

	// MaxDistance specifies the maximum distance threshold for results.
	// Only results within max_distance from the query vector are returned.
	// Cannot be used with min_score.
	MaxDistance *float64 `json:"max_distance,omitempty"`

	// Boost multiplies the score of all results by the boost value.
	// Default is 1.0.
	Boost *float64 `json:"boost,omitempty"`

	// MethodParameters allows fine-tuning of the k-NN algorithm.
	// For HNSW: ef_search controls the size of the dynamic list during search.
	// For IVF: nprobe controls the number of clusters to search.
	MethodParameters *KNNMethodParameters `json:"method_parameters,omitempty"`

	// Rescore enables rescoring of the k-NN results.
	// When true, results are re-ranked using exact distance calculations.
	Rescore *bool `json:"rescore,omitempty"`

	// RescoreContext provides additional context for rescoring.
	RescoreContext *KNNRescoreContext `json:"rescore_context,omitempty"`
}

// KNNMethodParameters contains algorithm-specific parameters for k-NN search.
type KNNMethodParameters struct {
	// EfSearch controls the size of the dynamic list used during HNSW search.
	// Higher values improve accuracy but increase latency.
	// Only applicable for HNSW engine.
	EfSearch *int `json:"ef_search,omitempty"`

	// Nprobe controls the number of clusters to search in IVF-based indices.
	// Higher values improve accuracy but increase latency.
	// Only applicable for IVF engine.
	Nprobe *int `json:"nprobe,omitempty"`
}

// KNNRescoreContext provides context for k-NN rescoring.
type KNNRescoreContext struct {
	// OversampleFactor controls how many candidates to fetch for rescoring.
	// A factor of 2.0 means fetch 2x the k value for rescoring.
	OversampleFactor *float64 `json:"oversample_factor,omitempty"`
}

// Bool represents a boolean query with must, should, filter, and must_not clauses.
type Bool struct {
	Must               []Query     `json:"must,omitempty"`
	Filter             []Query     `json:"filter,omitempty"`
	Should             []Query     `json:"should,omitempty"`
	MustNot            []Query     `json:"must_not,omitempty"`
	MinimumShouldMatch interface{} `json:"minimum_should_match,omitempty"`
}

// Source specifies which fields to include or exclude from the _source in results.
type Source struct {
	Includes []string `json:"includes,omitempty"`
	Excludes []string `json:"excludes,omitempty"`
}

// Match represents a full-text match query configuration.
type Match struct {
	Query               string `json:"query,omitempty"`
	Fuzziness           string `json:"fuzziness,omitempty"`
	FuzzyTranspositions bool   `json:"fuzzy_transpositions,omitempty"`
	Operator            string `json:"operator,omitempty"`
	MinimumShouldMatch  int64  `json:"minimum_should_match,omitempty"`
	Analyzer            string `json:"analyzer,omitempty"`
	ZeroTermsQuery      string `json:"zero_terms_query,omitempty"`
	Lenient             bool   `json:"lenient,omitempty"`
	PrefixLength        int64  `json:"prefix_length,omitempty"`
	MaxExpansions       int64  `json:"max_expansions,omitempty"`
	Boost               int64  `json:"boost,omitempty"`
}

type MultiMatch struct {
	Query                           string   `json:"query,omitempty"`
	Fields                          []string `json:"fields,omitempty"`
	Fuzziness                       string   `json:"fuzziness,omitempty"`
	FuzzyTranspositions             bool     `json:"fuzzy_transpositions,omitempty"`
	Operator                        string   `json:"operator,omitempty"`
	MinimumShouldMatch              int64    `json:"minimum_should_match,omitempty"`
	Analyzer                        string   `json:"analyzer,omitempty"`
	ZeroTermsQuery                  string   `json:"zero_terms_query,omitempty"`
	Lenient                         bool     `json:"lenient,omitempty"`
	PrefixLength                    int64    `json:"prefix_length,omitempty"`
	MaxExpansions                   int64    `json:"max_expansions,omitempty"`
	Boost                           int64    `json:"boost,omitempty"`
	Type                            string   `json:"type,omitempty"`
	TieBreaker                      float64  `json:"tie_breaker,omitempty"`
	AutoGenerateSynonymsPhraseQuery bool     `json:"auto_generate_synonyms_phrase_query,omitempty"`
}

type MatchBoolPrefix struct {
	Query               string `json:"query,omitempty"`
	Fuzziness           string `json:"fuzziness,omitempty"`
	FuzzyTranspositions bool   `json:"fuzzy_transpositions,omitempty"`
	MaxExpansions       int64  `json:"max_expansions,omitempty"`
	PrefixLength        int64  `json:"prefix_length,omitempty"`
	Operator            string `json:"operator,omitempty"`
	MinimumShouldMatch  int64  `json:"minimum_should_match,omitempty"`
	Analyzer            string `json:"analyzer,omitempty"`
}

type MatchPhrase struct {
	Query          string `json:"query,omitempty"`
	Slop           int64  `json:"slop,omitempty"`
	Analyzer       string `json:"analyzer,omitempty"`
	ZeroTermsQuery string `json:"zero_terms_query,omitempty"`
}

type MatchPhrasePrefix struct {
	Query         string `json:"query,omitempty"`
	Analyzer      string `json:"analyzer,omitempty"`
	MaxExpansions int64  `json:"max_expansions,omitempty"`
	Slop          int64  `json:"slop,omitempty"`
}

type QueryString struct {
	Query                           string `json:"query,omitempty"`
	DefaultField                    string `json:"default_field,omitempty"`
	Type                            string `json:"type,omitempty"`
	Fuzziness                       string `json:"fuzziness,omitempty"`
	FuzzyTranspositions             bool   `json:"fuzzy_transpositions,omitempty"`
	FuzzyMaxExpansions              int64  `json:"fuzzy_max_expansions,omitempty"`
	FuzzyPrefixLength               int64  `json:"fuzzy_prefix_length,omitempty"`
	MinimumShouldMatch              int64  `json:"minimum_should_match,omitempty"`
	DefaultOperator                 string `json:"default_operator,omitempty"`
	Analyzer                        string `json:"analyzer,omitempty"`
	Lenient                         bool   `json:"lenient,omitempty"`
	Boost                           int64  `json:"boost,omitempty"`
	AllowLeadingWildcard            bool   `json:"allow_leading_wildcard,omitempty"`
	EnablePositionIncrements        bool   `json:"enable_position_increments,omitempty"`
	PhraseSlop                      int64  `json:"phrase_slop,omitempty"`
	MaxDeterminizedStates           int64  `json:"max_determinized_states,omitempty"`
	TimeZone                        string `json:"time_zone,omitempty"`
	QuoteFieldSuffix                string `json:"quote_field_suffix,omitempty"`
	QuoteAnalyzer                   string `json:"quote_analyzer,omitempty"`
	AnalyzeWildcard                 bool   `json:"analyze_wildcard,omitempty"`
	AutoGenerateSynonymsPhraseQuery bool   `json:"auto_generate_synonyms_phrase_query,omitempty"`
}

type SimpleQueryString struct {
	Query                           string   `json:"query,omitempty"`
	Fields                          []string `json:"fields,omitempty"`
	Flags                           string   `json:"flags,omitempty"`
	FuzzyTranspositions             bool     `json:"fuzzy_transpositions,omitempty"`
	FuzzyMaxExpansions              int64    `json:"fuzzy_max_expansions,omitempty"`
	FuzzyPrefixLength               int64    `json:"fuzzy_prefix_length,omitempty"`
	MinimumShouldMatch              int64    `json:"minimum_should_match,omitempty"`
	DefaultOperator                 string   `json:"default_operator,omitempty"`
	Analyzer                        string   `json:"analyzer,omitempty"`
	Lenient                         bool     `json:"lenient,omitempty"`
	QuoteFieldSuffix                string   `json:"quote_field_suffix,omitempty"`
	AnalyzeWildcard                 bool     `json:"analyze_wildcard,omitempty"`
	AutoGenerateSynonymsPhraseQuery bool     `json:"auto_generate_synonyms_phrase_query,omitempty"`
}
