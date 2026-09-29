// Package relay turns an upstream ThreatWinds service's failed HTTP response
// into the *catcher.SdkError a *relaying* service answers its own client with:
// the upstream's status becomes this service's status, its severity this
// service's severity, and its occurrence id this failure's id.
//
// # Why this is not in utils
//
// utils.DoReq reads the same response into a *utils.RemoteError, which is a
// plain error precisely so that a caller's own `catcher.Error("calling X
// failed", err, args)` keeps the caller's message, args and status while still
// inheriting the upstream's occurrence id. That is what almost every caller
// wants, and it needs nothing from catcher — which is why utils imports
// nothing from catcher, and why binaries that only fetch a file (the ETL
// modules, through entities) do not link gin, protobuf and mongo-driver's bson
// just to make an HTTP request, or inherit catcher's stdout-writing goroutine
// at init.
//
// Relaying is the rarer, opt-in decision, and it is the one that genuinely
// needs an *SdkError. It lives here so the cost lands on the callers that make
// that choice — services that already have gin linked because they serve HTTP —
// and on nobody else.
package relay

import (
	"errors"
	"net/http"

	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/utils"
)

// remoteErrorMessage is the message carried by every *catcher.SdkError built in
// this package. It is deliberately constant and carries no upstream detail, for
// two independent reasons:
//
//   - Disclosure. catcher.SecureString returns Msg verbatim — and only Msg —
//     for a status >= 500, and catcher.GinError writes that string into the
//     relaying service's own x-error response header and JSON error body.
//     Anything folded into Msg therefore crosses the next service boundary
//     unfiltered, all the way to the end client: an upstream's "pq: password
//     authentication failed for user \"casework\"" included.
//   - Grouping. Code is an md5 of Msg and is documented as the error's *type* —
//     "shared by every occurrence of the same message anywhere". It is a type
//     fingerprint only for as long as Msg is per-type. Fold a response body in
//     and every distinct upstream body produces a distinct Code, which is
//     exactly the per-occurrence/per-type conflation Code exists to avoid.
//
// The remote's own text goes to Cause (suppressed by SecureString at >= 500),
// its status to Args["status"], and its occurrence id to ErrorID. Both halves
// are pinned by TestSdkErrorFromResponse_MessageAndCodeAreConstant and
// utils' TestDoReq_WrapKeepsMessageAndCodeStableAcrossBodies.
const remoteErrorMessage = "remote service error"

// SdkErrorFromResponse turns a failed HTTP response into a *catcher.SdkError,
// reading only the status line and the error headers catcher.GinError sets on
// the way out of a ThreatWinds service:
//
//	x-error-id → ErrorID (and Args["error_id"]), adopted so one failure keeps
//	             a single id across every service that touches it — but only
//	             when it is a well-formed UUID, see utils.RemoteError.ID.
//	             Absent or malformed → the id utils minted for this response,
//	             so the failure is identified exactly once either way.
//	x-error    → Cause, the remote's own rendered error text, bounded by utils
//	             to its maximum detail size. Absent → "unknown cause", the same
//	             text catcher itself uses.
//	status     → Severity, via calcSeverityFromStatus, and — clamped by
//	             relayStatus — Args["status"] (an int, as everywhere else in
//	             the SDK).
//
// Args["status"] is the status a *relaying* handler will answer its own client
// with, since that is the key catcher.GinError reads. Using this function is
// therefore a decision to pass the upstream's status through: an upstream 401
// becomes your client's 401. Most callers do not want that (their client's
// credentials are not the ones that were rejected) and should let
// catcher.Error/catcher.New build an error with their own args["status"] around
// the *utils.RemoteError DoReq returns — which still inherits the occurrence id.
//
// Relaying a status < 500 also relays detail: catcher.SecureString returns the
// full Error() text — Cause and Args included — for anything under 500, and
// Cause here is the upstream's own message. That is SecureString's deliberate
// design (a client error is explained to the client), but the explanation now
// comes from a service the client never called, so relay a 4xx only when the
// upstream's text is genuinely about the caller's request.
//
// It never touches resp.Body: the body may already have been consumed, may be
// nil on a hand-built response, and is the caller's to read and to close.
//
// The error is built with catcher.New, not catcher.Error, so nothing is logged
// inside a transport helper — the error is logged once, wherever the caller
// ends up handling it (or automatically, if it reaches catcher.GinError).
//
// Msg is the constant remoteErrorMessage and never the upstream's text; see
// that constant's doc for the two invariants that depend on it. Code follows
// from Msg: it identifies the error's type (an md5 of the message, per
// catcher.SdkError's field doc) and is therefore the same for every error this
// package builds, while the per-occurrence, greppable id is ErrorID.
// Conflating the two is precisely the drift catcher.GinError was fixed to
// remove.
//
// A nil response, a nil Header map and absent headers are all handled: a nil
// response is reported as a 500, since "no response at all" is a local
// failure and 500 is the status catcher would otherwise default to.
func SdkErrorFromResponse(resp *http.Response) *catcher.SdkError {
	return sdkErrorFrom(utils.RemoteErrorFromResponse(resp))
}

