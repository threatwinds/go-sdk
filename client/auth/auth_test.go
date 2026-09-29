package auth_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/threatwinds/go-sdk/client"
	"github.com/threatwinds/go-sdk/client/auth"
)

// ---------------------------------------------------------------------------
// mockRT implements http.RoundTripper for unit tests.
// ---------------------------------------------------------------------------

type mockRT struct {
	roundTripper func(req *http.Request) (*http.Response, error)
}

func (m *mockRT) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.roundTripper != nil {
		return m.roundTripper(req)
	}
	return nil, nil
}

func mockResp(status int, headers http.Header, body string) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// setup creates a root client with a mock transport and returns the auth client.
func setup(t *testing.T) (*client.Client, *mockRT) {
	t.Helper()
	m := &mockRT{}
	c, err := client.New(
		client.WithAPIKey("k", "s"),
		client.WithHTTPClient(&http.Client{Transport: m}),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c, m
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateSession(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" {
			t.Errorf("method = %q, want POST", req.Method)
		}
		if req.URL.Path != "/session" {
			t.Errorf("path = %q, want /session", req.URL.Path)
		}

		var body auth.CreateSessionRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body.Email != "alice@example.com" {
			t.Errorf("email = %q, want alice@example.com", body.Email)
		}
		if body.Kind != "web" {
			t.Errorf("kind = %q, want web", body.Kind)
		}

		return mockResp(200, nil, `{
			"bearer":"tok-123",
			"sessionID":"sess-abc",
			"expireAt":"2026-06-01T00:00:00Z",
			"verificationCodeID":"vc-99"
		}`), nil
	}

	resp, err := c.Auth().CreateSession(context.Background(), auth.CreateSessionRequest{
		Email: "alice@example.com",
		Kind:  "web",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Bearer != "tok-123" {
		t.Errorf("Bearer = %q, want tok-123", resp.Bearer)
	}
	if resp.SessionID != "sess-abc" {
		t.Errorf("SessionID = %q, want sess-abc", resp.SessionID)
	}
	if resp.ExpireAt != "2026-06-01T00:00:00Z" {
		t.Errorf("ExpireAt = %q, want 2026-06-01T00:00:00Z", resp.ExpireAt)
	}
	if resp.VerificationCodeID != "vc-99" {
		t.Errorf("VerificationCodeID = %q, want vc-99", resp.VerificationCodeID)
	}
}

func TestGetSession(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/session" {
			t.Errorf("path = %q, want /session", req.URL.Path)
		}
		// Verify API key auth headers are set.
		apiKey := req.Header.Get("Api-Key")
		apiSecret := req.Header.Get("Api-Secret")
		if apiKey != "k" {
			t.Errorf("Api-Key = %q, want k", apiKey)
		}
		if apiSecret != "s" {
			t.Errorf("Api-Secret = %q, want s", apiSecret)
		}

		return mockResp(200, nil, `{
			"sessionID":"sess-abc",
			"userID":"user-1",
			"roles":["admin","editor"],
			"groups":["eng"],
			"verified":true,
			"expireAt":"2026-06-01T00:00:00Z"
		}`), nil
	}

	resp, err := c.Auth().GetSession(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SessionID != "sess-abc" {
		t.Errorf("SessionID = %q, want sess-abc", resp.SessionID)
	}
	if resp.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", resp.UserID)
	}
	if len(resp.Roles) != 2 || resp.Roles[0] != "admin" {
		t.Errorf("Roles = %v, want [admin editor]", resp.Roles)
	}
	if !resp.Verified {
		t.Error("Verified = false, want true")
	}
}

func TestListEmails(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/emails" {
			t.Errorf("path = %q, want /emails", req.URL.Path)
		}

		return mockResp(200, nil, `[
			{"id":"e1","address":"a@example.com","verified":true,"preferred":true},
			{"id":"e2","address":"b@example.com","verified":false,"preferred":false}
		]`), nil
	}

	emails, err := c.Auth().ListEmails(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(emails) != 2 {
		t.Fatalf("len(emails) = %d, want 2", len(emails))
	}
	if emails[0].Address != "a@example.com" {
		t.Errorf("emails[0].Address = %q, want a@example.com", emails[0].Address)
	}
	if !emails[0].Preferred {
		t.Error("emails[0].Preferred = false, want true")
	}
	if emails[1].Verified {
		t.Error("emails[1].Verified = true, want false")
	}
}

