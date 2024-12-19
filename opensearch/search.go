package opensearch

import (
	"context"
	"encoding/json"
	gosdk "github.com/threatwinds/go-sdk"
	"io"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

func (q SearchRequest) SearchIn(ctx context.Context, index []string) (SearchResult, error) {
	if q.Source == nil {
		q.Source = new(Source)
	}

	j, err := json.Marshal(q)
	if err != nil {
		return SearchResult{}, gosdk.Error(gosdk.Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "failed to encode search request",
		})
	}

	reader := strings.NewReader(string(j))

	req := opensearchapi.SearchRequest{
		Index: index,
		Body:  reader,
	}

	resp, err := req.Do(ctx, client)
	if err != nil {
		return SearchResult{}, gosdk.Error(gosdk.Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "failed to execute search request",
		})
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResult{}, gosdk.Error(gosdk.Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "failed to read search response",
		})
	}

	if resp.StatusCode != http.StatusOK {
		return SearchResult{}, gosdk.Error(gosdk.Trace(), map[string]interface{}{
			"error":      "failed to execute search request",
			"statusCode": resp.Status,
			"response":   string(body),
		})
	}

	var result SearchResult

	err = json.Unmarshal(body, &result)
	if err != nil {
		return SearchResult{}, gosdk.Error(gosdk.Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "failed to decode search response",
		})
	}

	return result, nil
}
