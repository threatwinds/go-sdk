package plugins

import "strings"

// GetPluginName extracts the base plugin name from a full path using the given separator.
// For example, given "/workdir/sockets/my-plugin_parsing.sock" and sep "_",
// it returns "my-plugin".
//
// Params:
//   - fullPath: full file path or identifier that contains the plugin name.
//   - sep: separator used to split suffixes (e.g., "_parsing", "_analysis").
//
// Returns:
//   - string: the plugin name without suffixes or extensions.
func GetPluginName(fullPath string, sep string) string {
	p := strings.Split(fullPath, "/")
	return strings.Split(p[len(p)-1], sep)[0]
}
