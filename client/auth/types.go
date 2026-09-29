package auth

// ListUsersOptions provides filtering for the admin user list endpoint.
type ListUsersOptions struct {
	Limit   int    `url:"limit,omitempty"`
	Page    int    `url:"page,omitempty"`
	Enabled *bool  `url:"enabled,omitempty"`
	Role    string `url:"role,omitempty"`
}

// ListOpts holds pagination parameters for admin list endpoints.
type ListOpts struct {
	Limit int
	Page  int
}

// — Session —

// CreateSessionRequest is the body for creating a new session.
type CreateSessionRequest struct {
	Email string `json:"email"`
	Kind  string `json:"kind"`
}

// VerifySessionRequest is the body for verifying a session.
type VerifySessionRequest struct {
	Code               string `json:"code"`
	VerificationCodeID string `json:"verificationCodeID"`
}

// CreateSessionResponse is returned when a session is created.
type CreateSessionResponse struct {
	Bearer             string `json:"bearer"`
	SessionID          string `json:"sessionID"`
	ExpireAt           string `json:"expireAt"`
	VerificationCodeID string `json:"verificationCodeID"`
}

// GetSessionResponse is returned when fetching the current session.
type GetSessionResponse struct {
	SessionID string   `json:"sessionID"`
	UserID    string   `json:"userID"`
	Roles     []string `json:"roles"`
	Groups    []string `json:"groups"`
	Verified  bool     `json:"verified"`
	ExpireAt  string   `json:"expireAt"`
}

// ActiveSession represents a session in a list.
type ActiveSession struct {
	SessionID string   `json:"sessionID"`
	UserID    string   `json:"userID"`
	Roles     []string `json:"roles"`
	Verified  bool     `json:"verified"`
	ExpireAt  string   `json:"expireAt"`
}

// — Email —

// CreateEmailRequest is the body for creating a new email address.
type CreateEmailRequest struct {
	Address string `json:"address"`
}

// VerifyEmailRequest is the body for verifying an email address.
type VerifyEmailRequest struct {
	Code               string `json:"code"`
	VerificationCodeID string `json:"verificationCodeID"`
}

// SetPreferredEmailRequest is the body for setting a preferred email.
type SetPreferredEmailRequest struct {
	EmailID string `json:"emailID"`
}

// CreateEmailResponse is returned when an email is created.
type CreateEmailResponse struct {
	VerificationCodeID string `json:"verificationCodeID"`
}

// Email represents a user's email address.
type Email struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Verified  bool   `json:"verified"`
	Preferred bool   `json:"preferred"`
}

// — KeyPair —

// CreateKeyPairRequest is the body for creating a new API key pair.
type CreateKeyPairRequest struct {
	Name string `json:"name"`
	Days int    `json:"days"`
}

// VerifyKeyPairRequest is the body for verifying a key pair.
type VerifyKeyPairRequest struct {
	Code               string `json:"code"`
	VerificationCodeID string `json:"verificationCodeID"`
}

// CreateKeyPairResponse is returned when a key pair is created.
type CreateKeyPairResponse struct {
	APIKey             string `json:"apiKey"`
	APISecret          string `json:"apiSecret"`
	KeyID              string `json:"keyID"`
	ExpireAt           string `json:"expireAt"`
	Verified           bool   `json:"verified"`
	VerificationCodeID string `json:"verificationCodeID"`
}

// CheckKeyPairResponse is returned when checking the current key pair.
type CheckKeyPairResponse struct {
	APIKey   string   `json:"apiKey"`
	KeyID    string   `json:"keyID"`
	Roles    []string `json:"roles"`
	Groups   []string `json:"groups"`
	Verified bool     `json:"verified"`
	ExpireAt string   `json:"expireAt"`
}

// KeyPair represents a user's API key pair.
type KeyPair struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Verified bool   `json:"verified"`
	ExpireAt string `json:"expireAt"`
}

// — User —

// CreateUserRequest is the body for creating a new user.
type CreateUserRequest struct {
	Email    string `json:"email"`
	FullName string `json:"fullName"`
	Alias    string `json:"alias"`
}

