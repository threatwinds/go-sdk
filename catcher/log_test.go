package catcher

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"
	"testing"
)

func TestSdkLog(t *testing.T) {
	t.Run("SdkLog String method", func(t *testing.T) {
		sdkLog := SdkLog{
			Code:  "test123",
			Trace: []string{"func1 10", "func2 20"},
			Msg:   "test message",
			Args:  map[string]any{"key": "value"},
		}

		result := sdkLog.String()

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
	})
}

func TestInfo(t *testing.T) {
	t.Run("Info function basic logging", func(t *testing.T) {
		// Capture log output
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		Info("test info message", map[string]any{
			"service": "test-service",
			"version": "1.0.0",
		})

		output := buf.String()

		// Should contain our message
		if !strings.Contains(output, "test info message") {
			t.Errorf("Log output should contain the message: %s", output)
		}

		// Should be valid JSON (after timestamp prefix)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		lastLine := lines[len(lines)-1]

		// Extract JSON part (after timestamp)
		jsonStart := strings.Index(lastLine, "{")
		if jsonStart == -1 {
			t.Errorf("Log output should contain JSON: %s", lastLine)
			return
		}

		jsonPart := lastLine[jsonStart:]
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
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		Info("test message", nil)

		output := buf.String()
		if !strings.Contains(output, "test message") {
			t.Errorf("Should handle nil args gracefully: %s", output)
		}
	})

	t.Run("Info with empty args", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		Info("test message", map[string]any{})

		output := buf.String()
		if !strings.Contains(output, "test message") {
			t.Errorf("Should handle empty args gracefully: %s", output)
		}
	})

	t.Run("Info code generation", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		// Same message should generate same code
		Info("identical message", map[string]any{"test": 1})
		firstOutput := buf.String()

		buf.Reset()
		Info("identical message", map[string]any{"test": 2})
		secondOutput := buf.String()

		// Extract codes from both outputs
		extractCode := func(output string) string {
			lines := strings.Split(strings.TrimSpace(output), "\n")
			lastLine := lines[len(lines)-1]
			jsonStart := strings.Index(lastLine, "{")
			if jsonStart == -1 {
				return ""
			}
			jsonPart := lastLine[jsonStart:]
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
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

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

		output := buf.String()

		// Should contain the message
		if !strings.Contains(output, "complex message") {
			t.Errorf("Should contain the message: %s", output)
		}

		// Should be valid JSON
		lines := strings.Split(strings.TrimSpace(output), "\n")
		lastLine := lines[len(lines)-1]
		jsonStart := strings.Index(lastLine, "{")
		if jsonStart != -1 {
			jsonPart := lastLine[jsonStart:]
			var parsed map[string]any
			err := json.Unmarshal([]byte(jsonPart), &parsed)
			if err != nil {
				t.Errorf("Should handle complex args: %v", err)
			}
		}
	})
}

func TestInfoVsError(t *testing.T) {
	t.Run("Info vs Error output differences", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		// Test Info
		Info("info message", map[string]any{"type": "info"})
		infoOutput := buf.String()

		buf.Reset()

		// Test Error
		Error("error message", nil, map[string]any{"type": "error"})
		errorOutput := buf.String()

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
			jsonStart := strings.Index(lastLine, "{")
			if jsonStart == -1 {
				return nil
			}
			jsonPart := lastLine[jsonStart:]
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
	// Redirect log output to discard
	log.SetOutput(bytes.NewBuffer(nil))
	defer log.SetOutput(os.Stderr)

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
		Code:  "benchmark123",
		Trace: []string{"func1 10", "func2 20", "func3 30"},
		Msg:   "benchmark message",
		Args: map[string]any{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sdkLog.String()
	}
}
