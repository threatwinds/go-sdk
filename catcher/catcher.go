package catcher

import (
	"context"
	"os"
	"sync"
)

var beauty bool
var async bool
var noTrace bool
var logChan chan string
var cancelFunc context.CancelFunc
var mu sync.Mutex

const (
	debugIcon    = "🔍"  // magnifying glass
	infoIcon     = "ℹ️" // information
	noticeIcon   = "📢"  // loudspeaker
	warningIcon  = "⚠️" // warning
	errorIcon    = "❌"  // cross mark
	criticalIcon = "🔥"  // fire
	alertIcon    = "🚨"  // rotating light
)

func init() {
	b := os.Getenv("CATCHER_BEAUTY") != "false"
	a := os.Getenv("CATCHER_ASYNC") != "false"
	nt := os.Getenv("CATCHER_NO_TRACE") != "false"

	Configure(b, a, nt)
}

// Configure sets the catcher configuration and can be called programmatically to override env variables and defaults.
// - b: Beautify output (colors, indentation, etc.)
// - a: Enable async mode (log messages are sent to a channel instead of printed to stdout)
// - nt: Disable stack trace printing
func Configure(b, a, nt bool) {
	mu.Lock()
	defer mu.Unlock()

	beauty = b
	noTrace = nt

	// Handle async mode transition
	if a && !async {
		// Enabling async
		async = true
		logChan = make(chan string, 10000)
		ctx, cancel := context.WithCancel(context.Background())
		cancelFunc = cancel
		go func(ctx context.Context, ch chan string) {
			for {
				select {
				case msg, ok := <-ch:
					if !ok {
						return
					}
					_, _ = os.Stdout.WriteString(msg + "\n")
				case <-ctx.Done():
					// Drain channel before exiting
					for {
						select {
						case msg, ok := <-ch:
							if !ok {
								return
							}
							_, _ = os.Stdout.WriteString(msg + "\n")
						default:
							return
						}
					}
				}
			}
		}(ctx, logChan)
	} else if !a && async {
		// Disabling async
		async = false
		if cancelFunc != nil {
			cancelFunc()
			cancelFunc = nil
		}
		// Drain and close channel
		if logChan != nil {
			close(logChan) // Ahora podemos cerrarlo porque printLog es seguro con mu y logChan=nil
			logChan = nil
		}
	} else if a && async {
		// Already async, nothing to do for the goroutine
	}
}

// GetSeverityIcon returns an icon based on the severity level
func GetSeverityIcon(severity string) string {
	switch severity {
	case "DEBUG":
		return debugIcon
	case "INFO":
		return infoIcon
	case "NOTICE":
		return noticeIcon
	case "WARNING":
		return warningIcon
	case "ERROR":
		return errorIcon
	case "CRITICAL":
		return criticalIcon
	case "ALERT":
		return alertIcon
	default:
		return errorIcon
	}
}
