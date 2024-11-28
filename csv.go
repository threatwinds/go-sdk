package go_sdk

import (
	"encoding/csv"
	"fmt"
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
	f, err := os.Open(url)
	if err != nil {
		return nil, fmt.Errorf("error opening file '%s': %s", url, err.Error())
	}
	defer f.Close()

	r := csv.NewReader(f)
	result, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV file '%s': %s", url, err.Error())
	}

	return result, nil
}
