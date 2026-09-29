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
// # Logging
//
// A non-exception error from f is swallowed silently, on purpose, exactly as
// before: this loop exists to tolerate failure indefinitely, and a function
// that errors on some cycles and succeeds on others is InfiniteLoop's
// designed steady state, not an incident. Logging that — even deduplicated —
// would turn ordinary operation into a permanent stream of log lines for
// something nobody needs to act on.
//
// A matching exception is different: it is the loop *terminating*. Nothing
// else observes an InfiniteLoop goroutine returning, so that is the one
// event on this path worth recording, and it is logged once, immediately
// before returning. See logLoopTermination for why that log is INFO, not
// ERROR.
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
				logLoopTermination(err)
				return
			}
			// For non-exception errors, just continue - don't log
		}

		time.Sleep(config.WaitTime)
	}
}

// logLoopTermination logs the error that stopped an InfiniteLoop. It is the
// only logging InfiniteLoop does: the loop returning is the significant
// event, not any individual swallowed cycle.
//
// An *SdkError logs itself, via its own idempotent Log() — so an error
// already logged at construction (Error) or at an earlier boundary is not
// logged a second time here. A plain error has no such flag, so it goes
// through the package's ordinary Log() path instead, which is this
// package's normal way to record a message that isn't itself an *SdkError.
//
// This logs at INFO, not ERROR. Nothing else in this file treats a matching
// exception as a failure: Retry, InfiniteRetry and RetryWithBackoff all
// return on an exception match "without additional logging", by design,
// because a matched exception is the intentional, expected way to stop
// retrying. InfiniteLoop's only current caller, auth-api's worker, is
// written the same way — every one of its five loops checks ctx.Done() and
// returns an error, evidently meant as the loop's graceful-shutdown signal,
// even though none of those call sites currently register that error text
// in their exceptions list (so today this path is unreached by them; that
// gap is in the caller, not something this function should paper over by
// guessing at severity). The shape of that code is still the best evidence
// available for what a matching exception means here: an expected,
// designed stop, not a crash. Logging a designed stop at ERROR severity
// would train whoever reads these logs to start ignoring real errors.
func logLoopTermination(err error) {
	if sdkErr, ok := err.(*SdkError); ok && sdkErr != nil {
		sdkErr.Log()
		return
	}
	Log("InfiniteLoop stopped: function returned a matching exception", map[string]any{"error": err.Error()})
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
