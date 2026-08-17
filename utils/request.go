package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/threatwinds/go-sdk/catcher"
)

// remoteErrorMessage is the message carried by every *catcher.SdkError built in
// this file. It is deliberately constant and carries no upstream detail, for
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
// TestDoReq_WrapKeepsMessageAndCodeStableAcrossBodies.
const remoteErrorMessage = "remote service error"

// maxErrorDetailSize bounds the upstream text this package will carry: the
// response body DoReq reads on a failure, and the x-error header value.
//
// Neither is bounded at the source — io.ReadAll has no limit of its own, and
// http.Transport.MaxResponseHeaderBytes defaults to 10MB — and both end up in
// the Cause of whatever error the caller builds, which catcher.SecureString
// renders (for a status < 500) into the relaying service's own x-error
// *response* header. An unbounded one is rejected by the load balancer well
// before it reaches anyone, turning a real failure into an opaque 502, and it
// compounds per hop, since each service quotes the one below it.
const maxErrorDetailSize = 4 * 1024

// RemoteError is the error DoReq returns when an upstream answers with a
// status >= 400.
//
// It is deliberately *not* a *catcher.SdkError. catcher.Error short-circuits
// when the cause it is handed is already an *SdkError: it returns that error
// unchanged and silently drops the wrapping call's message, args and status
// override (see catcher.Error's doc comment). A transport helper returning an
// *SdkError therefore converts every existing
// `catcher.Error("calling X failed", err, args)` in the org into a no-op wrap —
// the caller's message, its arg keys and the HTTP status it asked to answer
// with all vanish, replaced by the upstream's. Worse, it does so only on the
// HTTP-failure path and not on the transport-failure paths beside it, so a
// single call site logs two different shapes depending on how the request
// happened to fail.
//
// A plain error keeps every caller's message, args and status exactly where
// they were, while catcher.ErrorIDCarrier (implemented below) still carries the
// upstream's occurrence id into the error the caller builds — which is the
// whole point of reading x-error-id in the first place.
//
// A caller that genuinely wants to relay the upstream's own status and severity
// — a proxy, rather than a service making a call of its own — can opt in
// explicitly with SdkErrorFromResponse.
type RemoteError struct {
	// Status is the upstream's HTTP status, exactly as received. It is not
	// necessarily a status the relaying service should answer with; see
	// SdkErrorFromResponse's note on Args["status"].
	Status int

	// ID is the upstream's x-error-id when it supplied a well-formed one: the
	// occurrence id that lets one failure be grepped across every service's
	// logs. Empty when the upstream sent none, or sent one catcher.AdoptErrorID
	// refused — this field means "the id the upstream gave us" and nothing
	// else, so that a caller can tell an inherited occurrence from a local one.
	// CatcherErrorID is what donates an id either way; see it for the fallback.
	ID string

	// Detail is the upstream's own error text — the first maxErrorDetailSize
	// bytes of its response body, or its x-error header when the body is
	// empty. Never the whole of an arbitrarily large body.
	Detail string

	// localID is minted when the upstream supplied no usable id, so that every
	// failure this package reports carries exactly one occurrence id from the
	// moment it is created — whether or not the caller ever hands it to
	// catcher. Unexported, and deliberately not folded into ID: ID's contract
	// above is what distinguishes "the upstream told us which occurrence this
	// is" from "we named it ourselves", and a caller reading ID to decide
	// whether a hop is traceable end to end must not be told yes when the
	// answer is no.
	localID string
}

// Error renders the failure in the exact shape DoReq has always produced, so
// existing callers that log this text (or match on it) are unaffected. The only
// difference is that the detail is now bounded.
func (e *RemoteError) Error() string {
	if e == nil {
		return "<nil>"
	}

	detail := e.Detail
	if detail == "" {
		detail = "unknown cause"
	}

	return fmt.Sprintf("error response (status=%d): %s", e.Status, detail)
}

// CatcherErrorID implements catcher.ErrorIDCarrier, so that a caller wrapping
// this error with catcher.Error or catcher.New builds its own error — its
// message, its args, its status — that nonetheless inherits the upstream's
// occurrence id, rather than minting a second, unrelated id for one failure.
//
// It falls back to the id minted for this response when the upstream supplied
// none, so the answer to "which occurrence is this?" is never empty. The
// difference between the two cases is not lost — it is exactly what ID reports
// — but it is not the caller's problem at the point of wrapping: either way
// this failure has one id, and the caller's error gets that one rather than a
// second.
func (e *RemoteError) CatcherErrorID() string {
	if e == nil {
		return ""
	}

	if e.ID != "" {
		return e.ID
	}

	return e.localID
}

var _ catcher.ErrorIDCarrier = (*RemoteError)(nil)

