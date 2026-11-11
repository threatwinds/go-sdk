package plugins

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

// stubStore answers the reads ExecuteStore makes and remembers the scope.
type stubStore struct {
	store.Store
	scope store.Scope
	count int64
}

func (s *stubStore) CountInWindow(_ context.Context, sc store.Scope, _ []store.Filter, _ time.Duration) (int64, error) {
	s.scope = sc
	return s.count, nil
}

func (s *stubStore) Count(_ context.Context, sc store.Scope, _ []store.Filter) (int64, error) {
	s.scope = sc
	return s.count, nil
}

func (s *stubStore) FetchN(context.Context, store.Scope, []store.Filter, int) ([]json.RawMessage, error) {
	return []json.RawMessage{json.RawMessage(`{"id":"x"}`)}, nil
}

func logsScope() store.Scope {
	return store.Scope{Tenant: "t", Dataset: "logs", DataType: "triggering-type"}
}

func TestExecuteStoreResolvesAnIndexPatternToADataType(t *testing.T) {
	t.Cleanup(func() { SetIndexPatternResolver(nil) })
	SetIndexPatternResolver(func(ds store.Dataset, p string) (string, bool) {
		if ds == "logs" && p == "v11-log-deceptive-bytes-*" {
			return "deceptive-bytes", true
		}
		if p == "v11-log-*" {
			return "", true
		}
		return "", false
	})

	st := &stubStore{count: 3}
	req := &SearchRequest{IndexPattern: "v11-log-deceptive-bytes-*", Within: "15m", Count: 3}
	if ok, _, err := req.ExecuteStore(context.Background(), st, logsScope(), nil); err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if st.scope.DataType != "deceptive-bytes" {
		t.Errorf("scope dataType = %q, want the pattern's", st.scope.DataType)
	}

	// A pattern over every dataType must not keep the triggering rule's.
	req = &SearchRequest{IndexPattern: "v11-log-*", Within: "15m", Count: 1}
	if _, _, err := req.ExecuteStore(context.Background(), st, logsScope(), nil); err != nil {
		t.Fatal(err)
	}
	if st.scope.DataType != "" {
		t.Errorf("scope dataType = %q, want it cleared for a pattern over every dataType", st.scope.DataType)
	}
}

func TestExecuteStoreRefusesAPatternItCannotResolve(t *testing.T) {
	t.Cleanup(func() { SetIndexPatternResolver(nil) })
	req := &SearchRequest{IndexPattern: "v11-log-x-*", Within: "1h", Count: 1}

	_, _, err := req.ExecuteStore(context.Background(), &stubStore{}, logsScope(), nil)
	if err == nil || !strings.Contains(err.Error(), "no resolver") {
		t.Fatalf("without a resolver: %v", err)
	}

	SetIndexPatternResolver(func(store.Dataset, string) (string, bool) { return "", false })
	_, _, err = req.ExecuteStore(context.Background(), &stubStore{}, logsScope(), nil)
	if err == nil || !strings.Contains(err.Error(), "not one this store reads") {
		t.Fatalf("with a resolver that does not know it: %v", err)
	}
}

func TestExecuteStoreKeepsTheDataTypeAheadOfAnIndexPattern(t *testing.T) {
	t.Cleanup(func() { SetIndexPatternResolver(nil) })
	SetIndexPatternResolver(func(store.Dataset, string) (string, bool) { return "from-pattern", true })

	st := &stubStore{count: 1}
	req := &SearchRequest{DataType: "explicit", IndexPattern: "v11-log-*", Within: "1h", Count: 1}
	if _, _, err := req.ExecuteStore(context.Background(), st, logsScope(), nil); err != nil {
		t.Fatal(err)
	}
	if st.scope.DataType != "explicit" {
		t.Errorf("scope dataType = %q, want the explicit one", st.scope.DataType)
	}
}

func TestExecuteStoreWithoutEitherKeepsTheTriggeringDataType(t *testing.T) {
	st := &stubStore{count: 1}
	req := &SearchRequest{Within: "1h", Count: 1}
	if _, _, err := req.ExecuteStore(context.Background(), st, logsScope(), nil); err != nil {
		t.Fatal(err)
	}
	if st.scope.DataType != "triggering-type" {
		t.Errorf("scope dataType = %q, want the triggering rule's", st.scope.DataType)
	}
}
