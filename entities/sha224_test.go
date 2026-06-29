package entities

import (
	"testing"
)

func TestValidateSHA224(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid SHA-224",
			input:    "a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b",
			expected: "a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b",
		},
		{
			name:     "valid SHA-224",
			input:    "d1e959eb179c911faea4624c60c5ac703447b85d51844e5dc6f54c20",
			expected: "d1e959eb179c911faea4624c60c5ac703447b85d51844e5dc6f54c20",
		},
		{
			name:     "valid SHA-224",
			input:    "f71837502ba8e10837bdd8d365adb85591895602fc552b48b7390abd",
			expected: "f71837502ba8e10837bdd8d365adb85591895602fc552b48b7390abd",
		},
	}

	invalidCases := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid SHA-224",
			input: "a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434",
		},
		{
			name:  "invalid SHA-224",
			input: "d1e959eb179c911faea4624c60c5c702647b85d51844e5dc6f54c20e810c197",
		},
		{
			name:  "invalid SHA-224",
			input: "f71837502ba8e10837bdd8d365adb85591895602fc552b48b7390ab",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateSHA224(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, actual)
			}
		})
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ValidateSHA224(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
