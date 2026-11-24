package os

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

func TestNewOpenSearchTransportPooling(t *testing.T) {
	tr := newOpenSearchTransport()
	if tr.MaxIdleConns != 128 {
		t.Fatalf("MaxIdleConns = %d, want 128", tr.MaxIdleConns)
	}
	if tr.MaxIdleConnsPerHost != 64 {
		t.Fatalf("MaxIdleConnsPerHost = %d, want 64", tr.MaxIdleConnsPerHost)
	}
	if tr.IdleConnTimeout != 90*time.Second {
		t.Fatalf("IdleConnTimeout = %v, want 90s", tr.IdleConnTimeout)
	}
	if !tr.ForceAttemptHTTP2 {
		t.Fatalf("ForceAttemptHTTP2 = false, want true")
	}
	if tr.TLSClientConfig == nil || !tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatalf("expected InsecureSkipVerify TLS config")
	}
}

func TestChunkBulkItemsByMaxBytes(t *testing.T) {
	payload := `{"msg":"` + strings.Repeat("a", 400) + `"}`
	items := make([]BulkItem, 10)
	for i := range items {
		items[i] = BulkItem{
			Index:     "events-2026.11.24",
			Operation: BulkOperationIndex,
			Document:  payload,
		}
	}
	// Each item is ~481 bytes estimated; 1024 byte cap yields 2 items per chunk -> 5 chunks.
	chunks := chunkBulkItems(items, 1024)
	if len(chunks) != 5 {
		t.Fatalf("got %d chunks, want 5", len(chunks))
	}
	total := 0
	for _, c := range chunks {
		if len(c) == 0 {
			t.Fatalf("empty chunk")
		}
		total += len(c)
	}
	if total != len(items) {
		t.Fatalf("total chunked items = %d, want %d", total, len(items))
	}
}

func TestBulkQueueMaxBatchBytesSplitsRequests(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"took":1,"errors":false,"items":[]}`))
	}))
	defer srv.Close()

	c, err := opensearchapi.NewClient(opensearchapi.Config{
		Client: opensearch.Config{Addresses: []string{srv.URL}},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var succeeded atomic.Int32
	bq := &BulkQueue{
		client: c,
		config: BulkQueueConfig{
			FlushInterval:  time.Hour,
			FlushThreshold: 1000,
			MaxBatchBytes:  1024,
			RetryDelay:     time.Millisecond,
			OnSuccess: func(count int, _ map[string]int) {
				succeeded.Add(int32(count))
			},
		},
		queue:       make([]BulkItem, 0),
		ticker:      time.NewTicker(time.Hour),
		stopCh:      make(chan struct{}),
		processName: "test",
	}
	defer bq.ticker.Stop()

	payload := `{"msg":"` + strings.Repeat("x", 400) + `"}`
	items := make([]BulkItem, 6)
	for i := range items {
		items[i] = BulkItem{
			Index:     "test-idx",
			Operation: BulkOperationIndex,
			Document:  payload,
		}
	}
	// Enqueue directly and flush synchronously to avoid timing flakes.
	bq.mutex.Lock()
	bq.queue = append(bq.queue, items...)
	bq.queuedBytes = 6 * estimateBulkItemBytes(items[0])
	bq.mutex.Unlock()

	if err := bq.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if got := requests.Load(); got != 3 {
		t.Fatalf("HTTP bulk requests = %d, want 3", got)
	}
	if got := succeeded.Load(); got != 6 {
		t.Fatalf("succeeded items = %d, want 6", got)
	}
	if bq.queuedBytes != 0 {
		t.Fatalf("queuedBytes = %d, want 0 after flush", bq.queuedBytes)
	}
}
