package helpers

import (
	"sync"

	"github.com/threatwinds/logger"
)

var loggerInstance *logger.Logger
var loggerOnce sync.Once

func NewLogger(level int) *logger.Logger {
	loggerOnce.Do(func() {
		loggerInstance = logger.NewLogger(&logger.Config{
			Level:     level,
			Format:    "text",
			Retries:   3,
			Wait:      5,
			StatusMap: map[int][]string{
				400: {"missing", "invalid"},
				100: {"no such file or directory"},
			},
		})
	})

	return loggerInstance
}

func Logger() *logger.Logger {
	if loggerInstance == nil {
		return NewLogger(200)
	}

	return loggerInstance
}
