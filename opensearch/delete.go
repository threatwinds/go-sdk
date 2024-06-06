package opensearch

import (
	"context"
	"fmt"
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
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("search engine status %d, response: %s", resp.StatusCode, body)
	}

	return nil
}
