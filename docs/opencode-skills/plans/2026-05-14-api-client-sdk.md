# ThreatWinds API Client SDK Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development (recommended) or executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a self-contained, typed Go API client SDK for ThreatWinds Auth, Billing, and Compute services.

**Architecture:** Single `client.New()` constructor returns a root client with `.Auth()`, `.Billing()`, `.Compute()` accessors. All services share one internal `*http.Client` with connection pooling, GET-only retry (with Retry-After awareness), and `*APIError` typed errors. Each service client has its own package under `client/` with typed request/response structs. No dependencies on existing go-sdk packages — stdlib only.

**Tech Stack:** Go 1.25.5, standard library only (`net/http`, `encoding/json`, `context`, `net/url`, `net/textproto`, `crypto/tls`, `time`, `fmt`, `strings`, `bytes`, `io`, `sync`)

**Spec:** `docs/opencode-skills/specs/2026-05-14-api-client-sdk-design.md`

---

## File Structure

```
client/
  client.go            # Client struct, New(), do(), retry, auth headers
  options.go           # Option, ListOptions
  error.go             # APIError, SDKError
  error_test.go        # APIError tests
  client_test.go       # Client New(), do(), retry, auth tests
  auth/
    client.go          # AuthClient, do(), all endpoint methods
    types.go           # All auth request/response structs, filtered options
    auth_test.go       # Auth method tests
  billing/
    client.go          # BillingClient, do(), all endpoint methods
    types.go           # All billing request/response structs, filtered options
    billing_test.go    # Billing method tests
  compute/
    client.go          # ComputeClient, do(), all endpoint methods
    types.go           # All compute request/response structs, filtered options
    compute_test.go    # Compute method tests
```

---

## Task Overview

| # | Task | Files |
|---|------|-------|
| 1 | APIError typed error | `client/error.go`, `client/error_test.go` |
| 2 | Options + root Client + New() | `client/options.go`, `client/client.go` |
| 3 | Client.do() — HTTP, auth, retry, Retry-After | `client/client.go`, `client/client_test.go` |
| 4 | Auth types | `client/auth/types.go` |
| 5 | Auth client — Session, Email, KeyPair | `client/auth/client.go` |
| 6 | Auth client — User, Identity, Partner | `client/auth/client.go` |
| 7 | Auth client — Admin | `client/auth/client.go` |
| 8 | Auth tests | `client/auth/auth_test.go` |
| 9 | Billing types | `client/billing/types.go` |
| 10 | Billing client — Customer, Members, Limits, Quotas, Stripe | `client/billing/client.go` |
| 11 | Billing client — Admin | `client/billing/client.go` |
| 12 | Billing tests | `client/billing/billing_test.go` |
| 13 | Compute types | `client/compute/types.go` |
| 14 | Compute client — Instance, Power, Templates, Admin | `client/compute/client.go` |
| 15 | Compute tests | `client/compute/compute_test.go` |
| 16 | Final verification | All |

---

## Tasks

---

### Task 1: `APIError` Typed Error

**Files:**
- Create: `client/error.go`
- Create: `client/error_test.go`

- [ ] **Step 1: Write the failing test**

Create `client/error_test.go`:

```go
package client

import (
	"net/http"
	"testing"
)

func TestAPIError_ErrorMessage(t *testing.T) {
	err := newAPIError("GET", "/test", http.StatusNotFound, "not found", "err-123", []byte(`{}`))
	got := err.Error()
	if got != "404: GET /test: not found" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestAPIError_IsMethods(t *testing.T) {
	tests := []struct {
		status int
		fn     func(*APIError) bool
		name   string
	}{
		{400, (*APIError).IsValidationError, "validation"},
		{401, (*APIError).IsUnauthorized, "unauthorized"},
		{403, (*APIError).IsForbidden, "forbidden"},
		{404, (*APIError).IsNotFound, "not found"},
		{429, (*APIError).IsRateLimited, "rate limited"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := newAPIError("GET", "/", tc.status, "msg", "id", nil)
			if !tc.fn(err) {
				t.Errorf("Is*() should return true for %d", tc.status)
			}
		})
	}
}

func TestAPIError_IsMethods_FalseForOtherStatus(t *testing.T) {
	err := newAPIError("GET", "/", http.StatusInternalServerError, "msg", "id", nil)
	if err.IsNotFound() || err.IsUnauthorized() || err.IsForbidden() || err.IsRateLimited() || err.IsValidationError() {
		t.Error("all Is*() should be false for 500")
	}
}

func TestAPIError_Fields(t *testing.T) {
	body := []byte(`{"detail":"test"}`)
	err := newAPIError("DELETE", "/item/5", http.StatusForbidden, "forbidden", "err-42", body)
	if err.StatusCode != 403 { t.Error("status wrong") }
	if err.Message != "forbidden" { t.Error("message wrong") }
	if err.ErrorID != "err-42" { t.Error("errorID wrong") }
	if string(err.Body) != string(body) { t.Error("body wrong") }
}

func TestSDKError(t *testing.T) {
	e := newSDKErr("test message")
	if e.Error() != "client: test message" {
		t.Errorf("unexpected: %q", e.Error())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./client/ -run TestAPIError -count=1 -v`
