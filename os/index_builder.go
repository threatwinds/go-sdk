package os

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/threatwinds/go-sdk/catcher"
)

// IndexSettings represents OpenSearch index settings
type IndexSettings struct {
	NumberOfShards       *int                   `json:"number_of_shards,omitempty"`
	NumberOfReplicas     *int                   `json:"number_of_replicas,omitempty"`
	RefreshInterval      string                 `json:"refresh_interval,omitempty"`
	MaxResultWindow      *int                   `json:"max_result_window,omitempty"`
	KNN                  *bool                  `json:"knn,omitempty"`
	KNNAlgoParamEFSearch *int                   `json:"knn.algo_param.ef_search,omitempty"`
	Analysis             *AnalysisSettings      `json:"analysis,omitempty"`
	Custom               map[string]interface{} `json:"-"`
}

// AnalysisSettings represents custom analyzers, tokenizers, and filters
type AnalysisSettings struct {
	Analyzer   map[string]interface{} `json:"analyzer,omitempty"`
	Tokenizer  map[string]interface{} `json:"tokenizer,omitempty"`
	Filter     map[string]interface{} `json:"filter,omitempty"`
	CharFilter map[string]interface{} `json:"char_filter,omitempty"`
	Normalizer map[string]interface{} `json:"normalizer,omitempty"`
}

// KNNMethod configures the k-NN algorithm for vector fields
type KNNMethod struct {
	Name       string                 `json:"name"`                 // hnsw, ivf, etc.
	SpaceType  string                 `json:"space_type"`           // cosinesimil, l2, innerproduct
	Engine     string                 `json:"engine"`               // nmslib, faiss, lucene
	Parameters map[string]interface{} `json:"parameters,omitempty"` // ef_construction, m, etc.
}

// MappingProperty represents a field mapping property
type MappingProperty struct {
	Type       string                     `json:"type"`
	Index      *bool                      `json:"index,omitempty"`
	Store      *bool                      `json:"store,omitempty"`
	Analyzer   string                     `json:"analyzer,omitempty"`
	Format     string                     `json:"format,omitempty"`
	Dimension  int                        `json:"dimension,omitempty"`
	Method     *KNNMethod                 `json:"method,omitempty"`
	Fields     map[string]MappingProperty `json:"fields,omitempty"`
	Properties map[string]MappingProperty `json:"properties,omitempty"`
}

// IndexCreateRequest represents the request body for creating an index
type IndexCreateRequest struct {
	Settings map[string]interface{} `json:"settings,omitempty"`
	Mappings map[string]interface{} `json:"mappings,omitempty"`
	Aliases  map[string]AliasConfig `json:"aliases,omitempty"`
}

// IndexBuilder provides a fluent API for creating indices
type IndexBuilder struct {
	ctx         context.Context
	name        string
	settings    map[string]interface{}
	mappings    map[string]interface{}
	properties  map[string]interface{}
	aliases     map[string]AliasConfig
	dynamic     string
	errors      []error
	processName string
}

// NewIndexBuilder creates a new IndexBuilder
func NewIndexBuilder(ctx context.Context, name string, processName string) *IndexBuilder {
	return &IndexBuilder{
		ctx:         ctx,
		name:        name,
		settings:    make(map[string]interface{}),
		mappings:    make(map[string]interface{}),
		properties:  make(map[string]interface{}),
		aliases:     make(map[string]AliasConfig),
		errors:      []error{},
		processName: processName,
	}
}

// --- Settings Methods ---

// Shards sets the number of primary shards
func (b *IndexBuilder) Shards(n int) *IndexBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["number_of_shards"] = n
	return b
}

// Replicas sets the number of replica shards
func (b *IndexBuilder) Replicas(n int) *IndexBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["number_of_replicas"] = n
	return b
}

// RefreshInterval sets the refresh interval
func (b *IndexBuilder) RefreshInterval(interval string) *IndexBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["refresh_interval"] = interval
	return b
}

