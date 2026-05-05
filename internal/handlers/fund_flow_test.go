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

func TestFundFlowHandler_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &FundFlowHandler{}
	r.GET("/api/fund-flow/flow", h.Flow)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fund-flow/flow", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "code is required")
}

func TestFundFlowHandler_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &FundFlowHandler{}
	r.GET("/api/fund-flow/flow", h.Flow)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fund-flow/flow?code=xyz123", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "invalid code format")
}

func TestFundFlowHandler_InvalidDays(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &FundFlowHandler{}
	r.GET("/api/fund-flow/flow", h.Flow)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fund-flow/flow?code=600176&days=999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFundFlowHandler_CacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockFlows := []models.MoneyFlowDay{
		{Date: "2024-01-15", MainNet: 100.5},
	}

	flowCache := cache.New[[]models.MoneyFlowDay]()
	h := &FundFlowHandler{
		flowCache: flowCache,
	}
	flowCache.Set("600176:5", mockFlows, 999*time.Second)

	r.GET("/api/fund-flow/flow", h.Flow)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/fund-flow/flow?code=600176&days=5", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["cached"])
	assert.Equal(t, "600176", resp["code"])
	items := resp["items"].([]interface{})
	assert.Len(t, items, 1)
}
