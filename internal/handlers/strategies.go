package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"alphapulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// StrategiesHandler handles strategy CRUD endpoints.
type StrategiesHandler struct {
	db *pgxpool.Pool
}

// NewStrategiesHandler creates a new StrategiesHandler.
func NewStrategiesHandler(db *pgxpool.Pool) *StrategiesHandler {
	return &StrategiesHandler{db: db}
}

type createStrategyRequest struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Scoring       map[string]interface{} `json:"scoring"`
	Filters       map[string]interface{} `json:"filters"`
	MaxCandidates *int                   `json:"max_candidates"`
}

type updateStrategyRequest struct {
	Name          *string                `json:"name"`
	Description   *string                `json:"description"`
	Scoring       map[string]interface{} `json:"scoring"`
	Filters       map[string]interface{} `json:"filters"`
	MaxCandidates *int                   `json:"max_candidates"`
}

// List handles GET /api/strategies — list all strategies.
//
// @Summary      获取策略列表
// @Description  获取所有自定义策略列表
// @Tags         strategies
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/strategies [get]
func (h *StrategiesHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := h.db.Query(ctx,
		`SELECT id, name, description, type, scoring, dimensions, filters,
		        max_candidates, is_active, created_at, updated_at
		 FROM strategies ORDER BY type DESC, created_at ASC`)
	if err != nil {
		zap.L().Error("query strategies", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "STRATEGIES_QUERY_FAILED", "failed to query strategies")
		return
	}
	defer rows.Close()

	strategies := make([]models.Strategy, 0)
	activeIDs := make([]string, 0)
	for rows.Next() {
		var s models.Strategy
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Type,
			&s.Scoring, &s.Dimensions, &s.Filters,
			&s.MaxCandidates, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			zap.L().Error("scan strategy row", zap.Error(err))
			writeError(c, http.StatusInternalServerError, "STRATEGY_SCAN_FAILED", "failed to scan strategy row")
			return
		}
		strategies = append(strategies, s)
		if s.IsActive {
			activeIDs = append(activeIDs, s.ID)
		}
	}

	c.JSON(http.StatusOK, models.StrategyListResponse{
		OK:               true,
		Strategies:       strategies,
		ActiveStrategies: activeIDs,
	})
}

// Create handles POST /api/strategies — create a new custom strategy.
//
// @Summary      创建策略
// @Description  创建新的自定义选股策略
// @Tags         strategies
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/strategies [post]
func (h *StrategiesHandler) Create(c *gin.Context) {
	var req createStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(c, http.StatusBadRequest, "INVALID_NAME", "策略名称不能为空")
		return
	}

	if len(req.Scoring) == 0 {
		writeError(c, http.StatusBadRequest, "INVALID_SCORING", "评分权重不能为空")
		return
	}

	// Normalize scoring weights to sum to 1
	totalWeight := 0.0
	for _, v := range req.Scoring {
		if f, ok := toFloat(v); ok && f > 0 {
			totalWeight += f
		}
	}
	if totalWeight <= 0 {
		writeError(c, http.StatusBadRequest, "INVALID_WEIGHTS", "权重总和必须大于0")
		return
	}

	normalizedScoring := make(map[string]float64)
	dimensions := make([]string, 0)
	for k, v := range req.Scoring {
		if f, ok := toFloat(v); ok && f > 0 {
			normalized := float64(int(f/totalWeight*10000)) / 10000
			normalizedScoring[k] = normalized
			dimensions = append(dimensions, k)
		}
	}

	scoringJSON, _ := json.Marshal(normalizedScoring)
	dimensionsJSON, _ := json.Marshal(dimensions)

	minScore := 0.0
	if req.Filters != nil {
		if ms, ok := req.Filters["min_score"]; ok {
			minScore, _ = toFloat(ms)
		}
	}
	filtersJSON, _ := json.Marshal(map[string]interface{}{"min_score": minScore})

	maxCandidates := 50
	if req.MaxCandidates != nil && *req.MaxCandidates > 0 {
		maxCandidates = *req.MaxCandidates
	}

	ctx := c.Request.Context()
	var strategy models.Strategy
	err := h.db.QueryRow(ctx,
		`INSERT INTO strategies (name, description, type, scoring, dimensions, filters, max_candidates)
		 VALUES ($1, $2, 'custom', $3, $4, $5, $6)
		 RETURNING id, name, description, type, scoring, dimensions, filters,
		           max_candidates, is_active, created_at, updated_at`,
		name, strings.TrimSpace(req.Description), scoringJSON, dimensionsJSON, filtersJSON, maxCandidates,
	).Scan(&strategy.ID, &strategy.Name, &strategy.Description, &strategy.Type,
		&strategy.Scoring, &strategy.Dimensions, &strategy.Filters,
		&strategy.MaxCandidates, &strategy.IsActive, &strategy.CreatedAt, &strategy.UpdatedAt)
	if err != nil {
		zap.L().Error("insert strategy", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "STRATEGY_CREATE_FAILED", "failed to create strategy")
		return
	}

	c.JSON(http.StatusOK, models.StrategyResponse{
		OK:       true,
		Strategy: strategy,
		Message:  "策略 '" + name + "' 创建成功",
	})
}

