package os

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// RawSearch executes a search using a pre-built raw request body (query,
// aggregations, sort, from/size, _source, etc.) against the given indices and
// returns the parsed SearchResult.
//
// Use this when the request body was assembled as a map — e.g. from an external
// query DSL — rather than constructed with QueryBuilder. Unlike SearchIn it does
// not inject any access-control (visibleBy) filtering.
func RawSearch(ctx context.Context, indices []string, body map[string]any) (SearchResult, error) {
	j, err := json.Marshal(body)
	if err != nil {
		return SearchResult{}, err
	}

	req := &opensearchapi.SearchReq{
		Indices: indices,
		Body:    strings.NewReader(string(j)),
	}

	resp, err := apiClient.Search(ctx, req)
	if err != nil {
		return SearchResult{}, err
	}

	// Map the v4 response to our SearchResult struct (same mapping as WideSearchIn).
	result := SearchResult{
		Took:     int64(resp.Took),
		TimedOut: resp.Timeout,
		Shards: Shards{
			Total:      int64(resp.Shards.Total),
			Successful: int64(resp.Shards.Successful),
			Skipped:    int64(resp.Shards.Skipped),
			Failed:     int64(resp.Shards.Failed),
		},
		Hits: Hits{
			Total: Total{
				Value:    int64(resp.Hits.Total.Value),
				Relation: resp.Hits.Total.Relation,
			},
			MaxScore: resp.Hits.MaxScore,
			Hits:     make([]Hit, len(resp.Hits.Hits)),
		},
	}

	for i, hit := range resp.Hits.Hits {
		var source HitSource
		if len(hit.Source) > 0 {
			if err := json.Unmarshal(hit.Source, &source); err != nil {
				return SearchResult{}, err
			}
		}

		var fields map[string]interface{}
		if len(hit.Fields) > 0 {
			if err := json.Unmarshal(hit.Fields, &fields); err != nil {
				return SearchResult{}, err
			}
		}

		result.Hits.Hits[i] = Hit{
			Index:  hit.Index,
			ID:     hit.ID,
			Score:  hit.Score,
			Source: source,
			Fields: fields,
			Sort:   hit.Sort,
		}
	}

	if len(resp.Aggregations) > 0 {
		result.Aggregations = make(map[string]interface{})
		if err := json.Unmarshal(resp.Aggregations, &result.Aggregations); err != nil {
			return SearchResult{}, err
		}
	}

	return result, nil
}
