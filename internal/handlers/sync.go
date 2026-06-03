package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/config"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Job name constants matching those registered in main.go.
const (
	JobTushareSync  = "tushare-daily-sync"
	JobTushareRetry = "tushare-daily-retry"
)

// SyncStatusInfo holds the current state of a data sync operation.
type SyncStatusInfo struct {
	Status     string     `json:"status"` // "idle", "running", "completed", "failed"
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// SyncConfigResponse is the response for sync config endpoints.
type SyncConfigResponse struct {
	OK           bool   `json:"ok"`
	SyncEnabled  bool   `json:"sync_enabled"`
	SyncTime     string `json:"sync_time"`
	RetryEnabled bool   `json:"retry_enabled"`
	RetryTime    string `json:"retry_time"`
}

// UpdateSyncConfigRequest is the request body for updating sync config.
type UpdateSyncConfigRequest struct {
	SyncEnabled  *bool   `json:"sync_enabled"`
	SyncTime     *string `json:"sync_time"`
	RetryEnabled *bool   `json:"retry_enabled"`
	RetryTime    *string `json:"retry_time"`
}

// SyncHandler handles manual data sync and sync schedule configuration.
type SyncHandler struct {
	db              *pgxpool.Pool
	cfg             *config.Config
	log             *zap.Logger
	scheduler       *services.Scheduler
	eastMoneySvc    *services.EastMoneyService
	cacheMap        map[string]cache.Sizer
	tushareSyncFn   func(ctx context.Context) // overrideable for testing
	mu              sync.RWMutex
	status          SyncStatusInfo
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(
	db *pgxpool.Pool,
	cfg *config.Config,
	log *zap.Logger,
	scheduler *services.Scheduler,
	eastMoneySvc *services.EastMoneyService,
	cacheMap map[string]cache.Sizer,
) *SyncHandler {
	return &SyncHandler{
		db:           db,
		cfg:          cfg,
		log:          log,
		scheduler:    scheduler,
		eastMoneySvc: eastMoneySvc,
		cacheMap:     cacheMap,
		status:       SyncStatusInfo{Status: "idle"},
	}
}

// SetScheduler sets the scheduler reference (called after scheduler is created in main).
func (h *SyncHandler) SetScheduler(s *services.Scheduler) {
	h.scheduler = s
}

// ApplySchedulerConfig reads DB config and updates the scheduler accordingly.
// Called once on startup after the scheduler is created.
func (h *SyncHandler) ApplySchedulerConfig(ctx context.Context) {
	if h.scheduler == nil || h.cfg.TushareToken == "" {
		return
	}
	h.applySchedulerChanges(ctx)
}

// TriggerSync handles POST /api/system/sync — starts an async Tushare data sync.
func (h *SyncHandler) TriggerSync(c *gin.Context) {
	h.mu.RLock()
	if h.status.Status == "running" {
		h.mu.RUnlock()
		c.JSON(http.StatusConflict, gin.H{
			"ok":     false,
			"error":  "sync already running",
			"status": "running",
		})
		return
	}
	h.mu.RUnlock()

	if h.cfg.TushareToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "Tushare not configured (no token)",
		})
		return
	}

	now := time.Now()
	h.mu.Lock()
	h.status = SyncStatusInfo{
		Status:    "running",
		StartedAt: &now,
	}
	h.mu.Unlock()

	go h.runSync()

	h.log.Info("manual data sync triggered")
	c.JSON(http.StatusOK, gin.H{
		"ok":     true,
		"status": "running",
	})
}

// SyncStatus handles GET /api/system/sync/status — returns current sync state.
func (h *SyncHandler) SyncStatus(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"ok":     true,
		"status": h.status.Status,
		"started_at": func() string {
			if h.status.StartedAt != nil {
				return h.status.StartedAt.Format(time.RFC3339)
			}
			return ""
		}(),
		"finished_at": func() string {
			if h.status.FinishedAt != nil {
				return h.status.FinishedAt.Format(time.RFC3339)
			}
			return ""
		}(),
		"error": h.status.Error,
	})
}