// Update handles PUT /api/strategies/:id — update an existing strategy.
//
// @Summary      更新策略
// @Description  根据 ID 更新策略配置
// @Tags         strategies
// @Accept       json
// @Produce      json
// @Param        id  path      string  true  "策略 ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/strategies/{id} [put]
func (h *StrategiesHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ID", "id is required")
		return
	}

	var req updateStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	ctx := c.Request.Context()

	// Check if strategy exists and is not builtin
	var stratType string
	err := h.db.QueryRow(ctx, `SELECT type FROM strategies WHERE id = $1`, id).Scan(&stratType)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "策略不存在")
		return
	}
	if stratType == "builtin" {
		writeError(c, http.StatusBadRequest, "BUILTIN_NOT_EDITABLE", "内置策略不可编辑")
		return
	}

	// Build dynamic update
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, "name = $"+strconv.Itoa(argIdx))
		args = append(args, strings.TrimSpace(*req.Name))
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, "description = $"+strconv.Itoa(argIdx))
		args = append(args, strings.TrimSpace(*req.Description))
		argIdx++
	}
	if req.Scoring != nil {
		totalWeight := 0.0
		for _, v := range req.Scoring {
			if f, ok := toFloat(v); ok && f > 0 {
				totalWeight += f
			}
		}
		if totalWeight > 0 {
			normalized := make(map[string]float64)
			dims := make([]string, 0)
			for k, v := range req.Scoring {
				if f, ok := toFloat(v); ok && f > 0 {
					normalized[k] = float64(int(f/totalWeight*10000)) / 10000
					dims = append(dims, k)
				}
			}
			scoringJSON, _ := json.Marshal(normalized)
			dimensionsJSON, _ := json.Marshal(dims)
			setClauses = append(setClauses, "scoring = $"+strconv.Itoa(argIdx))
			args = append(args, scoringJSON)
			argIdx++
			setClauses = append(setClauses, "dimensions = $"+strconv.Itoa(argIdx))
			args = append(args, dimensionsJSON)
			argIdx++
		}
	}
	if req.Filters != nil {
		minScore := 0.0
		if ms, ok := req.Filters["min_score"]; ok {
			minScore, _ = toFloat(ms)
		}
		filtersJSON, _ := json.Marshal(map[string]interface{}{"min_score": minScore})
		setClauses = append(setClauses, "filters = $"+strconv.Itoa(argIdx))
		args = append(args, filtersJSON)
		argIdx++
	}
	if req.MaxCandidates != nil && *req.MaxCandidates > 0 {
		setClauses = append(setClauses, "max_candidates = $"+strconv.Itoa(argIdx))
		args = append(args, *req.MaxCandidates)
		argIdx++
	}

	if len(setClauses) == 0 {
		writeError(c, http.StatusBadRequest, "NO_CHANGES", "no fields to update")
		return
	}

	setClauses = append(setClauses, "updated_at = now()")
	query := "UPDATE strategies SET " + strings.Join(setClauses, ", ") +
		" WHERE id = $" + strconv.Itoa(argIdx) +
		" RETURNING id, name, description, type, scoring, dimensions, filters," +
		" max_candidates, is_active, created_at, updated_at"
	args = append(args, id)

	var strategy models.Strategy
	err = h.db.QueryRow(ctx, query, args...).Scan(
		&strategy.ID, &strategy.Name, &strategy.Description, &strategy.Type,
		&strategy.Scoring, &strategy.Dimensions, &strategy.Filters,
		&strategy.MaxCandidates, &strategy.IsActive, &strategy.CreatedAt, &strategy.UpdatedAt)
	if err != nil {
		zap.L().Error("update strategy", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "STRATEGY_UPDATE_FAILED", "failed to update strategy")
		return
	}

	c.JSON(http.StatusOK, models.StrategyResponse{
		OK:       true,
		Strategy: strategy,
		Message:  "策略 '" + strategy.Name + "' 更新成功",
	})
}

