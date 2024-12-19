package go_sdk

import (
	"encoding/csv"
	"os"
)

// ReadCSV reads a CSV file from the given URL and returns its contents as a slice of string slices.
// If an error occurs while opening or reading the file, it logs the error and returns nil.
//
// Parameters:
//   - url: The path to the CSV file.
//
// Returns:
//   - [][]string: The contents of the CSV file.
//   - error: An error object if an error occurs, otherwise nil.
func ReadCSV(url string) ([][]string, error) {
	file, err := os.Open(url)
	if err != nil {
		return nil, Error(Trace(), map[string]interface{}{
			"cause": err,
			"file":  url,
			"error": "error opening CSV file",
		})
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	result, err := reader.ReadAll()
	if err != nil {
		return nil, Error(Trace(), map[string]interface{}{
			"cause": err,
			"file":  url,
			"error": "error reading CSV file",
		})
	}

	return result, nil
}
