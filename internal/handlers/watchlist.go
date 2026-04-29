package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type WatchlistHandler struct {
	db           *pgxpool.Pool
	logger       *zap.Logger
	alpha300Svc  *services.Alpha300Cache // optional, set via SetAlpha300
}

// SetAlpha300 injects the Alpha300 cache for watchlist sync.
func (h *WatchlistHandler) SetAlpha300(svc *services.Alpha300Cache) {
	h.alpha300Svc = svc
}

type addWatchlistRequest struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	GroupName string `json:"group_name"`
}

type batchAddWatchlistRequest struct {
	Codes []string `json:"codes"`
}

func NewWatchlistHandler(db *pgxpool.Pool, logger *zap.Logger) *WatchlistHandler {
	return &WatchlistHandler{db: db, logger: logger}
}

// SyncAlpha300TopN adds Alpha300 top N candidates into the watchlist.
// Called by the scheduler; does NOT remove existing stocks.
func (h *WatchlistHandler) SyncAlpha300TopN(ctx context.Context, n int) (int, error) {
	if h.alpha300Svc == nil {
		return 0, nil
	}
	candidates, err := h.alpha300Svc.GetTopN(ctx, n)
	if err != nil {
		return 0, err
	}
	added := 0
	for _, cand := range candidates {
		code := cleanCode(cand.Code)
		if code == "" {
			continue
		}
		tag, err := h.db.Exec(ctx,
			`INSERT INTO watchlist (code, name, group_name)
			 VALUES ($1, $2, 'alpha300')
			 ON CONFLICT (code) DO UPDATE
			 SET name = COALESCE(NULLIF(EXCLUDED.name, ''), watchlist.name)`,
			code, cand.Name)
		if err != nil {
			h.logger.Warn("sync alpha300: upsert failed", zap.String("code", code), zap.Error(err))
			continue
		}
		added += int(tag.RowsAffected())
	}
	h.logger.Info("alpha300 auto-sync completed", zap.Int("added", added), zap.Int("candidates", len(candidates)))
	return added, nil
}

func (h *WatchlistHandler) List(c *gin.Context) {
	h.logger.Info("watchlist list requested")
	rows, err := h.db.Query(
		c.Request.Context(),
		`SELECT id, code, COALESCE(name, ''), group_name, added_at
		 FROM watchlist
		 ORDER BY added_at DESC`,
	)
	if err != nil {
		h.logger.Error("failed to query watchlist", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "WATCHLIST_LIST_FAILED", "failed to load watchlist")
		return
	}
	defer rows.Close()

	items := make([]models.WatchlistItem, 0)
	for rows.Next() {
		var item models.WatchlistItem
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.GroupName, &item.AddedAt); err != nil {
			h.logger.Error("failed to scan watchlist item", zap.Error(err))
			writeError(c, http.StatusInternalServerError, "WATCHLIST_SCAN_FAILED", "failed to scan watchlist item")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		h.logger.Error("watchlist rows iteration error", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "WATCHLIST_LIST_FAILED", "failed to load watchlist")
		return
	}

	h.logger.Info("watchlist listed", zap.Int("count", len(items)))
	c.JSON(http.StatusOK, items)
}

func (h *WatchlistHandler) Add(c *gin.Context) {
	h.logger.Info("watchlist add requested")
	var req addWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	item, err := h.upsertWatchlistItem(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "code is required" {
			writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
			return
		}
		if strings.Contains(err.Error(), "must be exactly 6 digits") {
			writeError(c, http.StatusBadRequest, "INVALID_CODE_FORMAT", err.Error())
			return
		}
		writeError(c, http.StatusInternalServerError, "WATCHLIST_ADD_FAILED", "failed to save watchlist item")
		h.logger.Error("failed to add watchlist item", zap.Error(err))
		return
	}

	h.logger.Info("watchlist item added", zap.String("code", item.Code))
	c.JSON(http.StatusOK, gin.H{"stock": item})
}

