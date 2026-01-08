package utils

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
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

var compiledPattern *regexp.Regexp
var compiledPatternOnce sync.Once

// SanitizeField removes all non-alphanumeric characters from a string
func SanitizeField(s *string) {
	const exp string = "[^a-zA-Z0-9.]"

	m := NewMeter("SanitizeField")
	defer m.Elapsed("finished")

	compiledPatternOnce.Do(func() {
		compiledPattern, _ = regexp.Compile(exp)
		if compiledPattern == nil {
			panic("failed to compile pattern")
		}
	})

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
