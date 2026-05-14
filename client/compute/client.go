package compute

import "context"

// doFunc is the signature for making HTTP requests to the API.
type doFunc func(ctx context.Context, method, path string, body []byte, out any) error

// Client provides access to the Compute API endpoints.
type Client struct {
	do doFunc
}

// NewClient creates a new Compute client backed by the root SDK client's do function.
func NewClient(d doFunc) *Client {
	return &Client{do: d}
}
