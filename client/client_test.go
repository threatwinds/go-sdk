package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/threatwinds/go-sdk/catcher"
)

func TestNew_NoAuth(t *testing.T) {
	_, err := New()
	if err == nil {
		t.Fatal("expected error with no auth, got nil")
	}
	if _, ok := err.(*SDKError); !ok {
		t.Fatalf("expected *SDKError, got %T", err)
	}
}

func TestNew_PartialAPIKey(t *testing.T) {
	// Key without secret should be rejected
	_, err := New(WithAPIKey("key", ""))
	if err == nil {
		t.Fatal("expected error with partial API key, got nil")
	}
	// Secret without key should be rejected
	_, err = New(WithAPIKey("", "secret"))
	if err == nil {
		t.Fatal("expected error with partial API key (secret only), got nil")
	}
}

func TestNew_DuplicateAuth(t *testing.T) {
	_, err := New(WithAPIKey("key", "secret"), WithBearer("token"))
	if err == nil {
		t.Fatal("expected error with both auth methods, got nil")
	}
	if _, ok := err.(*SDKError); !ok {
		t.Fatalf("expected *SDKError, got %T", err)
	}
}

func TestNew_APIKeyValue(t *testing.T) {
	c, err := New(WithAPIKey("my-key", "my-secret"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.apiKey != "my-key" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "my-key")
	}
	if c.apiSecret != "my-secret" {
		t.Errorf("apiSecret = %q, want %q", c.apiSecret, "my-secret")
	}
}

