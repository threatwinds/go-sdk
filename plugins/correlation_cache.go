package plugins

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

// CorrelationCache spares the store the same correlation search when events of one stream keep
// asking it. A search that found its matches is remembered for ttl, so while a rule is already
// firing the next events reuse the answer; a search that found nothing is never remembered, so the
// event that reaches a rule's threshold is not held back. Searches for the same key that are in
// flight at the same time share one run.
type CorrelationCache struct {
	ttl time.Duration
	max int

	mu       sync.Mutex
	hits     map[string]cachedHits
	inflight map[string]*flight
}

type cachedHits struct {
	docs    []json.RawMessage
	expires time.Time
}

type flight struct {
	done  chan struct{}
	found bool
	docs  []json.RawMessage
	err   error
}

// NewCorrelationCache keeps at most max remembered searches.
func NewCorrelationCache(ttl time.Duration, max int) *CorrelationCache {
	return &CorrelationCache{
		ttl:      ttl,
		max:      max,
		hits:     make(map[string]cachedHits),
		inflight: make(map[string]*flight),
	}
}

// Do returns the remembered answer for key, or runs run and remembers it when it found matches.
func (c *CorrelationCache) Do(key string, run func() (bool, []json.RawMessage, error)) (bool, []json.RawMessage, error) {
	now := time.Now()

	c.mu.Lock()
	if h, ok := c.hits[key]; ok && now.Before(h.expires) {
		c.mu.Unlock()

		return true, h.docs, nil
	}

	if f, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		<-f.done

		return f.found, f.docs, f.err
	}

	f := &flight{done: make(chan struct{})}
	c.inflight[key] = f
	c.mu.Unlock()

	f.found, f.docs, f.err = run()

	c.mu.Lock()
	delete(c.inflight, key)

	if f.err == nil && f.found {
		if len(c.hits) >= c.max {
			c.evict(now)
		}

		c.hits[key] = cachedHits{docs: f.docs, expires: time.Now().Add(c.ttl)}
	}
	c.mu.Unlock()

	close(f.done)

	return f.found, f.docs, f.err
}

// evict drops what expired and, if that was not enough, everything.
func (c *CorrelationCache) evict(now time.Time) {
	for k, h := range c.hits {
		if !now.Before(h.expires) {
			delete(c.hits, k)
		}
	}

	if len(c.hits) >= c.max {
		clear(c.hits)
	}
}

// CacheKey identifies what ExecuteStore would search for: the request, the scope it starts from and
// the filters once the event's values are filled in.
func (e *SearchRequest) CacheKey(scope store.Scope, previous *string) (string, error) {
	filters, err := e.filters(previous)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%p|%s|%s|%s|%v", e, scope.Tenant, scope.Dataset, scope.DataType, filters), nil
}
