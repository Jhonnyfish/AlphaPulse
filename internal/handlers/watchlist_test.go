package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testWatchlistHandler() *WatchlistHandler {
	return &WatchlistHandler{logger: zap.NewNop()}
}

func TestWatchlistAdd_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/watchlist", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	h := testWatchlistHandler()
	h.Add(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_REQUEST")
}

func TestWatchlistAdd_EmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"code":"","name":"test"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/watchlist", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h := testWatchlistHandler()
	h.Add(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_CODE")
}

func TestWatchlistDelete_EmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/watchlist/", nil)
	c.Params = gin.Params{{Key: "code", Value: ""}}

	h := testWatchlistHandler()
	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_CODE")
}

func TestWatchlistBatchAdd_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/watchlist/batch", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	h := testWatchlistHandler()
	h.BatchAdd(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_REQUEST")
}

func TestWatchlistBatchAdd_EmptyCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"codes":[]}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/watchlist/batch", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h := testWatchlistHandler()
	h.BatchAdd(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_CODES")
}

func TestWatchlistSync_NoAlpha300(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/watchlist/sync", nil)

	h := testWatchlistHandler()
	h.Sync(c)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "ALPHA300_NOT_CONFIGURED")
}
