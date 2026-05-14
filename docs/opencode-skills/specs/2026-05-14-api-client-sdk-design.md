# ThreatWinds Go SDK — API Client Design

**Date:** 2026-05-14
**Status:** Approved
**Scope:** Foundation + Core Services (Auth, Billing, Compute)

---

## Overview

A typed Go API client SDK for the ThreatWinds platform. Provides a single entry point (`client.New()`) with accessors for Auth, Billing, and Compute service clients. All services share a single HTTP client instance with unified authentication, retry, and error handling.

**Phase 1 services:** Auth, Billing, Compute
**Future phases:** Search, Ingest, Analytics, Feeds, AI, Tunnel, Gateway

---

## Prerequisites

Before SDK implementation, the following server-side API migration must complete:

All Auth/Billing/Compute admin paginated endpoints must be migrated from the current `page`/`pageSize` model to the unified Search API pagination model:

- **Query params:** `limit` / `page` (default 10/1, consistent with Search API)
- **Response fields:** `pages` (total page count), `items` (total matching items) inlined alongside the resource array
- **Response envelope example:**
  ```json
  {
    "pages": 5,
    "items": 47,
    "users": [ ... ]
  }
  ```

Non-paginated endpoints (user-facing lists, single-resource lookups) remain unchanged.

---

## Package Structure

 ```
client/
  client.go          # Client struct, New(), service accessor methods, Do()
  options.go         # Functional options (WithEndpoint, WithAPIKey, etc.), ListOptions
  error.go           # APIError typed error
  auth/
    client.go        # AuthClient — all /api/auth/v2 endpoints
    types.go         # Request/response structs, filtered options
  billing/
    client.go        # BillingClient — all /api/billing/v1 endpoints
    types.go         # Request/response structs, filtered options
  compute/
    client.go        # ComputeClient — all /api/compute/v1 endpoints
    types.go         # Request/response structs, filtered options
```

---

## Root Client

### Constructor

```go
package client

type Client struct { /* internal */ }

func New(opts ...Option) (*Client, error)
```

### Service Accessors

```go
func (c *Client) Auth() *auth.Client
func (c *Client) Billing() *billing.Client
func (c *Client) Compute() *compute.Client
```

### Options

```go
// Endpoint configuration
WithEndpoint(url string)              // Default: https://api.threatwinds.com

// Authentication — mutually exclusive, error at New() if both set
WithAPIKey(keyID, secret string)      // Sets api-key + api-secret headers
WithBearer(token string)              // Sets Authorization: Bearer header

// HTTP transport
WithHTTPClient(hc *http.Client)       // Full custom HTTP client
WithTimeout(d time.Duration)          // Default: 30s

// Retry
WithMaxRetries(n int)                 // GET-only retry count, Default: 3
```

### Usage

```go
sdk, err := client.New(
    client.WithEndpoint("https://api.threatwinds.com"),
    client.WithAPIKey(keyID, secret),
)
if err != nil {
    // handle
}

// Session
session, err := sdk.Auth().GetSession(ctx)

// Compute
instances, err := sdk.Compute().ListInstances(ctx)

// Billing with pagination
members, err := sdk.Billing().ListMembers(ctx, &client.ListOptions{
    Limit: 50,
    Page:  1,
})
```

---

## Authentication

Two mutually exclusive modes. The SDK applies auth headers automatically on every request.

### API Key Pair

```
api-key:     <keyID>
api-secret:  <secret>
```

### Bearer Token

```
Authorization: Bearer <token>
```

Note: The server already implements sliding session expiry. Every authenticated request through the gateway auto-extends the session TTL. No client-side token refresh is needed.

---

## HTTP Client and Request Flow

The SDK includes its own internal HTTP client with connection pooling, retry logic, and typed errors. No dependencies on existing `utils/` or `catcher/` packages.

### Flow

```
1. Build URL: endpoint + service base path + resource path + query params
2. Marshal request body to JSON (if present)
3. Apply auth headers
4. Set Content-Type: application/json, User-Agent
5. Execute via persistent shared *http.Client (connection pooled)
6. 2xx → unmarshal into typed response, return
7. Retryable → honor Retry-After header, fallback to exponential backoff
8. 4xx/5xx → parse x-error / x-error-id headers, return *APIError
```

### Retry Policy

- **Retryable methods:** GET only
- **Retryable statuses:** 429, 502, 503, 504
- **Retry-After header:** If present, parse and honor (HTTP date or delay-seconds format). If absent, use exponential backoff.
- **Backoff:** 100ms base, 4x multiplier (100ms → 400ms → 1600ms)
- **POST/PUT/DELETE:** Never retried (non-idempotent safety)