func TestAdminListUsers(t *testing.T) {
	c, m := setup(t)

	enabled := true

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/admin/users" {
			t.Errorf("path = %q, want /admin/users", req.URL.Path)
		}

		q := req.URL.Query()
		if q.Get("limit") != "10" {
			t.Errorf("limit = %q, want 10", q.Get("limit"))
		}
		if q.Get("page") != "2" {
			t.Errorf("page = %q, want 2", q.Get("page"))
		}
		if q.Get("enabled") != "true" {
			t.Errorf("enabled = %q, want true", q.Get("enabled"))
		}
		if q.Get("role") != "admin" {
			t.Errorf("role = %q, want admin", q.Get("role"))
		}

		return mockResp(200, nil, `{
			"pages":5,
			"items":47,
			"users":[
				{"id":"u1","emails":[{"id":"e1","address":"a@x.com","verified":true,"preferred":true}],"roles":["admin"],"verified":true,"enabled":true}
			]
		}`), nil
	}

	resp, err := c.Auth().AdminListUsers(context.Background(), &auth.ListUsersOptions{
		Limit:   10,
		Page:    2,
		Enabled: &enabled,
		Role:    "admin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Pages != 5 {
		t.Errorf("Pages = %d, want 5", resp.Pages)
	}
	if resp.Items != 47 {
		t.Errorf("Items = %d, want 47", resp.Items)
	}
	if len(resp.Users) != 1 {
		t.Fatalf("len(Users) = %d, want 1", len(resp.Users))
	}
	if resp.Users[0].ID != "u1" {
		t.Errorf("Users[0].ID = %q, want u1", resp.Users[0].ID)
	}
}

func TestAdminListUsers_Pagination(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		return mockResp(200, nil, `{
			"pages":3,
			"items":25,
			"users":[
				{"id":"u1","emails":[],"roles":["admin"],"verified":true,"enabled":true},
				{"id":"u2","emails":[],"roles":["viewer"],"verified":false,"enabled":false}
			]
		}`), nil
	}

	resp, err := c.Auth().AdminListUsers(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify pagination fields are parsed correctly.
	if resp.Pages != 3 {
		t.Errorf("Pages = %d, want 3", resp.Pages)
	}
	if resp.Items != 25 {
		t.Errorf("Items = %d, want 25", resp.Items)
	}
	if len(resp.Users) != 2 {
		t.Fatalf("len(Users) = %d, want 2", len(resp.Users))
	}
	if resp.Users[0].ID != "u1" {
		t.Errorf("Users[0].ID = %q, want u1", resp.Users[0].ID)
	}
	if resp.Users[1].ID != "u2" {
		t.Errorf("Users[1].ID = %q, want u2", resp.Users[1].ID)
	}
	if !resp.Users[0].Enabled {
		t.Error("Users[0].Enabled = false, want true")
	}
	if resp.Users[1].Enabled {
		t.Error("Users[1].Enabled = true, want false")
	}
}

func TestDeleteSession(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", req.Method)
		}
		// Path should contain the session ID.
		expectedPath := "/session/sess-delete-me"
		if req.URL.Path != expectedPath {
			t.Errorf("path = %q, want %s", req.URL.Path, expectedPath)
		}

		return mockResp(204, nil, ""), nil
	}

	err := c.Auth().DeleteSession(context.Background(), "sess-delete-me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPIError_401(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		h := make(http.Header)
		h.Set("X-Error", "invalid bearer token")
		h.Set("X-Error-Id", "auth-401")
		return mockResp(401, h, `{"error":"unauthorized"}`), nil
	}

	_, err := c.Auth().GetSession(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*client.APIError)
	if !ok {
		t.Fatalf("expected *client.APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid bearer token" {
		t.Errorf("Message = %q, want 'invalid bearer token'", apiErr.Message)
	}
	if apiErr.ErrorID != "auth-401" {
		t.Errorf("ErrorID = %q, want 'auth-401'", apiErr.ErrorID)
	}
	if !apiErr.IsUnauthorized() {
		t.Error("IsUnauthorized() = false, want true")
	}
}
