package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/logger"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ScreenerResult represents a single screener result item.
type ScreenerResult struct {
	Code               string  `json:"code"`
	Name               string  `json:"name"`
	Rank               int     `json:"rank"`
	Score              float64 `json:"score"`
	Close              float64 `json:"close"`
	Momentum           float64 `json:"momentum"`
	Trend              float64 `json:"trend"`
	Volatility         float64 `json:"volatility"`
	Liquidity          float64 `json:"liquidity"`
	Industry           string  `json:"industry"`
	RecommendationTier string  `json:"recommendation_tier"`
	LeaderSignal       string  `json:"leader_signal"`
	HarvestRiskLevel   string  `json:"harvest_risk_level"`
	FocusReason        string  `json:"focus_reason"`
	InWatchlist        bool    `json:"in_watchlist"`
}

// ScreenerResponse is the full response for /api/screener.
type ScreenerResponse struct {
	OK              bool             `json:"ok"`
	Count           int              `json:"count"`
	TotalCandidates int              `json:"total_candidates"`
	Filtered        int              `json:"filtered"`
	Results         []ScreenerResult `json:"results"`
	Filters         map[string]any   `json:"filters"`
}

// ScreenerHandler handles stock screener requests.
type ScreenerHandler struct {
	alpha300       *services.Alpha300Service
	db             *pgxpool.Pool
	screenerCache  *cache.Cache[ScreenerResponse]
}

// NewScreenerHandler creates a new ScreenerHandler.
func NewScreenerHandler(alpha300 *services.Alpha300Service, db *pgxpool.Pool) *ScreenerHandler {
	return &ScreenerHandler{
		alpha300:      alpha300,
		db:            db,
		screenerCache: cache.New[ScreenerResponse](),
	}
}

// CacheStats returns cache stats for monitoring.
func (h *ScreenerHandler) CacheStats() map[string]interface{} {
	return map[string]interface{}{
		"screener": h.screenerCache,
	}
}

// Screener handles GET /api/screener
// @Summary 选股器
// @Description 从 Alpha300 候选股中按多维度条件筛选，支持评分/动量/趋势/波动率/板块/等级过滤
// @Tags screener
// @Accept json
// @Produce json
// @Param min_score query number false "最低综合评分 (0-100)"
// @Param max_score query number false "最高综合评分 (0-100)"
// @Param min_momentum query number false "最低动量评分"
// @Param min_trend query number false "最低趋势评分"
// @Param max_volatility query number false "最高波动率"
// @Param sector query string false "板块关键词过滤"
// @Param tier query string false "等级过滤 (S/A/B/C)，逗号分隔"
// @Param limit query int false "结果上限 (1-100)" default(50)
// @Success 200 {object} ScreenerResponse
// @Router /api/screener [get]
func (h *ScreenerHandler) Screener(c *gin.Context) {
	start := time.Now()

	// Parse filter parameters
	var minScore, maxScore, minMomentum, minTrend, maxVolatility *float64
	if v := c.Query("min_score"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			minScore = &f
		}
	}
	if v := c.Query("max_score"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			maxScore = &f
		}
	}
	if v := c.Query("min_momentum"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			minMomentum = &f
		}
	}
	if v := c.Query("min_trend"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			minTrend = &f
		}
	}
	if v := c.Query("max_volatility"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			maxVolatility = &f
		}
	}
	sector := c.Query("sector")
	tierFilter := c.Query("tier")
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	// Build cache key from all filter params
	cacheKey := buildScreenerCacheKey(minScore, maxScore, minMomentum, minTrend, maxVolatility, sector, tierFilter, limit)

	// Check cache
	if cached, ok := h.screenerCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	// Parse tier filter into a set
	var tierSet map[string]bool
	if tierFilter != "" {
		tierSet = make(map[string]bool)
		for _, t := range strings.Split(tierFilter, ",") {
			tierSet[strings.ToUpper(strings.TrimSpace(t))] = true
		}
	}

	// Fetch candidates from Alpha300 (request more to have room for filtering)
	candidates, err := h.alpha300.FetchCandidates(c.Request.Context(), 300)
	if err != nil {
		logger.Error("screener: failed to fetch candidates", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ScreenerResponse{
			OK:    false,
			Count: 0,
			Results: []ScreenerResult{},
			Filters: buildFiltersMap(minScore, maxScore, minMomentum, minTrend, maxVolatility, sector, tierFilter, limit),
		})
		return
	}

	// Load watchlist codes for in_watchlist flag
	watchlistCodes := h.loadWatchlistCodes(c)

	// Filter and convert candidates
	var results []ScreenerResult
	for _, cand := range candidates {
		// Tier filter
		if tierSet != nil {
			tier := strings.ToUpper(cand.RecommendationTier)
			if tier == "" {
				tier = "UNKNOWN"
			}
			if !tierSet[tier] {
				continue
			}
		}

		// Sector filter (match against industry field)
		if sector != "" && !strings.Contains(cand.Industry, sector) {
			continue
		}

		// Score filter
		if minScore != nil && cand.Score < *minScore {
			continue
		}
		if maxScore != nil && cand.Score > *maxScore {
			continue
		}

		// Momentum filter
		if minMomentum != nil && cand.Momentum < *minMomentum {
			continue
		}

		// Trend filter
		if minTrend != nil && cand.Trend < *minTrend {
			continue
		}

		// Volatility filter
		if maxVolatility != nil && cand.Volatility > *maxVolatility {
			continue
		}

		results = append(results, ScreenerResult{
			Code:               cand.Code,
			Name:               cand.Name,
			Rank:               cand.Rank,
			Score:              cand.Score,
			Close:              cand.Close,
			Momentum:           cand.Momentum,
			Trend:              cand.Trend,
			Volatility:         cand.Volatility,
			Liquidity:          cand.Liquidity,
			Industry:           cand.Industry,
			RecommendationTier: cand.RecommendationTier,
			LeaderSignal:       cand.LeaderSignal,
			HarvestRiskLevel:   cand.HarvestRiskLevel,
			FocusReason:        cand.FocusReason,
			InWatchlist:        watchlistCodes[cand.Code],
		})
	}

	// Sort by score descending
	sortScreenerResults(results)

	// Apply limit
	if len(results) > limit {
		results = results[:limit]
	}

	response := ScreenerResponse{
		OK:              true,
		Count:           len(results),
		TotalCandidates: len(candidates),
		Filtered:        len(results),
		Results:         results,
		Filters:         buildFiltersMap(minScore, maxScore, minMomentum, minTrend, maxVolatility, sector, tierFilter, limit),
	}

	// Cache for 5 minutes
	h.screenerCache.Set(cacheKey, response, 5*time.Minute)

	logger.Info("screener completed",
		zap.Int("total_candidates", len(candidates)),
		zap.Int("results", len(results)),
		zap.Duration("latency", time.Since(start)))

	c.JSON(http.StatusOK, response)
}