// MaxResultWindow sets the maximum result window size
func (b *IndexBuilder) MaxResultWindow(size int) *IndexBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["max_result_window"] = size
	return b
}

// EnableKNN enables k-NN functionality for the index
func (b *IndexBuilder) EnableKNN() *IndexBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["knn"] = true
	return b
}

// KNNAlgoParamEFSearch sets the ef_search parameter for k-NN
func (b *IndexBuilder) KNNAlgoParamEFSearch(ef int) *IndexBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["knn.algo_param.ef_search"] = ef
	return b
}

// Analysis sets custom analysis settings
func (b *IndexBuilder) Analysis(analysis *AnalysisSettings) *IndexBuilder {
	if analysis != nil {
		if b.settings["index"] == nil {
			b.settings["index"] = make(map[string]interface{})
		}
		analysisMap := make(map[string]interface{})
		if analysis.Analyzer != nil {
			analysisMap["analyzer"] = analysis.Analyzer
		}
		if analysis.Tokenizer != nil {
			analysisMap["tokenizer"] = analysis.Tokenizer
		}
		if analysis.Filter != nil {
			analysisMap["filter"] = analysis.Filter
		}
		if analysis.CharFilter != nil {
			analysisMap["char_filter"] = analysis.CharFilter
		}
		if analysis.Normalizer != nil {
			analysisMap["normalizer"] = analysis.Normalizer
		}
		b.settings["index"].(map[string]interface{})["analysis"] = analysisMap
	}
	return b
}

// CustomSetting sets a custom index setting
func (b *IndexBuilder) CustomSetting(key string, value interface{}) *IndexBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})[key] = value
	return b
}

// --- Mapping Methods ---

// Mapping sets the entire mapping properties
func (b *IndexBuilder) Mapping(properties map[string]interface{}) *IndexBuilder {
	for k, v := range properties {
		b.properties[k] = v
	}
	return b
}

// DynamicMapping sets the dynamic mapping behavior
func (b *IndexBuilder) DynamicMapping(dynamic string) *IndexBuilder {
	b.dynamic = dynamic
	return b
}

// AddField adds a field with a specific type
func (b *IndexBuilder) AddField(name string, fieldType string) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type": fieldType,
	}
	return b
}

// AddTextField adds a text field with optional analyzer
func (b *IndexBuilder) AddTextField(name string, analyzer string) *IndexBuilder {
	prop := map[string]interface{}{
		"type": "text",
	}
	if analyzer != "" {
		prop["analyzer"] = analyzer
	}
	// Add keyword sub-field for exact matching
	prop["fields"] = map[string]interface{}{
		"keyword": map[string]interface{}{
			"type":         "keyword",
			"ignore_above": 256,
		},
	}
	b.properties[name] = prop
	return b
}

// AddKeywordField adds a keyword field
func (b *IndexBuilder) AddKeywordField(name string) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type": "keyword",
	}
	return b
}

// AddDateField adds a date field with optional format
func (b *IndexBuilder) AddDateField(name string, format string) *IndexBuilder {
	prop := map[string]interface{}{
		"type": "date",
	}
	if format != "" {
		prop["format"] = format
	}
	b.properties[name] = prop
	return b
}

// AddIntegerField adds an integer field
func (b *IndexBuilder) AddIntegerField(name string) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type": "integer",
	}
	return b
}

// AddLongField adds a long field
func (b *IndexBuilder) AddLongField(name string) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type": "long",
	}
	return b
}

// AddFloatField adds a float field
func (b *IndexBuilder) AddFloatField(name string) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type": "float",
	}
	return b
}

// AddDoubleField adds a double field
func (b *IndexBuilder) AddDoubleField(name string) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type": "double",
	}
	return b
}

// AddBooleanField adds a boolean field
func (b *IndexBuilder) AddBooleanField(name string) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type": "boolean",
	}
	return b
}

