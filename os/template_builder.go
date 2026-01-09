package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// IndexTemplate represents an index template definition
type IndexTemplate struct {
	Name          string                 `json:"-"`
	IndexPatterns []string               `json:"index_patterns"`
	Template      TemplateContent        `json:"template,omitempty"`
	Priority      *int                   `json:"priority,omitempty"`
	Version       *int64                 `json:"version,omitempty"`
	Meta          map[string]interface{} `json:"_meta,omitempty"`
	ComposedOf    []string               `json:"composed_of,omitempty"`
}

// TemplateContent represents the template content (settings, mappings, aliases)
type TemplateContent struct {
	Settings map[string]interface{}         `json:"settings,omitempty"`
	Mappings map[string]interface{}         `json:"mappings,omitempty"`
	Aliases  map[string]TemplateAliasConfig `json:"aliases,omitempty"`
}

// TemplateAliasConfig represents alias configuration in a template
type TemplateAliasConfig struct {
	Filter        map[string]interface{} `json:"filter,omitempty"`
	IndexRouting  string                 `json:"index_routing,omitempty"`
	SearchRouting string                 `json:"search_routing,omitempty"`
	Routing       string                 `json:"routing,omitempty"`
	IsWriteIndex  *bool                  `json:"is_write_index,omitempty"`
	IsHidden      *bool                  `json:"is_hidden,omitempty"`
}

// TemplateBuilder provides a fluent API for creating index templates
type TemplateBuilder struct {
	ctx        context.Context
	name       string
	patterns   []string
	settings   map[string]interface{}
	mappings   map[string]interface{}
	properties map[string]interface{}
	aliases    map[string]TemplateAliasConfig
	priority   *int
	version    *int64
	meta       map[string]interface{}
	composedOf []string
	dynamic    string
	errors     []error
}

// NewTemplateBuilder creates a new template builder
func NewTemplateBuilder(ctx context.Context, name string) *TemplateBuilder {
	return &TemplateBuilder{
		ctx:        ctx,
		name:       name,
		patterns:   []string{},
		settings:   make(map[string]interface{}),
		mappings:   make(map[string]interface{}),
		properties: make(map[string]interface{}),
		aliases:    make(map[string]TemplateAliasConfig),
		errors:     []error{},
	}
}

// IndexPatterns sets the index patterns this template applies to
func (b *TemplateBuilder) IndexPatterns(patterns ...string) *TemplateBuilder {
	b.patterns = patterns
	return b
}

// Priority sets the template priority (higher = more important)
func (b *TemplateBuilder) Priority(priority int) *TemplateBuilder {
	b.priority = &priority
	return b
}

// Version sets the template version
func (b *TemplateBuilder) Version(version int64) *TemplateBuilder {
	b.version = &version
	return b
}

// Meta sets template metadata
func (b *TemplateBuilder) Meta(meta map[string]interface{}) *TemplateBuilder {
	b.meta = meta
	return b
}

// ComposedOf sets the component templates to compose from
func (b *TemplateBuilder) ComposedOf(components ...string) *TemplateBuilder {
	b.composedOf = components
	return b
}

// Shards sets the number of primary shards
func (b *TemplateBuilder) Shards(n int) *TemplateBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["number_of_shards"] = n
	return b
}

// Replicas sets the number of replica shards
func (b *TemplateBuilder) Replicas(n int) *TemplateBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["number_of_replicas"] = n
	return b
}

// RefreshInterval sets the refresh interval
func (b *TemplateBuilder) RefreshInterval(interval string) *TemplateBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["refresh_interval"] = interval
	return b
}

// EnableKNN enables k-NN support for the template
func (b *TemplateBuilder) EnableKNN() *TemplateBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["knn"] = true
	return b
}

// KNNAlgoParamEFSearch sets the ef_search parameter for k-NN
func (b *TemplateBuilder) KNNAlgoParamEFSearch(ef int) *TemplateBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["knn.algo_param.ef_search"] = ef
	return b
}

// Settings sets custom index settings
func (b *TemplateBuilder) Settings(settings map[string]interface{}) *TemplateBuilder {
	for k, v := range settings {
		b.settings[k] = v
	}
	return b
}

