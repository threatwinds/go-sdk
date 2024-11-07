package go_sdk

import (
	"encoding/csv"
	"os"

	"github.com/threatwinds/logger"
)

// ReadCSV reads a CSV file from the given URL and returns its contents as a slice of string slices.
// If an error occurs while opening or reading the file, it logs the error and returns nil.
//
// Parameters:
//   - url: The path to the CSV file.
//
// Returns:
//   - [][]string: The contents of the CSV file.
//   - *logger.Error: An error object if an error occurs, otherwise nil.
func ReadCSV(url string) ([][]string, *logger.Error) {
	f, err := os.Open(url)
	if err != nil {
		Logger().ErrorF("error opening file '%s': %s", url, err.Error())
	}
	defer f.Close()

	r := csv.NewReader(f)
	result, err := r.ReadAll()
	if err != nil {
		Logger().ErrorF("error reading CSV file '%s': %s", url, err.Error())
	}

	return result, nil
}