func TestNew_BearerValue(t *testing.T) {
	c, err := New(WithBearer("my-token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.bearer != "my-token" {
		t.Errorf("bearer = %q, want %q", c.bearer, "my-token")
	}
}

func TestNew_Defaults(t *testing.T) {
	c, err := New(WithBearer("token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.endpoint != DefaultEndpoint {
		t.Errorf("endpoint = %q, want %q", c.endpoint, DefaultEndpoint)
	}
	if c.maxRetries != defaultMaxRetries {
		t.Errorf("maxRetries = %d, want %d", c.maxRetries, defaultMaxRetries)
	}
}

func TestNew_CustomTimeout(t *testing.T) {
	c, err := New(WithBearer("token"), WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The timeout is applied to the http.Client; verify the client has the right timeout.
	if c.httpClient.Timeout != 10*time.Second {
		t.Errorf("httpClient.Timeout = %v, want %v", c.httpClient.Timeout, 10*time.Second)
	}
}

func TestNew_CustomMaxRetries(t *testing.T) {
	c, err := New(WithBearer("token"), WithMaxRetries(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.maxRetries != 5 {
		t.Errorf("maxRetries = %d, want %d", c.maxRetries, 5)
	}
}

func TestNew_CustomHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	c, err := New(WithBearer("token"), WithHTTPClient(custom))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.httpClient != custom {
		t.Error("httpClient should be the custom client")
	}
}

func TestServiceAccessors(t *testing.T) {
	c, err := New(WithBearer("token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Auth() == nil {
		t.Error("Auth() returned nil")
	}
	if c.Billing() == nil {
		t.Error("Billing() returned nil")
	}
	if c.Compute() == nil {
		t.Error("Compute() returned nil")
	}
}

func TestClose_SDKCreatedTransport(t *testing.T) {
	c, err := New(WithBearer("token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.transport == nil {
		t.Fatal("SDK-created client should have a non-nil transport")
	}
	// Close should not panic.
	c.Close()
	// Calling Close multiple times should be safe.
	c.Close()
}

func TestClose_UserSuppliedClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	c, err := New(WithBearer("token"), WithHTTPClient(custom))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.transport != nil {
		t.Fatal("user-supplied client should have a nil transport")
	}
	// Close should be a no-op, not panic.
	c.Close()
}

// ---------------------------------------------------------------------------
// mockRT implements http.RoundTripper for unit tests.
// ---------------------------------------------------------------------------

type mockRT struct {
	roundTripper func(req *http.Request) (*http.Response, error)
}

func (m *mockRT) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.roundTripper != nil {
		return m.roundTripper(req)
	}
	return nil, nil
}

func mockResp(status int, headers http.Header, body string) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// ---------------------------------------------------------------------------
// TestDo_* tests
// ---------------------------------------------------------------------------

func TestDo_Success(t *testing.T) {
	var receivedMethod string
	var receivedPath string
	var receivedKey string
	var receivedSecret string

	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			receivedMethod = req.Method
			receivedPath = req.URL.Path
			receivedKey = req.Header.Get("Api-Key")
			receivedSecret = req.Header.Get("Api-Secret")
			return mockResp(200, nil, `{"name":"alice","role":"admin"}`), nil
		},
	}

	c, _ := New(WithAPIKey("k1", "s1"), WithHTTPClient(&http.Client{Transport: rt}))
	c.endpoint = "https://api.example.com"

	var result map[string]string
	err := c.do(context.Background(), http.MethodGet, "/users/1", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "GET" {
		t.Errorf("method = %q, want GET", receivedMethod)
	}
	if receivedPath != "/users/1" {
		t.Errorf("path = %q, want /users/1", receivedPath)
	}
	if receivedKey != "k1" {
		t.Errorf("Api-Key = %q, want k1", receivedKey)
	}
	if receivedSecret != "s1" {
		t.Errorf("Api-Secret = %q, want s1", receivedSecret)
	}
	if result["name"] != "alice" {
		t.Errorf("result[name] = %q, want alice", result["name"])
	}
}

func TestDo_BearerAuth(t *testing.T) {
	var receivedAuth string

	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			receivedAuth = req.Header.Get("Authorization")
			return mockResp(200, nil, `{"ok":true}`), nil
		},
	}

	c, _ := New(WithBearer("my-token"), WithHTTPClient(&http.Client{Transport: rt}))
	c.endpoint = "https://api.example.com"

	var result map[string]bool
	err := c.do(context.Background(), http.MethodGet, "/ping", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedAuth != "Bearer my-token" {
		t.Errorf("Authorization = %q, want 'Bearer my-token'", receivedAuth)
	}
	if !result["ok"] {
		t.Error("expected ok=true")
	}
}

func TestDo_PostBody(t *testing.T) {
	var receivedMethod string
	var receivedCT string
	var receivedBody []byte

	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			receivedMethod = req.Method
			receivedCT = req.Header.Get("Content-Type")
			receivedBody, _ = io.ReadAll(req.Body)
			return mockResp(200, nil, `{"id":"created-1"}`), nil
		},
	}

	c, _ := New(WithBearer("tok"), WithHTTPClient(&http.Client{Transport: rt}))
	c.endpoint = "https://api.example.com"

	payload := map[string]string{"name": "server-1"}
	var result map[string]string
	err := c.do(context.Background(), http.MethodPost, "/servers", payload, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "POST" {
		t.Errorf("method = %q, want POST", receivedMethod)
	}
	if receivedCT != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", receivedCT)
	}

	var parsed map[string]string
	if err := json.Unmarshal(receivedBody, &parsed); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if parsed["name"] != "server-1" {
		t.Errorf("request body name = %q, want server-1", parsed["name"])
	}
	if result["id"] != "created-1" {
		t.Errorf("result[id] = %q, want created-1", result["id"])
	}
}

func TestDo_APIError(t *testing.T) {
	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			h := make(http.Header)
			h.Set("X-Error", "user not found")
			h.Set("X-Error-Id", "err-123")
			return mockResp(404, h, `{"error":"not found"}`), nil
		},
	}

	c, _ := New(WithBearer("tok"), WithHTTPClient(&http.Client{Transport: rt}))
	c.endpoint = "https://api.example.com"

	var result map[string]string
	err := c.do(context.Background(), http.MethodGet, "/users/999", nil, &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Message != "user not found" {
		t.Errorf("message = %q, want 'user not found'", apiErr.Message)
	}
	if apiErr.ErrorID != "err-123" {
		t.Errorf("errorID = %q, want err-123", apiErr.ErrorID)
	}
}

func TestDo_RetryRetryAfter(t *testing.T) {
	attempts := 0

	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				h := make(http.Header)
				h.Set("Retry-After", "0")
				h.Set("X-Error", "rate limited")
				h.Set("X-Error-Id", "rl-1")
				return mockResp(429, h, `{"error":"rate limited"}`), nil
			}
			return mockResp(200, nil, `{"ok":true}`), nil
		},
	}

	c, _ := New(WithBearer("tok"), WithHTTPClient(&http.Client{Transport: rt}), WithMaxRetries(3))
	c.endpoint = "https://api.example.com"

	var result map[string]bool
	err := c.do(context.Background(), http.MethodGet, "/data", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2 (initial + 1 retry)", attempts)
	}
	if !result["ok"] {
		t.Error("expected ok=true")
	}
}