// CustomSetting sets a custom setting
func (b *TemplateBuilder) CustomSetting(key string, value interface{}) *TemplateBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})[key] = value
	return b
}

// Mapping sets the full mappings object
func (b *TemplateBuilder) Mapping(properties map[string]interface{}) *TemplateBuilder {
	for k, v := range properties {
		b.properties[k] = v
	}
	return b
}

// DynamicMapping sets the dynamic mapping mode
func (b *TemplateBuilder) DynamicMapping(dynamic string) *TemplateBuilder {
	b.dynamic = dynamic
	return b
}

// AddField adds a field to the mapping
func (b *TemplateBuilder) AddField(name string, fieldType string) *TemplateBuilder {
	b.properties[name] = map[string]interface{}{
		"type": fieldType,
	}
	return b
}

// AddTextField adds a text field with optional analyzer
func (b *TemplateBuilder) AddTextField(name string, analyzer string) *TemplateBuilder {
	prop := map[string]interface{}{
		"type": "text",
		"fields": map[string]interface{}{
			"keyword": map[string]interface{}{
				"type": "keyword",
			},
		},
	}
	if analyzer != "" {
		prop["analyzer"] = analyzer
	}
	b.properties[name] = prop
	return b
}

// AddKeywordField adds a keyword field
func (b *TemplateBuilder) AddKeywordField(name string) *TemplateBuilder {
	b.properties[name] = map[string]interface{}{
		"type": "keyword",
	}
	return b
}

// AddDateField adds a date field with optional format
func (b *TemplateBuilder) AddDateField(name string, format string) *TemplateBuilder {
	prop := map[string]interface{}{
		"type": "date",
	}
	if format != "" {
		prop["format"] = format
	}
	b.properties[name] = prop
	return b
}

