package plugins

import "testing"

func TestIsActionResult(t *testing.T) {
	for _, value := range []string{"", "success", "failed", "denied"} {
		if !IsActionResult(value) {
			t.Errorf("IsActionResult(%q) = false, want true", value)
		}
	}
	for _, value := range []string{
		"blocked", "failure", "accepted", "fail", "done", "error",
		"Success", "FAILED", "Denied", " success", "success ",
	} {
		if IsActionResult(value) {
			t.Errorf("IsActionResult(%q) = true, want false", value)
		}
	}
}

func TestIsUnsuccessfulActionResult(t *testing.T) {
	for _, value := range []string{"failed", "denied", "blocked"} {
		if !IsUnsuccessfulActionResult(value) {
			t.Errorf("IsUnsuccessfulActionResult(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"", "success", "failure", "accepted", "Failed", "DENIED"} {
		if IsUnsuccessfulActionResult(value) {
			t.Errorf("IsUnsuccessfulActionResult(%q) = true, want false", value)
		}
	}
}
