package go_sdk

import (
	"os"
	"strconv"
	"strings"

	"github.com/threatwinds/logger"
)

// getEnvStr retrieves the value of the environment variable named by the key `name`.
// If the variable is not present and `required` is true, it returns an error indicating
// that the configuration is required. If the variable is not present and `required` is false,
// it returns the default value `def`.
//
// Parameters:
//   - name: The name of the environment variable to retrieve.
//   - def: The default value to return if the environment variable is not set and not required.
//   - required: A boolean indicating whether the environment variable is required.
//
// Returns:
//   - string: The value of the environment variable, or the default value if not set and not required.
//   - *logger.Error: An error if the environment variable is required but not set, otherwise nil.
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

// getEnvInt retrieves an environment variable as an integer.
// 
// Parameters:
// - name: The name of the environment variable.
// - def: The default value to use if the environment variable is not set.
// - required: A boolean indicating if the environment variable is required.
//
// Returns:
// - int64: The integer value of the environment variable.
// - *logger.Error: An error object if the environment variable is required but not set, or if the value cannot be parsed as an integer.
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

// getEnvStrSlice retrieves an environment variable as a slice of strings.
// The environment variable is expected to be a comma-separated list of values.
// If the environment variable is not set, the default value is used.
// If the environment variable is required and not set, an error is returned.
//
// Parameters:
//   - name: The name of the environment variable.
//   - def: The default value to use if the environment variable is not set.
//   - required: A boolean indicating if the environment variable is required.
//
// Returns:
//   - []string: A slice of strings obtained from the environment variable.
//   - *logger.Error: An error object if the environment variable is required but not set.
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

// getEnv initializes and returns an Env struct with values retrieved from environment variables.
// It retrieves the following environment variables:
// - NODE_NAME: The name of the node (string).
// - NODE_GROUPS: A comma-separated list of node groups (slice of strings).
// - WORK_DIR: The working directory (string).
// - LOG_LEVEL: The logging level (integer).
// If any required environment variable is missing or invalid, the function will panic with an error message.
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
