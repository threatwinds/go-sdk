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

// loadCfg loads configuration files from the "pipeline" directory within the working directory.
// It reads all YAML files, unmarshals them into Config objects, and merges their contents into the receiver Config object.
// The function updates the Pipeline, DisabledRules, Tenants, Patterns, and Plugins fields of the receiver Config object.
// If an error occurs while reading or unmarshalling a file, the function logs the error and continues with the next file.
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

// updateCfg updates the global configuration by loading new values
// into a temporary Config object and then replacing the current
// configuration with the new one. It ensures thread safety by using
// a mutex lock. After updating the configuration, it logs the new
// configuration state. If there is an error during the marshalling
// of the configuration, it logs the error.
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

// GetCfg initializes the configuration if it hasn't been initialized yet,
// and starts a goroutine to periodically update the configuration every 60 seconds.
// It waits for the initial configuration to be set before returning it.
// The function returns a pointer to the Config struct.
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
