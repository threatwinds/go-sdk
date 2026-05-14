package client

import (
	"net/http"
	"testing"
)

func TestAPIError_ErrorMessage(t *testing.T) {
	err := newAPIError("GET", "/test", http.StatusNotFound, "not found", "err-123", "", []byte(`{}`))
	got := err.Error()
	if got != "404: GET /test: not found" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestAPIError_IsMethods(t *testing.T) {
	tests := []struct {
		status int
		fn     func(*APIError) bool
		name   string
	}{
		{400, (*APIError).IsValidationError, "validation"},
		{401, (*APIError).IsUnauthorized, "unauthorized"},
		{403, (*APIError).IsForbidden, "forbidden"},
		{404, (*APIError).IsNotFound, "not found"},
		{429, (*APIError).IsRateLimited, "rate limited"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := newAPIError("GET", "/", tc.status, "msg", "id", "", nil)
			if !tc.fn(err) {
				t.Errorf("Is*() should return true for %d", tc.status)
			}
		})
	}
}

func TestAPIError_IsMethods_FalseForOtherStatus(t *testing.T) {
	err := newAPIError("GET", "/", http.StatusInternalServerError, "msg", "id", "", nil)
	if err.IsNotFound() || err.IsUnauthorized() || err.IsForbidden() || err.IsRateLimited() || err.IsValidationError() {
		t.Error("all Is*() should be false for 500")
	}
}

func TestAPIError_Fields(t *testing.T) {
	body := []byte(`{"detail":"test"}`)
	err := newAPIError("DELETE", "/item/5", http.StatusForbidden, "forbidden", "err-42", "", body)
	if err.StatusCode != 403 {
		t.Error("status wrong")
	}
	if err.Message != "forbidden" {
		t.Error("message wrong")
	}
	if err.ErrorID != "err-42" {
		t.Error("errorID wrong")
	}
	if string(err.Body) != string(body) {
		t.Error("body wrong")
	}
}

func TestSDKError(t *testing.T) {
	e := newSDKErr("test message")
	if e.Error() != "client: test message" {
		t.Errorf("unexpected: %q", e.Error())
	}
}
