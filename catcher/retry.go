package catcher

import (
	"strings"
	"time"
)

// RetryConfig defines configuration options for retry operations
type RetryConfig struct {
	MaxRetries int           // Maximum number of retries (0 = infinite)
	WaitTime   time.Duration // Wait time between retries
}

// DefaultRetryConfig provides sensible defaults for retry operations
var DefaultRetryConfig = &RetryConfig{
	MaxRetries: 5,
	WaitTime:   1 * time.Second,
}

// IsException checks if an error matches any of the specified exception patterns
func IsException(err error, exceptions ...string) bool {
	if err == nil {
		return false
	}

	for _, exception := range exceptions {
		if strings.Contains(err.Error(), exception) {
			return true
		}
	}
	return false
}

// IsSdkException checks if an SdkError matches any of the specified exception patterns
// This provides enhanced checking for SdkError types including message and cause
func IsSdkException(err *SdkError, exceptions ...string) bool {
	if err == nil {
		return false
	}

	// Check main message
	for _, exception := range exceptions {
		if strings.Contains(err.Msg, exception) {
			return true
		}

		// Check cause if it exists
		if err.Cause != nil && strings.Contains(*err.Cause, exception) {
			return true
		}
	}
	return false
}

// Retry executes a function repeatedly until it succeeds, the maximum retries are reached,
// or a matching exception is encountered. Enhanced version of logger.Retry for catcher system.
//
// Parameters:
//   - f: Function to execute that returns an error
//   - config: Retry configuration (use nil for defaults)
//   - exceptions: Error patterns that should stop retrying immediately
//
// Returns:
//   - error: nil on success, last error on failure or exception match
func Retry(f func() error, config *RetryConfig, exceptions ...string) error {
	if config == nil {
		config = DefaultRetryConfig
	}

	var retries = 0
	for {
		err := f()
		if err != nil {
			retries++

			// Check if this is an exception that should stop retrying
			if IsException(err, exceptions...) {
				// Return the original error without additional logging
				return err
			}

			// Check if we've exceeded max retries
			if config.MaxRetries > 0 && retries >= config.MaxRetries {
				// Return the original error without additional logging
				return err
			}

			time.Sleep(config.WaitTime)
		} else {
			// Success - don't log success messages
			return nil
		}
	}
}

// InfiniteRetry executes a function repeatedly until it succeeds or returns an error
// containing specified exception patterns. Enhanced version of logger.InfiniteRetry.
//
// Parameters:
//   - f: Function to execute that returns an error
//   - config: Retry configuration (MaxRetries is ignored, use nil for defaults)
//   - exceptions: Error patterns that should stop retrying immediately
//
// Returns:
//   - error: nil on success, error on exception match
func InfiniteRetry(f func() error, config *RetryConfig, exceptions ...string) error {
	if config == nil {
		config = DefaultRetryConfig
	}

	var retries = 0

	for {
		err := f()
		if err != nil {
			retries++

			// Check if this is an exception that should stop retrying
			if IsException(err, exceptions...) {
				return err
			}

			time.Sleep(config.WaitTime)
		} else {
			// Success - don't log success messages
			return nil
		}
	}
}

// InfiniteLoop continuously executes a provided function until it produces a matching
// exception error. Enhanced version of logger.InfiniteLoop for catcher system.
//
// Parameters:
//   - f: Function to execute repeatedly
//   - config: Configuration for wait time and logging (MaxRetries is ignored)
//   - exceptions: Error patterns that should stop the loop
func InfiniteLoop(f func() error, config *RetryConfig, exceptions ...string) {
	if config == nil {
		config = DefaultRetryConfig
	}

	for {
		err := f()

		if err != nil {
			// Check if this is an exception that should stop the loop
			if IsException(err, exceptions...) {
				return
			}
			// For non-exception errors, just continue - don't log
		}

		time.Sleep(config.WaitTime)
	}
}

// InfiniteRetryIfXError retries a function f() infinitely only if the error returned
// matches the specified exception. Enhanced version of logger.InfiniteRetryIfXError.
//
// This function provides advanced error filtering:
// - Retries only if error matches the specific exception
// - Returns immediately on different errors or success
// - Logs the exception only once to avoid log saturation
// - Logs when the issue is resolved
//
// Parameters:
//   - f: Function to execute that returns an error
//   - config: Retry configuration (MaxRetries is ignored)
//   - exception: Specific error pattern to retry on
//
// Returns:
//   - error: nil on success, non-matching error immediately, or context error
func InfiniteRetryIfXError(f func() error, config *RetryConfig, exception string) error {
	if config == nil {
		config = DefaultRetryConfig
	}

	for {
		err := f()

		// If error matches the specific exception, keep retrying
		if err != nil && IsException(err, exception) {
			time.Sleep(config.WaitTime)
			continue
		}

		// Return the result (nil for success, or different error)
		return err
	}
}

// RetryWithBackoff executes a function with exponential backoff retry strategy.
// This is a new enhanced retry function not available in the original logger.
//
// Parameters:
//   - f: Function to execute that returns an error
//   - config: Base retry configuration
//   - maxBackoff: Maximum backoff duration
//   - backoffMultiplier: Multiplier for exponential backoff (typically 2.0)
//   - exceptions: Error patterns that should stop retrying immediately
//
// Returns:
//   - error: nil on success, last error on failure or exception match
func RetryWithBackoff(f func() error, config *RetryConfig, maxBackoff time.Duration, backoffMultiplier float64, exceptions ...string) error {
	if config == nil {
		config = DefaultRetryConfig
	}

	var retries = 0
	currentWait := config.WaitTime

	for {
		err := f()
		if err != nil {
			retries++

			// Check if this is an exception that should stop retrying
			if IsException(err, exceptions...) {
				return err
			}

			// Check if we've exceeded max retries
			if config.MaxRetries > 0 && retries >= config.MaxRetries {
				return err
			}

			time.Sleep(currentWait)

			// Calculate next backoff duration
			nextWait := time.Duration(float64(currentWait) * backoffMultiplier)
			if nextWait > maxBackoff {
				currentWait = maxBackoff
			} else {
				currentWait = nextWait
			}
		} else {
			// Success - don't log success messages
			return nil
		}
	}
}