func TestDo_NoRetryOnPost(t *testing.T) {
	attempts := 0

	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			attempts++
			h := make(http.Header)
			h.Set("X-Error", "service unavailable")
			h.Set("X-Error-Id", "su-1")
			return mockResp(503, h, `{"error":"unavailable"}`), nil
		},
	}

	c, _ := New(WithBearer("tok"), WithHTTPClient(&http.Client{Transport: rt}), WithMaxRetries(3))
	c.endpoint = "https://api.example.com"

	payload := map[string]string{"action": "deploy"}
	var result map[string]string
	err := c.do(context.Background(), http.MethodPost, "/deploy", payload, &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on POST)", attempts)
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 503 {
		t.Errorf("status = %d, want 503", apiErr.StatusCode)
	}
}

func TestDo_ContextCancel(t *testing.T) {
	// Use a test server that sleeps to ensure context is cancelled before response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	}))
	defer server.Close()

	c, _ := New(WithBearer("tok"))
	c.endpoint = server.URL

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately.
	cancel()

	var result map[string]string
	err := c.do(ctx, http.MethodGet, "/slow", nil, &result)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Errorf("error = %v, expected context cancellation error", err)
	}
}

// The context branches of do() must not throw away the failure that caused the
// retry. Returning the bare ctx.Err() reports "context deadline exceeded" about
// a request that in fact got a 503 from a real service, and loses the
// upstream's occurrence id, its status, IsRateLimited() and RetryAfter() with
// it — so the two services' log lines for one failure stop being correlatable,
// which is the whole point of reading x-error-id.

func TestDo_ContextExpiredDuringBackoffKeepsUpstreamFailure(t *testing.T) {
	attempts := 0
	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			attempts++
			h := make(http.Header)
			h.Set("Retry-After", "60") // long enough that the ctx wins the select
			h.Set("X-Error", "service unavailable")
			h.Set("X-Error-Id", upstreamErrorID)
			return mockResp(503, h, `{"error":"unavailable"}`), nil
		},
	}

	c, _ := New(WithBearer("tok"), WithHTTPClient(&http.Client{Transport: rt}), WithMaxRetries(2))
	c.endpoint = "https://api.example.com"

	// Already expired: the backoff select therefore takes ctx.Done()
	// immediately, deterministically, without waiting out the Retry-After.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.do(ctx, http.MethodGet, "/api/billing/v1/quota", nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if attempts != 1 {
		t.Fatalf("setup: expected one attempt before the context branch, got %d", attempts)
	}

	// Both truths survive: the caller can still match the context failure...
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected the context error to remain matchable, got %v", err)
	}
	// ...and the upstream failure it was retrying.
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected the *APIError to survive, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", apiErr.StatusCode)
	}
	if apiErr.RetryAfter() != "60" {
		t.Errorf("retryAfter = %q, want 60", apiErr.RetryAfter())
	}

	// And the id the upstream minted still becomes the caller's error's id,
	// rather than a second, unrelated one.
	wrapped := catcher.New("calling billing-api failed", err, map[string]any{"status": http.StatusBadGateway})
	if wrapped.ErrorID != upstreamErrorID {
		t.Errorf("expected the upstream id %q to survive, got %q", upstreamErrorID, wrapped.ErrorID)
	}
}

// The same guarantee for the other context branch — the check at the top of the
// next iteration. With a zero backoff both branches are live and which one runs
// depends on how the select is scheduled, which is exactly why they must not
// differ: whichever fires, the upstream failure is still there.
func TestDo_ContextExpiredBetweenAttemptsKeepsUpstreamFailure(t *testing.T) {
	var cancel context.CancelFunc
	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			h := make(http.Header)
			h.Set("Retry-After", "0")
			h.Set("X-Error", "service unavailable")
			h.Set("X-Error-Id", upstreamErrorID)
			cancel() // expired by the time this response is processed
			return mockResp(503, h, `{"error":"unavailable"}`), nil
		},
	}

	c, _ := New(WithBearer("tok"), WithHTTPClient(&http.Client{Transport: rt}), WithMaxRetries(2))
	c.endpoint = "https://api.example.com"

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	err := c.do(ctx, http.MethodGet, "/api/billing/v1/quota", nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected the context error to remain matchable, got %v", err)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected the 503 *APIError to survive, got %T: %v", err, err)
	}
	if got := catcher.New("calling billing-api failed", err, nil).ErrorID; got != upstreamErrorID {
		t.Errorf("expected the upstream id %q to survive, got %q", upstreamErrorID, got)
	}
}

