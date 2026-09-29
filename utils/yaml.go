package utils

import (
	"fmt"
	"os"

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
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", f, err)
	}

	bytes, err := k8syaml.YAMLToJSON(content)
	if err != nil {
		return nil, fmt.Errorf("error converting YAML to JSON for file %s: %w", f, err)
	}

	return bytes, nil
}

// ReadYaml reads a YAML file and converts its content into a specified type.
// The function can also handle JSON mode if specified.
//
// Type Parameters:
//
//	t: The type into which the YAML content will be converted.
//
// Parameters:
//
//	f: The file path to the YAML file.
//	jsonMode: A boolean flag indicating whether to use JSON mode for conversion.
//
// Returns:
//
//	*t: A pointer to the converted content of type t.
//	error: A pointer to an error object if an error occurs, otherwise nil.
func ReadYaml[t any](f string, jsonMode bool) (*t, error) {
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", f, err)
	}

	var value = new(t)
	if jsonMode {
		err = k8syaml.Unmarshal(content, value)
		if err != nil {
			return nil, fmt.Errorf("error decoding file %s: %w", f, err)
		}
	} else {
		err = yaml.Unmarshal(content, value)
		if err != nil {
			return nil, fmt.Errorf("error decoding file %s: %w", f, err)
		}
	}

	return value, nil
}
