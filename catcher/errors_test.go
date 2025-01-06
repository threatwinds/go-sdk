package catcher

import (
	"errors"
	"testing"
)

func TestTrace(t *testing.T) {
	t.Run("test error", func(t *testing.T) {
		if err := Error("any error", nil, nil); err == nil {
			t.Errorf("should return error")
			return
		}
	})

	t.Run("test error with arg", func(t *testing.T) {
		if err := Error("any error with arg", errors.New("and cause"), map[string]any{"argument": "value"}); err == nil {
			t.Errorf("should return error")
			return
		}
	})

	t.Run("cast from error", func(t *testing.T) {
		var err error
		err = Error("any error with arg", errors.New("and cause"), map[string]any{"argument": "value"})

		e := Error("casting error", err, nil)
		if e == nil {
			t.Error("expected an SdkError")
			return
		}
		if e.Msg != "any error with arg" {
			t.Error("expected an SdkError")
			return
		}
	})

	t.Run("new error", func(t *testing.T) {
		err := errors.New("any error")
		e := Error("error from Go error", err, nil)
		if e == nil {
			t.Error("expected an SdkError")
			return
		}

		if e.Msg != "error from Go error" {
			t.Error("expected an SdkError")
			return
		}

		if *e.Cause != "any error" {
			t.Error("expected an SdkError")
			return
		}
	})
}
