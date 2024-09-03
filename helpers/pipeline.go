package helpers

type Pipeline struct {
	DataTypes []string `yaml:"data_types"`
	Steps     []Step   `yaml:"steps"`
}

type Step struct {
	Kv       *Kv       `yaml:"kv,omitempty"`
	Grok     *Grok     `yaml:"grok,omitempty"`
	Trim     *Trim     `yaml:"trim,omitempty"`
	Json     *Json     `yaml:"json,omitempty"`
	Csv      *Csv      `yaml:"csv,omitempty"`
	Rename   *Rename   `yaml:"rename,omitempty"`
	Cast     *Cast     `yaml:"cast,omitempty"`
	Reformat *Reformat `yaml:"reformat,omitempty"`
	Delete   *Delete   `yaml:"delete,omitempty"`
	Drop     *Drop     `yaml:"drop,omitempty"`
	Add      *Add      `yaml:"add,omitempty"`
	Dynamic  *Dynamic  `yaml:"dynamic,omitempty"`
}

type Dynamic struct {
	Plugin string        `yaml:"plugin"`
	Args   []interface{} `yaml:"args"`
}

type Reformat struct {
	Fields     []string `yaml:"fields"`
	Function   string   `yaml:"function"`
	FromFormat string   `yaml:"from_format"`
	ToFormat   string   `yaml:"to_format"`
}

type Grok struct {
	Patterns []Pattern `yaml:"patterns"`
}

type Pattern struct {
	FieldName string `yaml:"field_name"`
	Pattern   string `yaml:"pattern"`
}

type Kv struct {
	FieldSplit string `yaml:"field_split"`
	ValueSplit string `yaml:"value_split"`
}

type Json struct {
	Source string `yaml:"source"`
}

type Csv struct {
	Source    string   `yaml:"source"`
	Separator string   `yaml:"separator"`
	Headers   []string `yaml:"headers"`
}

type Trim struct {
	Function  string   `yaml:"function"`
	Substring string   `yaml:"substring"`
	Fields    []string `yaml:"fields"`
}

type Delete struct {
	Fields []string `yaml:"fields"`
}

type Rename struct {
	To   string   `yaml:"to"`
	From []string `yaml:"from"`
}

type Cast struct {
	To     string   `yaml:"to"`
	Fields []string `yaml:"fields"`
}

type Drop struct {
	Where Where `yaml:"where"`
}

type Add struct {
	Function string                 `yaml:"function"`
	Params   map[string]interface{} `yaml:"params"`
	Where    Where                  `yaml:"where"`
}

type Where struct {
	Variables  []Variable `yaml:"variables"`
	Expression string     `yaml:"expression"`
}

type Variable struct {
	Get    string `yaml:"get"`
	As     string `yaml:"as"`
	OfType string `yaml:"of_type"`
}
