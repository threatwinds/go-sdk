package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RolloverConditions represents conditions for triggering a rollover
type RolloverConditions struct {
	MaxAge              string `json:"max_age,omitempty"`
	MaxDocs             int64  `json:"max_docs,omitempty"`
	MaxSize             string `json:"max_size,omitempty"`
	MaxPrimaryShardSize string `json:"max_primary_shard_size,omitempty"`
	MaxPrimaryShardDocs int64  `json:"max_primary_shard_docs,omitempty"`
	MinAge              string `json:"min_age,omitempty"`
	MinDocs             int64  `json:"min_docs,omitempty"`
	MinSize             string `json:"min_size,omitempty"`
	MinPrimaryShardSize string `json:"min_primary_shard_size,omitempty"`
	MinPrimaryShardDocs int64  `json:"min_primary_shard_docs,omitempty"`
}

// RolloverResult represents the result of a rollover operation
type RolloverResult struct {
	OldIndex           string          `json:"old_index"`
	NewIndex           string          `json:"new_index"`
	RolledOver         bool            `json:"rolled_over"`
	DryRun             bool            `json:"dry_run"`
	Acknowledged       bool            `json:"acknowledged"`
	ShardsAcknowledged bool            `json:"shards_acknowledged"`
	Conditions         map[string]bool `json:"conditions"`
}

// RolloverBuilder provides a fluent API for rollover operations
type RolloverBuilder struct {
	ctx         context.Context
	alias       string
	newIndex    string
	conditions  RolloverConditions
	settings    map[string]interface{}
	mappings    map[string]interface{}
	aliases     map[string]interface{}
	dryRun      bool
	errors      []error
	processName string
}

// NewRolloverBuilder creates a new rollover builder for the specified alias
func NewRolloverBuilder(ctx context.Context, alias string, processName string) *RolloverBuilder {
	return &RolloverBuilder{
		ctx:         ctx,
		alias:       alias,
		errors:      []error{},
		processName: processName,
	}
}

// MaxAge sets the maximum age condition
func (b *RolloverBuilder) MaxAge(age string) *RolloverBuilder {
	b.conditions.MaxAge = age
	return b
}

// MaxDocs sets the maximum document count condition
func (b *RolloverBuilder) MaxDocs(docs int64) *RolloverBuilder {
	b.conditions.MaxDocs = docs
	return b
}

// MaxSize sets the maximum index size condition
func (b *RolloverBuilder) MaxSize(size string) *RolloverBuilder {
	b.conditions.MaxSize = size
	return b
}

// MaxPrimaryShardSize sets the maximum primary shard size condition
func (b *RolloverBuilder) MaxPrimaryShardSize(size string) *RolloverBuilder {
	b.conditions.MaxPrimaryShardSize = size
	return b
}

// MaxPrimaryShardDocs sets the maximum primary shard document count condition
func (b *RolloverBuilder) MaxPrimaryShardDocs(docs int64) *RolloverBuilder {
	b.conditions.MaxPrimaryShardDocs = docs
	return b
}

// MinAge sets the minimum age condition
func (b *RolloverBuilder) MinAge(age string) *RolloverBuilder {
	b.conditions.MinAge = age
	return b
}

// MinDocs sets the minimum document count condition
func (b *RolloverBuilder) MinDocs(docs int64) *RolloverBuilder {
	b.conditions.MinDocs = docs
	return b
}

// MinSize sets the minimum index size condition
func (b *RolloverBuilder) MinSize(size string) *RolloverBuilder {
	b.conditions.MinSize = size
	return b
}

// MinPrimaryShardSize sets the minimum primary shard size condition
func (b *RolloverBuilder) MinPrimaryShardSize(size string) *RolloverBuilder {
	b.conditions.MinPrimaryShardSize = size
	return b
}

// MinPrimaryShardDocs sets the minimum primary shard document count condition
func (b *RolloverBuilder) MinPrimaryShardDocs(docs int64) *RolloverBuilder {
	b.conditions.MinPrimaryShardDocs = docs
	return b
}

// NewIndexName sets a custom name for the new index
func (b *RolloverBuilder) NewIndexName(name string) *RolloverBuilder {
	b.newIndex = name
	return b
}

// Settings sets custom settings for the new index
func (b *RolloverBuilder) Settings(settings map[string]interface{}) *RolloverBuilder {
	b.settings = settings
	return b
}

// Mappings sets custom mappings for the new index
func (b *RolloverBuilder) Mappings(mappings map[string]interface{}) *RolloverBuilder {
	b.mappings = mappings
	return b
}

// Aliases sets custom aliases for the new index
func (b *RolloverBuilder) Aliases(aliases map[string]interface{}) *RolloverBuilder {
	b.aliases = aliases
	return b
}

// DryRun enables dry run mode (check conditions without rolling over)
func (b *RolloverBuilder) DryRun() *RolloverBuilder {
	b.dryRun = true
	return b
}