### Transport

Persistent `*http.Client` with Go's default connection pooling. TLS 1.2+ enforced. Callers can inject a custom `*http.Client` via `WithHTTPClient()` for custom transport, proxying, or logging.

---

## Typed Errors

```go
type APIError struct {
    StatusCode int    `json:"status_code"`
    Message    string `json:"message"`      // from x-error response header
    ErrorID    string `json:"error_id"`     // from x-error-id response header
    Body       []byte `json:"-"`            // raw response body
}

func (e *APIError) Error() string

func (e *APIError) IsNotFound() bool        // 404
func (e *APIError) IsUnauthorized() bool    // 401
func (e *APIError) IsForbidden() bool       // 403
func (e *APIError) IsRateLimited() bool     // 429
func (e *APIError) IsValidationError() bool // 400
```

The raw response body is always captured and attached to the error, enabling callers to inspect server-side error details.

---

## Pagination

All paginated endpoints use the unified model (post-migration). Each paginated method returns a per-endpoint typed response struct with inline `Pages` and `Items` fields.

### ListOptions (basic pagination)

```go
type ListOptions struct {
    Limit int // 0 = server default (10)
    Page  int // 0 = server default (1)
}
```

Used for simple paginated endpoints that only need `limit` and `page`.

### FilteredOptions (pagination + filters)

Endpoints with additional filter query parameters use service-specific options structs:

```go
// Auth admin user filtering
type AdminListUsersOptions struct {
    Limit   int    `url:"limit,omitempty"`
    Page    int    `url:"page,omitempty"`
    Enabled *bool  `url:"enabled,omitempty"`
    Role    string `url:"role,omitempty"`
}

// Billing admin customer filtering
type AdminListCustomersOptions struct {
    Limit    int    `url:"limit,omitempty"`
    Page     int    `url:"page,omitempty"`
    TierName string `url:"tierName,omitempty"`
    Status   string `url:"status,omitempty"`
}

// Compute admin instance filtering
type AdminListInstancesOptions struct {
    Limit      int    `url:"limit,omitempty"`
    Page       int    `url:"page,omitempty"`
    UserID     string `url:"userID,omitempty"`
    CustomerID string `url:"customerID,omitempty"`
    Status     string `url:"status,omitempty"`
    Zone       string `url:"zone,omitempty"`
    TemplateID string `url:"templateID,omitempty"`
}

// Billing admin tier filtering
type AdminListTiersOptions struct {
    Active *bool `url:"active,omitempty"`
}
```

**Paginated response pattern:**

```go
type ListUsersResponse struct {
    Pages int    `json:"pages"`
    Items int    `json:"items"`
    Users []User `json:"users"`
}

type ListMembersResponse struct {
    Pages   int      `json:"pages"`
    Items   int      `json:"items"`
    Members []Member `json:"members"`
}

type AdminListInstancesResponse struct {
    Pages     int        `json:"pages"`
    Items     int        `json:"items"`
    Instances []Instance `json:"instances"`
}
```

**Non-paginated endpoints** return raw slices or single objects:

```go
func (c *AuthClient) ListEmails(ctx context.Context) ([]Email, error)
func (c *ComputeClient) ListInstances(ctx context.Context) ([]Instance, error)
func (c *AuthClient) GetSession(ctx context.Context) (*Session, error)
```

No generic wrapper. Each endpoint has its own response type with domain-specific field names.

---

## Service Clients — Endpoint Coverage

### Auth Client (`/api/auth/v2`)

**Session**
- `CreateSession(ctx, req CreateSessionRequest) (*CreateSessionResponse, error)`
- `GetSession(ctx) (*GetSessionResponse, error)`
- `VerifySession(ctx, req VerifySessionRequest) error`
- `ExtendSession(ctx) error`
- `DeleteSession(ctx, sessionID string) error`
- `ListSessions(ctx) ([]ActiveSession, error)`

**Email**
- `CreateEmail(ctx, req CreateEmailRequest) (*CreateEmailResponse, error)`
- `VerifyEmail(ctx, req VerifyEmailRequest) error`
- `ListEmails(ctx) ([]Email, error)`
- `SetPreferredEmail(ctx, req SetPreferredEmailRequest) error`
- `DeleteEmail(ctx, emailID string) error`

**KeyPair**
- `CreateKeyPair(ctx, req CreateKeyPairRequest) (*CreateKeyPairResponse, error)`
- `VerifyKeyPair(ctx, req VerifyKeyPairRequest) error`
- `CheckKeyPair(ctx) (*CheckKeyPairResponse, error)`
- `ListKeyPairs(ctx) ([]KeyPair, error)`
- `DeleteKeyPair(ctx, keyID string) error`

