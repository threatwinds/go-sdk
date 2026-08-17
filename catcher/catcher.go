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

// logDone is closed by the async writer goroutine when it has drained
// everything it was given and is about to return. Configure waits on it when
// switching async off, so "async logging is disabled" is a state that has
// actually been reached by the time Configure returns, rather than one the
// process is heading towards. See the disable branch for why that matters.
var logDone chan struct{}
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
		logDone = make(chan struct{})
		ctx, cancel := context.WithCancel(context.Background())
		cancelFunc = cancel
		go func(ctx context.Context, ch chan string, done chan struct{}) {
			defer close(done)

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
		}(ctx, logChan, logDone)
	} else if !a && async {
		// Disabling async. The writer is asked to stop, and then waited
		// for: it writes to whatever os.Stdout points at *when each line is
		// written*, so a caller that switches to synchronous logging and
		// then redirects, replaces or closes stdout — a test capturing
		// output, a process installing its own writer — would otherwise
		// still have a live goroutine flushing earlier lines into the new
		// destination. Waiting costs a drain of what is already queued and
		// makes "logging is synchronous now" true on return.
		//
		// The channel is deliberately not closed. printLog reads logChan
		// under mu but sends outside it, so a sender can be holding a
		// reference to this channel right now; closing it would make that
		// send panic, and losing a queued INFO line is not worth a panic on
		// a logging path. Dropping the reference is enough — the next
		// printLog reads logChan == nil and writes synchronously.
		async = false
		if cancelFunc != nil {
			cancelFunc()
			cancelFunc = nil
		}

		done := logDone
		logChan = nil
		logDone = nil

		if done != nil {
			<-done
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
