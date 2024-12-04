package go_sdk

import "time"

type Metter struct {
	StartTime time.Time
	Function  string
	Options   MetterOptions
}

type MetterOptions struct {
	LogSlow bool
	SlowThreshold time.Duration
}

func NewMetter(function string, options ...MetterOptions) *Metter {
	if len(options) == 0 {
		options = []MetterOptions{{LogSlow: true, SlowThreshold: 500 * time.Millisecond}}
	}

	return &Metter{
		StartTime: time.Now(),
		Function:  function,
		Options:   options[0],
	}
}

func (m *Metter) Elapsed() time.Duration {
	elapsed := time.Since(m.StartTime)
	if m.Options.LogSlow {
		if elapsed > 500*time.Millisecond {
			Logger().Info("slow function: %s, elapsed %v", m.Function, elapsed)
		}
	}
	return elapsed
}
