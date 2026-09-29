package catcher

import (
	"fmt"
	"log"
	"log/slog"
	"testing"
)

func BenchmarkCatcherInfoAsyncNoTrace(b *testing.B) {
	benchToDevNull(b, false, true, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", map[string]any{"key": "value"})
	}
	b.StopTimer()
}

func BenchmarkCatcherErrorAsyncNoTrace(b *testing.B) {
	benchToDevNull(b, false, true, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Error("benchmark message", nil, map[string]any{"key": "value"})
	}
	b.StopTimer()
}

func BenchmarkCatcherInfoSyncNoTrace(b *testing.B) {
	benchToDevNull(b, false, false, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", map[string]any{"key": "value"})
	}
}

func BenchmarkCatcherErrorSyncNoTrace(b *testing.B) {
	benchToDevNull(b, false, false, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Error("benchmark message", nil, map[string]any{"key": "value"})
	}
}

func BenchmarkCatcherInfoSyncWithTrace(b *testing.B) {
	benchToDevNull(b, false, false, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", map[string]any{"key": "value"})
	}
}

func BenchmarkCatcherErrorSyncWithTrace(b *testing.B) {
	benchToDevNull(b, false, false, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Error("benchmark message", nil, map[string]any{"key": "value"})
	}
}

func BenchmarkSlogJSON(b *testing.B) {
	// To be fair to catcher, which writes to a discarded stdout, slog writes
	// to a discarded destination too.
	logger := slog.New(slog.NewJSONHandler(devNull(b), nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkStandardLog(b *testing.B) {
	originalOutput := log.Writer()
	log.SetOutput(devNull(b))
	b.Cleanup(func() { log.SetOutput(originalOutput) })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Println("benchmark message key=value")
	}
}

func deepCallCatcher(level int) error {
	if level <= 0 {
		return Error("base error", nil, map[string]any{"level": 0})
	}
	err := deepCallCatcher(level - 1)
	return Error("propagated error", err, map[string]any{"level": level})
}

func deepCallSlog(level int, logger *slog.Logger) error {
	if level <= 0 {
		return fmt.Errorf("base error")
	}
	err := deepCallSlog(level-1, logger)
	return fmt.Errorf("level %d: %w", level, err)
}

func BenchmarkCatcherNestedErrors3(b *testing.B) {
	benchToDevNull(b, false, true, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deepCallCatcher(3)
	}
	b.StopTimer()
}

func BenchmarkCatcherNestedErrors6(b *testing.B) {
	benchToDevNull(b, false, true, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deepCallCatcher(6)
	}
	b.StopTimer()
}

func BenchmarkSlogNestedErrors3(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(devNull(b), nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := deepCallSlog(3, logger)
		logger.Error("top error", "error", err)
	}
}

func BenchmarkSlogNestedErrors6(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(devNull(b), nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := deepCallSlog(6, logger)
		logger.Error("top error", "error", err)
	}
}

func BenchmarkCatcherInfoAsyncParallel(b *testing.B) {
	benchToDevNull(b, false, true, true)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Info("parallel info message", map[string]any{
				"worker_id": i % 10,
				"iteration": i,
				"status":    "active",
			})
			i++
		}
	})
	b.StopTimer()
}

func BenchmarkCatcherErrorAsyncParallel(b *testing.B) {
	benchToDevNull(b, false, true, true)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = Error("parallel error message", nil, map[string]any{
				"worker_id": i % 10,
				"iteration": i,
				"severity":  "high",
			})
			i++
		}
	})
	b.StopTimer()
}

func BenchmarkCatcherInfoSyncParallel(b *testing.B) {
	benchToDevNull(b, false, false, true)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Info("parallel info message", map[string]any{
				"worker_id": i % 10,
				"iteration": i,
				"status":    "active",
			})
			i++
		}
	})
}

func BenchmarkCatcherErrorSyncParallel(b *testing.B) {
	benchToDevNull(b, false, false, true)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = Error("parallel error message", nil, map[string]any{
				"worker_id": i % 10,
				"iteration": i,
				"severity":  "high",
			})
			i++
		}
	})
}

func BenchmarkSlogJSONParallel(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(devNull(b), nil))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.Info("parallel info message",
				"worker_id", i%10,
				"iteration", i,
				"status", "active")
			i++
		}
	})
}
