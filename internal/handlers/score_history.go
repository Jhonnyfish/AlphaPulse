package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"alphapulse/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ScoreHistoryEntry represents a single score history record.
type ScoreHistoryEntry struct {
	Score      float64          `json:"score"`
	Dimensions map[string]float64 `json:"dimensions"`
	RecordedAt string           `json:"recorded_at"`
}

// ScoreHistoryResponse is the full response for /api/score-history/:code.
type ScoreHistoryResponse struct {
	Code    string              `json:"code"`
	Count   int                 `json:"count"`
	History []ScoreHistoryEntry `json:"history"`
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
// @Description 获取指定股票的历史评分记录（最近 30 条）
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

	c.JSON(http.StatusOK, ScoreHistoryResponse{
		Code:    code,
		Count:   len(history),
		History: history,
	})
}

// RecordScore records a score entry for a stock (best-effort, used by analyze handler).
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

// PruneOldEntries removes entries older than 90 days (call periodically).
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

// ParseDimensions parses dimension scores from an analysis result.
func ParseDimensions(analysis map[string]any) map[string]float64 {
	dims := make(map[string]float64)
	for _, key := range []string{"order_flow", "volume_price", "valuation", "volatility",
		"money_flow", "technical", "sector", "sentiment"} {
		if dimRaw, ok := analysis[key]; ok {
			if dimMap, ok := dimRaw.(map[string]any); ok {
				if scoreRaw, ok := dimMap["score"]; ok {
					switch v := scoreRaw.(type) {
					case float64:
						dims[key] = v
					case int:
						dims[key] = float64(v)
					case string:
						if f, err := strconv.ParseFloat(v, 64); err == nil {
							dims[key] = f
						}
					}
				}
			}
		}
	}
	return dims
}
