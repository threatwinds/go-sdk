package os

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CatIndex is one row of the _cat/indices response: the index-listing shape with
// docs count, store size and creation date (string values, as _cat returns them).
type CatIndex struct {
	Index        string `json:"index"`
	Health       string `json:"health"`
	Status       string `json:"status"`
	UUID         string `json:"uuid"`
	DocsCount    string `json:"docs.count"`
	DocsDeleted  string `json:"docs.deleted"`
	StoreSize    string `json:"store.size"`
	CreationDate string `json:"creation.date"`
}

// ListIndices returns the indices matching a pattern via _cat/indices, including
// docs count, store size and creation date. Pass "*" or "" to list all indices.
// A pattern that matches no index returns an empty slice (not an error).
func ListIndices(ctx context.Context, pattern string) ([]CatIndex, error) {
	if pattern == "" {
		pattern = "*"
	}
	path := fmt.Sprintf(
		"/_cat/indices/%s?format=json&h=index,health,status,uuid,docs.count,docs.deleted,store.size,creation.date",
		pattern,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cat indices request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list indices: %w", err)
	}
	defer resp.Body.Close()

	// _cat/indices returns 404 when the pattern matches no index — treat as empty.
	if resp.StatusCode == http.StatusNotFound {
		return []CatIndex{}, nil
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cat indices request failed: %s", string(b))
	}

	var indices []CatIndex
	if err := json.NewDecoder(resp.Body).Decode(&indices); err != nil {
		return nil, fmt.Errorf("failed to decode cat indices response: %w", err)
	}
	return indices, nil
}
