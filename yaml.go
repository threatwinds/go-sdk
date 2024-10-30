package go_sdk

import (
	"os"

	"github.com/threatwinds/logger"
	"gopkg.in/yaml.v3"
	k8syaml "sigs.k8s.io/yaml"
)

func ReadPbYaml(f string) ([]byte, *logger.Error) {
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, Logger().ErrorF("error opening file '%s': %s", f, err.Error())
	}

	bytes, err := k8syaml.YAMLToJSON(content)
	if err != nil {
		return nil, Logger().ErrorF("error converting YAML file '%s' to JSON: %s", f, err.Error())
	}

	return bytes, nil
}

func ReadYaml[t any](f string, jsonMode bool) (*t, *logger.Error) {
	content, err := os.ReadFile(f)
	if err != nil {
		return nil, Logger().ErrorF("error opening file '%s': %s", f, err.Error())
	}

	var value = new(t)
	if jsonMode {
		err = k8syaml.Unmarshal(content, value)
		if err != nil {
			return nil, Logger().ErrorF("error decoding YAML file '%s': %s", f, err.Error())
		}
	} else {
		err = yaml.Unmarshal(content, value)
		if err != nil {
			return nil, Logger().ErrorF("error decoding YAML file '%s': %s", f, err.Error())
		}
	}

	return value, nil
}
