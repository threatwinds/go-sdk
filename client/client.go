package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/threatwinds/go-sdk/client/auth"
	"github.com/threatwinds/go-sdk/client/billing"
	"github.com/threatwinds/go-sdk/client/compute"
)

const userAgent = "threatwinds-go-sdk"

// Client is the root entry point for the ThreatWinds API SDK.
type Client struct {
	endpoint   string
	apiKey     string
	apiSecret  string
	bearer     string
	httpClient *http.Client
	transport  *http.Transport // non-nil only when SDK created the transport
	maxRetries int

	auth    *auth.Client
	billing *billing.Client
	compute *compute.Client
	once    sync.Once
}

// New creates a configured API client. At least one authentication option
// (WithAPIKey or WithBearer) must be provided, but not both.
func New(opts ...Option) (*Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate authentication: exactly one method required.
	hasAPIKey := cfg.apiKey != "" && cfg.apiSecret != ""
	hasBearer := cfg.bearer != ""
	if !hasAPIKey && !hasBearer {
		return nil, newSDKErr("authentication required: provide WithAPIKey or WithBearer")
	}
	if hasAPIKey && hasBearer {
		return nil, newSDKErr("conflicting authentication: use WithAPIKey or WithBearer, not both")
	}

	// Build HTTP client.
	var (
		httpClient *http.Client
		transport  *http.Transport
	)
	if cfg.httpClient != nil {
		httpClient = cfg.httpClient
	} else {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		httpClient = &http.Client{
			Timeout:   cfg.timeout,
			Transport: transport,
		}
	}

	return &Client{
		endpoint:   cfg.endpoint,
		apiKey:     cfg.apiKey,
		apiSecret:  cfg.apiSecret,
		bearer:     cfg.bearer,
		httpClient: httpClient,
		transport:  transport,
		maxRetries: cfg.maxRetries,
	}, nil
}

// Auth returns the Auth service client (lazy-initialized).
func (c *Client) Auth() *auth.Client {
	c.once.Do(c.initServices)
	return c.auth
}

// Billing returns the Billing service client (lazy-initialized).
func (c *Client) Billing() *billing.Client {
	c.once.Do(c.initServices)
	return c.billing
}

// Compute returns the Compute service client (lazy-initialized).
func (c *Client) Compute() *compute.Client {
	c.once.Do(c.initServices)
	return c.compute
}

// Close releases idle HTTP connections held by the SDK's internal transport.
// It is a no-op when the caller supplied their own http.Client via
// WithHTTPClient. Callers should invoke Close() when they are done with the
// client to avoid leaking TCP connections in long-running processes.
func (c *Client) Close() {
	if c.transport != nil {
		c.transport.CloseIdleConnections()
	}
}

func (c *Client) initServices() {
	d := func(ctx context.Context, method, path string, body []byte, out any) error {
		return c.do(ctx, method, path, body, out)
	}
	c.auth = auth.NewClient(d)
	c.billing = billing.NewClient(d)
	c.compute = compute.NewClient(d)
}

// ---------------------------------------------------------------------------
// do() — HTTP execution engine with retry, auth, and Retry-After support.
// ---------------------------------------------------------------------------

// do executes an HTTP request with retry logic for GET requests.
// For GET requests, it retries up to maxRetries times on retryable
// statuses (429, 502, 503, 504). Non-GET requests are not retried.
func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}

		err := c.doOnce(ctx, method, path, body, out)
		if err == nil {
			return nil
		}
		lastErr = err

		if !c.isRetryable(err, method) {
			return err
		}

		delay := c.parseRetryAfter(err)
		if delay == 0 {
			delay = c.backoff(attempt)
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

// doOnce executes a single HTTP request without retry.
func (c *Client) doOnce(ctx context.Context, method, path string, body, out interface{}) error {
	// Build URL.
	urlStr := c.endpoint + path

	// Create request body if present.
	var reader io.Reader
	if body != nil {
		var data []byte
		if b, ok := body.([]byte); ok {
			data = b
		} else {
			var err error
			data, err = json.Marshal(body)
			if err != nil {
				return err
			}
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, reader)
	if err != nil {
		return err
	}

	// Set Content-Type for requests with a body.
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Apply authentication headers.
	c.applyAuth(req)

	// Set User-Agent.
	req.Header.Set("User-Agent", userAgent)

	// Execute request.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read full response body.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Handle error responses.
	if resp.StatusCode >= 400 {
		message := resp.Header.Get("X-Error")
		errorID := resp.Header.Get("X-Error-Id")
		retryAfter := resp.Header.Get("Retry-After")
		return newAPIError(method, path, resp.StatusCode, message, errorID, retryAfter, respBody)
	}

	// Handle 204 No Content.
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	// Unmarshal JSON response into out.
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return err
		}
	}

	return nil
}

// applyAuth sets the appropriate authentication headers on the request.
func (c *Client) applyAuth(req *http.Request) {
	if c.apiKey != "" && c.apiSecret != "" {
		req.Header.Set("Api-Key", c.apiKey)
		req.Header.Set("Api-Secret", c.apiSecret)
	} else if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
}

// isRetryable returns true if the error is retryable for the given HTTP method.
// Only GET requests with retryable status codes (429, 502, 503, 504) are retried.
func (c *Client) isRetryable(err error, method string) bool {
	if method != http.MethodGet {
		return false
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	switch apiErr.StatusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return true
	}
	return false
}

// parseRetryAfter parses the Retry-After header from an APIError.
// If the value is numeric, it's treated as seconds.
// If it's an HTTP date (RFC 1123), it computes the time until that date.
// If empty or invalid, returns 0 (fallback to exponential backoff).
func (c *Client) parseRetryAfter(err error) time.Duration {
	apiErr, ok := err.(*APIError)
	if !ok {
		return 0
	}
	raw := apiErr.RetryAfter()
	if raw == "" {
		return 0
	}

	// Try numeric (seconds) first.
	if seconds, err := strconv.Atoi(raw); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}

	// Try HTTP date (RFC 1123).
	if t, err := time.Parse(time.RFC1123, raw); err == nil {
		until := time.Until(t)
		if until > 0 {
			return until
		}
		return 0
	}

	return 0
}

// backoff returns the exponential backoff duration for the given attempt.
// Base: 100ms, multiplier: 4x. Formula: 100ms * 4^attempt.
func (c *Client) backoff(attempt int) time.Duration {
	d := 100 * time.Millisecond
	for i := 0; i < attempt; i++ {
		d *= 4
	}
	return d
}
