//go:build debug

package plugins

import "os"

var WorkDir string = os.Getenv("WORK_DIR")

const lockFile string = "config.lock"
