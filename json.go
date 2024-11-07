package go_sdk

import (
	"encoding/json"
	"os"

	"github.com/threatwinds/logger"
)


// ReadJSON reads a JSON file and unmarshals its content into a specified type.
// The function takes a file path as input and returns a pointer to the unmarshaled
// value of the specified type and a pointer to a logger.Error if an error occurs.
//
// Type Parameters:
//   t: The type into which the JSON content should be unmarshaled.
//
// Parameters:
//   f: The file path of the JSON file to be read.
//
// Returns:
//   *t: A pointer to the unmarshaled value of the specified type.
//   *logger.Error: A pointer to a logger.Error if an error occurs, otherwise nil.
func ReadJSON[t any](f string) (*t, *logger.Error) {
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, Logger().ErrorF("error opening file '%s': %s", f, err.Error())
	}

	var value = new(t)

	err = json.Unmarshal(content, value)
	if err != nil {
		return nil, Logger().ErrorF("error reading JSON file '%s': %s", f, err.Error())
	}

	return value, nil
}
