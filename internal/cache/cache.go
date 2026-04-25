package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

type entry[T any] struct {
	value     T
	expiresAt time.Time
}

type Sizer interface {
	Len() int
}

// CacheStats holds hit/miss counters for a cache instance.
type CacheStats struct {
	Hits   int64
	Misses int64
}

// HitRate returns the cache hit rate as a percentage (0.0–100.0).
// Returns 0 if no requests have been made.
func (s CacheStats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total) * 100.0
}

type Cache[T any] struct {
	mu    sync.RWMutex
	items map[string]entry[T]
	hits  atomic.Int64
	misses atomic.Int64
}

func New[T any]() *Cache[T] {
	return &Cache[T]{
		items: make(map[string]entry[T]),
	}
}

func (c *Cache[T]) Set(key string, value T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = entry[T]{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	var zero T
	if !ok {
		c.misses.Add(1)
		return zero, false
	}
	if time.Now().After(item.expiresAt) {
		c.misses.Add(1)
		c.Delete(key)
		return zero, false
	}

	c.hits.Add(1)
	return item.value, true
}

func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

func (c *Cache[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	count := 0
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
			continue
		}
		count++
	}

	return count
}

// Stats returns cache hit/miss counters and hit rate.
func (c *Cache[T]) Stats() CacheStats {
	return CacheStats{
		Hits:   c.hits.Load(),
		Misses: c.misses.Load(),
	}
}

// ResetStats zeroes the hit/miss counters.
func (c *Cache[T]) ResetStats() {
	c.hits.Store(0)
	c.misses.Store(0)
}
