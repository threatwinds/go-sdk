package client

import (
	"net/http"
	"testing"
	"time"
)

func TestNew_NoAuth(t *testing.T) {
	_, err := New()
	if err == nil {
		t.Fatal("expected error with no auth, got nil")
	}
	if _, ok := err.(*SDKError); !ok {
		t.Fatalf("expected *SDKError, got %T", err)
	}
}

func TestNew_PartialAPIKey(t *testing.T) {
	// Key without secret should be rejected
	_, err := New(WithAPIKey("key", ""))
	if err == nil {
		t.Fatal("expected error with partial API key, got nil")
	}
	// Secret without key should be rejected
	_, err = New(WithAPIKey("", "secret"))
	if err == nil {
		t.Fatal("expected error with partial API key (secret only), got nil")
	}
}

func TestNew_DuplicateAuth(t *testing.T) {
	_, err := New(WithAPIKey("key", "secret"), WithBearer("token"))
	if err == nil {
		t.Fatal("expected error with both auth methods, got nil")
	}
	if _, ok := err.(*SDKError); !ok {
		t.Fatalf("expected *SDKError, got %T", err)
	}
}

func TestNew_APIKeyValue(t *testing.T) {
	c, err := New(WithAPIKey("my-key", "my-secret"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.apiKey != "my-key" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "my-key")
	}
	if c.apiSecret != "my-secret" {
		t.Errorf("apiSecret = %q, want %q", c.apiSecret, "my-secret")
	}
}

func TestNew_BearerValue(t *testing.T) {
	c, err := New(WithBearer("my-token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.bearer != "my-token" {
		t.Errorf("bearer = %q, want %q", c.bearer, "my-token")
	}
}

func TestNew_Defaults(t *testing.T) {
	c, err := New(WithBearer("token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.endpoint != DefaultEndpoint {
		t.Errorf("endpoint = %q, want %q", c.endpoint, DefaultEndpoint)
	}
	if c.maxRetries != defaultMaxRetries {
		t.Errorf("maxRetries = %d, want %d", c.maxRetries, defaultMaxRetries)
	}
}

func TestNew_CustomTimeout(t *testing.T) {
	c, err := New(WithBearer("token"), WithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The timeout is applied to the http.Client; verify the client has the right timeout.
	if c.httpClient.Timeout != 10*time.Second {
		t.Errorf("httpClient.Timeout = %v, want %v", c.httpClient.Timeout, 10*time.Second)
	}
}

func TestNew_CustomMaxRetries(t *testing.T) {
	c, err := New(WithBearer("token"), WithMaxRetries(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.maxRetries != 5 {
		t.Errorf("maxRetries = %d, want %d", c.maxRetries, 5)
	}
}

func TestNew_CustomHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	c, err := New(WithBearer("token"), WithHTTPClient(custom))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.httpClient != custom {
		t.Error("httpClient should be the custom client")
	}
}

func TestServiceAccessors(t *testing.T) {
	c, err := New(WithBearer("token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Auth() == nil {
		t.Error("Auth() returned nil")
	}
	if c.Billing() == nil {
		t.Error("Billing() returned nil")
	}
	if c.Compute() == nil {
		t.Error("Compute() returned nil")
	}
}
