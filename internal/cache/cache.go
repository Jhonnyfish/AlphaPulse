package cache

import (
	"sync"
	"time"
)

type entry[T any] struct {
	value     T
	expiresAt time.Time
}

type Sizer interface {
	Len() int
}

type Cache[T any] struct {
	mu    sync.RWMutex
	items map[string]entry[T]
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
		return zero, false
	}
	if time.Now().After(item.expiresAt) {
		c.Delete(key)
		return zero, false
	}

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
