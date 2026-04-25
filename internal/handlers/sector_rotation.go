package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	alphacache "alphapulse/internal/cache"
	apperrors "alphapulse/internal/errors"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const sectorRotationHistoryPath = "/home/finn/alphapulse/data/sector_rotation_history.json"

// SectorRotationHandler handles sector rotation endpoints.
type SectorRotationHandler struct {
	eastMoney    *services.EastMoneyService
	db           *pgxpool.Pool
	logger       *zap.Logger
	rotationCache *alphacache.Cache[[]models.SectorRotationItem]
	mu           sync.Mutex // protects history file access
}

// NewSectorRotationHandler creates a new SectorRotationHandler.
func NewSectorRotationHandler(eastMoney *services.EastMoneyService, db *pgxpool.Pool, logger *zap.Logger) *SectorRotationHandler {
	return &SectorRotationHandler{
		eastMoney:    eastMoney,
		db:           db,
		logger:       logger,
		rotationCache: alphacache.New[[]models.SectorRotationItem](),
	}
}

// Rotation godoc
// @Summary      Get sector rotation analysis
// @Description  Returns sector rotation strength dashboard with breadth, net flow, and strength scores
// @Tags         sector-rotation
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.SectorRotationResponse
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/sector-rotation [get]
func (h *SectorRotationHandler) Rotation(c *gin.Context) {
	if cached, ok := h.rotationCache.Get("rotation:all"); ok {
		c.JSON(http.StatusOK, models.SectorRotationResponse{
			OK:      true,
			Sectors: cached,
			Summary: computeRotationSummary(cached),
			Cached:  true,
		})
		return
	}

	sectors, err := h.eastMoney.FetchSectorRotation(c.Request.Context())
	if err != nil {
		h.logger.Warn("failed to fetch sector rotation",
			zap.Error(err),
		)
		writeAppError(c, apperrors.Internal(err))
		return
	}

	h.rotationCache.Set("rotation:all", sectors, 5*time.Minute)

	// Save snapshot for history (best-effort)
	go h.saveSnapshot(sectors)

	c.JSON(http.StatusOK, models.SectorRotationResponse{
		OK:      true,
		Sectors: sectors,
		Summary: computeRotationSummary(sectors),
	})
}

// RotationHistory godoc
// @Summary      Get sector rotation history
// @Description  Returns stored sector rotation snapshots for historical trend analysis
// @Tags         sector-rotation
// @Accept       json
// @Produce      json
// @Param        days  query    int  false "Number of days to look back (default 5)"
// @Success      200  {object}  models.SectorRotationHistoryResponse
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/sector-rotation/history [get]
func (h *SectorRotationHandler) RotationHistory(c *gin.Context) {
	days := 5
	if rawDays := c.Query("days"); rawDays != "" {
		parsed, err := strconv.Atoi(rawDays)
		if err != nil || parsed < 1 {
			writeAppError(c, apperrors.BadRequest("days must be a positive integer"))
			return
		}
		days = parsed
	}

	h.mu.Lock()
	data, err := h.loadHistory()
	h.mu.Unlock()
	if err != nil {
		h.logger.Warn("failed to load sector rotation history", zap.Error(err))
		c.JSON(http.StatusOK, models.SectorRotationHistoryResponse{
			OK:        true,
			Snapshots: []models.SectorRotationSnapshot{},
			Total:     0,
		})
		return
	}

	// Filter by days
	cutoff := time.Now().In(chinaTZ()).AddDate(0, 0, -days).Format("2006-01-02")
	var filtered []models.SectorRotationSnapshot
	for _, s := range data {
		if s.Timestamp >= cutoff {
			filtered = append(filtered, s)
		}
	}

	// Return last 30 by default
	total := len(filtered)
	if total > 30 {
		filtered = filtered[total-30:]
	}

	c.JSON(http.StatusOK, models.SectorRotationHistoryResponse{
		OK:        true,
		Snapshots: filtered,
		Total:     total,
	})
}

type sectorRotationHistoryFile struct {
	Snapshots []models.SectorRotationSnapshot `json:"snapshots"`
}

func (h *SectorRotationHandler) loadHistory() ([]models.SectorRotationSnapshot, error) {
	if _, err := os.Stat(sectorRotationHistoryPath); os.IsNotExist(err) {
		return []models.SectorRotationSnapshot{}, nil
	}
	data, err := os.ReadFile(sectorRotationHistoryPath)
	if err != nil {
		return nil, err
	}
	var file sectorRotationHistoryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return file.Snapshots, nil
}

func (h *SectorRotationHandler) saveSnapshot(sectors []models.SectorRotationItem) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now().In(chinaTZ())
	today := now.Format("2006-01-02")

	snapshots, _ := h.loadHistory()

	// Only keep one snapshot per day — replace today's if it exists
	replaced := false
	for i, s := range snapshots {
		if s.Timestamp == today {
			snapshots[i].Sectors = sectors
			replaced = true
			break
		}
	}
	if !replaced {
		snapshots = append(snapshots, models.SectorRotationSnapshot{
			Timestamp: today,
			Sectors:   sectors,
		})
	}

	// Keep last 90 days of history
	if len(snapshots) > 90 {
		snapshots = snapshots[len(snapshots)-90:]
	}

	file := sectorRotationHistoryFile{Snapshots: snapshots}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		h.logger.Warn("failed to marshal sector rotation history", zap.Error(err))
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(sectorRotationHistoryPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.logger.Warn("failed to create history directory", zap.Error(err))
		return
	}

	if err := os.WriteFile(sectorRotationHistoryPath, data, 0o644); err != nil {
		h.logger.Warn("failed to save sector rotation history", zap.Error(err))
	}
}

func computeRotationSummary(sectors []models.SectorRotationItem) models.SectorRotationSummary {
	if len(sectors) == 0 {
		return models.SectorRotationSummary{}
	}
	var totalBreadth, totalNetFlow float64
	strongCount, weakCount := 0, 0
	for _, s := range sectors {
		totalBreadth += s.BreadthRatio
		totalNetFlow += s.NetFlow
		if s.StrengthScore >= 7 {
			strongCount++
		}
		if s.StrengthScore < 5 {
			weakCount++
		}
	}
	return models.SectorRotationSummary{
		AvgBreadth:   roundTo(totalBreadth/float64(len(sectors)), 4),
		TotalNetFlow: totalNetFlow,
		StrongCount:  strongCount,
		WeakCount:    weakCount,
	}
}

func roundTo(v float64, decimals int) float64 {
	m := 1.0
	for i := 0; i < decimals; i++ {
		m *= 10
	}
	return float64(int(v*m+0.5)) / m
}
