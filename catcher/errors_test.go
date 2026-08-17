package catcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it. Modeled on the same os.Pipe technique used by
// log_test.go and exit_log_test.go. It only produces deterministic results
// for synchronous writes — ERROR severity and above (see printLog in log.go)
// — since lower severities are handed to an async channel drained by a
// goroutine that may flush after os.Stdout has already been restored.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = original

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// lastJSONLine parses the last non-empty line of output as JSON. Every
// assertion in this file that needs log content produces exactly one line
// per captured call, but taking the last line (rather than assuming there is
// only one) matches the defensive pattern already used in log_test.go.
//
// beauty (package-level, toggled by other tests via Configure and never
// reset) may prefix the line with a severity icon and a space ahead of the
// JSON object; the prefix is stripped by locating the opening brace rather
// than by asserting beauty's current value, so this helper works regardless
// of what earlier tests left it set to.
func lastJSONLine(t *testing.T, output string) map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	last := lines[len(lines)-1]

	if i := strings.IndexByte(last, '{'); i > 0 {
		last = last[i:]
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(last), &parsed); err != nil {
		t.Fatalf("expected a JSON log line, got %q: %v", last, err)
	}
	return parsed
}

func TestTrace(t *testing.T) {
	t.Run("test error", func(t *testing.T) {
		err := Error("any error", nil, nil)
		if err == nil {
			t.Errorf("should return error")
			return
		}

		// Verify new fields are set
		if err.Timestamp == "" {
			t.Error("expected timestamp to be set")
		}

		if err.Severity == "" {
			t.Error("expected severity to be set")
		}

		if err.Severity != "ERROR" {
			t.Errorf("expected default severity 'ERROR', got %s", err.Severity)
		}
	})

	t.Run("test error with arg", func(t *testing.T) {
		err := Error("any error with arg", errors.New("and cause"), map[string]any{"argument": "value"})
		if err == nil {
			t.Errorf("should return error")
			return
		}

		// Verify new fields are set
		if err.Timestamp == "" {
			t.Error("expected timestamp to be set")
		}

		if err.Severity != "ERROR" {
			t.Errorf("expected default severity 'ERROR', got %s", err.Severity)
		}
	})

	t.Run("cast from error", func(t *testing.T) {
		var err error
		err = Error("any error with arg", errors.New("and cause"), map[string]any{"argument": "value"})

		e := Error("casting error", err, nil)
		if e == nil {
			t.Error("expected an SdkError")
			return
		}
		if e.Msg != "any error with arg" {
			t.Error("expected an SdkError")
			return
		}
	})

	t.Run("new error", func(t *testing.T) {
		err := errors.New("any error")
		e := Error("error from Go error", err, nil)
		if e == nil {
			t.Error("expected an SdkError")
			return
		}

		if e.Msg != "error from Go error" {
			t.Error("expected an SdkError")
			return
		}

		if *e.Cause != "any error" {
			t.Error("expected an SdkError")
			return
		}

		// Verify new fields
		if e.Timestamp == "" {
			t.Error("expected timestamp to be set")
		}

		if e.Severity != "ERROR" {
			t.Errorf("expected default severity 'ERROR', got %s", e.Severity)
		}
	})

	t.Run("severity calculation", func(t *testing.T) {
		tests := []struct {
			status   int
			expected string
		}{
			{200, "INFO"},
			{400, "WARNING"},
			{401, "WARNING"},
			{500, "ERROR"},
			{503, "CRITICAL"},
			{510, "ALERT"},
		}

		for _, test := range tests {
			err := Error("test message", nil, map[string]any{"status": test.status})
			if err.Severity != test.expected {
				t.Errorf("status %d: expected severity %s, got %s", test.status, test.expected, err.Severity)
			}
		}
	})
}

