package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/threatwinds/go-sdk/store"
)

const (
	defaultMaxRetries = 3
	defaultRetryDelay = time.Second
)

// Insert writes one record. An explicit id upserts, which on MergeTree means
// the newest row wins at merge time rather than immediately — reads can see
// both until then, so callers that need immediacy should not rely on it.
func (d *Driver) Insert(ctx context.Context, s store.Scope, id string, doc any) error {
	raw, err := toJSON(doc)
	if err != nil {
		return err
	}
	w, err := d.BulkWriter(s.Dataset)
	if err != nil {
		return err
	}
	if id != "" {
		raw, err = withID(raw, id)
		if err != nil {
			return err
		}
	}
	if err := w.Write(s, raw); err != nil {
		return err
	}
	// Waits, unlike the batched path: a caller inserting one record and then
	// reading it back should find it, and an insert that returned before it
	// landed would make that a race.
	if bw, ok := w.(*bulkWriter); ok {
		return bw.flush(ctx, true)
	}
	return w.Close(ctx)
}

// BulkWriter batches rows into one INSERT. ClickHouse pays a fixed cost per
// insert regardless of size — every one creates a part that then has to be
// merged — so batching is not an optimisation here, it is how you avoid
// drowning the server in small parts.
func (d *Driver) BulkWriter(dataset store.Dataset) (store.BulkWriter, error) {
	tbl, err := d.table(store.Scope{Dataset: dataset})
	if err != nil {
		return nil, err
	}
	maxRetries := d.cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	retryDelay := d.cfg.RetryDelay
	if retryDelay <= 0 {
		retryDelay = defaultRetryDelay
	}
	return &bulkWriter{
		conn:       d.conn,
		table:      tbl,
		tenantCol:  d.cfg.TenantColumn,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
		onReject:   d.cfg.OnReject,
	}, nil
}

type bulkWriter struct {
	conn      driver.Conn
	table     string
	tenantCol string

	maxRetries int
	retryDelay time.Duration
	onReject   func(row []byte, err error)

	mu   sync.Mutex
	rows []string
}

// Write stamps the tenant onto the row rather than trusting the document, so a
// record cannot be filed under a tenant other than the one it was written for.
func (b *bulkWriter) Write(s store.Scope, doc []byte) error {
	if s.Tenant == "" || s.Tenant == store.AllTenants {
		return store.ErrNoTenant
	}
	raw, err := withField(doc, b.tenantCol, s.Tenant)
	if err != nil {
		return err
	}

	b.mu.Lock()
	b.rows = append(b.rows, string(raw))
	b.mu.Unlock()
	return nil
}

// Flush does not wait for the server to finish. Batches are the throughput
// path, and blocking each one on a round trip is what they exist to avoid.
func (b *bulkWriter) Flush(ctx context.Context) error { return b.flush(ctx, false) }

type dedupTokenKey struct{}

func WithDeduplicationToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, dedupTokenKey{}, token)
}

// flush sends whatever is buffered. It does not clear a row until an insert
// that actually included it has succeeded: ClickHouse parses one
// INSERT ... FORMAT JSONEachRow as a single unit, so one bad row fails the
// whole batch, and clearing the buffer before knowing that would lose every
// row alongside it — not just the bad one. insertResilient is what keeps that
// from meaning "buffer stays full forever": it retries the batch as a whole
// for the transient case, and only pulls it apart once a retry still fails.
func (b *bulkWriter) flush(ctx context.Context, wait bool) error {
	b.mu.Lock()
	n := len(b.rows)
	if n == 0 {
		b.mu.Unlock()
		return nil
	}
	rows := append([]string(nil), b.rows[:n]...)
	b.mu.Unlock()

	if token, ok := ctx.Value(dedupTokenKey{}).(string); ok && token != "" {
		ctx = clickhouse.Context(ctx, clickhouse.WithSettings(clickhouse.Settings{
			"insert_deduplication_token": token,
		}))
	}

	if err := b.insertResilient(ctx, rows); err != nil {
		// insertResilient only returns an error when ctx ended mid-attempt —
		// these n rows were never confirmed landed or rejected, so leave the
		// buffer as it is and let the next flush pick them up again.
		return err
	}

	// Every row insertResilient touched is now resolved: inserted, or
	// quarantined and reported through onReject. Only the prefix this flush
	// took a snapshot of is dropped — Write may have appended more past index
	// n while the network call was in flight, and those belong to the next
	// flush, not this one.
	b.mu.Lock()
	b.rows = b.rows[n:]
	b.mu.Unlock()

	return nil
}

