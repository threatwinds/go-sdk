package os

import (
	"sync"
	"time"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// BulkOperation represents the type of bulk operation to perform.
type BulkOperation string

const (
	// BulkOperationCreate creates a document (fails if exists).
	BulkOperationCreate BulkOperation = "create"
	// BulkOperationIndex creates or replaces a document.
	BulkOperationIndex BulkOperation = "index"
	// BulkOperationUpdate updates an existing document.
	BulkOperationUpdate BulkOperation = "update"
	// BulkOperationDelete deletes a document.
	BulkOperationDelete BulkOperation = "delete"
)

// BulkItem represents a single item in the bulk queue.
type BulkItem struct {
	// Index is the target index name.
	Index string
	// DocumentID is optional; if empty, OpenSearch auto-generates one.
	DocumentID string
	// Operation is the bulk operation type (create, index, update, delete).
	Operation BulkOperation
	// Document is the document data (not used for delete operations).
	Document any
	// Routing is optional routing value for the document.
	Routing string
}

// BulkQueueConfig holds configuration for the BulkQueue.
type BulkQueueConfig struct {
	// FlushInterval is how often the queue automatically flushes (default: 10s).
	FlushInterval time.Duration
	// FlushThreshold is the number of items that triggers an automatic flush (default: 0 = disabled).
	FlushThreshold int
	// MaxRetries is the number of times to retry failed bulk requests (default: 0 = no retry).
	MaxRetries int
	// RetryDelay is the base delay between retries with exponential backoff (default: 1s).
	RetryDelay time.Duration
	// OnError is an optional callback for handling bulk errors.
	OnError func(failedItems []BulkItem, err error)
	// OnSuccess is an optional callback when a bulk request succeeds.
	OnSuccess func(successCount int, indexCounts map[string]int)
}

// DefaultBulkQueueConfig returns a BulkQueueConfig with sensible defaults.
func DefaultBulkQueueConfig() BulkQueueConfig {
	return BulkQueueConfig{
		FlushInterval:  10 * time.Second,
		FlushThreshold: 0,
		MaxRetries:     0,
		RetryDelay:     time.Second,
	}
}

// BulkQueue handles bulk operations with automatic batching and flushing.
type BulkQueue struct {
	client      *opensearchapi.Client
	config      BulkQueueConfig
	queue       []BulkItem
	mutex       sync.RWMutex
	ticker      *time.Ticker
	stopCh      chan struct{}
	wg          sync.WaitGroup
	processName string
}

// BulkResponse contains the result of a bulk operation.
type BulkResponse struct {
	// SuccessCount is the number of successfully processed items.
	SuccessCount int
	// FailedCount is the number of failed items.
	FailedCount int
	// IndexCounts maps index names to the number of documents indexed in each.
	IndexCounts map[string]int
	// Errors contains details about failed items.
	Errors []BulkItemError
}

// BulkItemError contains error details for a failed bulk item.
type BulkItemError struct {
	// Index is the original item index in the batch.
	Index int
	// Operation is the operation that failed.
	Operation string
	// DocumentIndex is the target index name.
	DocumentIndex string
	// DocumentID is the document ID (if provided).
	DocumentID string
	// Status is the HTTP status code.
	Status int
	// ErrorType is the OpenSearch error type.
	ErrorType string
	// ErrorReason is the human-readable error reason.
	ErrorReason string
	// CauseType is the root cause error type (if available).
	CauseType string
	// CauseReason is the root cause error reason (if available).
	CauseReason string
}
