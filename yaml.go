package go_sdk

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	k8syaml "sigs.k8s.io/yaml"
)

// ReadPbYaml reads a YAML file, converts its content to JSON, and returns the JSON bytes.
// If an error occurs while reading the file or converting its content, it returns an error.
//
// Parameters:
//   - f: The file path of the YAML file to be read.
//
// Returns:
//   - []byte: The JSON bytes converted from the YAML file.
//   - error: An error object if an error occurs, otherwise nil.
func ReadPbYaml(f string) ([]byte, error) {
	// Add path validation
	if err := validateFilePath(f); err != nil {
		return nil, fmt.Errorf("invalid file path: %s", err)
	}

	content, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("error opening file '%s': %s", f, err.Error())
	}

	bytes, err := k8syaml.YAMLToJSON(content)
	if err != nil {
		return nil, fmt.Errorf("error converting YAML file '%s' to JSON: %s", f, err.Error())
	}

	return bytes, nil
}

// ReadYaml reads a YAML file and unmarshals its content into a specified type.
// The function can also handle JSON mode if specified.
//
// Type Parameters:
//
//	t: The type into which the YAML content will be unmarshaled.
//
// Parameters:
//
//	f: The file path to the YAML file.
//	jsonMode: A boolean flag indicating whether to use JSON mode for unmarshaling.
//
// Returns:
//
//	*t: A pointer to the unmarshaled content of type t.
//	error: A pointer to an error object if an error occurs, otherwise nil.
func ReadYaml[t any](f string, jsonMode bool) (*t, error) {
	// Add path validation
	if err := validateFilePath(f); err != nil {
		return nil, fmt.Errorf("invalid file path: %s", err)
	}

	content, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("error opening file '%s': %s", f, err.Error())
	}

	var value = new(t)
	if jsonMode {
		err = k8syaml.Unmarshal(content, value)
		if err != nil {
			return nil, fmt.Errorf("error decoding YAML file '%s': %s", f, err.Error())
		}
	} else {
		err = yaml.Unmarshal(content, value)
		if err != nil {
			return nil, fmt.Errorf("error decoding YAML file '%s': %s", f, err.Error())
		}
	}

	return value, nil
}

// Helper function to validate file paths
func validateFilePath(path string) error {
	// Add validation logic for file paths
	// Check for directory traversal attempts
	if strings.Contains(path, "..") {
		return errors.New("path contains invalid characters")
	}
	
	return nil
}
