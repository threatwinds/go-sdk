package plugins

import (
	"bytes"
	"hash/fnv"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type RegexpCache struct {
	cache *expirable.LRU[string, *regexp.Regexp]
	once  sync.Once
	locks [1024]sync.Mutex
}

func (c *RegexpCache) getLock(key string) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return &c.locks[h.Sum32()%1024]
}

func (c *RegexpCache) Get(pattern string) (*regexp.Regexp, error) {
	c.once.Do(func() {
		c.cache = expirable.NewLRU[string, *regexp.Regexp](10000, nil, time.Hour*24)
	})

	finalPattern := pattern
	// If the pattern contains template markers, try to expand it using global patterns
	if strings.Contains(pattern, "{{") {
		cfgMutex.RLock()
		if cfg != nil && len(cfg.Patterns) > 0 {
			t, err := template.New("pattern").Parse(pattern)
			if err == nil {
				var output bytes.Buffer
				err = t.Execute(&output, cfg.Patterns)
				if err == nil {
					finalPattern = output.String()
				}
			}
		}
		cfgMutex.RUnlock()
	}

	if compiledPattern, ok := c.cache.Get(finalPattern); ok {
		return compiledPattern, nil
	}

	lock := c.getLock(finalPattern)
	lock.Lock()
	defer lock.Unlock()

	if compiledPattern, ok := c.cache.Get(finalPattern); ok {
		return compiledPattern, nil
	}

	compiledPattern, err := regexp.Compile(finalPattern)
	if err != nil {
		return nil, err
	}

	c.cache.Add(finalPattern, compiledPattern)

	return compiledPattern, nil
}