func (h *ScreenerHandler) loadWatchlistCodes(c *gin.Context) map[string]bool {
	codes := make(map[string]bool)
	rows, err := h.db.Query(c.Request.Context(), "SELECT code FROM watchlist")
	if err != nil {
		return codes
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err == nil {
			codes[code] = true
		}
	}
	return codes
}

func buildScreenerCacheKey(minScore, maxScore, minMomentum, minTrend, maxVolatility *float64, sector, tier string, limit int) string {
	var b strings.Builder
	b.WriteString("screener:")
	if minScore != nil {
		b.WriteString("ms=" + strconv.FormatFloat(*minScore, 'f', 1, 64) + ";")
	}
	if maxScore != nil {
		b.WriteString("xs=" + strconv.FormatFloat(*maxScore, 'f', 1, 64) + ";")
	}
	if minMomentum != nil {
		b.WriteString("mm=" + strconv.FormatFloat(*minMomentum, 'f', 1, 64) + ";")
	}
	if minTrend != nil {
		b.WriteString("mt=" + strconv.FormatFloat(*minTrend, 'f', 1, 64) + ";")
	}
	if maxVolatility != nil {
		b.WriteString("xv=" + strconv.FormatFloat(*maxVolatility, 'f', 1, 64) + ";")
	}
	if sector != "" {
		b.WriteString("sec=" + sector + ";")
	}
	if tier != "" {
		b.WriteString("tier=" + tier + ";")
	}
	b.WriteString("lim=" + strconv.Itoa(limit))
	return b.String()
}

func buildFiltersMap(minScore, maxScore, minMomentum, minTrend, maxVolatility *float64, sector, tier string, limit int) map[string]any {
	m := map[string]any{
		"limit": limit,
	}
	if minScore != nil {
		m["min_score"] = *minScore
	}
	if maxScore != nil {
		m["max_score"] = *maxScore
	}
	if minMomentum != nil {
		m["min_momentum"] = *minMomentum
	}
	if minTrend != nil {
		m["min_trend"] = *minTrend
	}
	if maxVolatility != nil {
		m["max_volatility"] = *maxVolatility
	}
	if sector != "" {
		m["sector"] = sector
	}
	if tier != "" {
		m["tier"] = tier
	}
	return m
}

func sortScreenerResults(results []ScreenerResult) {
	// Simple insertion sort by score descending (good enough for <=100 items)
	for i := 1; i < len(results); i++ {
		key := results[i]
		j := i - 1
		for j >= 0 && results[j].Score < key.Score {
			results[j+1] = results[j]
			j--
		}
		results[j+1] = key
	}
}
