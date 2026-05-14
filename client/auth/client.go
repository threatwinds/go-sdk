package auth

import (
	"context"
	"encoding/json"
	"net/url"
)

const basePath = "/api/auth/v2"

// doFunc is the signature for making HTTP requests to the API.
type doFunc func(ctx context.Context, method, path string, body []byte, out any) error

// Client provides access to the Auth API endpoints.
type Client struct {
	do doFunc
}

// NewClient creates a new Auth client backed by the root SDK client's do function.
func NewClient(d doFunc) *Client {
	return &Client{do: d}
}

// — Session —

// CreateSession creates a new session.
func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (*CreateSessionResponse, error) {
	b, _ := json.Marshal(req)
	var out CreateSessionResponse
	return &out, c.do(ctx, "POST", "/session", b, &out)
}

// GetSession returns the current session.
func (c *Client) GetSession(ctx context.Context) (*GetSessionResponse, error) {
	var out GetSessionResponse
	return &out, c.do(ctx, "GET", "/session", nil, &out)
}

// VerifySession verifies a session with the provided code.
func (c *Client) VerifySession(ctx context.Context, req VerifySessionRequest) error {
	b, _ := json.Marshal(req)
	return c.do(ctx, "PUT", "/session/verification", b, new(struct{}))
}

// ExtendSession extends the current session TTL.
func (c *Client) ExtendSession(ctx context.Context) error {
	return c.do(ctx, "PUT", "/session/extend", nil, new(struct{}))
}

// DeleteSession deletes a session by ID.
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	return c.do(ctx, "DELETE", "/session/"+url.PathEscape(sessionID), nil, new(struct{}))
}

// ListSessions lists all active sessions for the current user.
func (c *Client) ListSessions(ctx context.Context) ([]ActiveSession, error) {
	var out []ActiveSession
	return out, c.do(ctx, "GET", "/sessions", nil, &out)
}

// — Email —

// CreateEmail creates a new email address for the user.
func (c *Client) CreateEmail(ctx context.Context, req CreateEmailRequest) (*CreateEmailResponse, error) {
	b, _ := json.Marshal(req)
	var out CreateEmailResponse
	return &out, c.do(ctx, "POST", "/email", b, &out)
}

// VerifyEmail verifies an email address with the provided code.
func (c *Client) VerifyEmail(ctx context.Context, req VerifyEmailRequest) error {
	b, _ := json.Marshal(req)
	return c.do(ctx, "PUT", "/email/verification", b, new(struct{}))
}

// ListEmails lists all email addresses for the user.
func (c *Client) ListEmails(ctx context.Context) ([]Email, error) {
	var out []Email
	return out, c.do(ctx, "GET", "/emails", nil, &out)
}

// SetPreferredEmail sets the preferred email address.
func (c *Client) SetPreferredEmail(ctx context.Context, req SetPreferredEmailRequest) error {
	b, _ := json.Marshal(req)
	return c.do(ctx, "PUT", "/email/preferred", b, new(struct{}))
}

// DeleteEmail deletes an email address by ID.
func (c *Client) DeleteEmail(ctx context.Context, emailID string) error {
	return c.do(ctx, "DELETE", "/email/"+url.PathEscape(emailID), nil, new(struct{}))
}

// — KeyPair —

// CreateKeyPair creates a new API key pair.
func (c *Client) CreateKeyPair(ctx context.Context, req CreateKeyPairRequest) (*CreateKeyPairResponse, error) {
	b, _ := json.Marshal(req)
	var out CreateKeyPairResponse
	return &out, c.do(ctx, "POST", "/keypair", b, &out)
}

// VerifyKeyPair verifies a key pair with the provided code.
func (c *Client) VerifyKeyPair(ctx context.Context, req VerifyKeyPairRequest) error {
	b, _ := json.Marshal(req)
	return c.do(ctx, "PUT", "/keypair/verification", b, new(struct{}))
}

// CheckKeyPair returns the current key pair details.
func (c *Client) CheckKeyPair(ctx context.Context) (*CheckKeyPairResponse, error) {
	var out CheckKeyPairResponse
	return &out, c.do(ctx, "GET", "/keypair", nil, &out)
}

// ListKeyPairs lists all key pairs for the user.
func (c *Client) ListKeyPairs(ctx context.Context) ([]KeyPair, error) {
	var out []KeyPair
	return out, c.do(ctx, "GET", "/keypairs", nil, &out)
}

// DeleteKeyPair deletes a key pair by ID.
func (c *Client) DeleteKeyPair(ctx context.Context, keyID string) error {
	return c.do(ctx, "DELETE", "/keypair/"+url.PathEscape(keyID), nil, new(struct{}))
}

// — User —

// CreateUser creates a new user account.
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (*CreateSessionResponse, error) {
	b, _ := json.Marshal(req)
	var out CreateSessionResponse
	return &out, c.do(ctx, "POST", "/user", b, &out)
}

// DeleteUser deletes the current user account.
func (c *Client) DeleteUser(ctx context.Context) error {
	return c.do(ctx, "DELETE", "/user", nil, new(struct{}))
}

// GetUserByID fetches a user by their ID.
func (c *Client) GetUserByID(ctx context.Context, userID string) (*GetUserResponse, error) {
	var out GetUserResponse
	return &out, c.do(ctx, "GET", "/user/"+url.PathEscape(userID), nil, &out)
}

// GetUserByEmail fetches a user by their email address.
func (c *Client) GetUserByEmail(ctx context.Context, email string) (*GetUserByEmailResponse, error) {
	var out GetUserByEmailResponse
	return &out, c.do(ctx, "GET", "/user/by-email?email="+url.QueryEscape(email), nil, &out)
}

// — Identity Verification —

// CreateVerification creates a new identity verification.
func (c *Client) CreateVerification(ctx context.Context) (*CreateVerificationResponse, error) {
	var out CreateVerificationResponse
	return &out, c.do(ctx, "POST", "/verify", nil, &out)
}

// GetVerificationStatus returns the current identity verification status.
func (c *Client) GetVerificationStatus(ctx context.Context) (*VerificationStatusResponse, error) {
	var out VerificationStatusResponse
	return &out, c.do(ctx, "GET", "/verify/status", nil, &out)
}

// — Partner —

// PartnerCreateUser creates a new user on behalf of a partner.
func (c *Client) PartnerCreateUser(ctx context.Context, req AdminCreateUserRequest) (*AdminCreateUserResponse, error) {
	b, _ := json.Marshal(req)
	var out AdminCreateUserResponse
	return &out, c.do(ctx, "POST", "/partners/user", b, &out)
}
