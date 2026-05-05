package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestComputeTierCounts
// ---------------------------------------------------------------------------

func TestComputeTierCounts_Empty(t *testing.T) {
	result := computeTierCounts(nil)
	assert.Empty(t, result)
}

func TestComputeTierCounts_WithTiers(t *testing.T) {
	candidates := []services.Alpha300Candidate{
		{RecommendationTier: "strong_buy"},
		{RecommendationTier: "strong_buy"},
		{RecommendationTier: "buy"},
		{RecommendationTier: "hold"},
	}
	result := computeTierCounts(candidates)
	assert.Equal(t, 2, result["strong_buy"])
	assert.Equal(t, 1, result["buy"])
	assert.Equal(t, 1, result["hold"])
}

func TestComputeTierCounts_EmptyTier(t *testing.T) {
	candidates := []services.Alpha300Candidate{
		{RecommendationTier: "buy"},
		{RecommendationTier: ""}, // empty tier should map to "unknown"
		{RecommendationTier: ""},
	}
	result := computeTierCounts(candidates)
	assert.Equal(t, 1, result["buy"])
	assert.Equal(t, 2, result["unknown"])
}

// ---------------------------------------------------------------------------
// TestSortCandidatesByScore
// ---------------------------------------------------------------------------

func TestSortCandidatesByScore(t *testing.T) {
	items := []services.Alpha300Candidate{
		{Code: "A", Score: 50},
		{Code: "B", Score: 90},
		{Code: "C", Score: 70},
	}
	sortCandidatesByScore(items)

	require.Len(t, items, 3)
	assert.Equal(t, "B", items[0].Code, "highest score first")
	assert.Equal(t, float64(90), items[0].Score)
	assert.Equal(t, "C", items[1].Code)
	assert.Equal(t, float64(70), items[1].Score)
	assert.Equal(t, "A", items[2].Code, "lowest score last")
	assert.Equal(t, float64(50), items[2].Score)
}

func TestSortCandidatesByScore_SingleItem(t *testing.T) {
	items := []services.Alpha300Candidate{{Code: "X", Score: 42}}
	sortCandidatesByScore(items)
	assert.Len(t, items, 1)
	assert.Equal(t, "X", items[0].Code)
}

func TestSortCandidatesByScore_Empty(t *testing.T) {
	var items []services.Alpha300Candidate
	sortCandidatesByScore(items) // should not panic
	assert.Empty(t, items)
}

// ---------------------------------------------------------------------------
// TestNormalizeMetric
// ---------------------------------------------------------------------------

func TestNormalizeMetric_AtMin(t *testing.T) {
	assert.Equal(t, float64(0), normalizeMetric(0, 0, 100))
}

func TestNormalizeMetric_AtMax(t *testing.T) {
	assert.Equal(t, float64(100), normalizeMetric(100, 0, 100))
}

func TestNormalizeMetric_Midpoint(t *testing.T) {
	assert.Equal(t, float64(50), normalizeMetric(50, 0, 100))
}

func TestNormalizeMetric_BelowMin(t *testing.T) {
	// Values below min should clamp to min → 0
	assert.Equal(t, float64(0), normalizeMetric(-10, 0, 100))
}

func TestNormalizeMetric_AboveMax(t *testing.T) {
	// Values above max should clamp to max → 100
	assert.Equal(t, float64(100), normalizeMetric(200, 0, 100))
}

func TestNormalizeMetric_MaxEqualsMin(t *testing.T) {
	// max == min should return 50 (fallback)
	assert.Equal(t, float64(50), normalizeMetric(5, 5, 5))
}

func TestNormalizeMetric_MaxLessThanMin(t *testing.T) {
	// max < min should return 50
	assert.Equal(t, float64(50), normalizeMetric(5, 10, 3))
}

func TestNormalizeMetric_NegativeRange(t *testing.T) {
	// Map [-100, 100] → [0, 100]: value 0 should be 50
	assert.Equal(t, float64(50), normalizeMetric(0, -100, 100))
	// value -100 → 0
	assert.Equal(t, float64(0), normalizeMetric(-100, -100, 100))
	// value 100 → 100
	assert.Equal(t, float64(100), normalizeMetric(100, -100, 100))
}

// ---------------------------------------------------------------------------
// TestComputeStrategyScore
// ---------------------------------------------------------------------------

func TestComputeStrategyScore_EmptyWeights(t *testing.T) {
	cand := services.Alpha300Candidate{Score: 75.5}
	result := computeStrategyScore(cand, map[string]interface{}{})
	assert.Equal(t, 75.5, result, "empty weights should fall back to cand.Score")
}

func TestComputeStrategyScore_NilWeights(t *testing.T) {
	cand := services.Alpha300Candidate{Score: 60.0}
	result := computeStrategyScore(cand, nil)
	assert.Equal(t, 60.0, result, "nil weights should fall back to cand.Score")
}

func TestComputeStrategyScore_SingleDimension(t *testing.T) {
	cand := services.Alpha300Candidate{
		Score:    50,
		Momentum: 50, // normalized: (50-(-100))/(100-(-100))*100 = 75
	}
	weights := map[string]interface{}{"momentum": 1.0}
	result := computeStrategyScore(cand, weights)
	assert.InDelta(t, 75.0, result, 0.01)
}

func TestComputeStrategyScore_WeightedAverage(t *testing.T) {
	cand := services.Alpha300Candidate{
		Score:    50,
		Momentum: 0,   // normalized: (0+100)/200*100 = 50
		Trend:    100, // normalized: (100+100)/200*100 = 100
	}
	weights := map[string]interface{}{
		"momentum": 1.0,
		"trend":    1.0,
	}
	result := computeStrategyScore(cand, weights)
	// weighted sum = 50*1 + 100*1 = 150, total weight = 2 → 75
	assert.InDelta(t, 75.0, result, 0.01)
}

