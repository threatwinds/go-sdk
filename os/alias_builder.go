package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AliasConfig represents a configuration for an alias
type AliasConfig struct {
	Filter        map[string]interface{} `json:"filter,omitempty"`
	Routing       string                 `json:"routing,omitempty"`
	SearchRouting string                 `json:"search_routing,omitempty"`
	IndexRouting  string                 `json:"index_routing,omitempty"`
	IsWriteIndex  *bool                  `json:"is_write_index,omitempty"`
	IsHidden      *bool                  `json:"is_hidden,omitempty"`
}

// AliasAction represents a single alias action (add or remove)
type AliasAction struct {
	ActionType   string                 `json:"-"` // "add" or "remove"
	Index        string                 `json:"index,omitempty"`
	Alias        string                 `json:"alias,omitempty"`
	Indices      []string               `json:"indices,omitempty"`
	Filter       map[string]interface{} `json:"filter,omitempty"`
	Routing      string                 `json:"routing,omitempty"`
	IsWriteIndex *bool                  `json:"is_write_index,omitempty"`
	IsHidden     *bool                  `json:"is_hidden,omitempty"`
}

// AliasInfo represents information about an alias
type AliasInfo struct {
	Alias  string
	Index  string
	Config AliasConfig
}

// AliasBuilder provides a fluent API for alias operations
type AliasBuilder struct {
	ctx     context.Context
	actions []AliasAction
	errors  []error
}

// NewAliasBuilder creates a new alias builder
func NewAliasBuilder(ctx context.Context) *AliasBuilder {
	return &AliasBuilder{
		ctx:     ctx,
		actions: []AliasAction{},
		errors:  []error{},
	}
}

// Add adds an alias to an index
func (b *AliasBuilder) Add(index, alias string) *AliasBuilder {
	b.actions = append(b.actions, AliasAction{
		ActionType: "add",
		Index:      index,
		Alias:      alias,
	})
	return b
}

// AddWithFilter adds an alias with a filter
func (b *AliasBuilder) AddWithFilter(index, alias string, filter map[string]interface{}) *AliasBuilder {
	b.actions = append(b.actions, AliasAction{
		ActionType: "add",
		Index:      index,
		Alias:      alias,
		Filter:     filter,
	})
	return b
}

// AddWriteIndex adds an alias as the write index
func (b *AliasBuilder) AddWriteIndex(index, alias string) *AliasBuilder {
	isWriteIndex := true
	b.actions = append(b.actions, AliasAction{
		ActionType:   "add",
		Index:        index,
		Alias:        alias,
		IsWriteIndex: &isWriteIndex,
	})
	return b
}

// AddWithRouting adds an alias with routing
func (b *AliasBuilder) AddWithRouting(index, alias, routing string) *AliasBuilder {
	b.actions = append(b.actions, AliasAction{
		ActionType: "add",
		Index:      index,
		Alias:      alias,
		Routing:    routing,
	})
	return b
}

// AddHidden adds a hidden alias
func (b *AliasBuilder) AddHidden(index, alias string) *AliasBuilder {
	isHidden := true
	b.actions = append(b.actions, AliasAction{
		ActionType: "add",
		Index:      index,
		Alias:      alias,
		IsHidden:   &isHidden,
	})
	return b
}

// AddWithConfig adds an alias with full configuration
func (b *AliasBuilder) AddWithConfig(index, alias string, config AliasConfig) *AliasBuilder {
	b.actions = append(b.actions, AliasAction{
		ActionType:   "add",
		Index:        index,
		Alias:        alias,
		Filter:       config.Filter,
		Routing:      config.Routing,
		IsWriteIndex: config.IsWriteIndex,
		IsHidden:     config.IsHidden,
	})
	return b
}

// Remove removes an alias from an index
func (b *AliasBuilder) Remove(index, alias string) *AliasBuilder {
	b.actions = append(b.actions, AliasAction{
		ActionType: "remove",
		Index:      index,
		Alias:      alias,
	})
	return b
}

// RemoveIndex removes all aliases from an index
func (b *AliasBuilder) RemoveIndex(index string) *AliasBuilder {
	b.actions = append(b.actions, AliasAction{
		ActionType: "remove",
		Index:      index,
		Alias:      "*",
	})
	return b
}

// Switch atomically switches an alias from one index to another
func (b *AliasBuilder) Switch(alias, fromIndex, toIndex string) *AliasBuilder {
	// Remove from old index, add to new index
	b.Remove(fromIndex, alias)
	b.Add(toIndex, alias)
	return b
}

// SwitchWriteIndex atomically switches the write index for an alias
func (b *AliasBuilder) SwitchWriteIndex(alias, fromIndex, toIndex string) *AliasBuilder {
	isWriteIndexFalse := false
	isWriteIndexTrue := true
	// Set old index as non-write
	b.actions = append(b.actions, AliasAction{
		ActionType:   "add",
		Index:        fromIndex,
		Alias:        alias,
		IsWriteIndex: &isWriteIndexFalse,
	})
	// Set new index as write
	b.actions = append(b.actions, AliasAction{
		ActionType:   "add",
		Index:        toIndex,
		Alias:        alias,
		IsWriteIndex: &isWriteIndexTrue,
	})
	return b
}

