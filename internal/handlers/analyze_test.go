package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAnalyzeRouter(h *AnalyzeHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/analyze", h.Analyze)
	r.GET("/stockinfo", h.StockInfo)
	return r
}

func TestAnalyzeMissingCode(t *testing.T) {
	h := NewAnalyzeHandler(nil, nil, zap.NewNop())
	r := setupAnalyzeRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/analyze", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_CODE", resp["code"])
}

func TestAnalyzeEmptyCode(t *testing.T) {
	h := NewAnalyzeHandler(nil, nil, zap.NewNop())
	r := setupAnalyzeRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/analyze?code=", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_CODE", resp["code"])
}

func TestAnalyzeInvalidCodeFormat(t *testing.T) {
	h := NewAnalyzeHandler(nil, nil, zap.NewNop())
	r := setupAnalyzeRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/analyze?code=abc", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_CODE_FORMAT", resp["code"])
}

func TestAnalyzeTooManyCodes(t *testing.T) {
	h := NewAnalyzeHandler(nil, nil, zap.NewNop())
	r := setupAnalyzeRouter(h)

	// 11 codes exceeds the max of 10
	codes := "600519,000001,300750,688001,601398,600036,000858,002594,600276,601012,600809"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/analyze?code="+codes, nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "TOO_MANY_CODES", resp["code"])
}

func TestStockInfoMissingCode(t *testing.T) {
	h := NewAnalyzeHandler(nil, nil, zap.NewNop())
	r := setupAnalyzeRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stockinfo", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_CODE", resp["code"])
}

func TestStockInfoInvalidCode(t *testing.T) {
	h := NewAnalyzeHandler(nil, nil, zap.NewNop())
	r := setupAnalyzeRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stockinfo?code=xyz123", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_CODE_FORMAT", resp["code"])
}