// generatedIDError attaches a locally minted occurrence id to a failure that
// has no upstream id to adopt — a transport error where no response was ever
// received, a malformed response body, or a response from an endpoint that is
// not a ThreatWinds service and therefore never sends x-error-id.
//
// The point is not to pretend an id was adopted. It is that a failure with no
// id is untraceable: an operator reading "bad status downloading from …" in one
// service's log has nothing to grep for it by. Minting here rather than leaving
// it to catcher also means the id in the rendered text and the id in the
// SdkError a caller builds around it are the same value, instead of the text
// having none and the log line having one the text cannot be matched to.
//
// It wraps rather than replaces: Unwrap keeps the original error reachable to
// errors.Is and errors.As, so a caller matching on os.ErrDeadlineExceeded or a
// *url.Error still finds it.
type generatedIDError struct {
	err error
	id  string
}

// withGeneratedErrorID wraps err so it carries an occurrence id of its own.
// A nil error is returned unchanged — wrapping one would turn "no failure" into
// a non-nil error interface holding a nil-ish value, which is how a success
// path starts being reported as a failure.
func withGeneratedErrorID(err error) error {
	if err == nil {
		return nil
	}

	return &generatedIDError{err: err, id: uuid.NewString()}
}

// Error renders the wrapped error's text with the occurrence id appended, so a
// caller that only ever logs the error — rather than wrapping it with catcher —
// still has something to grep by.
func (e *generatedIDError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.err == nil {
		return fmt.Sprintf("unknown cause (error_id=%s)", e.id)
	}

	return fmt.Sprintf("%s (error_id=%s)", e.err.Error(), e.id)
}

func (e *generatedIDError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.err
}

func (e *generatedIDError) CatcherErrorID() string {
	if e == nil {
		return ""
	}

	return e.id
}

var _ catcher.ErrorIDCarrier = (*generatedIDError)(nil)

// SdkErrorFromResponse turns a failed HTTP response into a *catcher.SdkError,
// reading only the status line and the error headers catcher.GinError sets on
// the way out of a ThreatWinds service:
//
//	x-error-id → ErrorID (and Args["error_id"]), adopted so one failure keeps
//	             a single id across every service that touches it — but only
//	             when it is a well-formed UUID, see adoptErrorID. Absent or
//	             malformed → catcher mints a fresh UUID, exactly as it would
//	             for any locally-raised error.
//	x-error    → Cause, the remote's own rendered error text, bounded to
//	             maxErrorDetailSize. Absent → "unknown cause", the same text
//	             catcher itself uses.
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
// the *RemoteError DoReq returns — which still inherits the occurrence id.
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
	return sdkErrorFrom(remoteErrorFrom(resp, nil))
}

// remoteErrorFrom is the single place an HTTP failure response is read into
// this package's own error type. The body is a parameter rather than read here
// because SdkErrorFromResponse must not touch resp.Body while DoReq has
// already read it; every other field — status, adopted error id, detail — is
// derived here, so the two entry points cannot drift apart.
func remoteErrorFrom(resp *http.Response, body []byte) *RemoteError {
	remote := &RemoteError{Status: http.StatusInternalServerError}

	var header http.Header
	if resp != nil {
		remote.Status = resp.StatusCode
		header = resp.Header
	}

	remote.ID = adoptErrorID(headerValue(header, "x-error-id"))
	if remote.ID == "" {
		// Nothing to adopt: the upstream sent no id, sent a malformed one, or
		// is not a ThreatWinds service at all. This response still gets exactly
		// one occurrence id, minted here so that both errors derivable from it
		// — the *RemoteError DoReq returns and the *SdkError
		// SdkErrorFromResponse builds — agree on which occurrence it is.
		remote.localID = uuid.NewString()
	}

	// The body is the richer of the two and the only one a non-ThreatWinds
	// endpoint supplies; x-error is the fallback for a response whose body was
	// empty or unread (SdkErrorFromResponse passes none).
	remote.Detail = truncateDetail(string(body))
	if remote.Detail == "" {
		remote.Detail = truncateDetail(headerValue(header, "x-error"))
	}

	return remote
}