// TriggerBackfill handles POST /api/system/sync/backfill — starts a historical
// data backfill. Accepts optional ?months=N query param (default 6, max 24).
func (h *SyncHandler) TriggerBackfill(c *gin.Context) {
	h.mu.RLock()
	if h.status.Status == "running" {
		h.mu.RUnlock()
		c.JSON(http.StatusConflict, gin.H{"ok": false, "error": "sync already running"})
		return
	}
	h.mu.RUnlock()

	if h.cfg.TushareToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Tushare not configured"})
		return
	}

	months := 6
	if m := c.Query("months"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && n > 0 && n <= 24 {
			months = n
		}
	}

	now := time.Now()
	h.mu.Lock()
	h.status = SyncStatusInfo{Status: "running", StartedAt: &now}
	h.mu.Unlock()

	go h.runBackfill(months)

	h.log.Info("historical backfill triggered", zap.Int("months", months))
	c.JSON(http.StatusOK, gin.H{"ok": true, "status": "running", "months": months})
}

func (h *SyncHandler) runBackfill(months int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	endDate := time.Now().Format("20060102")
	startDate := time.Now().AddDate(0, -months, 0).Format("20060102")

	tushareSvc := services.NewTushareService(h.cfg.TushareToken, h.cfg.HTTPTimeout)
	ts := services.NewTushareSync(tushareSvc, h.eastMoneySvc, h.db, h.log)
	if err := ts.RunBackfill(ctx, startDate, endDate); err != nil {
		h.log.Error("backfill failed", zap.Error(err))
	}

	// Clear caches
	for name, s := range h.cacheMap {
		if cl, ok := s.(cache.Clearer); ok {
			n := cl.Clear()
			h.log.Info("cache cleared after backfill", zap.String("cache", name), zap.Int("entries", n))
		}
	}

	h.mu.Lock()
	now := time.Now()
	h.status = SyncStatusInfo{Status: "completed", StartedAt: h.status.StartedAt, FinishedAt: &now}
	h.mu.Unlock()
	h.log.Info("historical backfill completed")
}

