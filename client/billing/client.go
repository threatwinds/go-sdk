package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// doFunc is the signature for making HTTP requests to the API.
type doFunc func(ctx context.Context, method, path string, body []byte, out any) error

// Client provides access to the Billing API endpoints.
type Client struct {
	do doFunc
}

// NewClient creates a new Billing client backed by the root SDK client's do function.
func NewClient(d doFunc) *Client {
	return &Client{do: d}
}

// — Customer —

// CreateCustomer creates a new customer.
func (c *Client) CreateCustomer(ctx context.Context, req CreateCustomerRequest) error {
	b, _ := json.Marshal(req)
	return c.do(ctx, "POST", "/customer", b, new(struct{}))
}

// GetCustomer returns the current customer.
func (c *Client) GetCustomer(ctx context.Context) (*GetCustomerResponse, error) {
	var out GetCustomerResponse
	return &out, c.do(ctx, "GET", "/customer", nil, &out)
}

// DeleteCustomer deletes the current customer.
func (c *Client) DeleteCustomer(ctx context.Context) error {
	return c.do(ctx, "DELETE", "/customer", nil, new(struct{}))
}

// GetCustomerTier returns the customer's current tier.
func (c *Client) GetCustomerTier(ctx context.Context) (*GetCustomerTierResponse, error) {
	var out GetCustomerTierResponse
	return &out, c.do(ctx, "GET", "/customer/tier", nil, &out)
}

// LeaveCustomer removes the current user from the customer.
func (c *Client) LeaveCustomer(ctx context.Context) error {
	return c.do(ctx, "DELETE", "/customer/leave", nil, new(struct{}))
}

// TransferOwnership transfers customer ownership to another user.
func (c *Client) TransferOwnership(ctx context.Context, req TransferOwnershipRequest) error {
	b, _ := json.Marshal(req)
	return c.do(ctx, "POST", "/customer/transfer-ownership", b, new(struct{}))
}

// — Members —

