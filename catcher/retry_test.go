package catcher

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestIsException(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		exceptions []string
		expected   bool
	}{
		{
			name:       "nil error",
			err:        nil,
			exceptions: []string{"test"},
			expected:   false,
		},
		{
			name:       "matching exception",
			err:        errors.New("database connection failed"),
			exceptions: []string{"database connection", "timeout"},
			expected:   true,
		},
		{
			name:       "no matching exception",
			err:        errors.New("validation error"),
			exceptions: []string{"connection", "timeout"},
			expected:   false,
		},
		{
			name:       "exact match",
			err:        errors.New("not found"),
			exceptions: []string{"not found"},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsException(tt.err, tt.exceptions...)
			if result != tt.expected {
				t.Errorf("IsException() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsSdkException(t *testing.T) {
	tests := []struct {
		name       string
		err        *SdkError
		exceptions []string
		expected   bool
	}{
		{
			name:       "nil error",
			err:        nil,
			exceptions: []string{"test"},
			expected:   false,
		},
		{
			name: "matching message",
			err: &SdkError{
				Msg: "database connection failed",
			},
			exceptions: []string{"connection"},
			expected:   true,
		},
		{
			name: "matching cause",
			err: &SdkError{
				Msg:   "operation failed",
				Cause: func() *string { s := "timeout occurred"; return &s }(),
			},
			exceptions: []string{"timeout"},
			expected:   true,
		},
		{
			name: "no match",
			err: &SdkError{
				Msg: "validation error",
			},
			exceptions: []string{"connection", "timeout"},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSdkException(tt.err, tt.exceptions...)
			if result != tt.expected {
				t.Errorf("IsSdkException() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestRetry(t *testing.T) {
	t.Run("immediate success", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			return nil
		}

		config := &RetryConfig{
			MaxRetries: 3,
			WaitTime:   10 * time.Millisecond,
		}

		err := Retry(f, config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("success after retries", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		}

		config := &RetryConfig{
			MaxRetries: 5,
			WaitTime:   10 * time.Millisecond,
		}

		err := Retry(f, config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			return errors.New("persistent error")
		}

		config := &RetryConfig{
			MaxRetries: 3,
			WaitTime:   10 * time.Millisecond,
		}

		err := Retry(f, config)
		if err == nil {
			t.Error("Expected error after max retries")
		}
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("exception stops retry", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			return errors.New("not found error")
		}

		config := &RetryConfig{
			MaxRetries: 5,
			WaitTime:   10 * time.Millisecond,
		}

		err := Retry(f, config, "not found")
		if err == nil {
			t.Error("Expected error due to exception")
		}
		if attempts != 1 {
			t.Errorf("Expected 1 attempt (stopped by exception), got %d", attempts)
		}
	})

	t.Run("default config", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			return nil
		}

		err := Retry(f, nil) // Use default config
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}
	})
}

func TestInfiniteRetry(t *testing.T) {
	t.Run("immediate success", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			return nil
		}

		config := &RetryConfig{
			WaitTime: 10 * time.Millisecond,
		}

		err := InfiniteRetry(f, config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("success after many retries", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			if attempts < 10 {
				return errors.New("temporary error")
			}
			return nil
		}

		config := &RetryConfig{
			WaitTime: 1 * time.Millisecond,
		}

		err := InfiniteRetry(f, config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 10 {
			t.Errorf("Expected 10 attempts, got %d", attempts)
		}
	})

	const fatalError string = "fatal error occurred"

	t.Run("exception stops retry", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			return errors.New(fatalError)
		}

		config := &RetryConfig{
			WaitTime: 10 * time.Millisecond,
		}

		err := InfiniteRetry(f, config, fatalError)
		if err == nil {
			t.Error("Expected error due to exception")
		}
		if attempts != 1 {
			t.Errorf("Expected 1 attempt (stopped by exception), got %d", attempts)
		}
	})
}

func TestInfiniteRetryIfXError(t *testing.T) {
	t.Run("immediate success", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			return nil
		}

		config := &RetryConfig{
			WaitTime: 10 * time.Millisecond,
		}

		err := InfiniteRetryIfXError(f, config, "connection")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("retries on specific error", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			if attempts < 5 {
				return errors.New("connection timeout")
			}
			return nil
		}

		config := &RetryConfig{
			WaitTime: 1 * time.Millisecond,
		}

		err := InfiniteRetryIfXError(f, config, "connection")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 5 {
			t.Errorf("Expected 5 attempts, got %d", attempts)
		}
	})

	t.Run("stops on different error", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			if attempts == 1 {
				return errors.New("connection timeout")
			}
			return errors.New("validation failed")
		}

		config := &RetryConfig{
			WaitTime: 1 * time.Millisecond,
		}

		err := InfiniteRetryIfXError(f, config, "connection")
		if err == nil {
			t.Error("Expected error when different error occurs")
		}
		if !IsException(err, "validation") {
			t.Errorf("Expected validation error, got %v", err)
		}
		if attempts != 2 {
			t.Errorf("Expected 2 attempts, got %d", attempts)
		}
	})
}

