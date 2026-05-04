package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"alphapulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDragonTigerRouter(h *DragonTigerHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/dragon-tiger", h.GetDragonTiger)
	r.GET("/dragon-tiger-history", h.GetHistory)
	r.GET("/institution-tracker", h.GetInstitutionTracker)
	return r
}

func TestDragonTigerCacheHit(t *testing.T) {
	h := NewDragonTigerHandler(nil)
	r := setupDragonTigerRouter(h)

	h.dragonTigerCache.Set("latest", models.DragonTigerResponse{
		OK: true,
		Items: []models.DragonTigerItem{{
			Code:      "600519",
			Name:      "贵州茅台",
			NetBuy:    12.5,
			TradeDate: "2026-04-25",
		}},
		Count:       1,
		TotalNetBuy: 12.5,
		Cached:      false,
	}, 5*time.Second)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dragon-tiger", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp models.DragonTigerResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.OK)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "600519", resp.Items[0].Code)
	assert.True(t, resp.Cached)
}

func TestDragonTigerHistoryInvalidDays(t *testing.T) {
	h := NewDragonTigerHandler(nil)
	r := setupDragonTigerRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dragon-tiger-history?days=0", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_DAYS", resp["code"])
}

func TestDragonTigerHistoryCacheHit(t *testing.T) {
	h := NewDragonTigerHandler(nil)
	r := setupDragonTigerRouter(h)

	h.historyCache.Set("history:5", models.DragonTigerHistoryResponse{
		OK:    true,
		Dates: []string{"2026-04-25"},
		DailySummary: []models.DailySummary{{
			Date:        "2026-04-25",
			Count:       1,
			TotalNetBuy: 10,
		}},
		Cached: false,
	}, 5*time.Second)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dragon-tiger-history?days=5", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp models.DragonTigerHistoryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Cached)
	assert.Equal(t, []string{"2026-04-25"}, resp.Dates)
}

func TestInstitutionTrackerCacheHit(t *testing.T) {
	h := NewDragonTigerHandler(nil)
	r := setupDragonTigerRouter(h)

	h.institutionCache.Set("institution:5", models.InstitutionTrackerResponse{
		OK: true,
		Institutions: []models.InstitutionStat{{
			Name:        "机构专用",
			Appearances: 3,
			TotalNet:    28.6,
			Dates:       []string{"2026-04-25"},
		}},
		Period: "5 days",
		Cached: false,
	}, 5*time.Second)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/institution-tracker?days=5", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp models.InstitutionTrackerResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Cached)
	assert.Len(t, resp.Institutions, 1)
	assert.Equal(t, "机构专用", resp.Institutions[0].Name)
}
