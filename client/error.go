package client

import "fmt"

// APIError is returned when an API call results in a 4xx or 5xx response.
// It implements the error interface.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Method     string `json:"-"`
	Path       string `json:"-"`
	Message    string `json:"message"`
	ErrorID    string `json:"error_id"`
	Body       []byte `json:"-"`
	retryAfter string `json:"-"` // internal: Retry-After header for retry logic
}

func newAPIError(method, path string, status int, message, errorID, retryAfter string, body []byte) *APIError {
	return &APIError{
		StatusCode: status,
		Method:     method,
		Path:       path,
		Message:    message,
		ErrorID:    errorID,
		retryAfter: retryAfter,
		Body:       body,
	}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%d: %s %s: %s", e.StatusCode, e.Method, e.Path, e.Message)
}

// HeaderGet returns the value of the named header stored on the error.
// Currently only "Retry-After" is supported.
func (e *APIError) HeaderGet(key string) string {
	if key == "Retry-After" {
		return e.retryAfter
	}
	return ""
}

func (e *APIError) IsNotFound() bool        { return e.StatusCode == 404 }
func (e *APIError) IsUnauthorized() bool    { return e.StatusCode == 401 }
func (e *APIError) IsForbidden() bool       { return e.StatusCode == 403 }
func (e *APIError) IsRateLimited() bool     { return e.StatusCode == 429 }
func (e *APIError) IsValidationError() bool { return e.StatusCode == 400 }

// SDKError is returned by New() for configuration errors (not HTTP errors).
type SDKError struct {
	msg string
}

func newSDKErr(msg string) *SDKError { return &SDKError{msg: msg} }
func (e *SDKError) Error() string    { return "client: " + e.msg }