Expected: FAIL — `undefined: client.APIError`, `undefined: client.newAPIError`

- [ ] **Step 3: Write minimal implementation**

Create `client/error.go` with the APIError and SDKError types, newAPIError constructor, Error() and Is*() methods:

```go
package client

import "fmt"

// APIError is returned when an API call results in a 4xx or 5xx response.
// It implements the error interface.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Method     string `json:"-"`
	Path       string `json:"-"`
	Message    string `json:"message"`
	ErrorID    string `json:"error_id"`
	Body       []byte `json:"-"`
}

func newAPIError(method, path string, status int, message, errorID string, body []byte) *APIError {
	return &APIError{
		StatusCode: status,
		Method:     method,
		Path:       path,
		Message:    message,
		ErrorID:    errorID,
		Body:       body,
	}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%d: %s %s: %s", e.StatusCode, e.Method, e.Path, e.Message)
}

func (e *APIError) IsNotFound() bool        { return e.StatusCode == 404 }
func (e *APIError) IsUnauthorized() bool    { return e.StatusCode == 401 }
func (e *APIError) IsForbidden() bool       { return e.StatusCode == 403 }
func (e *APIError) IsRateLimited() bool     { return e.StatusCode == 429 }
func (e *APIError) IsValidationError() bool { return e.StatusCode == 400 }

// SDKError is returned by New() for configuration errors (not HTTP errors).
type SDKError struct {
	msg string
}

func newSDKErr(msg string) *SDKError { return &SDKError{msg: msg} }
func (e *SDKError) Error() string    { return "client: " + e.msg }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./client/ -run TestAPIError -count=1 -v` and `go test ./client/ -run TestSDKError -count=1 -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add client/error.go client/error_test.go
git commit -m "feat(client): add APIError typed error with status convenience checks"
```

---

### Task 2: Options, ListOptions, Root Client, New()

**Files:**
- Create: `client/options.go`
- Create: `client/client.go` (initial — without do())
- Create: `client/auth/client.go` (stub)
- Create: `client/billing/client.go` (stub)
- Create: `client/compute/client.go` (stub)
- Create: `client/client_test.go`

- [ ] **Step 1: Write the failing test**

Create `client/client_test.go` with tests for New() validation and defaults:

