package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/logger"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// CandidatesPayload is the response payload for the candidates endpoint.
type CandidatesPayload struct {
	Limit      int                          `json:"limit"`
	Items      []services.Alpha300Candidate `json:"items"`
	TierCounts map[string]int               `json:"tier_counts"`
	FetchedAt  string                       `json:"fetched_at"`
	StrategyID string                       `json:"strategy_id,omitempty"`
}

// StrategyCandidatePayload is the response when filtering by a custom strategy.
type StrategyCandidatePayload struct {
	StrategyID    string                       `json:"strategy_id"`
	StrategyName  string                       `json:"strategy_name"`
	Items         []services.Alpha300Candidate `json:"items"`
	TotalScanned  int                          `json:"total_scanned"`
	FetchedAt     string                       `json:"fetched_at"`
	MinScore      float64                      `json:"min_score"`
	MaxCandidates int                          `json:"max_candidates"`
}

// CandidatesHandler handles stock candidate requests.
type CandidatesHandler struct {
	alpha300        *services.Alpha300Service
	db              *pgxpool.Pool
	candidatesCache *cache.Cache[CandidatesPayload]
}

// NewCandidatesHandler creates a new CandidatesHandler.
func NewCandidatesHandler(alpha300 *services.Alpha300Service, db *pgxpool.Pool) *CandidatesHandler {
	return &CandidatesHandler{
		alpha300:        alpha300,
		db:              db,
		candidatesCache: cache.New[CandidatesPayload](),
	}
}

// CacheStats returns cache stats for monitoring.
func (h *CandidatesHandler) CacheStats() map[string]interface{} {
	return map[string]interface{}{
		"candidates": h.candidatesCache,
	}
}

// Candidates handles GET /api/candidates
// @Summary 获取 Alpha300 候选股列表
// @Description 从 Alpha300 排名系统获取候选股票，支持按策略过滤
// @Tags candidates
// @Accept json
// @Produce json
// @Param limit query int false "结果上限" default(50)
// @Param strategy query string false "策略名称或ID" default(builtin)
// @Success 200 {object} map[string]interface{}
// @Router /api/candidates [get]
func (h *CandidatesHandler) Candidates(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 300 {
		limit = 50
	}

	strategyID := c.Query("strategy")

	// If no strategy specified or "builtin", use standard Alpha300 flow
	if strategyID == "" || strategyID == "builtin" {
		h.candidatesBuiltin(c, limit)
		return
	}

	// Custom strategy: look up from DB and filter
	h.candidatesByStrategy(c, strategyID, limit)
}

// candidatesBuiltin returns standard Alpha300 candidates.
func (h *CandidatesHandler) candidatesBuiltin(c *gin.Context, limit int) {
	cacheKey := "candidates:" + strconv.Itoa(limit)

	// Check cache first
	if cached, ok := h.candidatesCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"data": cached,
		})
		return
	}

	// Fetch from Alpha300 API
	candidates, err := h.alpha300.FetchCandidates(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": "failed to fetch candidates",
		})
		return
	}

	// Mark in_watchlist by querying the watchlist table
	h.markInWatchlist(c, candidates)

	// Compute tier counts
	tierCounts := computeTierCounts(candidates)

	payload := CandidatesPayload{
		Limit:      limit,
		Items:      candidates,
		TierCounts: tierCounts,
		FetchedAt:  time.Now().Format(time.RFC3339),
	}

	// Cache for 5 minutes
	h.candidatesCache.Set(cacheKey, payload, 5*time.Minute)

	c.JSON(http.StatusOK, gin.H{
		"ok":   true,
		"data": payload,
	})
}

