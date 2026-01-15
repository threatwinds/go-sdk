package plugins

import "os"

var WorkDir = func() string {
	workDir := os.Getenv("WORK_DIR")

	if workDir == "" {
		workDir = "/workdir"
	}

	return workDir
}()

const lockFile string = "config.lock"
