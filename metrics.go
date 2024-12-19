package go_sdk

import (
	"log"
	"sync"
	"time"
)

// Meter represents a structure to measure the execution time of a function.
// It contains the start time, the name of the function, and additional options.
type Meter struct {
	StartTime time.Time
	Function  string
	Options   MeterOptions
	mutex     sync.RWMutex // Add mutex for thread safety
}

// MeterOptions defines the configuration options for metering.
// It includes options to log slow operations and set a threshold for what is considered slow.
//
// Fields:
//
//	LogSlow: A boolean indicating whether to log operations that are considered slow.
//	SlowThreshold: A time.Duration value that specifies the threshold duration for slow operations.
type MeterOptions struct {
	LogSlow       bool
	SlowThreshold time.Duration
}

// NewMeter creates a new Meter instance to measure the execution time of a function.
// It takes a function name as a string and an optional list of MeterOptions.
// If no options are provided, it defaults to logging slow executions with a threshold of 50 milliseconds.
//
// Parameters:
//   - function: The name of the function to be measured.
//   - options: Optional MeterOptions to customize the behavior of the Meter.
//
// Returns:
//
//	A pointer to a Meter instance.
func NewMeter(function string, options ...MeterOptions) *Meter {
	if len(options) == 0 {
		options = []MeterOptions{{LogSlow: true, SlowThreshold: 50 * time.Millisecond}}
	}

	return &Meter{
		StartTime: time.Now(),
		Function:  function,
		Options:   options[0],
	}
}

// Elapsed calculates the duration since the Meter's StartTime.
// If the LogSlow option is enabled and the elapsed time exceeds the configured threshold,
// it logs an informational message indicating that the function is slow.
// Returns the elapsed time as a time.Duration.
func (m *Meter) Elapsed(point string) time.Duration {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	elapsed := time.Since(m.StartTime)
	if m.Options.LogSlow {
		if elapsed > m.Options.SlowThreshold {
			log.Println(Error("slow operation", nil, map[string]any{
				"function": m.Function,
				"elapsed":  elapsed,
				"point":    point,
				"advice":   "consider to increase the processing power",
			}))
		}
	}
	return elapsed
}

// Reset sets the StartTime of the Meter to the current time.
// This method is typically used to reset the timing metrics.
func (m *Meter) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.StartTime = time.Now()
}
