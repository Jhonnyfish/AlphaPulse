package handlers

import (
	"context"
	"net/http"
	"strings"

	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// CustomAlertsHandler handles custom alert endpoints.
type CustomAlertsHandler struct {
	db              *pgxpool.Pool
	tencentService  *services.TencentService
}

// NewCustomAlertsHandler creates a new CustomAlertsHandler.
func NewCustomAlertsHandler(db *pgxpool.Pool, tencentService *services.TencentService) *CustomAlertsHandler {
	return &CustomAlertsHandler{db: db, tencentService: tencentService}
}

type createAlertRequest struct {
	Code      string  `json:"code"`
	Type      string  `json:"type"`
	Threshold float64 `json:"threshold"`
	Enabled   *bool   `json:"enabled"`
}

// List handles GET /api/custom-alerts — return all custom alerts.
//
// @Summary      获取预警列表
// @Description  获取所有自定义预警
// @Tags         custom-alerts
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/custom-alerts [get]
func (h *CustomAlertsHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := h.db.Query(ctx,
		`SELECT id, code, name, type, threshold, enabled, triggered, created_at
		 FROM custom_alerts ORDER BY created_at DESC`)
	if err != nil {
		zap.L().Error("query custom alerts", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "ALERTS_QUERY_FAILED", "failed to query alerts")
		return
	}
	defer rows.Close()

	alerts := make([]models.CustomAlert, 0)
	for rows.Next() {
		var a models.CustomAlert
		if err := rows.Scan(&a.ID, &a.Code, &a.Name, &a.Type,
			&a.Threshold, &a.Enabled, &a.Triggered, &a.CreatedAt); err != nil {
			zap.L().Error("scan alert row", zap.Error(err))
			writeError(c, http.StatusInternalServerError, "ALERT_SCAN_FAILED", "failed to scan alert")
			return
		}
		alerts = append(alerts, a)
	}

	c.JSON(http.StatusOK, models.CustomAlertListResponse{
		OK:     true,
		Alerts: alerts,
	})
}

// Create handles POST /api/custom-alerts — create a new alert.
//
// @Summary      创建预警
// @Description  创建新的自定义预警规则
// @Tags         custom-alerts
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/custom-alerts [post]
func (h *CustomAlertsHandler) Create(c *gin.Context) {
	var req createAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "股票代码不能为空")
		return
	}

	// Normalize code
	code = normalizeStockCode(code)
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE_FORMAT", "股票代码格式错误，应为 600176、sh600176 或 600176.SH")
		return
	}

	validTypes := map[string]bool{
		"price_above": true, "price_below": true,
		"change_above": true, "change_below": true,
	}
	if !validTypes[req.Type] {
		writeError(c, http.StatusBadRequest, "INVALID_TYPE", "提醒类型无效，应为: price_above, price_below, change_above, change_below")
		return
	}

	if req.Threshold == 0 {
		writeError(c, http.StatusBadRequest, "INVALID_THRESHOLD", "阈值不能为空")
		return
	}

	// Try to get stock name
	stockName := code
	if quote, err := h.tencentService.FetchQuote(c.Request.Context(), code); err == nil && quote.Name != "" {
		stockName = quote.Name
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	ctx := c.Request.Context()
	id := uuid.New().String()

	var alert models.CustomAlert
	err := h.db.QueryRow(ctx,
		`INSERT INTO custom_alerts (id, code, name, type, threshold, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, code, name, type, threshold, enabled, triggered, created_at`,
		id, code, stockName, req.Type, req.Threshold, enabled,
	).Scan(&alert.ID, &alert.Code, &alert.Name, &alert.Type,
		&alert.Threshold, &alert.Enabled, &alert.Triggered, &alert.CreatedAt)
	if err != nil {
		zap.L().Error("insert alert", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "ALERT_CREATE_FAILED", "failed to create alert")
		return
	}

	c.JSON(http.StatusOK, models.CustomAlertResponse{
		OK:    true,
		Alert: alert,
	})
}

