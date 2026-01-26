package plugins

import (
	"sync"
	"testing"
)

func TestRegexpCache_Get(t *testing.T) {
	cache := &RegexpCache{}

	t.Run("Basic compilation", func(t *testing.T) {
		pattern := `[a-z]+`
		re, err := cache.Get(pattern)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if re == nil {
			t.Fatal("Expected compiled regexp, got nil")
		}
		if re.String() != pattern {
			t.Errorf("Expected pattern %s, got %s", pattern, re.String())
		}
	})

	t.Run("Caching works", func(t *testing.T) {
		pattern := `\d+`
		re1, _ := cache.Get(pattern)
		re2, _ := cache.Get(pattern)
		if re1 != re2 {
			t.Error("Expected same pointer for cached pattern")
		}
	})

	t.Run("Invalid pattern", func(t *testing.T) {
		pattern := `[`
		_, err := cache.Get(pattern)
		if err == nil {
			t.Error("Expected error for invalid pattern, got nil")
		}
	})

	t.Run("Template expansion", func(t *testing.T) {
		// Setup global config patterns
		cfgMutex.Lock()
		oldCfg := cfg
		cfg = &Config{
			Patterns: map[string]string{
				"IP": `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`,
			},
		}
		cfgMutex.Unlock()

		defer func() {
			cfgMutex.Lock()
			cfg = oldCfg
			cfgMutex.Unlock()
		}()

		pattern := `{{.IP}}`
		re, err := cache.Get(pattern)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		expected := `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`
		if re.String() != expected {
			t.Errorf("Expected expanded pattern %s, got %s", expected, re.String())
		}

		// Test combination
		pattern2 := `src={{.IP}} dst={{.IP}}`
		re2, err := cache.Get(pattern2)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		expected2 := `src=\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3} dst=\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`
		if re2.String() != expected2 {
			t.Errorf("Expected expanded pattern %s, got %s", expected2, re2.String())
		}
	})

	t.Run("Template expansion missing pattern", func(t *testing.T) {
		cfgMutex.Lock()
		oldCfg := cfg
		cfg = &Config{
			Patterns: map[string]string{},
		}
		cfgMutex.Unlock()

		defer func() {
			cfgMutex.Lock()
			cfg = oldCfg
			cfgMutex.Unlock()
		}()

		pattern := `{{.MISSING}}`
		re, err := cache.Get(pattern)
		if err != nil {
			t.Fatalf("Expected no error (template expansion failure should fall back to raw pattern), got %v", err)
		}
		// If template expansion fails (missing key), it might return empty string or error in Execute
		// In regexp.go, if err != nil it keeps original pattern.
		// template.Execute with missing key in map usually results in "<no value>" or empty depending on options.
		// By default it might just be empty if not found in map.

		// Let's see what it actually does.
		t.Logf("Pattern for missing template: %s", re.String())
	})

	t.Run("Concurrency", func(t *testing.T) {
		pattern := `concurrent-[a-z]+`
		var wg sync.WaitGroup
		numGoroutines := 100
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				_, err := cache.Get(pattern)
				if err != nil {
					t.Errorf("Concurrent Get failed: %v", err)
				}
			}()
		}
		wg.Wait()
	})
}
