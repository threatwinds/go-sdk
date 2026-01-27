package os

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// SearchIn executes a search with automatic visibleBy.keyword filtering based on provided groups.
// It enforces group-based access control by filtering documents that are visible to the specified groups.
// Use this for user-facing queries that require access control.
// It returns a SearchResult and an error, if any.
func (q SearchRequest) SearchIn(ctx context.Context, index []string, groups []string) (SearchResult, error) {
	if q.Query == nil {
		q.Query = &Query{}
	}

	var groupList = make([]interface{}, 0, len(groups))
	for _, group := range groups {
		groupList = append(groupList, group)
	}

	visibilityFilter := Query{Terms: map[string][]interface{}{"visibleBy.keyword": groupList}}

	// If it's a k-NN query, we need to apply the filter to the k-NN filter clause
	if len(q.Query.KNN) > 0 {
		for _, knn := range q.Query.KNN {
			if knn.Filter == nil {
				knn.Filter = &Query{}
			}
			if knn.Filter.Bool == nil {
				knn.Filter.Bool = &Bool{}
			}

			// Clean up existing visibility filters if any
			for i := range knn.Filter.Bool.Filter {
				delete(knn.Filter.Bool.Filter[i].Terms, "visibleBy")
				delete(knn.Filter.Bool.Filter[i].Terms, "visibleBy.keyword")
			}

			// Add visibility filter
			knn.Filter.Bool.Filter = append(knn.Filter.Bool.Filter, visibilityFilter)
		}
	} else {
		// Standard Bool query path
		if q.Query.Bool == nil {
			q.Query.Bool = &Bool{}
		}

		for i := range q.Query.Bool.Filter {
			delete(q.Query.Bool.Filter[i].Terms, "visibleBy")
			delete(q.Query.Bool.Filter[i].Terms, "visibleBy.keyword")
		}

		q.Query.Bool.Filter = append(q.Query.Bool.Filter, visibilityFilter)
	}

	return q.WideSearchIn(ctx, index)
}

// WideSearchIn executes a search without access control filtering.
// Use this for admin or system operations that don't require group-based filtering.
// It returns a SearchResult and an error, if any.
func (q SearchRequest) WideSearchIn(ctx context.Context, index []string) (SearchResult, error) {
	if q.Source == nil {
		q.Source = new(Source)
	}

	j, err := json.Marshal(q)
	if err != nil {
		return SearchResult{}, err
	}

	reader := strings.NewReader(string(j))

	req := &opensearchapi.SearchReq{
		Indices: index,
		Body:    reader,
	}

	resp, err := apiClient.Search(ctx, req)
	if err != nil {
		return SearchResult{}, err
	}

	// Map the v4 response to our SearchResult struct
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

	// Convert SearchHit array to our Hit array
	for i, hit := range resp.Hits.Hits {
		// Parse Source from json.RawMessage to HitSource (map[string]interface{})
		var source HitSource
		if len(hit.Source) > 0 {
			if err := json.Unmarshal(hit.Source, &source); err != nil {
				return SearchResult{}, err
			}
		}

		// Parse Fields from json.RawMessage to map[string]interface{}
		var fields map[string]interface{}
		if len(hit.Fields) > 0 {
			if err := json.Unmarshal(hit.Fields, &fields); err != nil {
				return SearchResult{}, err
			}
		}

		result.Hits.Hits[i] = Hit{
			Index:   hit.Index,
			ID:      hit.ID,
			Version: 0, // Version field is not available in v4 SearchHit
			Score:   hit.Score,
			Source:  source,
			Fields:  fields,
			Sort:    hit.Sort,
		}
	}

	// Handle aggregations
	if len(resp.Aggregations) > 0 {
		result.Aggregations = make(map[string]interface{})
		err = json.Unmarshal(resp.Aggregations, &result.Aggregations)
		if err != nil {
			return SearchResult{}, err
		}
	}

	return result, nil
}
