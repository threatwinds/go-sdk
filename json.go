package go_sdk

import (
	"encoding/json"
	"os"

	"github.com/threatwinds/logger"
)

func ReadJSON[t any](f string) (*t, *logger.Error) {
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, Logger().ErrorF("error opening file: %s", err.Error())
	}

	var value = new(t)

	err = json.Unmarshal(content, value)
	if err != nil {
		return nil, Logger().ErrorF("error reading rule file: %s", err.Error())
	}

	return value, nil
}
