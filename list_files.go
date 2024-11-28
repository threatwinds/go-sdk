package go_sdk

import (
	"os"
	"path/filepath"

	"github.com/threatwinds/logger"
)

// ListFiles walks through the directory specified by the route and returns a slice of file paths
// that match the given filter. The filter should be a file extension (e.g., ".txt").
//
// Parameters:
//   - route: The root directory to start the file search.
//   - filter: The file extension to filter files by.
//
// Returns:
//   - A slice of strings containing the paths of the files that match the filter.
//
// If an error occurs during the file walk, it logs the error and panics if the error is not
// "no such file or directory".
func ListFiles(route string, filter string) []string {
	var files []string

	err := filepath.Walk(route, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == filter {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		if !logger.Is(err, "no such file or directory") {
			panic(err)
		}
	}

	return files
}
