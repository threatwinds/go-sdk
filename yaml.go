package go_sdk

import (
	"encoding/json"
	"os"

	"github.com/threatwinds/logger"
	"sigs.k8s.io/yaml"
)

func ReadYAML[t any](f string) (*t, *logger.Error) {
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, Logger().ErrorF("error opening file: %s", err.Error())
	}

	var value = new(t)

	jsonData, err := yaml.YAMLToJSON(content)
	if err != nil {
		return nil, Logger().ErrorF("error converting YAML to JSON: %s", err.Error())
	}

	err = json.Unmarshal(jsonData, value)
	if err != nil {
		return nil, Logger().ErrorF("error reading YAML file: %s", err.Error())
	}

	return value, nil
}
