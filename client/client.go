package client

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"sync"

	"github.com/threatwinds/go-sdk/client/auth"
	"github.com/threatwinds/go-sdk/client/billing"
	"github.com/threatwinds/go-sdk/client/compute"
)

// Client is the root entry point for the ThreatWinds API SDK.
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

// New creates a configured API client. At least one authentication option
// (WithAPIKey or WithBearer) must be provided, but not both.
func New(opts ...Option) (*Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate authentication: exactly one method required.
	hasAPIKey := cfg.apiKey != "" || cfg.apiSecret != ""
	hasBearer := cfg.bearer != ""
	if !hasAPIKey && !hasBearer {
		return nil, newSDKErr("authentication required: provide WithAPIKey or WithBearer")
	}
	if hasAPIKey && hasBearer {
		return nil, newSDKErr("conflicting authentication: use WithAPIKey or WithBearer, not both")
	}

	// Build HTTP client.
	var httpClient *http.Client
	if cfg.httpClient != nil {
		httpClient = cfg.httpClient
	} else {
		httpClient = &http.Client{
			Timeout: cfg.timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		}
	}

	return &Client{
		endpoint:   cfg.endpoint,
		apiKey:     cfg.apiKey,
		apiSecret:  cfg.apiSecret,
		bearer:     cfg.bearer,
		httpClient: httpClient,
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

func (c *Client) initServices() {
	c.auth = auth.NewClient(c)
	c.billing = billing.NewClient(c)
	c.compute = compute.NewClient(c)
}

// PathEscape wraps url.PathEscape for use by service clients.
func (c *Client) PathEscape(s string) string {
	return url.PathEscape(s)
}
