package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"alphapulse/internal/middleware"
	"alphapulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// StockNotesHandler handles stock note CRUD endpoints.
type StockNotesHandler struct {
	db *pgxpool.Pool
}

// NewStockNotesHandler creates a new StockNotesHandler.
func NewStockNotesHandler(db *pgxpool.Pool) *StockNotesHandler {
	return &StockNotesHandler{db: db}
}

type createNoteRequest struct {
	Code    string   `json:"code"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type updateNoteRequest struct {
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

// GetNotes handles GET /api/stock-notes/:code — return all notes for a stock.
// @Summary      获取股票备注列表
// @Description  获取指定股票代码的所有备注记录
// @Tags         stock-notes
// @Produce      json
// @Param        code  path  string  true  "股票代码"
// @Success      200  {object}  models.StockNoteListResponse
// @Router       /api/stock-notes/{code} [get]
func (h *StockNotesHandler) GetNotes(c *gin.Context) {
	code := normalizeStockCode(c.Param("code"))
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "股票代码格式错误")
		return
	}

	user, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
		return
	}

	rows, err := h.db.Query(c.Request.Context(),
		`SELECT id, code, content, tags, created_at, updated_at
		 FROM stock_notes WHERE user_id = $1 AND code = $2
		 ORDER BY created_at DESC`, user.ID, code)
	if err != nil {
		zap.L().Error("query stock notes", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "NOTES_QUERY_FAILED", "failed to query notes")
		return
	}
	defer rows.Close()

	notes := make([]models.StockNote, 0)
	for rows.Next() {
		var n models.StockNote
		var tagsJSON []byte
		if err := rows.Scan(&n.ID, &n.Code, &n.Content, &tagsJSON, &n.CreatedAt, &n.UpdatedAt); err != nil {
			zap.L().Error("scan note row", zap.Error(err))
			continue
		}
		n.Tags = parseJSONArray(tagsJSON)
		notes = append(notes, n)
	}

	c.JSON(http.StatusOK, models.StockNoteListResponse{
		OK:    true,
		Notes: notes,
	})
}

// CreateNote handles POST /api/stock-notes — create a new note.
// @Summary      创建股票备注
// @Description  为指定股票创建一条新备注
// @Tags         stock-notes
// @Accept       json
// @Produce      json
// @Param        body  body  createNoteRequest  true  "备注数据"
// @Success      200  {object}  models.StockNoteResponse
// @Router       /api/stock-notes [post]
func (h *StockNotesHandler) CreateNote(c *gin.Context) {
	var req createNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	code := strings.TrimSpace(req.Code)
	content := strings.TrimSpace(req.Content)
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "股票代码不能为空")
		return
	}
	if content == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CONTENT", "备注内容不能为空")
		return
	}

	code = normalizeStockCode(code)
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE_FORMAT", "股票代码格式错误")
		return
	}

	tags := normalizeTags(req.Tags)
	user, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
		return
	}
	id := uuid.New().String()
	now := time.Now()

	// Marshal tags to JSON bytes for JSONB column
	tagsJSON, _ := json.Marshal(tags)

	var n models.StockNote
	var tagsRaw []byte
	err := h.db.QueryRow(c.Request.Context(),
		`INSERT INTO stock_notes (id, user_id, code, content, tags, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, code, content, tags, created_at, updated_at`,
		id, user.ID, code, content, tagsJSON, now, now,
	).Scan(&n.ID, &n.Code, &n.Content, &tagsRaw, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		zap.L().Error("insert stock note", zap.Error(err),
			zap.String("user_id", user.ID), zap.String("code", code))
		writeError(c, http.StatusInternalServerError, "NOTE_CREATE_FAILED", "failed to create note: "+err.Error())
		return
	}
	n.Tags = parseJSONArray(tagsRaw)

	c.JSON(http.StatusOK, models.StockNoteResponse{OK: true, Note: n})
}

// UpdateNote handles PUT /api/stock-notes/:id — update a note.
// @Summary      更新股票备注
// @Description  更新指定ID的备注内容和标签
// @Tags         stock-notes
// @Accept       json
// @Produce      json
// @Param        id    path  string            true  "备注ID"
// @Param        body  body  updateNoteRequest  true  "更新数据"
// @Success      200  {object}  models.StockNoteResponse
// @Router       /api/stock-notes/{id} [put]
func (h *StockNotesHandler) UpdateNote(c *gin.Context) {
	noteID := c.Param("id")
	if noteID == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ID", "id is required")
		return
	}

	var req updateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CONTENT", "备注内容不能为空")
		return
	}

	tags := normalizeTags(req.Tags)
	user, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
		return
	}
	now := time.Now()

	tagsJSON, _ := json.Marshal(tags)

	var n models.StockNote
	var tagsRaw []byte
	err := h.db.QueryRow(c.Request.Context(),
		`UPDATE stock_notes SET content = $1, tags = $2, updated_at = $3
		 WHERE id = $4 AND user_id = $5
		 RETURNING id, code, content, tags, created_at, updated_at`,
		content, tagsJSON, now, noteID, user.ID,
	).Scan(&n.ID, &n.Code, &n.Content, &tagsRaw, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "备注不存在")
		return
	}
	n.Tags = parseJSONArray(tagsRaw)

	c.JSON(http.StatusOK, models.StockNoteResponse{OK: true, Note: n})
}

// DeleteNote handles DELETE /api/stock-notes/:id — delete a note.
// @Summary      删除股票备注
// @Description  删除指定ID的备注记录
// @Tags         stock-notes
// @Produce      json
// @Param        id  path  string  true  "备注ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/stock-notes/{id} [delete]
func (h *StockNotesHandler) DeleteNote(c *gin.Context) {
	noteID := c.Param("id")
	if noteID == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ID", "id is required")
		return
	}

	user, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
		return
	}

	tag, err := h.db.Exec(c.Request.Context(),
		`DELETE FROM stock_notes WHERE id = $1 AND user_id = $2`, noteID, user.ID)
	if err != nil {
		zap.L().Error("delete stock note", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "NOTE_DELETE_FAILED", "failed to delete note")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "备注不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "备注已删除"})
}

// AllTags handles GET /api/stock-notes/tags/all — return all unique tags for the user.
// @Summary      获取所有标签
// @Description  返回当前用户所有备注中使用过的唯一标签列表
// @Tags         stock-notes
// @Produce      json
// @Success      200  {object}  models.StockNoteTagsResponse
// @Router       /api/stock-notes/tags/all [get]
func (h *StockNotesHandler) AllTags(c *gin.Context) {
	userID := c.GetString("user_id")
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT DISTINCT jsonb_array_elements_text(tags) AS tag
		 FROM stock_notes WHERE user_id = $1
		 ORDER BY tag`, userID)
	if err != nil {
		zap.L().Error("query all tags", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "TAGS_QUERY_FAILED", "failed to query tags")
		return
	}
	defer rows.Close()

	tags := make([]string, 0)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err == nil {
			tags = append(tags, tag)
		}
	}

	c.JSON(http.StatusOK, models.StockNoteTagsResponse{OK: true, Tags: tags})
}

// normalizeTags cleans, deduplicates, and filters empty tags.
func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	seen := make(map[string]bool)
	var result []string
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" && !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	if len(result) == 0 {
		return []string{}
	}
	return result
}
