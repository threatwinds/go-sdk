package plugins

import (
	"hash/fnv"
	"regexp"
	"sync"
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

	if compiledPattern, ok := c.cache.Get(pattern); ok {
		return compiledPattern, nil
	}

	lock := c.getLock(pattern)
	lock.Lock()
	defer lock.Unlock()

	if compiledPattern, ok := c.cache.Get(pattern); ok {
		return compiledPattern, nil
	}

	compiledPattern, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	c.cache.Add(pattern, compiledPattern)

	return compiledPattern, nil
}
