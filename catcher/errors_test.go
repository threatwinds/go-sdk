package catcher

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

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
