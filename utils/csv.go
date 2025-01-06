package utils

import (
	"encoding/csv"
	"github.com/threatwinds/go-sdk/catcher"
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
		return nil, catcher.Error("error opening CSV file", err, map[string]any{"file": url})
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	result, err := reader.ReadAll()
	if err != nil {
		return nil, catcher.Error("error reading CSV file", err, map[string]any{"file": url})
	}

	return result, nil
}