// AddKNNVectorField adds a k-NN vector field
func (b *TemplateBuilder) AddKNNVectorField(name string, dimension int, method *KNNMethod) *TemplateBuilder {
	if dimension <= 0 {
		b.errors = append(b.errors, fmt.Errorf("knn vector dimension must be positive"))
		return b
	}

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

// AddNestedField adds a nested field with properties
func (b *TemplateBuilder) AddNestedField(name string, properties map[string]interface{}) *TemplateBuilder {
	b.properties[name] = map[string]interface{}{
		"type":       "nested",
		"properties": properties,
	}
	return b
}

// Alias adds an alias to the template
func (b *TemplateBuilder) Alias(alias string) *TemplateBuilder {
	b.aliases[alias] = TemplateAliasConfig{}
	return b
}

// WriteAlias adds an alias as the write index
func (b *TemplateBuilder) WriteAlias(alias string) *TemplateBuilder {
	isWriteIndex := true
	b.aliases[alias] = TemplateAliasConfig{
		IsWriteIndex: &isWriteIndex,
	}
	return b
}

// AliasWithConfig adds an alias with full configuration
func (b *TemplateBuilder) AliasWithConfig(alias string, config TemplateAliasConfig) *TemplateBuilder {
	b.aliases[alias] = config
	return b
}

// Build returns the index template request body
func (b *TemplateBuilder) Build() (IndexTemplate, error) {
	if len(b.errors) > 0 {
		return IndexTemplate{}, fmt.Errorf("template builder has %d errors: %v", len(b.errors), b.errors)
	}

	if b.name == "" {
		return IndexTemplate{}, fmt.Errorf("template name is required")
	}

	if len(b.patterns) == 0 {
		return IndexTemplate{}, fmt.Errorf("at least one index pattern is required")
	}

	template := IndexTemplate{
		Name:          b.name,
		IndexPatterns: b.patterns,
		Priority:      b.priority,
		Version:       b.version,
		Meta:          b.meta,
		ComposedOf:    b.composedOf,
	}

	// Build template content
	if len(b.settings) > 0 {
		template.Template.Settings = b.settings
	}

	if len(b.properties) > 0 || b.dynamic != "" {
		template.Template.Mappings = make(map[string]interface{})
		if len(b.properties) > 0 {
			template.Template.Mappings["properties"] = b.properties
		}
		if b.dynamic != "" {
			template.Template.Mappings["dynamic"] = b.dynamic
		}
	}

	if len(b.aliases) > 0 {
		template.Template.Aliases = b.aliases
	}

	return template, nil
}

// BuildWithErrors returns the template and any accumulated errors
func (b *TemplateBuilder) BuildWithErrors() (IndexTemplate, []error) {
	if b.name == "" {
		b.errors = append(b.errors, fmt.Errorf("template name is required"))
	}
	if len(b.patterns) == 0 {
		b.errors = append(b.errors, fmt.Errorf("at least one index pattern is required"))
	}

	template := IndexTemplate{
		Name:          b.name,
		IndexPatterns: b.patterns,
		Priority:      b.priority,
		Version:       b.version,
		Meta:          b.meta,
		ComposedOf:    b.composedOf,
	}

	if len(b.settings) > 0 {
		template.Template.Settings = b.settings
	}

	if len(b.properties) > 0 || b.dynamic != "" {
		template.Template.Mappings = make(map[string]interface{})
		if len(b.properties) > 0 {
			template.Template.Mappings["properties"] = b.properties
		}
		if b.dynamic != "" {
			template.Template.Mappings["dynamic"] = b.dynamic
		}
	}

	if len(b.aliases) > 0 {
		template.Template.Aliases = b.aliases
	}

	return template, b.errors
}

// Ensure creates or updates the index template (idempotent)
func (b *TemplateBuilder) Ensure() error {
	template, err := b.Build()
	if err != nil {
		return err
	}

	// Prepare the request body (without Name field)
	body := map[string]interface{}{
		"index_patterns": template.IndexPatterns,
	}

	if template.Template.Settings != nil || template.Template.Mappings != nil || template.Template.Aliases != nil {
		templateContent := make(map[string]interface{})
		if template.Template.Settings != nil {
			templateContent["settings"] = template.Template.Settings
		}
		if template.Template.Mappings != nil {
			templateContent["mappings"] = template.Template.Mappings
		}
		if template.Template.Aliases != nil {
			templateContent["aliases"] = template.Template.Aliases
		}
		body["template"] = templateContent
	}

	if template.Priority != nil {
		body["priority"] = *template.Priority
	}
	if template.Version != nil {
		body["version"] = *template.Version
	}
	if template.Meta != nil {
		body["_meta"] = template.Meta
	}
	if len(template.ComposedOf) > 0 {
		body["composed_of"] = template.ComposedOf
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	// Use _index_template API for composable templates
	path := fmt.Sprintf("/_index_template/%s", b.name)
	req, err := http.NewRequestWithContext(b.ctx, "PUT", path, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to create/update template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create/update template: %s", string(bodyBytes))
	}

	return nil
}

// GetIndexTemplate retrieves an index template by name
func GetIndexTemplate(ctx context.Context, name string) (*IndexTemplate, error) {
	path := fmt.Sprintf("/_index_template/%s", name)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get template: %s", string(bodyBytes))
	}

	var result struct {
		IndexTemplates []struct {
			Name          string        `json:"name"`
			IndexTemplate IndexTemplate `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.IndexTemplates) == 0 {
		return nil, nil
	}

	template := result.IndexTemplates[0].IndexTemplate
	template.Name = result.IndexTemplates[0].Name
	return &template, nil
}

// DeleteIndexTemplate deletes an index template
func DeleteIndexTemplate(ctx context.Context, name string) error {
	path := fmt.Sprintf("/_index_template/%s", name)
	req, err := http.NewRequestWithContext(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete template: %s", string(bodyBytes))
	}

	return nil
}

// IndexTemplateExists checks if an index template exists
func IndexTemplateExists(ctx context.Context, name string) (bool, error) {
	path := fmt.Sprintf("/_index_template/%s", name)
	req, err := http.NewRequestWithContext(ctx, "HEAD", path, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return false, fmt.Errorf("failed to check template: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

// ListIndexTemplates lists all index templates matching a pattern
func ListIndexTemplates(ctx context.Context, pattern string) ([]IndexTemplate, error) {
	path := "/_index_template"
	if pattern != "" && pattern != "*" {
		path = fmt.Sprintf("/_index_template/%s", pattern)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return []IndexTemplate{}, nil
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list templates: %s", string(bodyBytes))
	}

	var result struct {
		IndexTemplates []struct {
			Name          string        `json:"name"`
			IndexTemplate IndexTemplate `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	templates := make([]IndexTemplate, len(result.IndexTemplates))
	for i, t := range result.IndexTemplates {
		templates[i] = t.IndexTemplate
		templates[i].Name = t.Name
	}

	return templates, nil
}

// ComponentTemplateBuilder provides a fluent API for creating component templates
type ComponentTemplateBuilder struct {
	ctx        context.Context
	name       string
	settings   map[string]interface{}
	mappings   map[string]interface{}
	properties map[string]interface{}
	aliases    map[string]TemplateAliasConfig
	version    *int64
	meta       map[string]interface{}
	dynamic    string
	errors     []error
}

// NewComponentTemplateBuilder creates a new component template builder
func NewComponentTemplateBuilder(ctx context.Context, name string) *ComponentTemplateBuilder {
	return &ComponentTemplateBuilder{
		ctx:        ctx,
		name:       name,
		settings:   make(map[string]interface{}),
		mappings:   make(map[string]interface{}),
		properties: make(map[string]interface{}),
		aliases:    make(map[string]TemplateAliasConfig),
		errors:     []error{},
	}
}

// Settings sets the component template settings
func (b *ComponentTemplateBuilder) Settings(settings map[string]interface{}) *ComponentTemplateBuilder {
	for k, v := range settings {
		b.settings[k] = v
	}
	return b
}

// Shards sets the number of primary shards
func (b *ComponentTemplateBuilder) Shards(n int) *ComponentTemplateBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["number_of_shards"] = n
	return b
}

// Replicas sets the number of replica shards
func (b *ComponentTemplateBuilder) Replicas(n int) *ComponentTemplateBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["number_of_replicas"] = n
	return b
}

// EnableKNN enables k-NN support
func (b *ComponentTemplateBuilder) EnableKNN() *ComponentTemplateBuilder {
	if b.settings["index"] == nil {
		b.settings["index"] = make(map[string]interface{})
	}
	b.settings["index"].(map[string]interface{})["knn"] = true
	return b
}

// Mapping sets the mappings
func (b *ComponentTemplateBuilder) Mapping(properties map[string]interface{}) *ComponentTemplateBuilder {
	for k, v := range properties {
		b.properties[k] = v
	}
	return b
}

// AddField adds a field to the mapping
func (b *ComponentTemplateBuilder) AddField(name string, fieldType string) *ComponentTemplateBuilder {
	b.properties[name] = map[string]interface{}{
		"type": fieldType,
	}
	return b
}

// Version sets the template version
func (b *ComponentTemplateBuilder) Version(version int64) *ComponentTemplateBuilder {
	b.version = &version
	return b
}

// Meta sets template metadata
func (b *ComponentTemplateBuilder) Meta(meta map[string]interface{}) *ComponentTemplateBuilder {
	b.meta = meta
	return b
}

// Ensure creates or updates the component template (idempotent)
func (b *ComponentTemplateBuilder) Ensure() error {
	if b.name == "" {
		return fmt.Errorf("component template name is required")
	}

	templateContent := make(map[string]interface{})
	if len(b.settings) > 0 {
		templateContent["settings"] = b.settings
	}
	if len(b.properties) > 0 || b.dynamic != "" {
		mappings := make(map[string]interface{})
		if len(b.properties) > 0 {
			mappings["properties"] = b.properties
		}
		if b.dynamic != "" {
			mappings["dynamic"] = b.dynamic
		}
		templateContent["mappings"] = mappings
	}
	if len(b.aliases) > 0 {
		templateContent["aliases"] = b.aliases
	}

	body := map[string]interface{}{
		"template": templateContent,
	}
	if b.version != nil {
		body["version"] = *b.version
	}
	if b.meta != nil {
		body["_meta"] = b.meta
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal component template: %w", err)
	}

	path := fmt.Sprintf("/_component_template/%s", b.name)
	req, err := http.NewRequestWithContext(b.ctx, "PUT", path, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to create/update component template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create/update component template: %s", string(bodyBytes))
	}

	return nil
}
