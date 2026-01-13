package utils

import (
	"testing"
	"time"
)

func TestMeter(t *testing.T) {
	m := NewMeter("TestMeter")

	time.Sleep(1 * time.Second)

	t.Log(m.Elapsed("end"))
}
