package catcher

import (
	"errors"
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
