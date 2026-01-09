package os

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// Update represents an OpenSearch document update request.
type Update struct {
	Doc map[string]interface{} `json:"doc"`
}

// Save updates the document in OpenSearch using the hit's index and ID.
// The document source must be modified before calling Save.
func (h Hit) Save(ctx context.Context) error {
	j, err := json.Marshal(Update{Doc: h.Source})
	if err != nil {
		return err
	}

	reader := strings.NewReader(string(j))

	req := opensearchapi.UpdateReq{
		Index:      h.Index,
		DocumentID: h.ID,
		Body:       reader,
	}

	_, err = apiClient.Update(ctx, req)
	if err != nil {
		return err
	}

	return nil
}
