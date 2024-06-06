package helpers

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/threatwinds/logger"
)

type Env struct {
	ClusterPort     int
	RestPort        int
	GrpcPort        int
	Workdir         string
	RulesRepository string
	SearchNodes     []string
}

var env = new(Env)
var envOnce sync.Once

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

func getEnvInt(name string, def string, required bool) (int, *logger.Error) {
	str, e := getEnvStr(name, def, required)
	if e != nil {
		return 0, e
	}

	val, err := strconv.Atoi(str)
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

func GetEnv() *Env {
	envOnce.Do(func() {
		var e *logger.Error
		
		env.ClusterPort, e = getEnvInt("CLUSTER_PORT", "8082", false)
		if e != nil {
			os.Exit(1)
		}

		env.RestPort, e = getEnvInt("REST_PORT", "8080", false)
		if e != nil {
			os.Exit(1)
		}

		env.GrpcPort, e = getEnvInt("GRPC_PORT", "8081", false)
		if e != nil {
			os.Exit(1)
		}

		env.Workdir, e = getEnvStr("WORK_DIR", "", true)
		if e != nil {
			os.Exit(1)
		}

		env.SearchNodes, e = getEnvStrSlice("SEARCH_NODES", "", true)
		if e != nil {
			os.Exit(1)
		}

		env.RulesRepository, e = getEnvStr("RULES_REPOSITORY", "", true)
		if e != nil {
			os.Exit(1)
		}
	})

	return env
}
