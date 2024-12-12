package go_sdk

import (
	"errors"
	"testing"
)

func TestTrace(t *testing.T) {
	t.Run("test nil", func(t *testing.T) {
		if Error(nil, Trace(), map[string]interface{}{"sada": "adadasd"}) != nil {
			t.Errorf("Error(Trace(), nil) should return nil")
		}
	})

	t.Run("test error", func(t *testing.T) {
		if err := Error(errors.New("test error"), Trace(), map[string]interface{}{"sada": "adadasd"}); err == nil {
			t.Errorf("Error(Trace(), NewError('test error')) should return error")
		} else {
			t.Log(err)
		}
	})

	t.Run("test error with trace", func(t *testing.T) {
		if err := Error(errors.New("test error"), Trace(), map[string]interface{}{"sada": "adadasd"}); err == nil {
			t.Errorf("Error(Trace(), NewError('test error')) should return error")
		} else {
			t.Log(err)
		}
	})
}
