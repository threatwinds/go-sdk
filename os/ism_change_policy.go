package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ISMFailedIndex describes one index that failed to change its ISM policy.
type ISMFailedIndex struct {
	IndexName string `json:"index_name"`
	IndexUUID string `json:"index_uuid"`
	Reason    string `json:"reason"`
}

// ISMChangePolicyResult is the response from _plugins/_ism/change_policy.
type ISMChangePolicyResult struct {
	UpdatedIndices int              `json:"updated_indices"`
	Failures       bool             `json:"failures"`
	FailedIndices  []ISMFailedIndex `json:"failed_indices"`
}

// ChangeISMPolicy changes the managed ISM policy (or transitions the state) for
// the indices matching index via POST _plugins/_ism/change_policy/{index}.
//
// body is the change request, e.g.
// {"policy_id": "...", "state": "...", "include": [{"state": "..."}]}.
func ChangeISMPolicy(ctx context.Context, index string, body any) (*ISMChangePolicyResult, error) {
	j, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal change_policy body: %w", err)
	}

	path := fmt.Sprintf("/_plugins/_ism/change_policy/%s", index)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(j))
	if err != nil {
		return nil, fmt.Errorf("failed to create change_policy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to change ISM policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("change ISM policy failed: %s", string(b))
	}

	var result ISMChangePolicyResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode change_policy response: %w", err)
	}
	return &result, nil
}
