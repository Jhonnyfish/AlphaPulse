package handlers

import (
	"net/http"
	"strconv"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CandidatesPayload is the response payload for the candidates endpoint.
type CandidatesPayload struct {
	Limit     int                        `json:"limit"`
	Items     []services.Alpha300Candidate `json:"items"`
	TierCounts map[string]int            `json:"tier_counts"`
	FetchedAt string                     `json:"fetched_at"`
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
// @Param strategy query string false "策略名称" default(builtin)
// @Success 200 {object} CandidatesPayload
// @Router /api/candidates [get]
func (h *CandidatesHandler) Candidates(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 300 {
		limit = 50
	}

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

	// Update in_watchlist flag
	for i := range candidates {
		candidates[i].InWatchlist = watchlistCodes[candidates[i].Code]
	}

	// Compute tier counts
	tierCounts := make(map[string]int)
	for _, c := range candidates {
		tier := c.RecommendationTier
		if tier == "" {
			tier = "unknown"
		}
		tierCounts[tier]++
	}

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
