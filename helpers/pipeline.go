package helpers

import (
	"path"
	"sync"
	"time"
)

type Config struct {
	Groks         []Grok                            `yaml:"groks,omitempty"`
	Trims         []Trim                            `yaml:"trims,omitempty"`
	Renames       []Rename                          `yaml:"renames,omitempty"`
	Casts         []Cast                            `yaml:"casts,omitempty"`
	Deletes       []Delete                          `yaml:"deletes,omitempty"`
	Tenants       []Tenant                          `yaml:"tenants,omitempty"`
	DisabledRules []int64                           `yaml:"disabled_rules,omitempty"`
	Plugins       map[string]map[string]interface{} `yaml:"plugins,omitempty"`
	Env           Env                               `yaml:"-"`
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

		c.Groks = append(c.Groks, nCfg.Groks...)
		c.Trims = append(c.Trims, nCfg.Trims...)
		c.Renames = append(c.Renames, nCfg.Renames...)
		c.Casts = append(c.Casts, nCfg.Casts...)
		c.Deletes = append(c.Deletes, nCfg.Deletes...)
		c.Tenants = append(c.Tenants, nCfg.Tenants...)
		c.DisabledRules = append(c.DisabledRules, nCfg.DisabledRules...)

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
