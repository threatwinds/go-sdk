package go_sdk

import (
	"errors"
	"regexp"
	"strings"
)

// ValidateReservedField validates a field to ensure it is not empty or a reserved field.
func ValidateReservedField(f string, allowEmpty bool) error {
	var reservedFields = []string{
		"raw",
		"dataType",
		"@timestamp",
		"dataSource",
	}

	if f == "" && !allowEmpty {
		return Error("error validating field", errors.New("field name cannot be empty"), nil)
	}

	for _, rf := range reservedFields {
		if f == rf {
			return Error("error validating field", errors.New("field cannot be a reserved field"),
				map[string]any{"reservedFields": reservedFields, "usedField": f})
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
		_ = Error("error compiling regexp", err, nil)
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
