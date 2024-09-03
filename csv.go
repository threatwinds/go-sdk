package go_sdk

import (
	"encoding/csv"
	"os"

	"github.com/threatwinds/logger"
)

func ReadCSV(url string) ([][]string, *logger.Error) {
	f, err := os.Open(url)
	if err != nil {
		Logger().ErrorF("error opening file: %s", err.Error())
	}
	defer f.Close()

	r := csv.NewReader(f)
	result, err := r.ReadAll()
	if err != nil {
		Logger().ErrorF("error reading csv file: %s", err.Error())
	}

	return result, nil
}
