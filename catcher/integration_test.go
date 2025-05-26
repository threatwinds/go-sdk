package catcher

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestSdkErrorIntegration tests integration with SdkError system
func TestSdkErrorIntegration(t *testing.T) {
	t.Run("retry with sdk errors", func(t *testing.T) {
		attempts := 0
		err := Retry(func() error {
			attempts++
			if attempts < 3 {
				return Error("operation failed", errors.New("database timeout"), map[string]any{
					"attempt": attempts,
					"status":  500,
				})
			}
			return nil
		}, &RetryConfig{
			MaxRetries: 5,
			WaitTime:   10 * time.Millisecond,
		})

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	const errorAuthenticationFailed = "authentication failed"

	t.Run("sdk error exception handling", func(t *testing.T) {
		attempts := 0
		err := Retry(func() error {
			attempts++
			return Error("authentication failed", errors.New("invalid token"), map[string]any{
				"status": 401,
			})
		}, &RetryConfig{
			MaxRetries: 5,
			WaitTime:   10 * time.Millisecond,
		}, errorAuthenticationFailed)

		if err == nil {
			t.Error("Expected error due to authentication exception")
		}
		if attempts != 1 {
			t.Errorf("Expected 1 attempt (stopped by exception), got %d", attempts)
		}

		// Verify it's an SdkError
		sdkErr := ToSdkError(err)
		if sdkErr == nil {
			t.Error("Expected SdkError, got different type")
		}
		if !strings.Contains(sdkErr.Msg, "authentication failed") {
			t.Errorf("Expected message to contain 'authentication failed', got: %s", sdkErr.Msg)
		}
	})
}

// TestErrorCreation tests the enhanced error creation
func TestErrorCreation(t *testing.T) {
	t.Run("error with metadata", func(t *testing.T) {
		originalErr := errors.New("connection refused")
		sdkErr := Error("database operation failed", originalErr, map[string]any{
			"operation": "insert",
			"table":     "entities",
			"status":    500,
		})

		if sdkErr.Msg != "database operation failed" {
			t.Errorf("Expected message 'database operation failed', got: %s", sdkErr.Msg)
		}

		if sdkErr.Cause == nil || *sdkErr.Cause != "connection refused" {
			t.Errorf("Expected cause 'connection refused', got: %v", sdkErr.Cause)
		}

		if sdkErr.Args["operation"] != "insert" {
			t.Errorf("Expected operation 'insert', got: %v", sdkErr.Args["operation"])
		}

		if sdkErr.Code == "" {
			t.Error("Expected non-empty error code")
		}

		if len(sdkErr.Trace) == 0 {
			t.Error("Expected non-empty trace")
		}
	})
}

// TestBackoffRetry tests the new backoff functionality
func TestBackoffRetry(t *testing.T) {
	t.Run("backoff timing", func(t *testing.T) {
		attempts := 0
		start := time.Now()

		err := RetryWithBackoff(func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		}, &RetryConfig{
			MaxRetries: 5,
			WaitTime:   50 * time.Millisecond,
		}, time.Second, 2.0)

		duration := time.Since(start)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
		// Should have waited: 50ms + 100ms = 150ms minimum
		if duration < 100*time.Millisecond {
			t.Errorf("Expected at least 100ms with backoff, got %v", duration)
		}
	})
}

// BenchmarkRetryPerformance compares performance of different retry methods
func BenchmarkRetryPerformance(b *testing.B) {
	config := &RetryConfig{
		MaxRetries: 3,
		WaitTime:   1 * time.Microsecond,
	}

	b.Run("new retry system", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Retry(func() error {
				return nil
			}, config)
		}
	})
}

const connectionFailed string = "connection failed"

// TestRealWorldScenarios tests scenarios common in ThreatWinds APIs
func TestRealWorldScenarios(t *testing.T) {
	t.Run("database connection scenario", func(t *testing.T) {
		attempts := 0
		connected := false

		err := InfiniteRetryIfXError(func() error {
			attempts++
			if attempts < 5 {
				return Error("database connection failed", errors.New("connection refused"), map[string]any{
					"host":   "localhost:5432",
					"status": 500,
				})
			}
			connected = true
			return nil
		}, &RetryConfig{
			WaitTime: 10 * time.Millisecond,
		}, connectionFailed)

		if err != nil {
			t.Errorf("Expected eventual success, got: %v", err)
		}
		if !connected {
			t.Error("Expected to be connected")
		}
		if attempts != 5 {
			t.Errorf("Expected 5 attempts, got %d", attempts)
		}
	})

	var errorPermanent string = "permanent"

	t.Run("external API with rate limiting", func(t *testing.T) {
		attempts := 0

		err := RetryWithBackoff(func() error {
			attempts++
			if attempts < 3 {
				return Error("rate limited", errors.New("too many requests"), map[string]any{
					"status":      429,
					"retry_after": "1s",
				})
			}
			return nil
		}, &RetryConfig{
			MaxRetries: 5,
			WaitTime:   10 * time.Millisecond,
		}, 500*time.Millisecond, 2.0, errorPermanent)

		if err != nil {
			t.Errorf("Expected success after rate limiting, got: %v", err)
		}
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})
}
