package helpers

import (
	"os"

	"github.com/threatwinds/logger"
	"gopkg.in/yaml.v3"
)

func ReadYAML[t any](f string) (*t, *logger.Error) {
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, Logger().ErrorF("error opening file: %s", err.Error())
	}

	var value = new(t)

	err = yaml.Unmarshal(content, value)
	if err != nil {
		return nil, Logger().ErrorF("error reading YAML file: %s", err.Error())
	}

	return value, nil
}
