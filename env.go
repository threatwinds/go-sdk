package go_sdk

import (
	"os"
	"strconv"
	"strings"

	"github.com/threatwinds/logger"
)

func getEnvStr(name, def string, required bool) (string, *logger.Error) {
	val := os.Getenv(name)

	if val == "" {
		if required {
			return "", Logger().ErrorF("configuration required: %s", name)
		} else {
			return def, nil
		}
	}

	return val, nil
}

func getEnvInt(name string, def string, required bool) (int64, *logger.Error) {
	str, e := getEnvStr(name, def, required)
	if e != nil {
		return 0, e
	}

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, Logger().ErrorF(err.Error())
	}

	return val, nil
}

func getEnvStrSlice(name, def string, required bool) ([]string, *logger.Error) {
	str, e := getEnvStr(name, def, required)
	if e != nil {
		return nil, e
	}

	var items = make([]string, 0, 1)
	for _, item := range strings.Split(str, ",") {
		items = append(items, strings.TrimSpace(item))
	}

	return items, nil
}

func getEnv() *Env {
	var env = new(Env)
	var e *logger.Error

	env.NodeName, e = getEnvStr("NODE_NAME", "", false)
	if e != nil {
		panic(e.Message)
	}

	env.NodeGroups, e = getEnvStrSlice("NODE_GROUPS", "", false)
	if e != nil {
		panic(e.Message)
	}

	env.Workdir, e = getEnvStr("WORK_DIR", "", true)
	if e != nil {
		panic(e.Message)
	}

	env.LogLevel, e = getEnvInt("LOG_LEVEL", "200", false)
	if e != nil {
		panic(e.Message)
	}

	return env
}
