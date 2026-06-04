package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SnapshotRepository describes a snapshot repository registration request, e.g.
// {Type: "fs", Settings: {"location": "/mnt/backups"}}.
type SnapshotRepository struct {
	Type     string         `json:"type"`
	Settings map[string]any `json:"settings"`
}

// RegisterSnapshotRepository registers (creates or updates) a snapshot repository
// via PUT /_snapshot/{name}.
func RegisterSnapshotRepository(ctx context.Context, name string, repo SnapshotRepository) error {
	body, err := json.Marshal(repo)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot repository: %w", err)
	}

	path := fmt.Sprintf("/_snapshot/%s", name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create snapshot repository request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to register snapshot repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register snapshot repository failed: %s", string(b))
	}
	return nil
}

// SnapshotRepositoryExists reports whether a snapshot repository is registered.
func SnapshotRepositoryExists(ctx context.Context, name string) (bool, error) {
	path := fmt.Sprintf("/_snapshot/%s", name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create snapshot repository request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return false, fmt.Errorf("failed to check snapshot repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("check snapshot repository failed: %s", string(b))
	}
	return true, nil
}
