package helpers

type AlertRule struct {
	ID        int64  `yaml:"id"`
	Name      string `yaml:"name"`
	Impact    Impact `yaml:"impact"`
	Tactic    string `yaml:"tactic"`
	Technique string `yaml:"technique"`
	Query     Query  `yaml:"query"`
}

type Impact struct {
	Confidentiality int `json:"confidentiality"`
	Integrity       int `json:"integrity"`
	Availability    int `json:"availability"`
}
