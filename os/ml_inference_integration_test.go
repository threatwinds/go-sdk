package os

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestMLInferenceIntegration tests the ML Commons inference API with real OpenSearch
// This test requires an OpenSearch instance running with ML Commons plugin and a deployed model
func TestMLInferenceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Connect to test OpenSearch using env vars
	nodes := os.Getenv("NODES")
	if nodes == "" {
		t.Skip("Skipping integration test: NODES env var not set")
	}
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Skip("Skipping integration test: could not connect to OpenSearch:", err)
	}

	ctx := context.Background()

	// Check if a model ID is provided for testing
	modelID := os.Getenv("ML_MODEL_ID")
	if modelID == "" {
		t.Skip("Skipping inference test: ML_MODEL_ID env var not set")
	}

	t.Run("SingleTextEmbedding", func(t *testing.T) {
		text := "This is a test sentence for embedding generation."
		embedding, err := MLPredictSingle(ctx, modelID, text)
		if err != nil {
			t.Fatalf("failed to generate embedding: %v", err)
		}

		if len(embedding) == 0 {
			t.Error("embedding is empty")
		}

		t.Logf("Generated embedding with %d dimensions", len(embedding))
	})

	t.Run("BatchTextEmbeddings", func(t *testing.T) {
		texts := []string{
			"First test sentence.",
			"Second test sentence.",
			"Third test sentence.",
		}

		embeddings, err := MLPredict(ctx, modelID, texts)
		if err != nil {
			t.Fatalf("failed to generate embeddings: %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Errorf("expected %d embeddings, got %d", len(texts), len(embeddings))
		}

		for i, emb := range embeddings {
			if len(emb) == 0 {
				t.Errorf("embedding %d is empty", i)
			}
		}

		t.Logf("Generated %d embeddings", len(embeddings))
	})

	t.Run("EmptyTextList", func(t *testing.T) {
		embeddings, err := MLPredict(ctx, modelID, []string{})
		if err != nil {
			t.Fatalf("failed with empty list: %v", err)
		}

		if len(embeddings) != 0 {
			t.Errorf("expected 0 embeddings, got %d", len(embeddings))
		}
	})
}

// TestMLModelManagementIntegration tests model group and model operations
func TestMLModelManagementIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	nodes := os.Getenv("NODES")
	if nodes == "" {
		t.Skip("Skipping integration test: NODES env var not set")
	}
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Skip("Skipping integration test: could not connect to OpenSearch:", err)
	}

	ctx := context.Background()

	t.Run("CreateAndDeleteModelGroup", func(t *testing.T) {
		groupName := "test-group-" + time.Now().Format("20060102150405")
		groupID, err := CreateMLModelGroup(ctx, groupName, "Test model group for integration tests")
		if err != nil {
			t.Fatalf("failed to create model group: %v", err)
		}

		t.Logf("Created model group: %s", groupID)

		// Get the model group
		group, err := GetMLModelGroup(ctx, groupID)
		if err != nil {
			t.Fatalf("failed to get model group: %v", err)
		}

		if group == nil {
			t.Fatal("model group not found")
		}

		if group.Name != groupName {
			t.Errorf("expected name %s, got %s", groupName, group.Name)
		}

		// Clean up
		if err := DeleteMLModelGroup(ctx, groupID); err != nil {
			t.Errorf("failed to delete model group: %v", err)
		}
	})

	t.Run("GetNonexistentModelGroup", func(t *testing.T) {
		group, err := GetMLModelGroup(ctx, "nonexistent-group-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if group != nil {
			t.Error("expected nil for nonexistent group")
		}
	})

	t.Run("GetNonexistentModel", func(t *testing.T) {
		model, err := GetMLModel(ctx, "nonexistent-model-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if model != nil {
			t.Error("expected nil for nonexistent model")
		}
	})

	t.Run("SearchModels", func(t *testing.T) {
		// Search for all models
		query := map[string]interface{}{
			"match_all": map[string]interface{}{},
		}

		models, err := SearchMLModels(ctx, query)
		if err != nil {
			t.Fatalf("failed to search models: %v", err)
		}

		t.Logf("Found %d models", len(models))
	})
}

// TestMLTaskIntegration tests task status checking
func TestMLTaskIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	nodes := os.Getenv("NODES")
	if nodes == "" {
		t.Skip("Skipping integration test: NODES env var not set")
	}
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Skip("Skipping integration test: could not connect to OpenSearch:", err)
	}

	ctx := context.Background()

	t.Run("GetNonexistentTask", func(t *testing.T) {
		task, err := GetMLTask(ctx, "nonexistent-task-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task != nil {
			t.Error("expected nil for nonexistent task")
		}
	})
}