func TestCalculateSeverity(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"100 range", 100, "DEBUG"},
		{"200 range", 200, "INFO"},
		{"300 range", 301, "NOTICE"},
		{"400 range", 404, "WARNING"},
		{"500 error", 500, "ERROR"},
		{"501 error", 501, "ERROR"},
		{"502 critical", 502, "CRITICAL"},
		{"503 critical", 503, "CRITICAL"},
		{"510 alert", 510, "ALERT"},
		{"600 default", 600, "ERROR"},
		{"string input", "400", "WARNING"},
		{"float input", 400.0, "WARNING"},
		{"invalid input", nil, "ERROR"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := calculateSeverity(test.input)
			if result != test.expected {
				t.Errorf("calculateSeverity(%v) = %s, expected %s", test.input, result, test.expected)
			}
		})
	}
}

func TestCastInt(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{"int", 42, 42},
		{"int64", int64(42), 42},
		{"float64", 42.5, 42},
		{"string valid", "42", 42},
		{"string invalid", "abc", 500},
		{"nil", nil, 500},
		{"unknown type", struct{}{}, 500},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := castInt(test.input)
			if result != test.expected {
				t.Errorf("castInt(%v) = %d, expected %d", test.input, result, test.expected)
			}
		})
	}
}

func TestSdkErrorTimestamp(t *testing.T) {
	t.Run("timestamp format", func(t *testing.T) {
		err := Error("test message", nil, nil)
		// Verify timestamp is in RFC3339Nano format
		_, parseErr := time.Parse(time.RFC3339Nano, err.Timestamp)
		if parseErr != nil {
			t.Errorf("timestamp should be in RFC3339Nano format: %v", parseErr)
		}
	})

	t.Run("timestamp uniqueness", func(t *testing.T) {
		err1 := Error("test message 1", nil, nil)
		time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamps
		err2 := Error("test message 2", nil, nil)

		if err1.Timestamp == err2.Timestamp {
			t.Error("different errors should have different timestamps")
		}
	})
}

func TestGinErrorResponseBody(t *testing.T) {
	t.Run("json error body with status arg", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		err := Error("validation failed", nil, map[string]any{"status": 400})
		err.GinError(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		if w.Header().Get("x-error-id") != err.Code {
			t.Errorf("expected x-error-id header to be %s, got %s", err.Code, w.Header().Get("x-error-id"))
		}

		if w.Header().Get("x-error") == "" {
			t.Error("expected x-error header to be set")
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json; charset=utf-8" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		errorObj, ok := resp["error"].(map[string]any)
		if !ok {
			t.Fatal("expected 'error' key in response body")
		}

		if errorObj["message"] != err.SecureString() {
			t.Errorf("expected error message %s, got %v", err.SecureString(), errorObj["message"])
		}

		if errorObj["type"] != err.Severity {
			t.Errorf("expected error type %s, got %v", err.Severity, errorObj["type"])
		}

		if errorObj["code"] != err.Code {
			t.Errorf("expected error code %s, got %v", err.Code, errorObj["code"])
		}
	})

	t.Run("json error body with default 500 status", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		err := Error("server error", nil, nil)
		err.GinError(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}

		if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
			t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
		}

		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		errorObj, ok := resp["error"].(map[string]any)
		if !ok {
			t.Fatal("expected 'error' key in response body")
		}

		if errorObj["type"] != "ERROR" {
			t.Errorf("expected error type ERROR, got %v", errorObj["type"])
		}
	})
}

