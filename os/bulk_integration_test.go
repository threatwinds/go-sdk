package os

import (
	"context"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// getTestNodes returns the OpenSearch node addresses for testing.
// Uses NODES env var, defaults to http://10.128.0.3:9200 for integration tests.
func getTestNodes() string {
	if nodes := os.Getenv("NODES"); nodes != "" {
		if strings.HasPrefix(nodes, "https://") {
			nodes = strings.Replace(nodes, "https://", "http://", 1)
		}
		if !strings.HasPrefix(nodes, "http://") && !strings.HasPrefix(nodes, "https://") {
			nodes = "http://" + nodes
		}
		return nodes
	}
	return "http://10.128.0.3:9200"
}

// refreshIndex forces OpenSearch to refresh the index so documents are searchable.
func refreshIndex(ctx context.Context, index string) {
	_, _ = apiClient.Indices.Refresh(ctx, &opensearchapi.IndicesRefreshReq{
		Indices: []string{index},
	})
}

// =============================================================================
// Integration Tests (require OpenSearch)
// =============================================================================

func TestBulkQueueIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	nodes := getTestNodes()
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Skipf("Skipping integration test: could not connect to OpenSearch: %v", err)
	}

	ctx := context.Background()
	testIndex := "twos-bulk-test-" + time.Now().Format("20060102-150405")

	// Cleanup at end
	defer func() {
		_, _ = apiClient.Indices.Delete(ctx, opensearchapi.IndicesDeleteReq{
			Indices: []string{testIndex},
		})
	}()

	t.Run("CreateQueueWithDefaults", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		if !queue.IsRunning() {
			t.Error("Expected queue to be running")
		}

		cfg := queue.Config()
		if cfg.FlushInterval != 10*time.Second {
			t.Errorf("Expected 10s flush interval, got %v", cfg.FlushInterval)
		}
	})

	t.Run("CreateQueueWithCustomConfig", func(t *testing.T) {
		var successCount atomic.Int32

		queue := NewBulkQueue("testing", BulkQueueConfig{
			FlushInterval:  1 * time.Second,
			FlushThreshold: 10,
			MaxRetries:     2,
			RetryDelay:     100 * time.Millisecond,
			OnSuccess: func(count int, indexCounts map[string]int) {
				successCount.Add(int32(count))
			},
		})
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		cfg := queue.Config()
		if cfg.FlushInterval != 1*time.Second {
			t.Errorf("Expected 1s flush interval, got %v", cfg.FlushInterval)
		}
		if cfg.FlushThreshold != 10 {
			t.Errorf("Expected threshold 10, got %d", cfg.FlushThreshold)
		}
	})

	t.Run("AddSingleDocument", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		doc := map[string]any{
			"message":   "test document",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"visibleBy": []string{"public"},
		}

		queue.Add(testIndex, doc)

		if queue.Size() != 1 {
			t.Errorf("Expected queue size 1, got %d", queue.Size())
		}

		// Flush and verify
		err := queue.Flush()
		if err != nil {
			t.Errorf("Flush failed: %v", err)
		}

		if queue.Size() != 0 {
			t.Errorf("Expected queue size 0 after flush, got %d", queue.Size())
		}

		// Wait for indexing
		time.Sleep(1 * time.Second)
	})

	t.Run("AddDocumentWithID", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		doc := map[string]any{
			"message":   "document with ID",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"visibleBy": []string{"public"},
		}

		queue.AddWithID(testIndex, "test-doc-1", doc)
		queue.Flush()

		// Wait for indexing
		time.Sleep(1 * time.Second)

		// Verify document exists
		req := SearchRequest{
			Query: &Query{
				Bool: &Bool{
					Filter: []Query{
						{IDs: map[string][]interface{}{"values": {"test-doc-1"}}},
					},
				},
			},
			Size: 1,
		}

		resp, err := req.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Hits.Total.Value != 1 {
			t.Errorf("Expected 1 document, got %d", resp.Hits.Total.Value)
		}
	})

	t.Run("AddBatch", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		docs := make([]any, 10)
		for i := 0; i < 10; i++ {
			docs[i] = map[string]any{
				"batch_id":  "batch-1",
				"item":      i,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"visibleBy": []string{"public"},
			}
		}

		queue.AddBatch(testIndex, docs)

		if queue.Size() != 10 {
			t.Errorf("Expected queue size 10, got %d", queue.Size())
		}

		queue.Flush()

		// Wait for indexing and force refresh
		time.Sleep(500 * time.Millisecond)
		refreshIndex(ctx, testIndex)

		// Verify documents exist (use .keyword for exact match on dynamic string fields)
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")
		query := builder.
			Term("batch_id.keyword", "batch-1").
			Size(20).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Hits.Total.Value != 10 {
			t.Errorf("Expected 10 documents, got %d", resp.Hits.Total.Value)
		}
	})

	t.Run("AddBatchWithIDs", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		docs := map[string]any{
			"id-a": map[string]any{"name": "doc-a", "visibleBy": []string{"public"}},
			"id-b": map[string]any{"name": "doc-b", "visibleBy": []string{"public"}},
			"id-c": map[string]any{"name": "doc-c", "visibleBy": []string{"public"}},
		}

		queue.AddBatchWithIDs(testIndex, docs)
		queue.Flush()

		// Wait for indexing
		time.Sleep(1 * time.Second)

		// Verify specific IDs exist
		req := SearchRequest{
			Query: &Query{
				Bool: &Bool{
					Filter: []Query{
						{IDs: map[string][]interface{}{"values": {"id-a", "id-b", "id-c"}}},
					},
				},
			},
			Size: 10,
		}

		resp, err := req.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Hits.Total.Value != 3 {
			t.Errorf("Expected 3 documents, got %d", resp.Hits.Total.Value)
		}
	})

	t.Run("AddItems", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		items := []BulkItem{
			{
				Index:      testIndex,
				DocumentID: "item-1",
				Operation:  BulkOperationIndex,
				Document:   map[string]any{"type": "item", "visibleBy": []string{"public"}},
			},
			{
				Index:      testIndex,
				DocumentID: "item-2",
				Operation:  BulkOperationIndex,
				Document:   map[string]any{"type": "item", "visibleBy": []string{"public"}},
			},
		}

		queue.AddItems(items)

		if queue.Size() != 2 {
			t.Errorf("Expected queue size 2, got %d", queue.Size())
		}

		queue.Flush()

		// Wait for indexing
		time.Sleep(1 * time.Second)
	})

	t.Run("CreateOperation", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		doc := map[string]any{
			"message":   "created document",
			"visibleBy": []string{"public"},
		}

		// First create should succeed
		queue.AddCreate(testIndex, "create-test-1", doc)
		queue.Flush()

		// Wait for indexing
		time.Sleep(1 * time.Second)

		// Verify document exists
		req := SearchRequest{
			Query: &Query{
				Bool: &Bool{
					Filter: []Query{
						{IDs: map[string][]interface{}{"values": {"create-test-1"}}},
					},
				},
			},
			Size: 1,
		}

		resp, err := req.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Hits.Total.Value != 1 {
			t.Errorf("Expected 1 document, got %d", resp.Hits.Total.Value)
		}
	})

	t.Run("UpdateOperation", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		// First create a document
		doc := map[string]any{
			"message":   "original message",
			"counter":   1,
			"visibleBy": []string{"public"},
		}
		queue.AddWithID(testIndex, "update-test-1", doc)
		queue.Flush()
		time.Sleep(1 * time.Second)

		// Now update it
		updateDoc := map[string]any{
			"message": "updated message",
			"counter": 2,
		}
		queue.AddUpdate(testIndex, "update-test-1", updateDoc)
		queue.Flush()
		time.Sleep(1 * time.Second)

		// Verify update
		req := SearchRequest{
			Query: &Query{
				Bool: &Bool{
					Filter: []Query{
						{IDs: map[string][]interface{}{"values": {"update-test-1"}}},
					},
				},
			},
			Size: 1,
		}

		resp, err := req.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Hits.Total.Value != 1 {
			t.Errorf("Expected 1 document, got %d", resp.Hits.Total.Value)
		}

		if len(resp.Hits.Hits) > 0 {
			var source map[string]any
			err := resp.Hits.Hits[0].Source.ParseSource(&source)
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}
			if source["message"] != "updated message" {
				t.Errorf("Expected 'updated message', got %v", source["message"])
			}
		}
	})

	t.Run("DeleteOperation", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		// First create a document
		doc := map[string]any{
			"message":   "document to delete",
			"visibleBy": []string{"public"},
		}
		queue.AddWithID(testIndex, "delete-test-1", doc)
		queue.Flush()
		time.Sleep(1 * time.Second)

		// Now delete it
		queue.AddDelete(testIndex, "delete-test-1")
		queue.Flush()
		time.Sleep(1 * time.Second)

		// Verify deletion
		req := SearchRequest{
			Query: &Query{
				Bool: &Bool{
					Filter: []Query{
						{IDs: map[string][]interface{}{"values": {"delete-test-1"}}},
					},
				},
			},
			Size: 1,
		}

		resp, err := req.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Hits.Total.Value != 0 {
			t.Errorf("Expected 0 documents after delete, got %d", resp.Hits.Total.Value)
		}
	})

	t.Run("AutoFlushByInterval", func(t *testing.T) {
		queue := NewBulkQueue("testing", BulkQueueConfig{
			FlushInterval: 500 * time.Millisecond,
		})
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		doc := map[string]any{
			"message":   "auto-flushed document",
			"batch":     "auto-flush-test",
			"visibleBy": []string{"public"},
		}

		queue.Add(testIndex, doc)

		if queue.Size() != 1 {
			t.Errorf("Expected queue size 1, got %d", queue.Size())
		}

		// Wait for auto-flush
		time.Sleep(1 * time.Second)

		// Queue should be empty after auto-flush
		if queue.Size() != 0 {
			t.Errorf("Expected queue size 0 after auto-flush, got %d", queue.Size())
		}
	})

	t.Run("AutoFlushByThreshold", func(t *testing.T) {
		queue := NewBulkQueue("testing", BulkQueueConfig{
			FlushInterval:  1 * time.Minute, // Long interval
			FlushThreshold: 5,               // Flush after 5 items
		})
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		// Add 4 items (below threshold)
		for i := 0; i < 4; i++ {
			queue.Add(testIndex, map[string]any{
				"threshold_test": i,
				"visibleBy":      []string{"public"},
			})
		}

		// Should still have items (threshold not reached)
		time.Sleep(100 * time.Millisecond)

		// Add 5th item to trigger threshold
		queue.Add(testIndex, map[string]any{
			"threshold_test": 4,
			"visibleBy":      []string{"public"},
		})

		// Wait for async flush to complete
		time.Sleep(500 * time.Millisecond)

		// Queue should be empty after threshold flush
		if queue.Size() != 0 {
			t.Errorf("Expected queue size 0 after threshold flush, got %d", queue.Size())
		}
	})

	t.Run("ConcurrentAdds", func(t *testing.T) {
		queue := NewBulkQueue("testing", BulkQueueConfig{
			FlushInterval: 30 * time.Second, // Don't auto-flush during test
		})
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		const goroutines = 10
		const itemsPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			go func(gid int) {
				defer wg.Done()
				for j := 0; j < itemsPerGoroutine; j++ {
					queue.Add(testIndex, map[string]any{
						"goroutine":  gid,
						"item":       j,
						"concurrent": true,
						"visibleBy":  []string{"public"},
					})
				}
			}(i)
		}

		wg.Wait()

		expectedTotal := goroutines * itemsPerGoroutine
		if queue.Size() != expectedTotal {
			t.Errorf("Expected queue size %d, got %d", expectedTotal, queue.Size())
		}

		// Flush all
		queue.Flush()
	})

	t.Run("Clear", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		// Add some items
		for i := 0; i < 10; i++ {
			queue.Add(testIndex, map[string]any{"i": i})
		}

		if queue.Size() != 10 {
			t.Errorf("Expected queue size 10, got %d", queue.Size())
		}

		// Clear without flushing
		queue.Clear()

		if queue.Size() != 0 {
			t.Errorf("Expected queue size 0 after clear, got %d", queue.Size())
		}
	})

	t.Run("StopFlushesRemaining", func(t *testing.T) {
		queue := NewBulkQueue("testing", BulkQueueConfig{
			FlushInterval: 1 * time.Minute, // Long interval
		})
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}

		// Add some items
		for i := 0; i < 5; i++ {
			queue.Add(testIndex, map[string]any{
				"stop_test": i,
				"visibleBy": []string{"public"},
			})
		}

		// Stop should flush remaining items
		queue.Stop()

		// Queue should not be running
		if queue.IsRunning() {
			t.Error("Expected queue to not be running after Stop")
		}

		// Wait for documents to be indexed
		time.Sleep(1 * time.Second)

		// Verify documents were indexed
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")
		query := builder.
			Exists("stop_test").
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Hits.Total.Value < 5 {
			t.Errorf("Expected at least 5 documents from stop flush, got %d", resp.Hits.Total.Value)
		}
	})

	t.Run("Callbacks", func(t *testing.T) {
		var successCount atomic.Int32
		var errorCalled atomic.Bool

		queue := NewBulkQueue("testing", BulkQueueConfig{
			FlushInterval: 1 * time.Second,
			OnSuccess: func(count int, indexCounts map[string]int) {
				successCount.Add(int32(count))
				t.Logf("OnSuccess called: count=%d, indexCounts=%v", count, indexCounts)
			},
			OnError: func(items []BulkItem, err error) {
				errorCalled.Store(true)
				t.Logf("OnError called: items=%d, err=%v", len(items), err)
			},
		})
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		// Add valid documents
		for i := 0; i < 5; i++ {
			queue.Add(testIndex, map[string]any{
				"callback_test": i,
				"visibleBy":     []string{"public"},
			})
		}

		queue.Flush()

		// Wait for callback
		time.Sleep(500 * time.Millisecond)

		if successCount.Load() < 5 {
			t.Errorf("Expected success count >= 5, got %d", successCount.Load())
		}
	})

	t.Run("MultipleIndices", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		testIndex2 := testIndex + "-2"
		defer func() {
			_, _ = apiClient.Indices.Delete(ctx, opensearchapi.IndicesDeleteReq{
				Indices: []string{testIndex2},
			})
		}()

		// Add items to multiple indices
		items := []BulkItem{
			{Index: testIndex, Operation: BulkOperationIndex, Document: map[string]any{"index": 1, "visibleBy": []string{"public"}}},
			{Index: testIndex2, Operation: BulkOperationIndex, Document: map[string]any{"index": 2, "visibleBy": []string{"public"}}},
			{Index: testIndex, Operation: BulkOperationIndex, Document: map[string]any{"index": 1, "visibleBy": []string{"public"}}},
			{Index: testIndex2, Operation: BulkOperationIndex, Document: map[string]any{"index": 2, "visibleBy": []string{"public"}}},
		}

		queue.AddItems(items)
		queue.Flush()

		// Wait for indexing
		time.Sleep(1 * time.Second)

		// Verify counts per index
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")
		query := builder.
			Term("index", 1).
			Size(10).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Hits.Total.Value < 2 {
			t.Errorf("Expected at least 2 documents in index 1, got %d", resp.Hits.Total.Value)
		}
	})

	t.Run("StructDocument", func(t *testing.T) {
		queue := NewBulkQueueWithDefaults("testing")
		if queue == nil {
			t.Fatal("Expected non-nil queue")
		}
		defer queue.Stop()

		type TestDoc struct {
			Name      string   `json:"name"`
			Value     int      `json:"value"`
			Active    bool     `json:"active"`
			VisibleBy []string `json:"visibleBy"`
		}

		doc := TestDoc{
			Name:      "struct-test",
			Value:     42,
			Active:    true,
			VisibleBy: []string{"public"},
		}

		queue.Add(testIndex, doc)
		queue.Flush()

		// Wait for indexing and force refresh
		time.Sleep(500 * time.Millisecond)
		refreshIndex(ctx, testIndex)

		// Verify document (use .keyword for exact match on dynamic string fields)
		builder := NewQueryBuilder(ctx, []string{testIndex}, "testing")
		query := builder.
			Term("name.keyword", "struct-test").
			Size(1).
			Build()

		resp, err := query.WideSearchIn(ctx, []string{testIndex})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if resp.Hits.Total.Value != 1 {
			t.Errorf("Expected 1 document, got %d", resp.Hits.Total.Value)
		}

		if len(resp.Hits.Hits) > 0 {
			var result TestDoc
			err := resp.Hits.Hits[0].Source.ParseSource(&result)
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}
			if result.Name != "struct-test" || result.Value != 42 || !result.Active {
				t.Errorf("Document mismatch: got %+v", result)
			}
		}
	})
}

