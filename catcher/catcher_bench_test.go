package catcher

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"testing"
)

func BenchmarkCatcherInfoAsyncNoTrace(b *testing.B) {
	Configure(false, true, true)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", map[string]any{"key": "value"})
	}
	b.StopTimer()
}

func BenchmarkCatcherErrorAsyncNoTrace(b *testing.B) {
	Configure(false, true, true)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Error("benchmark message", nil, map[string]any{"key": "value"})
	}
	b.StopTimer()
}

func BenchmarkCatcherInfoSyncNoTrace(b *testing.B) {
	Configure(false, false, true)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", map[string]any{"key": "value"})
	}
}

func BenchmarkCatcherErrorSyncNoTrace(b *testing.B) {
	Configure(false, false, true)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Error("benchmark message", nil, map[string]any{"key": "value"})
	}
}

func BenchmarkCatcherInfoSyncWithTrace(b *testing.B) {
	Configure(false, false, false)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", map[string]any{"key": "value"})
	}
}

func BenchmarkCatcherErrorSyncWithTrace(b *testing.B) {
	Configure(false, false, false)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Error("benchmark message", nil, map[string]any{"key": "value"})
	}
}

func BenchmarkSlogJSON(b *testing.B) {
	// Para ser justos con catcher que usa os.Stdout, slog debería usar os.Stdout redirigido a DevNull
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkStandardLog(b *testing.B) {
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	log.SetOutput(os.Stdout)
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
	Configure(false, true, true)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deepCallCatcher(3)
	}
	b.StopTimer()
}

func BenchmarkCatcherNestedErrors6(b *testing.B) {
	Configure(false, true, true)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deepCallCatcher(6)
	}
	b.StopTimer()
}

func BenchmarkSlogNestedErrors3(b *testing.B) {
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := deepCallSlog(3, logger)
		logger.Error("top error", "error", err)
	}
}

func BenchmarkSlogNestedErrors6(b *testing.B) {
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := deepCallSlog(6, logger)
		logger.Error("top error", "error", err)
	}
}

func BenchmarkCatcherInfoAsyncParallel(b *testing.B) {
	Configure(false, true, true)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

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
	Configure(false, true, true)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

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
	Configure(false, false, true)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

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
	Configure(false, false, true)
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

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
	originalStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = originalStdout }()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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
