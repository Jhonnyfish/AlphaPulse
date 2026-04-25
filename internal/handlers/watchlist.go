package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"alphapulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WatchlistHandler struct {
	db *pgxpool.Pool
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

func (h *WatchlistHandler) upsertWatchlistItem(ctx context.Context, req addWatchlistRequest) (models.WatchlistItem, error) {
	code := cleanCode(req.Code)
	if code == "" {
		return models.WatchlistItem{}, errors.New("code is required")
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
