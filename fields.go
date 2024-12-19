package go_sdk

import (
	"log"
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
		return Error(Trace(), map[string]interface{}{
			"error": "error validating field",
			"cause": "field cannot be empty",
		})
	}

	for _, rf := range reservedFields {
		if f == rf {
			return Error(Trace(), map[string]interface{}{
				"error":          "error validating field",
				"cause":          "field cannot be a reserved field",
				"reservedFields": reservedFields,
				"usedField":      f,
			})
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
		log.Println(Error(Trace(), map[string]interface{}{
			"advise": "consider to review the error cause",
			"cause":  err.Error(),
			"error":  "error compiling regexp",
		}))
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