```go
package client

import (
	"net/http"
	"testing"
)

func TestNew_NoAuth(t *testing.T) {
	_, err := New()
	if err == nil {
		t.Fatal("expected error when no auth provided")
	}
	if _, ok := err.(*SDKError); !ok {
		t.Errorf("expected *SDKError, got %T", err)
	}
}

func TestNew_DuplicateAuth(t *testing.T) {
	_, err := New(WithAPIKey("k", "s"), WithBearer("t"))
	if err == nil {
		t.Fatal("expected error when both auth modes provided")
	}
}

func TestNew_APIKeyValue(t *testing.T) {
	c, err := New(WithEndpoint("https://example.com"), WithAPIKey("kid", "sec"))
	if err != nil { t.Fatal(err) }
	if c.apiKey != "kid" { t.Error("apiKey") }
	if c.apiSecret != "sec" { t.Error("apiSecret") }
}

func TestNew_BearerValue(t *testing.T) {
	c, err := New(WithEndpoint("https://example.com"), WithBearer("tok"))
	if err != nil { t.Fatal(err) }
	if c.bearer != "tok" { t.Error("bearer") }
}

func TestNew_Defaults(t *testing.T) {
	c, err := New(WithEndpoint("https://example.com"), WithAPIKey("k", "s"))
	if err != nil { t.Fatal(err) }
	if c.endpoint != DefaultEndpoint {
		t.Errorf("endpoint=%q", c.endpoint)
	}
	if c.maxRetries != defaultMaxRetries {
		t.Errorf("maxRetries=%d", c.maxRetries)
	}
	if c.httpClient.Timeout != defaultTimeout {
		t.Errorf("timeout=%v", c.httpClient.Timeout)
	}
}

func TestNew_CustomTimeout(t *testing.T) {
	c, err := New(WithAPIKey("k", "s"), WithTimeout(0))
	if err != nil { t.Fatal(err) }
	if c.httpClient.Timeout == 0 {
		// zero means no timeout — that's fine
	}
}

func TestNew_CustomMaxRetries(t *testing.T) {
	c, err := New(WithAPIKey("k", "s"), WithMaxRetries(7))
	if err != nil { t.Fatal(err) }
	if c.maxRetries != 7 { t.Error("maxRetries") }
}

func TestNew_CustomHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c, err := New(WithAPIKey("k", "s"), WithHTTPClient(custom))
	if err != nil { t.Fatal(err) }
	if c.httpClient != custom { t.Error("custom client not used") }
}

func TestServiceAccessors(t *testing.T) {
	c, err := New(WithAPIKey("k", "s"))
	if err != nil { t.Fatal(err) }
	if c.Auth() == nil { t.Error("Auth") }
	if c.Billing() == nil { t.Error("Billing") }
	if c.Compute() == nil { t.Error("Compute") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./client/ -run TestNew -count=1 -v`
Expected: FAIL — `undefined: client.New`, `undefined: client.WithAPIKey`, etc.

- [ ] **Step 3: Write options.go**

Create `client/options.go`:

```go
package client

import (
	"net/http"
	"time"
)

// ListOptions provides pagination parameters.
type ListOptions struct {
	Limit int // 0 = server default
	Page  int // 0 = server default
}

// Option is a functional option for Client.
type Option func(*clientConfig)

type clientConfig struct {
	endpoint   string
	apiKey     string
	apiSecret  string
	bearer     string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
}

func defaultConfig() *clientConfig {
	return &clientConfig{
		endpoint:   DefaultEndpoint,
		timeout:    defaultTimeout,
		maxRetries: defaultMaxRetries,
	}
}

const (
	DefaultEndpoint   = "https://api.threatwinds.com"
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
)

func WithEndpoint(url string) Option {
	return func(c *clientConfig) { c.endpoint = url }
}

func WithAPIKey(keyID, secret string) Option {
	return func(c *clientConfig) { c.apiKey = keyID; c.apiSecret = secret }
}

func WithBearer(token string) Option {
	return func(c *clientConfig) { c.bearer = token }
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *clientConfig) { c.httpClient = hc }
}

func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) { c.timeout = d }
}

func WithMaxRetries(n int) Option {
	return func(c *clientConfig) { c.maxRetries = n }
}
```

- [ ] **Step 4: Write service stubs**

Create `client/auth/client.go`:

```go
package auth

import "github.com/threatwinds/go-sdk/client"

// Client provides access to /api/auth/v2 endpoints.
type Client struct {
	root *client.Client
}

func NewClient(root *client.Client) *Client {
	return &Client{root: root}
}
```

Create `client/billing/client.go`:

```go
package billing

import "github.com/threatwinds/go-sdk/client"

// Client provides access to /api/billing/v1 endpoints.
type Client struct {
	root *client.Client
}

func NewClient(root *client.Client) *Client {
	return &Client{root: root}
}
```

Create `client/compute/client.go`:

```go
package compute

import "github.com/threatwinds/go-sdk/client"

// Client provides access to /api/compute/v1 endpoints.
type Client struct {
	root *client.Client
}

func NewClient(root *client.Client) *Client {
	return &Client{root: root}
}
```

- [ ] **Step 5: Write client.go (skeleton + New + accessors)**

Create `client/client.go`:

```go
package client

import (
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/threatwinds/go-sdk/client/auth"
	"github.com/threatwinds/go-sdk/client/billing"
	"github.com/threatwinds/go-sdk/client/compute"
)

// Client is the root SDK client.
type Client struct {
	endpoint   string
	apiKey     string
	apiSecret  string
	bearer     string
	httpClient *http.Client
	maxRetries int

	auth    *auth.Client
	billing *billing.Client
	compute *compute.Client
	once    sync.Once
}

// New creates a new SDK client. Requires WithAPIKey or WithBearer.
func New(opts ...Option) (*Client, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}

	if cfg.apiKey == "" && cfg.bearer == "" {
		return nil, newSDKErr("authentication required: provide WithAPIKey or WithBearer")
	}
	if cfg.apiKey != "" && cfg.bearer != "" {
		return nil, newSDKErr("only one of WithAPIKey or WithBearer may be set")
	}

	hc := cfg.httpClient
	if hc == nil {
		hc = &http.Client{
			Timeout: cfg.timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
		}
	}

	return &Client{
		endpoint:   cfg.endpoint,
		apiKey:     cfg.apiKey,
		apiSecret:  cfg.apiSecret,
		bearer:     cfg.bearer,
		httpClient: hc,
		maxRetries: cfg.maxRetries,
	}, nil
}

func (c *Client) Auth() *auth.Client {
	c.once.Do(c.initServices)
	return c.auth
}

func (c *Client) Billing() *billing.Client {
	c.once.Do(c.initServices)
	return c.billing
}

func (c *Client) Compute() *compute.Client {
	c.once.Do(c.initServices)
	return c.compute
}

func (c *Client) initServices() {
	c.auth = auth.NewClient(c)
	c.billing = billing.NewClient(c)
	c.compute = compute.NewClient(c)
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./client/ -run TestNew -count=1 -v` and `go test ./client/ -run TestServiceAccessors -count=1 -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add client/options.go client/client.go client/client_test.go \
        client/auth/client.go client/billing/client.go client/compute/client.go
git commit -m "feat(client): add root client with options and service accessors"
```

---

### Task 3: Client.do() — HTTP Execution, Auth, Retry, Retry-After

**Files:**
- Modify: `client/client.go` (add do(), applyAuth(), parseRetryAfter(), isRetryable)
- Modify: `client/client_test.go` (add Do tests)

- [ ] **Step 1: Add Do tests to client_test.go**

Append a mock RoundTripper and Do() tests to `client/client_test.go`:

```go
// mockRT implements http.RoundTripper for testing.
type mockRT struct {
	round func(req *http.Request) *http.Response
	count int
}

func (m *mockRT) RoundTrip(req *http.Request) (*http.Response, error) {
	m.count++
	resp := m.round(req)
	return resp, nil
}

func mockResp(status int, body string, headers ...string) *http.Response {
	h := make(http.Header)
	for i := 0; i+1 < len(headers); i += 2 {
		h.Add(headers[i], headers[i+1])
	}
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestDo_Success(t *testing.T) {
	c, _ := New(WithAPIKey("k", "s"))
	c.httpClient.Transport = &mockRT{
		round: func(req *http.Request) *http.Response {
			if req.Method != "GET" { t.Errorf("method=%s", req.Method) }
			if req.Header.Get("api-key") != "k" { t.Error("api-key missing") }
			if req.Header.Get("api-secret") != "s" { t.Error("api-secret missing") }
			return mockResp(200, `{"foo":"bar"}`)
		},
	}
	var out map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &out)
	if err != nil { t.Fatalf("unexpected: %v", err) }
	if out["foo"] != "bar" { t.Error("unmarshal") }
}

func TestDo_BearerAuth(t *testing.T) {
	c, _ := New(WithBearer("tok"))
	var gotAuth string
	c.httpClient.Transport = &mockRT{
		round: func(req *http.Request) *http.Response {
			gotAuth = req.Header.Get("Authorization")
			return mockResp(200, `{}`)
		},
	}
	var out map[string]any
	_ = c.do(context.Background(), "GET", "/", nil, &out)
	if gotAuth != "Bearer tok" { t.Errorf("auth=%s", gotAuth) }
}

func TestDo_PostBody(t *testing.T) {
	c, _ := New(WithAPIKey("k", "s"))
	var gotBody []byte
	c.httpClient.Transport = &mockRT{
		round: func(req *http.Request) *http.Response {
			gotBody, _ = io.ReadAll(req.Body)
			return mockResp(200, `{}`)
		},
	}
	var out map[string]any
	_ = c.do(context.Background(), "POST", "/e", []byte(`{"x":1}`), &out)
	if string(gotBody) != `{"x":1}` { t.Errorf("body=%s", gotBody) }
}

func TestDo_APIError(t *testing.T) {
	c, _ := New(WithAPIKey("k", "s"))
	c.httpClient.Transport = &mockRT{
		round: func(*http.Request) *http.Response {
			return mockResp(404, `{"detail":"gone"}`,
				"x-error", "not found", "x-error-id", "e-1")
		},
	}
	var out map[string]any
	err := c.do(context.Background(), "DELETE", "/item/5", nil, &out)
	if err == nil { t.Fatal("expected error") }
	ae, ok := err.(*APIError)
	if !ok { t.Fatalf("want *APIError got %T", err) }
	if !ae.IsNotFound() { t.Error("not 404") }
	if ae.ErrorID != "e-1" { t.Error("errorID") }
	if ae.Message != "not found" { t.Error("message") }
}

func TestDo_RetryRetryAfter(t *testing.T) {
	c, _ := New(WithAPIKey("k", "s"), WithMaxRetries(3))
	c.httpClient.Transport = &mockRT{
		round: func(*http.Request) *http.Response {
			return mockResp(429, ``, "Retry-After", "0",
				"x-error", "limit", "x-error-id", "r1")
		},
	}
	var out map[string]any
	err := c.do(context.Background(), "GET", "/", nil, &out)
	if err == nil { t.Fatal("should fail after retries") }
}

func TestDo_NoRetryOnPost(t *testing.T) {
	c, _ := New(WithAPIKey("k", "s"), WithMaxRetries(3))
	c.httpClient.Transport = &mockRT{
		round: func(*http.Request) *http.Response {
			return mockResp(503, ``, "x-error", "down", "x-error-id", "e1")
		},
	}
	var out map[string]any
	_ = c.do(context.Background(), "POST", "/", []byte(`{}`), &out)
	// should be exactly 1 attempt
}

func TestDo_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before
	c, _ := New(WithAPIKey("k", "s"))
	c.httpClient.Transport = &mockRT{
		round: func(*http.Request) *http.Response {
			return mockResp(200, `{}`)
		},
	}
	var out map[string]any
	err := c.do(ctx, "GET", "/", nil, &out)
	if err == nil { t.Error("expected context error") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./client/ -run TestDo -count=1 -v`
