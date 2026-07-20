package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// MLModelConfig contains configuration for registering an ML model
type MLModelConfig struct {
	Name         string `json:"name"`
	Version      string `json:"version,omitempty"`
	ModelFormat  string `json:"model_format,omitempty"` // TORCH_SCRIPT, ONNX
	ModelGroupID string `json:"model_group_id,omitempty"`
	Description  string `json:"description,omitempty"`
	FunctionName string `json:"function_name,omitempty"` // TEXT_EMBEDDING, etc.
	// For pre-trained models from model hub
	ModelContentHashValue string `json:"model_content_hash_value,omitempty"`
	URL                   string `json:"url,omitempty"`
	// ModelConfig contains model-specific configuration required for pre-trained models
	// This includes embedding_dimension, model_type, framework_type, etc.
	ModelConfig *MLModelInnerConfig `json:"model_config,omitempty"`
	// For remote/connector-based models
	ConnectorID string `json:"connector_id,omitempty"`
}

// MLModelInnerConfig contains model-specific configuration for pre-trained models
type MLModelInnerConfig struct {
	// ModelType is the type of model (e.g., "bert", "distilbert")
	ModelType string `json:"model_type,omitempty"`
	// EmbeddingDimension is the output dimension of the embedding model
	EmbeddingDimension int `json:"embedding_dimension,omitempty"`
	// FrameworkType is the framework used (e.g., "sentence_transformers", "huggingface_transformers")
	FrameworkType string `json:"framework_type,omitempty"`
	// AllConfig is a JSON string containing additional model configuration
	AllConfig string `json:"all_config,omitempty"`
}

// MLModelInfo contains information about a registered ML model
type MLModelInfo struct {
	ModelID                 string `json:"model_id"`
	Name                    string `json:"name"`
	ModelGroupID            string `json:"model_group_id,omitempty"`
	Algorithm               string `json:"algorithm,omitempty"`
	ModelVersion            string `json:"model_version,omitempty"`
	ModelFormat             string `json:"model_format,omitempty"`
	ModelState              string `json:"model_state"` // DEPLOYED, REGISTERED, DEPLOYING, DEPLOY_FAILED, etc.
	ModelContentSizeInBytes int64  `json:"model_content_size_in_bytes,omitempty"`
	Description             string `json:"description,omitempty"`
	FunctionName            string `json:"function_name,omitempty"`
	CreatedTime             int64  `json:"created_time,omitempty"`
	LastUpdatedTime         int64  `json:"last_updated_time,omitempty"`
}