// GetUserResponse is returned when fetching a user by ID.
type GetUserResponse struct {
	Alias string `json:"alias"`
}

// GetUserByEmailResponse is returned when fetching a user by email.
type GetUserByEmailResponse struct {
	UserID string `json:"userID"`
}

// — Identity Verification —

// CreateVerificationResponse is returned when creating an identity verification.
type CreateVerificationResponse struct {
	URL          string `json:"url"`
	SessionID    string `json:"sessionId"`
	ClientSecret string `json:"clientSecret"`
	Status       string `json:"status"`
}

// VerificationStatusResponse is returned when checking verification status.
type VerificationStatusResponse struct {
	Status      string `json:"status"`
	ExpiresAt   string `json:"expiresAt"`
	Attempts    int    `json:"attempts"`
	MaxAttempts int    `json:"maxAttempts"`
}

// — Partner —

// AdminCreateUserRequest is the body for partner user creation.
type AdminCreateUserRequest struct {
	Email     string   `json:"email"`
	FullName  string   `json:"fullName"`
	Alias     string   `json:"alias"`
	Roles     []string `json:"roles"`
	PortalURL string   `json:"portalURL"`
	Notify    bool     `json:"notify"`
}

// AdminCreateUserResponse is returned when a partner creates a user.
type AdminCreateUserResponse struct {
	UserID string `json:"userID"`
}

// — Admin —

// AssignRoleRequest is the body for assigning a role to a user.
type AssignRoleRequest struct {
	Role   string `json:"role"`
	UserID string `json:"userID"`
}

// AssignRoleResponse is returned when a role is assigned.
type AssignRoleResponse struct {
	UserID string `json:"userID"`
	Role   string `json:"role"`
}

// ListRolesResponse is returned when listing roles for a user.
type ListRolesResponse struct {
	UserID string   `json:"userID"`
	Roles  []string `json:"roles"`
}

// AdminCreateSessionRequest is the body for admin session creation.
type AdminCreateSessionRequest struct {
	Kind string `json:"kind"`
}

// AdminCreateSessionResponse is returned when an admin creates a session.
type AdminCreateSessionResponse struct {
	Bearer    string `json:"bearer"`
	SessionID string `json:"sessionID"`
}

// AdminSetPreferredEmailRequest is the body for admin setting preferred email.
type AdminSetPreferredEmailRequest struct {
	EmailID string `json:"emailID"`
}

// AdminUserDetailResponse is returned when fetching a user detail as admin.
type AdminUserDetailResponse struct {
	ID       string   `json:"id"`
	Emails   []Email  `json:"emails"`
	Roles    []string `json:"roles"`
	Verified bool     `json:"verified"`
}

// AdminVerificationResponse is returned when fetching admin verification status.
type AdminVerificationResponse struct {
	Status    string `json:"status"`
	ExpiresAt string `json:"expiresAt"`
}

// Verification represents a verification record.
type Verification struct {
	Status    string `json:"status"`
	ExpiresAt string `json:"expiresAt"`
}

// — Paginated Responses —

// UserResponse is a user in a paginated list.
type UserResponse struct {
	ID       string   `json:"id"`
	Emails   []Email  `json:"emails"`
	Roles    []string `json:"roles"`
	Verified bool     `json:"verified"`
	Enabled  bool     `json:"enabled"`
}

// ListUsersResponse is the paginated response for listing users.
type ListUsersResponse struct {
	Pages int            `json:"pages"`
	Items int            `json:"items"`
	Users []UserResponse `json:"users"`
}

// AdminListSessionsResponse is the paginated response for admin session listing.
type AdminListSessionsResponse struct {
	Pages    int             `json:"pages"`
	Items    int             `json:"items"`
	Sessions []ActiveSession `json:"sessions"`
}

// AdminListKeyPairsResponse is the paginated response for admin key pair listing.
type AdminListKeyPairsResponse struct {
	Pages    int       `json:"pages"`
	Items    int       `json:"items"`
	KeyPairs []KeyPair `json:"keyPairs"`
}
