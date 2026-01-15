package plugins

import (
	"sync"
	"testing"
)

var sCache = NewCELCache("cel_stress_test")

func TestEvaluateStress(t *testing.T) {
	data := `{"field1": "value1", "field2": 123, "field3": true}`
	expressions := []string{
		`exists("field1")`,
		`safe("field1", "default") == "value1"`,
		`equals("field2", 123)`,
		`safe("field3", false) == true`,
		`regexMatch("field1", "val.*")`,
	}

	numGoroutines := 100
	iterationsPerGoroutine := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				expr := expressions[j%len(expressions)]
				res, err := sCache.Evaluate(&data, expr)
				if err != nil {
					t.Errorf("Goroutine %d iteration %d: error evaluating %s: %v", id, j, expr, err)
					return
				}
				if !res {
					t.Errorf("Goroutine %d iteration %d: expected true for %s, got false", id, j, expr)
					return
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestEvaluateConcurrentCompilation(t *testing.T) {
	// This test forces multiple goroutines to attempt to compile the same exact expression
	// at the same time, to validate that the sharded locks system works.
	data := `{"field": "value"}`
	expr := `exists("field")`

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := sCache.Evaluate(&data, expr)
			if err != nil {
				t.Errorf("error evaluating: %v", err)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkEvaluateWithCache(b *testing.B) {
	data := `{"field1": "value1", "field2": 123}`
	expr := `exists("field1") && equals("field2", 123)`

	// First evaluation to warm up the cache
	_, _ = sCache.Evaluate(&data, expr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sCache.Evaluate(&data, expr)
	}
}