// AddObjectField adds an object field with nested properties
func (b *IndexBuilder) AddObjectField(name string, properties map[string]interface{}) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	return b
}

// AddNestedField adds a nested field with properties
func (b *IndexBuilder) AddNestedField(name string, properties map[string]interface{}) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type":       "nested",
		"properties": properties,
	}
	return b
}

// AddKNNVectorField adds a k-NN vector field
func (b *IndexBuilder) AddKNNVectorField(name string, dimension int, method *KNNMethod) *IndexBuilder {
	prop := map[string]interface{}{
		"type":      "knn_vector",
		"dimension": dimension,
	}
	if method != nil {
		methodMap := map[string]interface{}{
			"name":       method.Name,
			"space_type": method.SpaceType,
			"engine":     method.Engine,
		}
		if method.Parameters != nil {
			methodMap["parameters"] = method.Parameters
		}
		prop["method"] = methodMap
	}
	b.properties[name] = prop
	return b
}

// AddFlatObjectField adds a flat_object field (for dynamic JSON)
func (b *IndexBuilder) AddFlatObjectField(name string) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type": "flat_object",
	}
	return b
}

// AddGeoPointField adds a geo_point field
func (b *IndexBuilder) AddGeoPointField(name string) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type": "geo_point",
	}
	return b
}

// AddIPField adds an IP address field
func (b *IndexBuilder) AddIPField(name string) *IndexBuilder {
	b.properties[name] = map[string]interface{}{
		"type": "ip",
	}
	return b
}

// AddProperty adds a custom property definition
func (b *IndexBuilder) AddProperty(name string, property MappingProperty) *IndexBuilder {
	propMap := map[string]interface{}{
		"type": property.Type,
	}
	if property.Index != nil {
		propMap["index"] = *property.Index
	}
	if property.Store != nil {
		propMap["store"] = *property.Store
	}
	if property.Analyzer != "" {
		propMap["analyzer"] = property.Analyzer
	}
	if property.Format != "" {
		propMap["format"] = property.Format
	}
	if property.Dimension > 0 {
		propMap["dimension"] = property.Dimension
	}
	if property.Method != nil {
		methodMap := map[string]interface{}{
			"name":       property.Method.Name,
			"space_type": property.Method.SpaceType,
			"engine":     property.Method.Engine,
		}
		if property.Method.Parameters != nil {
			methodMap["parameters"] = property.Method.Parameters
		}
		propMap["method"] = methodMap
	}
	if len(property.Fields) > 0 {
		fields := make(map[string]interface{})
		for k, v := range property.Fields {
			fields[k] = propertyToMap(v)
		}
		propMap["fields"] = fields
	}
	if len(property.Properties) > 0 {
		props := make(map[string]interface{})
		for k, v := range property.Properties {
			props[k] = propertyToMap(v)
		}
		propMap["properties"] = props
	}
	b.properties[name] = propMap
	return b
}

// --- Alias Methods ---

// Alias adds a simple alias
func (b *IndexBuilder) Alias(alias string) *IndexBuilder {
	b.aliases[alias] = AliasConfig{}
	return b
}

// AliasWithConfig adds an alias with configuration
func (b *IndexBuilder) AliasWithConfig(alias string, config AliasConfig) *IndexBuilder {
	b.aliases[alias] = config
	return b
}

// WriteAlias adds an alias marked as the write index
func (b *IndexBuilder) WriteAlias(alias string) *IndexBuilder {
	isWrite := true
	b.aliases[alias] = AliasConfig{
		IsWriteIndex: &isWrite,
	}
	return b
}

// --- Terminal Methods ---