// BackfillStock handles POST /api/system/sync/backfill-stock — backfills kline
// data for a single stock. Body: {"code":"600519","months":6}. Only one API call.
func (h *SyncHandler) BackfillStock(c *gin.Context) {
	var req struct {
		Code   string `json:"code"`
		Months int    `json:"months"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "code is required"})
		return
	}
	if req.Months <= 0 {
		req.Months = 6
	}

	if h.cfg.TushareToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Tushare not configured"})
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		tushareSvc := services.NewTushareService(h.cfg.TushareToken, h.cfg.HTTPTimeout)
		ts := services.NewTushareSync(tushareSvc, h.eastMoneySvc, h.db, h.log)
		inserted, err := ts.BackfillStockKlines(ctx, req.Code, req.Months)
		if err != nil {
			h.log.Error("stock backfill failed", zap.String("code", req.Code), zap.Error(err))
		} else {
			h.log.Info("stock backfill done", zap.String("code", req.Code), zap.Int("inserted", inserted))
		}
	}()

	c.JSON(http.StatusOK, gin.H{"ok": true, "status": "running", "code": req.Code, "months": req.Months})
}

// GetSyncConfig handles GET /api/system/sync/config — returns current cron config.
func (h *SyncHandler) GetSyncConfig(c *gin.Context) {
	resp := SyncConfigResponse{
		OK:           true,
		SyncEnabled:  true,
		SyncTime:     "21:00",
		RetryEnabled: true,
		RetryTime:    "23:00",
	}

	rows, err := h.db.Query(c.Request.Context(),
		"SELECT key, value FROM system_config WHERE key LIKE 'tushare_%'")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var key, val string
			if rows.Scan(&key, &val) != nil {
				continue
			}
			switch key {
			case "tushare_sync_enabled":
				resp.SyncEnabled = val == "true"
			case "tushare_sync_time":
				resp.SyncTime = val
			case "tushare_retry_enabled":
				resp.RetryEnabled = val == "true"
			case "tushare_retry_time":
				resp.RetryTime = val
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateSyncConfig handles PUT /api/system/sync/config — updates cron schedule.
func (h *SyncHandler) UpdateSyncConfig(c *gin.Context) {
	var req UpdateSyncConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}

	// Validate time formats
	if req.SyncTime != nil && !isValidTime(*req.SyncTime) {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid sync_time format, expected HH:MM"})
		return
	}
	if req.RetryTime != nil && !isValidTime(*req.RetryTime) {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid retry_time format, expected HH:MM"})
		return
	}

	ctx := c.Request.Context()

	// Upsert each provided field
	upserts := map[string]string{}
	if req.SyncEnabled != nil {
		upserts["tushare_sync_enabled"] = strconv.FormatBool(*req.SyncEnabled)
	}
	if req.SyncTime != nil {
		upserts["tushare_sync_time"] = *req.SyncTime
	}
	if req.RetryEnabled != nil {
		upserts["tushare_retry_enabled"] = strconv.FormatBool(*req.RetryEnabled)
	}
	if req.RetryTime != nil {
		upserts["tushare_retry_time"] = *req.RetryTime
	}

	for key, val := range upserts {
		_, err := h.db.Exec(ctx,
			`INSERT INTO system_config (key, value, updated_at) VALUES ($1, $2, NOW())
			 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`,
			key, val)
		if err != nil {
			h.log.Error("failed to update config", zap.String("key", key), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "failed to save config"})
			return
		}
	}

	// Apply scheduler changes
	h.applySchedulerChanges(ctx)

	h.log.Info("sync config updated", zap.Any("changes", upserts))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// applySchedulerChanges reads current config from DB and updates running scheduler jobs.
func (h *SyncHandler) applySchedulerChanges(ctx context.Context) {
	if h.scheduler == nil {
		return
	}

	syncEnabled, syncTime := true, "21:00"
	retryEnabled, retryTime := true, "23:00"

	rows, err := h.db.Query(ctx,
		"SELECT key, value FROM system_config WHERE key LIKE 'tushare_%'")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var key, val string
			if rows.Scan(&key, &val) != nil {
				continue
			}
			switch key {
			case "tushare_sync_enabled":
				syncEnabled = val == "true"
			case "tushare_sync_time":
				syncTime = val
			case "tushare_retry_enabled":
				retryEnabled = val == "true"
			case "tushare_retry_time":
				retryTime = val
			}
		}
	}

	// Re-create the sync function that the scheduler runs
	makeSyncFn := func() func() {
		return func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			tushareSvc := services.NewTushareService(h.cfg.TushareToken, h.cfg.HTTPTimeout)
			ts := services.NewTushareSync(tushareSvc, h.eastMoneySvc, h.db, h.log)
			ts.RunDaily(syncCtx)
		}
	}

	if syncEnabled {
		syncH, syncM := parseTime(syncTime)
		h.scheduler.AddOrUpdateJob(JobTushareSync, syncH, syncM, makeSyncFn())
	} else {
		_ = h.scheduler.RemoveJob(JobTushareSync)
	}

	if retryEnabled {
		retryH, retryM := parseTime(retryTime)
		h.scheduler.AddOrUpdateJob(JobTushareRetry, retryH, retryM, makeSyncFn())
	} else {
		_ = h.scheduler.RemoveJob(JobTushareRetry)
	}
}

// runSync executes the full Tushare sync pipeline in the background.
func (h *SyncHandler) runSync() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var syncErr error
	if h.tushareSyncFn != nil {
		// Test override
		h.tushareSyncFn(ctx)
	} else {
		tushareSvc := services.NewTushareService(h.cfg.TushareToken, h.cfg.HTTPTimeout)
		ts := services.NewTushareSync(tushareSvc, h.eastMoneySvc, h.db, h.log)
		ts.RunDaily(ctx)
	}

	// Clear all caches so fresh data is served
	totalCleared := 0
	for name, s := range h.cacheMap {
		if cl, ok := s.(cache.Clearer); ok {
			n := cl.Clear()
			totalCleared += n
			h.log.Info("cache cleared after sync", zap.String("cache", name), zap.Int("entries", n))
		}
	}

	h.mu.Lock()
	now := time.Now()
	errMsg := ""
	if syncErr != nil {
		errMsg = syncErr.Error()
		h.status = SyncStatusInfo{
			Status:     "failed",
			StartedAt:  h.status.StartedAt,
			FinishedAt: &now,
			Error:      errMsg,
		}
	} else {
		h.status = SyncStatusInfo{
			Status:     "completed",
			StartedAt:  h.status.StartedAt,
			FinishedAt: &now,
		}
	}
	h.mu.Unlock()

	h.log.Info("manual sync completed",
		zap.String("result", h.status.Status),
		zap.Int("cache_cleared", totalCleared),
	)
}

// isValidTime checks if a string is a valid HH:MM time format.
func isValidTime(s string) bool {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return false
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return false
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return false
	}
	return true
}

// parseTime parses "HH:MM" into hour and minute integers.
func parseTime(s string) (int, int) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 21, 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h, m
}
