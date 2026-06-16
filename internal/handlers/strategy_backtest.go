package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
)

// StrategyBacktestHandler handles the new strategy backtest engine API.
type StrategyBacktestHandler struct {
	eastMoney *services.EastMoneyService
	tushareDB *services.TushareDB
}

// NewStrategyBacktestHandler creates a new handler.
func NewStrategyBacktestHandler(eastMoney *services.EastMoneyService) *StrategyBacktestHandler {
	return &StrategyBacktestHandler{eastMoney: eastMoney}
}

// SetTushareDB sets the local database data source as primary source.
func (h *StrategyBacktestHandler) SetTushareDB(db *services.TushareDB) {
	h.tushareDB = db
}

// Backtest handles GET /api/strategy/backtest?code=xxx&days=30&strategy=balanced
func (h *StrategyBacktestHandler) Backtest(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing code"})
		return
	}

	days := 30
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}
	if days > 500 {
		days = 500
	}

	strategyID := strings.TrimSpace(c.Query("strategy"))
	if strategyID == "" {
		strategyID = "balanced"
	}

	if h.tushareDB != nil {
		result := services.RunStrategyBacktestWithStrategy(c.Request.Context(), h.tushareDB, code, days, strategyID)
		if result.Error == "" {
			c.JSON(http.StatusOK, result)
			return
		}
	}

	result := services.RunStrategyBacktestWithStrategy(c.Request.Context(), h.eastMoney, code, days, strategyID)
	c.JSON(http.StatusOK, result)
}
