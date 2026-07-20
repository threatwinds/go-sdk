package catcher

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		printLog(fmt.Sprint(GetSeverityIcon(sdkLog.Severity), " ", sdkLog.JSON()))
	} else {
		printLog(sdkLog.JSON())
	}
}

func printLog(msg string) {
	mu.Lock()
	isAsync := async
	ch := logChan
	mu.Unlock()

	if isAsync && ch != nil {
		select {
		case ch <- msg:
		default:
			// Si el canal está lleno, imprimir directamente para no perder logs críticos
			// aunque esto cause latencia temporalmente.
			fmt.Println(msg)
		}
	} else {
		fmt.Println(msg)
	}
}

// JSON returns the JSON-encoded string representation of the SdkLog instance.
func (e SdkLog) JSON() string {
	a, _ := json.Marshal(e)

	return string(a)
}
