package handlers

import (
	"context"
	"net/http"
	"time"

	"alphapulse/internal/cache"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SystemHandler struct {
	db       *pgxpool.Pool
	version  string
	started  time.Time
	cacheMap map[string]cache.Sizer
}

func NewSystemHandler(db *pgxpool.Pool, version string, started time.Time, cacheMap map[string]cache.Sizer) *SystemHandler {
	return &SystemHandler{
		db:       db,
		version:  version,
		started:  started,
		cacheMap: cacheMap,
	}
}

func (h *SystemHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status := "ok"
	if err := h.db.Ping(ctx); err != nil {
		status = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"version": h.version,
		"uptime":  time.Since(h.started).Round(time.Second).String(),
	})
}

func (h *SystemHandler) Info(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := h.db.Ping(ctx); err != nil {
		dbStatus = "error"
	}

	// Database connection pool statistics
	stats := h.db.Stat()
	poolStats := gin.H{
		"status":              dbStatus,
		"total_conns":         stats.TotalConns(),
		"acquired_conns":      stats.AcquiredConns(),
		"idle_conns":          stats.IdleConns(),
		"constructing_conns":  stats.ConstructingConns(),
		"acquire_count":       stats.AcquireCount(),
		"acquire_duration_ms": stats.AcquireDuration().Milliseconds(),
	}

	// Cache statistics with hit rates
	cacheInfo := gin.H{}
	for name, value := range h.cacheMap {
		entry := gin.H{"size": value.Len()}
		if cs, ok := value.(cache.CacheStater); ok {
			s := cs.Stats()
			entry["hits"] = s.Hits
			entry["misses"] = s.Misses
			entry["hit_rate"] = s.HitRate()
		}
		cacheInfo[name] = entry
	}

	c.JSON(http.StatusOK, gin.H{
		"db":      poolStats,
		"cache":   cacheInfo,
		"version": h.version,
		"uptime":  time.Since(h.started).Round(time.Second).String(),
	})
}

// DataSourceHealth checks whether external data sources (Tencent, EastMoney) are reachable.
func (h *SystemHandler) DataSourceHealth(eastMoneyCheck, tencentCheck func(context.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		type sourceStatus struct {
			Status  string `json:"status"`
			Latency string `json:"latency,omitempty"`
			Error   string `json:"error,omitempty"`
		}

		checkSource := func(name string, checkFn func(context.Context) error) sourceStatus {
			start := time.Now()
			err := checkFn(ctx)
			latency := time.Since(start).Round(time.Millisecond)
			if err != nil {
				return sourceStatus{
					Status:  "error",
					Latency: latency.String(),
					Error:   err.Error(),
				}
			}
			return sourceStatus{
				Status:  "ok",
				Latency: latency.String(),
			}
		}

		eastMoney := checkSource("eastmoney", eastMoneyCheck)
		tencent := checkSource("tencent", tencentCheck)

		overall := "ok"
		if eastMoney.Status != "ok" || tencent.Status != "ok" {
			overall = "degraded"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    overall,
			"eastmoney": eastMoney,
			"tencent":   tencent,
		})
	}
}
