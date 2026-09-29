package catcher

import (
	"encoding/json"
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
		// The trace assertion below is the only coverage that Log's
		// trace-collection branch ever populates the field, and tracing is
		// off by default — so this test has to ask for it.
		withTrace(t)

		output := captureStdout(t, func() {
			Info("test info message", map[string]any{
				"service": "test-service",
				"version": "1.0.0",
			})
		})

		// Should contain our message
		if !strings.Contains(output, "test info message") {
			t.Errorf("Log output should contain the message: %s", output)
		}

		parsed := lastJSONLine(t, output)

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
		output := captureStdout(t, func() {
			Info("test message", nil)
		})

		if !strings.Contains(output, "test message") {
			t.Errorf("Should handle nil args gracefully: %s", output)
		}
	})

	t.Run("Info with empty args", func(t *testing.T) {
		output := captureStdout(t, func() {
			Info("test message", map[string]any{})
		})

		if !strings.Contains(output, "test message") {
			t.Errorf("Should handle empty args gracefully: %s", output)
		}
	})

	t.Run("Info code generation", func(t *testing.T) {
		firstOutput := captureStdout(t, func() {
			Info("identical message", map[string]any{"test": 1})
		})

		secondOutput := captureStdout(t, func() {
			Info("identical message", map[string]any{"test": 2})
		})

		// Extract codes from both outputs
		extractCode := func(output string) string {
			if code, ok := lastJSONLine(t, output)["code"].(string); ok {
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
		complexArgs := map[string]any{
			"string":  "value",
			"number":  42,
			"boolean": true,
			"array":   []string{"a", "b", "c"},
			"nested": map[string]any{
				"inner": "value",
			},
		}

		output := captureStdout(t, func() {
			Info("complex message", complexArgs)
		})

		// Should contain the message
		if !strings.Contains(output, "complex message") {
			t.Errorf("Should contain the message: %s", output)
		}

		// Should be valid JSON, and the args should survive the round trip
		// with their structure intact.
		args, ok := lastJSONLine(t, output)["args"].(map[string]any)
		if !ok {
			t.Fatal("Expected args to be a map")
		}
		nested, ok := args["nested"].(map[string]any)
		if !ok {
			t.Fatalf("Expected nested args to be a map, got %v", args["nested"])
		}
		if nested["inner"] != "value" {
			t.Errorf("Expected nested inner 'value', got %v", nested["inner"])
		}
	})
}

func TestInfoVsError(t *testing.T) {
	t.Run("Info vs Error output differences", func(t *testing.T) {
		// Both branches are asserted to carry a trace, which is not the
		// default configuration.
		withTrace(t)

		infoOutput := captureStdout(t, func() {
			Info("info message", map[string]any{"type": "info"})
		})

		errorOutput := captureStdout(t, func() {
			Error("error message", nil, map[string]any{"type": "error"})
		})

		// Both should contain their respective messages
		if !strings.Contains(infoOutput, "info message") {
			t.Error("Info output should contain info message")
		}
		if !strings.Contains(errorOutput, "error message") {
			t.Error("Error output should contain error message")
		}

		infoJSON := lastJSONLine(t, infoOutput)
		errorJSON := lastJSONLine(t, errorOutput)

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
	// Discard the output, under the package defaults rather than whatever
	// configuration a previously selected benchmark happened to leave behind.
	benchToDevNull(b, true, true, true)

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
		output := captureStdout(t, func() {
			Info("timestamp test", map[string]any{"test": true})
		})

		parsed := lastJSONLine(t, output)

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
