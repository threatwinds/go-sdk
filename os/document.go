package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// Index indexes a document in OpenSearch (upsert - creates or replaces)
func Index(ctx context.Context, index, id string, doc []byte) error {
	req := opensearchapi.IndexReq{
		Index:      index,
		Body:       bytes.NewReader(doc),
		DocumentID: id,
	}

	_, err := apiClient.Index(ctx, req)
	return err
}

// Get retrieves a document by ID
func Get(ctx context.Context, index, id string) ([]byte, error) {
	req := opensearchapi.DocumentGetReq{
		Index:      index,
		DocumentID: id,
	}

	resp, err := apiClient.Document.Get(ctx, req)
	if err != nil {
		// Check if it's a 404 (document not found)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not_found") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if !resp.Found {
		return nil, nil
	}

	return resp.Source, nil
}

// Delete removes a document by ID
func Delete(ctx context.Context, index, id string) error {
	req := opensearchapi.DocumentDeleteReq{
		Index:      index,
		DocumentID: id,
	}

	_, err := apiClient.Document.Delete(ctx, req)
	if err != nil {
		// Ignore 404 errors (document doesn't exist)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not_found") {
			return nil
		}
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// Exists checks if a document exists
func Exists(ctx context.Context, index, id string) (bool, error) {
	req := opensearchapi.DocumentExistsReq{
		Index:      index,
		DocumentID: id,
	}

	_, err := apiClient.Document.Exists(ctx, req)
	if err != nil {
		// Check if it's a 404 (document not found)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not_found") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check document existence: %w", err)
	}

	return true, nil
}

// BulkIndexItem represents an item for bulk indexing
type BulkIndexItem struct {
	Index    string
	ID       string
	Doc      []byte
	IsDelete bool
}

// BulkIndex performs bulk index/delete operations
func BulkIndex(ctx context.Context, items []BulkIndexItem) error {
	if len(items) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, item := range items {
		if item.IsDelete {
			// Delete action
			action := map[string]interface{}{
				"delete": map[string]interface{}{
					"_index": item.Index,
					"_id":    item.ID,
				},
			}
			actionBytes, err := json.Marshal(action)
			if err != nil {
				return fmt.Errorf("failed to marshal delete action: %w", err)
			}
			buf.Write(actionBytes)
			buf.WriteString("\n")
		} else {
			// Index action
			action := map[string]interface{}{
				"index": map[string]interface{}{
					"_index": item.Index,
					"_id":    item.ID,
				},
			}
			actionBytes, err := json.Marshal(action)
			if err != nil {
				return fmt.Errorf("failed to marshal index action: %w", err)
			}
			buf.Write(actionBytes)
			buf.WriteString("\n")
			buf.Write(item.Doc)
			buf.WriteString("\n")
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "/_bulk", &buf)
	if err != nil {
		return fmt.Errorf("failed to create bulk request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("bulk request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bulk request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response to check for errors
	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []struct {
			Index  *bulkItemResult `json:"index,omitempty"`
			Delete *bulkItemResult `json:"delete,omitempty"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		return fmt.Errorf("failed to decode bulk response: %w", err)
	}

	if bulkResp.Errors {
		// Collect error messages
		var errMsgs []string
		for i, item := range bulkResp.Items {
			var result *bulkItemResult
			if item.Index != nil {
				result = item.Index
			} else if item.Delete != nil {
				result = item.Delete
			}
			if result != nil && result.Error != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("item %d: %s - %s", i, result.Error.Type, result.Error.Reason))
			}
		}
		return fmt.Errorf("bulk operation had errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

type bulkItemResult struct {
	Index  string `json:"_index"`
	ID     string `json:"_id"`
	Status int    `json:"status"`
	Error  *struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error,omitempty"`
}

// SearchRaw performs a raw search query
func SearchRaw(ctx context.Context, index string, query []byte) (*SearchResultRaw, error) {
	req := &opensearchapi.SearchReq{
		Indices: []string{index},
		Body:    bytes.NewReader(query),
	}

	resp, err := apiClient.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	result := &SearchResultRaw{
		Total:    int64(resp.Hits.Total.Value),
		MaxScore: float64(*resp.Hits.MaxScore),
		Hits:     make([]HitRaw, len(resp.Hits.Hits)),
	}

	for i, hit := range resp.Hits.Hits {
		result.Hits[i] = HitRaw{
			ID:     hit.ID,
			Index:  hit.Index,
			Score:  float64(hit.Score),
			Source: hit.Source,
		}
	}

	return result, nil
}

// SearchResultRaw represents raw search results
type SearchResultRaw struct {
	Total    int64
	MaxScore float64
	Hits     []HitRaw
}

// HitRaw represents a raw search hit
type HitRaw struct {
	ID     string
	Index  string
	Score  float64
	Source json.RawMessage
}
