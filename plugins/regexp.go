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
	maxDepth := 10
	for i := 0; i < maxDepth && strings.Contains(finalPattern, "{{"); i++ {
		currentCfg := GetCfg("regexp")
		if currentCfg != nil && len(currentCfg.Patterns) > 0 {
			t, err := template.New("pattern").Parse(finalPattern)
			if err == nil {
				var output bytes.Buffer
				err = t.Execute(&output, currentCfg.Patterns)
				if err == nil {
					newPattern := output.String()
					if newPattern == finalPattern {
						break // No more changes
					}
					finalPattern = newPattern
				}
			}
		}
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
