package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildScreenerCacheKey(t *testing.T) {
	// Test with no filters
	key := buildScreenerCacheKey(nil, nil, nil, nil, nil, "", "", 50)
	assert.Contains(t, key, "screener:")
	assert.Contains(t, key, "lim=50")

	// Test with tier filter
	key = buildScreenerCacheKey(nil, nil, nil, nil, nil, "", "S,A", 50)
	assert.Contains(t, key, "tier=S,A")

	// Test with sector filter
	key = buildScreenerCacheKey(nil, nil, nil, nil, nil, "银行", "", 50)
	assert.Contains(t, key, "sec=银行")

	// Test with score filters
	minScore := 60.0
	maxScore := 90.0
	key = buildScreenerCacheKey(&minScore, &maxScore, nil, nil, nil, "", "", 50)
	assert.Contains(t, key, "ms=60.0")
	assert.Contains(t, key, "xs=90.0")
}

func TestBuildFiltersMap(t *testing.T) {
	// Test with no filters
	m := buildFiltersMap(nil, nil, nil, nil, nil, "", "", 50)
	assert.Equal(t, 50, m["limit"])
	assert.Nil(t, m["min_score"])
	assert.Nil(t, m["sector"])

	// Test with all filters
	minScore := 60.0
	maxScore := 90.0
	minMomentum := 5.0
	minTrend := 3.0
	maxVolatility := 20.0
	m = buildFiltersMap(&minScore, &maxScore, &minMomentum, &minTrend, &maxVolatility, "科技", "S", 30)
	assert.Equal(t, 30, m["limit"])
	assert.Equal(t, 60.0, m["min_score"])
	assert.Equal(t, 90.0, m["max_score"])
	assert.Equal(t, 5.0, m["min_momentum"])
	assert.Equal(t, 3.0, m["min_trend"])
	assert.Equal(t, 20.0, m["max_volatility"])
	assert.Equal(t, "科技", m["sector"])
	assert.Equal(t, "S", m["tier"])
}

func TestSortScreenerResults(t *testing.T) {
	results := []ScreenerResult{
		{Code: "001", Score: 50.0},
		{Code: "002", Score: 80.0},
		{Code: "003", Score: 30.0},
		{Code: "004", Score: 90.0},
		{Code: "005", Score: 60.0},
	}

	sortScreenerResults(results)

	assert.Equal(t, "004", results[0].Code) // 90
	assert.Equal(t, "002", results[1].Code) // 80
	assert.Equal(t, "005", results[2].Code) // 60
	assert.Equal(t, "001", results[3].Code) // 50
	assert.Equal(t, "003", results[4].Code) // 30
}

func TestSortScreenerResultsEmpty(t *testing.T) {
	var results []ScreenerResult
	sortScreenerResults(results) // should not panic
	assert.Empty(t, results)
}

func TestSortScreenerResultsSingle(t *testing.T) {
	results := []ScreenerResult{
		{Code: "001", Score: 50.0},
	}
	sortScreenerResults(results)
	assert.Len(t, results, 1)
	assert.Equal(t, "001", results[0].Code)
}
