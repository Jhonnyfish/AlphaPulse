package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	apperrors "alphapulse/internal/errors"

	"github.com/gin-gonic/gin"
)

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, errorResponse{
		Error: message,
		Code:  code,
	})
}

// writeAppError writes a unified AppError as a JSON response.
func writeAppError(c *gin.Context, err *apperrors.AppError) {
	c.JSON(err.Code, gin.H{
		"error": err.Message,
		"code":  http.StatusText(err.Code),
	})
}

func cleanCode(value string) string {
	return strings.TrimSpace(value)
}

func randomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	value := hex.EncodeToString(bytes)
	if len(value) > length {
		value = value[:length]
	}
	return strings.ToUpper(value), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// parseJSONArray decodes a JSON byte slice into a string slice.
func parseJSONArray(data []byte) []string {
	if len(data) == 0 {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal(data, &tags); err != nil {
		return []string{}
	}
	return tags
}
