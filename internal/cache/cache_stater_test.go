package cache

import (
	"testing"
	"time"
)

func TestCacheStaterInterface(t *testing.T) {
	// Verify that Cache[T] implements both Sizer and CacheStater
	c := New[string]()
	c.Set("key", "value", 5*time.Second)
	c.Get("key")
	c.Get("miss")

	// Test Sizer interface
	var sizer Sizer = c
	if sizer.Len() != 1 {
		t.Errorf("expected Sizer.Len() = 1, got %d", sizer.Len())
	}

	// Test CacheStater interface
	var stater CacheStater = c
	stats := stater.Stats()
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
	if stats.HitRate() != 50.0 {
		t.Errorf("expected 50%% hit rate, got %.1f", stats.HitRate())
	}
}
