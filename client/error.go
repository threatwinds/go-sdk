package client

import (
	"fmt"

	"github.com/threatwinds/go-sdk/catcher"
)

// APIError is returned when an API call results in a 4xx or 5xx response.
// It implements the error interface.
//
// It is deliberately a plain error rather than a *catcher.SdkError: catcher.Error
// short-circuits when handed an *SdkError cause and returns it unchanged,
// discarding the wrapping call's message, args and status override. A caller
// that wraps this error therefore keeps its own framing, and inherits only the
// upstream's occurrence id, via CatcherErrorID below. See utils.RemoteError,
// which is the same decision for the same reason on the generic transport
// helper.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Method     string `json:"-"`
	Path       string `json:"-"`
	Message    string `json:"message"`
	// ErrorID is the upstream's x-error-id response header exactly as
	// received, unfiltered — this is a client-side type and the field means
	// "what the server said". Whether that value is fit to become *this*
	// process's occurrence id is a separate question, answered at the point of
	// donation by CatcherErrorID rather than here, so that a caller inspecting
	// the field still sees what actually came back.
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

// Error renders the failure. It is nil-safe, and has to be now that this error
// is handed to catcher as a cause: catcher.build calls Error() on whatever
// cause it is given to fill Cause, and errors.As — which is how catcher finds
// this type — matches on type alone, so a typed-nil *APIError boxed in a
// non-nil error interface reaches this method on a nil receiver. That is the
// same trap catcher guards for *SdkError (see ToSdkError) and utils.RemoteError
// guards for itself, and it renders the same way: "<nil>".
func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}

	return fmt.Sprintf("%d: %s %s: %s", e.StatusCode, e.Method, e.Path, e.Message)
}

// CatcherErrorID implements catcher.ErrorIDCarrier, so that a caller wrapping
// this error with catcher.Error or catcher.New builds its own error — its
// message, its args, its status — that nonetheless inherits the upstream
// ThreatWinds service's occurrence id, rather than minting a second, unrelated
// id for one failure.
//
// Without it the id read at client.doOnce went nowhere: APIError has no Unwrap,
// so catcher's errors.As walk found no carrier, and the id was not in Error()'s
// text either (it renders status, method, path and message only). catcher
// minted a fresh UUID, and the two services' log lines for one failure could no
// longer be correlated by id — the exact failure this interface exists to
// close, left open here only because this package did not import catcher.
//
// The guard is applied here rather than when the header is read: ErrorID keeps
// its meaning of "what the server sent", while only the value that becomes an
// occurrence id in this process is required to be a canonical UUID. A malformed
// one is not donated, and catcher mints its own instead — see
// catcher.AdoptErrorID for why a remote-controlled id cannot be taken on trust.
//
// Nil-safe: errors.As matches on type alone, so a typed-nil *APIError boxed in
// a non-nil error would otherwise reach this method on a nil receiver. catcher
// also guards that case, but a method reachable through an interface should not
// depend on its only caller being careful.
func (e *APIError) CatcherErrorID() string {
	if e == nil {
		return ""
	}

	return catcher.AdoptErrorID(e.ErrorID)
}

var _ catcher.ErrorIDCarrier = (*APIError)(nil)

// RetryAfter returns the Retry-After header value stored on the error,
// used by the retry logic to determine backoff duration.
func (e *APIError) RetryAfter() string {
	if e == nil {
		return ""
	}

	return e.retryAfter
}

// The status predicates are nil-safe for the same reason Error is: the usual
// way a caller reaches one is errors.As, which sets its target from a type
// match and will happily hand back a nil pointer for a typed-nil in the chain.
// All of them answer false for a nil receiver — "this is not that kind of
// failure" — rather than panicking inside an `if err.IsNotFound()`.
func (e *APIError) IsNotFound() bool        { return e != nil && e.StatusCode == 404 }
func (e *APIError) IsUnauthorized() bool    { return e != nil && e.StatusCode == 401 }
func (e *APIError) IsForbidden() bool       { return e != nil && e.StatusCode == 403 }
func (e *APIError) IsRateLimited() bool     { return e != nil && e.StatusCode == 429 }
func (e *APIError) IsValidationError() bool { return e != nil && e.StatusCode == 400 }

// SDKError is returned by New() for configuration errors (not HTTP errors).
//
// It deliberately does not implement catcher.ErrorIDCarrier. It is raised
// before any request is made — a missing or conflicting authentication option —
// so there is no upstream, no x-error-id, and nothing to adopt. A caller
// wrapping it with catcher gets a freshly minted id, which is the correct
// outcome for a failure that originated in this process: implementing the
// interface to return "" would be indistinguishable in behaviour and would
// falsely advertise that this error type participates in cross-service
// propagation.
type SDKError struct {
	msg string
}

func newSDKErr(msg string) *SDKError { return &SDKError{msg: msg} }

// Error is nil-safe for the same reason APIError.Error is: it may be handed to
// catcher as a cause, and catcher renders any cause it is given.
func (e *SDKError) Error() string {
	if e == nil {
		return "<nil>"
	}

	return "client: " + e.msg
}