func TestRetryWithBackoff(t *testing.T) {
	t.Run("immediate success", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			return nil
		}

		config := &RetryConfig{
			MaxRetries: 3,
			WaitTime:   10 * time.Millisecond,
		}

		err := RetryWithBackoff(f, config, time.Second, 2.0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("success with backoff", func(t *testing.T) {
		attempts := 0
		start := time.Now()
		f := func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		}

		config := &RetryConfig{
			MaxRetries: 5,
			WaitTime:   50 * time.Millisecond, // Base wait time
		}

		err := RetryWithBackoff(f, config, time.Second, 2.0)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
		// Should have waited: 50ms + 100ms = 150ms (with some tolerance)
		if duration < 100*time.Millisecond {
			t.Errorf("Expected at least 100ms with backoff, got %v", duration)
		}
	})

	t.Run("max backoff limit", func(t *testing.T) {
		attempts := 0
		f := func() error {
			attempts++
			if attempts < 4 {
				return errors.New("temporary error")
			}
			return nil
		}

		config := &RetryConfig{
			MaxRetries: 5,
			WaitTime:   10 * time.Millisecond,
		}

		maxBackoff := 15 * time.Millisecond
		err := RetryWithBackoff(f, config, maxBackoff, 2.0)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 4 {
			t.Errorf("Expected 4 attempts, got %d", attempts)
		}
	})
}

// withSyncLogging disables async logging for the duration of the calling
// test and restores whatever Configure state was in effect before it,
// afterward. It exists because captureStdout only produces deterministic
// results for synchronous writes (see its doc comment in errors_test.go);
// InfiniteLoop's termination log is INFO severity, which would otherwise
// race the async drain goroutine exactly like the already-flaky TestInfo.
func withSyncLogging(t *testing.T) {
	t.Helper()

	mu.Lock()
	b, a, nt := beauty, async, noTrace
	mu.Unlock()

	Configure(b, false, nt)
	t.Cleanup(func() {
		Configure(b, a, nt)
	})
}

func TestInfiniteLoop(t *testing.T) {
	t.Run("non-exception errors are swallowed silently, however many iterations run", func(t *testing.T) {
		withSyncLogging(t)

		calls := 0
		output := captureStdout(t, func() {
			InfiniteLoop(func() error {
				calls++
				if calls >= 25 {
					return errors.New("stop-now")
				}
				return errors.New("persistent failure")
			}, &RetryConfig{WaitTime: time.Millisecond}, "stop-now")
		})

		if calls != 25 {
			t.Fatalf("expected the loop to run 25 iterations before stopping, got %d", calls)
		}
		if strings.Contains(output, "persistent failure") {
			t.Errorf("swallowed non-exception errors must never be logged, got: %q", output)
		}

		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 1 {
			t.Fatalf("expected exactly one log line (the termination log only), got %d: %q", len(lines), output)
		}
	})

	t.Run("the exception path that terminates the loop logs exactly once", func(t *testing.T) {
		withSyncLogging(t)

		output := captureStdout(t, func() {
			InfiniteLoop(func() error {
				return errors.New("terminate-marker")
			}, &RetryConfig{WaitTime: time.Millisecond}, "terminate-marker")
		})

		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 1 {
			t.Fatalf("expected exactly one log line, got %d: %q", len(lines), output)
		}

		entry := lastJSONLine(t, output)
		if entry["msg"] != "InfiniteLoop stopped: function returned a matching exception" {
			t.Errorf("unexpected termination message: %v", entry["msg"])
		}
	})

	t.Run("an *SdkError already logged via Log() is not logged twice by the exception path", func(t *testing.T) {
		withSyncLogging(t)

		sdkErr := New("worker shutting down", nil, nil) // New does not log at construction

		output := captureStdout(t, func() {
			sdkErr.Log() // the one log line we expect to see

			InfiniteLoop(func() error {
				return sdkErr
			}, &RetryConfig{WaitTime: time.Millisecond}, "shutting down")
		})

		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 1 {
			t.Fatalf("Log()'s idempotency should have suppressed the duplicate, got %d lines: %q", len(lines), output)
		}

		entry := lastJSONLine(t, output)
		if entry["msg"] != "worker shutting down" {
			t.Errorf("unexpected log entry: %v", entry)
		}
	})

	t.Run("a plain non-SdkError exception is still logged, informationally", func(t *testing.T) {
		withSyncLogging(t)

		output := captureStdout(t, func() {
			InfiniteLoop(func() error {
				return errors.New("shutdown requested")
			}, &RetryConfig{WaitTime: time.Millisecond}, "shutdown requested")
		})

		entry := lastJSONLine(t, output)
		if entry["severity"] != "INFO" {
			t.Errorf("expected INFO severity for a designed loop termination, got %v", entry["severity"])
		}

		args, ok := entry["args"].(map[string]any)
		if !ok || args["error"] != "shutdown requested" {
			t.Errorf("expected the terminating error text in args, got %v", entry)
		}
	})
}

// Benchmark tests
func BenchmarkRetry(b *testing.B) {
	config := &RetryConfig{
		MaxRetries: 3,
		WaitTime:   1 * time.Microsecond,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Retry(func() error {
			return nil
		}, config)
	}
}

func BenchmarkIsException(b *testing.B) {
	err := errors.New("database connection failed")
	exceptions := []string{"connection", "timeout", "network"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsException(err, exceptions...)
	}
}