// ListMembers lists members of the current customer with optional pagination.
func (c *Client) ListMembers(ctx context.Context, opts *ListOpts) (*ListMembersResponse, error) {
	q := make(url.Values)
	if opts != nil {
		if opts.Limit > 0 {
			q.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Page > 0 {
			q.Set("page", fmt.Sprintf("%d", opts.Page))
		}
	}
	path := "/customer/members"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out ListMembersResponse
	return &out, c.do(ctx, "GET", path, nil, &out)
}

// AddMember adds a member to the current customer.
func (c *Client) AddMember(ctx context.Context, req AddMemberRequest) error {
	b, _ := json.Marshal(req)
	return c.do(ctx, "POST", "/customer/member", b, new(struct{}))
}

// UpdateMember updates a member's role in the current customer.
func (c *Client) UpdateMember(ctx context.Context, req UpdateMemberRequest) error {
	b, _ := json.Marshal(req)
	return c.do(ctx, "PUT", "/customer/member", b, new(struct{}))
}

// RemoveMember removes a member from the current customer by user ID.
func (c *Client) RemoveMember(ctx context.Context, userID string) error {
	return c.do(ctx, "DELETE", "/customer/member/"+url.PathEscape(userID), nil, new(struct{}))
}

// — Limits —

// GetAllLimits returns all limits for the customer.
func (c *Client) GetAllLimits(ctx context.Context) (*GetAllLimitsResponse, error) {
	var out GetAllLimitsResponse
	return &out, c.do(ctx, "GET", "/limits", nil, &out)
}

// GetServiceLimits returns limits for a specific service.
func (c *Client) GetServiceLimits(ctx context.Context, serviceName string) (*GetServiceLimitsResponse, error) {
	var out GetServiceLimitsResponse
	return &out, c.do(ctx, "GET", "/limits/"+url.PathEscape(serviceName), nil, &out)
}

// GetSpecificLimit returns a specific limit for a service and feature.
func (c *Client) GetSpecificLimit(ctx context.Context, serviceName, featureKey string) (*GetSpecificLimitResponse, error) {
	var out GetSpecificLimitResponse
	return &out, c.do(ctx, "GET", "/limits/"+url.PathEscape(serviceName)+"/"+url.PathEscape(featureKey), nil, &out)
}

// GetUsage returns usage data for the current user.
func (c *Client) GetUsage(ctx context.Context) (*GetUserUsageResponse, error) {
	var out GetUserUsageResponse
	return &out, c.do(ctx, "GET", "/limits/usage", nil, &out)
}

// GetUserUsage returns granular usage data for a specific user.
func (c *Client) GetUserUsage(ctx context.Context, userID string) (*GranularUsageResponse, error) {
	var out GranularUsageResponse
	return &out, c.do(ctx, "GET", "/limits/usage/users?userId="+url.QueryEscape(userID), nil, &out)
}

// — IP Limits —

// GetAllIPLimits returns all IP rate limits (no auth required).
func (c *Client) GetAllIPLimits(ctx context.Context) (*GetAllIPLimitsResponse, error) {
	var out GetAllIPLimitsResponse
	return &out, c.do(ctx, "GET", "/limits/ip", nil, &out)
}

// GetServiceIPLimits returns IP rate limits for a specific service (no auth required).
func (c *Client) GetServiceIPLimits(ctx context.Context, serviceName string) (*GetServiceIPLimitsResponse, error) {
	var out GetServiceIPLimitsResponse
	return &out, c.do(ctx, "GET", "/limits/ip/"+url.PathEscape(serviceName), nil, &out)
}

// GetIPUsage returns IP usage data (no auth required).
func (c *Client) GetIPUsage(ctx context.Context) (*GetIPUsageResponse, error) {
	var out GetIPUsageResponse
	return &out, c.do(ctx, "GET", "/limits/usage/ip", nil, &out)
}

// — Quotas —

// GetAllQuotas returns all quotas for the customer.
func (c *Client) GetAllQuotas(ctx context.Context) (*GetAllQuotasResponse, error) {
	var out GetAllQuotasResponse
	return &out, c.do(ctx, "GET", "/quotas", nil, &out)
}

// GetServiceQuotas returns quotas for a specific service.
func (c *Client) GetServiceQuotas(ctx context.Context, serviceName string) (*GetServiceQuotasResponse, error) {
	var out GetServiceQuotasResponse
	return &out, c.do(ctx, "GET", "/quotas/"+url.PathEscape(serviceName), nil, &out)
}

// GetSpecificQuota returns a specific quota for a service and feature.
func (c *Client) GetSpecificQuota(ctx context.Context, serviceName, featureKey string) (*GetSpecificQuotaResponse, error) {
	var out GetSpecificQuotaResponse
	return &out, c.do(ctx, "GET", "/quotas/"+url.PathEscape(serviceName)+"/"+url.PathEscape(featureKey), nil, &out)
}

// GetQuotaUsage returns quota usage data for the current user.
func (c *Client) GetQuotaUsage(ctx context.Context) (*GetQuotaUsageResponse, error) {
	var out GetQuotaUsageResponse
	return &out, c.do(ctx, "GET", "/quotas/usage", nil, &out)
}

// GetUserQuotaUsage returns granular quota usage for a specific user.
func (c *Client) GetUserQuotaUsage(ctx context.Context, userID string) (*GranularUsageResponse, error) {
	var out GranularUsageResponse
	return &out, c.do(ctx, "GET", "/quotas/usage/users?userId="+url.QueryEscape(userID), nil, &out)
}

// — Stripe —

// StartPortal starts a Stripe billing portal session.
func (c *Client) StartPortal(ctx context.Context) (*StartPortalResponse, error) {
	var out StartPortalResponse
	return &out, c.do(ctx, "GET", "/stripe/customer", nil, &out)
}

// UpgradeToPro creates a Stripe checkout session for upgrading to Pro.
func (c *Client) UpgradeToPro(ctx context.Context, req UpgradeToProRequest) (*UpgradeToProResponse, error) {
	b, _ := json.Marshal(req)
	var out UpgradeToProResponse
	return &out, c.do(ctx, "POST", "/stripe/upgrade", b, &out)
}

// — Admin —

// AdminBillingStatus returns the billing status for a customer as admin.
func (c *Client) AdminBillingStatus(ctx context.Context) (*GetBillingStatusResponse, error) {
	var out GetBillingStatusResponse
	return &out, c.do(ctx, "GET", "/admin/billing/status", nil, &out)
}

// AdminListCustomers lists customers with optional filtering.
func (c *Client) AdminListCustomers(ctx context.Context, opts *AdminListCustomersOptions) (*ListCustomersResponse, error) {
	q := make(url.Values)
	if opts != nil {
		if opts.Limit > 0 {
			q.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Page > 0 {
			q.Set("page", fmt.Sprintf("%d", opts.Page))
		}
		if opts.TierName != "" {
			q.Set("tierName", opts.TierName)
		}
		if opts.Status != "" {
			q.Set("status", opts.Status)
		}
	}
	path := "/admin/customers"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out ListCustomersResponse
	return &out, c.do(ctx, "GET", path, nil, &out)
}

// AdminGetCustomer fetches a customer by ID as admin.
func (c *Client) AdminGetCustomer(ctx context.Context, customerID string) (*AdminGetCustomerResponse, error) {
	var out AdminGetCustomerResponse
	return &out, c.do(ctx, "GET", "/admin/customer/"+url.PathEscape(customerID), nil, &out)
}

// AdminDeleteCustomer deletes a customer by ID as admin.
func (c *Client) AdminDeleteCustomer(ctx context.Context, customerID string) error {
	return c.do(ctx, "DELETE", "/admin/customer/"+url.PathEscape(customerID), nil, new(struct{}))
}

// AdminListCustomerMembers lists members of a customer as admin.
func (c *Client) AdminListCustomerMembers(ctx context.Context, customerID string, opts *ListOpts) (*AdminListCustomerMembersResponse, error) {
	q := make(url.Values)
	if opts != nil {
		if opts.Limit > 0 {
			q.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Page > 0 {
			q.Set("page", fmt.Sprintf("%d", opts.Page))
		}
	}
	path := "/admin/customer/" + url.PathEscape(customerID) + "/members"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out AdminListCustomerMembersResponse
	return &out, c.do(ctx, "GET", path, nil, &out)
}

// AdminAddMember adds a member to a customer as admin.
func (c *Client) AdminAddMember(ctx context.Context, customerID string, req AdminAddCustomerMemberRequest) error {
	b, _ := json.Marshal(req)
	return c.do(ctx, "POST", "/admin/customer/"+url.PathEscape(customerID)+"/member", b, new(struct{}))
}

// AdminRemoveMember removes a member from a customer as admin.
func (c *Client) AdminRemoveMember(ctx context.Context, customerID, userID string) error {
	return c.do(ctx, "DELETE", "/admin/customer/"+url.PathEscape(customerID)+"/member/"+url.PathEscape(userID), nil, new(struct{}))
}

// AdminListTiers lists all billing tiers with optional filtering.
func (c *Client) AdminListTiers(ctx context.Context, opts *AdminListTiersOptions) (*GetAllTiersResponse, error) {
	q := make(url.Values)
	if opts != nil && opts.Active != nil {
		q.Set("active", fmt.Sprintf("%t", *opts.Active))
	}
	path := "/admin/tiers"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out GetAllTiersResponse
	return &out, c.do(ctx, "GET", path, nil, &out)
}

// AdminSetDefaultTier sets a tier as the default tier.
func (c *Client) AdminSetDefaultTier(ctx context.Context, tierID string) error {
	return c.do(ctx, "PUT", "/admin/tiers/"+url.PathEscape(tierID)+"/default", nil, new(struct{}))
}
