package helpers

import (
	"path"
	"sync"
	"time"
)

type Config struct {
	Grok          []Grok                            `yaml:"groks,omitempty"`
	Trim          []Trim                            `yaml:"trims,omitempty"`
	Rename        []Rename                          `yaml:"renames,omitempty"`
	Cast          []Cast                            `yaml:"casts,omitempty"`
	Reformat      []Reformat                        `yaml:"reformats,omitempty"`
	Delete        []Delete                          `yaml:"deletes,omitempty"`
	Tenants       []Tenant                          `yaml:"tenants,omitempty"`
	Patterns      map[string]string                 `yaml:"patterns,omitempty"`
	DisabledRules []int64                           `yaml:"disabled_rules,omitempty"`
	Plugins       map[string]map[string]interface{} `yaml:"plugins,omitempty"`
	Env           Env                               `yaml:"-"`
}

type Reformat struct {
	DataTypes []string `yaml:"data_types"`
	Fields    []string `yaml:"fields"`
	Function  string   `yaml:"function"`
	Format    string   `yaml:"format"`
	ToFormat  string   `yaml:"to_format"`
}

type Asset struct {
	Name            string   `yaml:"name"`
	Hostnames       []string `yaml:"hostnames"`
	IPs             []string `yaml:"ips"`
	Confidentiality int      `yaml:"confidentiality"`
	Availability    int      `yaml:"availability"`
	Integrity       int      `yaml:"integrity"`
}

type Grok struct {
	DataTypes []string  `yaml:"data_types"`
	Patterns  []Pattern `yaml:"patterns"`
}

type Trim struct {
	DataTypes []string `yaml:"data_types"`
	Type      string   `yaml:"type"`
	Substring string   `yaml:"substring"`
	Fields    []string `yaml:"fields"`
}

type Delete struct {
	DataTypes []string `yaml:"data_types"`
	Fields    []string `yaml:"fields"`
}

type Pattern struct {
	FieldName string `yaml:"field_name"`
	Pattern   string `yaml:"pattern"`
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

		c.Grok = append(c.Grok, nCfg.Grok...)
		c.Trim = append(c.Trim, nCfg.Trim...)
		c.Rename = append(c.Rename, nCfg.Rename...)
		c.Cast = append(c.Cast, nCfg.Cast...)
		c.Reformat = append(c.Reformat, nCfg.Reformat...)
		c.Delete = append(c.Delete, nCfg.Delete...)
		c.Tenants = append(c.Tenants, nCfg.Tenants...)
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
