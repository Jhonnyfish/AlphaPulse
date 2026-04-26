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
)

func setupTradingJournalTestRouter(h *TradingJournalHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/trading-journal", h.Create)
	return r
}

func TestTradingJournalCreateInvalidBody(t *testing.T) {
	h := NewTradingJournalHandler(nil)
	r := setupTradingJournalTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/trading-journal", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestTradingJournalCreateMissingCode(t *testing.T) {
	h := NewTradingJournalHandler(nil)
	r := setupTradingJournalTestRouter(h)

	body := map[string]interface{}{
		"code":     "",
		"type":     "buy",
		"price":    10.5,
		"quantity": 100,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/trading-journal", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "code is required", resp["error"])
}

func TestTradingJournalCreateInvalidType(t *testing.T) {
	h := NewTradingJournalHandler(nil)
	r := setupTradingJournalTestRouter(h)

	body := map[string]interface{}{
		"code":     "600000",
		"type":     "hold",
		"price":    10.5,
		"quantity": 100,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/trading-journal", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "type must be buy or sell", resp["error"])
}

func TestTradingJournalCreateInvalidPrice(t *testing.T) {
	h := NewTradingJournalHandler(nil)
	r := setupTradingJournalTestRouter(h)

	body := map[string]interface{}{
		"code":     "600000",
		"type":     "buy",
		"price":    0,
		"quantity": 100,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/trading-journal", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "price must be greater than 0", resp["error"])
}

func TestTradingJournalCreateInvalidQuantity(t *testing.T) {
	h := NewTradingJournalHandler(nil)
	r := setupTradingJournalTestRouter(h)

	body := map[string]interface{}{
		"code":     "600000",
		"type":     "sell",
		"price":    15.0,
		"quantity": 0,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/trading-journal", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "quantity must be greater than 0", resp["error"])
}
