package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := New[string]()

	c.Set("key1", "value1", 5*time.Second)

	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected to find key1")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %s", val)
	}
}

func TestCacheExpiration(t *testing.T) {
	c := New[string]()

	c.Set("key1", "value1", 50*time.Millisecond)

	// Should exist immediately
	_, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	_, ok = c.Get("key1")
	if ok {
		t.Error("expected key1 to be expired")
	}
}

func TestCacheDelete(t *testing.T) {
	c := New[int]()

	c.Set("key1", 42, 5*time.Second)
	c.Delete("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected key1 to be deleted")
	}
}

func TestCacheMiss(t *testing.T) {
	c := New[string]()

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected miss for nonexistent key")
	}
}

func TestCacheOverwrite(t *testing.T) {
	c := New[int]()

	c.Set("key1", 1, 5*time.Second)
	c.Set("key1", 2, 5*time.Second)

	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}
	if val != 2 {
		t.Errorf("expected 2 after overwrite, got %d", val)
	}
}

func TestCacheLen(t *testing.T) {
	c := New[string]()

	c.Set("a", "1", 5*time.Second)
	c.Set("b", "2", 5*time.Second)
	c.Set("c", "3", 50*time.Millisecond)

	if got := c.Len(); got != 3 {
		t.Errorf("expected Len() = 3, got %d", got)
	}

	// Wait for "c" to expire
	time.Sleep(60 * time.Millisecond)

	if got := c.Len(); got != 2 {
		t.Errorf("expected Len() = 2 after expiration, got %d", got)
	}
}

func TestCacheConcurrent(t *testing.T) {
	c := New[int]()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Set("key", n, 5*time.Second)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Get("key")
		}()
	}

	wg.Wait()

	// Should still work
	c.Set("final", 999, 5*time.Second)
	val, ok := c.Get("final")
	if !ok || val != 999 {
		t.Error("cache corrupted after concurrent access")
	}
}

func TestCacheZeroValue(t *testing.T) {
	c := New[int]()

	// Get returns zero value for missing keys
	val, ok := c.Get("missing")
	if ok {
		t.Error("expected ok=false for missing key")
	}
	if val != 0 {
		t.Errorf("expected zero value, got %d", val)
	}
}

func TestCacheStatsHitMiss(t *testing.T) {
	c := New[string]()

	c.Set("a", "1", 5*time.Second)

	// 2 hits, 1 miss
	c.Get("a")
	c.Get("a")
	c.Get("missing")

	stats := c.Stats()
	if stats.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
}

func TestCacheStatsHitRate(t *testing.T) {
	c := New[int]()

	c.Set("x", 10, 5*time.Second)

	// 3 hits, 1 miss → 75%
	c.Get("x")
	c.Get("x")
	c.Get("x")
	c.Get("nope")

	stats := c.Stats()
	if stats.Hits != 3 || stats.Misses != 1 {
		t.Errorf("expected 3 hits/1 miss, got %d/%d", stats.Hits, stats.Misses)
	}

	rate := stats.HitRate()
	if rate != 75.0 {
		t.Errorf("expected hit rate 75.0, got %.1f", rate)
	}
}

func TestCacheStatsHitRateZeroRequests(t *testing.T) {
	c := New[string]()
	stats := c.Stats()
	if stats.HitRate() != 0 {
		t.Errorf("expected 0 hit rate for empty cache, got %.1f", stats.HitRate())
	}
}

func TestCacheStatsExpiredCountsAsMiss(t *testing.T) {
	c := New[string]()

	c.Set("key", "val", 30*time.Millisecond)

	// Hit
	c.Get("key")

	// Wait for expiration — should count as miss
	time.Sleep(40 * time.Millisecond)
	c.Get("key")

	stats := c.Stats()
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss (expired), got %d", stats.Misses)
	}
}

func TestCacheResetStats(t *testing.T) {
	c := New[int]()

	c.Set("k", 1, 5*time.Second)
	c.Get("k")
	c.Get("miss")

	stats := c.Stats()
	if stats.Hits == 0 || stats.Misses == 0 {
		t.Fatal("expected non-zero stats before reset")
	}

	c.ResetStats()

	stats = c.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("expected zero stats after reset, got hits=%d misses=%d", stats.Hits, stats.Misses)
	}
}
