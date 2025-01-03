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
		return gosdk.Error("cannot delete document", err, map[string]any{
			"id": h.ID,
		})
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return gosdk.Error("cannot delete document", err, map[string]any{
				"id": h.ID,
			})
		}

		return gosdk.Error("cannot delete document", nil, map[string]any{
			"id":       h.ID,
			"response": string(body),
			"status":   resp.StatusCode,
		})
	}

	return nil
}
