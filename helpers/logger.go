package helpers

import (
	"sync"

	"github.com/threatwinds/logger"
)

var loggerInstance *logger.Logger
var loggerOnce sync.Once

func Logger() *logger.Logger {
	loggerOnce.Do(func() {
		loggerInstance = logger.NewLogger(&logger.Config{
			Level:   getEnv().LogLevel,
			Format:  "text",
			Retries: 3,
			Wait:    5,
			StatusMap: map[int][]string{
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
