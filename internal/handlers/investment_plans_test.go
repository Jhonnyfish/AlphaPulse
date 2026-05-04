package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInvestmentPlansHandler_Wiring(t *testing.T) {
	assert.NotNil(t, (*InvestmentPlansHandler)(nil).List)
	assert.NotNil(t, (*InvestmentPlansHandler)(nil).Upsert)
	assert.NotNil(t, (*InvestmentPlansHandler)(nil).Delete)
}

func newTestPlansHandler(t *testing.T) (*InvestmentPlansHandler, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "plans.json")
	h := &InvestmentPlansHandler{path: path, log: zap.NewNop()}
	return h, path
}

func TestInvestmentPlansHandler_List_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestPlansHandler(t)

	r := gin.New()
	r.GET("/api/investment-plans", h.List)

	req := httptest.NewRequest(http.MethodGet, "/api/investment-plans", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["ok"])
	assert.NotNil(t, resp["plans"])
}

func TestInvestmentPlansHandler_Upsert_And_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestPlansHandler(t)

	r := gin.New()
	r.POST("/api/investment-plans", h.Upsert)
	r.GET("/api/investment-plans", h.List)

	// Create a plan
	body := map[string]interface{}{
		"code":        "600001",
		"name":        "Test Stock",
		"target_price": 15.5,
		"stop_loss":    12.0,
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/investment-plans", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	assert.Equal(t, true, createResp["ok"])

	// List should now have 1 plan
	req2 := httptest.NewRequest(http.MethodGet, "/api/investment-plans", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	var listResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &listResp))
	plans := listResp["plans"].(map[string]interface{})
	assert.Len(t, plans, 1)
	assert.Contains(t, plans, "600001")
}

func TestInvestmentPlansHandler_Upsert_EmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestPlansHandler(t)

	r := gin.New()
	r.POST("/api/investment-plans", h.Upsert)

	body := map[string]interface{}{"code": "", "name": "test"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/investment-plans", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_CODE", resp["code"])
}

func TestInvestmentPlansHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, path := newTestPlansHandler(t)

	// Pre-populate a plan file
	data := investmentPlanData{
		Plans: map[string]*investmentPlan{
			"600001": {Code: "600001", Name: "Test", CreatedAt: "2025-01-01T00:00:00Z"},
		},
	}
	raw, _ := json.Marshal(data)
	require.NoError(t, os.WriteFile(path, raw, 0o644))

	r := gin.New()
	r.DELETE("/api/investment-plans/:code", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/investment-plans/600001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["ok"])
	assert.Equal(t, "600001", resp["deleted"])
}

func TestInvestmentPlansHandler_Delete_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestPlansHandler(t)

	r := gin.New()
	r.DELETE("/api/investment-plans/:code", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/investment-plans/999999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInvestmentPlansHandler_Upsert_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, path := newTestPlansHandler(t)

	// Pre-populate
	data := investmentPlanData{
		Plans: map[string]*investmentPlan{
			"600001": {Code: "600001", Name: "Old Name", TargetPrice: 10.0, CreatedAt: "2025-01-01T00:00:00Z"},
		},
	}
	raw, _ := json.Marshal(data)
	require.NoError(t, os.WriteFile(path, raw, 0o644))

	r := gin.New()
	r.POST("/api/investment-plans", h.Upsert)

	// Update with new name and target price
	body := map[string]interface{}{
		"code":         "600001",
		"name":         "New Name",
		"target_price": 20.0,
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/investment-plans", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	plan := resp["plan"].(map[string]interface{})
	assert.Equal(t, "New Name", plan["name"])
	assert.Equal(t, 20.0, plan["target_price"])
}
