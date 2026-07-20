package os

import (
	"encoding/json"
	"testing"
)

func TestMLModelConfig_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		config   MLModelConfig
		expected map[string]interface{}
	}{
		{
			name: "basic config",
			config: MLModelConfig{
				Name:         "test-model",
				Version:      "1.0.0",
				FunctionName: "TEXT_EMBEDDING",
			},
			expected: map[string]interface{}{
				"name":          "test-model",
				"version":       "1.0.0",
				"function_name": "TEXT_EMBEDDING",
			},
		},
		{
			name: "with model group",
			config: MLModelConfig{
				Name:         "test-model",
				ModelGroupID: "group-123",
				Description:  "Test model description",
			},
			expected: map[string]interface{}{
				"name":           "test-model",
				"model_group_id": "group-123",
				"description":    "Test model description",
			},
		},
		{
			name: "with URL for remote model",
			config: MLModelConfig{
				Name:        "remote-model",
				URL:         "https://example.com/model.zip",
				ModelFormat: "TORCH_SCRIPT",
			},
			expected: map[string]interface{}{
				"name":         "remote-model",
				"url":          "https://example.com/model.zip",
				"model_format": "TORCH_SCRIPT",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("failed to marshal config: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			for key, expectedValue := range tt.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("missing key %q in result", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("key %q: expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestMLModelInfo_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"model_id": "abc123",
		"name": "test-model",
		"model_state": "DEPLOYED",
		"algorithm": "TEXT_EMBEDDING",
		"model_version": "1.0.0",
		"model_format": "TORCH_SCRIPT",
		"function_name": "TEXT_EMBEDDING"
	}`

	var info MLModelInfo
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if info.ModelID != "abc123" {
		t.Errorf("ModelID: expected abc123, got %s", info.ModelID)
	}
	if info.Name != "test-model" {
		t.Errorf("Name: expected test-model, got %s", info.Name)
	}
	if info.ModelState != MLModelStateDeployed {
		t.Errorf("ModelState: expected %s, got %s", MLModelStateDeployed, info.ModelState)
	}
	if info.Algorithm != "TEXT_EMBEDDING" {
		t.Errorf("Algorithm: expected TEXT_EMBEDDING, got %s", info.Algorithm)
	}
}

func TestMLTaskInfo_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"task_id": "task-123",
		"task_type": "REGISTER_MODEL",
		"state": "COMPLETED",
		"model_id": "model-456"
	}`

	var info MLTaskInfo
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if info.TaskID != "task-123" {
		t.Errorf("TaskID: expected task-123, got %s", info.TaskID)
	}
	if info.State != MLTaskStateCompleted {
		t.Errorf("State: expected %s, got %s", MLTaskStateCompleted, info.State)
	}
	if info.ModelID != "model-456" {
		t.Errorf("ModelID: expected model-456, got %s", info.ModelID)
	}
}

func TestMLModelGroup_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"model_group_id": "group-789",
		"name": "test-group",
		"description": "Test model group",
		"latest_version": 3,
		"access": "public"
	}`

	var group MLModelGroup
	if err := json.Unmarshal([]byte(jsonData), &group); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if group.ModelGroupID != "group-789" {
		t.Errorf("ModelGroupID: expected group-789, got %s", group.ModelGroupID)
	}
	if group.Name != "test-group" {
		t.Errorf("Name: expected test-group, got %s", group.Name)
	}
	if group.LatestVersion != 3 {
		t.Errorf("LatestVersion: expected 3, got %d", group.LatestVersion)
	}
}

func TestMLModelStateConstants(t *testing.T) {
	// Verify state constants match OpenSearch ML Commons values
	if MLModelStateDeployed != "DEPLOYED" {
		t.Errorf("MLModelStateDeployed: expected DEPLOYED, got %s", MLModelStateDeployed)
	}
	if MLModelStateRegistered != "REGISTERED" {
		t.Errorf("MLModelStateRegistered: expected REGISTERED, got %s", MLModelStateRegistered)
	}
	if MLModelStateDeploying != "DEPLOYING" {
		t.Errorf("MLModelStateDeploying: expected DEPLOYING, got %s", MLModelStateDeploying)
	}
}

func TestMLTaskStateConstants(t *testing.T) {
	// Verify task state constants match OpenSearch ML Commons values
	if MLTaskStateCompleted != "COMPLETED" {
		t.Errorf("MLTaskStateCompleted: expected COMPLETED, got %s", MLTaskStateCompleted)
	}
	if MLTaskStateFailed != "FAILED" {
		t.Errorf("MLTaskStateFailed: expected FAILED, got %s", MLTaskStateFailed)
	}
	if MLTaskStateRunning != "RUNNING" {
		t.Errorf("MLTaskStateRunning: expected RUNNING, got %s", MLTaskStateRunning)
	}
}
