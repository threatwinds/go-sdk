package catcher

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"net/http"
	"runtime"
	"strconv"
	"time"
)

// errorParam represents a single error field in the JSON response.
type errorParam struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	Param   string `json:"param,omitempty"`
}

// errorResponse is the JSON envelope returned to the client on error.
type errorResponse struct {
	Error errorParam `json:"error"`
}

// SdkError is a struct that implements the Go error interface.
type SdkError struct {
	Timestamp string         `json:"timestamp"`
	Code      string         `json:"code"`
	Trace     []string       `json:"trace,omitempty"`
	Msg       string         `json:"msg"`
	Cause     *string        `json:"cause,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
	Severity  string         `json:"severity"`

	// cause holds the original error this SdkError was built from, so that
	// Unwrap can expose it to errors.Is/errors.As. It is deliberately
	// unexported: the wire format (JSON tags above) is unchanged, so every
	// existing consumer that reads .Cause as a string, or round-trips an
	// SdkError through JSON, keeps working exactly as before. An SdkError
	// decoded from JSON (e.g. received from another service) has a nil
	// cause and Unwraps to nil, which is correct — the original error value
	// was never transmitted, only its rendered text in Cause.
	cause error
}

// Error returns the error message.
func (e SdkError) Error() string {
	args, _ := json.Marshal(e.Args)

	causeText := "unknown cause"
	if e.Cause != nil {
		causeText = *e.Cause
	}

	return fmt.Sprintf("%s: %s. Args: %s", e.Msg, causeText, args)
}

// Unwrap returns the original error this SdkError was constructed from, if
// any, so errors.Is and errors.As can traverse beneath it.
//
// This uses a pointer receiver, unlike Error, JSON and SecureString below
// (which stay on value receivers for backwards compatibility — changing
// them would stop a bare SdkError value, as opposed to *SdkError, from
// satisfying the error interface). errors.Is/errors.As call Unwrap on every
// node of a chain without a nil check first. A value receiver has to
// dereference the receiver to build its copy, which panics immediately when
// called on a typed-nil *SdkError — and this package already produces
// exactly that value in the wild (see ToSdkError's typed-nil handling
// below). A pointer receiver instead returns nil for a nil receiver,
// matching Unwrap's documented contract of "no further error", so a routine
// errors.Is/As call can never turn into a panic. Since Error() (below)
// returns *SdkError everywhere in this package, *SdkError's method set —
// which includes both this pointer-receiver method and the value-receiver
// ones — is what every real caller's error chain actually carries.
func (e *SdkError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// JSON returns the JSON string representation of the SdkError.
func (e SdkError) JSON() string {
	jLog, _ := json.Marshal(e)
	return string(jLog)
}

// SecureString returns the error message if the status code is >= 500, otherwise it returns the full error description.
func (e SdkError) SecureString() string {
	status, ok := e.Args["status"]
	if ok {
		if castInt(status) >= 500 {
			return e.Msg
		}
	}

	return e.Error()
}

// Error tries to cast the cause as an SdkError, if it is not an SdkError, it creates a new SdkError with the given parameters.
// It logs the error message and returns the error.
// If cause is nil, it will store a blank string in the Cause field.
// The field Code is a hash of the message and trace. It is used to identify the recurrence of an error.
// Params:
// msg: the error message.
// cause: the error that caused this error.
// args: a map of additional information. Recognized keys when used with GinError():
//
//		"status" (int)         → HTTP status code (default 500)
//		"retry" (int)          → Retry-After header value in seconds
//		"code_override" (string) → overrides the error code in the JSON body
//		"param" (string)       → included as "param" in the JSON error detail
//
// # This package logs at construction, not at handling
//
// If cause is already an *SdkError, Error does NOT construct a new error and
// does NOT log anything: it returns cause unchanged, exactly as built and
// logged the first time it was constructed. msg, args, and any
// args["status"] override are silently ignored in that case — the returned
// error's identity, Code, Severity and Args must stay stable as it
// propagates back up through however many call sites already have it, and
// re-logging it once per layer on the way up is not something this
// function does.
//
// This means passing an existing *SdkError as cause is never the right way
// to add logged context. To log context about an error you already have,
// pass nil as cause and fold the error text into args instead:
//
//	sdkErr := catcher.ToSdkError(err)
//	catcher.Error("context about the failure", nil, map[string]any{"cause": sdkErr.Error()})
//
// Returns:
// *SdkError: the error. This type implements the Go error interface.
func Error(msg string, cause error, args map[string]any) *SdkError {
	var err *SdkError
	if err = ToSdkError(cause); err == nil {
		var trace []string
		if !noTrace {
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

		// effectiveCause normalizes cause to nil in the two cases that must
		// not participate in the chain: a genuinely nil cause, and a
		// non-nil `error` interface wrapping a nil *SdkError (see
		// ToSdkError). Both render as "unknown cause" in the Cause string
		// below, and for the same reason both must Unwrap to nil rather
		// than to a nil pointer: calling cause.Error() on a typed-nil
		// *SdkError would dereference a nil receiver (Error() has a value
		// receiver) and panic, and handing that same nil pointer out via
		// Unwrap would only defer the panic to whatever errors.Is/As caller
		// eventually calls a method on it.
		effectiveCause := cause
		if sdkCause, ok := cause.(*SdkError); ok && sdkCause == nil {
			effectiveCause = nil
		}

		causeText := "unknown cause"
		if effectiveCause != nil {
			causeText = effectiveCause.Error()
		}

		err = &SdkError{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Code:      hex.EncodeToString(sum[:]),
			Trace:     trace,
			Args:      args,
			Msg:       msg,
			Cause:     pointerOf(causeText),
			cause:     effectiveCause,
		}

		statusCode, ok := args["status"]
		if !ok {
			err.Severity = "ERROR"
		} else {
			err.Severity = calculateSeverity(statusCode)
		}

		if beauty {
			printLog(fmt.Sprint(GetSeverityIcon(err.Severity), " ", err.JSON()), err.Severity)
		} else {
			printLog(err.JSON(), err.Severity)
		}
	}

	return err
}

func calculateSeverity(value interface{}) string {
	statusCode := castInt(value)

	if statusCode >= 100 && statusCode < 200 {
		return "DEBUG"
	} else if statusCode >= 200 && statusCode < 300 {
		return "INFO"
	} else if statusCode >= 300 && statusCode < 400 {
		return "NOTICE"
	} else if statusCode >= 400 && statusCode < 500 {
		return "WARNING"
	} else if statusCode >= 500 && statusCode < 502 {
		return "ERROR"
	} else if statusCode >= 502 && statusCode < 509 {
		return "CRITICAL"
	} else if statusCode >= 509 && statusCode < 511 {
		return "ALERT"
	}

	return "ERROR"
}

// ToSdkError tries to cast an error to a SdkError.
// If the error isn't an SdkError, it returns nil.
//
// It also returns nil when err's dynamic value is a nil *SdkError boxed in a
// non-nil error interface — e.g. a function declared to return *SdkError that
// returns nil, then gets assigned to an `error` variable. errors.As matches
// that case (the concrete type is right) and sets the target to that nil
// pointer; returning it as-is would hand callers a *SdkError that is not
// == nil as a concrete pointer but panics the moment any value-receiver
// method (Error, JSON, SecureString) is called on it.
func ToSdkError(err error) *SdkError {
	if err == nil {
		return nil
	}

	var sdkError *SdkError
	if errors.As(err, &sdkError) && sdkError != nil {
		return sdkError
	}

	return nil
}

// GinError is a helper function to return an error to the client using Gin framework context.
// It sets the headers x-error and x-error-id with the error message and UUID respectively,
// writes a JSON error body, and sets the status code.
// Additional Args keys:
//   - "retry": N — sets the Retry-After header to N seconds.
//   - "code_override": string — overrides the error code in the response body.
//   - "param": string — included as "param" in the response error detail.
func (e SdkError) GinError(c *gin.Context) {
	secureMsg := e.SecureString()
	c.Header("x-error-id", e.Code)
	c.Header("x-error", secureMsg)

	if retryVal, ok := e.Args["retry"]; ok {
		retry := castInt(retryVal)
		if retry > 0 {
			c.Header("Retry-After", strconv.Itoa(retry))
		}
	}

	resp := errorResponse{
		Error: errorParam{
			Message: secureMsg,
			Type:    e.Severity,
			Code:    e.Code,
		},
	}

	if codeOverride, ok := e.Args["code_override"]; ok {
		resp.Error.Code, _ = codeOverride.(string)
	}
	if param, ok := e.Args["param"]; ok {
		if p, ok := param.(string); ok {
			resp.Error.Param = p
		}
	}

	statusCode := http.StatusInternalServerError
	if status, ok := e.Args["status"]; ok {
		statusCode = castInt(status)
	}
	c.AbortWithStatusJSON(statusCode, resp)
}

func pointerOf[t any](s t) *t {
	return &s
}

func castInt(value interface{}) int {
	if value == nil {
		return 500
	}

	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		val, err := strconv.Atoi(v)
		if err != nil {
			return 500
		}
		return val
	default:
		return 500
	}
}
