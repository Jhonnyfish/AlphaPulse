package services

import (
	"context"
	"sync"
	"time"
)

const alpha300CacheTTL = 6 * time.Hour

// Alpha300Cache provides a shared, cached view of Alpha300 candidates.
// Only the candidates endpoint triggers a refresh; all other consumers
// read from this cache without hitting the external API.
type Alpha300Cache struct {
	source *Alpha300Service

	mu        sync.RWMutex
	items     []Alpha300Candidate
	fetchedAt time.Time
	refreshing bool
}

// NewAlpha300Cache creates a new cache backed by the given Alpha300Service.
func NewAlpha300Cache(source *Alpha300Service) *Alpha300Cache {
	return &Alpha300Cache{source: source}
}

// GetTopN returns up to n cached candidates. If the cache is empty or stale,
// it fetches fresh data from the external API.
func (c *Alpha300Cache) GetTopN(ctx context.Context, n int) ([]Alpha300Candidate, error) {
	// Fast path: valid cache
	c.mu.RLock()
	if len(c.items) > 0 && time.Since(c.fetchedAt) < alpha300CacheTTL {
		result := c.copyTopN(n)
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Slow path: need refresh
	return c.refreshAndReturn(ctx, n)
}

// Refresh forces a cache refresh from the external API.
// Called by the candidates endpoint to keep data fresh.
func (c *Alpha300Cache) Refresh(ctx context.Context) ([]Alpha300Candidate, error) {
	return c.refreshAndReturn(ctx, 0) // 0 = return all
}

// CachedAt returns when the cache was last refreshed.
func (c *Alpha300Cache) CachedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fetchedAt
}

// IsStale returns true if the cache needs refresh.
func (c *Alpha300Cache) IsStale() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items) == 0 || time.Since(c.fetchedAt) >= alpha300CacheTTL
}

func (c *Alpha300Cache) refreshAndReturn(ctx context.Context, n int) ([]Alpha300Candidate, error) {
	c.mu.Lock()
	if c.refreshing {
		// Another goroutine is already refreshing; wait for it
		c.mu.Unlock()
		// Poll until refresh completes (max 10s)
		deadline := time.After(10 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-deadline:
				// Timeout — return whatever we have
				c.mu.RLock()
				result := c.copyTopN(n)
				c.mu.RUnlock()
				if len(result) == 0 {
					return nil, ctx.Err()
				}
				return result, nil
			case <-ticker.C:
				c.mu.RLock()
				if !c.refreshing && len(c.items) > 0 {
					result := c.copyTopN(n)
					c.mu.RUnlock()
					return result, nil
				}
				c.mu.RUnlock()
			}
		}
	}
	c.refreshing = true
	c.mu.Unlock()

	// Do the actual fetch (outside lock)
	items, err := c.source.FetchCandidates(ctx, 300)

	c.mu.Lock()
	c.refreshing = false
	if err == nil {
		c.items = items
		c.fetchedAt = time.Now()
	}
	stored := c.items
	c.mu.Unlock()

	if err != nil {
		// If we have stale data, return it rather than failing
		if len(stored) > 0 {
			return c.sliceTopN(stored, n), nil
		}
		return nil, err
	}
	return c.sliceTopN(items, n), nil
}

// copyTopN returns a copy of the top N items (caller must hold RLock).
func (c *Alpha300Cache) copyTopN(n int) []Alpha300Candidate {
	return c.sliceTopN(c.items, n)
}

func (c *Alpha300Cache) sliceTopN(items []Alpha300Candidate, n int) []Alpha300Candidate {
	if n <= 0 || n >= len(items) {
		out := make([]Alpha300Candidate, len(items))
		copy(out, items)
		return out
	}
	out := make([]Alpha300Candidate, n)
	copy(out, items[:n])
	return out
}
