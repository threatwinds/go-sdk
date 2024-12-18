package go_sdk

import (
	"errors"
	"testing"
)

func TestTrace(t *testing.T) {
	t.Run("test error", func(t *testing.T) {
		if err := Error(Trace(), map[string]interface{}{"sada": "adadasd"}); err == nil {
			t.Errorf("Error(Trace(), NewError('test error')) should return error")
		} else {
			t.Log(err)
		}
	})

	t.Run("test error with trace", func(t *testing.T) {
		if err := Error(Trace(), map[string]interface{}{"error": errors.New("test error").Error(), "sada": "adadas"}); err == nil {
			t.Errorf("Error(Trace(), NewError('test error')) should return error")
		} else {
			t.Log(err)
		}
	})

	t.Run("cast from error", func(t *testing.T) {
		var err error
		err = Error(Trace(), map[string]interface{}{"error": "test"})

		e := ToSdkError(err)
		if e == nil {
			t.Error("expected an SdkError")
			return
		}

		t.Log(e.Error())
	})
}