// Delete handles DELETE /api/custom-alerts/:id — delete an alert.
//
// @Summary      删除预警
// @Description  根据 ID 删除预警
// @Tags         custom-alerts
// @Produce      json
// @Param        id  path      string  true  "预警 ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/custom-alerts/{id} [delete]
func (h *CustomAlertsHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ID", "id is required")
		return
	}

	tag, err := h.db.Exec(c.Request.Context(),
		`DELETE FROM custom_alerts WHERE id = $1`, id)
	if err != nil {
		zap.L().Error("delete alert", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "ALERT_DELETE_FAILED", "failed to delete alert")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "提醒不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "提醒已删除"})
}

// Check handles GET /api/custom-alerts/check — check all enabled alerts.
//
// @Summary      检查预警
// @Description  检查所有启用的预警是否触发
// @Tags         custom-alerts
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/custom-alerts/check [get]
func (h *CustomAlertsHandler) Check(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := h.db.Query(ctx,
		`SELECT id, code, name, type, threshold, enabled, triggered, created_at
		 FROM custom_alerts WHERE enabled = true AND triggered = false`)
	if err != nil {
		zap.L().Error("query alerts for check", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "ALERTS_QUERY_FAILED", "failed to query alerts")
		return
	}
	defer rows.Close()

	alerts := make([]models.CustomAlert, 0)
	for rows.Next() {
		var a models.CustomAlert
		if err := rows.Scan(&a.ID, &a.Code, &a.Name, &a.Type,
			&a.Threshold, &a.Enabled, &a.Triggered, &a.CreatedAt); err != nil {
			continue
		}
		alerts = append(alerts, a)
	}

	if len(alerts) == 0 {
		c.JSON(http.StatusOK, models.CustomAlertCheckResponse{
			OK:        true,
			Triggered: []models.TriggeredAlert{},
			Checked:   0,
		})
		return
	}

	// Collect unique codes and batch fetch quotes
	codeSet := make(map[string]bool)
	for _, a := range alerts {
		codeSet[a.Code] = true
	}

	type quoteData struct {
		price     float64
		changePct float64
		name      string
	}
	quotesMap := make(map[string]quoteData)

	for code := range codeSet {
		if quote, err := h.tencentService.FetchQuote(context.Background(), code); err == nil {
			quotesMap[code] = quoteData{
				price:     quote.Price,
				changePct: quote.ChangePercent,
				name:      quote.Name,
			}
		}
	}

	// Check each alert
	triggered := make([]models.TriggeredAlert, 0)
	for _, alert := range alerts {
		quote, ok := quotesMap[alert.Code]
		if !ok {
			continue
		}

		isTriggered := false
		switch alert.Type {
		case "price_above":
			isTriggered = quote.price >= alert.Threshold
		case "price_below":
			isTriggered = quote.price <= alert.Threshold
		case "change_above":
			isTriggered = quote.changePct >= alert.Threshold
		case "change_below":
			isTriggered = quote.changePct <= -abs(alert.Threshold)
		}

		if isTriggered {
			h.db.Exec(ctx, `UPDATE custom_alerts SET triggered = true WHERE id = $1`, alert.ID)

			triggered = append(triggered, models.TriggeredAlert{
				CustomAlert:      alert,
				CurrentPrice:     quote.price,
				CurrentChangePct: quote.changePct,
				StockName:        quote.name,
			})
		}
	}

	c.JSON(http.StatusOK, models.CustomAlertCheckResponse{
		OK:        true,
		Triggered: triggered,
		Checked:   len(alerts),
	})
}

// normalizeStockCode converts various code formats to sh/sz prefix + 6 digits.
func normalizeStockCode(code string) string {
	code = strings.TrimSpace(strings.ToLower(code))

	// Already has prefix
	if strings.HasPrefix(code, "sh") || strings.HasPrefix(code, "sz") || strings.HasPrefix(code, "bj") {
		if len(code) == 8 {
			return code
		}
	}

	// Has .sh / .sz / .bj suffix
	for _, suffix := range []string{".sh", ".sz", ".bj"} {
		if strings.HasSuffix(code, suffix) {
			digits := strings.TrimSuffix(code, suffix)
			if len(digits) == 6 {
				return strings.TrimPrefix(suffix, ".") + digits
			}
		}
	}

	// Just 6 digits — infer prefix
	if len(code) == 6 {
		switch code[0] {
		case '6', '9':
			return "sh" + code
		case '0', '3':
			return "sz" + code
		case '4', '8':
			return "bj" + code
		}
	}

	return ""
}

