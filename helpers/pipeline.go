package helpers

import (
	"path"
	"sync"
	"time"

	"github.com/threatwinds/logger"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Kv            []Kv                              `yaml:"kv,omitempty"`
	Grok          []Grok                            `yaml:"grok,omitempty"`
	Trim          []Trim                            `yaml:"trim,omitempty"`
	Rename        []Rename                          `yaml:"rename,omitempty"`
	Cast          []Cast                            `yaml:"cast,omitempty"`
	Reformat      []Reformat                        `yaml:"reformat,omitempty"`
	Delete        []Delete                          `yaml:"delete,omitempty"`
	Tenants       []Tenant                          `yaml:"tenants,omitempty"`
	Drop          []Drop                            `yaml:"drop,omitempty"`
	Add           []Add                             `yaml:"add,omitempty"`
	Patterns      map[string]string                 `yaml:"patterns,omitempty"`
	DisabledRules []int64                           `yaml:"disabled_rules,omitempty"`
	Plugins       map[string]map[string]interface{} `yaml:"plugins,omitempty"`
	Env           Env                               `yaml:"-"`
}

type Reformat struct {
	DataTypes  []string `yaml:"data_types"`
	Fields     []string `yaml:"fields"`
	Function   string   `yaml:"function"`
	FromFormat string   `yaml:"from_format"`
	ToFormat   string   `yaml:"to_format"`
}

type Asset struct {
	Name            string   `yaml:"name"`
	Hostnames       []string `yaml:"hostnames"`
	IPs             []string `yaml:"ips"`
	Confidentiality int32    `yaml:"confidentiality"`
	Availability    int32    `yaml:"availability"`
	Integrity       int32    `yaml:"integrity"`
}

type Grok struct {
	DataTypes []string  `yaml:"data_types"`
	Patterns  []Pattern `yaml:"patterns"`
}

type Pattern struct {
	FieldName string `yaml:"field_name"`
	Pattern   string `yaml:"pattern"`
}

type Kv struct {
	DataTypes  []string   `yaml:"data_types"`
	Properties []Property `yaml:"properties"`
}

type Property struct {
	FieldSplit string `yaml:"field_split"`
	ValueSplit string `yaml:"value_split"`
}

type Trim struct {
	DataTypes []string `yaml:"data_types"`
	Function  string   `yaml:"function"`
	Substring string   `yaml:"substring"`
	Fields    []string `yaml:"fields"`
}

type Delete struct {
	DataTypes []string `yaml:"data_types"`
	Fields    []string `yaml:"fields"`
}

type Rename struct {
	DataTypes []string `yaml:"data_types"`
	To        string   `yaml:"to"`
	From      []string `yaml:"from"`
}

type Cast struct {
	DataTypes []string `yaml:"data_types"`
	To        string   `yaml:"to"`
	Fields    []string `yaml:"fields"`
}

type Drop struct {
	DataTypes []string `yaml:"data_types"`
	Where     Where    `yaml:"where"`
}

type Add struct {
	DataTypes []string               `yaml:"data_types"`
	Function  string                 `yaml:"function"`
	Params    map[string]interface{} `yaml:"params"`
	Where     Where                  `yaml:"where"`
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

type Tenant struct {
	Name          string  `yaml:"name"`
	Id            string  `yaml:"id"`
	Assets        []Asset `yaml:"assets"`
	DisabledRules []int64 `yaml:"disabled_rules"`
}

var cfg *Config
var cfgOnce sync.Once
var cfgMutex sync.RWMutex

func (c *Config) loadCfg() {
	cFiles := ListFiles(path.Join(getEnv().Workdir, "pipeline"), ".yaml")
	for _, cFile := range cFiles {
		nCfg, e := ReadYAML[Config](cFile)
		if e != nil {
			continue
		}

		c.Kv = append(c.Kv, nCfg.Kv...)
		c.Grok = append(c.Grok, nCfg.Grok...)
		c.Trim = append(c.Trim, nCfg.Trim...)
		c.Rename = append(c.Rename, nCfg.Rename...)
		c.Cast = append(c.Cast, nCfg.Cast...)
		c.Reformat = append(c.Reformat, nCfg.Reformat...)
		c.Delete = append(c.Delete, nCfg.Delete...)
		c.Tenants = append(c.Tenants, nCfg.Tenants...)
		c.Drop = append(c.Drop, nCfg.Drop...)
		c.Add = append(c.Add, nCfg.Add...)
		c.DisabledRules = append(c.DisabledRules, nCfg.DisabledRules...)

		for name, pattern := range nCfg.Patterns {
			c.Patterns[name] = pattern
		}

		for name, plugin := range nCfg.Plugins {
			c.Plugins[name] = plugin
		}
	}

	c.Env = getEnv()
}

func GetCfg() *Config {
	cfgOnce.Do(func() {
		cfg = new(Config)

		go func() {
			for {
				cfgMutex.Lock()

				tmpCfg := new(Config)
				tmpCfg.Plugins = make(map[string]map[string]interface{})
				tmpCfg.Patterns = make(map[string]string)
				tmpCfg.loadCfg()

				*cfg = *tmpCfg

				cfgMutex.Unlock()

				time.Sleep(60 * time.Second)
			}
		}()

		time.Sleep(5 * time.Second)
	})

	cfgMutex.RLock()
	defer cfgMutex.RUnlock()

	return cfg
}

func PluginCfg[t any](name string) (*t, *logger.Error) {
	cfg := GetCfg()
	if cfg.Plugins[name] == nil {
		return nil, Logger().ErrorF("plugin %s not found", name)
	}

	tmpYaml, err := yaml.Marshal(cfg.Plugins[name])
	if err != nil {
		return nil, Logger().ErrorF("error reading plugin config: %s", err.Error())
	}

	finalCfg := new(t)

	err = yaml.Unmarshal(tmpYaml, finalCfg)
	if err != nil {
		return nil, Logger().ErrorF("error writing plugin config: %s", err.Error())
	}

	return finalCfg, nil
}
