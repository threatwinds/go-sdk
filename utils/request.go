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
	"time"

	"github.com/threatwinds/go-sdk/catcher"
)

// remoteErrorMessage is the message carried by every error SdkErrorFromResponse
// builds. It is deliberately constant and carries no upstream detail:
// catcher.SecureString returns Msg verbatim — and only Msg — for a status >=
// 500, so anything folded into it crosses the next service boundary
// unfiltered. The remote's own text goes to Cause, its status to
// Args["status"], and its occurrence id to ErrorID.
const remoteErrorMessage = "remote service error"

// SdkErrorFromResponse turns a failed HTTP response into a *catcher.SdkError,
// reading only the status line and the error headers catcher.GinError sets on
// the way out of a ThreatWinds service:
//
//	x-error-id → ErrorID (and Args["error_id"]), adopted verbatim so one
//	             failure keeps a single id across every service that touches
//	             it. Absent (a non-ThreatWinds endpoint, or an error from an
//	             intermediary) → catcher mints a fresh UUID, exactly as it
//	             would for any locally-raised error.
//	x-error    → Cause, the remote's own rendered error text. Absent →
//	             "unknown cause", the same text catcher itself uses.
//	status     → Args["status"] (an int, as everywhere else in the SDK) and
//	             Severity, via calcSeverityFromStatus.
//
// It never touches resp.Body: the body may already have been consumed, may be
// nil on a hand-built response, and is the caller's to read and to close.
//
// The error is built with catcher.New, not catcher.Error, so nothing is logged
// inside a transport helper — the error is logged once, wherever the caller
// ends up handling it (or automatically, if it reaches catcher.GinError).
//
// Code is *not* the upstream's id. Code identifies the error's type (an md5 of
// the message, per catcher.SdkError's field doc) and is therefore the same for
// every error this function builds; the per-occurrence, greppable id is
// ErrorID. Conflating the two is precisely the drift catcher.GinError was
// fixed to remove.
//
// A nil response, a nil Header map and absent headers are all handled: a nil
// response is reported as a 500, since "no response at all" is a local
// failure and 500 is the status catcher would otherwise default to.
func SdkErrorFromResponse(resp *http.Response) *catcher.SdkError {
	return sdkErrorFromResponse(resp, remoteErrorMessage)
}

// sdkErrorFromResponse is the single place an HTTP response is translated into
// an *SdkError. The message is a parameter only so DoReq can keep its
// long-standing text (which embeds the response body it has already read);
// every other field — status, severity, cause, adopted error id — is derived
// here, so the two call sites cannot drift apart.
func sdkErrorFromResponse(resp *http.Response, msg string) *catcher.SdkError {
	status := http.StatusInternalServerError

	var header http.Header
	if resp != nil {
		status = resp.StatusCode
		header = resp.Header
	}

	args := map[string]any{"status": status}
	if errorID := headerValue(header, "x-error-id"); errorID != "" {
		args["error_id"] = errorID
	}

	var cause error
	if causeText := headerValue(header, "x-error"); causeText != "" {
		cause = errors.New(causeText)
	}

	err := catcher.New(msg, cause, args)

	// The trace catcher captures here is this process's stack — DoReq and its
	// caller — which describes where the response was *received*, not where
	// the failure happened. The frames that explain the failure are in the
	// upstream service's own log line, findable by ErrorID. Clearing it also
	// makes the result deterministic rather than dependent on CATCHER_NO_TRACE
	// (which already omits traces by default).
	err.Trace = nil

	// Severity is stated here rather than left to catcher's derivation from
	// args["status"]: the two ladders agree today (calcSeverityFromStatus
	// mirrors catcher.calculateSeverity, which is unexported and so cannot be
	// called from here), and stating it keeps this helper's contract —
	// severity follows the HTTP status of the response — true regardless of
	// how catcher chooses to read its args.
	err.Severity = calcSeverityFromStatus(status)

	return err
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
//     processing, otherwise nil.
func DoReq[response any](url string, data []byte, method string, headers map[string]string, skipTlsVerification bool) (response, int, error) {
	var result response

	if len(data) > maxMessageSize {
		return result, http.StatusBadRequest, fmt.Errorf("cannot convert to object: data size exceeds limit (size=%d bytes, limit=%d bytes)", len(data), maxMessageSize)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("error creating request: %w", err)
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
		return result, http.StatusInternalServerError, fmt.Errorf("error doing request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		// One extraction path: SdkErrorFromResponse adopts the upstream's
		// x-error-id as ErrorID, its x-error as Cause, and derives status and
		// severity — see its doc comment. Only the message is supplied here,
		// byte-for-byte the text this call site has always produced, because
		// it embeds the body DoReq has already read (SdkErrorFromResponse
		// never reads a body itself).
		//
		// catcher.New under the hood, so DoReq never logs: this error is
		// logged, once, wherever the caller ends up handling it.
		return result, resp.StatusCode, sdkErrorFromResponse(resp,
			fmt.Sprintf("error response (status=%d): %s", resp.StatusCode, string(body)))
	}

	if resp.StatusCode == http.StatusNoContent {
		return result, resp.StatusCode, nil
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, resp.StatusCode, fmt.Errorf("error parsing response: %w", err)
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

	out, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", file, err)
	}
	defer func() { _ = out.Close() }()

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

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
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
		return nil, fmt.Errorf("error downloading from %s: %w", url, err)
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
		return nil, fmt.Errorf("bad status downloading from %s: %s", url, resp.Status)
	}

	return resp.Body, nil
}