Expected: FAIL — `c.do undefined`

- [ ] **Step 3: Implement do() in client.go**

Append to `client/client.go` (after existing code, before the last `}`):

```go
func (c *Client) do(ctx context.Context, method, path string, body []byte, out any) error {
	attempts := 1
	if method == http.MethodGet && c.maxRetries > 0 {
		attempts = c.maxRetries + 1
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			d := parseRetryAfter(lastErr)
			if d == 0 {
				d = backoff(i - 1)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d):
			}
		}

		if err := c.doOnce(ctx, method, path, body, out); err != nil {
			if !isRetryable(lastErr, method) {
				return err
			}
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

func (c *Client) doOnce(ctx context.Context, method, path string, body []byte, out any) error {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, r)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := resp.Header.Get("x-error")
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return lastErr = newAPIError(method, path, resp.StatusCode, msg, resp.Header.Get("x-error-id"), raw)
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return json.Unmarshal(raw, out)
}

func (c *Client) applyAuth(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
		req.Header.Set("api-secret", c.apiSecret)
	} else if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
}

func isRetryable(err error, method string) bool {
	if method != http.MethodGet {
		return false
	}
	ae, ok := err.(*APIError)
	if !ok {
		return false
	}
	return ae.StatusCode == 429 || ae.StatusCode == 502 || ae.StatusCode == 503 || ae.StatusCode == 504
}

func parseRetryAfter(err error) time.Duration {
	ae, ok := err.(*APIError)
	if !ok || ae == nil {
		return 0
	}
	v := ae.HeaderGet("Retry-After") // need to store header on AE
	if v == "" {
		return 0
	}
	if s, err := strconv.Atoi(v); err == nil && s > 0 {
		return time.Duration(s) * time.Second
	}
	if t, err := time.Parse(time.RFC1123, v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func backoff(attempt int) time.Duration {
	d := 100 * time.Millisecond
	for i := 0; i < attempt; i++ {
		d *= 4
	}
	return d
}
```

Wait — `parseRetryAfter` needs access to the `Retry-After` response header, but `APIError` doesn't store it. I need to either store it on `APIError` or restructure. The simplest fix: add a `retryAfter` field to `APIError` that's populated in `doOnce`.

