package plugins

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/threatwinds/go-sdk/utils"
)

type SocketType string

const (
	NotificationSocket SocketType = "notification"
	AnalysisSocket     SocketType = "analysis"
	CorrelationSocket  SocketType = "correlation"
)

func (t *SocketType) String() string {
	return string(*t)
}

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
