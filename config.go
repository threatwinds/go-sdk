package go_sdk

import (
	"encoding/json"
	"path"
	"sync"
	"time"

	"github.com/threatwinds/logger"
	"google.golang.org/protobuf/encoding/protojson"
)

var cfg *Config
var cfgOnce sync.Once
var cfgMutex sync.RWMutex
var cfgFirst bool = true

func (c *Config) loadCfg() {
	cFiles := ListFiles(path.Join(getEnv().Workdir, "pipeline"), ".yaml")
	for _, cFile := range cFiles {
		var nCfg = new(Config)
		b, e := ReadPbYaml(cFile)
		if e != nil {
			continue
		}

		err := protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(b, nCfg)
		if err != nil {
			Logger().ErrorF("error decoding JSON from YAML file '%s': %s", cFile, err.Error())
			continue
		}

		c.Pipeline = append(c.Pipeline, nCfg.Pipeline...)

		c.DisabledRules = append(c.DisabledRules, nCfg.DisabledRules...)

		c.Tenants = append(c.Tenants, nCfg.Tenants...)

		for name, pattern := range nCfg.Patterns {
			c.Patterns[name] = pattern
		}

		for name, plugin := range nCfg.Plugins {
			c.Plugins[name] = plugin
		}
	}

	c.Env = getEnv()
}

func updateCfg() {
	cfgMutex.Lock()

	tmpCfg := new(Config)
	tmpCfg.Plugins = make(map[string]*Value)
	tmpCfg.Patterns = make(map[string]string)
	tmpCfg.loadCfg()

	*cfg = *tmpCfg

	cfgMutex.Unlock()

	cfgStr, err := protojson.Marshal(cfg)
	if err != nil {
		Logger().ErrorF("error marshalling config: %s", err.Error())
	}

	Logger().LogF(100, "config updated: %s", cfgStr)

	cfgFirst = false
}

func GetCfg() *Config {
	cfgOnce.Do(func() {
		cfg = new(Config)

		go func() {
			for {
				updateCfg()
				time.Sleep(60 * time.Second)
			}
		}()
	})

	for cfgFirst {
		time.Sleep(10 * time.Second)
	}

	cfgMutex.RLock()
	defer cfgMutex.RUnlock()

	return cfg
}

func PluginCfg[t any](name string) (*t, *logger.Error) {
	cfg := GetCfg()
	if cfg.Plugins[name] == nil {
		return nil, Logger().ErrorF("plugin %s not found", name)
	}

	tmpJson, err := protojson.Marshal(cfg.Plugins[name])
	if err != nil {
		return nil, Logger().ErrorF("error reading plugin config: %s", err.Error())
	}

	finalCfg := new(t)

	err = json.Unmarshal(tmpJson, finalCfg)
	if err != nil {
		return nil, Logger().ErrorF("error writing plugin config: %s", err.Error())
	}

	return finalCfg, nil
}
