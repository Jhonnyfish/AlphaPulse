package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"alphapulse/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ScoreHistoryEntry represents a single score history record.
type ScoreHistoryEntry struct {
	Score      float64            `json:"score"`
	Dimensions map[string]float64 `json:"dimensions"`
	RecordedAt string             `json:"recorded_at"`
}

// ScoreTrend holds trend information for a score.
type ScoreTrend struct {
	Current  float64 `json:"current"`
	Prev7D   float64 `json:"prev_7d"`
	Prev30D  float64 `json:"prev_30d"`
	Change7D float64 `json:"change_7d"`
	Change30D float64 `json:"change_30d"`
	Trend7D  string  `json:"trend_7d"`  // "rising", "falling", "stable"
	Trend30D string  `json:"trend_30d"`
}

// DimensionTrend holds trend for a single dimension.
type DimensionTrend struct {
	Current  float64 `json:"current"`
	Change7D float64 `json:"change_7d"`
	Trend    string  `json:"trend"`
}

// ComparisonData holds comparison with sector/market.
type ComparisonData struct {
	// Stock metrics
	StockScore    float64 `json:"stock_score"`
	StockChange5D float64 `json:"stock_change_5d"`
	StockChange20D float64 `json:"stock_change_20d"`
	// Sector average
	SectorAvgScore    float64 `json:"sector_avg_score"`
	SectorAvgChange5D float64 `json:"sector_avg_change_5d"`
	SectorName        string  `json:"sector_name"`
	// Market (CSI 300)
	MarketChange5D  float64 `json:"market_change_5d"`
	MarketChange20D float64 `json:"market_change_20d"`
	// Relative strength
	VsSector  float64 `json:"vs_sector"`  // positive = outperforming
	VsMarket  float64 `json:"vs_market"`  // positive = outperforming
}

// ScoreHistoryResponse is the full response for /api/score-history/:code.
type ScoreHistoryResponse struct {
	Code       string              `json:"code"`
	Count      int                 `json:"count"`
	History    []ScoreHistoryEntry `json:"history"`
	Trend      *ScoreTrend         `json:"trend,omitempty"`
	DimTrends  map[string]DimensionTrend `json:"dim_trends,omitempty"`
	Comparison *ComparisonData     `json:"comparison,omitempty"`
}

// ScoreHistoryHandler handles score history requests.
type ScoreHistoryHandler struct {
	db *pgxpool.Pool
}

// NewScoreHistoryHandler creates a new ScoreHistoryHandler.
func NewScoreHistoryHandler(db *pgxpool.Pool) *ScoreHistoryHandler {
	return &ScoreHistoryHandler{db: db}
}

// GetHistory handles GET /api/score-history/:code
// @Summary 获取股票评分历史
// @Description 获取指定股票的历史评分记录（最近30条）及趋势分析
// @Tags score-history
// @Accept json
// @Produce json
// @Param code path string true "股票代码（6位数字）"
// @Success 200 {object} ScoreHistoryResponse
// @Router /api/score-history/{code} [get]
func (h *ScoreHistoryHandler) GetHistory(c *gin.Context) {
	code := c.Param("code")
	if code == "" || len(code) > 6 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid code"})
		return
	}

	rows, err := h.db.Query(c.Request.Context(),
		`SELECT score, dimensions, recorded_at
		 FROM score_history
		 WHERE code = $1
		 ORDER BY recorded_at DESC
		 LIMIT 30`, code)
	if err != nil {
		logger.Error("score history query failed",
			zap.String("code", code),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "query failed"})
		return
	}
	defer rows.Close()

	var history []ScoreHistoryEntry
	for rows.Next() {
		var score float64
		var dimensionsJSON []byte
		var recordedAt time.Time

		if err := rows.Scan(&score, &dimensionsJSON, &recordedAt); err != nil {
			logger.Error("score history scan failed", zap.Error(err))
			continue
		}

		dimensions := make(map[string]float64)
		if len(dimensionsJSON) > 0 {
			_ = json.Unmarshal(dimensionsJSON, &dimensions)
		}

		history = append(history, ScoreHistoryEntry{
			Score:      score,
			Dimensions: dimensions,
			RecordedAt: recordedAt.Format(time.RFC3339),
		})
	}

	if history == nil {
		history = []ScoreHistoryEntry{}
	}

	// Calculate trends
	trend := h.calculateTrend(history)
	dimTrends := h.calculateDimensionTrends(history)

	// Calculate comparison with sector/market
	comparison := h.calculateComparison(c.Request.Context(), code)

	c.JSON(http.StatusOK, ScoreHistoryResponse{
		Code:       code,
		Count:      len(history),
		History:    history,
		Trend:      trend,
		DimTrends:  dimTrends,
		Comparison: comparison,
	})
}

