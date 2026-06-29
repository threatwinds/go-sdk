package entities

import (
	"fmt"
	"testing"
)

func TestValidateObject(t *testing.T) {
	validCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid UUID",
			input:    "123e4567-e89b-12d3-a456-426655440000",
			expected: "123e4567-e89b-12d3-a456-426655440000",
		},
		{
			name:     "valid MD5",
			input:    "d41d8cd98f00b204e9800998ecf8427e",
			expected: "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name:     "valid SHA3256",
			input:    "3D4F6B8A9ACD3303AD669B5E1392BAE394E6F3A7E0678FAF1EFA2DD42E396B77",
			expected: "3d4f6b8a9acd3303ad669b5e1392bae394e6f3a7e0678faf1efa2dd42e396b77",
		},
	}

	invalidCases := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid object",
			input: "invalid",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateIdentifier(tc.input)
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
			_, _, err := ValidateIdentifier(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
			expectedErr := fmt.Sprintf("invalid object: %v", tc.input)
			if err.Error() != expectedErr {
				t.Errorf("expected error message %q, but got %q", expectedErr, err.Error())
			}
		})
	}
}
