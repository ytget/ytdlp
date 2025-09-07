package botguard

import "sync"

// MemoryCache is a simple in-memory cache for Botguard outputs.
type MemoryCache struct {
	mu   sync.RWMutex
	data map[string]Output
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{data: make(map[string]Output)}
}

func (c *MemoryCache) Get(key string) (Output, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	return v, ok
}

func (c *MemoryCache) Set(key string, value Output) {
	c.mu.Lock()
	c.data[key] = value
	c.mu.Unlock()
}

