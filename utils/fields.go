package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateReservedField validates a field to ensure it is not empty or a reserved field.
func ValidateReservedField(f string, allowEmpty bool) error {
	var reservedFields = []string{
		"raw",
		"@timestamp",
	}

	if f == "" && !allowEmpty {
		return fmt.Errorf("error validating field: field name cannot be empty")
	}

	for _, rf := range reservedFields {
		if f == rf {
			return fmt.Errorf("error validating field: field cannot be a reserved field (reserved: %v, used: %s)", reservedFields, f)
		}
	}

	return nil
}

func SanitizeField(s *string) {
	const exp string = "[^a-zA-Z0-9.]"

	m := NewMeter("SanitizeField")
	defer m.Elapsed("finished")

	// compile the pattern
	compiledPattern, err := regexp.Compile(exp)
	if err != nil {
		return
	}

	// find the first match
	match := compiledPattern.FindAllString(*s, -1)
	if len(match) == 0 {
		return
	}

	for _, m := range match {
		// replace all occurrences of the match with an empty string
		*s = strings.ReplaceAll(*s, m, "")
	}
}
