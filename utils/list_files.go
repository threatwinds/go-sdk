package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ListFiles walks through the directory specified by the route and returns a slice of file paths
// that match the given filter, sorted in natural numeric order by filename. The filter should be
// a file extension (e.g., ".txt").
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
		if !strings.Contains(err.Error(), "no such file or directory") {
			panic(fmt.Errorf("cannot walk through directory %s: %w", route, err))
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return naturalLess(filepath.Base(files[i]), filepath.Base(files[j]))
	})

	return files
}

// naturalLess compares two strings using natural sort order, where numeric
// segments are compared by their numeric value rather than lexicographically.
// For example: "2.yaml" < "10.yaml" (unlike lexicographic where "10" < "2").
func naturalLess(a, b string) bool {
	for len(a) > 0 && len(b) > 0 {
		// Find the next numeric or non-numeric segment in each string
		aNum, aSegEnd := leadingNumber(a)
		bNum, bSegEnd := leadingNumber(b)

		if aSegEnd > 0 && bSegEnd > 0 {
			// Both start with a number — compare numerically
			if aNum != bNum {
				return aNum < bNum
			}
			a = a[aSegEnd:]
			b = b[bSegEnd:]
			continue
		}

		if aSegEnd > 0 != (bSegEnd > 0) {
			// One starts with a digit and the other doesn't — digits sort first
			return aSegEnd > 0
		}

		// Both start with non-digit characters — compare character by character
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a = a[1:]
		b = b[1:]
	}

	return len(a) < len(b)
}

// leadingNumber extracts a leading numeric segment from s.
// Returns the parsed number and the length of the numeric prefix.
// If s does not start with a digit, returns (0, 0).
func leadingNumber(s string) (int, int) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, 0
	}
	n, _ := strconv.Atoi(s[:i])
	return n, i
}
