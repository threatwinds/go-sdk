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
	Trace     []string       `json:"trace"`
	Msg       string         `json:"msg"`
	Args      map[string]any `json:"args,omitempty"`
	Severity  string         `json:"severity"`
}

// Info logs a message with a unique code, stack trace, and optional contextual arguments in a structured format.
func Info(msg string, args map[string]any) {
	pc := make([]uintptr, 25)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])

	var trace = make([]string, 0, 10)
	for {
		frame, more := frames.Next()

		trace = append(trace, fmt.Sprint(frame.Function, " ", frame.Line))
		if !more {
			break
		}
	}

	sum := md5.Sum([]byte(msg))

	sdkLog := SdkLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Code:      hex.EncodeToString(sum[:]),
		Trace:     trace,
		Args:      args,
		Msg:       msg,
		Severity:  "INFO",
	}

	fmt.Println(sdkLog.String())
}

// String returns the JSON-encoded string representation of the SdkLog instance.
func (e SdkLog) String() string {
	a, _ := json.Marshal(e)
	return string(a)
}
