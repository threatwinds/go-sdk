package plugins

import (
	"fmt"
	"github.com/threatwinds/go-sdk/utils"
	"path/filepath"
	"strings"
)

func GetOrderedSockets(t string) []string {
	var pList = make([]string, 0, 3)
	order := PluginCfg(t, false).Get("order").Array()

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
