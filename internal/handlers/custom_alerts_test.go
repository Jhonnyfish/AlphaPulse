package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeStockCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Already-prefixed formats (len 8)
		{"sh prefix lowercase", "sh600176", "sh600176"},
		{"sz prefix lowercase", "sz000001", "sz000001"},
		{"bj prefix lowercase", "bj430047", "bj430047"},
		{"sh prefix uppercase", "SH600176", "sh600176"},
		{"sz prefix uppercase", "SZ000001", "sz000001"},
		{"bj prefix uppercase", "BJ430047", "bj430047"},
		{"sh prefix with spaces", "  sh600176  ", "sh600176"},

		// Suffix formats (.sh/.sz/.bj)
		{"dot sh suffix", "600176.sh", "sh600176"},
		{"dot sz suffix", "000001.sz", "sz000001"},
		{"dot bj suffix", "430047.bj", "bj430047"},
		{"dot sh uppercase", "600176.SH", "sh600176"},
		{"dot sz uppercase", "000001.SZ", "sz000001"},

		// Bare 6-digit codes — infer prefix
		{"6-digit starts with 6 → sh", "600519", "sh600519"},
		{"6-digit starts with 9 → sh", "900901", "sh900901"},
		{"6-digit starts with 0 → sz", "000001", "sz000001"},
		{"6-digit starts with 3 → sz", "300750", "sz300750"},
		{"6-digit starts with 4 → bj", "430047", "bj430047"},
		{"6-digit starts with 8 → bj", "830799", "bj830799"},

		// Prefixed but wrong length → empty
		{"sh prefix too short", "sh60017", ""},
		{"sh prefix too long", "sh6001760", ""},
		{"sz prefix wrong length", "sz00001", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeStockCode(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalizeStockCodeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},
		{"tab and newline", "\t\n", ""},
		{"too short", "12345", ""},
		{"too long", "1234567", ""},
		{"5 digits", "60017", ""},
		{"7 digits", "6001760", ""},
		{"single char", "a", ""},
		{"letters only", "abcdef", ""},
		{"starts with 1", "100000", ""},
		{"starts with 2", "200000", ""},
		{"starts with 5", "500000", ""},
		{"starts with 7", "700000", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeStockCode(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCustomAlertsCreateInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPost, "/api/custom-alerts",
		strings.NewReader("this is not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := NewCustomAlertsHandler(nil, nil)
	handler.Create(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_BODY")
}

func TestCustomAlertsCreateMissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"code":"","type":"price_above","threshold":100}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/custom-alerts",
		strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := NewCustomAlertsHandler(nil, nil)
	handler.Create(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_CODE")
}

func TestCustomAlertsCreateInvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"code":"600519","type":"invalid_type","threshold":100}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/custom-alerts",
		strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := NewCustomAlertsHandler(nil, nil)
	handler.Create(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_TYPE")
}

func TestCustomAlertsCreateInvalidThreshold(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"code":"600519","type":"price_above","threshold":0}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/custom-alerts",
		strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := NewCustomAlertsHandler(nil, nil)
	handler.Create(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_THRESHOLD")
}

func TestCustomAlertsDeleteEmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/custom-alerts/", nil)
	// Simulate empty :id param (gin would strip trailing slash)
	c.Params = gin.Params{}

	handler := NewCustomAlertsHandler(nil, nil)
	handler.Delete(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_ID")
}
