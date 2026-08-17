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
