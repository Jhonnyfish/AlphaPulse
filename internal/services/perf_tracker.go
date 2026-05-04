package services

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	apperrors "alphapulse/internal/errors"

	"go.uber.org/zap"
)

const (
	defaultSlowQueriesPath = "/home/finn/.hermes/scripts/slow_queries.json"
	maxSlowQueryEntries    = 500
	maxRollingDurations    = 1000
)

const SlowRequestThreshold = 2 * time.Second

type SlowQueryEntry struct {
	Timestamp  string  `json:"timestamp"`
	Method     string  `json:"method"`
	Path       string  `json:"path"`
	Endpoint   string  `json:"endpoint"`
	DurationMs float64 `json:"duration_ms"`
	StatusCode int     `json:"status_code"`
	ClientIP   string  `json:"client_ip"`
}

type SlowQueriesStats struct {
	AvgDurationMs  float64 `json:"avg_duration_ms"`
	MaxDurationMs  float64 `json:"max_duration_ms"`
	SlowestEndpoint string `json:"slowest_endpoint"`
}

type SlowQueriesResponse struct {
	Ok    bool             `json:"ok"`
	Items []SlowQueryEntry `json:"items"`
	Total int              `json:"total"`
	Stats SlowQueriesStats `json:"stats"`
}

type EndpointPerformanceStat struct {
	Endpoint      string  `json:"endpoint"`
	Method        string  `json:"method"`
	Path          string  `json:"path"`
	Count         int64   `json:"count"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
	P50DurationMs float64 `json:"p50_duration_ms"`
	P95DurationMs float64 `json:"p95_duration_ms"`
	P99DurationMs float64 `json:"p99_duration_ms"`
	MaxDurationMs float64 `json:"max_duration_ms"`
	SlowCount     int64   `json:"slow_count"`
}

type PerformanceSummary struct {
	TotalEndpoints int     `json:"total_endpoints"`
	TotalRequests  int64   `json:"total_requests"`
	AvgDurationMs  float64 `json:"avg_duration_ms"`
	P50DurationMs  float64 `json:"p50_duration_ms"`
	P95DurationMs  float64 `json:"p95_duration_ms"`
	P99DurationMs  float64 `json:"p99_duration_ms"`
	MaxDurationMs  float64 `json:"max_duration_ms"`
	SlowCount      int64   `json:"slow_count"`
	SlowestEndpoint string `json:"slowest_endpoint"`
}

type PerformanceStatsResponse struct {
	Ok            bool                      `json:"ok"`
	Endpoints     []EndpointPerformanceStat `json:"endpoints"`
	TotalRequests int64                     `json:"total_requests"`
	Summary       PerformanceSummary        `json:"summary"`
}

type PerfTracker struct {
	mu              sync.RWMutex
	endpoints       map[string]*endpointStats
	slowQueries     []SlowQueryEntry
	slowQueriesPath string
	log             *zap.Logger
}

type endpointStats struct {
	Method    string
	Path      string
	Count     int64
	TotalMs   float64
	MaxMs     float64
	Durations []float64
	SlowCount int64
}

func NewPerfTracker() *PerfTracker {
	path := os.Getenv("SLOW_QUERIES_PATH")
	if path == "" {
		path = defaultSlowQueriesPath
	}

	tracker := &PerfTracker{
		endpoints:       make(map[string]*endpointStats),
		slowQueriesPath: path,
		log:             zap.L(),
	}
	tracker.loadSlowQueries()

	return tracker
}

func (t *PerfTracker) RecordRequest(method, path string, durationMs float64, statusCode int, clientIP string) {
	if method == "" {
		method = "UNKNOWN"
	}
	if path == "" {
		path = "/"
	}

	endpoint := endpointKey(method, path)
	entry := SlowQueryEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		Method:     method,
		Path:       path,
		Endpoint:   endpoint,
		DurationMs: durationMs,
		StatusCode: statusCode,
		ClientIP:   clientIP,
	}

	var slowSnapshot []SlowQueryEntry

	t.mu.Lock()

	stats, ok := t.endpoints[endpoint]
	if !ok {
		stats = &endpointStats{
			Method: method,
			Path:   path,
		}
		t.endpoints[endpoint] = stats
	}

	stats.Count++
	stats.TotalMs += durationMs
	if durationMs > stats.MaxMs {
		stats.MaxMs = durationMs
	}
	stats.Durations = appendRollingDuration(stats.Durations, durationMs)

	if durationMs > float64(SlowRequestThreshold/time.Millisecond) {
		stats.SlowCount++
		t.slowQueries = appendSlowQuery(t.slowQueries, entry)
		slowSnapshot = cloneSlowQueries(t.slowQueries)
	}

	t.mu.Unlock()

	if len(slowSnapshot) > 0 {
		t.persistSlowQueries(slowSnapshot)
	}
}

func (t *PerfTracker) GetSlowQueries() SlowQueriesResponse {
	t.mu.RLock()
	defer t.mu.RUnlock()

	resp := SlowQueriesResponse{
		Ok:    true,
		Items: make([]SlowQueryEntry, 0),
		Total: len(t.slowQueries),
		Stats: SlowQueriesStats{
			SlowestEndpoint: "N/A",
		},
	}

	if len(t.slowQueries) == 0 {
		return resp
	}

	start := 0
	if len(t.slowQueries) > 100 {
		start = len(t.slowQueries) - 100
	}

	resp.Items = make([]SlowQueryEntry, 0, len(t.slowQueries)-start)

	var totalDuration float64
	var maxDuration float64
	var slowestEndpoint string

	for _, item := range t.slowQueries {
		totalDuration += item.DurationMs
		if item.DurationMs > maxDuration {
			maxDuration = item.DurationMs
			slowestEndpoint = item.Endpoint
		}
	}

	for i := len(t.slowQueries) - 1; i >= start; i-- {
		resp.Items = append(resp.Items, t.slowQueries[i])
	}

	resp.Stats.AvgDurationMs = totalDuration / float64(len(t.slowQueries))
	resp.Stats.MaxDurationMs = maxDuration
	if slowestEndpoint != "" {
		resp.Stats.SlowestEndpoint = slowestEndpoint
	}

	return resp
}

func (t *PerfTracker) GetPerformanceStats() PerformanceStatsResponse {
	t.mu.RLock()
	defer t.mu.RUnlock()

	resp := PerformanceStatsResponse{
		Ok:        true,
		Endpoints: make([]EndpointPerformanceStat, 0, len(t.endpoints)),
		Summary: PerformanceSummary{
			TotalEndpoints: len(t.endpoints),
		},
	}

	if len(t.endpoints) == 0 {
		return resp
	}

	overallDurations := make([]float64, 0)
	var totalDuration float64
	var maxDuration float64
	var slowCount int64
	var slowestEndpoint string

	for endpoint, stats := range t.endpoints {
		durations := append([]float64(nil), stats.Durations...)
		sort.Float64s(durations)

		item := EndpointPerformanceStat{
			Endpoint:      endpoint,
			Method:        stats.Method,
			Path:          stats.Path,
			Count:         stats.Count,
			AvgDurationMs: stats.TotalMs / float64(stats.Count),
			P50DurationMs: percentile(durations, 50),
			P95DurationMs: percentile(durations, 95),
			P99DurationMs: percentile(durations, 99),
			MaxDurationMs: stats.MaxMs,
			SlowCount:     stats.SlowCount,
		}
		resp.Endpoints = append(resp.Endpoints, item)

		resp.TotalRequests += stats.Count
		totalDuration += stats.TotalMs
		slowCount += stats.SlowCount
		if stats.MaxMs > maxDuration {
			maxDuration = stats.MaxMs
			slowestEndpoint = endpoint
		}
		overallDurations = append(overallDurations, durations...)
	}

	sort.Slice(resp.Endpoints, func(i, j int) bool {
		if resp.Endpoints[i].Count == resp.Endpoints[j].Count {
			return resp.Endpoints[i].Endpoint < resp.Endpoints[j].Endpoint
		}
		return resp.Endpoints[i].Count > resp.Endpoints[j].Count
	})

	sort.Float64s(overallDurations)

	resp.Summary.TotalRequests = resp.TotalRequests
	resp.Summary.AvgDurationMs = totalDuration / float64(resp.TotalRequests)
	resp.Summary.P50DurationMs = percentile(overallDurations, 50)
	resp.Summary.P95DurationMs = percentile(overallDurations, 95)
	resp.Summary.P99DurationMs = percentile(overallDurations, 99)
	resp.Summary.MaxDurationMs = maxDuration
	resp.Summary.SlowCount = slowCount
	resp.Summary.SlowestEndpoint = slowestEndpoint

	return resp
}

func (t *PerfTracker) loadSlowQueries() {
	raw, err := os.ReadFile(t.slowQueriesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}

		t.log.Warn("perf tracker: read slow queries failed",
			zap.Error(apperrors.Internal(err)),
			zap.String("path", t.slowQueriesPath),
		)
		return
	}

	var entries []SlowQueryEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.log.Warn("perf tracker: parse slow queries failed",
			zap.Error(apperrors.Internal(err)),
			zap.String("path", t.slowQueriesPath),
		)
		return
	}

	if len(entries) > maxSlowQueryEntries {
		entries = append([]SlowQueryEntry(nil), entries[len(entries)-maxSlowQueryEntries:]...)
	}

	t.mu.Lock()
	t.slowQueries = entries
	t.mu.Unlock()
}

func (t *PerfTracker) persistSlowQueries(entries []SlowQueryEntry) {
	if err := os.MkdirAll(filepath.Dir(t.slowQueriesPath), 0o755); err != nil {
		t.log.Warn("perf tracker: create slow query directory failed",
			zap.Error(apperrors.Internal(err)),
			zap.String("path", t.slowQueriesPath),
		)
		return
	}

	raw, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.log.Warn("perf tracker: marshal slow queries failed", zap.Error(apperrors.Internal(err)))
		return
	}

	tmpPath := t.slowQueriesPath + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o644); err != nil {
		t.log.Warn("perf tracker: write slow queries failed",
			zap.Error(apperrors.Internal(err)),
			zap.String("path", tmpPath),
		)
		return
	}

	if err := os.Rename(tmpPath, t.slowQueriesPath); err != nil {
		t.log.Warn("perf tracker: replace slow queries failed",
			zap.Error(apperrors.Internal(err)),
			zap.String("path", t.slowQueriesPath),
		)
	}
}

func endpointKey(method, path string) string {
	return method + " " + path
}

func appendRollingDuration(values []float64, value float64) []float64 {
	if len(values) < maxRollingDurations {
		return append(values, value)
	}

	copy(values, values[1:])
	values[len(values)-1] = value
	return values
}

func appendSlowQuery(entries []SlowQueryEntry, entry SlowQueryEntry) []SlowQueryEntry {
	if len(entries) < maxSlowQueryEntries {
		return append(entries, entry)
	}

	copy(entries, entries[1:])
	entries[len(entries)-1] = entry
	return entries
}

func cloneSlowQueries(entries []SlowQueryEntry) []SlowQueryEntry {
	if len(entries) == 0 {
		return nil
	}

	cloned := make([]SlowQueryEntry, len(entries))
	copy(cloned, entries)
	return cloned
}

func percentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if len(sortedValues) == 1 {
		return sortedValues[0]
	}
	if p <= 0 {
		return sortedValues[0]
	}
	if p >= 100 {
		return sortedValues[len(sortedValues)-1]
	}

	rank := (p / 100) * float64(len(sortedValues)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))
	if lower == upper {
		return sortedValues[lower]
	}

	weight := rank - float64(lower)
	return sortedValues[lower] + (sortedValues[upper]-sortedValues[lower])*weight
}
