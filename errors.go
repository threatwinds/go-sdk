package go_sdk

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"runtime"
)

type ErrorObject struct {
	Code  string                 `json:"code"`
	Trace []string               `json:"trace,omitempty"`
	Args  map[string]interface{} `json:"args,omitempty"`
}

func (e ErrorObject) Error() string {
	a, _ := json.Marshal(e)
	return string(a)
}

func Error(trace []string, args map[string]interface{}) error {
	sum := md5.Sum([]byte(fmt.Sprint(trace, args)))

	return ErrorObject{
		Code:  hex.EncodeToString(sum[:]),
		Trace: trace,
		Args:  args,
	}
}

func Trace() []string {
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

	return trace
}

// GinError is a helper function to return an error to the client using Gin framework context.
// It sets the headers x-error and x-error-id with the error message and UUID respectively and sets the status code.
func (e ErrorObject) GinError(c *gin.Context) {
	c.Header("x-error-id", e.Code)

	message, ok := e.Args["message"].(string)
	if !ok {
		c.Header("x-error", "internal server error")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	} else {
		c.Header("x-error", message)
	}

	status, ok := e.Args["status"].(int)
	if !ok {
		c.AbortWithStatus(http.StatusBadRequest)
	} else {
		c.AbortWithStatus(status)
	}
}
