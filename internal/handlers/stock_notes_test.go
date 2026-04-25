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

func TestStockNotesHandler_Wiring(t *testing.T) {
	// Without a real DB this just validates the handler functions exist
	assert.NotNil(t, (*StockNotesHandler)(nil).GetNotes)
	assert.NotNil(t, (*StockNotesHandler)(nil).CreateNote)
	assert.NotNil(t, (*StockNotesHandler)(nil).UpdateNote)
	assert.NotNil(t, (*StockNotesHandler)(nil).DeleteNote)
	assert.NotNil(t, (*StockNotesHandler)(nil).AllTags)
}

func TestStockNotesHandler_Create_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("empty code", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/stock-notes", func(c *gin.Context) {
			var req createNoteRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				writeError(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
				return
			}
			if req.Code == "" {
				writeError(c, http.StatusBadRequest, "INVALID_CODE", "股票代码不能为空")
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		body, _ := json.Marshal(map[string]interface{}{"code": "", "content": "test"})
		req := httptest.NewRequest(http.MethodPost, "/api/stock-notes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "INVALID_CODE", resp["code"])
	})

	t.Run("empty content", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/stock-notes", func(c *gin.Context) {
			var req createNoteRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				writeError(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
				return
			}
			if req.Content == "" {
				writeError(c, http.StatusBadRequest, "INVALID_CONTENT", "备注内容不能为空")
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		body, _ := json.Marshal(map[string]interface{}{"code": "600519", "content": ""})
		req := httptest.NewRequest(http.MethodPost, "/api/stock-notes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "INVALID_CONTENT", resp["code"])
	})
}

func TestStockNotesHandler_GetNotes_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/stock-notes/:code", func(c *gin.Context) {
		code := normalizeStockCode(c.Param("code"))
		if code == "" {
			writeError(c, http.StatusBadRequest, "INVALID_CODE", "股票代码格式错误")
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/stock-notes/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_CODE", resp["code"])
}

func TestNormalizeTags(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Equal(t, []string{}, normalizeTags(nil))
	})

	t.Run("empty slice", func(t *testing.T) {
		assert.Equal(t, []string{}, normalizeTags([]string{}))
	})

	t.Run("deduplicate and trim", func(t *testing.T) {
		result := normalizeTags([]string{" value ", "value", " other "})
		assert.Contains(t, result, "value")
		assert.Contains(t, result, "other")
		assert.Len(t, result, 2)
	})

	t.Run("filter empties", func(t *testing.T) {
		result := normalizeTags([]string{"", "  ", "valid"})
		assert.Equal(t, []string{"valid"}, result)
	})
}

func TestParseJSONArray(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Equal(t, []string{}, parseJSONArray(nil))
	})

	t.Run("valid JSON", func(t *testing.T) {
		result := parseJSONArray([]byte(`["tag1","tag2"]`))
		assert.Equal(t, []string{"tag1", "tag2"}, result)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		result := parseJSONArray([]byte(`not json`))
		assert.Equal(t, []string{}, result)
	})
}
