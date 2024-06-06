package helpers

import (
	"os"
	"path"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	Groks         []Grok                             `yaml:"groks,omitempty"`
	Trims         []Trim                             `yaml:"trims,omitempty"`
	Renames       []Rename                           `yaml:"renames,omitempty"`
	Casts         []Cast                             `yaml:"casts,omitempty"`
	Deletes       []Delete                           `yaml:"deletes,omitempty"`
	Tenants       []Tenant                           `yaml:"tenants,omitempty"`
	DisabledRules []int64                            `yaml:"disabled_rules,omitempty"`
	Plugins       *map[string]map[string]interface{} `yaml:"plugins,omitempty"`
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
	Name          string    `yaml:"name"`
	Id            uuid.UUID `yaml:"id"`
	Assets        []Asset   `yaml:"assets"`
	DisabledRules []int64   `yaml:"disabled_rules"`
}

var cfg *Config
var cfgOnce sync.Once

func (cfg *Config) loadCfg() {
	cFiles := ListFiles(path.Join(GetEnv().Workdir, "pipeline"), ".yaml")
	for _, cFile := range cFiles {
		nCfg, e := ReadYAML[Config](cFile)
		if e != nil {
			os.Exit(1)
		}

		cfg.Groks = append(cfg.Groks, nCfg.Groks...)
		cfg.Trims = append(cfg.Trims, nCfg.Trims...)
		cfg.Renames = append(cfg.Renames, nCfg.Renames...)
		cfg.Casts = append(cfg.Casts, nCfg.Casts...)
		cfg.Deletes = append(cfg.Deletes, nCfg.Deletes...)
		cfg.Tenants = append(cfg.Tenants, nCfg.Tenants...)
		cfg.DisabledRules = append(cfg.DisabledRules, nCfg.DisabledRules...)

		if nCfg.Plugins != nil {
			// merge plugins
			plugins := *cfg.Plugins
			for plugin, pCfg := range *nCfg.Plugins {
				plugins[plugin] = pCfg
			}
			*cfg.Plugins = plugins
		}
	}
}

func GetCfg() *Config {
	cfgOnce.Do(func() {
		cfg = new(Config)

		cfg.Plugins = new(map[string]map[string]interface{})
		*cfg.Plugins = make(map[string]map[string]interface{})

		cfg.loadCfg()

		go func() {
			for {
				time.Sleep(60 * time.Second)
				cfg.loadCfg()
			}
		}()
	})

	return cfg
}