**User**
- `CreateUser(ctx, req CreateUserRequest) (*CreateSessionResponse, error)`
- `DeleteUser(ctx) error`
- `GetUserByID(ctx, userID string) (*GetUserResponse, error)`
- `GetUserByEmail(ctx, email string) (*GetUserByEmailResponse, error)`

**Identity Verification**
- `CreateVerification(ctx) (*CreateVerificationResponse, error)`
- `GetVerificationStatus(ctx) (*VerificationStatusResponse, error)`

**Partner**
- `PartnerCreateUser(ctx, req AdminCreateUserRequest) (*AdminCreateUserResponse, error)`

**Admin** (requires admin role)
- `AdminListUsers(ctx, opts *AdminListUsersOptions) (*ListUsersResponse, error)`
- `AdminGetUser(ctx, userID string) (*AdminUserDetailResponse, error)`
- `AdminDeleteUser(ctx, userID string) error`
- `AdminDisableUser(ctx, userID string) error`
- `AdminEnableUser(ctx, userID string) error`
- `AdminVerifyUser(ctx, userID string) error`
- `AdminUnverifyUser(ctx, userID string) error`
- `AdminAssignRole(ctx, userID string, req AssignRoleRequest) (*AssignRoleResponse, error)`
- `AdminRemoveRole(ctx, userID, roleName string) error`
- `AdminListRoles(ctx, userID string) (*ListRolesResponse, error)`
- `AdminCreateSession(ctx, userID string, req AdminCreateSessionRequest) (*AdminCreateSessionResponse, error)`
- `AdminListSessions(ctx, opts *ListOptions) (*AdminListSessionsResponse, error)`
- `AdminVerifySession(ctx, sessionID string) error`
- `AdminDeleteSession(ctx, sessionID string) error`
- `AdminListKeyPairs(ctx, opts *ListOptions) (*AdminListKeyPairsResponse, error)`
- `AdminCreateKeyPair(ctx, userID string) error`
- `AdminVerifyKeyPair(ctx, keyID string) error`
- `AdminDeleteKeyPair(ctx, keyID string) error`
- `AdminCreateEmail(ctx, userID string) error`
- `AdminVerifyEmail(ctx, emailID string) error`
- `AdminSetPreferredEmail(ctx, req AdminSetPreferredEmailRequest) error`
- `AdminDeleteEmail(ctx, emailID string) error`
- `AdminGetVerification(ctx, userID string) (*AdminVerificationResponse, error)`
- `AdminResetVerification(ctx, userID string) (*Verification, error)`
- `AdminRevokeVerification(ctx, userID string) (*Verification, error)`

### Billing Client (`/api/billing/v1`)

**Customer**
- `CreateCustomer(ctx, req CreateCustomerRequest) error`
- `GetCustomer(ctx) (*GetCustomerResponse, error)`
- `DeleteCustomer(ctx) error`
- `GetCustomerTier(ctx) (*GetCustomerTierResponse, error)`
- `LeaveCustomer(ctx) error`
- `TransferOwnership(ctx, req TransferOwnershipRequest) error`

**Members**
- `ListMembers(ctx, opts *ListOptions) (*ListMembersResponse, error)`
- `AddMember(ctx, req AddMemberRequest) error`
- `UpdateMember(ctx, req UpdateMemberRequest) error`
- `RemoveMember(ctx, userID string) error`

**Limits**
- `GetAllLimits(ctx) (*GetAllLimitsResponse, error)`
- `GetServiceLimits(ctx, serviceName string) (*GetServiceLimitsResponse, error)`
- `GetSpecificLimit(ctx, serviceName, featureKey string) (*GetSpecificLimitResponse, error)`
- `GetUsage(ctx) (*GetUserUsageResponse, error)`
- `GetUserUsage(ctx, userID string) (*GranularUsageResponse, error)`

**IP Limits** (no auth required)
- `GetAllIPLimits(ctx) (*GetAllIPLimitsResponse, error)`
- `GetServiceIPLimits(ctx, serviceName string) (*GetServiceIPLimitsResponse, error)`
- `GetIPUsage(ctx) (*GetIPUsageResponse, error)`

**Quotas**
- `GetAllQuotas(ctx) (*GetAllQuotasResponse, error)`
- `GetServiceQuotas(ctx, serviceName string) (*GetServiceQuotasResponse, error)`
- `GetSpecificQuota(ctx, serviceName, featureKey string) (*GetSpecificQuotaResponse, error)`
- `GetQuotaUsage(ctx) (*GetQuotaUsageResponse, error)`
- `GetUserQuotaUsage(ctx, userID string) (*GranularUsageResponse, error)`