func TestGinErrorRetryAfter(t *testing.T) {
	t.Run("retry header set from args", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		err := Error("model warming up", nil, map[string]any{
			"status":        503,
			"retry":         600,
			"code_override": "model_warming_up",
			"param":         "silas-1.6",
		})
		err.GinError(c)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}

		retryAfter := w.Header().Get("Retry-After")
		if retryAfter != "600" {
			t.Errorf("expected Retry-After header '600', got %q", retryAfter)
		}

		var resp map[string]any
		if jsonErr := json.Unmarshal(w.Body.Bytes(), &resp); jsonErr != nil {
			t.Fatalf("failed to unmarshal response body: %v", jsonErr)
		}

		errorObj, ok := resp["error"].(map[string]any)
		if !ok {
			t.Fatal("expected 'error' key in response body")
		}

		if errorObj["code"] != "model_warming_up" {
			t.Errorf("expected code_override 'model_warming_up', got %v", errorObj["code"])
		}

		if errorObj["param"] != "silas-1.6" {
			t.Errorf("expected param 'silas-1.6', got %v", errorObj["param"])
		}
	})

	t.Run("no retry header when not configured", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		err := Error("server error", nil, map[string]any{"status": 500})
		err.GinError(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}

		retryAfter := w.Header().Get("Retry-After")
		if retryAfter != "" {
			t.Errorf("expected no Retry-After header, got %q", retryAfter)
		}
	})

	t.Run("retry zero does not set header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		err := Error("warm", nil, map[string]any{
			"status": 503,
			"retry":  0,
		})
		err.GinError(c)

		// 0 is still a valid int, so "0" should appear — but callers using 0
		// signal "do not retry", so we treat it as unset.
		retryAfter := w.Header().Get("Retry-After")
		if retryAfter != "" {
			t.Errorf("expected no Retry-After when retry=0, got %q", retryAfter)
		}
	})
}

// TestErrorConstructPath covers the branch of Error() that builds a brand
// new *SdkError: a nil cause, a plain (non-SdkError) cause, and — the
// regression case for the "typed-nil pointer" crash — a cause whose static
// type is *SdkError but whose value is nil, boxed into a non-nil `error`
// interface. All three must construct a fresh error, log exactly one line,
// and must never panic.
func TestErrorConstructPath(t *testing.T) {
	var typedNilCause *SdkError // nil *SdkError, boxed into `error` below

	tests := []struct {
		name          string
		cause         error
		expectedCause string
	}{
		{
			name:          "nil cause",
			cause:         nil,
			expectedCause: "unknown cause",
		},
		{
			name:          "plain error cause",
			cause:         errors.New("boom"),
			expectedCause: "boom",
		},
		{
			name:          "typed-nil *SdkError cause",
			cause:         typedNilCause,
			expectedCause: "unknown cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got *SdkError
			var recovered any

			output := captureStdout(t, func() {
				defer func() { recovered = recover() }()
				got = Error("construct path: "+tt.name, tt.cause, nil)
			})

			if recovered != nil {
				t.Fatalf("Error() panicked: %v", recovered)
			}
			if got == nil {
				t.Fatal("expected a constructed *SdkError, got nil")
			}
			if got.Cause == nil || *got.Cause != tt.expectedCause {
				t.Errorf("expected Cause %q, got %v", tt.expectedCause, got.Cause)
			}
			if got.Severity != "ERROR" {
				t.Errorf("expected default severity ERROR, got %s", got.Severity)
			}

			line := lastJSONLine(t, output)
			if line["msg"] != got.Msg {
				t.Errorf("expected logged msg %q, got %v", got.Msg, line["msg"])
			}
			if line["cause"] != tt.expectedCause {
				t.Errorf("expected logged cause %q, got %v", tt.expectedCause, line["cause"])
			}
			if line["code"] != got.Code {
				t.Errorf("expected logged code %q, got %v", got.Code, line["code"])
			}
		})
	}
}

// TestErrorShortCircuitPreservesIdentityAndDoesNotLog covers the branch
// where cause is already an *SdkError. This package logs at construction,
// not at handling: once the original *SdkError was built, its line was
// already written, and a later wrap can never reach back and edit that
// line — the only way to "add" to it would be a second line, which is
// exactly the noise this contract avoids. So the short-circuit returns
// cause unchanged and logs nothing: msg and args at this call site are
// silently ignored (see the doc comment on Error for the correct way to
// log context about an existing error).
func TestErrorShortCircuitPreservesIdentityAndDoesNotLog(t *testing.T) {
	original := Error("original failure", errors.New("root cause"), map[string]any{"status": 503})
	if original == nil {
		t.Fatal("setup: expected a constructed original error")
	}
	if original.Severity != "CRITICAL" {
		t.Fatalf("setup: expected original severity CRITICAL, got %s", original.Severity)
	}

	var annotated *SdkError
	output := captureStdout(t, func() {
		annotated = Error("layer msg", original, map[string]any{"extra": "value"})
	})

	// Identity: same pointer, not a copy or a new error.
	if annotated != original {
		t.Fatalf("expected Error() to return the same *SdkError pointer, got a different one")
	}
	// The returned value's own fields must be untouched by this call.
	if annotated.Msg != "original failure" {
		t.Errorf("returned error's Msg must not change, got %q", annotated.Msg)
	}
	if annotated.Code != original.Code {
		t.Errorf("returned error's Code must not change")
	}
	if annotated.Severity != "CRITICAL" {
		t.Errorf("returned error's Severity must not change, got %s", annotated.Severity)
	}
	if len(annotated.Args) != 1 || annotated.Args["status"] != 503 {
		t.Errorf("returned error's Args must not change, got %v", annotated.Args)
	}

	// This call's msg/args must be silently ignored, including not logging
	// anything — no line, annotation or otherwise, is emitted here.
	if got := strings.TrimSpace(output); got != "" {
		t.Errorf("expected no log output for a short-circuited *SdkError cause, got %q", got)
	}
}

