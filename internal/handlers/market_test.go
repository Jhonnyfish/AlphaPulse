package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
)

func setupTestRouter(h *MarketHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/market/quote", h.Quote)
	r.GET("/market/kline", h.Kline)
	r.GET("/market/sectors", h.Sectors)
	r.GET("/market/overview", h.Overview)
	r.GET("/market/news", h.News)
	return r
}

func TestQuoteMissingCode(t *testing.T) {
	h := NewMarketHandler(nil, nil, nil)
	r := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/market/quote", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "INVALID_CODE" {
		t.Errorf("expected INVALID_CODE, got %v", resp["code"])
	}
}

func TestQuoteInvalidCodeFormat(t *testing.T) {
	h := NewMarketHandler(nil, nil, nil)
	r := setupTestRouter(h)

	tests := []string{"12345", "abcdef", "1234567", "12345a"}
	for _, code := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/market/quote?code="+code, nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("code=%s: expected 400, got %d", code, w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["code"] != "INVALID_CODE_FORMAT" {
			t.Errorf("code=%s: expected INVALID_CODE_FORMAT, got %v", code, resp["code"])
		}
	}
}

func TestQuoteValidCodeFormat(t *testing.T) {
	// Create a mock tencent API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`v_sh600519="1~贵州茅台~600519~1850.00~1840.00~1845.00~12345~6000~6345~1849.00~100~1848.00~200~1847.00~300~1846.00~400~1845.00~500~1850.00~100~1851.00~200~1852.00~300~1853.00~400~1854.00~500~15:00:03~20260425~10.00~0.54~1860.00~1835.00~1850.00/12345/228000000~12345~22800.00~1.23~25.50~~1860.00~1835.00~1.36~22800.00~22800.00~38.50~2024.00~22.50~-1.23~1860.00~1835.00~0.67~1840.00~1840.00~1640.50~-15.75~0.85~GP-A~-2.66~8.66~0.25~1850.00~12345~22800.00~2026-04-25 15:00:03~0";`))
	}))
	defer mockServer.Close()

	// Note: We can't easily redirect the tencent service to our mock server
	// because the URL is constructed internally. This test verifies that
	// valid codes pass validation and reach the service layer.
	// The actual API call will fail (no network), but we get a 500 not 400.
	svc := services.NewTencentService(2 * time.Second)
	h := NewMarketHandler(nil, svc, nil)
	r := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/market/quote?code=600519", nil)
	r.ServeHTTP(w, req)

	// Should NOT be 400 (validation passed), will be 500 (network error)
	if w.Code == http.StatusBadRequest {
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		t.Errorf("valid code 600519 should not get 400, got: %v", resp)
	}
}

func TestKlineMissingCode(t *testing.T) {
	h := NewMarketHandler(nil, nil, nil)
	r := setupTestRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/market/kline", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestKlineInvalidDays(t *testing.T) {
	h := NewMarketHandler(nil, nil, nil)
	r := setupTestRouter(h)

	tests := []string{"abc", "0", "-1"}
	for _, days := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/market/kline?code=600519&days="+days, nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("days=%s: expected 400, got %d", days, w.Code)
		}
	}
}

func TestCacheStats(t *testing.T) {
	h := NewMarketHandler(nil, nil, nil)
	stats := h.CacheStats()

	expected := []string{"quote", "kline", "sectors", "overview", "news"}
	for _, name := range expected {
		if _, ok := stats[name]; !ok {
			t.Errorf("missing cache stat for %q", name)
		}
	}
}

func TestQuoteCacheHit(t *testing.T) {
	h := NewMarketHandler(nil, nil, nil)
	r := setupTestRouter(h)

	// Pre-populate cache
	h.quoteCache.Set("600519", models.Quote{
		Code: "600519", Name: "贵州茅台", Price: 1850.00,
	}, 5*time.Second)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/market/quote?code=600519", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var quote models.Quote
	json.Unmarshal(w.Body.Bytes(), &quote)
	if quote.Price != 1850.00 {
		t.Errorf("expected cached price 1850.00, got %.2f", quote.Price)
	}
}
