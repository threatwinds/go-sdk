package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

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

// — Instance —

// CreateInstance creates a new compute instance.
func (c *Client) CreateInstance(ctx context.Context, req InstanceCreateRequest) (*Instance, error) {
	b, _ := json.Marshal(req)
	var out Instance
	return &out, c.do(ctx, "POST", "/instances", b, &out)
}

// ListInstances lists all instances for the current user.
func (c *Client) ListInstances(ctx context.Context) ([]Instance, error) {
	var out []Instance
	return out, c.do(ctx, "GET", "/instances", nil, &out)
}

// GetInstance fetches a single instance by ID.
func (c *Client) GetInstance(ctx context.Context, instanceID string) (*Instance, error) {
	var out Instance
	return &out, c.do(ctx, "GET", "/instances/"+url.PathEscape(instanceID), nil, &out)
}

// DeleteInstance deletes an instance by ID.
func (c *Client) DeleteInstance(ctx context.Context, instanceID string) error {
	return c.do(ctx, "DELETE", "/instances/"+url.PathEscape(instanceID), nil, new(struct{}))
}

// — Power Management —

// StartInstance starts a stopped instance.
func (c *Client) StartInstance(ctx context.Context, instanceID string) error {
	return c.do(ctx, "POST", "/instances/"+url.PathEscape(instanceID)+"/start", nil, new(struct{}))
}

// StopInstance stops a running instance.
func (c *Client) StopInstance(ctx context.Context, instanceID string) error {
	return c.do(ctx, "POST", "/instances/"+url.PathEscape(instanceID)+"/stop", nil, new(struct{}))
}

// RestartInstance restarts an instance.
func (c *Client) RestartInstance(ctx context.Context, instanceID string) error {
	return c.do(ctx, "POST", "/instances/"+url.PathEscape(instanceID)+"/restart", nil, new(struct{}))
}

// ResetInstance resets an instance to its original state.
func (c *Client) ResetInstance(ctx context.Context, instanceID string) error {
	return c.do(ctx, "POST", "/instances/"+url.PathEscape(instanceID)+"/reset", nil, new(struct{}))
}

// — Templates —

// ListTemplates lists all available instance templates.
func (c *Client) ListTemplates(ctx context.Context) ([]Template, error) {
	var out []Template
	return out, c.do(ctx, "GET", "/templates", nil, &out)
}

// ListTemplateZones lists all zones available for a template.
func (c *Client) ListTemplateZones(ctx context.Context, templateID string) ([]string, error) {
	var out []string
	return out, c.do(ctx, "GET", "/templates/"+url.PathEscape(templateID)+"/zones", nil, &out)
}

// — Admin —

// AdminListInstances lists instances with optional filtering.
func (c *Client) AdminListInstances(ctx context.Context, opts *AdminListInstancesOptions) (*AdminListInstancesResponse, error) {
	q := make(url.Values)
	if opts != nil {
		if opts.Limit > 0 {
			q.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Page > 0 {
			q.Set("page", fmt.Sprintf("%d", opts.Page))
		}
		if opts.UserID != "" {
			q.Set("userID", opts.UserID)
		}
		if opts.CustomerID != "" {
			q.Set("customerID", opts.CustomerID)
		}
		if opts.Status != "" {
			q.Set("status", opts.Status)
		}
		if opts.Zone != "" {
			q.Set("zone", opts.Zone)
		}
		if opts.TemplateID != "" {
			q.Set("templateID", opts.TemplateID)
		}
	}
	path := "/admin/instances"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out AdminListInstancesResponse
	return &out, c.do(ctx, "GET", path, nil, &out)
}

// AdminGetInstance fetches an instance by ID as admin.
func (c *Client) AdminGetInstance(ctx context.Context, instanceID string) (*Instance, error) {
	var out Instance
	return &out, c.do(ctx, "GET", "/admin/instances/"+url.PathEscape(instanceID), nil, &out)
}

// AdminDeleteInstance deletes an instance by ID as admin.
func (c *Client) AdminDeleteInstance(ctx context.Context, instanceID string) error {
	return c.do(ctx, "DELETE", "/admin/instances/"+url.PathEscape(instanceID), nil, new(struct{}))
}

// AdminStartInstance starts an instance as admin.
func (c *Client) AdminStartInstance(ctx context.Context, instanceID string) error {
	return c.do(ctx, "POST", "/admin/instances/"+url.PathEscape(instanceID)+"/start", nil, new(struct{}))
}

// AdminStopInstance stops an instance as admin.
func (c *Client) AdminStopInstance(ctx context.Context, instanceID string) error {
	return c.do(ctx, "POST", "/admin/instances/"+url.PathEscape(instanceID)+"/stop", nil, new(struct{}))
}

// AdminRestartInstance restarts an instance as admin.
func (c *Client) AdminRestartInstance(ctx context.Context, instanceID string) error {
	return c.do(ctx, "POST", "/admin/instances/"+url.PathEscape(instanceID)+"/restart", nil, new(struct{}))
}

// AdminResetInstance resets an instance as admin.
func (c *Client) AdminResetInstance(ctx context.Context, instanceID string) error {
	return c.do(ctx, "POST", "/admin/instances/"+url.PathEscape(instanceID)+"/reset", nil, new(struct{}))
}