// sdkErrorFrom converts a RemoteError into the *SdkError a relaying caller
// wants, applying the message, status and severity rules documented on
// SdkErrorFromResponse.
func sdkErrorFrom(remote *RemoteError) *catcher.SdkError {
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
	// remoteErrorFrom minted for the same response, so that a caller holding
	// both the *RemoteError and this *SdkError sees one incident rather than
	// two. Args["error_id"] is deliberately left alone: it records that the
	// upstream told us the id, which stays true only in the adopted case.
	err.ErrorID = remote.CatcherErrorID()

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

// adoptErrorID accepts an upstream's x-error-id only when it is a canonical
// 36-character UUID — the shape catcher itself always mints and therefore the
// only shape a ThreatWinds service can legitimately send. See
// catcher.AdoptErrorID for why a remote-controlled id cannot be taken on trust.
//
// The rule lives in catcher because catcher owns the id's format: it is the
// only thing that mints one and the only thing that has to keep every id in the
// logs the same shape. This package and client both defer to it so the two
// cannot drift apart. The name is kept here for the call site's sake.
func adoptErrorID(id string) string {
	return catcher.AdoptErrorID(id)
}

// truncateDetail bounds upstream-supplied text to maxErrorDetailSize; see that
// constant for why. The cut is repaired to valid UTF-8, since a byte-wise cut
// can land mid-rune and the result is written into a log line and an HTTP
// header.
func truncateDetail(detail string) string {
	if len(detail) <= maxErrorDetailSize {
		return detail
	}

	return strings.ToValidUTF8(detail[:maxErrorDetailSize], "") + "… (truncated)"
}

// headerValue reads a header value, tolerating a Header map whose keys were
// never canonicalized. http.Header.Get canonicalizes the *lookup* key
// (textproto.CanonicalMIMEHeaderKey), so it finds "X-Error-Id" — what net/http
// stores for any response that came off the wire — but silently misses an
// entry a caller stored under the literal key "x-error-id", which is legal
// since http.Header is a plain map and is what hand-built responses (mocks,
// round-trippers, cached responses) tend to contain. Missing it would drop the
// upstream's error id, so both spellings are checked.
func headerValue(h http.Header, key string) string {
	if len(h) == 0 {
		return ""
	}

	if v := h.Get(key); v != "" {
		return v
	}

	for _, v := range h[key] {
		if v != "" {
			return v
		}
	}

	return ""
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

// DoReq sends an HTTP request and processes the response.
//
// This function sends an HTTP request to the specified URL with the given
// method, data, and headers. It returns the response body unmarshalled into
// the specified response type, the HTTP status code, and an error if any
// occurred during the process.
//
// Type Parameters:
//   - response: The type into which the response body will be unmarshalled.
//
// Parameters:
//   - url: The URL to which the request is sent.
//   - data: The request payload as a byte slice.
//   - method: The HTTP method to use for the request (e.g., "GET", "POST").
//   - headers: A map of headers to include in the request.
//
// Returns:
//   - response: The response body unmarshalled into the specified type.
//   - int: The HTTP status code of the response.
//   - error: An error if any occurred during the request or response
//     processing, otherwise nil. For a response with a status >= 400 that
//     error is a *RemoteError — a plain error, so wrapping it with
//     catcher.Error/catcher.New builds the caller's own error normally,
//     keeping the caller's message, args and status while inheriting the
//     upstream's occurrence id. Use errors.As to reach the upstream's status
//     and detail; use SdkErrorFromResponse if you mean to relay the upstream's
//     status as your own.
func DoReq[response any](url string, data []byte, method string, headers map[string]string, skipTlsVerification bool) (response, int, error) {
	var result response

	// Every error returned below carries an occurrence id: adopted from the
	// upstream's x-error-id where there is one to adopt (the *RemoteError
	// branch), generated where there is not. These branches are the latter —
	// they fail before any response exists, or on a response that could not be
	// read — so there is no id to inherit and nothing to read a header from.
	if len(data) > maxMessageSize {
		return result, http.StatusBadRequest, withGeneratedErrorID(fmt.Errorf("cannot convert to object: data size exceeds limit (size=%d bytes, limit=%d bytes)", len(data), maxMessageSize))
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return result, http.StatusInternalServerError, withGeneratedErrorID(fmt.Errorf("error creating request: %w", err))
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	// Configure HTTP client with security settings
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: skipTlsVerification,
		},
		DisableCompression: true,
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
	}
	defer tr.CloseIdleConnections()

	resp, err := client.Do(req)
	if err != nil {
		return result, http.StatusInternalServerError, withGeneratedErrorID(fmt.Errorf("error doing request: %w", err))
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		// The failure body is read here, and bounded, rather than with the
		// success body below. An error body is only ever rendered into a log
		// line or an error string — it is never unmarshalled — so there is
		// nothing to gain from holding a large one, and every reason not to:
		// it ends up in the Cause of whatever error the caller builds, which
		// a relaying service writes back out in its own x-error response
		// header. maxErrorDetailSize+1 is read so truncateDetail can tell that
		// it was cut. A read error is deliberately not reported: the status
		// line is the failure worth reporting, and a partial body is still
		// useful detail.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorDetailSize+1))

		// A plain *RemoteError, not an *SdkError: see RemoteError's doc for
		// why returning an *SdkError would silently gut every existing
		// `catcher.Error(msg, err, args)` wrap in the org. Nothing is logged
		// here either — the caller's wrap logs once, wherever it handles it.
		return result, resp.StatusCode, remoteErrorFrom(resp, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, http.StatusInternalServerError, withGeneratedErrorID(fmt.Errorf("error reading response body: %w", err))
	}

	if resp.StatusCode == http.StatusNoContent {
		return result, resp.StatusCode, nil
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, resp.StatusCode, withGeneratedErrorID(fmt.Errorf("error parsing response: %w", err))
	}

	return result, resp.StatusCode, nil
}

// DownloadOption defines a functional option for configuring the download request.
type DownloadOption func(*downloadConfig)

type downloadConfig struct {
	headers             map[string]string
	timeout             time.Duration
	skipTlsVerification bool
}

// WithHeaders sets the headers for the download request.
func WithHeaders(headers map[string]string) DownloadOption {
	return func(c *downloadConfig) {
		c.headers = headers
	}
}

// WithTimeout sets the timeout for the download request.
func WithTimeout(timeout time.Duration) DownloadOption {
	return func(c *downloadConfig) {
		c.timeout = timeout
	}
}

// WithSkipTlsVerification sets the skipTlsVerification for the download request.
func WithSkipTlsVerification(skip bool) DownloadOption {
	return func(c *downloadConfig) {
		c.skipTlsVerification = skip
	}
}

// Download downloads the content from the specified URL and saves it to the specified file.
// It returns an error if any error occurs during the process.
//
// Parameters:
//   - url: The URL from which to download the content.
//   - file: The path to the file where the content should be saved.
//   - opts: Optional functional options to configure the download (e.g., WithHeaders, WithTimeout).
//
// Returns:
//   - error: An error object if an error occurs, otherwise nil.
func Download(url, file string, opts ...DownloadOption) error {
	config := &downloadConfig{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(config)
	}

	// The two filesystem errors in this function are left as plain errors on
	// purpose. Creating a file and writing to it are local operations with no
	// remote involved and no request in flight, so there is nothing to
	// correlate across a service boundary — an occurrence id on them would be a
	// value only ever seen once, in one process, by the caller that already has
	// the error in hand. The HTTP half of the download is DownloadStream's, and
	// its errors carry one.
	out, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", file, err)
	}
	defer func() { _ = out.Close() }()

	// Returned unchanged: it already carries the id DownloadStream generated
	// for it, and re-wrapping would mint a second one for one failure.
	resp, err := DownloadStream(url, opts...)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Close() }()

	_, err = io.Copy(out, resp)
	if err != nil {
		return fmt.Errorf("error saving file %s: %w", file, err)
	}

	return nil
}