// Build returns the rollover request body
func (b *RolloverBuilder) Build() (map[string]interface{}, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("rollover builder has %d errors: %v", len(b.errors), b.errors)
	}

	if b.alias == "" {
		return nil, fmt.Errorf("alias is required")
	}

	body := make(map[string]interface{})

	// Build conditions
	conditions := make(map[string]interface{})
	if b.conditions.MaxAge != "" {
		conditions["max_age"] = b.conditions.MaxAge
	}
	if b.conditions.MaxDocs > 0 {
		conditions["max_docs"] = b.conditions.MaxDocs
	}
	if b.conditions.MaxSize != "" {
		conditions["max_size"] = b.conditions.MaxSize
	}
	if b.conditions.MaxPrimaryShardSize != "" {
		conditions["max_primary_shard_size"] = b.conditions.MaxPrimaryShardSize
	}
	if b.conditions.MaxPrimaryShardDocs > 0 {
		conditions["max_primary_shard_docs"] = b.conditions.MaxPrimaryShardDocs
	}
	if b.conditions.MinAge != "" {
		conditions["min_age"] = b.conditions.MinAge
	}
	if b.conditions.MinDocs > 0 {
		conditions["min_docs"] = b.conditions.MinDocs
	}
	if b.conditions.MinSize != "" {
		conditions["min_size"] = b.conditions.MinSize
	}
	if b.conditions.MinPrimaryShardSize != "" {
		conditions["min_primary_shard_size"] = b.conditions.MinPrimaryShardSize
	}
	if b.conditions.MinPrimaryShardDocs > 0 {
		conditions["min_primary_shard_docs"] = b.conditions.MinPrimaryShardDocs
	}

	if len(conditions) > 0 {
		body["conditions"] = conditions
	}

	if b.settings != nil {
		body["settings"] = b.settings
	}
	if b.mappings != nil {
		body["mappings"] = b.mappings
	}
	if b.aliases != nil {
		body["aliases"] = b.aliases
	}

	return body, nil
}

// Execute performs the rollover operation
func (b *RolloverBuilder) Execute() (*RolloverResult, error) {
	body, err := b.Build()
	if err != nil {
		return nil, err
	}

	var bodyJSON []byte
	if len(body) > 0 {
		bodyJSON, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal rollover request: %w", err)
		}
	}

	// Build the path
	path := fmt.Sprintf("/%s/_rollover", b.alias)
	if b.newIndex != "" {
		path = fmt.Sprintf("/%s/_rollover/%s", b.alias, b.newIndex)
	}
	if b.dryRun {
		path += "?dry_run=true"
	}

	var reqBody io.Reader
	if len(bodyJSON) > 0 {
		reqBody = bytes.NewReader(bodyJSON)
	}

	req, err := http.NewRequestWithContext(b.ctx, "POST", path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if len(bodyJSON) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute rollover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rollover failed: %s", string(bodyBytes))
	}

	var result RolloverResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode rollover response: %w", err)
	}

	return &result, nil
}

// SetupRolloverAlias creates the initial index and alias for rollover
// This is idempotent - if the alias already exists, it does nothing
// The setupFunc parameter is called with a new IndexBuilder for configuration
func SetupRolloverAlias(ctx context.Context, indexPrefix, alias, processName string, setupFunc func(*IndexBuilder) *IndexBuilder) error {
	// Check if alias already exists
	exists, err := AliasExists(ctx, alias, processName)
	if err != nil {
		return fmt.Errorf("failed to check alias existence: %w", err)
	}

	if exists {
		// Alias already exists, nothing to do
		return nil
	}

	// Create the initial index name
	initialIndex := BuildInitialRolloverIndex(indexPrefix)

	// Check if initial index exists
	indexExists, err := IndexExists(ctx, initialIndex)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if !indexExists {
		// Create the initial index with the write alias
		builder := NewIndexBuilder(ctx, initialIndex, processName).WriteAlias(alias)
		if setupFunc != nil {
			builder = setupFunc(builder)
		}
		err = builder.Ensure()
		if err != nil {
			return fmt.Errorf("failed to create initial index: %w", err)
		}
	} else {
		// Index exists but alias doesn't - add the alias
		err = NewAliasBuilder(ctx, processName).
			AddWriteIndex(initialIndex, alias).
			Ensure()
		if err != nil {
			return fmt.Errorf("failed to add alias to existing index: %w", err)
		}
	}

	return nil
}

// ForceRollover performs an unconditional rollover (no conditions)
func ForceRollover(ctx context.Context, alias, processName string) (*RolloverResult, error) {
	return NewRolloverBuilder(ctx, alias, processName).Execute()
}

// CheckRolloverConditions checks if rollover conditions are met without rolling over
func CheckRolloverConditions(ctx context.Context, alias, processName string, conditions RolloverConditions) (*RolloverResult, error) {
	builder := NewRolloverBuilder(ctx, alias, processName).DryRun()

	if conditions.MaxAge != "" {
		builder.MaxAge(conditions.MaxAge)
	}
	if conditions.MaxDocs > 0 {
		builder.MaxDocs(conditions.MaxDocs)
	}
	if conditions.MaxSize != "" {
		builder.MaxSize(conditions.MaxSize)
	}
	if conditions.MaxPrimaryShardSize != "" {
		builder.MaxPrimaryShardSize(conditions.MaxPrimaryShardSize)
	}
	if conditions.MaxPrimaryShardDocs > 0 {
		builder.MaxPrimaryShardDocs(conditions.MaxPrimaryShardDocs)
	}

	return builder.Execute()
}

// GetCurrentWriteIndex returns the current write index for an alias
func GetCurrentWriteIndex(ctx context.Context, alias, processName string) (string, error) {
	return GetWriteIndex(ctx, alias, processName)
}