// Delete handles DELETE /api/strategies/:id — delete a custom strategy.
//
// @Summary      删除策略
// @Description  根据 ID 删除策略
// @Tags         strategies
// @Produce      json
// @Param        id  path      string  true  "策略 ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/strategies/{id} [delete]
func (h *StrategiesHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ID", "id is required")
		return
	}

	ctx := c.Request.Context()

	var name, stratType string
	err := h.db.QueryRow(ctx, `SELECT name, type FROM strategies WHERE id = $1`, id).Scan(&name, &stratType)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "策略不存在")
		return
	}
	if stratType == "builtin" {
		writeError(c, http.StatusBadRequest, "BUILTIN_NOT_DELETABLE", "内置策略不可删除")
		return
	}

	_, err = h.db.Exec(ctx, `DELETE FROM strategies WHERE id = $1`, id)
	if err != nil {
		zap.L().Error("delete strategy", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "STRATEGY_DELETE_FAILED", "failed to delete strategy")
		return
	}

	c.JSON(http.StatusOK, models.StrategyActionResponse{
		OK:      true,
		Message: "策略 '" + name + "' 已删除",
	})
}

// Activate handles POST /api/strategies/:id/activate — activate a strategy.
//
// @Summary      激活策略
// @Description  激活指定策略用于选股
// @Tags         strategies
// @Produce      json
// @Param        id  path      string  true  "策略 ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/strategies/{id}/activate [post]
func (h *StrategiesHandler) Activate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ID", "id is required")
		return
	}

	ctx := c.Request.Context()

	var exists bool
	err := h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM strategies WHERE id = $1)`, id).Scan(&exists)
	if err != nil || !exists {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "策略不存在")
		return
	}

	_, err = h.db.Exec(ctx, `UPDATE strategies SET is_active = true, updated_at = now() WHERE id = $1`, id)
	if err != nil {
		zap.L().Error("activate strategy", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "STRATEGY_ACTIVATE_FAILED", "failed to activate strategy")
		return
	}

	activeIDs := h.getActiveIDs(c)

	c.JSON(http.StatusOK, models.StrategyActionResponse{
		OK:               true,
		Message:          "策略已激活",
		ActiveStrategies: activeIDs,
	})
}

// Deactivate handles POST /api/strategies/:id/deactivate — deactivate a strategy.
//
// @Summary      停用策略
// @Description  停用指定策略
// @Tags         strategies
// @Produce      json
// @Param        id  path      string  true  "策略 ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/strategies/{id}/deactivate [post]
func (h *StrategiesHandler) Deactivate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ID", "id is required")
		return
	}

	ctx := c.Request.Context()

	_, err := h.db.Exec(ctx, `UPDATE strategies SET is_active = false, updated_at = now() WHERE id = $1`, id)
	if err != nil {
		zap.L().Error("deactivate strategy", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "STRATEGY_DEACTIVATE_FAILED", "failed to deactivate strategy")
		return
	}

	activeIDs := h.getActiveIDs(c)

	c.JSON(http.StatusOK, models.StrategyActionResponse{
		OK:               true,
		Message:          "策略已停用",
		ActiveStrategies: activeIDs,
	})
}

// getActiveIDs returns a list of active strategy IDs.
func (h *StrategiesHandler) getActiveIDs(ctx *gin.Context) []string {
	rows, err := h.db.Query(ctx.Request.Context(),
		`SELECT id FROM strategies WHERE is_active = true ORDER BY name`)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// toFloat attempts to convert an interface{} to float64.
func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
