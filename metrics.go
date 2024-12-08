package go_sdk

import (
	"sync"
	"time"
)

// Metter represents a structure to measure the execution time of a function.
// It contains the start time, the name of the function, and additional options.
type Metter struct {
	StartTime time.Time
	Function  string
	Options   MetterOptions
	mutex     sync.RWMutex // Add mutex for thread safety
}

// MetterOptions defines the configuration options for metering.
// It includes options to log slow operations and set a threshold for what is considered slow.
//
// Fields:
//
//	LogSlow: A boolean indicating whether to log operations that are considered slow.
//	SlowThreshold: A time.Duration value that specifies the threshold duration for slow operations.
type MetterOptions struct {
	LogSlow       bool
	SlowThreshold time.Duration
}

// NewMetter creates a new Metter instance to measure the execution time of a function.
// It takes a function name as a string and an optional list of MetterOptions.
// If no options are provided, it defaults to logging slow executions with a threshold of 50 milliseconds.
//
// Parameters:
//   - function: The name of the function to be measured.
//   - options: Optional MetterOptions to customize the behavior of the Metter.
//
// Returns:
//
//	A pointer to a Metter instance.
func NewMetter(function string, options ...MetterOptions) *Metter {
	if len(options) == 0 {
		options = []MetterOptions{{LogSlow: true, SlowThreshold: 50 * time.Millisecond}}
	}

	return &Metter{
		StartTime: time.Now(),
		Function:  function,
		Options:   options[0],
	}
}

// Elapsed calculates the duration since the Metter's StartTime.
// If the LogSlow option is enabled and the elapsed time exceeds the configured threshold,
// it logs an informational message indicating that the function is slow.
// Returns the elapsed time as a time.Duration.
func (m *Metter) Elapsed(point ...string) time.Duration {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	elapsed := time.Since(m.StartTime)
	if m.Options.LogSlow {
		if elapsed > m.Options.SlowThreshold {
			Logger().Info("slow function '%s', elapsed %v to reach point '%v'", m.Function, elapsed, point)
		}
	}
	return elapsed
}

// Reset sets the StartTime of the Metter to the current time.
// This method is typically used to reset the timing metrics.
func (m *Metter) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.StartTime = time.Now()
}