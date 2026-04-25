package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestReportsList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "daily_report_20260101.md"), []byte("# Report\n\nSome preview line here."), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "analysis_20260102.md"), []byte("# Analysis\n\nAnother preview."), 0o644)

	h := &ReportsHandler{reportsDir: tmpDir}

	r := gin.New()
	r.GET("/api/reports", h.ListReports)

	req := httptest.NewRequest(http.MethodGet, "/api/reports", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "daily_report_20260101.md")
	assert.Contains(t, w.Body.String(), "analysis_20260102.md")
	assert.Contains(t, w.Body.String(), `"ok":true`)
}

func TestReportsListEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	h := &ReportsHandler{reportsDir: tmpDir}

	r := gin.New()
	r.GET("/api/reports", h.ListReports)

	req := httptest.NewRequest(http.MethodGet, "/api/reports", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Should return empty array, not null
	assert.Contains(t, w.Body.String(), `"reports":[]`)
}

func TestGetReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	content := "# Test Report\n\nReport content here."
	os.WriteFile(filepath.Join(tmpDir, "test_report.md"), []byte(content), 0o644)

	h := &ReportsHandler{reportsDir: tmpDir}

	r := gin.New()
	r.GET("/api/reports/:filename", h.GetReport)

	req := httptest.NewRequest(http.MethodGet, "/api/reports/test_report.md", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Report content here.")
	assert.Contains(t, w.Body.String(), `"ok":true`)
}

func TestGetReportNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	h := &ReportsHandler{reportsDir: tmpDir}

	r := gin.New()
	r.GET("/api/reports/:filename", h.GetReport)

	req := httptest.NewRequest(http.MethodGet, "/api/reports/nonexistent.md", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRedirectToAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &ReportsHandler{}

	r := gin.New()
	r.GET("/reports", h.RedirectToAPI)

	req := httptest.NewRequest(http.MethodGet, "/reports", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/api/reports")
}

func TestDailyReportList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "daily_report_20260401.md"), []byte("# Daily\n\nReport 1."), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "daily_report_20260402.md"), []byte("# Daily\n\nReport 2."), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "other_report.md"), []byte("# Other"), 0o644)

	h := &ReportsHandler{reportsDir: tmpDir}

	r := gin.New()
	r.GET("/api/daily-report/list", h.DailyReportList)

	req := httptest.NewRequest(http.MethodGet, "/api/daily-report/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "daily_report_20260401.md")
	assert.Contains(t, body, "daily_report_20260402.md")
	// Should NOT include non-daily reports
	assert.NotContains(t, body, "other_report.md")
}

func TestDailyReportLatest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	oldPath := filepath.Join(tmpDir, "daily_report_20260401.md")
	newPath := filepath.Join(tmpDir, "daily_report_20260405.md")
	os.WriteFile(oldPath, []byte("# Old Report"), 0o644)
	os.WriteFile(newPath, []byte("# Latest Report"), 0o644)

	// Set different modification times
	now := time.Now()
	os.Chtimes(oldPath, now.Add(-2*time.Hour), now.Add(-2*time.Hour))
	os.Chtimes(newPath, now, now)

	h := &ReportsHandler{reportsDir: tmpDir}

	r := gin.New()
	r.GET("/api/daily-report/latest", h.DailyReportLatest)

	req := httptest.NewRequest(http.MethodGet, "/api/daily-report/latest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "daily_report_20260405.md")
	assert.Contains(t, w.Body.String(), "Latest Report")
}

func TestDailyReportLatestEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	h := &ReportsHandler{reportsDir: tmpDir}

	r := gin.New()
	r.GET("/api/daily-report/latest", h.DailyReportLatest)

	req := httptest.NewRequest(http.MethodGet, "/api/daily-report/latest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestClassifyReport(t *testing.T) {
	assert.Equal(t, "daily_report", classifyReport("daily_report_20260101.md"))
	assert.Equal(t, "8dim_analysis", classifyReport("8dim_600001.md"))
	assert.Equal(t, "other", classifyReport("random_report.md"))
}

func TestExtractDate(t *testing.T) {
	assert.Equal(t, "20260101", extractDate("daily_report_20260101.md"))
	assert.Equal(t, "20260405", extractDate("analysis_20260405.md"))
}

func TestSafePath(t *testing.T) {
	tmpDir := t.TempDir()
	h := &ReportsHandler{reportsDir: tmpDir}

	// Valid path
	path, ok := h.safePath("report.md")
	assert.True(t, ok)
	assert.Equal(t, filepath.Join(tmpDir, "report.md"), path)

	// Absolute path rejected
	_, ok = h.safePath("/etc/passwd")
	assert.False(t, ok)

	// Cleaned traversal: foo/../bar.md → bar.md (valid after cleaning)
	path, ok = h.safePath("foo/../bar.md")
	assert.True(t, ok)
	assert.Equal(t, filepath.Join(tmpDir, "bar.md"), path)
}
