package go_sdk

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime"
)

type ErrorObject struct {
	Code  string                 `json:"code"`
	Trace []string               `json:"trace"`
	Args  map[string]interface{} `json:"args"`
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
