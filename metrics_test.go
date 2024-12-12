package go_sdk

import (
	"testing"
	"time"
)

func TestMetter(t *testing.T) {
	m := NewMetter("TestMetter")

	time.Sleep(1 * time.Second)

	t.Log(m.Elapsed("end"))
}
