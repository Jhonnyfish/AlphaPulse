package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCompareRouter(h *CompareHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/compare/sector", h.SectorCompare)
	r.GET("/compare/backtest", h.BacktestCompare)
	return r
}

func TestSectorCompareMissingCode(t *testing.T) {
	h := NewCompareHandler(nil, nil)
	r := setupCompareRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/compare/sector", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["ok"])
	assert.Contains(t, resp["error"], "code is required")
}

func TestBacktestCompareMissingCodes(t *testing.T) {
	h := NewCompareHandler(nil, nil)
	r := setupCompareRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/compare/backtest", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["ok"])
}

func TestBacktestCompareSingleCode(t *testing.T) {
	h := NewCompareHandler(nil, nil)
	r := setupCompareRouter(h)

	// Only 1 code — needs 2-5
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/compare/backtest?codes=600519", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["ok"])
}

func TestBacktestCompareTooManyCodes(t *testing.T) {
	h := NewCompareHandler(nil, nil)
	r := setupCompareRouter(h)

	// 6 codes exceeds max of 5
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/compare/backtest?codes=600519,000001,300750,688001,601398,600036", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["ok"])
}

// TestBacktestCompareDedupCodes removed — requires real EastMoney service
