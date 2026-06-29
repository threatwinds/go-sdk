package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SQLColumn describes one column in an OpenSearch SQL response schema.
type SQLColumn struct {
	Name  string `json:"name"`
	Alias string `json:"alias,omitempty"`
	Type  string `json:"type"`
}

// SQLResult is the raw response from the OpenSearch SQL plugin (_plugins/_sql).
type SQLResult struct {
	Schema   []SQLColumn `json:"schema"`
	Datarows [][]any     `json:"datarows"`
	Total    int64       `json:"total"`
	Size     int64       `json:"size"`
}

// QuerySQL runs a query against the OpenSearch SQL plugin and returns the raw
// schema/datarows result. The query must be a single SELECT statement; callers
// are responsible for validating it before calling.
func QuerySQL(ctx context.Context, query string) (*SQLResult, error) {
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SQL query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/_plugins/_sql", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SQL query failed: %s", string(b))
	}

	var result SQLResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode SQL response: %w", err)
	}
	return &result, nil
}

// SearchSQL runs a SQL query and maps each datarow to a map keyed by the column
// name (or alias when present), matching the row shape of a normal search.
func SearchSQL(ctx context.Context, query string) ([]map[string]any, error) {
	result, err := QuerySQL(ctx, query)
	if err != nil {
		return nil, err
	}

	rows := make([]map[string]any, 0, len(result.Datarows))
	for _, dr := range result.Datarows {
		row := make(map[string]any, len(result.Schema))
		for i, col := range result.Schema {
			key := col.Name
			if col.Alias != "" {
				key = col.Alias
			}
			if i < len(dr) {
				row[key] = dr[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// CountSQL runs a COUNT-style SQL query and returns the first numeric cell of the
// first datarow (the count). Returns 0 when there are no rows.
func CountSQL(ctx context.Context, query string) (int64, error) {
	result, err := QuerySQL(ctx, query)
	if err != nil {
		return 0, err
	}
	if len(result.Datarows) == 0 || len(result.Datarows[0]) == 0 {
		return 0, nil
	}
	switch v := result.Datarows[0][0].(type) {
	case float64:
		return int64(v), nil
	case json.Number:
		n, _ := v.Int64()
		return n, nil
	default:
		return 0, nil
	}
}
