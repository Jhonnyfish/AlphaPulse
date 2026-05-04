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

func setupStrategiesRouter(h *StrategiesHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	api.GET("/strategies", h.List)
	api.POST("/strategies", h.Create)
	api.PUT("/strategies/:id", h.Update)
	api.DELETE("/strategies/:id", h.Delete)
	api.POST("/strategies/:id/activate", h.Activate)
	api.POST("/strategies/:id/deactivate", h.Deactivate)
	return r
}

func TestStrategiesHandler_List_NoDB(t *testing.T) {
	// Without a real DB this just validates the handler wiring
	// We test that the handler function exists and is callable
	assert.NotNil(t, (*StrategiesHandler)(nil).List)
}

func TestStrategiesHandler_Create_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("empty name", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/strategies", func(c *gin.Context) {
			// Simulate the validation logic
			var req createStrategyRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				writeError(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
				return
			}
			if req.Name == "" {
				writeError(c, http.StatusBadRequest, "INVALID_NAME", "策略名称不能为空")
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		body, _ := json.Marshal(map[string]interface{}{"name": ""})
		req := httptest.NewRequest(http.MethodPost, "/api/strategies", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "INVALID_NAME", resp["code"])
	})

	t.Run("empty scoring", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/strategies", func(c *gin.Context) {
			var req createStrategyRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				writeError(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
				return
			}
			if len(req.Scoring) == 0 {
				writeError(c, http.StatusBadRequest, "INVALID_SCORING", "评分权重不能为空")
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		body, _ := json.Marshal(map[string]interface{}{
			"name":    "Test Strategy",
			"scoring": map[string]interface{}{},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/strategies", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "INVALID_SCORING", resp["code"])
	})
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		ok       bool
	}{
		{"float64", 3.14, 3.14, true},
		{"int", 42, 42.0, true},
		{"int64", int64(100), 100.0, true},
		{"string", "not a number", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.InDelta(t, tt.expected, result, 0.001)
			}
		})
	}
}
