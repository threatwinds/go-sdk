package plugins

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/threatwinds/go-sdk/utils"
)

// SocketType represents the type of plugin socket used for gRPC communication.
// It is used to compose socket filenames and to look up plugin ordering in config.
// Valid values are defined in the constants below.

type SocketType string

const (
	// NotificationSocket identifies Notification plugins sockets: <name>_notification.sock
	NotificationSocket SocketType = "notification"
	// AnalysisSocket identifies Analysis plugins sockets: <name>_analysis.sock
	AnalysisSocket SocketType = "analysis"
	// CorrelationSocket identifies Correlation plugins sockets: <name>_correlation.sock
	CorrelationSocket SocketType = "correlation"
)

// String returns the string representation of the SocketType.
func (t *SocketType) String() string {
	return string(*t)
}

// GetOrderedSockets returns an ordered list of socket file paths for the given
// socket type, based on the plugin order specified in configuration (env/config).
// If no configuration is present, an empty list is returned.
func GetOrderedSockets(t SocketType) []string {
	var pList = make([]string, 0, 3)
	cfg := PluginCfg(t.String())
	if !cfg.Exists() {
		return pList
	}
	order := cfg.Get("order").Array()

	for _, name := range order {
		pList = append(pList, filepath.Join(
			WorkDir, "sockets",
			fmt.Sprintf("%s_%s.sock", name.String(), t)))
	}

	return pList
}

// GetParsingSockets scans the sockets directory and returns a map of parsing
// plugin names to their corresponding socket file paths.
func GetParsingSockets() map[string]string {
	files := utils.ListFiles(
		filepath.Join(WorkDir, "sockets"), ".sock")

	var pList = make(map[string]string, 3)

	for _, f := range files {
		if strings.HasSuffix(f, "_parsing.sock") {
			pList[GetPluginName(f, "_")] = f
		}
	}

	return pList
}
