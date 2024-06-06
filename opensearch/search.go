package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
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
		return SearchResult{}, err
	}

	reader := strings.NewReader(string(j))

	req := opensearchapi.SearchRequest{
		Index: index,
		Body:  reader,
	}

	resp, err := req.Do(ctx, client)
	if err != nil {
		return SearchResult{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResult{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return SearchResult{}, fmt.Errorf("search engine status %d, response: %s", resp.StatusCode, body)
	}

	var result SearchResult

	err = json.Unmarshal(body, &result)
	if err != nil {
		return SearchResult{}, err
	}

	return result, nil
}