// Build returns the IndexCreateRequest without executing
func (b *IndexBuilder) Build() (IndexCreateRequest, error) {
	if len(b.errors) > 0 {
		return IndexCreateRequest{}, catcher.Error("failed to build index request", errors.New("please see the errors list in the arguments"), map[string]any{
			"errors": b.errors,
			"index":  b.name,
		})
	}

	request := IndexCreateRequest{}

	// Build settings
	if len(b.settings) > 0 {
		request.Settings = b.settings
	}

	// Build mappings
	if len(b.properties) > 0 || b.dynamic != "" {
		request.Mappings = make(map[string]interface{})
		if b.dynamic != "" {
			request.Mappings["dynamic"] = b.dynamic
		}
		if len(b.properties) > 0 {
			request.Mappings["properties"] = b.properties
		}
	}

	// Build aliases
	if len(b.aliases) > 0 {
		request.Aliases = b.aliases
	}

	return request, nil
}

// BuildWithErrors returns the IndexCreateRequest and any accumulated errors
func (b *IndexBuilder) BuildWithErrors() (IndexCreateRequest, []error) {
	request, _ := b.Build()
	if len(b.errors) == 0 && b.name == "" {
		return request, append(b.errors, catcher.Error("failed to build index request", errors.New("index name is required"), nil))
	}
	return request, b.errors
}

// Ensure creates the index if it doesn't exist (idempotent)
func (b *IndexBuilder) Ensure() error {
	if b.name == "" {
		return catcher.Error("failed to ensure index", errors.New("index name is required"), nil)
	}

	// Check if index already exists
	exists, err := IndexExists(b.ctx, b.name)
	if err != nil {
		return catcher.Error("failed to ensure index", fmt.Errorf("failed to check if index exists: %w", err), map[string]any{
			"index": b.name,
		})
	}

	if exists {
		// Index already exists - idempotent success
		return nil
	}

	// Build the request
	request, err := b.Build()
	if err != nil {
		return err
	}

	// Marshal request body
	body, err := json.Marshal(request)
	if err != nil {
		return catcher.Error("failed to ensure index", fmt.Errorf("failed to marshal index request: %w", err), map[string]any{
			"index": b.name,
		})
	}

	// Create the index
	req := opensearchapi.IndicesCreateReq{
		Index: b.name,
		Body:  strings.NewReader(string(body)),
	}

	resp, err := apiClient.Indices.Create(b.ctx, req)
	if err != nil {
		// Check if it's a "resource already exists" error - treat as idempotent success
		errStr := err.Error()
		if strings.Contains(errStr, "resource_already_exists_exception") ||
			strings.Contains(errStr, "already exists") {
			return nil
		}
		return catcher.Error("failed to ensure index", fmt.Errorf("failed to create index: %w", err), map[string]any{
			"index": b.name,
		})
	}

	if !resp.Acknowledged {
		return catcher.Error("failed to ensure index", errors.New("index creation not acknowledged"), map[string]any{
			"index": b.name,
		})
	}

	return nil
}

// propertyToMap converts a MappingProperty to a map
func propertyToMap(p MappingProperty) map[string]interface{} {
	m := map[string]interface{}{
		"type": p.Type,
	}
	if p.Index != nil {
		m["index"] = *p.Index
	}
	if p.Store != nil {
		m["store"] = *p.Store
	}
	if p.Analyzer != "" {
		m["analyzer"] = p.Analyzer
	}
	if p.Format != "" {
		m["format"] = p.Format
	}
	if p.Dimension > 0 {
		m["dimension"] = p.Dimension
	}
	if p.Method != nil {
		methodMap := map[string]interface{}{
			"name":       p.Method.Name,
			"space_type": p.Method.SpaceType,
			"engine":     p.Method.Engine,
		}
		if p.Method.Parameters != nil {
			methodMap["parameters"] = p.Method.Parameters
		}
		m["method"] = methodMap
	}
	if len(p.Fields) > 0 {
		fields := make(map[string]interface{})
		for k, v := range p.Fields {
			fields[k] = propertyToMap(v)
		}
		m["fields"] = fields
	}
	if len(p.Properties) > 0 {
		props := make(map[string]interface{})
		for k, v := range p.Properties {
			props[k] = propertyToMap(v)
		}
		m["properties"] = props
	}
	return m
}