// insertResilient inserts rows, retrying the batch as a whole for a bounded
// number of attempts — the case that covers almost every failure: the server
// was briefly overloaded, a part merge ate its memory, a connection blipped.
// None of that is the rows' fault, and none of it survives a few seconds and
// a retry.
//
// A batch that still fails after every retry is content ClickHouse will not
// accept as sent, which retrying again would not fix. Bisecting it isolates
// which row(s) those are: everything that is not the culprit still gets
// inserted, at the cost of one extra round trip per half. A row that still
// fails on its own, after its own retries, is quarantined via onReject
// instead of retried forever.
//
// Returns an error only when ctx is done mid-attempt: that is the one case
// where rows were not resolved either way (inserted or quarantined) and
// belong back in the buffer for the next flush, rather than reported lost.
func (b *bulkWriter) insertResilient(ctx context.Context, rows []string) error {
	if len(rows) == 0 {
		return nil
	}

	var err error
	for attempt := 0; ; attempt++ {
		if err = b.insertOnce(ctx, rows); err == nil {
			return nil
		}
		if attempt >= b.maxRetries {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(b.retryDelay):
		}
	}

	if len(rows) == 1 {
		b.reject(rows[0], err)
		return nil
	}

	mid := len(rows) / 2
	err1 := b.insertResilient(ctx, rows[:mid])
	err2 := b.insertResilient(ctx, rows[mid:])
	if err1 != nil {
		return err1
	}
	return err2
}

func (b *bulkWriter) insertOnce(ctx context.Context, rows []string) error {
	// JSONEachRow lets each row carry whatever fields it has, which is what
	// keeps records of different shapes in one table.
	batch := "INSERT INTO " + b.table + " FORMAT JSONEachRow\n" + joinLines(rows)
	if err := b.conn.AsyncInsert(ctx, batch, true); err != nil {
		return fmt.Errorf("clickhouse: bulk insert of %d rows: %w", len(rows), err)
	}
	return nil
}

func (b *bulkWriter) reject(row string, err error) {
	if b.onReject != nil {
		b.onReject([]byte(row), err)
	}
}

func (b *bulkWriter) Close(ctx context.Context) error { return b.flush(ctx, true) }

func (d *Driver) UpdateWhere(ctx context.Context, s store.Scope, filters []store.Filter, patch map[string]any) (int64, error) {
	if len(patch) == 0 {
		return 0, nil
	}
	tbl, err := d.table(s)
	if err != nil {
		return 0, err
	}
	pred, args, err := d.where(s, filters)
	if err != nil {
		return 0, err
	}

	sets := make([]string, 0, len(patch))
	setArgs := make([]any, 0, len(patch))
	for k, v := range patch {
		col, err := column(k)
		if err != nil {
			return 0, err
		}
		sets = append(sets, col+" = ?")
		setArgs = append(setArgs, v)
	}

	// ALTER ... UPDATE is a mutation: asynchronous, rewriting whole parts. It
	// is the wrong tool for anything frequent, and the row count is not
	// knowable up front, so this reports nothing rather than a wrong number.
	q := fmt.Sprintf("ALTER TABLE %s UPDATE %s WHERE %s", tbl, joinComma(sets), pred)
	if err := d.conn.Exec(ctx, q, append(setArgs, args...)...); err != nil {
		return 0, fmt.Errorf("clickhouse: update: %w", err)
	}
	return 0, nil
}

// Flush is a no-op: ClickHouse is read-your-writes for synchronous inserts.
func (d *Driver) Flush(ctx context.Context, dataset store.Dataset) error { return nil }

func toJSON(doc any) ([]byte, error) {
	switch t := doc.(type) {
	case []byte:
		return t, nil
	case json.RawMessage:
		return t, nil
	case string:
		return []byte(t), nil
	default:
		return json.Marshal(doc)
	}
}

func withID(raw []byte, id string) ([]byte, error) { return withField(raw, "id", id) }

func withField(raw []byte, field, value string) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("clickhouse: document is not a JSON object: %w", err)
	}
	m[field] = value
	return json.Marshal(m)
}

func joinLines(rows []string) string {
	out := ""
	for i, r := range rows {
		if i > 0 {
			out += "\n"
		}
		out += r
	}
	return out
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
