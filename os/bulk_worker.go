package os

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/threatwinds/go-sdk/catcher"
)

// worker runs the background process that sends bulk requests at configured intervals.
func (bq *BulkQueue) worker() {
	defer bq.wg.Done()

	for {
		select {
		case <-bq.ticker.C:
			_ = bq.processBulk()
		case <-bq.stopCh:
			// Process any remaining items before stopping
			_ = bq.processBulk()
			return
		}
	}
}

// processBulk processes all items in the queue and sends them to OpenSearch.
func (bq *BulkQueue) processBulk() error {
	bq.mutex.Lock()
	if len(bq.queue) == 0 {
		bq.mutex.Unlock()
		return nil
	}

	// Move all items from queue to a local slice (atomic swap)
	items := make([]BulkItem, len(bq.queue))
	copy(items, bq.queue)
	bq.queue = bq.queue[:0]
	bq.mutex.Unlock()

	// Try to send with retries
	var lastErr error
	for attempt := 0; attempt <= bq.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := bq.config.RetryDelay * time.Duration(1<<(attempt-1))
			time.Sleep(delay)
		}

		response, err := bq.sendBulkRequest(items)
		if err != nil {
			lastErr = err
			continue
		}

		// Check for partial failures
		if response.FailedCount > 0 {
			lastErr = catcher.Error("bulk request had partial failures", nil, map[string]any{
				"success_count": response.SuccessCount,
				"failed_count":  response.FailedCount,
			})

			// Extract failed items for callback
			if bq.config.OnError != nil {
				failedItems := make([]BulkItem, 0, response.FailedCount)
				for _, bulkErr := range response.Errors {
					if bulkErr.Index < len(items) {
						failedItems = append(failedItems, items[bulkErr.Index])
					}
				}
				bq.config.OnError(failedItems, lastErr)
			}
		}

		// Call success callback
		if bq.config.OnSuccess != nil && response.SuccessCount > 0 {
			bq.config.OnSuccess(response.SuccessCount, response.IndexCounts)
		}

		return lastErr
	}

	// All retries exhausted
	if bq.config.OnError != nil {
		bq.config.OnError(items, lastErr)
	}

	return lastErr
}

// sendBulkRequest sends a bulk request to OpenSearch and returns the response.
func (bq *BulkQueue) sendBulkRequest(items []BulkItem) (*BulkResponse, error) {
	if len(items) == 0 {
		return &BulkResponse{}, nil
	}

	var body strings.Builder
	indexCounts := make(map[string]int)

	for i, item := range items {
		indexCounts[item.Index]++

		action, err := buildBulkAction(item)
		if err != nil {
			return nil, catcher.Error("failed to build bulk action", err, map[string]any{
				"item_index": i,
				"index":      item.Index,
				"operation":  item.Operation,
			})
		}

		actionBytes, err := json.Marshal(action)
		if err != nil {
			return nil, catcher.Error("failed to marshal action", err, map[string]any{
				"item_index": i,
				"index":      item.Index,
			})
		}
		body.WriteString(string(actionBytes) + "\n")

		// Delete operations don't have a document body
		if item.Operation != BulkOperationDelete {
			docBytes, err := marshalDocument(item)
			if err != nil {
				return nil, catcher.Error("failed to marshal document", err, map[string]any{
					"item_index": i,
					"index":      item.Index,
				})
			}
			body.WriteString(string(docBytes) + "\n")
		}
	}

	req := opensearchapi.BulkReq{
		Body: strings.NewReader(body.String()),
	}

	resp, err := bq.client.Bulk(context.Background(), req)
	if err != nil {
		return nil, catcher.Error("bulk request failed", err, map[string]any{
			"items_count":  len(items),
			"index_counts": indexCounts,
		})
	}

	response := &BulkResponse{
		IndexCounts: indexCounts,
	}

	if resp.Errors {
		response.Errors = extractBulkErrors(resp, items)
		response.FailedCount = len(response.Errors)
		response.SuccessCount = len(items) - response.FailedCount
	} else {
		response.SuccessCount = len(items)
	}

	if response.FailedCount == 0 {
		catcher.Info("Successfully processed bulk request", map[string]any{
			"items_count":  len(items),
			"index_counts": indexCounts,
		})
	} else {
		catcher.Info("Bulk request completed with errors", map[string]any{
			"items_count":   len(items),
			"success_count": response.SuccessCount,
			"failed_count":  response.FailedCount,
			"index_counts":  indexCounts,
		})
	}

	return response, nil
}

// buildBulkAction creates the action metadata for a bulk item.
func buildBulkAction(item BulkItem) (map[string]any, error) {
	actionMeta := map[string]any{
		"_index": item.Index,
	}

	if item.DocumentID != "" {
		actionMeta["_id"] = item.DocumentID
	}

	if item.Routing != "" {
		actionMeta["routing"] = item.Routing
	}

	return map[string]any{
		string(item.Operation): actionMeta,
	}, nil
}

// marshalDocument marshals the document for a bulk item.
func marshalDocument(item BulkItem) ([]byte, error) {
	// For update operations, wrap in "doc" if not already wrapped
	if item.Operation == BulkOperationUpdate {
		if m, ok := item.Document.(map[string]any); ok {
			// Check if already wrapped
			if _, hasDoc := m["doc"]; !hasDoc {
				if _, hasScript := m["script"]; !hasScript {
					// Wrap in "doc"
					return json.Marshal(map[string]any{"doc": item.Document})
				}
			}
		}
	}

	return json.Marshal(item.Document)
}

// extractBulkErrors extracts error details from the bulk response.
func extractBulkErrors(resp *opensearchapi.BulkResp, items []BulkItem) []BulkItemError {
	var errors []BulkItemError

	for i, responseItem := range resp.Items {
		for operation, itemResp := range responseItem {
			if itemResp.Error != nil {
				bulkErr := BulkItemError{
					Index:         i,
					Operation:     operation,
					DocumentIndex: itemResp.Index,
					DocumentID:    itemResp.ID,
					Status:        itemResp.Status,
					ErrorType:     itemResp.Error.Type,
					ErrorReason:   itemResp.Error.Reason,
				}

				if itemResp.Error.Cause.Type != "" {
					bulkErr.CauseType = itemResp.Error.Cause.Type
					bulkErr.CauseReason = itemResp.Error.Cause.Reason
				}

				errors = append(errors, bulkErr)
			}
		}
	}

	return errors
}
