package entities

import (
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ValidateCountry validates a given country string value by converting it to title case and generating a SHA3-256 hash.
// Returns the validated country string value, its SHA3-256 hash, and an error if the value is not a string.
func ValidateCountry(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	v = cases.Title(language.English).String(strings.ToLower(v))

	return v, GenerateSHA3256(v), nil
}
