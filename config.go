package go_sdk

import (
	"path"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protojson"
)

var cfg *Config
var cfgOnce sync.Once
var cfgMutex sync.RWMutex

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
}

// GetCfg initializes the configuration if it hasn't been initialized yet,
// and starts a goroutine to periodically update the configuration every 60 seconds.
// It waits for the initial configuration to be set before returning it.
// The function returns a pointer to the Config struct.
func GetCfg() *Config {
	var first bool
	
	cfgOnce.Do(func() {
		first = true
		cfg = new(Config)

		go func() {
			for {
				updateCfg()
				first = false
				time.Sleep(60 * time.Second)
			}
		}()
	})

	for first {
		time.Sleep(1 * time.Second)
	}

	cfgMutex.RLock()
	defer cfgMutex.RUnlock()

	return cfg
}

// PluginCfg retrieves the configuration for a specified plugin by name and unmarshals it into the provided type.
// The function returns a pointer to the configuration of the specified type and a pointer to a error if any error occurs.
//
// Parameters:
//
//	pluginName: The name of the plugin whose configuration is to be retrieved.
//
// Returns:
//
//	gjson.Result: An object containing the configuration of the specified plugin.
func PluginCfg(pluginName string) gjson.Result {
	cfg := GetCfg()

	pConfig, ok := cfg.Plugins[pluginName]
	if !ok {
		panic("plugin config not found")
	}

	bJson, err := protojson.Marshal(pConfig)
	if err != nil {
		panic(err)
	}

	pJson := gjson.ParseBytes(bJson)

	return pJson
}