// DownloadStream downloads the content from the specified URL and returns it as an io.ReadCloser.
// The caller is responsible for closing the returned stream.
//
// Parameters:
//   - url: The URL from which to download the content.
//   - opts: Optional functional options to configure the download (e.g., WithHeaders, WithTimeout).
//
// Returns:
//   - io.ReadCloser: The response body as a stream.
//   - error: An error object if an error occurs, otherwise nil.
func DownloadStream(url string, opts ...DownloadOption) (io.ReadCloser, error) {
	config := &downloadConfig{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(config)
	}

	// No error header is read anywhere in this function, and that is not an
	// omission. The URL here is a third party's — a feed, a MaxMind archive, a
	// vendor's CDN — not a ThreatWinds service, so there is no x-error-id to
	// adopt: catcher.GinError is what emits that header, and nothing outside
	// this org runs it. Reading it anyway would mean adopting a stranger's
	// value as this org's occurrence id on the strength of the header name
	// alone. Every error below therefore carries a locally generated id
	// instead, which is honest about where the id came from and still gives an
	// operator one value to grep the failure by.
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, withGeneratedErrorID(fmt.Errorf("error creating request: %w", err))
	}

	for k, v := range config.headers {
		req.Header.Add(k, v)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: config.skipTlsVerification,
		},
	}

	client := &http.Client{
		Timeout:   config.timeout,
		Transport: tr,
	}
	defer tr.CloseIdleConnections()

	resp, err := client.Do(req)
	if err != nil {
		return nil, withGeneratedErrorID(fmt.Errorf("error downloading from %s: %w", url, err))
	}

	// Deliberately not routed through SdkErrorFromResponse. This predicate is
	// not "the response is an error", it is "the response is not the exact
	// success this function requires": 204, 206 and every 3xx land here too.
	// Feeding those to SdkErrorFromResponse would stamp Severity INFO/NOTICE
	// on a failure and put a 2xx/3xx into Args["status"], which
	// catcher.GinError then uses as the HTTP status — a relaying handler would
	// answer 206 with an error envelope. Tightening the predicate to >= 400 is
	// what would make the two the same shape, and that is a behavioural change
	// to who gets a file written (Download would start accepting a 3xx body),
	// not a refactor.
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, withGeneratedErrorID(fmt.Errorf("bad status downloading from %s: %s", url, resp.Status))
	}

	return resp.Body, nil
}
