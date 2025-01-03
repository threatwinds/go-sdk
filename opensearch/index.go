package opensearch

import (
	"context"
	"encoding/json"
	gosdk "github.com/threatwinds/go-sdk"
	"io"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

// IndexDoc indexes a document in OpenSearch.
// It takes a document, an index name, and an ID as input parameters.
// The document is marshalled to JSON and sent to OpenSearch for indexing.
// Returns an error if there is an issue with marshalling the document to JSON,
// if there is an issue with the request to OpenSearch, or if the response status code is not 200, 201, or 202.
func IndexDoc(ctx context.Context, doc interface{}, index, id string) error {
	j, err := json.Marshal(doc)
	if err != nil {
		return gosdk.Error("cannot encode document to JSON", err, nil)
	}

	reader := strings.NewReader(string(j))

	req := opensearchapi.IndexRequest{
		Index:      index,
		Body:       reader,
		OpType:     "create",
		DocumentID: id,
	}

	resp, err := req.Do(ctx, client)
	if err != nil {
		return gosdk.Error("cannot index document", err, nil)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return gosdk.Error("cannot read response body", err, nil)
		}

		return gosdk.Error("cannot index document", err, map[string]any{
			"response": string(body),
			"status":   resp.StatusCode,
		})
	}

	return nil
}
