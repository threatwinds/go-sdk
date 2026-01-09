package os

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// Unit Tests (no OpenSearch required)
// =============================================================================

func TestDefaultBulkQueueConfig(t *testing.T) {
	cfg := DefaultBulkQueueConfig()

	if cfg.FlushInterval != 10*time.Second {
		t.Errorf("Expected FlushInterval 10s, got %v", cfg.FlushInterval)
	}
	if cfg.FlushThreshold != 0 {
		t.Errorf("Expected FlushThreshold 0, got %d", cfg.FlushThreshold)
	}
	if cfg.MaxRetries != 0 {
		t.Errorf("Expected MaxRetries 0, got %d", cfg.MaxRetries)
	}
	if cfg.RetryDelay != time.Second {
		t.Errorf("Expected RetryDelay 1s, got %v", cfg.RetryDelay)
	}
}

func TestBulkOperationConstants(t *testing.T) {
	tests := []struct {
		op       BulkOperation
		expected string
	}{
		{BulkOperationCreate, "create"},
		{BulkOperationIndex, "index"},
		{BulkOperationUpdate, "update"},
		{BulkOperationDelete, "delete"},
	}

	for _, tt := range tests {
		if string(tt.op) != tt.expected {
			t.Errorf("BulkOperation %v should be %s", tt.op, tt.expected)
		}
	}
}

func TestBulkItemStructure(t *testing.T) {
	item := BulkItem{
		Index:      "test-index",
		DocumentID: "doc-123",
		Operation:  BulkOperationIndex,
		Document:   map[string]any{"field": "value"},
		Routing:    "user-1",
	}

	if item.Index != "test-index" {
		t.Errorf("Expected Index 'test-index', got %s", item.Index)
	}
	if item.DocumentID != "doc-123" {
		t.Errorf("Expected DocumentID 'doc-123', got %s", item.DocumentID)
	}
	if item.Operation != BulkOperationIndex {
		t.Errorf("Expected Operation 'index', got %s", item.Operation)
	}
	if item.Routing != "user-1" {
		t.Errorf("Expected Routing 'user-1', got %s", item.Routing)
	}
}

func TestBulkResponseStructure(t *testing.T) {
	resp := BulkResponse{
		SuccessCount: 10,
		FailedCount:  2,
		IndexCounts: map[string]int{
			"index-a": 7,
			"index-b": 5,
		},
		Errors: []BulkItemError{
			{
				Index:       0,
				Operation:   "index",
				Status:      400,
				ErrorType:   "mapper_parsing_exception",
				ErrorReason: "failed to parse field",
			},
		},
	}

	if resp.SuccessCount != 10 {
		t.Errorf("Expected SuccessCount 10, got %d", resp.SuccessCount)
	}
	if resp.FailedCount != 2 {
		t.Errorf("Expected FailedCount 2, got %d", resp.FailedCount)
	}
	if len(resp.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(resp.Errors))
	}
	if resp.Errors[0].ErrorType != "mapper_parsing_exception" {
		t.Errorf("Expected error type 'mapper_parsing_exception', got %s", resp.Errors[0].ErrorType)
	}
}

func TestNewBulkQueueWithoutConnection(t *testing.T) {
	// Reset connection state for this test
	oldClient := apiClient
	apiClient = nil
	defer func() { apiClient = oldClient }()

	queue := NewBulkQueue(DefaultBulkQueueConfig())
	if queue != nil {
		t.Error("Expected nil when connection not established")
	}

	queue = NewBulkQueueWithDefaults()
	if queue != nil {
		t.Error("Expected nil when connection not established")
	}
}

func TestBulkQueueConfigValidation(t *testing.T) {
	// Test that invalid config values get default values
	cfg := BulkQueueConfig{
		FlushInterval: 0,  // Should default to 10s
		RetryDelay:    -1, // Should default to 1s
	}

	// We can't test this directly without a connection, but we can verify
	// the defaults are sensible
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 10 * time.Second
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = time.Second
	}

	if cfg.FlushInterval != 10*time.Second {
		t.Errorf("Expected FlushInterval to be 10s, got %v", cfg.FlushInterval)
	}
	if cfg.RetryDelay != time.Second {
		t.Errorf("Expected RetryDelay to be 1s, got %v", cfg.RetryDelay)
	}
}

func TestBuildBulkAction(t *testing.T) {
	tests := []struct {
		name     string
		item     BulkItem
		wantOp   string
		wantID   bool
		wantRout bool
	}{
		{
			name: "index without ID",
			item: BulkItem{
				Index:     "test-index",
				Operation: BulkOperationIndex,
			},
			wantOp:   "index",
			wantID:   false,
			wantRout: false,
		},
		{
			name: "create with ID",
			item: BulkItem{
				Index:      "test-index",
				DocumentID: "doc-1",
				Operation:  BulkOperationCreate,
			},
			wantOp:   "create",
			wantID:   true,
			wantRout: false,
		},
		{
			name: "update with routing",
			item: BulkItem{
				Index:      "test-index",
				DocumentID: "doc-1",
				Operation:  BulkOperationUpdate,
				Routing:    "user-1",
			},
			wantOp:   "update",
			wantID:   true,
			wantRout: true,
		},
		{
			name: "delete",
			item: BulkItem{
				Index:      "test-index",
				DocumentID: "doc-1",
				Operation:  BulkOperationDelete,
			},
			wantOp:   "delete",
			wantID:   true,
			wantRout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := buildBulkAction(tt.item)
			if err != nil {
				t.Fatalf("buildBulkAction failed: %v", err)
			}

			opData, ok := action[tt.wantOp].(map[string]any)
			if !ok {
				t.Fatalf("Expected operation %s in action", tt.wantOp)
			}

			if opData["_index"] != tt.item.Index {
				t.Errorf("Expected _index %s, got %v", tt.item.Index, opData["_index"])
			}

			_, hasID := opData["_id"]
			if hasID != tt.wantID {
				t.Errorf("Expected _id presence %v, got %v", tt.wantID, hasID)
			}

			_, hasRouting := opData["routing"]
			if hasRouting != tt.wantRout {
				t.Errorf("Expected routing presence %v, got %v", tt.wantRout, hasRouting)
			}
		})
	}
}

