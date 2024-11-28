package go_sdk

import (
	"encoding/json"
	"fmt"
	"os"
)

// ReadJSON reads a JSON file and unmarshals its content into a specified type.
// The function takes a file path as input and returns a pointer to the unmarshaled
// value of the specified type and a pointer to an error if an error occurs.
//
// Type Parameters:
//
//	t: The type into which the JSON content should be unmarshaled.
//
// Parameters:
//
//	f: The file path of the JSON file to be read.
//
// Returns:
//
//	*t: A pointer to the unmarshaled value of the specified type.
//	error: An error object if any error occurs during the process.
func ReadJSON[t any](f string) (*t, error) {
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("error opening file '%s': %s", f, err.Error())
	}

	var value = new(t)

	err = json.Unmarshal(content, value)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file '%s': %s", f, err.Error())
	}

	return value, nil
}
