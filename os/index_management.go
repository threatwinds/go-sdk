package os

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// IndexInfo represents information about an index
type IndexInfo struct {
	Name     string                 `json:"name"`
	Settings map[string]interface{} `json:"settings"`
	Mappings map[string]interface{} `json:"mappings"`
	Aliases  map[string]interface{} `json:"aliases"`
}

// IndexStats represents index statistics
type IndexStats struct {
	DocsCount      int64 `json:"docs_count"`
	DocsDeleted    int64 `json:"docs_deleted"`
	StoreSizeBytes int64 `json:"store_size_bytes"`
	PrimaryShards  int   `json:"primary_shards"`
	ReplicaShards  int   `json:"replica_shards"`
}

// IndexExists checks if an index exists
func IndexExists(ctx context.Context, name string) (bool, error) {
	req := opensearchapi.IndicesExistsReq{
		Indices: []string{name},
	}

	_, err := apiClient.Indices.Exists(ctx, req)
	if err != nil {
		// Check if it's a 404 (index doesn't exist)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "index_not_found") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check index existence: %w", err)
	}

	return true, nil
}

// DeleteIndex deletes an index
func DeleteIndex(ctx context.Context, name string) error {
	req := opensearchapi.IndicesDeleteReq{
		Indices: []string{name},
	}

	_, err := apiClient.Indices.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	return nil
}

// GetIndex retrieves information about an index
func GetIndex(ctx context.Context, name string) (*IndexInfo, error) {
	req := opensearchapi.IndicesGetReq{
		Indices: []string{name},
	}

	resp, err := apiClient.Indices.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get index: %w", err)
	}

	// Parse the response
	for indexName, data := range resp.Indices {
		info := &IndexInfo{
			Name: indexName,
		}

		// Parse settings from RawMessage
		if len(data.Settings) > 0 {
			var settings map[string]interface{}
			if err := json.Unmarshal(data.Settings, &settings); err == nil {
				if indexSettings, ok := settings["index"].(map[string]interface{}); ok {
					info.Settings = indexSettings
				} else {
					info.Settings = settings
				}
			}
		}

		// Parse mappings from RawMessage
		if len(data.Mappings) > 0 {
			var mappings map[string]interface{}
			if err := json.Unmarshal(data.Mappings, &mappings); err == nil {
				info.Mappings = mappings
			}
		}

		// Parse aliases
		if data.Aliases != nil {
			info.Aliases = make(map[string]interface{})
			for aliasName, aliasData := range data.Aliases {
				info.Aliases[aliasName] = aliasData
			}
		}

		return info, nil
	}

	return nil, fmt.Errorf("index not found: %s", name)
}

// GetIndexSettings retrieves the settings of an index
func GetIndexSettings(ctx context.Context, name string) (map[string]interface{}, error) {
	info, err := GetIndex(ctx, name)
	if err != nil {
		return nil, err
	}
	return info.Settings, nil
}

// GetIndexStats retrieves statistics for an index
func GetIndexStats(ctx context.Context, name string) (*IndexStats, error) {
	req := &opensearchapi.IndicesStatsReq{
		Indices: []string{name},
	}

	resp, err := apiClient.Indices.Stats(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get index stats: %w", err)
	}

	stats := &IndexStats{}

	// Access _all stats
	stats.DocsCount = int64(resp.All.Primaries.Docs.Count)
	stats.DocsDeleted = int64(resp.All.Primaries.Docs.Deleted)
	stats.StoreSizeBytes = resp.All.Primaries.Store.SizeInBytes

	return stats, nil
}

// RefreshIndex refreshes one or more indices
func RefreshIndex(ctx context.Context, names ...string) error {
	req := &opensearchapi.IndicesRefreshReq{
		Indices: names,
	}

	_, err := apiClient.Indices.Refresh(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to refresh index: %w", err)
	}

	return nil
}

// CloseIndex closes an index
func CloseIndex(ctx context.Context, name string) error {
	req := opensearchapi.IndicesCloseReq{
		Index: name,
	}

	_, err := apiClient.Indices.Close(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to close index: %w", err)
	}

	return nil
}

// OpenIndex opens a closed index
func OpenIndex(ctx context.Context, name string) error {
	req := opensearchapi.IndicesOpenReq{
		Index: name,
	}

	_, err := apiClient.Indices.Open(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to open index: %w", err)
	}

	return nil
}

// IndexSettingsUpdater provides a fluent API for updating index settings
type IndexSettingsUpdater struct {
	ctx      context.Context
	index    string
	settings map[string]interface{}
	errors   []error
}

// NewIndexSettingsUpdater creates a new settings updater
func NewIndexSettingsUpdater(ctx context.Context, index string) *IndexSettingsUpdater {
	return &IndexSettingsUpdater{
		ctx:      ctx,
		index:    index,
		settings: make(map[string]interface{}),
		errors:   []error{},
	}
}

// Replicas sets the number of replicas
func (u *IndexSettingsUpdater) Replicas(n int) *IndexSettingsUpdater {
	if u.settings["index"] == nil {
		u.settings["index"] = make(map[string]interface{})
	}
	u.settings["index"].(map[string]interface{})["number_of_replicas"] = n
	return u
}

// RefreshInterval sets the refresh interval
func (u *IndexSettingsUpdater) RefreshInterval(interval string) *IndexSettingsUpdater {
	if u.settings["index"] == nil {
		u.settings["index"] = make(map[string]interface{})
	}
	u.settings["index"].(map[string]interface{})["refresh_interval"] = interval
	return u
}

// MaxResultWindow sets the maximum result window size
func (u *IndexSettingsUpdater) MaxResultWindow(size int) *IndexSettingsUpdater {
	if u.settings["index"] == nil {
		u.settings["index"] = make(map[string]interface{})
	}
	u.settings["index"].(map[string]interface{})["max_result_window"] = size
	return u
}

// CustomSetting sets a custom setting
func (u *IndexSettingsUpdater) CustomSetting(key string, value interface{}) *IndexSettingsUpdater {
	if u.settings["index"] == nil {
		u.settings["index"] = make(map[string]interface{})
	}
	u.settings["index"].(map[string]interface{})[key] = value
	return u
}

// Update applies the settings changes using the low-level client
func (u *IndexSettingsUpdater) Update() error {
	if len(u.errors) > 0 {
		return fmt.Errorf("updater has %d errors: %v", len(u.errors), u.errors)
	}

	body, err := json.Marshal(u.settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Use low-level client for settings update
	path := fmt.Sprintf("/%s/_settings", u.index)
	req, err := http.NewRequestWithContext(u.ctx, "PUT", path, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to update index settings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update settings: %s", string(bodyBytes))
	}

	return nil
}
