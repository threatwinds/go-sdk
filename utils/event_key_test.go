package utils

import "testing"

func TestEventKeyDefaultsToEvent(t *testing.T) {
	t.Cleanup(func() { eventKey.Store(nil) })
	eventKey.Store(nil)
	if got := EventKey(); got != "event" {
		t.Fatalf("EventKey() = %q, want event", got)
	}
}

func TestSetEventKeyIsReadBackByEventKey(t *testing.T) {
	t.Cleanup(func() { eventKey.Store(nil) })
	for _, k := range []string{"log", "event"} {
		if err := SetEventKey(k); err != nil {
			t.Fatal(err)
		}
		if got := EventKey(); got != k {
			t.Errorf("EventKey() = %q, want %q", got, k)
		}
	}
}

func TestTheEventKeyCanComeFromTheEnvironment(t *testing.T) {
	t.Cleanup(func() { eventKey.Store(nil) })
	env := func(v string) func(string) string {
		return func(name string) string {
			if name != EventKeyEnv {
				t.Errorf("read %q, want %q", name, EventKeyEnv)
			}
			return v
		}
	}

	eventKey.Store(nil)
	if err := setEventKeyFromEnv(env("")); err != nil || EventKey() != "event" {
		t.Fatalf("unset must leave the default: %v, %q", err, EventKey())
	}
	if err := setEventKeyFromEnv(env("log")); err != nil || EventKey() != "log" {
		t.Fatalf("log: %v, %q", err, EventKey())
	}
	if err := setEventKeyFromEnv(env("nonsense")); err == nil {
		t.Fatal("a wrong value must be an error, not silently ignored")
	}
	if EventKey() != "log" {
		t.Errorf("a rejected value must leave the key as it was, got %q", EventKey())
	}
}
