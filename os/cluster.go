package os

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ClusterHealth holds the fields from the _cluster/health response.
type ClusterHealth struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

// GetClusterHealth returns the cluster health from _cluster/health.
func GetClusterHealth(ctx context.Context) (*ClusterHealth, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/_cluster/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster health request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cluster health request failed: %s", string(b))
	}

	var health ClusterHealth
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode cluster health: %w", err)
	}
	return &health, nil
}
