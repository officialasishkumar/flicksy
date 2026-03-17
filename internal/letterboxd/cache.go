package letterboxd

import (
	"sync"
	"time"
)

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

type cache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
	ttl   time.Duration
}

func newCache(ttl time.Duration) *cache {
	return &cache{
		items: make(map[string]cacheItem),
		ttl:   ttl,
	}
}

func (c *cache) get(key string) ([]byte, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	return item.value, true
}

func (c *cache) set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *cache) clearPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
		}
	}
}

func (c *cache) clearAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
}