**Stripe**
- `StartPortal(ctx) (*StartPortalResponse, error)`
- `UpgradeToPro(ctx, req UpgradeToProRequest) (*UpgradeToProResponse, error)`

**Admin**
- `AdminBillingStatus(ctx) (*GetBillingStatusResponse, error)`
- `AdminListCustomers(ctx, opts *AdminListCustomersOptions) (*ListCustomersResponse, error)`
- `AdminGetCustomer(ctx, customerID string) (*AdminGetCustomerResponse, error)`
- `AdminDeleteCustomer(ctx, customerID string) error`
- `AdminListCustomerMembers(ctx, customerID string, opts *ListOptions) (*AdminListCustomerMembersResponse, error)`
- `AdminAddMember(ctx, customerID string, req AdminAddCustomerMemberRequest) error`
- `AdminRemoveMember(ctx, customerID, userID string) error`
- `AdminListTiers(ctx, opts *AdminListTiersOptions) (*GetAllTiersResponse, error)`
- `AdminSetDefaultTier(ctx, tierID string) error`

### Compute Client (`/api/compute/v1`)

**Instance**
- `CreateInstance(ctx, req InstanceCreateRequest) (*Instance, error)`
- `ListInstances(ctx) ([]Instance, error)`
- `GetInstance(ctx, instanceID string) (*Instance, error)`
- `DeleteInstance(ctx, instanceID string) error`

**Power Management**
- `StartInstance(ctx, instanceID string) error`
- `StopInstance(ctx, instanceID string) error`
- `RestartInstance(ctx, instanceID string) error`
- `ResetInstance(ctx, instanceID string) error`

**Templates**
- `ListTemplates(ctx) ([]Template, error)`
- `ListTemplateZones(ctx, templateID string) ([]string, error)`

**Admin** (requires compute_admin role)
- `AdminListInstances(ctx, opts *AdminListInstancesOptions) (*AdminListInstancesResponse, error)`
- `AdminGetInstance(ctx, instanceID string) (*Instance, error)`
- `AdminDeleteInstance(ctx, instanceID string) error`
- `AdminStartInstance(ctx, instanceID string) error`
- `AdminStopInstance(ctx, instanceID string) error`
- `AdminRestartInstance(ctx, instanceID string) error`
- `AdminResetInstance(ctx, instanceID string) error`

---

## Testing Strategy

- **Unit tests** with mocked `http.RoundTripper` — test auth injection, `APIError` parsing (x-error/x-error-id, status code mapping), retry logic, retry-after handling, pagination options, request building
- **Table-driven tests** — parameterize scenarios: success, 401, 404, 429 with Retry-After, 429 without Retry-After, 503 retry, empty list, paginated list, auth mode conflict
- **No integration tests** — SDK tests against mocks; integration testing belongs in the API services
- **Coverage target:** 80%+ on client logic

---

## Out of Scope (Future Phases)

- Search, Ingest, Analytics, Feeds, AI, Tunnel, Gateway service clients
- Auto-paginating iterator (`EachPage()`)
- Credential file storage / profile management
- Session auto-refresh (server already handles sliding expiry)
- Request/response logging (callers use custom `http.RoundTripper`)
- Code generation from OpenAPI specs
- Streaming/SSE support (AI API phase)

---

## Dependencies

No new external dependencies. The SDK uses only the Go standard library:
- `net/http` — HTTP client and transport
- `encoding/json` — JSON serialization
- `context` — Request cancellation
- `net/url` — URL building
- `time` — Retry timing and Retry-After parsing

The client is entirely self-contained. The existing go-sdk dependencies (gin, cel-go, opensearch, catcher, etc.) are NOT needed by the client package.

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Single `New()` constructor | Simplest DX, shared HTTP client, matches CLI mental model |
| Mutual auth exclusion | Prevents ambiguous auth state; error fast at construction |
| GET-only retry | POST/PUT/DELETE may be non-idempotent; matches CLI behavior |
| Retry-After honored first | Server knows the correct backoff; client backoff is fallback |
| Self-contained HTTP client | No dependency on go-sdk internals; clean, standalone package |
| `APIError` typed error | Simple, focused error type with convenience checks; no external deps |
| Per-endpoint response structs | Honest representation of API; no leaky generic abstraction |
| Inline pagination fields | Matches Search API pattern; no nested object indirection |
| No built-in logging | Callers already have tools (custom transport, proxy); keep SDK focused |
| All fully typed | Type safety, IDE autocomplete, compile-time error detection |
