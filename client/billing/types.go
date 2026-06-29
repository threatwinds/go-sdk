package billing

import "time"

// — Filtered Options —

// AdminListCustomersOptions provides filtering for the admin customer list endpoint.
type AdminListCustomersOptions struct {
	Limit    int    `url:"limit,omitempty"`
	Page     int    `url:"page,omitempty"`
	TierName string `url:"tierName,omitempty"`
	Status   string `url:"status,omitempty"`
}

// AdminListTiersOptions provides filtering for the admin tier list endpoint.
type AdminListTiersOptions struct {
	Active *bool `url:"active,omitempty"`
}

// ListOpts holds pagination parameters for admin list endpoints.
type ListOpts struct {
	Limit int
	Page  int
}

// — Customer —

// CreateCustomerRequest is the body for creating a new customer.
type CreateCustomerRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	BillingAddress string `json:"billingAddress"`
}

// GetCustomerResponse is returned when fetching the current customer.
type GetCustomerResponse struct {
	ID   string `json:"id"`
	GCID string `json:"gcid"`
}

// GetCustomerTierResponse is returned when fetching the customer's tier.
type GetCustomerTierResponse struct {
	TierID             string `json:"tierId"`
	TierName           string `json:"tierName"`
	SubscriptionStatus string `json:"subscriptionStatus"`
	Stacks             int    `json:"stacks"`
}

// TransferOwnershipRequest is the body for transferring customer ownership.
type TransferOwnershipRequest struct {
	NewOwnerUserID string `json:"newOwnerUserID"`
}

// — Members —

// AddMemberRequest is the body for adding a member to a customer.
type AddMemberRequest struct {
	UserID string `json:"userID"`
	Role   string `json:"role"`
}

// UpdateMemberRequest is the body for updating a member's role.
type UpdateMemberRequest struct {
	UserID string `json:"userID"`
	Role   string `json:"role"`
}

// Member represents a member of a customer.
type Member struct {
	UserID   string    `json:"userID"`
	Role     string    `json:"role"`
	Email    string    `json:"email"`
	JoinedAt time.Time `json:"joinedAt"`
}

// — Limits —

// LimitDetail contains the details of a single limit.
type LimitDetail struct {
	Value       any    `json:"value"`
	Window      string `json:"window"`
	Description string `json:"description"`
}

// GetAllLimitsResponse is returned when fetching all limits for the customer.
type GetAllLimitsResponse struct {
	CustomerID string         `json:"customerId"`
	TierName   string         `json:"tierName"`
	Limits     map[string]any `json:"limits"`
}

// GetServiceLimitsResponse is returned when fetching limits for a specific service.
type GetServiceLimitsResponse struct {
	Limits map[string]any `json:"limits"`
}

// GetSpecificLimitResponse is returned when fetching a specific limit.
type GetSpecificLimitResponse struct {
	Value       any    `json:"value"`
	Window      string `json:"window"`
	Description string `json:"description"`
}

// GetUserUsageResponse is returned when fetching usage for the current user.
type GetUserUsageResponse struct {
	Services map[string]any `json:"services"`
}

// GranularUsageResponse is returned when fetching granular usage for a user.
type GranularUsageResponse struct {
	Events []any `json:"events"`
	Total  int   `json:"total"`
}

// — IP Limits —

// GetAllIPLimitsResponse is returned when fetching all IP limits.
type GetAllIPLimitsResponse struct {
	IPLimits map[string]any `json:"ipLimits"`
}

// GetServiceIPLimitsResponse is returned when fetching IP limits for a specific service.
type GetServiceIPLimitsResponse struct {
	Limits map[string]any `json:"limits"`
}

// GetIPUsageResponse is returned when fetching IP usage.
type GetIPUsageResponse struct {
	Usage map[string]any `json:"usage"`
}

// — Quotas —

// GetAllQuotasResponse is returned when fetching all quotas for the customer.
type GetAllQuotasResponse struct {
	Quotas map[string]any `json:"quotas"`
}

// GetServiceQuotasResponse is returned when fetching quotas for a specific service.
type GetServiceQuotasResponse struct {
	Quotas map[string]any `json:"quotas"`
}

// GetSpecificQuotaResponse is returned when fetching a specific quota.
type GetSpecificQuotaResponse struct {
	Value any    `json:"value"`
	Unit  string `json:"unit"`
}

// GetQuotaUsageResponse is returned when fetching quota usage.
type GetQuotaUsageResponse struct {
	Usage map[string]any `json:"usage"`
}

// — Stripe —

// UpgradeToProRequest is the body for upgrading to Pro via Stripe.
type UpgradeToProRequest struct {
	SuccessURL string `json:"successURL"`
	CancelURL  string `json:"cancelURL"`
}

// StartPortalResponse is returned when starting a Stripe billing portal session.
type StartPortalResponse struct {
	URL string `json:"url"`
}

// UpgradeToProResponse is returned when creating a Pro upgrade checkout.
type UpgradeToProResponse struct {
	CheckoutURL string `json:"checkoutURL"`
}

// — Admin —

// GetBillingStatusResponse is returned when fetching billing status as admin.
type GetBillingStatusResponse struct {
	CustomerID         string `json:"customerId"`
	TierName           string `json:"tierName"`
	SubscriptionStatus string `json:"subscriptionStatus"`
	StripeCustomerID   string `json:"stripeCustomerID"`
	Stacks             int    `json:"stacks"`
}

// AdminGetCustomerResponse is returned when fetching a customer as admin.
type AdminGetCustomerResponse struct {
	ID                 string    `json:"id"`
	GCID               string    `json:"gcid"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	BillingAddress     string    `json:"billingAddress"`
	TierName           string    `json:"tierName"`
	SubscriptionStatus string    `json:"subscriptionStatus"`
	CreatedAt          time.Time `json:"createdAt"`
}

// AdminCustomer is a customer in a paginated admin list.
type AdminCustomer struct {
	ID                 string `json:"id"`
	GCID               string `json:"gcid"`
	Name               string `json:"name"`
	Email              string `json:"email"`
	TierName           string `json:"tierName"`
	SubscriptionStatus string `json:"subscriptionStatus"`
	CreatedAt          string `json:"createdAt"`
}

// AdminAddCustomerMemberRequest is the body for admin adding a member to a customer.
type AdminAddCustomerMemberRequest struct {
	UserID string `json:"userID"`
	Role   string `json:"role"`
}

// Tier represents a billing tier.
type Tier struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int    `json:"price"`
	IsActive    bool   `json:"isActive"`
}

// — Paginated Responses —

// ListMembersResponse is the paginated response for listing members.
type ListMembersResponse struct {
	Pages   int      `json:"pages"`
	Items   int      `json:"items"`
	Members []Member `json:"members"`
}

// ListCustomersResponse is the paginated response for admin customer listing.
type ListCustomersResponse struct {
	Pages     int             `json:"pages"`
	Items     int             `json:"items"`
	Customers []AdminCustomer `json:"customers"`
}

// AdminListCustomerMembersResponse is the paginated response for admin member listing.
type AdminListCustomerMembersResponse struct {
	Pages   int      `json:"pages"`
	Items   int      `json:"items"`
	Members []Member `json:"members"`
}

// GetAllTiersResponse is returned when fetching all tiers.
type GetAllTiersResponse struct {
	Tiers []Tier `json:"tiers"`
}