// MLModelGroup represents a model group for organizing models
type MLModelGroup struct {
	ModelGroupID  string `json:"model_group_id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	LatestVersion int    `json:"latest_version,omitempty"`
	Access        string `json:"access,omitempty"` // public, private, restricted
	CreatedTime   int64  `json:"created_time,omitempty"`
}

// MLTaskInfo contains information about an async ML task
type MLTaskInfo struct {
	TaskID         string `json:"task_id"`
	TaskType       string `json:"task_type,omitempty"`
	State          string `json:"state"` // CREATED, RUNNING, COMPLETED, FAILED, CANCELLED
	ModelID        string `json:"model_id,omitempty"`
	Error          string `json:"error,omitempty"`
	CreateTime     int64  `json:"create_time,omitempty"`
	LastUpdateTime int64  `json:"last_update_time,omitempty"`
}

// MLModelState constants
const (
	MLModelStateDeployed     = "DEPLOYED"
	MLModelStateRegistered   = "REGISTERED"
	MLModelStateDeploying    = "DEPLOYING"
	MLModelStateDeployFailed = "DEPLOY_FAILED"
	MLModelStateUndeployed   = "UNDEPLOYED"
)

// MLTaskState constants
const (
	MLTaskStateCreated   = "CREATED"
	MLTaskStateRunning   = "RUNNING"
	MLTaskStateCompleted = "COMPLETED"
	MLTaskStateFailed    = "FAILED"
	MLTaskStateCancelled = "CANCELLED"
)

// CreateMLModelGroup creates a new model group for organizing ML models
func CreateMLModelGroup(ctx context.Context, name, description string) (string, error) {
	body := map[string]interface{}{
		"name":        name,
		"description": description,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal model group request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "/_plugins/_ml/model_groups/_register", bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return "", fmt.Errorf("failed to create model group: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create model group (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		ModelGroupID string `json:"model_group_id"`
		Status       string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ModelGroupID, nil
}

// GetMLModelGroup retrieves a model group by ID
func GetMLModelGroup(ctx context.Context, groupID string) (*MLModelGroup, error) {
	path := fmt.Sprintf("/_plugins/_ml/model_groups/%s", groupID)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get model group: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get model group (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var group MLModelGroup
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &group, nil
}

// RegisterMLModel registers a new ML model
// Returns the model ID and task ID for async registration
func RegisterMLModel(ctx context.Context, cfg *MLModelConfig) (modelID string, taskID string, err error) {
	bodyJSON, err := json.Marshal(cfg)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal model config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "/_plugins/_ml/models/_register", bytes.NewReader(bodyJSON))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to register model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to register model (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		ModelID string `json:"model_id"`
		TaskID  string `json:"task_id"`
		Status  string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ModelID, result.TaskID, nil
}

// DeployMLModel deploys a registered model
// Returns the task ID for async deployment
func DeployMLModel(ctx context.Context, modelID string) (string, error) {
	path := fmt.Sprintf("/_plugins/_ml/models/%s/_deploy", modelID)
	req, err := http.NewRequestWithContext(ctx, "POST", path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return "", fmt.Errorf("failed to deploy model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to deploy model (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.TaskID, nil
}

// UndeployMLModel undeploys a deployed model
func UndeployMLModel(ctx context.Context, modelID string) error {
	path := fmt.Sprintf("/_plugins/_ml/models/%s/_undeploy", modelID)
	req, err := http.NewRequestWithContext(ctx, "POST", path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to undeploy model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to undeploy model (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetMLModel retrieves model information by ID
func GetMLModel(ctx context.Context, modelID string) (*MLModelInfo, error) {
	path := fmt.Sprintf("/_plugins/_ml/models/%s", modelID)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get model (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var model MLModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&model); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &model, nil
}

// SearchMLModels searches for models by criteria
func SearchMLModels(ctx context.Context, query map[string]interface{}) ([]*MLModelInfo, error) {
	bodyJSON, err := json.Marshal(map[string]interface{}{
		"query": query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "/_plugins/_ml/models/_search", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to search models (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source MLModelInfo `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]*MLModelInfo, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		model := hit.Source
		models[i] = &model
	}

	return models, nil
}

// GetMLModelByName retrieves a model by name
func GetMLModelByName(ctx context.Context, name string) (*MLModelInfo, error) {
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"term": map[string]interface{}{
						"name.keyword": name,
					},
				},
			},
		},
	}

	models, err := SearchMLModels(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return nil, nil
	}

	return models[0], nil
}

// GetMLTask retrieves task information by ID
func GetMLTask(ctx context.Context, taskID string) (*MLTaskInfo, error) {
	path := fmt.Sprintf("/_plugins/_ml/tasks/%s", taskID)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get task (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var task MLTaskInfo
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &task, nil
}

// DeleteMLModel deletes a model by ID
func DeleteMLModel(ctx context.Context, modelID string) error {
	path := fmt.Sprintf("/_plugins/_ml/models/%s", modelID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}
	defer resp.Body.Close()

	// Ignore 404 (model doesn't exist)
	if resp.StatusCode == 404 {
		return nil
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete model (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// DeleteMLModelGroup deletes a model group by ID
func DeleteMLModelGroup(ctx context.Context, groupID string) error {
	path := fmt.Sprintf("/_plugins/_ml/model_groups/%s", groupID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to delete model group: %w", err)
	}
	defer resp.Body.Close()

	// Ignore 404 (model group doesn't exist)
	if resp.StatusCode == 404 {
		return nil
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete model group (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
