package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"alphapulse/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DeepAnalysisRequest is the request for deep analysis.
type DeepAnalysisRequest struct {
	Code string `json:"code" binding:"required"`
}

// DeepAnalysisResponse is the response for deep analysis.
type DeepAnalysisResponse struct {
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Report  string `json:"report,omitempty"`
	Error   string `json:"error,omitempty"`
}

// DeepAnalysisHandler handles deep analysis requests.
type DeepAnalysisHandler struct{}

// NewDeepAnalysisHandler creates a new DeepAnalysisHandler.
func NewDeepAnalysisHandler() *DeepAnalysisHandler {
	return &DeepAnalysisHandler{}
}

// Analyze handles POST /api/deep-analysis
// @Summary 触发深度分析
// @Description 调用Hermes Agent对股票进行深度分析
// @Tags deep-analysis
// @Accept json
// @Produce json
// @Param request body DeepAnalysisRequest true "股票代码"
// @Success 200 {object} DeepAnalysisResponse
// @Router /api/deep-analysis [post]
func (h *DeepAnalysisHandler) Analyze(c *gin.Context) {
	var req DeepAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, DeepAnalysisResponse{
			OK:    false,
			Error: "invalid request: " + err.Error(),
		})
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" || len(code) > 6 {
		c.JSON(http.StatusBadRequest, DeepAnalysisResponse{
			OK:    false,
			Error: "invalid stock code",
		})
		return
	}

	// Get stock name from API (optional, for better prompt)
	stockName := h.getStockName(code)

	// Build prompt for Hermes Agent
	prompt := fmt.Sprintf("对%s(%s)进行深度分析", stockName, code)

	logger.Info("triggering deep analysis",
		zap.String("code", code),
		zap.String("name", stockName))

	// Call Hermes Agent CLI
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "hermes", "run", prompt, "--skill", "stock-deep-analysis")
	output, err := cmd.CombinedOutput()

	if err != nil {
		logger.Error("hermes agent failed",
			zap.String("code", code),
			zap.Error(err),
			zap.String("output", string(output)))
		c.JSON(http.StatusInternalServerError, DeepAnalysisResponse{
			OK:    false,
			Code:  code,
			Error: "analysis failed: " + err.Error(),
		})
		return
	}

	report := strings.TrimSpace(string(output))
	if report == "" {
		c.JSON(http.StatusInternalServerError, DeepAnalysisResponse{
			OK:    false,
			Code:  code,
			Error: "empty report",
		})
		return
	}

	logger.Info("deep analysis completed",
		zap.String("code", code),
		zap.Int("report_length", len(report)))

	c.JSON(http.StatusOK, DeepAnalysisResponse{
		OK:     true,
		Code:   code,
		Report: report,
	})
}

// getStockName fetches stock name from AlphaPulse API.
func (h *DeepAnalysisHandler) getStockName(code string) string {
	// Try to get from local API
	cmd := exec.Command("curl", "-s",
		fmt.Sprintf("http://localhost:8080/api/analyze?code=%s", code))
	output, err := cmd.Output()
	if err != nil {
		return code
	}

	var result struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return code
	}

	if result.Name != "" {
		return result.Name
	}
	return code
}
