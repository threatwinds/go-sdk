package utils

import (
	"regexp"
	"sync"
)

type RegexpCache struct {
	cache map[string]*regexp.Regexp
	mutex sync.RWMutex
}

func (c *RegexpCache) Get(pattern string) (*regexp.Regexp, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if compiledPattern, ok := c.cache[pattern]; ok {
		return compiledPattern, nil
	}

	compiledPattern, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	go c.set(pattern, compiledPattern)

	return compiledPattern, nil
}

func (c *RegexpCache) set(pattern string, compiledPattern *regexp.Regexp) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache[pattern] = compiledPattern
}
