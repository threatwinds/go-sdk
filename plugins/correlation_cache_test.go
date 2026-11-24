package plugins

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCorrelationCacheRemembersOnlyMatches(t *testing.T) {
	c := NewCorrelationCache(time.Minute, 10)
	var runs atomic.Int32
	docs := []json.RawMessage{json.RawMessage(`{"a":1}`)}

	hit := func() (bool, []json.RawMessage, error) { runs.Add(1); return true, docs, nil }
	miss := func() (bool, []json.RawMessage, error) { runs.Add(1); return false, nil, nil }

	for i := 0; i < 3; i++ {
		if ok, _, _ := c.Do("hit", hit); !ok {
			t.Fatal("expected a match")
		}
	}
	if runs.Load() != 1 {
		t.Fatalf("a match must be reused, ran %d times", runs.Load())
	}

	for i := 0; i < 3; i++ {
		if ok, _, _ := c.Do("miss", miss); ok {
			t.Fatal("expected no match")
		}
	}
	if runs.Load() != 4 {
		t.Fatalf("no-match must not be remembered, ran %d times", runs.Load())
	}

	boom := errors.New("boom")
	if _, _, err := c.Do("err", func() (bool, []json.RawMessage, error) { return true, docs, boom }); !errors.Is(err, boom) {
		t.Fatal("error must pass through")
	}
	var again atomic.Int32
	_, _, _ = c.Do("err", func() (bool, []json.RawMessage, error) { again.Add(1); return false, nil, nil })
	if again.Load() != 1 {
		t.Fatal("an error must not be remembered")
	}
}

func TestCorrelationCacheExpires(t *testing.T) {
	c := NewCorrelationCache(30*time.Millisecond, 10)
	var runs atomic.Int32
	f := func() (bool, []json.RawMessage, error) { runs.Add(1); return true, nil, nil }

	_, _, _ = c.Do("k", f)
	time.Sleep(60 * time.Millisecond)
	_, _, _ = c.Do("k", f)

	if runs.Load() != 2 {
		t.Fatalf("expired entry must run again, ran %d", runs.Load())
	}
}

func TestCorrelationCacheSharesConcurrentRuns(t *testing.T) {
	c := NewCorrelationCache(time.Minute, 10)
	var runs atomic.Int32
	release := make(chan struct{})

	f := func() (bool, []json.RawMessage, error) {
		runs.Add(1)
		<-release
		return false, nil, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _, _ = c.Do("k", f) }()
	}

	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if runs.Load() != 1 {
		t.Fatalf("concurrent searches must share one run, ran %d", runs.Load())
	}
}

func TestCorrelationCacheBounded(t *testing.T) {
	c := NewCorrelationCache(time.Minute, 3)
	for i := 0; i < 20; i++ {
		_, _, _ = c.Do(string(rune('a'+i)), func() (bool, []json.RawMessage, error) { return true, nil, nil })
	}

	if len(c.hits) > 3 {
		t.Fatalf("cache grew to %d", len(c.hits))
	}
}
