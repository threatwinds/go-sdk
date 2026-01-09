package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// QueryType represents the type of query being performed
type QueryType int

const (
	QueryTypeTerm QueryType = iota
	QueryTypeTerms
	QueryTypeMatch
	QueryTypeMatchPhrase
	QueryTypeRange
	QueryTypeSort
	QueryTypeAggregation
	QueryTypeExists
	QueryTypeWildcard
	QueryTypeFuzzy             // Works like Match - needs text fields
	QueryTypeRegexp            // Works on keyword/text, prefers keyword
	QueryTypeMatchPhrasePrefix // Works like Match - needs text fields
	QueryTypePrefix            // Works on keyword fields
)

// ConflictStrategy determines how to handle field type conflicts across indices
type ConflictStrategy int

const (
	// MostCommon uses the most common type across indices
	MostCommon ConflictStrategy = iota
	// MostPermissive uses the most permissive type (text > keyword, long > integer)
	MostPermissive
	// Strict returns error on any conflict
	Strict
	// MostRecent uses type from the most recent index (by name sort)
	MostRecent
)

// FieldInfo contains information about a field's type and characteristics
type FieldInfo struct {
	BaseField     string              // e.g., "type", "attributes.file"
	Type          string              // e.g., "text", "keyword", "integer"
	Fields        map[string]string   // sub-fields: {"keyword": "keyword", "raw": "keyword"}
	AllowsMatch   bool                // true for text fields
	AllowsTerm    bool                // true for keyword/numeric/date/ip
	SourceIndices []string            // indices where field exists
	HasConflict   bool                // true if type differs across indices
	ConflictTypes map[string][]string // type -> [indices with that type]
}

// MergedMapping holds merged field definitions from multiple indices
type MergedMapping struct {
	IndexPattern  string
	Indices       []string
	Fields        map[string]FieldInfo
	ConflictCount int
	FetchedAt     time.Time
}

// MappingCache caches merged mappings with LRU eviction
type MappingCache struct {
	cache    *lru.Cache[string, *MergedMapping]
	ttl      time.Duration
	strategy ConflictStrategy
	strict   bool
	mu       sync.RWMutex
}

// FieldMapper manages mapping cache and field resolution
type FieldMapper struct {
	cache *MappingCache
}

// MapperOption configures the FieldMapper
type MapperOption func(*MappingCache)

// WithCacheTTL sets the cache TTL duration
func WithCacheTTL(ttl time.Duration) MapperOption {
	return func(c *MappingCache) {
		c.ttl = ttl
	}
}

// WithMaxCacheSize sets the maximum cache size
func WithMaxCacheSize(size int) MapperOption {
	return func(c *MappingCache) {
		cache, _ := lru.New[string, *MergedMapping](size)
		c.cache = cache
	}
}

// WithConflictStrategy sets how to handle field type conflicts
func WithConflictStrategy(strategy ConflictStrategy) MapperOption {
	return func(c *MappingCache) {
		c.strategy = strategy
	}
}

// WithStrictMode enables strict mode (error on unknown fields)
func WithStrictMode(strict bool) MapperOption {
	return func(c *MappingCache) {
		c.strict = strict
	}
}

// NewFieldMapper creates a new FieldMapper with the given options
func NewFieldMapper(opts ...MapperOption) *FieldMapper {
	cache, _ := lru.New[string, *MergedMapping](50) // default size
	mc := &MappingCache{
		cache:    cache,
		ttl:      5 * time.Minute,
		strategy: MostCommon,
		strict:   false,
	}

	for _, opt := range opts {
		opt(mc)
	}

	return &FieldMapper{cache: mc}
}

// GetMergedMapping fetches and merges mappings for all indices matching the pattern
func (f *FieldMapper) GetMergedMapping(ctx context.Context, indexPattern string) (*MergedMapping, error) {
	return f.cache.GetOrFetch(ctx, indexPattern)
}

// Invalidate removes a pattern from the cache
func (f *FieldMapper) Invalidate(pattern string) {
	f.cache.Invalidate(pattern)
}

// Clear removes all cached mappings
func (f *FieldMapper) Clear() {
	f.cache.Clear()
}