// BenchmarkBulkQueueAdd benchmarks adding documents to the queue
func BenchmarkBulkQueueAdd(b *testing.B) {
	nodes := getTestNodes()
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		b.Skipf("Skipping benchmark: could not connect to OpenSearch: %v", err)
	}

	queue := NewBulkQueue("testing", BulkQueueConfig{
		FlushInterval: 1 * time.Hour, // Don't auto-flush during benchmark
	})
	if queue == nil {
		b.Fatal("Expected non-nil queue")
	}
	defer queue.Stop()

	doc := map[string]any{
		"field1": "value1",
		"field2": 123,
		"field3": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.Add("benchmark-index", doc)
	}
	b.StopTimer()

	queue.Clear()
}

// BenchmarkBulkQueueAddBatch benchmarks batch adding documents
func BenchmarkBulkQueueAddBatch(b *testing.B) {
	nodes := getTestNodes()
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		b.Skipf("Skipping benchmark: could not connect to OpenSearch: %v", err)
	}

	queue := NewBulkQueue("testing", BulkQueueConfig{
		FlushInterval: 1 * time.Hour, // Don't auto-flush during benchmark
	})
	if queue == nil {
		b.Fatal("Expected non-nil queue")
	}
	defer queue.Stop()

	docs := make([]any, 100)
	for i := 0; i < 100; i++ {
		docs[i] = map[string]any{
			"field1": "value1",
			"field2": i,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.AddBatch("benchmark-index", docs)
	}
	b.StopTimer()

	queue.Clear()
}

// BenchmarkBulkQueueConcurrentAdd benchmarks concurrent document addition
func BenchmarkBulkQueueConcurrentAdd(b *testing.B) {
	nodes := getTestNodes()
	err := Connect([]string{nodes}, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		b.Skipf("Skipping benchmark: could not connect to OpenSearch: %v", err)
	}

	queue := NewBulkQueue("testing", BulkQueueConfig{
		FlushInterval: 1 * time.Hour, // Don't auto-flush during benchmark
	})
	if queue == nil {
		b.Fatal("Expected non-nil queue")
	}
	defer queue.Stop()

	doc := map[string]any{
		"field1": "value1",
		"field2": 123,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			queue.Add("benchmark-index", doc)
		}
	})
	b.StopTimer()

	queue.Clear()
}
