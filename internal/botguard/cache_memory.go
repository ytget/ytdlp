package botguard

import "sync"

// MemoryCache is a simple in-memory cache for Botguard outputs.
type MemoryCache struct {
	mu   sync.RWMutex
	data map[string]Output
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{data: make(map[string]Output)}
}

// Get retrieves a cached output by key
func (c *MemoryCache) Get(key string) (Output, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	return v, ok
}

// Set stores a value in the cache
func (c *MemoryCache) Set(key string, value Output) {
	c.mu.Lock()
	c.data[key] = value
	c.mu.Unlock()
}