// GetOrFetch retrieves from cache or fetches and caches
func (c *MappingCache) GetOrFetch(ctx context.Context, pattern string) (*MergedMapping, error) {
	c.mu.RLock()
	if cached, ok := c.cache.Get(pattern); ok {
		// Check TTL
		if time.Since(cached.FetchedAt) < c.ttl {
			c.mu.RUnlock()
			return cached, nil
		}
	}
	c.mu.RUnlock()

	// Fetch and merge
	merged, err := c.fetchAndMerge(ctx, pattern)
	if err != nil {
		return nil, err
	}

	// Cache result
	c.mu.Lock()
	c.cache.Add(pattern, merged)
	c.mu.Unlock()

	return merged, nil
}

// Invalidate removes the pattern from cache
func (c *MappingCache) Invalidate(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Remove(pattern)
}

// Clear removes all cached mappings
func (c *MappingCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Purge()
}

// fetchAndMerge fetches mappings for the pattern and merges them
func (c *MappingCache) fetchAndMerge(ctx context.Context, pattern string) (*MergedMapping, error) {
	// Check if apiClient is initialized
	if apiClient == nil {
		return nil, fmt.Errorf("opensearch client not initialized, call Connect() first")
	}

	// Fetch mappings from OpenSearch
	req := &opensearchapi.MappingGetReq{
		Indices: []string{pattern},
	}

	resp, err := apiClient.Indices.Mapping.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mappings: %w", err)
	}

	// Extract indices from response
	indices := make([]string, 0, len(resp.Indices))
	for indexName := range resp.Indices {
		indices = append(indices, indexName)
	}

	if len(indices) == 0 {
		return &MergedMapping{
			IndexPattern: pattern,
			Indices:      []string{},
			Fields:       make(map[string]FieldInfo),
			FetchedAt:    time.Now(),
		}, nil
	}

	// Merge mappings
	merged := &MergedMapping{
		IndexPattern: pattern,
		Indices:      indices,
		Fields:       make(map[string]FieldInfo),
		FetchedAt:    time.Now(),
	}

	// Process each index - resp.Indices has Mappings as json.RawMessage
	for indexName, indexData := range resp.Indices {
		// Parse the Mappings json.RawMessage into a map
		var mappingsData map[string]interface{}
		if err := json.Unmarshal(indexData.Mappings, &mappingsData); err != nil {
			continue
		}

		properties, ok := mappingsData["properties"].(map[string]interface{})
		if !ok {
			continue
		}

		// Extract fields recursively
		c.extractFields(properties, "", indexName, merged)
	}

	// Resolve conflicts
	merged.ConflictCount = c.resolveConflicts(merged)

	return merged, nil
}

// extractFields recursively extracts field definitions
func (c *MappingCache) extractFields(properties map[string]interface{}, prefix string, indexName string, merged *MergedMapping) {
	for fieldName, fieldData := range properties {
		fullFieldName := fieldName
		if prefix != "" {
			fullFieldName = prefix + "." + fieldName
		}

		fieldMap, ok := fieldData.(map[string]interface{})
		if !ok {
			continue
		}

		fieldType, _ := fieldMap["type"].(string)

		// Get or create FieldInfo
		info, exists := merged.Fields[fullFieldName]
		if !exists {
			info = FieldInfo{
				BaseField:     fullFieldName,
				Fields:        make(map[string]string),
				SourceIndices: []string{},
				ConflictTypes: make(map[string][]string),
			}
		}

		// Add the index to the source list
		if !contains(info.SourceIndices, indexName) {
			info.SourceIndices = append(info.SourceIndices, indexName)
		}

		// Handle type
		if fieldType != "" {
			// Track type per index for conflict detection
			info.ConflictTypes[fieldType] = append(info.ConflictTypes[fieldType], indexName)

			// Set a primary type if not set
			if info.Type == "" {
				info.Type = fieldType
				info.AllowsMatch = isTextType(fieldType)
				info.AllowsTerm = isTermType(fieldType)
			} else if info.Type != fieldType {
				// Type conflict detected
				info.HasConflict = true
			}
		}

		// Extract sub-fields (multi-fields)
		if fields, ok := fieldMap["fields"].(map[string]interface{}); ok {
			for subFieldName, subFieldData := range fields {
				subFieldMap, ok := subFieldData.(map[string]interface{})
				if !ok {
					continue
				}
				subFieldType, _ := subFieldMap["type"].(string)
				if subFieldType != "" {
					info.Fields[subFieldName] = subFieldType
				}
			}
		}

		// Handle nested properties
		if nestedProps, ok := fieldMap["properties"].(map[string]interface{}); ok {
			c.extractFields(nestedProps, fullFieldName, indexName, merged)
		}

		merged.Fields[fullFieldName] = info
	}
}

