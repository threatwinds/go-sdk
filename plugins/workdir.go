package plugins

import "os"

// WorkDir is the folder on which the EventProcessor and all plugins are going to store their configuration files and temporary data
var WorkDir = func() string {
	workDir := os.Getenv("WORK_DIR")

	if workDir == "" {
		workDir = "/workdir"
	}

	return workDir
}() // This cannot be part of the main config system because the main config system depends on it to find the configuration files

const lockFile string = "config.lock"
