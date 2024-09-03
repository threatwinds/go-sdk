package go_sdk

type Pipeline struct {
	DataTypes []string `yaml:"dataTypes"`
	Steps     []Step   `yaml:"steps"`
}