// resolveConflicts resolves type conflicts based on strategy
func (c *MappingCache) resolveConflicts(merged *MergedMapping) int {
	conflictCount := 0

	for fieldName, info := range merged.Fields {
		if !info.HasConflict {
			continue
		}

		conflictCount++

		switch c.strategy {
		case Strict:
			// Caller will handle conflicts
			continue

		case MostCommon:
			// Use the most common type
			maxCount := 0
			mostCommonType := info.Type
			for typ, indices := range info.ConflictTypes {
				if len(indices) > maxCount {
					maxCount = len(indices)
					mostCommonType = typ
				}
			}
			info.Type = mostCommonType
			info.AllowsMatch = isTextType(mostCommonType)
			info.AllowsTerm = isTermType(mostCommonType)

		case MostPermissive:
			// Prefer text over keyword, long over integer
			types := make([]string, 0, len(info.ConflictTypes))
			for typ := range info.ConflictTypes {
				types = append(types, typ)
			}
			mostPermissive := getMostPermissiveType(types)
			info.Type = mostPermissive
			info.AllowsMatch = isTextType(mostPermissive)
			info.AllowsTerm = isTermType(mostPermissive)

		case MostRecent:
			// Use type from the most recent index (last in the sorted list)
			// Assume indices are already sorted by name
			if len(info.SourceIndices) > 0 {
				latestIndex := info.SourceIndices[len(info.SourceIndices)-1]
				for typ, indices := range info.ConflictTypes {
					if contains(indices, latestIndex) {
						info.Type = typ
						info.AllowsMatch = isTextType(typ)
						info.AllowsTerm = isTermType(typ)
						break
					}
				}
			}
		}

		merged.Fields[fieldName] = info
	}

	return conflictCount
}

// GetFieldInfo retrieves field info from merged mapping
func (m *MergedMapping) GetFieldInfo(field string) (FieldInfo, bool) {
	info, ok := m.Fields[field]
	return info, ok
}

// HasConflicts returns true if any fields have type conflicts
func (m *MergedMapping) HasConflicts() bool {
	return m.ConflictCount > 0
}

// GetConflicts returns a list of fields with type conflicts
func (m *MergedMapping) GetConflicts() []FieldInfo {
	conflicts := make([]FieldInfo, 0, m.ConflictCount)
	for _, info := range m.Fields {
		if info.HasConflict {
			conflicts = append(conflicts, info)
		}
	}
	return conflicts
}

// ResolveFieldName returns the correct field name for the given query type
func (m *MergedMapping) ResolveFieldName(field string, queryType QueryType) (string, error) {
	info, ok := m.Fields[field]
	if !ok {
		// Field not in mapping - return as-is (might be dynamic field)
		return field, nil
	}

	switch queryType {
	case QueryTypeTerm, QueryTypeTerms, QueryTypeRange, QueryTypeSort, QueryTypeAggregation, QueryTypePrefix:
		// These query types need keyword/exact match fields
		if info.Type == "text" {
			// Check for .keyword sub-field
			if _, hasKeyword := info.Fields["keyword"]; hasKeyword {
				return field + ".keyword", nil
			}
			// No keyword sub-field available
			return "", fmt.Errorf("field '%s' is text type with no .keyword sub-field, cannot use with %s query", field, queryTypeString(queryType))
		}
		// For keyword, numeric, date, ip, etc. - use as-is
		return field, nil

	case QueryTypeMatch, QueryTypeMatchPhrase, QueryTypeFuzzy, QueryTypeMatchPhrasePrefix:
		// These query types need text fields
		if info.Type == "text" {
			return field, nil
		}
		// For non-text fields, we could auto-convert to Term query
		// Return error for now, let builder handle conversion
		return "", fmt.Errorf("field '%s' is %s type, cannot use with %s query (use Term query instead)", field, info.Type, queryTypeString(queryType))

	case QueryTypeRegexp:
		// Regexp works best on keyword fields but can work on text
		if info.Type == "text" {
			// Check for .keyword sub-field - prefer it for regexp
			if _, hasKeyword := info.Fields["keyword"]; hasKeyword {
				return field + ".keyword", nil
			}
			// Fall through to text field (slower but works)
		}
		return field, nil

	case QueryTypeExists, QueryTypeWildcard:
		// These work with any field type
		return field, nil

	default:
		return field, nil
	}
}