// calculateTrend computes 7-day and 30-day score trends.
func (h *ScoreHistoryHandler) calculateTrend(history []ScoreHistoryEntry) *ScoreTrend {
	if len(history) == 0 {
		return nil
	}

	current := history[0].Score

	// Find 7-day ago score
	prev7D := current
	for _, entry := range history {
		t, _ := time.Parse(time.RFC3339, entry.RecordedAt)
		if time.Since(t) >= 7*24*time.Hour {
			prev7D = entry.Score
			break
		}
	}

	// Find 30-day ago score
	prev30D := current
	for _, entry := range history {
		t, _ := time.Parse(time.RFC3339, entry.RecordedAt)
		if time.Since(t) >= 30*24*time.Hour {
			prev30D = entry.Score
			break
		}
	}

	change7D := current - prev7D
	change30D := current - prev30D

	trend7D := "stable"
	if change7D > 3 {
		trend7D = "rising"
	} else if change7D < -3 {
		trend7D = "falling"
	}

	trend30D := "stable"
	if change30D > 5 {
		trend30D = "rising"
	} else if change30D < -5 {
		trend30D = "falling"
	}

	return &ScoreTrend{
		Current:   current,
		Prev7D:    prev7D,
		Prev30D:   prev30D,
		Change7D:  change7D,
		Change30D: change30D,
		Trend7D:   trend7D,
		Trend30D:  trend30D,
	}
}

// calculateDimensionTrends computes per-dimension trends.
func (h *ScoreHistoryHandler) calculateDimensionTrends(history []ScoreHistoryEntry) map[string]DimensionTrend {
	if len(history) < 2 {
		return nil
	}

	dimTrends := make(map[string]DimensionTrend)
	current := history[0].Dimensions

	// Find 7-day ago dimensions
	var prev7DDims map[string]float64
	for _, entry := range history {
		t, _ := time.Parse(time.RFC3339, entry.RecordedAt)
		if time.Since(t) >= 7*24*time.Hour {
			prev7DDims = entry.Dimensions
			break
		}
	}

	if prev7DDims == nil {
		return nil
	}

	for dim, currentVal := range current {
		prevVal, ok := prev7DDims[dim]
		if !ok {
			continue
		}
		change := currentVal - prevVal
		trend := "stable"
		if change > 5 {
			trend = "improving"
		} else if change < -5 {
			trend = "deteriorating"
		}
		dimTrends[dim] = DimensionTrend{
			Current:  currentVal,
			Change7D: change,
			Trend:    trend,
		}
	}

	return dimTrends
}

