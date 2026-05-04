package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupPortfolioTestRouter(h *PortfolioHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/portfolio", h.Add)
	r.PUT("/api/portfolio/:id", h.Update)
	r.DELETE("/api/portfolio/:id", h.Delete)
	return r
}

func newTestPortfolioHandler() *PortfolioHandler {
	return NewPortfolioHandler(nil, nil, nil, zap.NewNop())
}

func TestPortfolioAddInvalidBody(t *testing.T) {
	h := newTestPortfolioHandler()
	r := setupPortfolioTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/portfolio", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestPortfolioAddMissingCode(t *testing.T) {
	h := newTestPortfolioHandler()
	r := setupPortfolioTestRouter(h)

	body := map[string]interface{}{
		"code":       "12345",
		"cost_price": 10.5,
		"quantity":   100,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/portfolio", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "股票代码格式错误", resp["error"])
}

func TestPortfolioAddInvalidCostPrice(t *testing.T) {
	h := newTestPortfolioHandler()
	r := setupPortfolioTestRouter(h)

	body := map[string]interface{}{
		"code":       "600000",
		"cost_price": 0,
		"quantity":   100,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/portfolio", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "成本价必须大于 0", resp["error"])
}

func TestPortfolioAddInvalidQuantity(t *testing.T) {
	h := newTestPortfolioHandler()
	r := setupPortfolioTestRouter(h)

	body := map[string]interface{}{
		"code":       "600000",
		"cost_price": 10.5,
		"quantity":   0,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/portfolio", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "持仓量必须大于 0", resp["error"])
}

func TestPortfolioUpdateInvalidBody(t *testing.T) {
	h := newTestPortfolioHandler()
	r := setupPortfolioTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/portfolio/test-id", bytes.NewBufferString("bad json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid request body", resp["error"])
}
