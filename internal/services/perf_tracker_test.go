package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPercentileEmpty(t *testing.T) {
	assert.Equal(t, 0.0, percentile(nil, 50))
}

func TestPercentileSingle(t *testing.T) {
	assert.Equal(t, 42.0, percentile([]float64{42}, 50))
}

func TestPercentileBoundaries(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5}
	assert.Equal(t, 1.0, percentile(vals, 0))
	assert.Equal(t, 5.0, percentile(vals, 100))
}

func TestPercentileInterpolation(t *testing.T) {
	vals := []float64{10, 20, 30, 40, 50}
	// p50 of 5 sorted values: rank = 0.5 * 4 = 2 → value at index 2 = 30
	assert.InDelta(t, 30.0, percentile(vals, 50), 0.01)
}

func TestPercentileP95(t *testing.T) {
	vals := make([]float64, 100)
	for i := range vals {
		vals[i] = float64(i + 1)
	}
	p95 := percentile(vals, 95)
	assert.True(t, p95 >= 95 && p95 <= 96, "p95 should be around 95-96, got %v", p95)
}

func TestNewPerfTrackerDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "slow_queries.json")
	os.Setenv("SLOW_QUERIES_PATH", path)
	defer os.Unsetenv("SLOW_QUERIES_PATH")

	tracker := NewPerfTracker()
	assert.NotNil(t, tracker)
}

func TestRecordAndGetPerformanceStats(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "slow_queries.json")
	os.Setenv("SLOW_QUERIES_PATH", path)
	defer os.Unsetenv("SLOW_QUERIES_PATH")

	tracker := NewPerfTracker()

	tracker.RecordRequest("GET", "/api/quote", 10.5, 200, "127.0.0.1")
	tracker.RecordRequest("GET", "/api/quote", 15.0, 200, "127.0.0.1")
	tracker.RecordRequest("GET", "/api/quote", 20.0, 200, "127.0.0.1")
	tracker.RecordRequest("POST", "/api/watchlist", 5.0, 201, "127.0.0.1")

	stats := tracker.GetPerformanceStats()
	assert.True(t, stats.Ok)
	assert.Equal(t, int64(4), stats.TotalRequests)
	assert.Len(t, stats.Endpoints, 2)

	// /api/quote should be first (3 requests > 1)
	assert.Equal(t, "GET /api/quote", stats.Endpoints[0].Endpoint)
	assert.Equal(t, int64(3), stats.Endpoints[0].Count)
	assert.InDelta(t, 15.17, stats.Endpoints[0].AvgDurationMs, 0.1)

	// Summary
	assert.Equal(t, int64(4), stats.Summary.TotalRequests)
}

func TestSlowQueryDetection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "slow_queries.json")
	os.Setenv("SLOW_QUERIES_PATH", path)
	defer os.Unsetenv("SLOW_QUERIES_PATH")

	tracker := NewPerfTracker()

	// Fast request — should NOT appear in slow queries
	tracker.RecordRequest("GET", "/api/fast", 50.0, 200, "127.0.0.1")

	// Slow request (>2000ms threshold)
	tracker.RecordRequest("GET", "/api/slow", 3500.0, 200, "10.0.0.1")

	slowResp := tracker.GetSlowQueries()
	assert.True(t, slowResp.Ok)
	assert.Equal(t, 1, slowResp.Total)
	require.Len(t, slowResp.Items, 1)
	assert.Equal(t, "GET", slowResp.Items[0].Method)
	assert.Equal(t, "/api/slow", slowResp.Items[0].Path)
	assert.InDelta(t, 3500.0, slowResp.Items[0].DurationMs, 0.01)
	assert.Equal(t, "10.0.0.1", slowResp.Items[0].ClientIP)

	// Stats
	assert.InDelta(t, 3500.0, slowResp.Stats.AvgDurationMs, 0.01)
	assert.InDelta(t, 3500.0, slowResp.Stats.MaxDurationMs, 0.01)
	assert.Equal(t, "GET /api/slow", slowResp.Stats.SlowestEndpoint)
}

func TestSlowQueryPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "slow_queries.json")

	os.Setenv("SLOW_QUERIES_PATH", path)
	defer os.Unsetenv("SLOW_QUERIES_PATH")

	tracker := NewPerfTracker()

	// Record a slow query
	tracker.RecordRequest("POST", "/api/heavy", 5000.0, 200, "10.0.0.2")

	// File should exist
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "/api/heavy")

	// Reload from file — new tracker should pick it up
	tracker2 := NewPerfTracker()
	slowResp := tracker2.GetSlowQueries()
	assert.GreaterOrEqual(t, slowResp.Total, 1)
}

func TestNoSlowQueriesReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "slow_queries.json")
	os.Setenv("SLOW_QUERIES_PATH", path)
	defer os.Unsetenv("SLOW_QUERIES_PATH")

	tracker := NewPerfTracker()
	resp := tracker.GetSlowQueries()
	assert.True(t, resp.Ok)
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Items)
	assert.Equal(t, "N/A", resp.Stats.SlowestEndpoint)
}

func TestAppendRollingDuration(t *testing.T) {
	// Under limit
	durs := []float64{1, 2}
	durs = appendRollingDuration(durs, 3)
	assert.Len(t, durs, 3)
	assert.Equal(t, 3.0, durs[2])

	// At limit — should roll
	durs = make([]float64, maxRollingDurations)
	for i := range durs {
		durs[i] = float64(i)
	}
	durs = appendRollingDuration(durs, 9999.0)
	assert.Len(t, durs, maxRollingDurations)
	assert.Equal(t, 9999.0, durs[maxRollingDurations-1])
	assert.Equal(t, 1.0, durs[0]) // first element shifted out, 1 is now first
}

func TestAppendSlowQueryLimit(t *testing.T) {
	entries := make([]SlowQueryEntry, maxSlowQueryEntries)
	for i := range entries {
		entries[i] = SlowQueryEntry{Path: "/old"}
	}
	entries = appendSlowQuery(entries, SlowQueryEntry{Path: "/new"})
	assert.Len(t, entries, maxSlowQueryEntries)
	assert.Equal(t, "/new", entries[maxSlowQueryEntries-1].Path)
}

func TestCloneSlowQueries(t *testing.T) {
	assert.Nil(t, cloneSlowQueries(nil))
	assert.Nil(t, cloneSlowQueries([]SlowQueryEntry{}))

	orig := []SlowQueryEntry{{Path: "/a"}, {Path: "/b"}}
	cloned := cloneSlowQueries(orig)
	assert.Equal(t, orig, cloned)
	// Mutating clone shouldn't affect original
	cloned[0].Path = "/modified"
	assert.Equal(t, "/a", orig[0].Path)
}
