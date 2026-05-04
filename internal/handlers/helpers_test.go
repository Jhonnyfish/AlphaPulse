package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "alphapulse/internal/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeError(c, http.StatusBadRequest, "TEST_ERROR", "something went wrong")

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "TEST_ERROR", resp.Code)
	assert.Equal(t, "something went wrong", resp.Error)
}

func TestWriteAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := &apperrors.AppError{
		Code:    http.StatusNotFound,
		Message: "resource not found",
	}
	writeAppError(c, err)

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "resource not found", resp["error"])
}

func TestCleanCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{" 600519 ", "600519"},
		{"000001", "000001"},
		{"\t300750\n", "300750"},
		{"", ""},
		{"  ", ""},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, cleanCode(tc.input))
	}
}

func TestRandomString(t *testing.T) {
	s1, err := randomString(8)
	require.NoError(t, err)
	assert.Len(t, s1, 8)

	s2, err := randomString(8)
	require.NoError(t, err)
	assert.Len(t, s2, 8)

	// Two random strings should (with overwhelming probability) differ
	assert.NotEqual(t, s1, s2)
}

func TestHashToken(t *testing.T) {
	h1 := hashToken("test-token")
	h2 := hashToken("test-token")
	h3 := hashToken("other-token")

	// Same input → same hash
	assert.Equal(t, h1, h2)
	// Different input → different hash
	assert.NotEqual(t, h1, h3)
	// SHA-256 produces 64 hex chars
	assert.Len(t, h1, 64)
}

func TestParseJSONArrayHelper(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []string
	}{
		{"nil input", nil, []string{}},
		{"empty bytes", []byte{}, []string{}},
		{"valid array", []byte(`["tag1","tag2"]`), []string{"tag1", "tag2"}},
		{"invalid json", []byte(`not json`), []string{}},
		{"empty array", []byte(`[]`), []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseJSONArray(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