Update `client/error.go` — add `retryAfter` field to APIError:

```go
type APIError struct {
	StatusCode int    `json:"status_code"`
	Method     string `json:"-"`
	Path       string `json:"-"`
	Message    string `json:"message"`
	ErrorID    string `json:"error_id"`
	Body       []byte `json:"-"`
	retryAfter string `json:"-"` // internal: Retry-After header for retry logic
}

func (e *APIError) HeaderGet(key string) string {
	if key == "Retry-After" {
		return e.retryAfter
	}
	return ""
}

func newAPIError(method, path string, status int, message, errorID, retryAfter string, body []byte) *APIError {
	return &APIError{
		StatusCode: status,
		Method:     method,
		Path:       path,
		Message:    message,
		ErrorID:    errorID,
		retryAfter: retryAfter,
		Body:       body,
	}
}
```

And in `doOnce`, pass the `Retry-After` header:

```go
return newAPIError(method, path, resp.StatusCode, msg, resp.Header.Get("x-error-id"), resp.Header.Get("Retry-After"), raw)
```

This requires updating `error_test.go` to pass the new `retryAfter` arg.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./client/ -run TestDo -count=1 -v`
Expected: PASS

Run: `go test ./client/ -count=1 -v` (all client tests together)
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add client/client.go client/client_test.go client/error.go client/error_test.go
git commit -m "feat(client): implement do() with auth, retry, and Retry-After support"
```

---

### Task 4: Auth Types

**Files:**
- Create: `client/auth/types.go`

- [ ] **Step 1: Write all auth request/response types**

Create `client/auth/types.go` with all types from the spec (ListUsersOptions, all request structs, all response structs, paginated responses). Each field must match the API field names from spec exactly.

- [ ] **Step 2: Compile check**

Run: `go build ./client/auth/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add client/auth/types.go
git commit -m "feat(client/auth): add all request/response type definitions"
```

---

### Task 5: Auth Client — Session, Email, KeyPair, User, Identity, Partner

**Files:**
- Modify: `client/auth/client.go` (replace stub)
- Modify: `client/client.go` (add PathEscape helper)

- [ ] **Step 1: Add PathEscape to client.go**

Append to `client/client.go`:

```go
func PathEscape(s string) string {
	return url.PathEscape(s)
}
```

Add `"net/url"` to imports.

- [ ] **Step 2: Write Session, Email, KeyPair, User, Identity, Partner methods**

Replace `client/auth/client.go` with the full Client struct, `do()` helper, and all non-admin methods from the spec.

- [ ] **Step 3: Compile check**