// Helper functions

// isTextType returns true for text field types (delegates to field type registry)
func isTextType(typ string) bool {
	return IsTextType(typ)
}

// isTermType returns true for term-compatible field types (delegates to field type registry)
func isTermType(typ string) bool {
	return IsTermType(typ)
}

// getMostPermissiveType returns the most permissive type (delegates to field type registry)
func getMostPermissiveType(types []string) string {
	return GetMostPermissiveType(types)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func queryTypeString(qt QueryType) string {
	switch qt {
	case QueryTypeTerm:
		return "Term"
	case QueryTypeTerms:
		return "Terms"
	case QueryTypeMatch:
		return "Match"
	case QueryTypeMatchPhrase:
		return "MatchPhrase"
	case QueryTypeRange:
		return "Range"
	case QueryTypeSort:
		return "Sort"
	case QueryTypeAggregation:
		return "Aggregation"
	case QueryTypeExists:
		return "Exists"
	case QueryTypeWildcard:
		return "Wildcard"
	case QueryTypeFuzzy:
		return "Fuzzy"
	case QueryTypeRegexp:
		return "Regexp"
	case QueryTypeMatchPhrasePrefix:
		return "MatchPhrasePrefix"
	case QueryTypePrefix:
		return "Prefix"
	default:
		return "Unknown"
	}
}

// MappingBuilder provides a fluent API for adding fields to existing indices
type MappingBuilder struct {
	ctx        context.Context
	index      string
	properties map[string]MappingProperty
	errors     []error
}

// NewMappingBuilder creates a new mapping builder for the specified index
func NewMappingBuilder(ctx context.Context, index string) *MappingBuilder {
	return &MappingBuilder{
		ctx:        ctx,
		index:      index,
		properties: make(map[string]MappingProperty),
		errors:     []error{},
	}
}

// AddField adds a field with the specified type
func (b *MappingBuilder) AddField(name string, fieldType string) *MappingBuilder {
	b.properties[name] = MappingProperty{
		Type: fieldType,
	}
	return b
}

// AddTextField adds a text field with an optional analyzer
func (b *MappingBuilder) AddTextField(name string, analyzer string) *MappingBuilder {
	prop := MappingProperty{
		Type: "text",
	}
	if analyzer != "" {
		prop.Analyzer = analyzer
	}
	// Add keyword to subfield for exact matching
	prop.Fields = map[string]MappingProperty{
		"keyword": {
			Type: "keyword",
		},
	}
	b.properties[name] = prop
	return b
}

// AddKeywordField adds a keyword field
func (b *MappingBuilder) AddKeywordField(name string) *MappingBuilder {
	b.properties[name] = MappingProperty{
		Type: "keyword",
	}
	return b
}

// AddDateField adds a date field with optional format
func (b *MappingBuilder) AddDateField(name string, format string) *MappingBuilder {
	prop := MappingProperty{
		Type: "date",
	}
	if format != "" {
		prop.Format = format
	}
	b.properties[name] = prop
	return b
}

// AddIntegerField adds an integer field
func (b *MappingBuilder) AddIntegerField(name string) *MappingBuilder {
	b.properties[name] = MappingProperty{
		Type: "integer",
	}
	return b
}

// AddLongField adds a long field
func (b *MappingBuilder) AddLongField(name string) *MappingBuilder {
	b.properties[name] = MappingProperty{
		Type: "long",
	}
	return b
}

// AddFloatField adds a float field
func (b *MappingBuilder) AddFloatField(name string) *MappingBuilder {
	b.properties[name] = MappingProperty{
		Type: "float",
	}
	return b
}

// AddDoubleField adds a double field
func (b *MappingBuilder) AddDoubleField(name string) *MappingBuilder {
	b.properties[name] = MappingProperty{
		Type: "double",
	}
	return b
}

// AddBooleanField adds a boolean field
func (b *MappingBuilder) AddBooleanField(name string) *MappingBuilder {
	b.properties[name] = MappingProperty{
		Type: "boolean",
	}
	return b
}

// AddKNNVectorField adds a k-NN vector field for similarity search
func (b *MappingBuilder) AddKNNVectorField(name string, dimension int, method *KNNMethod) *MappingBuilder {
	if dimension <= 0 {
		b.errors = append(b.errors, fmt.Errorf("knn vector dimension must be positive"))
		return b
	}

	prop := MappingProperty{
		Type:      "knn_vector",
		Dimension: dimension,
	}
	if method != nil {
		prop.Method = method
	}
	b.properties[name] = prop
	return b
}

// AddNestedField adds a nested field with properties
func (b *MappingBuilder) AddNestedField(name string, properties map[string]MappingProperty) *MappingBuilder {
	b.properties[name] = MappingProperty{
		Type:       "nested",
		Properties: properties,
	}
	return b
}

// AddObjectField adds an object field with properties
func (b *MappingBuilder) AddObjectField(name string, properties map[string]MappingProperty) *MappingBuilder {
	b.properties[name] = MappingProperty{
		Type:       "object",
		Properties: properties,
	}
	return b
}

// AddProperty adds a custom property definition
func (b *MappingBuilder) AddProperty(name string, property MappingProperty) *MappingBuilder {
	b.properties[name] = property
	return b
}

// Build returns the mapping update request body
func (b *MappingBuilder) Build() (map[string]interface{}, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("mapping builder has %d errors: %v", len(b.errors), b.errors)
	}

	if len(b.properties) == 0 {
		return nil, fmt.Errorf("no properties defined")
	}

	// Convert MappingProperty to map[string]interface{} for JSON marshaling
	props := make(map[string]interface{})
	for name, prop := range b.properties {
		props[name] = b.propertyToMap(prop)
	}

	return map[string]interface{}{
		"properties": props,
	}, nil
}

