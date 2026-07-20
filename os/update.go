package os

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// Update represents an OpenSearch document update request.
type Update struct {
	Doc map[string]any `json:"doc"`
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

// UpdateByQuery performs an update on documents matching a query.
func UpdateByQuery(ctx context.Context, indices []string, body map[string]interface{}) ([]byte, error) {
	j, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	reader := strings.NewReader(string(j))

	req := opensearchapi.UpdateByQueryReq{
		Indices: indices,
		Body:    reader,
	}

	resp, err := apiClient.UpdateByQuery(ctx, req)
	if err != nil {
		return nil, err
	}

	// Read and return the response body which contains stats like "updated", "failures", etc.
	// Since UpdateByQuery can be complex, we return the raw bytes for the caller to parse as needed,
	// or we could define a specific struct if required.
	// Users might want to check for failures or conflicts.
	// The opensearch-go client returns a response that gives us access to the body.

	// Check for HTTP errors (although apiClient usually handles transport errors, we might get 200 with failures)
	// UpdateByQuery is synchronous by default unless waitForCompletion=false is set (not exposed here yet).

	// We return the raw source which is a []byte or similar from the response?
	// opensearchapi.UpdateByQueryResp doesn't strictly have a "Bytes()" method easily accessible?
	// Actually, `resp` is the typed response object, likely parsed.
	// Let's see: apiClient.UpdateByQuery returns (*opensearchapi.UpdateByQueryResp, error).
	// We can simply marshal it back to bytes or return it?
	// To keep it simple and consistent with minimal types, let's return error only?
	// But the user might want to know how many updated.

	// Let's return the *opensearchapi.UpdateByQueryResp if we import it,
	// but we are in package 'os' which imports 'opensearchapi'.
	// Exposing that type in our public API couples us tightly but that's already happening?
	// No, our wrappers usually hide it.
	// Let's define a simple result struct.

	b, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return b, nil
}
