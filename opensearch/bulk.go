package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

type BulkItem struct {
	Index  string
	Id     string
	Body   []byte
	Action string
}

type Header struct {
	Index *string `json:"_index,omitempty"`
	Id    *string `json:"_id,omitempty"`
}

type IndexAction struct {
	Index *Header `json:"index,omitempty"`
}

// Bulk sends a bulk request to the OpenSearch server with the provided items.
// It takes a context and a slice of BulkItem as parameters and returns an error if any occurs.
//
// The BulkItem struct contains the following fields:
// - Index: The index name where the document will be stored.
// - Id: The document ID.
// - Body: The document content in byte slice format.
// - Action: The action to be performed (e.g., "index").
//
// The function constructs a bulk request by iterating over the items and creating
// the necessary JSON payload for each item based on its action. Currently, only the "index"
// action is supported.
//
// The function then sends the bulk request to the OpenSearch server using the opensearchapi.BulkRequest
// and checks the response status code. If the status code indicates an error, it reads the response
// body and returns an error with the status code and response body.
//
// Parameters:
// - ctx: The context for the request.
// - items: A slice of BulkItem containing the items to be indexed.
//
// Returns:
// - error: An error if any occurs during the request or response processing.
func Bulk(ctx context.Context, items []BulkItem) error {
	req := new(opensearchapi.BulkRequest)

	nd, err := generateNd(items)
	if err != nil {
		return err
	}

	req.Body = strings.NewReader(nd)

	resp, err := req.Do(ctx, client)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("search engine status %d, response: %s", resp.StatusCode, body)
	}

	return nil
}

func generateNd(items []BulkItem) (string, error) {
	var nd string

	for _, item := range items {
		switch item.Action {
		case "index":
			var cl *bytes.Buffer = new(bytes.Buffer)

			err := json.Compact(cl, item.Body)
			if err != nil {
				return nd, err
			}

			aH := IndexAction{
				Index: &Header{
					Index: &item.Index,
					Id:    &item.Id,
				},
			}

			bAH, err := json.Marshal(aH)
			if err != nil {
				return nd, err
			}

			nd += strings.Join([]string{string(bAH), cl.String()}, "\n")+ "\n"
		}
	}

	return nd, nil
}
