package utils

import (
	"encoding/json"
	"github.com/threatwinds/go-sdk/catcher"
	"os"
)

// ReadJSON reads a JSON file and parses its content into a specified type.
// The function takes a file path as input and returns a pointer to the parsed
// value of the specified type and a pointer to an error if an error occurs.
//
// Type Parameters:
//
//	t: The type into which the JSON content should be parsed.
//
// Parameters:
//
//	f: The file path of the JSON file to be read.
//
// Returns:
//
//	*t: A pointer to the parsed value of the specified type.
//	error: An error object if any error occurs during the process.
func ReadJSON[t any](f string) (*t, error) {
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, catcher.Error("error reading JSON file", err, map[string]any{"file": f})
	}

	var value = new(t)

	err = json.Unmarshal(content, value)
	if err != nil {
		return nil, catcher.Error("error parsing JSON file", err, map[string]any{"file": f})
	}

	return value, nil
}
