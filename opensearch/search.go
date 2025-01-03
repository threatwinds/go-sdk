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
		return SearchResult{}, gosdk.Error("cannot encode search request", err, nil)
	}

	reader := strings.NewReader(string(j))

	req := opensearchapi.SearchRequest{
		Index: index,
		Body:  reader,
	}

	resp, err := req.Do(ctx, client)
	if err != nil {
		return SearchResult{}, gosdk.Error("cannot execute search request", err, nil)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResult{}, gosdk.Error("cannot read search response", err, nil)
	}

	if resp.StatusCode != http.StatusOK {
		return SearchResult{}, gosdk.Error("cannot execute search request", nil, map[string]interface{}{
			"status":   resp.StatusCode,
			"response": string(body),
		})
	}

	var result SearchResult

	err = json.Unmarshal(body, &result)
	if err != nil {
		return SearchResult{}, gosdk.Error("cannot decode search response", err, nil)
	}

	return result, nil
}
