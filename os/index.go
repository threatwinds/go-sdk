package os

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// IndexDoc indexes a document in OpenSearch.
// It takes a document, an index name, and an ID as input parameters.
// The document is marshalled to JSON and sent to OpenSearch for indexing.
// Returns an error if there is an issue with marshalling the document to JSON,
// if there is an issue with the request to OpenSearch, or if the response status code is not 200, 201, or 202.
func IndexDoc(ctx context.Context, doc interface{}, index, id string) error {
	j, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	reader := strings.NewReader(string(j))

	req := opensearchapi.IndexReq{
		Index:      index,
		Body:       reader,
		DocumentID: id,
		Params: opensearchapi.IndexParams{
			OpType: "create",
		},
	}

	_, err = apiClient.Index(ctx, req)
	if err != nil {
		return err
	}

	return nil
}
