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
)

type WatchlistHandler struct {
	db           *pgxpool.Pool
	alpha300Svc  *services.Alpha300Service // optional, set via SetAlpha300
}

// SetAlpha300 injects the Alpha300 service for watchlist sync.
func (h *WatchlistHandler) SetAlpha300(svc *services.Alpha300Service) {
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

func NewWatchlistHandler(db *pgxpool.Pool) *WatchlistHandler {
	return &WatchlistHandler{db: db}
}

func (h *WatchlistHandler) List(c *gin.Context) {
	rows, err := h.db.Query(
		c.Request.Context(),
		`SELECT id, code, COALESCE(name, ''), group_name, added_at
		 FROM watchlist
		 ORDER BY added_at DESC`,
	)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "WATCHLIST_LIST_FAILED", "failed to load watchlist")
		return
	}
	defer rows.Close()

	items := make([]models.WatchlistItem, 0)
	for rows.Next() {
		var item models.WatchlistItem
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.GroupName, &item.AddedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "WATCHLIST_SCAN_FAILED", "failed to scan watchlist item")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "WATCHLIST_LIST_FAILED", "failed to load watchlist")
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *WatchlistHandler) Add(c *gin.Context) {
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
		return
	}

	c.JSON(http.StatusOK, gin.H{"stock": item})
}

func (h *WatchlistHandler) Delete(c *gin.Context) {
	code := cleanCode(c.Param("code"))
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}

	result, err := h.db.Exec(c.Request.Context(), `DELETE FROM watchlist WHERE code = $1`, code)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "WATCHLIST_DELETE_FAILED", "failed to delete watchlist item")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "WATCHLIST_NOT_FOUND", "watchlist item not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *WatchlistHandler) BatchAdd(c *gin.Context) {
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
			writeError(c, http.StatusInternalServerError, "WATCHLIST_BATCH_FAILED", "failed to add watchlist items")
			return
		}
		added += int(tag.RowsAffected())
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "TX_COMMIT_FAILED", "failed to save watchlist items")
		return
	}

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

	candidates, err := h.alpha300Svc.FetchCandidates(ctx, limit)
	if err != nil {
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
