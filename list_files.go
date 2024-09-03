package go_sdk

import (
	"os"
	"path/filepath"

	"github.com/threatwinds/logger"
)

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
		Logger().ErrorF("error listing files: %s", err.Error())
		if !logger.Is(err, "no such file or directory") {
			panic(err)
		}
	}

	return files
}
