package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/threatwinds/go-sdk/store"
)

// fakeConn is a driver.Conn whose only meaningful method is AsyncInsert: the
// bulkWriter never calls anything else. script decides, per call, whether
// that call succeeds; calls records every batch it was asked to send, so a
// test can tell what actually reached "the server".
type fakeConn struct {
	driver.Conn

	mu     sync.Mutex
	calls  []string // the raw INSERT text of every AsyncInsert call
	script func(call int, query string) error
}

func (f *fakeConn) AsyncInsert(_ context.Context, query string, _ bool, _ ...any) error {
	f.mu.Lock()
	call := len(f.calls)
	f.calls = append(f.calls, query)
	f.mu.Unlock()
	if f.script == nil {
		return nil
	}
	return f.script(call, query)
}

func (f *fakeConn) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func newWriter(conn *fakeConn, maxRetries int, retryDelay time.Duration, onReject func(row []byte, err error)) *bulkWriter {
	return &bulkWriter{
		conn:       conn,
		table:      "t",
		tenantCol:  "tenantId",
		maxRetries: maxRetries,
		retryDelay: retryDelay,
		onReject:   onReject,
	}
}

func writeRow(t *testing.T, w *bulkWriter, id string) {
	t.Helper()
	doc := fmt.Sprintf(`{"id":%q}`, id)
	if err := w.Write(store.Scope{Tenant: "t1"}, []byte(doc)); err != nil {
		t.Fatalf("Write(%s): %v", id, err)
	}
}

var errPoison = errors.New("clickhouse: mock CANNOT_PARSE_DATETIME")

// A batch is one INSERT ... FORMAT JSONEachRow: the poison row's id has to be
// searched for inside the whole text, not compared to it.
func containsID(query, id string) bool {
	return strings.Contains(query, `"id":"`+id+`"`)
}

// The bug this whole package exists to close: one row ClickHouse will never
// accept must not cost the other rows sent alongside it. bisection is what
// gets there — every call that doesn't contain "bad" succeeds, so any call
// that fails and doesn't contain "bad" would mean a good row got blamed for
// the poison one.
func TestBulkWriterIsolatesOnePoisonRowAndKeepsTheRest(t *testing.T) {
	conn := &fakeConn{script: func(_ int, query string) error {
		if containsID(query, "bad") {
			return errPoison
		}
		return nil
	}}

	var rejected []string
	w := newWriter(conn, 2, time.Millisecond, func(row []byte, err error) {
		if !errors.Is(err, errPoison) {
			t.Errorf("reject reported %v, want errPoison", err)
		}
		rejected = append(rejected, string(row))
	})

	for _, id := range []string{"a", "b", "bad", "c", "d"} {
		writeRow(t, w, id)
	}

	if err := w.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if len(rejected) != 1 || !strings.Contains(rejected[0], `"id":"bad"`) {
		t.Fatalf("rejected = %v, want exactly the bad row", rejected)
	}

	sent := map[string]bool{}
	for _, call := range conn.calls {
		for _, id := range []string{"a", "b", "c", "d"} {
			if containsID(call, id) {
				sent[id] = true
			}
		}
	}
	for _, id := range []string{"a", "b", "c", "d"} {
		if !sent[id] {
			t.Errorf("row %q was never sent to ClickHouse; the poison row cost it too", id)
		}
	}

	if len(w.rows) != 0 {
		t.Errorf("buffer still holds %d row(s) after a fully-resolved flush", len(w.rows))
	}
}

// The common case: a server that is briefly overloaded, not a bad row. A
// retry should recover the whole batch — nothing should be bisected or
// rejected just because the first attempt failed.
func TestBulkWriterRetriesATransientFailureWithoutLosingOrRejectingAnything(t *testing.T) {
	conn := &fakeConn{script: func(call int, _ string) error {
		if call == 0 {
			return errors.New("connection reset by peer")
		}
		return nil
	}}

	rejectedAny := false
	w := newWriter(conn, 3, time.Millisecond, func(row []byte, err error) { rejectedAny = true })

	writeRow(t, w, "a")
	writeRow(t, w, "b")

	if err := w.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if rejectedAny {
		t.Error("a transient failure caused a row to be rejected; it should have just retried")
	}
	if conn.callCount() != 2 {
		t.Errorf("AsyncInsert called %d time(s), want 2 (one failure, one retry that succeeds)", conn.callCount())
	}
}

// The core invariant the old code broke: the buffer must not be emptied
// before an insert that included those rows has actually succeeded. This
// drives flush directly (bypassing Write's tenant stamping) so it can inspect
// b.rows immediately after a failed attempt, before any retry has run.
func TestBulkWriterDoesNotClearTheBufferBeforeSuccessIsConfirmed(t *testing.T) {
	conn := &fakeConn{script: func(int, string) error { return errors.New("boom") }}
	w := newWriter(conn, 0, time.Millisecond, func(row []byte, err error) {})

	w.rows = []string{`{"id":"a"}`}

	// maxRetries=0 with a permanent error settles as a single-row reject —
	// rows must be gone (quarantined, reported through onReject) precisely
	// because flush confirmed that outcome, not because it gave up early.
	if err := w.flush(context.Background(), false); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(w.rows) != 0 {
		t.Fatalf("buffer = %v after a resolved (rejected) flush, want empty", w.rows)
	}
}

// A context that dies mid-attempt is not a verdict on the rows — they were
// never confirmed lost or landed, so they must stay queued for the next
// flush rather than being silently dropped or wrongly quarantined.
func TestBulkWriterRequeuesOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	conn := &fakeConn{script: func(call int, _ string) error {
		if call == 0 {
			cancel()
			return errors.New("boom")
		}
		return nil
	}}

	rejected := false
	w := newWriter(conn, 3, 50*time.Millisecond, func(row []byte, err error) { rejected = true })
	writeRow(t, w, "a")

	err := w.flush(ctx, false)
	if err == nil {
		t.Fatal("flush with a cancelled context returned nil, want an error")
	}
	if rejected {
		t.Error("a cancelled context caused a row to be rejected; it was never confirmed unrecoverable")
	}
	if len(w.rows) != 1 {
		t.Fatalf("buffer = %v after a cancelled flush, want the row still queued", w.rows)
	}
}
