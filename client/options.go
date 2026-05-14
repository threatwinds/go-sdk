package client

import (
	"net/http"
	"time"
)

// DefaultEndpoint is the default ThreatWinds API base URL.
const DefaultEndpoint = "https://api.threatwinds.com"

const (
	defaultTimeout     = 30 * time.Second
	defaultMaxRetries  = 3
)

// ListOptions holds pagination parameters for list endpoints.
type ListOptions struct {
	Limit int
	Page  int
}

// Option configures a Client via the functional options pattern.
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

// WithEndpoint sets the API base URL.
func WithEndpoint(url string) Option {
	return func(c *clientConfig) {
		c.endpoint = url
	}
}

// WithAPIKey sets API key authentication (key ID + secret).
func WithAPIKey(keyID, secret string) Option {
	return func(c *clientConfig) {
		c.apiKey = keyID
		c.apiSecret = secret
	}
}

// WithBearer sets bearer token authentication.
func WithBearer(token string) Option {
	return func(c *clientConfig) {
		c.bearer = token
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *clientConfig) {
		c.httpClient = hc
	}
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) Option {
	return func(c *clientConfig) {
		c.maxRetries = n
	}
}