// Build returns the alias update request body
func (b *AliasBuilder) Build() (map[string]interface{}, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("alias builder has %d errors: %v", len(b.errors), b.errors)
	}

	if len(b.actions) == 0 {
		return nil, fmt.Errorf("no actions defined")
	}

	actions := make([]map[string]interface{}, 0, len(b.actions))
	for _, action := range b.actions {
		actionMap := make(map[string]interface{})
		if action.Index != "" {
			actionMap["index"] = action.Index
		}
		if len(action.Indices) > 0 {
			actionMap["indices"] = action.Indices
		}
		if action.Alias != "" {
			actionMap["alias"] = action.Alias
		}
		if action.Filter != nil {
			actionMap["filter"] = action.Filter
		}
		if action.Routing != "" {
			actionMap["routing"] = action.Routing
		}
		if action.IsWriteIndex != nil {
			actionMap["is_write_index"] = *action.IsWriteIndex
		}
		if action.IsHidden != nil {
			actionMap["is_hidden"] = *action.IsHidden
		}

		actions = append(actions, map[string]interface{}{
			action.ActionType: actionMap,
		})
	}

	return map[string]interface{}{
		"actions": actions,
	}, nil
}

// BuildWithErrors returns the request body and any accumulated errors
func (b *AliasBuilder) BuildWithErrors() (map[string]interface{}, []error) {
	if len(b.actions) == 0 {
		return nil, append(b.errors, fmt.Errorf("no actions defined"))
	}

	actions := make([]map[string]interface{}, 0, len(b.actions))
	for _, action := range b.actions {
		actionMap := make(map[string]interface{})
		if action.Index != "" {
			actionMap["index"] = action.Index
		}
		if len(action.Indices) > 0 {
			actionMap["indices"] = action.Indices
		}
		if action.Alias != "" {
			actionMap["alias"] = action.Alias
		}
		if action.Filter != nil {
			actionMap["filter"] = action.Filter
		}
		if action.Routing != "" {
			actionMap["routing"] = action.Routing
		}
		if action.IsWriteIndex != nil {
			actionMap["is_write_index"] = *action.IsWriteIndex
		}
		if action.IsHidden != nil {
			actionMap["is_hidden"] = *action.IsHidden
		}

		actions = append(actions, map[string]interface{}{
			action.ActionType: actionMap,
		})
	}

	return map[string]interface{}{
		"actions": actions,
	}, b.errors
}

// Ensure executes the alias actions (idempotent atomic operation)
func (b *AliasBuilder) Ensure() error {
	body, err := b.Build()
	if err != nil {
		return err
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal alias actions: %w", err)
	}

	// Use a low-level client for alias update
	req, err := http.NewRequestWithContext(b.ctx, "POST", "/_aliases", bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to update aliases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update aliases: %s", string(bodyBytes))
	}

	return nil
}

// GetAliases retrieves aliases for the given index pattern
func GetAliases(ctx context.Context, indexPattern string) ([]AliasInfo, error) {
	path := fmt.Sprintf("/%s/_alias", indexPattern)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get aliases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return []AliasInfo{}, nil
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get aliases: %s", string(bodyBytes))
	}

	var result map[string]struct {
		Aliases map[string]AliasConfig `json:"aliases"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var aliases []AliasInfo
	for indexName, indexData := range result {
		for aliasName, config := range indexData.Aliases {
			aliases = append(aliases, AliasInfo{
				Alias:  aliasName,
				Index:  indexName,
				Config: config,
			})
		}
	}

	return aliases, nil
}

// AliasExists checks if an alias exists
func AliasExists(ctx context.Context, alias string) (bool, error) {
	path := fmt.Sprintf("/_alias/%s", alias)
	req, err := http.NewRequestWithContext(ctx, "HEAD", path, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return false, fmt.Errorf("failed to check alias: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

// GetIndicesForAlias returns all indices that have the given alias
func GetIndicesForAlias(ctx context.Context, alias string) ([]string, error) {
	path := fmt.Sprintf("/_alias/%s", alias)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get indices for alias: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return []string{}, nil
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get indices for alias: %s", string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	indices := make([]string, 0, len(result))
	for indexName := range result {
		indices = append(indices, indexName)
	}

	return indices, nil
}

// DeleteAlias removes an alias from an index
func DeleteAlias(ctx context.Context, index, alias string) error {
	return NewAliasBuilder(ctx).Remove(index, alias).Ensure()
}

// GetWriteIndex returns the write index for an alias
func GetWriteIndex(ctx context.Context, alias string) (string, error) {
	aliases, err := GetAliases(ctx, "*")
	if err != nil {
		return "", err
	}

	for _, info := range aliases {
		if info.Alias == alias && info.Config.IsWriteIndex != nil && *info.Config.IsWriteIndex {
			return info.Index, nil
		}
	}

	// If no explicit write index, check if alias points to single index
	indices, err := GetIndicesForAlias(ctx, alias)
	if err != nil {
		return "", err
	}

	if len(indices) == 1 {
		return indices[0], nil
	}

	return "", fmt.Errorf("no write index found for alias: %s", alias)
}
