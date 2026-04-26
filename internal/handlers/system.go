package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"alphapulse/internal/cache"
	apperrors "alphapulse/internal/errors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const defaultActivityLogPath = "/home/finn/.hermes/scripts/activity_log.json"

type SystemHandler struct {
	db       *pgxpool.Pool
	version  string
	started  time.Time
	cacheMap map[string]cache.Sizer
	log      *zap.Logger
}

func NewSystemHandler(db *pgxpool.Pool, version string, started time.Time, cacheMap map[string]cache.Sizer) *SystemHandler {
	return &SystemHandler{
		db:       db,
		version:  version,
		started:  started,
		cacheMap: cacheMap,
		log:      zap.L(),
	}
}

// Health checks the service and database connectivity.
//
// @Summary      健康检查
// @Description  检查服务和数据库连接状态
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /health [get]
func (h *SystemHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status := "ok"
	if err := h.db.Ping(ctx); err != nil {
		status = "degraded"
		h.log.Warn("health check: db ping failed", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"version": h.version,
		"uptime":  time.Since(h.started).Round(time.Second).String(),
	})
}

// Info returns system information including DB pool stats and cache info.
//
// @Summary      系统信息
// @Description  返回数据库连接池和缓存信息
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/system/info [get]
func (h *SystemHandler) Info(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := h.db.Ping(ctx); err != nil {
		dbStatus = "error"
		h.log.Warn("info: db ping failed", zap.Error(err))
	}

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

// DataSourceHealth checks whether external data sources are reachable.
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
				return sourceStatus{Status: "error", Latency: latency.String(), Error: err.Error()}
			}
			return sourceStatus{Status: "ok", Latency: latency.String()}
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

// SystemStatus handles GET /api/system-status — return uptime, cache, DB stats.
//
// @Summary      系统状态
// @Description  返回运行时间、缓存和数据库统计
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/system-status [get]
func (h *SystemHandler) SystemStatus(c *gin.Context) {
	uptime := time.Since(h.started)
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	uptimeLabel := ""
	if hours > 0 {
		uptimeLabel = formatCN("%d小时%d分钟", hours, minutes)
	} else {
		uptimeLabel = formatCN("%d分钟", minutes)
	}

	cacheCount := 0
	for _, s := range h.cacheMap {
		cacheCount += s.Len()
	}

	// DB connection stats
	dbStats := h.db.Stat()
	totalConns := int(dbStats.TotalConns())
	acquiredConns := int(dbStats.AcquiredConns())

	tz, _ := time.LoadLocation("Asia/Shanghai")
	serverTime := time.Now().In(tz).Format(time.RFC3339)

	c.JSON(http.StatusOK, gin.H{
		"ok":              true,
		"uptime":          uptimeLabel,
		"uptime_seconds":  int(uptime.Seconds()),
		"cache_count":     cacheCount,
		"db_total_conns":  totalConns,
		"db_active_conns": acquiredConns,
		"server_time":     serverTime,
	})
}

// CacheClear handles POST /api/cache/clear — clear all in-memory caches.
//
// @Summary      清除缓存
// @Description  清除所有内存缓存
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/cache/clear [post]
func (h *SystemHandler) CacheClear(c *gin.Context) {
	totalCleared := 0
	for name, s := range h.cacheMap {
		if cl, ok := s.(cache.Clearer); ok {
			n := cl.Clear()
			totalCleared += n
			h.log.Info("cache cleared", zap.String("cache", name), zap.Int("entries", n))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"cleared": totalCleared,
	})
}

// activityLogEntry represents one entry in the activity log.
type activityLogEntry struct {
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	Timestamp string `json:"timestamp"`
}

// ActivityLog handles GET /api/activity-log — return last 50 entries.
//
// @Summary      活动日志
// @Description  返回最近50条活动日志
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/activity-log [get]
func (h *SystemHandler) ActivityLog(c *gin.Context) {
	logPath := os.Getenv("ACTIVITY_LOG_PATH")
	if logPath == "" {
		logPath = defaultActivityLogPath
	}

	entries := make([]activityLogEntry, 0)

	raw, err := os.ReadFile(logPath)
	if err != nil {
		// File doesn't exist — return empty list
		c.JSON(http.StatusOK, gin.H{"ok": true, "entries": entries})
		return
	}

	if err := json.Unmarshal(raw, &entries); err != nil {
		h.log.Warn("failed to parse activity log", zap.Error(err))
		writeAppError(c, apperrors.Internal(err))
		return
	}

	// Return last 50, most recent first
	if len(entries) > 50 {
		entries = entries[len(entries)-50:]
	}
	// Reverse
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"entries": entries,
	})
}

// SlowQueries handles GET /api/slow-queries — placeholder for slow query tracking.
//
// @Summary      慢查询
// @Description  返回慢查询记录
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/slow-queries [get]
func (h *SystemHandler) SlowQueries(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"items": []interface{}{},
		"total": 0,
		"stats": gin.H{
			"avg_duration_ms":  0,
			"max_duration_ms":  0,
			"slowest_endpoint": "N/A",
		},
	})
}

// PerformanceStats handles GET /api/performance-stats — placeholder.
//
// @Summary      性能统计
// @Description  返回性能统计数据
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/performance-stats [get]
func (h *SystemHandler) PerformanceStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":             true,
		"endpoints":      []interface{}{},
		"total_requests": 0,
	})
}

// Status handles GET /api/status — simple status probe.
//
// @Summary      服务状态
// @Description  简单的服务状态探测
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/status [get]
func (h *SystemHandler) Status(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	dbOK := h.db.Ping(ctx) == nil
	cacheCount := 0
	for _, s := range h.cacheMap {
		cacheCount += s.Len()
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": gin.H{
			"db":     dbOK,
			"cache":  cacheCount,
			"uptime": int(time.Since(h.started).Seconds()),
		},
	})
}

// requestCounter tracks total requests for performance stats.
var requestCounter atomic.Int64

func formatCN(format string, args ...interface{}) string {
	// Simple Chinese format helper
	buf := make([]byte, 0, 64)
	// Use fmt.Sprintf-like logic
	idx := 0
	argIdx := 0
	for idx < len(format) {
		if format[idx] == '%' && idx+1 < len(format) && format[idx+1] == 'd' {
			if argIdx < len(args) {
				n := 0
				switch v := args[argIdx].(type) {
				case int:
					n = v
				}
				buf = appendInt(buf, n)
				argIdx++
			}
			idx += 2
			continue
		}
		buf = append(buf, format[idx])
		idx++
	}

	// Map digit bytes to Chinese characters
	result := make([]byte, 0, len(buf))
	i := 0
	for i < len(buf) {
		b := buf[i]
		if b >= '0' && b <= '9' {
			// Read the full number
			numStart := i
			for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
				i++
			}
			numStr := buf[numStart:i]
			result = append(result, numStr...)
		} else {
			result = append(result, b)
			i++
		}
	}

	return string(result)
}

func appendInt(buf []byte, n int) []byte {
	if n == 0 {
		return append(buf, '0')
	}
	if n < 0 {
		buf = append(buf, '-')
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		buf = append(buf, digits[i])
	}
	return buf
}