// TestErrorNLayersNeverLog pins the N-layers case: re-raising an
// already-*SdkError through three call sites must not produce any log
// output at any layer, and every call must still return the original
// pointer unchanged. This is what stops the discard fix from being
// re-implemented as per-layer annotation logging later without someone
// deliberately deciding to pay for it again — the earlier version of this
// fix did exactly that, and it was reverted because N re-raises produced N
// log lines for a single failure, which is real, billable noise across 18
// services.
func TestErrorNLayersNeverLog(t *testing.T) {
	original := Error("layer 0", errors.New("root cause"), nil)

	var layer1, layer2, layer3 *SdkError
	output := captureStdout(t, func() {
		layer1 = Error("layer 1", original, nil)
		layer2 = Error("layer 2", layer1, nil)
		layer3 = Error("layer 3", layer2, nil)
	})

	for i, e := range []*SdkError{layer1, layer2, layer3} {
		if e != original {
			t.Fatalf("layer %d: expected the original pointer to be returned unchanged", i+1)
		}
	}

	if got := strings.TrimSpace(output); got != "" {
		t.Errorf("expected zero log lines across all 3 re-raises, got %q", got)
	}
}

// TestErrorStatusArgSeverityConstructPath covers args["status"] in the
// branch of Error() that builds a brand new *SdkError — including with a
// typed-nil *SdkError cause, to confirm the crash fix in ToSdkError doesn't
// also disturb normal severity calculation on that path.
func TestErrorStatusArgSeverityConstructPath(t *testing.T) {
	var typedNilCause *SdkError

	tests := []struct {
		name     string
		cause    error
		status   int
		expected string
	}{
		{"nil cause", nil, 503, "CRITICAL"},
		{"plain error cause", errors.New("boom"), 400, "WARNING"},
		{"typed-nil *SdkError cause", typedNilCause, 510, "ALERT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Error("status test", tt.cause, map[string]any{"status": tt.status})
			if got.Severity != tt.expected {
				t.Errorf("expected severity %s, got %s", tt.expected, got.Severity)
			}
		})
	}
}

// TestErrorStatusArgSeverityAnnotatePath covers args["status"] in the
// short-circuit branch, where it is ignored entirely along with msg and the
// rest of args: the returned pointer's Severity stays pinned to the
// original error's severity, and nothing is logged.
func TestErrorStatusArgSeverityAnnotatePath(t *testing.T) {
	original := Error("original", nil, map[string]any{"status": 503}) // CRITICAL
	if original.Severity != "CRITICAL" {
		t.Fatalf("setup: expected CRITICAL, got %s", original.Severity)
	}

	var annotated *SdkError
	output := captureStdout(t, func() {
		// 400 would compute to WARNING if it were applied here; it must not be.
		annotated = Error("layer", original, map[string]any{"status": 400})
	})

	if annotated.Severity != "CRITICAL" {
		t.Errorf("status arg must not change the returned error's severity, got %s", annotated.Severity)
	}

	if got := strings.TrimSpace(output); got != "" {
		t.Errorf("expected no log output when short-circuiting on an existing *SdkError, got %q", got)
	}
}