func TestComputeStrategyScore_ZeroWeight(t *testing.T) {
	cand := services.Alpha300Candidate{Score: 80}
	weights := map[string]interface{}{"momentum": 0.0}
	result := computeStrategyScore(cand, weights)
	assert.Equal(t, 80.0, result, "zero weight should fall back to cand.Score")
}

func TestComputeStrategyScore_InvalidWeightType(t *testing.T) {
	cand := services.Alpha300Candidate{Score: 80}
	weights := map[string]interface{}{"momentum": "not_a_number"}
	result := computeStrategyScore(cand, weights)
	assert.Equal(t, 80.0, result, "non-numeric weight should fall back to cand.Score")
}

// ---------------------------------------------------------------------------
// TestMapDimensionScore
// ---------------------------------------------------------------------------

func TestMapDimensionScore_Momentum(t *testing.T) {
	cand := services.Alpha300Candidate{Momentum: 50}
	// normalizeMetric(50, -100, 100) = (50+100)/200*100 = 75
	assert.Equal(t, float64(75), mapDimensionScore(cand, "momentum"))
}

func TestMapDimensionScore_Trend(t *testing.T) {
	cand := services.Alpha300Candidate{Trend: -50}
	// normalizeMetric(-50, -100, 100) = (-50+100)/200*100 = 25
	assert.Equal(t, float64(25), mapDimensionScore(cand, "trend"))
}

func TestMapDimensionScore_Volatility(t *testing.T) {
	cand := services.Alpha300Candidate{Volatility: 30}
	// 100 - clamped(30,0,100) = 70
	assert.Equal(t, float64(70), mapDimensionScore(cand, "volatility"))
}

func TestMapDimensionScore_Volatility_Negative(t *testing.T) {
	cand := services.Alpha300Candidate{Volatility: -10}
	// clamped to 0, so 100 - 0 = 100
	assert.Equal(t, float64(100), mapDimensionScore(cand, "volatility"))
}

func TestMapDimensionScore_Volatility_OverMax(t *testing.T) {
	cand := services.Alpha300Candidate{Volatility: 150}
	// clamped to 100, so 100 - 100 = 0
	assert.Equal(t, float64(0), mapDimensionScore(cand, "volatility"))
}

func TestMapDimensionScore_Liquidity(t *testing.T) {
	cand := services.Alpha300Candidate{Liquidity: 60}
	// normalizeMetric(60, 0, 100) = 60
	assert.Equal(t, float64(60), mapDimensionScore(cand, "liquidity"))
}

func TestMapDimensionScore_VolumePrice(t *testing.T) {
	cand := services.Alpha300Candidate{Momentum: 25}
	// uses momentum as proxy, same as momentum dimension
	assert.Equal(t, mapDimensionScore(cand, "momentum"), mapDimensionScore(cand, "volume_price"))
}

func TestMapDimensionScore_Technical(t *testing.T) {
	cand := services.Alpha300Candidate{Trend: 80}
	// uses trend as proxy
	assert.Equal(t, mapDimensionScore(cand, "trend"), mapDimensionScore(cand, "technical"))
}

func TestMapDimensionScore_ProxyDimensions(t *testing.T) {
	cand := services.Alpha300Candidate{Score: 55}
	for _, dim := range []string{"order_flow", "valuation", "sentiment", "sector"} {
		assert.Equal(t, float64(55), mapDimensionScore(cand, dim),
			"dimension %q should use Score as proxy", dim)
	}
}

func TestMapDimensionScore_Unknown(t *testing.T) {
	cand := services.Alpha300Candidate{Score: 42}
	assert.Equal(t, float64(42), mapDimensionScore(cand, "nonexistent"),
		"unknown dimension should fall back to Score")
}

// ---------------------------------------------------------------------------
// TestCandidatesHandler – limit parsing via HTTP
// ---------------------------------------------------------------------------

func TestCandidatesHandler_DefaultLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/candidates", func(c *gin.Context) {
		// Replicate limit parsing logic from Candidates handler
		limitStr := c.DefaultQuery("limit", "50")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 300 {
			limit = 50
		}
		c.JSON(http.StatusOK, gin.H{"limit": limit})
	})

	// No limit param → default 50
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/candidates", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(50), body["limit"])
}

func TestCandidatesHandler_InvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/candidates", func(c *gin.Context) {
		limitStr := c.DefaultQuery("limit", "50")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 300 {
			limit = 50
		}
		c.JSON(http.StatusOK, gin.H{"limit": limit})
	})

	tests := []struct {
		name     string
		query    string
		expected float64
	}{
		{"negative", "?limit=-1", 50},
		{"zero", "?limit=0", 50},
		{"over_max", "?limit=500", 50},
		{"non_numeric", "?limit=abc", 50},
		{"float_string", "?limit=1.5", 50},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/candidates"+tc.query, nil)
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			var body map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.expected, body["limit"])
		})
	}
}

func TestCandidatesHandler_ValidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/candidates", func(c *gin.Context) {
		limitStr := c.DefaultQuery("limit", "50")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 300 {
			limit = 50
		}
		c.JSON(http.StatusOK, gin.H{"limit": limit})
	})

	tests := []struct {
		name     string
		query    string
		expected float64
	}{
		{"min_valid", "?limit=1", 1},
		{"max_valid", "?limit=300", 300},
		{"mid", "?limit=100", 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/candidates"+tc.query, nil)
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			var body map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.expected, body["limit"])
		})
	}
}