// calculateComparison computes stock vs sector/market comparison.
func (h *ScoreHistoryHandler) calculateComparison(ctx context.Context, code string) *ComparisonData {
	comp := &ComparisonData{}

	// Get current score from latest history entry
	var currentScore float64
	err := h.db.QueryRow(ctx,
		`SELECT score FROM score_history WHERE code = $1 ORDER BY recorded_at DESC LIMIT 1`,
		code).Scan(&currentScore)
	if err != nil {
		return nil
	}
	comp.StockScore = currentScore

	// Get stock's sector
	var sector string
	err = h.db.QueryRow(ctx,
		`SELECT industry FROM tushare_stock_basic WHERE ts_code = $1 OR code = $1 LIMIT 1`,
		code).Scan(&sector)
	if err == nil {
		comp.SectorName = sector

		// Get sector average score (from recent score_history entries)
		var sectorAvg float64
		err = h.db.QueryRow(ctx,
			`SELECT COALESCE(AVG(sh.score), 50)
			 FROM score_history sh
			 JOIN tushare_stock_basic sb ON sh.code = sb.code
			 WHERE sb.industry = $1
			 AND sh.recorded_at > NOW() - INTERVAL '3 days'`,
			sector).Scan(&sectorAvg)
		if err == nil {
			comp.SectorAvgScore = sectorAvg
		}
	}

	// Calculate stock's 5-day and 20-day price change from klines
	var price5D, price20D, priceNow float64
	h.db.QueryRow(ctx,
		`SELECT close FROM tushare_daily WHERE ts_code = $1 OR code = $1
		 ORDER BY trade_date DESC LIMIT 1`, code).Scan(&priceNow)
	h.db.QueryRow(ctx,
		`SELECT close FROM tushare_daily WHERE ts_code = $1 OR code = $1
		 ORDER BY trade_date DESC OFFSET 5 LIMIT 1`, code).Scan(&price5D)
	h.db.QueryRow(ctx,
		`SELECT close FROM tushare_daily WHERE ts_code = $1 OR code = $1
		 ORDER BY trade_date DESC OFFSET 20 LIMIT 1`, code).Scan(&price20D)

	if priceNow > 0 && price5D > 0 {
		comp.StockChange5D = ((priceNow - price5D) / price5D) * 100
	}
	if priceNow > 0 && price20D > 0 {
		comp.StockChange20D = ((priceNow - price20D) / price20D) * 100
	}

	// Get market (CSI 300 index) changes
	var marketNow, market5D, market20D float64
	h.db.QueryRow(ctx,
		`SELECT close FROM tushare_index_daily WHERE ts_code = '000300.SH'
		 ORDER BY trade_date DESC LIMIT 1`).Scan(&marketNow)
	h.db.QueryRow(ctx,
		`SELECT close FROM tushare_index_daily WHERE ts_code = '000300.SH'
		 ORDER BY trade_date DESC OFFSET 5 LIMIT 1`).Scan(&market5D)
	h.db.QueryRow(ctx,
		`SELECT close FROM tushare_index_daily WHERE ts_code = '000300.SH'
		 ORDER BY trade_date DESC OFFSET 20 LIMIT 1`).Scan(&market20D)

	if marketNow > 0 && market5D > 0 {
		comp.MarketChange5D = ((marketNow - market5D) / market5D) * 100
	}
	if marketNow > 0 && market20D > 0 {
		comp.MarketChange20D = ((marketNow - market20D) / market20D) * 100
	}

	// Relative strength
	comp.VsSector = comp.StockChange5D - comp.SectorAvgChange5D
	comp.VsMarket = comp.StockChange5D - comp.MarketChange5D

	return comp
}

// RecordScore records a score entry for a stock.
func (h *ScoreHistoryHandler) RecordScore(code string, score float64, dimensions map[string]float64) {
	dimJSON, _ := json.Marshal(dimensions)
	_, err := h.db.Exec(nil,
		`INSERT INTO score_history (code, score, dimensions) VALUES ($1, $2, $3)`,
		code, score, dimJSON)
	if err != nil {
		logger.Warn("record score history failed",
			zap.String("code", code),
			zap.Error(err))
	}
}

// PruneOldEntries removes entries older than 90 days.
func (h *ScoreHistoryHandler) PruneOldEntries() {
	tag, err := h.db.Exec(nil,
		`DELETE FROM score_history WHERE recorded_at < NOW() - INTERVAL '90 days'`)
	if err != nil {
		logger.Warn("prune score history failed", zap.Error(err))
		return
	}
	if tag.RowsAffected() > 0 {
		logger.Info("pruned old score history entries", zap.Int64("count", tag.RowsAffected()))
	}
}
