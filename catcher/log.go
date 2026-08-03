package catcher

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

// SdkLog represents a structured log entry with unique code, stack trace, message, and optional additional arguments.
type SdkLog struct {
	Timestamp string         `json:"timestamp"`
	Code      string         `json:"code"`
	Trace     []string       `json:"trace,omitempty"`
	Msg       string         `json:"msg"`
	Args      map[string]any `json:"args,omitempty"`
	Severity  string         `json:"severity"`
}

// Info logs a message with a unique code, stack trace, and optional contextual arguments in a structured format.
func Info(msg string, args map[string]any) {
	Log(msg, args)
}

// Warn logs a message with WARNING severity.
func Warn(msg string, args map[string]any) {
	if args == nil {
		args = make(map[string]any)
	}
	args["status"] = 400 // Triggers WARNING severity in calculateSeverity
	Log(msg, args)
}

// Log logs a message with a unique code, stack trace, and optional contextual arguments in a structured format.
func Log(msg string, args map[string]any) {
	mu.Lock()
	nt := noTrace
	b := beauty
	mu.Unlock()

	var trace []string
	if !nt {
		pc := make([]uintptr, 25)
		n := runtime.Callers(2, pc)
		frames := runtime.CallersFrames(pc[:n])

		trace = make([]string, 0, 10)
		for {
			frame, more := frames.Next()

			trace = append(trace, fmt.Sprint(frame.Function, " ", frame.Line))
			if !more {
				break
			}
		}
	}

	sum := md5.Sum([]byte(msg))

	sdkLog := SdkLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Code:      hex.EncodeToString(sum[:]),
		Trace:     trace,
		Args:      args,
		Msg:       msg,
	}

	statusCode, ok := args["status"]
	if !ok {
		sdkLog.Severity = "INFO"
	} else {
		sdkLog.Severity = calculateSeverity(statusCode)
	}

	if b {
		printLog(fmt.Sprint(GetSeverityIcon(sdkLog.Severity), " ", sdkLog.JSON()), sdkLog.Severity)
	} else {
		printLog(sdkLog.JSON(), sdkLog.Severity)
	}
}

// isFatalSeverity reports whether a message is severe enough that losing it
// would leave an operator with nothing to go on.
func isFatalSeverity(severity string) bool {
	switch severity {
	case "ERROR", "CRITICAL", "ALERT":
		return true
	default:
		return false
	}
}

// printLog emits one already-formatted line.
//
// Severity decides how. Anything ERROR or above is written synchronously, on
// purpose: those are overwhelmingly logged immediately before os.Exit or a
// log.Fatal, and the async path hands the message to a channel that a goroutine
// drains later — a goroutine that never runs if the process is already gone.
// That is not theoretical. compute-api's package init logs the reason it cannot
// start and exits; with async on, it exited 1 with zero bytes on both stdout and
// stderr, in tests and in production alike. Async is the default (catcher's init
// enables it unless CATCHER_ASYNC is literally "false"), and package init runs
// before main can reconfigure anything, so no service could opt out of that in
// time.
//
// Lower severities keep the async path. They are the high-volume ones the
// buffering exists for, and losing an INFO line to a racing exit costs nothing.
func printLog(msg, severity string) {
	mu.Lock()
	isAsync := async
	ch := logChan
	mu.Unlock()

	if isAsync && ch != nil && !isFatalSeverity(severity) {
		select {
		case ch <- msg:
		default:
			// Si el canal está lleno, imprimir directamente para no perder logs críticos
			// aunque esto cause latencia temporalmente.
			fmt.Println(msg)
		}
		return
	}

	// Written straight to the fd rather than via fmt.Println so the bytes are
	// gone before an os.Exit on the very next line can discard them.
	_, _ = os.Stdout.WriteString(msg + "\n")
}

// JSON returns the JSON-encoded string representation of the SdkLog instance.
func (e SdkLog) JSON() string {
	a, _ := json.Marshal(e)

	return string(a)
}
