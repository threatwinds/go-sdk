package go_sdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
)

type ErrorObject struct {
	Message string                 `json:"message"`
	Trace   []string               `json:"trace"`
	Args    map[string]interface{} `json:"args"`
}

func Error(err error, trace []string, args map[string]interface{}) error {
	if err != nil {
		a, _ := json.Marshal(ErrorObject{
			Message: err.Error(),
			Trace:   trace,
			Args:    args,
		})
		return errors.New(string(a))
	}

	return nil
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