// A context that ends before anything was attempted has no failure to carry, so
// the context error is returned as it is rather than joined with nothing.
func TestExpiredWithNoPriorFailure(t *testing.T) {
	if got := expiredWith(context.Canceled, nil); got != context.Canceled {
		t.Errorf("expected the bare context error, got %v", got)
	}
}

// The end-to-end path: an upstream ThreatWinds service's x-error-id has to
// reach the error a calling service builds, so one failure keeps one id across
// the hop. TestDo_APIError above covers reading the header; these cover what
// happens to it afterwards.

func TestDo_UpstreamErrorIDReachesCatcher(t *testing.T) {
	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			h := make(http.Header)
			h.Set("X-Error", "session expired")
			h.Set("X-Error-Id", upstreamErrorID)
			return mockResp(401, h, `{"error":"unauthorized"}`), nil
		},
	}

	c, _ := New(WithBearer("tok"), WithHTTPClient(&http.Client{Transport: rt}))
	c.endpoint = "https://api.example.com"

	err := c.do(context.Background(), http.MethodGet, "/api/auth/v1/session", nil, nil)

	wrapped := catcher.New("failed to validate session with auth-api", err,
		map[string]any{"status": http.StatusBadGateway})
	if wrapped.ErrorID != upstreamErrorID {
		t.Errorf("expected the upstream id %q to survive the hop, got %q", upstreamErrorID, wrapped.ErrorID)
	}
	if wrapped.Args["status"] != http.StatusBadGateway {
		t.Errorf("the caller's status must survive too, got %v", wrapped.Args["status"])
	}
}

func TestDo_GeneratesErrorIDWhenUpstreamSendsNone(t *testing.T) {
	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			return mockResp(500, make(http.Header), `{"error":"boom"}`), nil
		},
	}

	c, _ := New(WithBearer("tok"), WithHTTPClient(&http.Client{Transport: rt}))
	c.endpoint = "https://api.example.com"

	err := c.do(context.Background(), http.MethodGet, "/api/auth/v1/session", nil, nil)

	wrapped := catcher.New("calling auth-api failed", err, map[string]any{"status": 500})
	if uuid.Validate(wrapped.ErrorID) != nil {
		t.Errorf("expected a generated UUID, got %q", wrapped.ErrorID)
	}
}

// A Header map built by hand — which is exactly what a custom RoundTripper
// installed through WithHTTPClient produces — can hold literal lowercase keys.
// http.Header.Get canonicalizes only the lookup key, so it would miss them and
// the upstream's id would be silently dropped.
func TestDo_ReadsNonCanonicalHeaderKeys(t *testing.T) {
	rt := &mockRT{
		roundTripper: func(req *http.Request) (*http.Response, error) {
			h := http.Header{
				"x-error":     []string{"session expired"},
				"x-error-id":  []string{upstreamErrorID},
				"retry-after": []string{"7"},
			}
			// 400, not a retryable status: do() sleeps out the Retry-After
			// delay even on its final attempt, so a retryable status here
			// would buy nothing but seven seconds of test runtime.
			return mockResp(400, h, ""), nil
		},
	}

	c, _ := New(WithBearer("tok"), WithHTTPClient(&http.Client{Transport: rt}))
	c.endpoint = "https://api.example.com"

	err := c.do(context.Background(), http.MethodGet, "/api/auth/v1/session", nil, nil)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.ErrorID != upstreamErrorID {
		t.Errorf("errorID = %q, want %q", apiErr.ErrorID, upstreamErrorID)
	}
	if apiErr.Message != "session expired" {
		t.Errorf("message = %q, want 'session expired'", apiErr.Message)
	}
	if apiErr.RetryAfter() != "7" {
		t.Errorf("retryAfter = %q, want 7", apiErr.RetryAfter())
	}
}