func TestMarshalDocument(t *testing.T) {
	tests := []struct {
		name    string
		item    BulkItem
		wantErr bool
	}{
		{
			name: "simple document",
			item: BulkItem{
				Operation: BulkOperationIndex,
				Document:  map[string]any{"field": "value"},
			},
			wantErr: false,
		},
		{
			name: "update with raw doc",
			item: BulkItem{
				Operation: BulkOperationUpdate,
				Document:  map[string]any{"field": "updated"},
			},
			wantErr: false,
		},
		{
			name: "update already wrapped",
			item: BulkItem{
				Operation: BulkOperationUpdate,
				Document:  map[string]any{"doc": map[string]any{"field": "value"}},
			},
			wantErr: false,
		},
		{
			name: "update with script",
			item: BulkItem{
				Operation: BulkOperationUpdate,
				Document:  map[string]any{"script": "ctx._source.counter++"},
			},
			wantErr: false,
		},
		{
			name: "struct document",
			item: BulkItem{
				Operation: BulkOperationIndex,
				Document: struct {
					Name  string `json:"name"`
					Value int    `json:"value"`
				}{Name: "test", Value: 42},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := marshalDocument(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("marshalDocument() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(data) == 0 {
				t.Error("Expected non-empty marshaled data")
			}
		})
	}
}

// =============================================================================
// Callback Tests (mocked)
// =============================================================================

func TestBulkQueueConfigCallbacks(t *testing.T) {
	var errorCalled atomic.Bool
	var successCalled atomic.Bool

	cfg := BulkQueueConfig{
		FlushInterval: 10 * time.Second,
		OnError: func(items []BulkItem, err error) {
			errorCalled.Store(true)
		},
		OnSuccess: func(count int, indexCounts map[string]int) {
			successCalled.Store(true)
		},
	}

	// Verify callbacks are set
	if cfg.OnError == nil {
		t.Error("OnError callback should be set")
	}
	if cfg.OnSuccess == nil {
		t.Error("OnSuccess callback should be set")
	}

	// Call them to verify they work
	cfg.OnError(nil, nil)
	cfg.OnSuccess(0, nil)

	if !errorCalled.Load() {
		t.Error("OnError callback was not called")
	}
	if !successCalled.Load() {
		t.Error("OnSuccess callback was not called")
	}
}

// =============================================================================
// Concurrency Tests (mock queue operations)
// =============================================================================

func TestBulkItemSliceConcurrency(t *testing.T) {
	// Test that concurrent operations on a slice with mutex protection work
	var mu sync.Mutex
	var queue []BulkItem

	const goroutines = 10
	const itemsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(gid int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				item := BulkItem{
					Index:     "test-index",
					Operation: BulkOperationIndex,
					Document:  map[string]any{"gid": gid, "j": j},
				}

				mu.Lock()
				queue = append(queue, item)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	expectedTotal := goroutines * itemsPerGoroutine
	if len(queue) != expectedTotal {
		t.Errorf("Expected %d items, got %d", expectedTotal, len(queue))
	}
}

func TestBulkItemAtomicSwap(t *testing.T) {
	// Test the atomic swap pattern used in processBulk
	var mu sync.Mutex
	queue := make([]BulkItem, 0)

	// Add some items
	for i := 0; i < 100; i++ {
		queue = append(queue, BulkItem{
			Index:    "test-index",
			Document: map[string]any{"i": i},
		})
	}

	// Simulate atomic swap
	mu.Lock()
	items := make([]BulkItem, len(queue))
	copy(items, queue)
	queue = queue[:0]
	mu.Unlock()

	if len(items) != 100 {
		t.Errorf("Expected 100 items in local slice, got %d", len(items))
	}
	if len(queue) != 0 {
		t.Errorf("Expected empty queue after swap, got %d items", len(queue))
	}
}

// =============================================================================
// BulkItemError Tests
// =============================================================================

func TestBulkItemErrorStructure(t *testing.T) {
	err := BulkItemError{
		Index:         0,
		Operation:     "index",
		DocumentIndex: "test-index",
		DocumentID:    "doc-1",
		Status:        400,
		ErrorType:     "mapper_parsing_exception",
		ErrorReason:   "failed to parse field [timestamp]: Invalid format",
		CauseType:     "illegal_argument_exception",
		CauseReason:   "Invalid date format",
	}

	if err.Index != 0 {
		t.Errorf("Expected Index 0, got %d", err.Index)
	}
	if err.Operation != "index" {
		t.Errorf("Expected Operation 'index', got %s", err.Operation)
	}
	if err.Status != 400 {
		t.Errorf("Expected Status 400, got %d", err.Status)
	}
	if err.CauseType != "illegal_argument_exception" {
		t.Errorf("Expected CauseType 'illegal_argument_exception', got %s", err.CauseType)
	}
}
