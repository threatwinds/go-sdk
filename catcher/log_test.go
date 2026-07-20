package catcher

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSdkLog(t *testing.T) {
	t.Run("SdkLog String method", func(t *testing.T) {
		timestamp := "2023-01-01T12:00:00.000000000Z"
		sdkLog := SdkLog{
			Timestamp: timestamp,
			Code:      "test123",
			Trace:     []string{"func1 10", "func2 20"},
			Msg:       "test message",
			Args:      map[string]any{"key": "value"},
			Severity:  "INFO",
		}

		result := sdkLog.JSON()

		// Should be valid JSON
		var parsed map[string]any
		err := json.Unmarshal([]byte(result), &parsed)
		if err != nil {
			t.Errorf("String() should return valid JSON: %v", err)
		}

		// Check required fields
		if parsed["code"] != "test123" {
			t.Errorf("Expected code 'test123', got %v", parsed["code"])
		}
		if parsed["msg"] != "test message" {
			t.Errorf("Expected msg 'test message', got %v", parsed["msg"])
		}
		if parsed["timestamp"] != timestamp {
			t.Errorf("Expected timestamp '%s', got %v", timestamp, parsed["timestamp"])
		}
		if parsed["severity"] != "INFO" {
			t.Errorf("Expected severity 'INFO', got %v", parsed["severity"])
		}
	})
}

func TestInfo(t *testing.T) {
	t.Run("Info function basic logging", func(t *testing.T) {
		// Capture stdout output since Info uses fmt.Println
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		Info("test info message", map[string]any{
			"service": "test-service",
			"version": "1.0.0",
		})

		w.Close()
		os.Stdout = originalStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Should contain our message
		if !strings.Contains(output, "test info message") {
			t.Errorf("Log output should contain the message: %s", output)
		}

		// Should be valid JSON (entire line is JSON now)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		lastLine := lines[len(lines)-1]

		// The entire line should be JSON now
		jsonPart := strings.TrimSpace(lastLine)
		var parsed map[string]any
		err := json.Unmarshal([]byte(jsonPart), &parsed)
		if err != nil {
			t.Errorf("Log output should be valid JSON: %v, got: %s", err, jsonPart)
		}

		// Check structure
		if parsed["msg"] != "test info message" {
			t.Errorf("Expected msg 'test info message', got %v", parsed["msg"])
		}

		if parsed["code"] == nil {
			t.Error("Expected code field to be present")
		}

		if parsed["trace"] == nil {
			t.Error("Expected trace field to be present")
		}

		if parsed["timestamp"] == nil {
			t.Error("Expected timestamp field to be present")
		}

		if parsed["severity"] != "INFO" {
			t.Errorf("Expected severity 'INFO', got %v", parsed["severity"])
		}

		// Check args
		args, ok := parsed["args"].(map[string]any)
		if !ok {
			t.Error("Expected args to be a map")
		} else {
			if args["service"] != "test-service" {
				t.Errorf("Expected service 'test-service', got %v", args["service"])
			}
		}
	})

	t.Run("Info with nil args", func(t *testing.T) {
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		Info("test message", nil)

		w.Close()
		os.Stdout = originalStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()
		if !strings.Contains(output, "test message") {
			t.Errorf("Should handle nil args gracefully: %s", output)
		}
	})

	t.Run("Info with empty args", func(t *testing.T) {
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		Info("test message", map[string]any{})

		w.Close()
		os.Stdout = originalStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()
		if !strings.Contains(output, "test message") {
			t.Errorf("Should handle empty args gracefully: %s", output)
		}
	})

	t.Run("Info code generation", func(t *testing.T) {
		// First call
		originalStdout := os.Stdout
		r1, w1, _ := os.Pipe()
		os.Stdout = w1

		Info("identical message", map[string]any{"test": 1})

		w1.Close()
		os.Stdout = originalStdout

		var buf1 bytes.Buffer
		buf1.ReadFrom(r1)
		firstOutput := buf1.String()

		// Second call
		r2, w2, _ := os.Pipe()
		os.Stdout = w2

		Info("identical message", map[string]any{"test": 2})

		w2.Close()
		os.Stdout = originalStdout

		var buf2 bytes.Buffer
		buf2.ReadFrom(r2)
		secondOutput := buf2.String()

		// Extract codes from both outputs
		extractCode := func(output string) string {
			lines := strings.Split(strings.TrimSpace(output), "\n")
			lastLine := lines[len(lines)-1]
			jsonPart := strings.TrimSpace(lastLine)
			var parsed map[string]any
			json.Unmarshal([]byte(jsonPart), &parsed)
			if code, ok := parsed["code"].(string); ok {
				return code
			}
			return ""
		}

		firstCode := extractCode(firstOutput)
		secondCode := extractCode(secondOutput)

		if firstCode == "" || secondCode == "" {
			t.Error("Should generate valid codes")
		}

		if firstCode != secondCode {
			t.Errorf("Same message should generate same code: %s vs %s", firstCode, secondCode)
		}
	})

	t.Run("Info with complex args", func(t *testing.T) {
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		complexArgs := map[string]any{
			"string":  "value",
			"number":  42,
			"boolean": true,
			"array":   []string{"a", "b", "c"},
			"nested": map[string]any{
				"inner": "value",
			},
		}

		Info("complex message", complexArgs)

		w.Close()
		os.Stdout = originalStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Should contain the message
		if !strings.Contains(output, "complex message") {
			t.Errorf("Should contain the message: %s", output)
		}

		// Should be valid JSON
		lines := strings.Split(strings.TrimSpace(output), "\n")
		lastLine := lines[len(lines)-1]
		jsonPart := strings.TrimSpace(lastLine)
		var parsed map[string]any
		err := json.Unmarshal([]byte(jsonPart), &parsed)
		if err != nil {
			t.Errorf("Should handle complex args: %v", err)
		}
	})
}

