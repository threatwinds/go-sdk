package os

import (
	"context"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// Delete removes the document from OpenSearch using the hit's index and ID.
func (h Hit) Delete(ctx context.Context) error {
	req := opensearchapi.DocumentDeleteReq{
		Index:      h.Index,
		DocumentID: h.ID,
	}

	_, err := apiClient.Document.Delete(ctx, req)
	if err != nil {
		return err
	}

	return nil
}
