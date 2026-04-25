package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	apperrors "alphapulse/internal/errors"
	"alphapulse/internal/logger"

	"go.uber.org/zap"
)

const alpha300APIBase = "http://100.105.100.98:5173/api/rank/latest"

// Alpha300Candidate represents a stock candidate from the Alpha300 ranking API.
type Alpha300Candidate struct {
	Code               string   `json:"code"`
	TsCode             string   `json:"ts_code"`
	Name               string   `json:"name"`
	Rank               int      `json:"rank"`
	Score              float64  `json:"score"`
	Close              float64  `json:"close"`
	ATR14              float64  `json:"atr14"`
	BuyLow             float64  `json:"buy_low"`
	BuyHigh            float64  `json:"buy_high"`
	SellLow            float64  `json:"sell_low"`
	SellHigh           float64  `json:"sell_high"`
	StopLoss           float64  `json:"stop_loss"`
	Momentum           float64  `json:"momentum"`
	Trend              float64  `json:"trend"`
	Volatility         float64  `json:"volatility"`
	Liquidity          float64  `json:"liquidity"`
	Industry           string   `json:"industry"`
	LimitUpToday       bool     `json:"limit_up_today"`
	LimitUpPrevDay     bool     `json:"limit_up_prev_day"`
	LeaderSignal       string   `json:"leader_signal"`
	HarvestRiskLevel   string   `json:"harvest_risk_level"`
	FocusRank          int      `json:"focus_rank"`
	FocusScore         float64  `json:"focus_score"`
	RecommendationTier string   `json:"recommendation_tier"`
	FocusReason        string   `json:"focus_reason"`
	HarvestRiskNote    string   `json:"harvest_risk_note"`
	InWatchlist        bool     `json:"in_watchlist"`
}

// Alpha300Response represents the raw API response from the Alpha300 ranking service.
type Alpha300Response struct {
	AsOfDate string              `json:"asOfDate"`
	RunID    string              `json:"runId"`
	Rows     []Alpha300RawRow    `json:"rows"`
}

// Alpha300RawRow represents a raw row from the Alpha300 API.
type Alpha300RawRow struct {
	Rank               int     `json:"rank"`
	TsCode             string  `json:"tsCode"`
	Name               string  `json:"name"`
	Score              float64 `json:"score"`
	Close              float64 `json:"close"`
	ATR14              float64 `json:"atr14"`
	BuyLow             float64 `json:"buyLow"`
	BuyHigh            float64 `json:"buyHigh"`
	SellLow            float64 `json:"sellLow"`
	SellHigh           float64 `json:"sellHigh"`
	StopLoss           float64 `json:"stopLoss"`
	Momentum           float64 `json:"momentum"`
	Trend              float64 `json:"trend"`
	Volatility         float64 `json:"volatility"`
	Liquidity          float64 `json:"liquidity"`
	Industry           string  `json:"industry"`
	LimitUpToday       bool    `json:"limitUpToday"`
	LimitUpPrevDay     bool    `json:"limitUpPrevDay"`
	LeaderSignal       string  `json:"leader_signal"`
	HarvestRiskLevel   string  `json:"harvestRiskLevel"`
	FocusRank          int     `json:"focusRank"`
	FocusScore         float64 `json:"focusScore"`
	RecommendationTier string  `json:"recommendationTier"`
	FocusReason        string  `json:"focusReason"`
	HarvestRiskNote    string  `json:"harvestRiskNote"`
}

// Alpha300Service fetches stock candidates from the Alpha300 ranking API.
type Alpha300Service struct {
	client *http.Client
}

// NewAlpha300Service creates a new Alpha300Service with the given timeout.
func NewAlpha300Service(timeout time.Duration) *Alpha300Service {
	return &Alpha300Service{
		client: &http.Client{Timeout: timeout},
	}
}

// normalizeCode strips exchange suffixes (.SH, .SZ) from a stock code.
func normalizeCode(tsCode string) string {
	code := strings.TrimSuffix(tsCode, ".SH")
	code = strings.TrimSuffix(code, ".SZ")
	return code
}

// FetchCandidates fetches stock candidates from the Alpha300 ranking API.
func (s *Alpha300Service) FetchCandidates(ctx context.Context, limit int) ([]Alpha300Candidate, error) {
	start := time.Now()

	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	reqURL := fmt.Sprintf("%s?%s", alpha300APIBase, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, apperrors.Internal(fmt.Errorf("create alpha300 request: %w", err))
	}
	req.Header.Set("User-Agent", "AlphaPulse/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		logger.Error("alpha300 fetch failed",
			zap.Error(err),
			zap.Duration("latency", time.Since(start)))
		return nil, apperrors.Internal(fmt.Errorf("fetch alpha300: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("alpha300 unexpected status",
			zap.Int("status", resp.StatusCode),
			zap.Duration("latency", time.Since(start)))
		return nil, apperrors.Internal(fmt.Errorf("alpha300 API returned status %d", resp.StatusCode))
	}

	var alphaResp Alpha300Response
	if err := json.NewDecoder(resp.Body).Decode(&alphaResp); err != nil {
		logger.Error("alpha300 decode failed",
			zap.Error(err),
			zap.Duration("latency", time.Since(start)))
		return nil, apperrors.Internal(fmt.Errorf("decode alpha300 response: %w", err))
	}

	candidates := make([]Alpha300Candidate, 0, len(alphaResp.Rows))
	for _, row := range alphaResp.Rows {
		candidates = append(candidates, Alpha300Candidate{
			Code:               normalizeCode(row.TsCode),
			TsCode:             row.TsCode,
			Name:               row.Name,
			Rank:               row.Rank,
			Score:              row.Score,
			Close:              row.Close,
			ATR14:              row.ATR14,
			BuyLow:             row.BuyLow,
			BuyHigh:            row.BuyHigh,
			SellLow:            row.SellLow,
			SellHigh:           row.SellHigh,
			StopLoss:           row.StopLoss,
			Momentum:           row.Momentum,
			Trend:              row.Trend,
			Volatility:         row.Volatility,
			Liquidity:          row.Liquidity,
			Industry:           row.Industry,
			LimitUpToday:       row.LimitUpToday,
			LimitUpPrevDay:     row.LimitUpPrevDay,
			LeaderSignal:       row.LeaderSignal,
			HarvestRiskLevel:   row.HarvestRiskLevel,
			FocusRank:          row.FocusRank,
			FocusScore:         row.FocusScore,
			RecommendationTier: row.RecommendationTier,
			FocusReason:        row.FocusReason,
			HarvestRiskNote:    row.HarvestRiskNote,
		})
	}

	logger.Info("alpha300 candidates fetched",
		zap.Int("count", len(candidates)),
		zap.Int("limit", limit),
		zap.String("as_of", alphaResp.AsOfDate),
		zap.Duration("latency", time.Since(start)))

	return candidates, nil
}
