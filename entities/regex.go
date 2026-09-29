package entities

import (
	"fmt"
	"regexp"
)

// ValidateRegEx validates if a given value matches a regular expression.
// It returns an error if the value does not match the expression.
func ValidateRegEx(regex, value string) error {
	expression := regexp.MustCompile(regex)
	matches := expression.FindAllString(value, -1)
	if len(matches) != 1 {
		return fmt.Errorf("value '%s' does not match with regexp '%s'", value, regex)
	}

	if matches[0] != value {
		return fmt.Errorf("value '%s' does not match with regexp '%s'", value, regex)
	}

	return nil
}

// ValidateRegexComp validates if a given value is a valid regular expression.
// It returns the validated value, its SHA3-256 hash, and an error if the value is not a string or is not a valid regular expression.
func ValidateRegexComp(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	_, err := regexp.Compile(v)
	if err != nil {
		return "", "", err
	}

	return v, GenerateSHA3256(v), nil
}
