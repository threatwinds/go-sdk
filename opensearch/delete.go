package opensearch

import (
	"context"
	gosdk "github.com/threatwinds/go-sdk"
	"io"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

// Delete deletes the document from the OpenSearch index.
func (h Hit) Delete(ctx context.Context) error {
	req := opensearchapi.DeleteRequest{
		Index:      h.Index,
		DocumentID: h.ID,
	}

	resp, err := req.Do(ctx, client)
	if err != nil {
		return gosdk.Error(gosdk.Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "failed to delete document",
		})
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return gosdk.Error(gosdk.Trace(), map[string]interface{}{
				"cause": err.Error(),
				"error": "failed to read response body",
			})
		}

		return gosdk.Error(gosdk.Trace(), map[string]interface{}{
			"response":   string(body),
			"error":      "failed to delete document",
			"statusCode": resp.StatusCode,
		})
	}

	return nil
}
