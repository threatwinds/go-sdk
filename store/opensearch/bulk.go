package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	sdkos "github.com/threatwinds/go-sdk/os"
	"github.com/threatwinds/go-sdk/store"
)

const (
	defaultFlushThreshold = 2000
	defaultFlushInterval  = 10 * time.Second
	defaultMaxRetries     = 3
)

func (d *Driver) BulkWriter(dataset store.Dataset) (store.BulkWriter, error) {
	if _, err := d.cfg.Layout.ReadIndices(store.Scope{Dataset: dataset}); err != nil {
		return nil, err
	}
	w := &bulkWriter{driver: d, dataset: dataset}
	w.queue = sdkos.NewBulkQueue("store_"+string(dataset), sdkos.BulkQueueConfig{
		FlushInterval:  defaultFlushInterval,
		FlushThreshold: defaultFlushThreshold,
		MaxRetries:     defaultMaxRetries,
		RetryDelay:     time.Second,
		OnError: func(failed []sdkos.BulkItem, err error) {
			w.recordFailure(len(failed), err)
		},
	})
	return w, nil
}

type bulkWriter struct {
	driver  *Driver
	dataset store.Dataset
	queue   *sdkos.BulkQueue

	mu       sync.Mutex
	lastErr  error
	lostDocs int
	closed   bool
}

func (w *bulkWriter) Write(s store.Scope, doc []byte) error {
	if s.Tenant == "" {
		return store.ErrNoTenant
	}
	if s.Dataset != w.dataset {
		return fmt.Errorf("opensearch: writer is for %s, got a %s record", w.dataset, s.Dataset)
	}
	if err := w.takeErr(); err != nil {
		return err
	}

	w.mu.Lock()
	closed := w.closed
	w.mu.Unlock()
	if closed {
		return fmt.Errorf("opensearch: bulk writer is closed")
	}

	index, err := w.driver.cfg.Layout.WriteIndex(s, w.driver.now())
	if err != nil {
		return err
	}

	w.queue.AddItem(sdkos.BulkItem{
		Index:     index,
		Operation: sdkos.BulkOperationIndex,
		Document:  json.RawMessage(doc),
	})
	return nil
}

func (w *bulkWriter) Flush(ctx context.Context) error {
	if err := w.queue.Flush(); err != nil {
		return err
	}
	return w.takeErr()
}

func (w *bulkWriter) Close(ctx context.Context) error {
	w.mu.Lock()
	w.closed = true
	w.mu.Unlock()

	w.queue.Stop()
	return w.takeErr()
}

func (w *bulkWriter) recordFailure(n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastErr = err
	w.lostDocs += n
}

// takeErr clears the failure as it reports it, so one failed bulk surfaces once
// instead of poisoning every later call.
func (w *bulkWriter) takeErr() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lastErr == nil {
		return nil
	}
	err := fmt.Errorf("opensearch: bulk write failed for %d document(s): %w", w.lostDocs, w.lastErr)
	w.lastErr, w.lostDocs = nil, 0
	return err
}
