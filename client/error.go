package client

import (
	"fmt"

	"github.com/google/uuid"
)

// adoptErrorID accepts an upstream's x-error-id only when it is a canonical
// 36-character UUID — the shape catcher always mints and therefore the only
// shape a ThreatWinds service can legitimately send. A value that fails is not
// adopted, and the failure falls back to the id minted for it locally, so a
// buggy or hostile endpoint cannot hand this process an id already in use by an
// unrelated in-flight failure, or one megabytes long
// (http.Transport.MaxResponseHeaderBytes allows 10MB of headers). The adopted
// value is echoed back out in the calling service's own x-error-id response
// header and in every log line about the failure, so neither is hypothetical.
//
// It is a copy of catcher.AdoptErrorID rather than a call to it, and that is
// the whole reason this package still imports nothing from the SDK. Importing
// catcher for these six lines grows this package's dependency closure from 193
// packages to 317 — gin, go-playground/validator, goccy/go-yaml, protobuf and
// mongo-driver's bson among them — and, worse, catcher's package init starts a
// permanent goroutine that writes to os.Stdout, which every binary linking this
// client would then inherit before main runs, with no opt-out. A CLI or an
// integrator that wanted an HTTP client does not get a logging framework with
// it. The copy cannot drift: TestAdoptErrorIDMatchesCatcher runs both over the
// same table, and it is a test, so catcher is linked into the test binary
// alone.
//
// The guard is not load-bearing for catcher's own invariant either — catcher's
// build applies its copy to every id it is handed, whatever a carrier donates.
// This one decides what *this package* reports and donates.
func adoptErrorID(id string) string {
	// uuid.Parse also accepts the urn:, braced and undashed forms; requiring
	// exactly 36 characters restricts it to the canonical one, so the value
	// written back out is the same shape every other id in the logs has.
	if len(id) != 36 {
		return ""
	}

	if _, err := uuid.Parse(id); err != nil {
		return ""
	}

	return id
}

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

	// localID is minted when the upstream supplied no usable id, so that this
	// failure carries exactly one occurrence id from the moment it is created,
	// whether or not any caller ever hands it to catcher. Without it,
	// CatcherErrorID answers "" on that case and each layer that wraps this
	// error mints an id of its own: one failure, wrapped by a handler for its
	// own response and again by a middleware or a caller, ends up with two
	// unrelated ids — precisely the property ErrorID exists to prevent. That is
	// not a corner case but today's default, since every service in the fleet
	// is pinned to a go-sdk whose GinError sends the 32-character md5 Code in
	// x-error-id, which adoptErrorID refuses.
	//
	// Unexported, and deliberately not folded into ErrorID: that field means
	// "what the server sent", and a caller reading it to decide whether a hop
	// is traceable end to end must not be told yes when the answer is no. Same
	// split, for the same reason, as utils.RemoteError.ID and its localID.
	//
	// It is minted in newAPIError rather than lazily in CatcherErrorID because
	// an id that appears on first use would differ between two wraps of the
	// same error unless it were also stored, and storing it there would mean
	// mutating an error value that callers hold concurrently. An *APIError
	// built as a struct literal — which no code path in this package does —
	// therefore has no local id and donates only what the server sent.
	localID string `json:"-"`
}

func newAPIError(method, path string, status int, message, errorID, retryAfter string, body []byte) *APIError {
	e := &APIError{
		StatusCode: status,
		Method:     method,
		Path:       path,
		Message:    message,
		ErrorID:    errorID,
		retryAfter: retryAfter,
		Body:       body,
	}

	// Nothing usable came back: the upstream sent no id, sent one the guard
	// refuses, or is not a ThreatWinds service at all. This failure still gets
	// exactly one occurrence id, minted here so that every wrap of it agrees on
	// which occurrence it is.
	if adoptErrorID(errorID) == "" {
		e.localID = uuid.NewString()
	}

	return e
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

// CatcherErrorID satisfies catcher.ErrorIDCarrier structurally — the interface
// is one method, and satisfying it requires declaring that method, not importing
// the package that declares the interface (see adoptErrorID for why this package
// imports nothing from the SDK). A caller wrapping this error with catcher.Error
// or catcher.New therefore builds its own error — its message, its args, its
// status — that nonetheless inherits this failure's occurrence id, rather than
// minting a second, unrelated id for one failure. The compile-time assertion
// that the interface is still the shape this method answers lives in
// error_test.go, where linking catcher costs nothing.
//
// Without it the id read at client.doOnce went nowhere: APIError has no Unwrap,
// so catcher's errors.As walk found no carrier, and the id was not in Error()'s
// text either (it renders status, method, path and message only). catcher
// minted a fresh UUID, and the two services' log lines for one failure could no
// longer be correlated by id — the exact failure this interface exists to
// close.
//
// The guard is applied here rather than when the header is read: ErrorID keeps
// its meaning of "what the server sent", while only the value that becomes an
// occurrence id in this process is required to be a canonical UUID. A malformed
// one is not donated — see adoptErrorID for why a remote-controlled id cannot
// be taken on trust.
//
// It falls back to the id minted for this failure when the upstream supplied
// none the guard would accept, so the answer to "which occurrence is this?" is
// never empty and never changes between two wraps of the same error. The
// difference between the two cases is not lost — it is exactly what ErrorID
// reports — but it is not the wrapping caller's problem: either way this
// failure has one id, and the caller's error gets that one rather than a
// second.
//
// Nil-safe: errors.As matches on type alone, so a typed-nil *APIError boxed in
// a non-nil error would otherwise reach this method on a nil receiver. catcher
// also guards that case, but a method reachable through an interface should not
// depend on its only caller being careful.
func (e *APIError) CatcherErrorID() string {
	if e == nil {
		return ""
	}

	if id := adoptErrorID(e.ErrorID); id != "" {
		return id
	}

	return e.localID
}

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
