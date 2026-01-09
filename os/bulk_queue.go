package os

import (
	"time"
)

// NewBulkQueue creates a new BulkQueue that uses the existing singleton connection.
// The singleton must be initialized with Connect() before calling this function.
// Returns nil if the connection hasn't been established.
func NewBulkQueue(config BulkQueueConfig) *BulkQueue {
	if apiClient == nil {
		return nil
	}

	if config.FlushInterval <= 0 {
		config.FlushInterval = 10 * time.Second
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = time.Second
	}

	bq := &BulkQueue{
		client: apiClient,
		config: config,
		queue:  make([]BulkItem, 0),
		ticker: time.NewTicker(config.FlushInterval),
		stopCh: make(chan struct{}),
	}

	bq.wg.Add(1)
	go bq.worker()

	return bq
}

// NewBulkQueueWithDefaults creates a new BulkQueue with default configuration.
// The singleton must be initialized with Connect() before calling this function.
// Returns nil if the connection hasn't been established.
func NewBulkQueueWithDefaults() *BulkQueue {
	return NewBulkQueue(DefaultBulkQueueConfig())
}

// Config returns the current configuration (read-only copy).
func (bq *BulkQueue) Config() BulkQueueConfig {
	return bq.config
}

// IsRunning returns true if the queue worker is still running.
func (bq *BulkQueue) IsRunning() bool {
	select {
	case <-bq.stopCh:
		return false
	default:
		return true
	}
}
