package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSignalHistory(t *testing.T) {
	// Create a temp signal history file
	dir := t.TempDir()
	path := filepath.Join(dir, "signal_history.json")

	entries := []signalHistoryEntry{
		{Timestamp: "2026-04-25T10:00:00Z", Code: "600519", Name: "贵州茅台", Level: "positive", Message: "test1"},
		{Timestamp: "2026-04-25T09:00:00Z", Code: "000001", Name: "平安银行", Level: "danger", Message: "test2"},
		{Timestamp: "2026-04-25T08:00:00Z", Code: "600519", Name: "贵州茅台", Level: "info", Message: "test3"},
	}
	data, err := json.Marshal(entries)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))

	// Test loading
	loaded := loadSignalHistory(path)
	assert.Len(t, loaded, 3)
	assert.Equal(t, "600519", loaded[0].Code)
}

func TestLoadSignalHistoryMissing(t *testing.T) {
	loaded := loadSignalHistory("/nonexistent/path.json")
	assert.Empty(t, loaded)
}

func TestEmptyAnomalies(t *testing.T) {
	result := emptyAnomalies()
	assert.True(t, result["ok"].(bool))
	assert.Equal(t, 0, result["scanned"])
	assert.NotEmpty(t, result["fetched_at"])
}

func TestMergeMap(t *testing.T) {
	base := map[string]interface{}{"ok": true, "cached": true}
	overlay := map[string]interface{}{"code": "600519", "name": "贵州茅台"}
	result := mergeMap(base, overlay)
	assert.True(t, result["ok"].(bool))
	assert.True(t, result["cached"].(bool))
	assert.Equal(t, "600519", result["code"])
	assert.Equal(t, "贵州茅台", result["name"])
}

func TestStructToMap(t *testing.T) {
	type testStruct struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	result := structToMap(testStruct{Code: "600519", Name: "贵州茅台"})
	assert.Equal(t, "600519", result["code"])
	assert.Equal(t, "贵州茅台", result["name"])
}
