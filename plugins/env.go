package plugins

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
//   - error: An error if the environment variable is required but not set, otherwise nil.
func getEnvStr(name, def string, required bool) (string, error) {
	val := os.Getenv(name)

	if val == "" {
		if required {
			return "", fmt.Errorf("missing required environment variable: %s", name)
		}

		return def, nil
	}

	return val, nil
}

// getEnvUInt32 retrieves an environment variable as an integer.
//
// Parameters:
// - name: The name of the environment variable.
// - def: The default value to use if the environment variable is not set.
// - required: A boolean indicating if the environment variable is required.
//
// Returns:
// - int64: The integer value of the environment variable.
// - error: An error object if the environment variable is required but not set, or if the value cannot be parsed as an integer.
func getEnvUInt32(name string, def string, required bool) (uint32, error) {
	str, err := getEnvStr(name, def, required)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse UInt32 from %s: %w", name, err)
	}

	return uint32(val), nil
}

// getEnvStrSlice retrieves an environment variable as a slice of strings.
// The environment variable is expected to be a comma-separated list of values.
// If the environment variable is not set, the default value is used.
// If the environment variable is required and not set, an error is returned.
//
// Parameters:
//   - name: The name of the environment variable.
//   - def: The default value to use if the environment variable is unset.
//   - required: A boolean indicating if the environment variable is required.
//
// Returns:
//   - []string: A slice of strings obtained from the environment variable.
//   - error: An error object if the environment variable is required but not set.
func getEnvStrSlice(name, def string, required bool) ([]string, error) {
	str, err := getEnvStr(name, def, required)
	if err != nil {
		return nil, err
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
	var err error

	env.NodeName, err = getEnvStr("NODE_NAME", "", false)
	if err != nil {
		panic(err)
	}

	if env.NodeName == "" {
		env.NodeName, err = os.Hostname()
		if err != nil {
			panic(fmt.Sprintf("failed to get hostname: %v", err))
		}
	}

	env.NodeGroups, err = getEnvStrSlice("NODE_GROUPS", "default", false)
	if err != nil {
		panic(err)
	}

	env.LogLevel, err = getEnvUInt32("LOG_LEVEL", "200", false)
	if err != nil {
		panic(err)
	}

	env.Mode, err = getEnvStr("MODE", "", true)
	if err != nil {
		panic(err)
	}

	return env
}