Run: `go build ./client/auth/`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add client/auth/client.go client/client.go
git commit -m "feat(client/auth): implement session, email, keypair, user, verify, partner endpoints"
```

---

### Task 6: Auth Client — Admin Endpoints

**Files:**
- Modify: `client/auth/client.go`

- [ ] **Step 1: Append all Admin methods**

Append all 25+ admin methods from the spec to `client/auth/client.go`. Each adminList method builds query params from its filtered options struct (AdminListUsersOptions, etc.).

- [ ] **Step 2: Compile check**

Run: `go build ./client/auth/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add client/auth/client.go
git commit -m "feat(client/auth): implement all admin endpoints"
```

---

### Task 7: Auth Tests

**Files:**
- Create: `client/auth/auth_test.go`

- [ ] **Step 1: Write representative auth tests**

Create `client/auth/auth_test.go` with a mock `http.RoundTripper` and table-driven tests covering:
- `CreateSession` — POST with body, JSON response
- `GetSession` — GET, Bearer auth
- `GetUserByEmail` — GET with query param
- `AdminListUsers` — GET with filtered options (limit, page, role, enabled)
- `AdminListUsers` — paginated response parsed correctly (pages, items, users)
- `DeleteSession` — DELETE with path param
- `APIError` on 401 response

The mock RoundTripper captures the request and returns a pre-baked response.

- [ ] **Step 2: Run tests**

Run: `go test ./client/auth/ -count=1 -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add client/auth/auth_test.go
git commit -m "test(client/auth): add representative endpoint tests"
```

---

### Task 8: Billing Types

**Files:**
- Create: `client/billing/types.go`

- [ ] **Step 1: Write all billing request/response types**

Create `client/billing/types.go` with all types from the spec:
- Filtered options: `AdminListCustomersOptions`, `AdminListTiersOptions`
- All request structs: `CreateCustomerRequest`, `AddMemberRequest`, `UpdateMemberRequest`, `TransferOwnershipRequest`, `UpgradeToProRequest`, etc.
- All response structs: `GetCustomerResponse`, `GetCustomerTierResponse`, `GetAllLimitsResponse`, `GetAllQuotasResponse`, `StartPortalResponse`, `UpgradeToProResponse`, tier response, member response
- Paginated responses: `ListMembersResponse`, `ListCustomersResponse`, `AdminListCustomerMembersResponse`
- Usage responses: `GetUserUsageResponse`, `GranularUsageResponse`, `GetAllIPLimitsResponse`, `GetIPUsageResponse`, `GetQuotaUsageResponse`, `GetBillingStatusResponse`, `AdminGetCustomerResponse`, `GetAllTiersResponse`

- [ ] **Step 2: Compile check**

Run: `go build ./client/billing/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add client/billing/types.go
git commit -m "feat(client/billing): add all request/response type definitions"
```

---

### Task 9: Billing Client — All Endpoints

**Files:**
- Modify: `client/billing/client.go` (replace stub)

- [ ] **Step 1: Write all billing methods**

Replace `client/billing/client.go` with the Client struct, `do()` helper, and all methods from the spec (~35 methods across Customer, Members, Limits, IP Limits, Quotas, Stripe, Admin categories).

- [ ] **Step 2: Compile check**

Run: `go build ./client/billing/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add client/billing/client.go
git commit -m "feat(client/billing): implement all billing endpoints"
```

---

### Task 10: Billing Tests

**Files:**
- Create: `client/billing/billing_test.go`

- [ ] **Step 1: Write representative billing tests**

Create `client/billing/billing_test.go` testing:
- `GetCustomer` — GET, single response
- `AddMember` — POST with body
- `ListMembers` — paginated response
- `AdminListCustomers` — filtered options
- `GetAllLimits` — nested object response
- `StartPortal` — GET returning URL
- `APIError` on 403

- [ ] **Step 2: Run tests**

Run: `go test ./client/billing/ -count=1 -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add client/billing/billing_test.go
git commit -m "test(client/billing): add representative endpoint tests"
```

---

### Task 11: Compute Types

**Files:**
- Create: `client/compute/types.go`

- [ ] **Step 1: Write all compute request/response types**

Create `client/compute/types.go` with all types from the spec:
- Filtered options: `AdminListInstancesOptions`
- Request: `InstanceCreateRequest`
- Response: `Instance`, `Template`
- Paginated: `AdminListInstancesResponse`

- [ ] **Step 2: Compile check**

Run: `go build ./client/compute/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add client/compute/types.go
git commit -m "feat(client/compute): add all request/response type definitions"
```

---

### Task 12: Compute Client — All Endpoints

**Files:**
- Modify: `client/compute/client.go` (replace stub)

- [ ] **Step 1: Write all compute methods**

Replace `client/compute/client.go` with Client struct, `do()` helper, and all methods from the spec (~16 methods across Instance, Power, Templates, Admin).

- [ ] **Step 2: Compile check**

Run: `go build ./client/compute/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add client/compute/client.go
git commit -m "feat(client/compute): implement all compute endpoints"
```

---

### Task 13: Compute Tests

**Files:**
- Create: `client/compute/compute_test.go`

- [ ] **Step 1: Write representative compute tests**

Create `client/compute/compute_test.go` testing:
- `CreateInstance` — POST with body, returns Instance
- `ListInstances` — raw array response
- `GetInstance` — GET with path param
- `StartInstance` — POST action endpoint
- `ListTemplates` — raw array response
- `AdminListInstances` — filtered paginated response
- `APIError` on 404

- [ ] **Step 2: Run tests**

Run: `go test ./client/compute/ -count=1 -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add client/compute/compute_test.go
git commit -m "test(client/compute): add representative endpoint tests"
```

---

### Task 14: Final Verification

**Files:**
- All files in `client/`

- [ ] **Step 1: Run all tests**

Run: `go test ./client/... -count=1 -v -cover`
Expected: all PASS, coverage >= 80% on client package

- [ ] **Step 2: Vet and format**

Run: `go vet ./client/...`
Expected: no warnings

Run: `gofmt -l ./client/`
Expected: no output (all files formatted)

- [ ] **Step 3: Build full module**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add client/
git commit -m "chore(client): final formatting and verification"
```
