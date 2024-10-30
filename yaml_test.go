package go_sdk

import (
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
)

func TestReadPbYaml(t *testing.T) {
	t.Run("ReadPYAML", func(t *testing.T) {
		b, e := ReadPbYaml("test.yaml")
		if e != nil {
			t.Errorf("Expected nil, got %s", e.Message)
		}

		var value = new(Config)
		value.Plugins = make(map[string]*Value)
		value.Patterns = make(map[string]string)

		err := protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(b, value)
		if err != nil {
			t.Errorf("error decoding JSON from YAML file: %s", err.Error())
		}

		for _, step := range value.Pipeline[0].Steps{
			if step.Add != nil{
				t.Log(step.Add.Params["value"].GetStringValue())
			}
		}

		t.Log(value)
	})
}