// candidatesByStrategy returns candidates filtered by a custom strategy.
func (h *CandidatesHandler) candidatesByStrategy(c *gin.Context, strategyID string, limit int) {
	ctx := c.Request.Context()

	// Look up strategy from DB
	var (
		stratName      string
		stratType      string
		scoringJSON    json.RawMessage
		filtersJSON    json.RawMessage
		maxCandidates  int
	)

	err := h.db.QueryRow(ctx,
		`SELECT name, type, scoring, filters, max_candidates
		 FROM strategies WHERE id = $1 OR name = $1`, strategyID,
	).Scan(&stratName, &stratType, &scoringJSON, &filtersJSON, &maxCandidates)
	if err != nil {
		writeError(c, http.StatusNotFound, "STRATEGY_NOT_FOUND",
			"Strategy '"+strategyID+"' not found")
		return
	}

	// If builtin strategy, use standard flow
	if stratType == "builtin" {
		h.candidatesBuiltin(c, limit)
		return
	}

	// Parse strategy filters
	var filters map[string]interface{}
	if err := json.Unmarshal(filtersJSON, &filters); err != nil {
		filters = map[string]interface{}{}
	}
	minScore := 0.0
	if ms, ok := filters["min_score"]; ok {
		if f, ok := toFloat(ms); ok {
			minScore = f
		}
	}

	// Parse strategy scoring weights
	var scoringWeights map[string]interface{}
	if err := json.Unmarshal(scoringJSON, &scoringWeights); err != nil {
		scoringWeights = map[string]interface{}{}
	}

	// Use strategy's max_candidates if provided, otherwise use request limit
	effectiveLimit := maxCandidates
	if effectiveLimit <= 0 {
		effectiveLimit = limit
	}

	// Fetch Alpha300 pool (use larger pool for filtering)
	poolSize := effectiveLimit * 3
	if poolSize < 100 {
		poolSize = 100
	}
	if poolSize > 300 {
		poolSize = 300
	}

	candidates, err := h.alpha300.FetchCandidates(ctx, poolSize)
	if err != nil {
		logger.L().Warn("strategy candidates: failed to fetch alpha300 pool",
			zap.String("strategy", strategyID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": "failed to fetch candidate pool",
		})
		return
	}

	// Mark in_watchlist
	h.markInWatchlist(c, candidates)

	// Score and filter candidates based on strategy weights
	// The Alpha300 candidates already have a score field.
	// For custom strategies, we compute a weighted score using available metrics.
	scored := make([]services.Alpha300Candidate, 0, len(candidates))
	for _, cand := range candidates {
		weightedScore := computeStrategyScore(cand, scoringWeights)
		if weightedScore >= minScore {
			// Override the score with our weighted score
			updated := cand
			updated.Score = weightedScore
			scored = append(scored, updated)
		}
	}

	// Sort by score descending
	sortCandidatesByScore(scored)

	// Trim to max
	if len(scored) > effectiveLimit {
		scored = scored[:effectiveLimit]
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": StrategyCandidatePayload{
			StrategyID:    strategyID,
			StrategyName:  stratName,
			Items:         scored,
			TotalScanned:  len(candidates),
			FetchedAt:     time.Now().Format(time.RFC3339),
			MinScore:      minScore,
			MaxCandidates: effectiveLimit,
		},
		"strategy": map[string]interface{}{
			"id":             strategyID,
			"name":           stratName,
			"type":           stratType,
			"scoring":        scoringWeights,
			"filters":        filters,
			"max_candidates": maxCandidates,
		},
	})
}

// markInWatchlist sets the InWatchlist flag for candidates in the watchlist.
func (h *CandidatesHandler) markInWatchlist(c *gin.Context, candidates []services.Alpha300Candidate) {
	watchlistCodes := make(map[string]bool)
	rows, dbErr := h.db.Query(c.Request.Context(), "SELECT code FROM watchlist")
	if dbErr == nil {
		defer rows.Close()
		for rows.Next() {
			var code string
			if err := rows.Scan(&code); err == nil {
				watchlistCodes[code] = true
			}
		}
	}

	for i := range candidates {
		candidates[i].InWatchlist = watchlistCodes[candidates[i].Code]
	}
}

// computeStrategyScore computes a weighted score for a candidate based on strategy weights.
func computeStrategyScore(cand services.Alpha300Candidate, weights map[string]interface{}) float64 {
	totalWeight := 0.0
	weightedSum := 0.0

	for dim, w := range weights {
		weight, ok := toFloat(w)
		if !ok || weight <= 0 {
			continue
		}
		totalWeight += weight

		// Map strategy dimension to candidate metrics
		dimScore := mapDimensionScore(cand, dim)
		weightedSum += dimScore * weight
	}

	if totalWeight > 0 {
		return weightedSum / totalWeight
	}
	// Fallback to Alpha300 score
	return cand.Score
}

// mapDimensionScore extracts a 0-100 score for a given dimension from the candidate.
func mapDimensionScore(cand services.Alpha300Candidate, dimension string) float64 {
	switch dimension {
	case "momentum":
		return normalizeMetric(cand.Momentum, -100, 100)
	case "trend":
		return normalizeMetric(cand.Trend, -100, 100)
	case "volatility":
		// Lower volatility is generally better for risk-adjusted returns
		v := cand.Volatility
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		return 100 - v
	case "liquidity":
		return normalizeMetric(cand.Liquidity, 0, 100)
	case "volume_price":
		// Use momentum as proxy for volume-price analysis
		return normalizeMetric(cand.Momentum, -100, 100)
	case "technical":
		// Use trend as proxy for technical analysis
		return normalizeMetric(cand.Trend, -100, 100)
	case "order_flow":
		// Use score as proxy
		return cand.Score
	case "valuation":
		// Use score as proxy (Alpha300 already factors in valuation)
		return cand.Score
	case "sentiment":
		// Use score as proxy
		return cand.Score
	case "sector":
		// Use score as proxy
		return cand.Score
	default:
		// Unknown dimension, use overall score
		return cand.Score
	}
}

// normalizeMetric maps a value from [min, max] range to [0, 100].
func normalizeMetric(value, min, max float64) float64 {
	if max <= min {
		return 50
	}
	if value < min {
		value = min
	}
	if value > max {
		value = max
	}
	return ((value - min) / (max - min)) * 100
}

// sortCandidatesByScore sorts candidates by score descending (bubble sort for small slices).
func sortCandidatesByScore(items []services.Alpha300Candidate) {
	n := len(items)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if items[j].Score < items[j+1].Score {
				items[j], items[j+1] = items[j+1], items[j]
			}
		}
	}
}

// computeTierCounts computes the tier distribution for candidates.
func computeTierCounts(candidates []services.Alpha300Candidate) map[string]int {
	tierCounts := make(map[string]int)
	for _, c := range candidates {
		tier := c.RecommendationTier
		if tier == "" {
			tier = "unknown"
		}
		tierCounts[tier]++
	}
	return tierCounts
}
