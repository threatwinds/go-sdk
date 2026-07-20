package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// MLPredictRequest represents a request to the ML predict API
type MLPredictRequest struct {
	TextDocs []string `json:"text_docs"`
}

// MLPredictResponse represents the response from the ML predict API
type MLPredictResponse struct {
	InferenceResults []MLInferenceResult `json:"inference_results"`
}

// MLInferenceResult represents a single inference result
type MLInferenceResult struct {
	Output []MLOutputData `json:"output"`
}

// MLOutputData represents the output data from inference
type MLOutputData struct {
	Name     string    `json:"name"`
	DataType string    `json:"data_type"`
	Shape    []int     `json:"shape"`
	Data     []float32 `json:"data"`
}

// MLPredict generates text embeddings for multiple texts using a deployed model
// POST /_plugins/_ml/_predict/text_embedding/{model_id}
func MLPredict(ctx context.Context, modelID string, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	body := MLPredictRequest{
		TextDocs: texts,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal predict request: %w", err)
	}

	path := fmt.Sprintf("/_plugins/_ml/_predict/text_embedding/%s", modelID)
	req, err := http.NewRequestWithContext(ctx, "POST", path, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call predict API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("predict API failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result MLPredictResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract embeddings from inference results
	embeddings := make([][]float32, len(result.InferenceResults))
	for i, inferenceResult := range result.InferenceResults {
		// Look for "sentence_embedding" output first, then fall back to first output
		for _, output := range inferenceResult.Output {
			if output.Name == "sentence_embedding" {
				embeddings[i] = output.Data
				break
			}
		}
		// Fall back to first output if no sentence_embedding found
		if embeddings[i] == nil && len(inferenceResult.Output) > 0 {
			embeddings[i] = inferenceResult.Output[0].Data
		}
	}

	return embeddings, nil
}

// MLPredictSingle generates a text embedding for a single text
func MLPredictSingle(ctx context.Context, modelID, text string) ([]float32, error) {
	embeddings, err := MLPredict(ctx, modelID, []string{text})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for text")
	}

	return embeddings[0], nil
}

// MLPredictBatch generates embeddings for texts in batches
// Useful for large numbers of texts that need to be processed in smaller chunks
func MLPredictBatch(ctx context.Context, modelID string, texts []string, batchSize int) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	if batchSize <= 0 {
		batchSize = 10 // default batch size
	}

	allEmbeddings := make([][]float32, 0, len(texts))

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := MLPredict(ctx, modelID, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to process batch %d-%d: %w", i, end, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}

// MLPredictGeneric performs a generic prediction with custom input
// POST /_plugins/_ml/models/{model_id}/_predict
func MLPredictGeneric(ctx context.Context, modelID string, input map[string]interface{}) (map[string]interface{}, error) {
	bodyJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal predict request: %w", err)
	}

	path := fmt.Sprintf("/_plugins/_ml/models/%s/_predict", modelID)
	req, err := http.NewRequestWithContext(ctx, "POST", path, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call predict API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("predict API failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
