package entities

import (
	"testing"
)

func TestValidateDate(t *testing.T) {
	validCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "valid date",
			input:    "2022-12-31",
			expected: "2022-12-31",
		},
		{
			name:     "valid date",
			input:    "2022-01-01",
			expected: "2022-01-01",
		},
	}

	invalidCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "invalid date",
			input:    "2022-02-30",
			expected: "",
		},
		{
			name:     "invalid date",
			input:    "2022-13-01",
			expected: "",
		},
		{
			name:     "invalid date",
			input:    "2022-01-32",
			expected: "",
		},
		{
			name:     "invalid date",
			input:    "2022-01-0",
			expected: "",
		},
		{
			name:     "invalid date",
			input:    "2022-01-",
			expected: "",
		},
		{
			name:     "invalid date",
			input:    "2022-01",
			expected: "",
		},
		{
			name:     "invalid date",
			input:    "2022",
			expected: "",
		},
		{
			name:     "invalid date",
			input:    "2022-01-01T00:00:00Z",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateDate(tc.input)
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
			_, _, err := ValidateDate(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}

func TestValidateDatetime(t *testing.T) {
	validCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "valid datetime",
			input:    "2022-12-31T23:59:59.999999999Z",
			expected: "2022-12-31T23:59:59.999999999Z",
		},
		{
			name:     "valid datetime",
			input:    "2022-01-01T00:00:00Z",
			expected: "2022-01-01T00:00:00Z",
		},
	}

	invalidCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "invalid datetime",
			input:    "2022-02-30T00:00:00Z",
			expected: "",
		},
		{
			name:     "invalid datetime",
			input:    "2022-13-01T00:00:00Z",
			expected: "",
		},
		{
			name:     "invalid datetime",
			input:    "2022-01-32T00:00:00Z",
			expected: "",
		},
		{
			name:     "invalid datetime",
			input:    "2022-01-0T00:00:00Z",
			expected: "",
		},
		{
			name:     "invalid datetime",
			input:    "2022-01-T00:00:00Z",
			expected: "",
		},
		{
			name:     "invalid datetime",
			input:    "2022-01-01T00:00:00",
			expected: "",
		},
		{
			name:     "invalid datetime",
			input:    "2022-01-01",
			expected: "",
		},
		{
			name:     "invalid datetime",
			input:    "2022",
			expected: "",
		},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, _, err := ValidateDatetime(tc.input)
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
			_, _, err := ValidateDatetime(tc.input)
			if err == nil {
				t.Fatalf("expected error, but got nil")
			}
		})
	}
}
