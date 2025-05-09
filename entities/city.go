package entities

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ValidateCity validates a city name by converting it to title case and generating a SHA3-256 hash.
// It takes a value of type interface{} and returns the validated city name, its SHA3-256 hash, and an error (if any).
func ValidateCity(value string) (string, string, error) {
	v := cases.Title(language.English).String(strings.ToLower(value))

	return v, GenerateSHA3256(v), nil
}