// BuildWithErrors returns the mapping and any accumulated errors
func (b *MappingBuilder) BuildWithErrors() (map[string]interface{}, []error) {
	if len(b.properties) == 0 {
		return nil, append(b.errors, fmt.Errorf("no properties defined"))
	}

	props := make(map[string]interface{})
	for name, prop := range b.properties {
		props[name] = b.propertyToMap(prop)
	}

	return map[string]interface{}{
		"properties": props,
	}, b.errors
}

// Ensure applies the mapping to the index (idempotent - adds new fields only)
func (b *MappingBuilder) Ensure() error {
	if b.index == "" {
		return fmt.Errorf("index name is required")
	}

	body, err := b.Build()
	if err != nil {
		return err
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	req := opensearchapi.MappingPutReq{
		Indices: []string{b.index},
		Body:    bytes.NewReader(bodyJSON),
	}

	_, err = apiClient.Indices.Mapping.Put(b.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update mapping: %w", err)
	}

	return nil
}

// propertyToMap converts MappingProperty to a map for JSON marshaling
func (b *MappingBuilder) propertyToMap(prop MappingProperty) map[string]interface{} {
	result := make(map[string]interface{})

	if prop.Type != "" {
		result["type"] = prop.Type
	}
	if prop.Analyzer != "" {
		result["analyzer"] = prop.Analyzer
	}
	if prop.Format != "" {
		result["format"] = prop.Format
	}
	if prop.Index != nil {
		result["index"] = *prop.Index
	}
	if prop.Store != nil {
		result["store"] = *prop.Store
	}
	if prop.Dimension > 0 {
		result["dimension"] = prop.Dimension
	}
	if prop.Method != nil {
		methodMap := map[string]interface{}{
			"name":       prop.Method.Name,
			"space_type": prop.Method.SpaceType,
			"engine":     prop.Method.Engine,
		}
		if prop.Method.Parameters != nil {
			methodMap["parameters"] = prop.Method.Parameters
		}
		result["method"] = methodMap
	}
	if len(prop.Fields) > 0 {
		fields := make(map[string]interface{})
		for name, field := range prop.Fields {
			fields[name] = b.propertyToMap(field)
		}
		result["fields"] = fields
	}
	if len(prop.Properties) > 0 {
		properties := make(map[string]interface{})
		for name, p := range prop.Properties {
			properties[name] = b.propertyToMap(p)
		}
		result["properties"] = properties
	}

	return result
}
