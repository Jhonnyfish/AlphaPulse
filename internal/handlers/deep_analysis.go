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
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DeepAnalysisRequest is the request for deep analysis.
type DeepAnalysisRequest struct {
	Code string `json:"code" binding:"required"`
}

// DeepAnalysisResponse is the response for deep analysis.
type DeepAnalysisResponse struct {
	OK     bool   `json:"ok"`
	Code   string `json:"code"`
	Report string `json:"report,omitempty"`
	Error  string `json:"error,omitempty"`
}

// DeepAnalysisHandler handles deep analysis requests.
type DeepAnalysisHandler struct {
	tushareDB *services.TushareDB
}

// NewDeepAnalysisHandler creates a new DeepAnalysisHandler.
func NewDeepAnalysisHandler() *DeepAnalysisHandler {
	return &DeepAnalysisHandler{}
}

// SetTushareDB sets the TushareDB service.
func (h *DeepAnalysisHandler) SetTushareDB(db *services.TushareDB) {
	h.tushareDB = db
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

	// Fetch stock data from DB directly
	stockData := h.fetchStockData(code)

	// Build prompt with data context
	var prompt string
	if stockData != "" {
		prompt = fmt.Sprintf("对%s(%s)进行深度分析。\n\n以下是该股票的最新数据（来自AlphaPulse数据库）：\n\n%s\n\n请基于以上数据，结合你的研究，生成完整的深度分析报告。",
			h.getStockNameFromData(stockData), code, stockData)
	} else {
		prompt = fmt.Sprintf("对%s进行深度分析", code)
	}

	logger.Info("triggering deep analysis",
		zap.String("code", code),
		zap.Bool("has_data", stockData != ""))

	// Call Hermes Agent CLI — deep analysis needs time for web research
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	hermesPath := "/home/finn/.local/bin/hermes"
	cmd := exec.CommandContext(ctx, hermesPath, "chat", "-q", prompt, "--skills", "stock-deep-analysis", "-Q")
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

// fetchStockData fetches stock data from DB and returns it as a formatted string.
func (h *DeepAnalysisHandler) fetchStockData(code string) string {
	if h.tushareDB == nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	quote, err := h.tushareDB.FetchQuoteFromDB(ctx, code)
	if err != nil {
		logger.Warn("failed to fetch quote for deep analysis", zap.String("code", code), zap.Error(err))
		return ""
	}

	// Format as structured data
	data := map[string]interface{}{
		"code":           quote.Code,
		"name":           quote.Name,
		"price":          quote.Price,
		"change":         quote.Change,
		"change_percent": quote.ChangePercent,
		"open":           quote.Open,
		"high":           quote.High,
		"low":            quote.Low,
		"prev_close":     quote.PrevClose,
		"volume":         quote.Volume,
		"turnover":       quote.Turnover,
		"pe":             quote.PE,
		"pb":             quote.PB,
		"total_mv":       quote.TotalMV,
		"amplitude":      quote.Amplitude,
		"updated_at":     quote.UpdatedAt,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}

	return string(jsonData)
}

// getStockNameFromData extracts stock name from JSON data string.
func (h *DeepAnalysisHandler) getStockNameFromData(data string) string {
	var result struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return ""
	}
	return result.Name
}
