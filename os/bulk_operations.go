package os

// Add adds a single document to the bulk queue with the "index" operation.
// This is the most common operation that creates or replaces a document.
func (bq *BulkQueue) Add(index string, doc any) {
	bq.AddWithID(index, "", doc)
}

// AddWithID adds a single document with a specific ID to the bulk queue.
func (bq *BulkQueue) AddWithID(index, docID string, doc any) {
	bq.AddItem(BulkItem{
		Index:      index,
		DocumentID: docID,
		Operation:  BulkOperationIndex,
		Document:   doc,
	})
}

// AddCreate adds a document using the "create" operation (fails if document exists).
func (bq *BulkQueue) AddCreate(index, docID string, doc any) {
	bq.AddItem(BulkItem{
		Index:      index,
		DocumentID: docID,
		Operation:  BulkOperationCreate,
		Document:   doc,
	})
}

// AddUpdate adds a document update operation.
// The doc should be the partial document or a map with "doc" or "script" fields.
func (bq *BulkQueue) AddUpdate(index, docID string, doc any) {
	bq.AddItem(BulkItem{
		Index:      index,
		DocumentID: docID,
		Operation:  BulkOperationUpdate,
		Document:   doc,
	})
}

// AddDelete adds a delete operation for a document.
func (bq *BulkQueue) AddDelete(index, docID string) {
	bq.AddItem(BulkItem{
		Index:      index,
		DocumentID: docID,
		Operation:  BulkOperationDelete,
	})
}

// AddItem adds a single BulkItem to the queue.
func (bq *BulkQueue) AddItem(item BulkItem) {
	bq.mutex.Lock()
	bq.queue = append(bq.queue, item)
	shouldFlush := bq.config.FlushThreshold > 0 && len(bq.queue) >= bq.config.FlushThreshold
	bq.mutex.Unlock()

	if shouldFlush {
		go bq.processBulk()
	}
}

// AddBatch adds multiple documents to the same index using the "index" operation.
func (bq *BulkQueue) AddBatch(index string, docs []any) {
	items := make([]BulkItem, len(docs))
	for i, doc := range docs {
		items[i] = BulkItem{
			Index:     index,
			Operation: BulkOperationIndex,
			Document:  doc,
		}
	}
	bq.AddItems(items)
}

// AddBatchWithIDs adds multiple documents with specific IDs to the same index.
func (bq *BulkQueue) AddBatchWithIDs(index string, docs map[string]any) {
	items := make([]BulkItem, 0, len(docs))
	for id, doc := range docs {
		items = append(items, BulkItem{
			Index:      index,
			DocumentID: id,
			Operation:  BulkOperationIndex,
			Document:   doc,
		})
	}
	bq.AddItems(items)
}

// AddItems adds multiple BulkItems to the queue.
func (bq *BulkQueue) AddItems(items []BulkItem) {
	if len(items) == 0 {
		return
	}

	bq.mutex.Lock()
	bq.queue = append(bq.queue, items...)
	shouldFlush := bq.config.FlushThreshold > 0 && len(bq.queue) >= bq.config.FlushThreshold
	bq.mutex.Unlock()

	if shouldFlush {
		go bq.processBulk()
	}
}

// Size returns the current number of items in the queue.
func (bq *BulkQueue) Size() int {
	bq.mutex.RLock()
	defer bq.mutex.RUnlock()
	return len(bq.queue)
}

// Flush immediately processes all items in the queue.
// This is a blocking call that waits for the bulk request to complete.
func (bq *BulkQueue) Flush() error {
	return bq.processBulk()
}

// Stop gracefully stops the bulk queue, flushing any remaining items.
func (bq *BulkQueue) Stop() {
	close(bq.stopCh)
	bq.ticker.Stop()
	bq.wg.Wait()
}

// Clear empties the queue without processing. Use with caution.
func (bq *BulkQueue) Clear() {
	bq.mutex.Lock()
	bq.queue = bq.queue[:0]
	bq.mutex.Unlock()
}