func (h *WatchlistHandler) Delete(c *gin.Context) {
	h.logger.Info("watchlist delete requested")
	code := cleanCode(c.Param("code"))
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}

	result, err := h.db.Exec(c.Request.Context(), `DELETE FROM watchlist WHERE code = $1`, code)
	if err != nil {
		h.logger.Error("failed to delete watchlist item", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "WATCHLIST_DELETE_FAILED", "failed to delete watchlist item")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "WATCHLIST_NOT_FOUND", "watchlist item not found")
		return
	}

	h.logger.Info("watchlist item deleted", zap.String("code", code))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *WatchlistHandler) BatchAdd(c *gin.Context) {
	h.logger.Info("watchlist batch add requested")
	var req batchAddWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if len(req.Codes) == 0 {
		writeError(c, http.StatusBadRequest, "INVALID_CODES", "codes must not be empty")
		return
	}

	tx, err := h.db.BeginTx(c.Request.Context(), pgx.TxOptions{})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "TX_START_FAILED", "failed to start transaction")
		return
	}
	defer tx.Rollback(c.Request.Context())

	added := 0
	seen := make(map[string]struct{})
	for _, code := range req.Codes {
		cleaned := cleanCode(code)
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		if err := services.ValidateStockCode(cleaned); err != nil {
			continue // skip invalid codes silently in batch mode
		}
		seen[cleaned] = struct{}{}

		tag, err := tx.Exec(
			c.Request.Context(),
			`INSERT INTO watchlist (code, group_name)
			 VALUES ($1, 'default')
			 ON CONFLICT (code) DO NOTHING`,
			cleaned,
		)
		if err != nil {
			h.logger.Error("failed to insert watchlist item in batch", zap.Error(err))
			writeError(c, http.StatusInternalServerError, "WATCHLIST_BATCH_FAILED", "failed to add watchlist items")
			return
		}
		added += int(tag.RowsAffected())
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "TX_COMMIT_FAILED", "failed to save watchlist items")
		return
	}

	h.logger.Info("batch add completed", zap.Int("added", added))
	c.JSON(http.StatusOK, gin.H{"added": added})
}

// Sync handles POST /api/watchlist/sync — sync Alpha300 candidates into the watchlist.
//
//	@Summary		Sync Alpha300 watchlist
//	@Description	Fetches Alpha300 ranking candidates and upserts them into the watchlist
//	@Tags			watchlist
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		object{limit=int}	false	"Sync options"
//	@Success		200		{object}	map[string]interface{}
//	@Router			/api/watchlist/sync [post]
func (h *WatchlistHandler) Sync(c *gin.Context) {
	h.logger.Info("watchlist sync requested")
	if h.alpha300Svc == nil {
		writeError(c, http.StatusServiceUnavailable, "ALPHA300_NOT_CONFIGURED", "Alpha300 service not configured")
		return
	}

	var req struct {
		Limit int `json:"limit"`
	}
	// Body is optional
	_ = c.ShouldBindJSON(&req)
	limit := req.Limit
	if limit <= 0 {
		limit = 30
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	candidates, err := h.alpha300Svc.GetTopN(ctx, limit)
	if err != nil {
		h.logger.Error("failed to fetch Alpha300 candidates", zap.Error(err))
		writeError(c, http.StatusBadGateway, "ALPHA300_FETCH_FAILED", "failed to fetch Alpha300 candidates")
		return
	}

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "TX_START_FAILED", "failed to start transaction")
		return
	}
	defer tx.Rollback(ctx)

	added := 0
	for _, cand := range candidates {
		code := cleanCode(cand.Code)
		if code == "" {
			continue
		}
		tag, err := tx.Exec(ctx,
			`INSERT INTO watchlist (code, name, group_name)
			 VALUES ($1, $2, 'alpha300')
			 ON CONFLICT (code) DO UPDATE
			 SET name = COALESCE(NULLIF(EXCLUDED.name, ''), watchlist.name)`,
			code, cand.Name)
		if err != nil {
			continue
		}
		added += int(tag.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "TX_COMMIT_FAILED", "failed to commit sync")
		return
	}

	h.logger.Info("Alpha300 sync completed", zap.Int("added", added), zap.Int("candidates", len(candidates)))
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"message": fmt.Sprintf("Synced %d Alpha300 stock(s).", added),
		"data": gin.H{
			"count":      added,
			"candidates": len(candidates),
			"limit":      limit,
		},
	})
}

func (h *WatchlistHandler) upsertWatchlistItem(ctx context.Context, req addWatchlistRequest) (models.WatchlistItem, error) {
	code := cleanCode(req.Code)
	if code == "" {
		return models.WatchlistItem{}, errors.New("code is required")
	}
	if err := services.ValidateStockCode(code); err != nil {
		return models.WatchlistItem{}, err
	}

	groupName := strings.TrimSpace(req.GroupName)
	if groupName == "" {
		groupName = "default"
	}

	var item models.WatchlistItem
	err := h.db.QueryRow(
		ctx,
		`INSERT INTO watchlist (code, name, group_name)
		 VALUES ($1, NULLIF($2, ''), $3)
		 ON CONFLICT (code) DO UPDATE
		 SET name = COALESCE(NULLIF(EXCLUDED.name, ''), watchlist.name),
		     group_name = EXCLUDED.group_name
		 RETURNING id, code, COALESCE(name, ''), group_name, added_at`,
		code,
		strings.TrimSpace(req.Name),
		groupName,
	).Scan(&item.ID, &item.Code, &item.Name, &item.GroupName, &item.AddedAt)
	return item, err
}
