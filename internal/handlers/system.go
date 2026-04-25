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

	cacheInfo := make(map[string]int, len(h.cacheMap))
	for name, value := range h.cacheMap {
		cacheInfo[name] = value.Len()
	}

	c.JSON(http.StatusOK, gin.H{
		"db":      dbStatus,
		"cache":   cacheInfo,
		"version": h.version,
	})
}
