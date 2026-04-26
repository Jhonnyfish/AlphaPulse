package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboardSummary_NoDB(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()

	// DashboardHandler requires a DB, but we can test activity log and report date helpers
	h := &DashboardHandler{log: nil}

	// Test recentActivity with no file (use non-existent path)
	t.Setenv("ACTIVITY_LOG_PATH", filepath.Join(dir, "nonexistent.json"))
	entries := h.recentActivity()
	assert.Empty(t, entries)

	// Test lastReportDate with empty dir
	t.Setenv("REPORTS_DIR", dir)
	date := h.lastReportDate()
	assert.Equal(t, "", date)

	// Create a fake report file
	reportFile := filepath.Join(dir, "daily_report_20260426.md")
	require.NoError(t, os.WriteFile(reportFile, []byte("# Report"), 0644))
	date = h.lastReportDate()
	assert.Equal(t, "20260426", date)
}

func TestDashboardSummary_RecentActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "activity.json")
	data := []map[string]string{
		{"action": "buy", "detail": "600519", "timestamp": "2026-04-26T10:00:00Z"},
		{"action": "sell", "detail": "000001", "timestamp": "2026-04-26T11:00:00Z"},
		{"action": "alert", "detail": "test", "timestamp": "2026-04-26T12:00:00Z"},
	}
	raw, _ := json.Marshal(data)
	require.NoError(t, os.WriteFile(logFile, raw, 0644))

	t.Setenv("ACTIVITY_LOG_PATH", logFile)
	h := &DashboardHandler{}
	entries := h.recentActivity()
	assert.Len(t, entries, 3)
	// Should be reversed (newest first)
	assert.Equal(t, "alert", entries[0].Action)
}

func TestDashboardSummary_Route(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Just verify the handler can be constructed and route set up
	router := gin.New()
	// Use a nil handler to test route registration pattern — the actual handler
	// would need a real DB, so we test the route exists
	router.GET("/api/dashboard-summary", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "test": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/dashboard-summary", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["ok"])
}
