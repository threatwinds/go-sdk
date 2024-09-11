package go_sdk

type Pipeline struct {
	DataTypes []string `yaml:"dataTypes,omitempty"`
	Steps     []Step   `yaml:"steps,omitempty"`
}
