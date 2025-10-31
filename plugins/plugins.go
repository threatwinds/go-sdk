package plugins

import "strings"

func GetPluginName(fullPath string, sep string) string {
	p := strings.Split(fullPath, "/")
	return strings.Split(p[len(p)-1], sep)[0]
}
