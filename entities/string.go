package entities

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ValidateString validates a string value and returns the original value, its SHA3-256 hash and an error.
// If the insensitive flag is set to true, the value is converted to lowercase before hashing.
func ValidateString(value interface{}, insensitive bool) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	replacement, _ := utf8.DecodeRuneInString(" ")
	allowed := "\r\n \r \n"
	v = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) ||
			unicode.IsDigit(r) ||
			unicode.IsPunct(r) ||
			unicode.IsSpace(r) ||
			unicode.IsNumber(r) ||
			unicode.IsSymbol(r) {
			return r
		}

		for _, c := range allowed {
			if r == c {
				return r
			}
		}

		return replacement
	}, v)

	if !utf8.ValidString(v) {
		return "", "", fmt.Errorf("value is not an UTF-8 valid string")
	}

	if strings.TrimSpace(v) == "" {
		return "", "", fmt.Errorf("value cannot be empty")
	}

	if insensitive {
		return strings.ToLower(v), GenerateSHA3256(strings.ToLower(v)), nil
	}

	return v, GenerateSHA3256(v), nil
}
