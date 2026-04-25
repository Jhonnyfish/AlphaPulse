package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSectorRotationHandler_CacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockSectors := []models.SectorRotationItem{
		{
			Code:          "BK0477",
			Name:          "半导体",
			ChangePct:     2.5,
			RisingCount:   30,
			FallingCount:  10,
			BreadthRatio:  0.75,
			NetFlow:       5000000,
			StrengthScore: 7.2,
		},
	}

	rotCache := cache.New[[]models.SectorRotationItem]()
	h := &SectorRotationHandler{
		rotationCache: rotCache,
	}
	rotCache.Set("rotation:all", mockSectors, 999*time.Second)

	r.GET("/api/sector-rotation", h.Rotation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sector-rotation", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.SectorRotationResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.OK)
	assert.True(t, resp.Cached)
	assert.Len(t, resp.Sectors, 1)
	assert.Equal(t, "半导体", resp.Sectors[0].Name)
	assert.Equal(t, 7.2, resp.Sectors[0].StrengthScore)
}

func TestSectorRotationHandler_EmptyHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	h := &SectorRotationHandler{}

	r.GET("/api/sector-rotation/history", h.RotationHistory)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sector-rotation/history?days=5", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.SectorRotationHistoryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.OK)
	assert.Equal(t, 0, resp.Total)
}

func TestSectorRotationHandler_InvalidDays(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	h := &SectorRotationHandler{}

	r.GET("/api/sector-rotation/history", h.RotationHistory)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sector-rotation/history?days=abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestComputeRotationSummary(t *testing.T) {
	sectors := []models.SectorRotationItem{
		{BreadthRatio: 0.8, NetFlow: 100, StrengthScore: 7.5},
		{BreadthRatio: 0.6, NetFlow: -50, StrengthScore: 4.0},
		{BreadthRatio: 0.7, NetFlow: 200, StrengthScore: 8.0},
	}

	summary := computeRotationSummary(sectors)

	assert.Equal(t, 2, summary.StrongCount)  // scores 7.5 and 8.0 are >= 7
	assert.Equal(t, 1, summary.WeakCount)     // score < 5
	assert.Equal(t, 250.0, summary.TotalNetFlow)
	assert.InDelta(t, 0.7, summary.AvgBreadth, 0.01)
}

func TestComputeRotationSummary_Empty(t *testing.T) {
	summary := computeRotationSummary(nil)
	assert.Equal(t, 0, summary.StrongCount)
	assert.Equal(t, 0, summary.WeakCount)
	assert.Equal(t, 0.0, summary.TotalNetFlow)
}
