package go_sdk

import (
	"sync"

	"github.com/threatwinds/logger"
)

var loggerInstance *logger.Logger
var loggerOnce sync.Once

// Logger initializes a logger instance and returns it.
func Logger() *logger.Logger {
	loggerOnce.Do(func() {
		level := int(getEnv().LogLevel)
		// Add validation for log level
		if level < 0 {
			level = 0
		}
		loggerInstance = logger.NewLogger(&logger.Config{
			Level:   level,
			Format:  "text",
			Retries: 3,
			Wait:    5,
			StatusMap: map[int][]string{
				403: {
					"permission denied",
				},
				407: {
					"connection key",
					"unauthorized",
				},
				400: {
					"missing",
					"invalid",
				},
				100: {
					"no such file or directory",
					"signal: interrupt",
					"context canceled",
				},
			},
		})
	})

	return loggerInstance
}