func TestInfoVsError(t *testing.T) {
	t.Run("Info vs Error output differences", func(t *testing.T) {
		// Test Info
		originalStdout := os.Stdout
		r1, w1, _ := os.Pipe()
		os.Stdout = w1

		Info("info message", map[string]any{"type": "info"})

		w1.Close()
		os.Stdout = originalStdout

		var buf1 bytes.Buffer
		buf1.ReadFrom(r1)
		infoOutput := buf1.String()

		// Test Error (Error uses fmt.Println too)
		r2, w2, _ := os.Pipe()
		os.Stdout = w2

		Error("error message", nil, map[string]any{"type": "error"})

		w2.Close()
		os.Stdout = originalStdout

		var buf2 bytes.Buffer
		buf2.ReadFrom(r2)
		errorOutput := buf2.String()

		// Both should contain their respective messages
		if !strings.Contains(infoOutput, "info message") {
			t.Error("Info output should contain info message")
		}
		if !strings.Contains(errorOutput, "error message") {
			t.Error("Error output should contain error message")
		}

		// Extract JSON from both
		extractJSON := func(output string) map[string]any {
			lines := strings.Split(strings.TrimSpace(output), "\n")
			lastLine := lines[len(lines)-1]
			jsonPart := strings.TrimSpace(lastLine)
			var parsed map[string]any
			json.Unmarshal([]byte(jsonPart), &parsed)
			return parsed
		}

		infoJSON := extractJSON(infoOutput)
		errorJSON := extractJSON(errorOutput)

		// Info should not have "cause" field
		if infoJSON["cause"] != nil {
			t.Error("Info logs should not have cause field")
		}

		// Error might have "cause" field (in this case nil, but field exists)
		// Both should have required fields
		for _, j := range []map[string]any{infoJSON, errorJSON} {
			if j["code"] == nil {
				t.Error("Both should have code field")
			}
			if j["trace"] == nil {
				t.Error("Both should have trace field")
			}
			if j["msg"] == nil {
				t.Error("Both should have msg field")
			}
		}
	})
}

// Benchmark tests
func BenchmarkInfo(b *testing.B) {
	// Redirect stdout to discard
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", map[string]any{
			"iteration": i,
			"benchmark": true,
		})
	}
}

func BenchmarkSdkLogString(b *testing.B) {
	sdkLog := SdkLog{
		Timestamp: "2023-01-01T12:00:00.000000000Z",
		Code:      "benchmark123",
		Trace:     []string{"func1 10", "func2 20", "func3 30"},
		Msg:       "benchmark message",
		Args: map[string]any{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		},
		Severity: "INFO",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sdkLog.JSON()
	}
}

func TestInfoTimestampAndSeverity(t *testing.T) {
	t.Run("Info timestamp format", func(t *testing.T) {
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		Info("timestamp test", map[string]any{"test": true})

		w.Close()
		os.Stdout = originalStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		lastLine := lines[len(lines)-1]
		jsonPart := strings.TrimSpace(lastLine)
		var parsed map[string]any
		err := json.Unmarshal([]byte(jsonPart), &parsed)
		if err != nil {
			t.Errorf("Should parse JSON: %v", err)
			return
		}

		// Verify timestamp format
		if timestamp, ok := parsed["timestamp"].(string); ok {
			_, err := time.Parse(time.RFC3339Nano, timestamp)
			if err != nil {
				t.Errorf("Timestamp should be in RFC3339Nano format: %v", err)
			}
		} else {
			t.Error("Timestamp should be a string")
		}

		// Verify severity is always INFO for Info function
		if parsed["severity"] != "INFO" {
			t.Errorf("Info function should always have severity 'INFO', got %v", parsed["severity"])
		}
	})
}