// sdkErrorFrom converts a RemoteError into the *SdkError a relaying caller
// wants, applying the message, status and severity rules documented on
// SdkErrorFromResponse.
func sdkErrorFrom(remote *utils.RemoteError) *catcher.SdkError {
	args := map[string]any{"status": relayStatus(remote.Status)}
	if remote.ID != "" {
		args["error_id"] = remote.ID
	}

	var cause error
	if remote.Detail != "" {
		cause = errors.New(remote.Detail)
	}

	err := catcher.New(remoteErrorMessage, cause, args)

	// One response, one occurrence id. When the upstream supplied one it is
	// already in args and this assignment is an identity; when it did not,
	// catcher minted its own a moment ago and this replaces it with the id
	// utils minted for the same response, so that a caller holding both the
	// *RemoteError and this *SdkError sees one incident rather than two.
	// Args["error_id"] is deliberately left alone: it records that the upstream
	// told us the id, which stays true only in the adopted case.
	//
	// Guarded rather than assigned unconditionally: a *RemoteError built from a
	// response always leaves an id to donate, but an *SdkError with an empty
	// ErrorID is the one state catcher guarantees can never exist, and a future
	// change to CatcherErrorID must not be able to produce one from here.
	if id := remote.CatcherErrorID(); id != "" {
		err.ErrorID = id
	}

	// The trace catcher captures here is this process's stack — DoReq and its
	// caller — which describes where the response was *received*, not where
	// the failure happened. The frames that explain the failure are in the
	// upstream service's own log line, findable by ErrorID. Clearing it also
	// makes the result deterministic rather than dependent on CATCHER_NO_TRACE
	// (which already omits traces by default).
	err.Trace = nil

	// Severity is stated here rather than left to catcher's derivation from
	// args["status"]: it follows the status the upstream actually returned,
	// which after relayStatus is not necessarily what args["status"] holds
	// (calcSeverityFromStatus mirrors catcher.calculateSeverity, which is
	// unexported and so cannot be called from here). Stating it keeps this
	// helper's contract — severity follows the HTTP status of the response —
	// true regardless of how catcher chooses to read its args.
	err.Severity = calcSeverityFromStatus(remote.Status)

	return err
}

// relayStatus maps an upstream status onto a status the relaying service can
// actually answer with, which is what Args["status"] means to
// catcher.GinError.
//
// Anything outside 4xx/5xx becomes a 500 — the same clamp the nil-response
// case has always had, and for the same reason. A 0 (a hand-built response, a
// mock, a cached response or a custom RoundTripper that never set StatusCode)
// is the sharp case: gin's responseWriter.WriteHeader ignores a non-positive
// code (`if code > 0 && ...`), so the error envelope would go out as a 200 and
// every client-side `if status >= 400` check would wave the failure through. A
// 204 or 304 is no better — gin's bodyAllowedForStatus is false for those, so
// the error body is dropped entirely — and a status above 599 reaches
// net/http's checkWriteHeaderCode, which panics outright above 999.
func relayStatus(status int) int {
	if status < 400 || status > 599 {
		return http.StatusInternalServerError
	}

	return status
}

// calcSeverityFromStatus maps an HTTP status code to a catcher severity level.
//
// It mirrors catcher.calculateSeverity, which is unexported. The duplication is
// deliberate and narrow: it takes a plain int rather than an interface{} that
// needs casting, and it keeps the severity of a response-derived error pinned
// by this package's own tests.
func calcSeverityFromStatus(status int) string {
	switch {
	case status >= 100 && status < 200:
		return "DEBUG"
	case status >= 200 && status < 300:
		return "INFO"
	case status >= 300 && status < 400:
		return "NOTICE"
	case status >= 400 && status < 500:
		return "WARNING"
	case status >= 500 && status < 502:
		return "ERROR"
	case status >= 502 && status < 509:
		return "CRITICAL"
	case status >= 509 && status < 511:
		return "ALERT"
	}

	return "ERROR"
}
