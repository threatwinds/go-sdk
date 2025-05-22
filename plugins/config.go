package plugins

import (
	"fmt"
	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/utils"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protojson"
)

var cfg *Config
var cfgOnce sync.Once
var cfgMutex sync.RWMutex

const WorkDir string = "/workdir"
const lockFile string = "config.lock"

// AcquireLock tries to acquire the lock file to prevent race conditions
// when loading or modifying configurations. It returns true if the lock
// was acquired successfully, false otherwise.
func AcquireLock() (bool, error) {
	lockPath := filepath.Join(WorkDir, lockFile)

	// Check if lock file exists
	if _, err := os.Stat(lockPath); err == nil {
		// Lock file exists, cannot acquire lock
		return false, nil
	} else if !os.IsNotExist(err) {
		// Error checking lock file
		return false, fmt.Errorf("error checking lock file: %v", err)
	}

	// Create lock file
	file, err := os.Create(lockPath)
	if err != nil {
		return false, fmt.Errorf("error creating lock file: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Write process ID to lock file for debugging purposes
	_, err = fmt.Fprintf(file, "%d", os.Getpid())
	if err != nil {
		// Try to remove the lock file if we couldn't write to it
		_ = os.Remove(lockPath)
		return false, fmt.Errorf("error writing to lock file: %v", err)
	}

	return true, nil
}

// ReleaseLock releases the lock file.
func ReleaseLock() error {
	lockPath := filepath.Join(WorkDir, lockFile)
	return os.Remove(lockPath)
}

// loadCfg loads configuration files from the "pipeline" directory within the working directory.
// It reads all YAML files, decodes them into Config objects, and merges their contents into the receiver Config object.
// The function updates the Pipeline, DisabledRules, Tenants, Patterns, and Plugins fields of the receiver Config object.
// If an error occurs while reading or unmarshalling a file, the function logs the error and continues with the next file.
func (c *Config) loadCfg() {
	pipelineFolder, err := utils.MkdirJoin(WorkDir, "pipeline")
	if err != nil {
		_ = catcher.Error("failed to create pipeline folder", err, map[string]interface{}{"dir": pipelineFolder})
		os.Exit(1)
	}

	cFiles := utils.ListFiles(pipelineFolder.String(), ".yaml")
	for _, cFile := range cFiles {
		var nCfg = new(Config)
		b, err := utils.ReadPbYaml(cFile)
		if err != nil {
			_ = catcher.Error("error reading YAML file", err, map[string]interface{}{"file": cFile})
			continue
		}

		err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(b, nCfg)
		if err != nil {
			_ = catcher.Error("error reading YAML file", err, map[string]interface{}{"file": cFile})
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

// RandomDuration returns a random time.Duration between min and max seconds. It panics if max <= 0.
func RandomDuration(min, max int) time.Duration {
	source := rand.NewSource(time.Now().UnixNano())

	r := rand.New(source)

	randomNumber := r.Intn(max)
	if randomNumber < min {
		randomNumber = min
	}

	return time.Duration(randomNumber) * time.Second
}

// updateCfg updates the global configuration by loading new values
// into a temporary Config object and then replacing the current
// configuration with the new one. It ensures thread safety by using
// a mutex lock and a lockfile mechanism to prevent race conditions
// with other components that might modify the configuration.
func updateCfg() {
	// Try to acquire the lock
	maxRetries := 5

	for i := 0; i < maxRetries; i++ {
		acquired, err := AcquireLock()
		if err != nil {
			_ = catcher.Error("failed to acquire lock", err, map[string]interface{}{"retry": i + 1})
		}

		if acquired {
			break
		}

		// Lock not acquired, wait and retry
		if i < maxRetries-1 {
			time.Sleep(RandomDuration(10, 60))
		} else {
			_ = catcher.Error("failed to acquire lock after multiple retries", nil, nil)
			return
		}
	}

	defer func() {
		// Release the lock when done
		if err := ReleaseLock(); err != nil {
			_ = catcher.Error("failed to release lock", err, nil)
		}
	}()

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
		time.Sleep(120 * time.Second)
	}

	cfgMutex.RLock()
	defer cfgMutex.RUnlock()

	return cfg
}

// PluginCfg retrieves the configuration for a specified plugin by name and unmarshal it into the provided type.
// The function returns a pointer to the configuration of the specified type and a pointer to an error if any error occurs.
//
// Parameters:
//
//	pluginName: The name of the plugin whose configuration is to be retrieved.
//	wait: A boolean value that determines whether the function should wait for the configuration to be available.
//
// Returns:
//
//	gjson.Result: An object containing the configuration of the specified plugin.
func PluginCfg(pluginName string, wait bool) gjson.Result {
	for {
		cfg := GetCfg()

		pConfig, ok := cfg.Plugins[pluginName]
		if !ok {
			if wait {
				time.Sleep(1 * time.Second)
				continue
			}

			panic("plugin config not found")
		}

		bJson, err := protojson.Marshal(pConfig)
		if err != nil {
			if wait {
				time.Sleep(1 * time.Second)
				continue
			}

			panic(err)
		}

		pJson := gjson.ParseBytes(bJson)

		return pJson
	}
}