// TestToSdkErrorDirect exercises ToSdkError in isolation, including the
// regression cases for the typed-nil-pointer defect: returning a typed-nil
// pointer instead of nil, and panicking on an *SdkError wrapped via
// fmt.Errorf("%w", ...) because it re-asserted on the outer wrapper instead
// of using the pointer errors.As already unwrapped.
func TestToSdkErrorDirect(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if got := ToSdkError(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("plain error", func(t *testing.T) {
		if got := ToSdkError(errors.New("boom")); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("*SdkError", func(t *testing.T) {
		sdkErr := Error("direct sdk error", nil, nil)
		got := ToSdkError(sdkErr)
		if got != sdkErr {
			t.Errorf("expected the same pointer back, got %v", got)
		}
	})

	t.Run("typed-nil *SdkError", func(t *testing.T) {
		var nilSdk *SdkError
		var recovered any
		var got *SdkError

		func() {
			defer func() { recovered = recover() }()
			got = ToSdkError(nilSdk)
		}()

		if recovered != nil {
			t.Fatalf("ToSdkError panicked: %v", recovered)
		}
		if got != nil {
			t.Errorf("expected nil for a typed-nil *SdkError, got %v", got)
		}
	})

	t.Run("*SdkError wrapped via fmt.Errorf", func(t *testing.T) {
		sdkErr := Error("wrapped sdk error", nil, nil)
		wrapped := fmt.Errorf("context: %w", sdkErr)

		got := ToSdkError(wrapped)
		if got != sdkErr {
			t.Errorf("expected ToSdkError to unwrap to the original pointer, got %v", got)
		}
	})
}

// TestErrorsIsFindsSentinelThroughSdkError covers the primary motivation for
// implementing Unwrap: a sentinel error wrapped by catcher.Error must still
// be discoverable with errors.Is through the *SdkError layer.
func TestErrorsIsFindsSentinelThroughSdkError(t *testing.T) {
	sentinel := errors.New("sentinel: not found")

	wrapped := Error("lookup failed", sentinel, nil)

	if !errors.Is(wrapped, sentinel) {
		t.Error("expected errors.Is to find the sentinel beneath the SdkError")
	}
}

// customError is a concrete error type distinct from errors.errorString,
// used to exercise errors.As beneath an SdkError.
type customError struct {
	code int
}

func (e *customError) Error() string {
	return fmt.Sprintf("custom error %d", e.code)
}

// TestErrorsAsFindsConcreteTypeThroughSdkError covers errors.As reaching a
// concrete custom error type beneath an SdkError via the new Unwrap method.
func TestErrorsAsFindsConcreteTypeThroughSdkError(t *testing.T) {
	original := &customError{code: 42}

	wrapped := Error("operation failed", original, nil)

	var target *customError
	if !errors.As(wrapped, &target) {
		t.Fatal("expected errors.As to find the customError beneath the SdkError")
	}
	if target.code != 42 {
		t.Errorf("expected code 42, got %d", target.code)
	}
}

// TestUnwrapNilCases covers Unwrap's two "nothing to traverse into" cases:
// a zero-value SdkError that was never passed through catcher.Error, and
// one round-tripped through JSON (whose unexported cause field can never be
// populated by json.Unmarshal in the first place).
func TestUnwrapNilCases(t *testing.T) {
	t.Run("zero-value SdkError", func(t *testing.T) {
		e := &SdkError{}
		if got := e.Unwrap(); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("round-tripped through JSON", func(t *testing.T) {
		original := Error("round trip", errors.New("root cause"), map[string]any{"status": 500})

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded SdkError
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got := decoded.Unwrap(); got != nil {
			t.Errorf("expected nil after JSON round-trip, got %v", got)
		}
	})

	t.Run("nil *SdkError receiver", func(t *testing.T) {
		var e *SdkError
		if got := e.Unwrap(); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

// TestErrorDoesNotPanicOnNilCause covers the latent bug fixed alongside
// Unwrap: Error() used to dereference e.Cause unconditionally, so a
// zero-value SdkError, or one decoded from JSON without a "cause" key,
// panicked on .Error().
func TestErrorDoesNotPanicOnNilCause(t *testing.T) {
	t.Run("zero-value SdkError", func(t *testing.T) {
		e := SdkError{}
		var msg string
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf(".Error() panicked: %v", r)
				}
			}()
			msg = e.Error()
		}()
		if msg == "" {
			t.Error("expected a non-empty error message")
		}
	})

	t.Run("Cause explicitly nil", func(t *testing.T) {
		e := SdkError{Msg: "something broke", Cause: nil}
		var msg string
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf(".Error() panicked: %v", r)
				}
			}()
			msg = e.Error()
		}()
		if msg == "" {
			t.Error("expected a non-empty error message")
		}
	})
}

// TestTwoLevelChainShortCircuitAndUnwrap is the two-level chain requested
// explicitly: catcher.Error(msg, catcher.Error(msg2, sentinel, nil), nil).
//
// The inner call constructs a fresh *SdkError whose unexported cause is the
// sentinel. The outer call passes that *SdkError as cause: ToSdkError finds
// it immediately (errors.As matches *SdkError at the very first node it
// examines, the inner error itself, because assignability is checked before
// Unwrap is ever consulted — see the reasoning below), so Error()
// short-circuits and returns the inner pointer unchanged, exactly as it did
// before this change. No second SdkError layer is ever built.
//
// errors.Is(outer, sentinel) still succeeds, but not because errors.As/Is
// traversed "through" one SdkError to a nested one beneath it — there is no
// nested SdkError-under-SdkError to traverse through, by construction (see
// TestErrorConstructPath and the comment on the construct branch: this
// package never sets the unexported cause field to another *SdkError,
// because that branch is only reached when ToSdkError already proved the
// cause does not resolve to one). outer *is* inner, and inner's own cause
// (set at its own construction) is the sentinel. Unwrap exposes exactly one
// level, and outer == inner means that one level is all errors.Is needs.
func TestTwoLevelChainShortCircuitAndUnwrap(t *testing.T) {
	sentinel := errors.New("sentinel: deep cause")

	var inner *SdkError
	output := captureStdout(t, func() {
		inner = Error("inner failure", sentinel, nil)
	})
	if strings.TrimSpace(output) == "" {
		t.Fatal("setup: expected the inner construct to log")
	}

	var outer *SdkError
	output = captureStdout(t, func() {
		outer = Error("outer failure", inner, nil)
	})

	// The short-circuit contract (pre-existing, unaffected by Unwrap):
	// identity preserved, nothing logged for the outer call.
	if outer != inner {
		t.Fatal("expected the short-circuit to return the inner pointer unchanged")
	}
	if strings.TrimSpace(output) != "" {
		t.Errorf("expected no log output for the short-circuited outer call, got %q", output)
	}

	// The new behavior under test: errors.Is reaches the sentinel.
	if !errors.Is(outer, sentinel) {
		t.Error("expected errors.Is(outer, sentinel) to succeed")
	}

	// Confirm there is exactly one level to unwrap, not two: unwrapping
	// once from outer reaches the sentinel directly.
	if got := errors.Unwrap(outer); got != sentinel {
		t.Errorf("expected a single Unwrap to reach the sentinel directly, got %v", got)
	}
}

// TestErrorAlreadySdkErrorStillShortCircuits pins the existing contract
// (unchanged by adding Unwrap): passing an *SdkError as cause returns the
// same pointer and logs nothing, regardless of how deep an unwrap chain
// that *SdkError itself carries.
func TestErrorAlreadySdkErrorStillShortCircuits(t *testing.T) {
	original := Error("pinned original", errors.New("root cause"), map[string]any{"status": 418})

	var got *SdkError
	output := captureStdout(t, func() {
		got = Error("layer msg", original, map[string]any{"extra": "value"})
	})

	if got != original {
		t.Fatal("expected the same *SdkError pointer back")
	}
	if strings.TrimSpace(output) != "" {
		t.Errorf("expected no log output, got %q", output)
	}
}
