package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CompareHandler handles stock comparison endpoints.
type CompareHandler struct {
	eastMoney *services.EastMoneyService
	tencent   *services.TencentService
}

// NewCompareHandler creates a new CompareHandler.
func NewCompareHandler(eastMoney *services.EastMoneyService, tencent *services.TencentService) *CompareHandler {
	return &CompareHandler{
		eastMoney: eastMoney,
		tencent:   tencent,
	}
}

// SectorCompare returns sector comparison data for a stock.
// @Summary      Sector comparison
// @Description  Shows which sectors a stock belongs to, top 5 stocks in primary sector, and rank
// @Tags         compare
// @Accept       json
// @Produce      json
// @Param        code  query    string  true  "Stock code, e.g. 600176"
// @Success      200   {object}  models.SectorCompareResult
// @Router       /api/compare/sector [get]
func (h *CompareHandler) SectorCompare(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "code is required"})
		return
	}
	code = services.NormalizeCode(code)

	ctx := c.Request.Context()

	// 1. Get stock sectors
	sectors, err := h.eastMoney.FetchStockSectors(ctx, code)
	if err != nil {
		zap.L().Error("fetch stock sectors failed", zap.String("code", code), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	if len(sectors) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"ok":           true,
			"code":         code,
			"sector_name":  "",
			"board_code":   "",
			"top5":         []interface{}{},
			"current_rank": 0,
			"total_count":  0,
		})
		return
	}

	primary := sectors[0]
	boardCode := primary.Code
	sectorName := primary.Name

	if boardCode == "" {
		c.JSON(http.StatusOK, gin.H{
			"ok":           true,
			"code":         code,
			"sector_name":  sectorName,
			"board_code":   "",
			"top5":         []interface{}{},
			"current_rank": 0,
			"total_count":  0,
		})
		return
	}

	// 2. Fetch sector members
	members, err := h.eastMoney.FetchSectorMembers(ctx, boardCode, 200)
	if err != nil {
		zap.L().Error("fetch sector members failed", zap.String("board", boardCode), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	// 3. Find rank
	rank := 0
	for idx, m := range members {
		if m.Code == code {
			rank = idx + 1
			break
		}
	}

	// 4. Top 5
	top5 := members
	if len(top5) > 5 {
		top5 = top5[:5]
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":           true,
		"code":         code,
		"sector_name":  sectorName,
		"board_code":   boardCode,
		"top5":         top5,
		"current_rank": rank,
		"total_count":  len(members),
	})
}

// BacktestCompare runs backtests for multiple stocks and compares results.
// @Summary      Multi-stock backtest comparison
// @Description  Run backtest for 2-5 stocks and return comparison results
// @Tags         compare
// @Accept       json
// @Produce      json
// @Param        codes  query    string  true  "Comma-separated stock codes (2-5), e.g. 600176,000001"
// @Param        days   query    int     false "Backtest period in days (10-240, default 30)"
// @Success      200    {object}  map[string]interface{}
// @Router       /api/compare/backtest [get]
func (h *CompareHandler) BacktestCompare(c *gin.Context) {
	codesRaw := c.Query("codes")
	if codesRaw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "codes is required"})
		return
	}

	// Parse and deduplicate codes
	seen := make(map[string]bool)
	var uniqueCodes []string
	for _, raw := range strings.FieldsFunc(codesRaw, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\t'
	}) {
		code := strings.TrimSpace(raw)
		if code == "" {
			continue
		}
		code = services.NormalizeCode(code)
		if !seen[code] {
			seen[code] = true
			uniqueCodes = append(uniqueCodes, code)
		}
	}

	if len(uniqueCodes) < 2 || len(uniqueCodes) > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请输入2-5只股票代码"})
		return
	}

	days := 30
	if d := c.Query("days"); d != "" {
		// Parse days - ignore error, keep default
		var parsed int
		if _, err := fmt.Sscanf(d, "%d", &parsed); err == nil && parsed >= 10 && parsed <= 240 {
			days = parsed
		}
	}

	// Run backtests in parallel
	ctx := c.Request.Context()
	results := make(map[string]services.BacktestResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, code := range uniqueCodes {
		wg.Add(1)
		go func(cd string) {
			defer wg.Done()
			bt := services.RunBacktest(ctx, h.eastMoney, cd, days)
			mu.Lock()
			results[cd] = bt
			mu.Unlock()
		}(code)
	}
	wg.Wait()

	// Build comparison
	type compareItem struct {
		Code         string               `json:"code"`
		Name         string               `json:"name"`
		SignalCount  int                  `json:"signal_count"`
		WinRate      float64              `json:"win_rate"`
		AvgReturnPct float64              `json:"avg_return_pct"`
		MaxDrawdown  float64              `json:"max_drawdown_pct"`
		EquityCurve  []float64            `json:"equity_curve"`
		Trades       []services.BacktestTrade `json:"trades"`
		Error        string               `json:"error,omitempty"`
	}

	comparison := make([]compareItem, 0, len(uniqueCodes))
	for _, code := range uniqueCodes {
		bt := results[code]
		equity := []float64{1.0}
		for _, t := range bt.Trades {
			last := equity[len(equity)-1]
			equity = append(equity, last*(1+t.ReturnPct/100))
		}
		comparison = append(comparison, compareItem{
			Code:         bt.Code,
			Name:         bt.Code, // Use code as name fallback
			SignalCount:  bt.SignalCount,
			WinRate:      bt.WinRate,
			AvgReturnPct: bt.AvgReturnPct,
			MaxDrawdown:  bt.MaxDrawdown,
			EquityCurve:  equity,
			Trades:       bt.Trades,
			Error:        bt.Error,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":     true,
		"days":   days,
		"count":  len(comparison),
		"results": comparison,
	})
}
