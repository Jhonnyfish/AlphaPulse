package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const defaultPlansPath = "/home/finn/.hermes/scripts/investment_plans.json"

// InvestmentPlansHandler handles investment plan CRUD endpoints.
type InvestmentPlansHandler struct {
	mu   sync.Mutex
	path string
	log  *zap.Logger
}

// NewInvestmentPlansHandler creates a new InvestmentPlansHandler.
func NewInvestmentPlansHandler(log *zap.Logger) *InvestmentPlansHandler {
	path := os.Getenv("INVESTMENT_PLANS_PATH")
	if path == "" {
		path = defaultPlansPath
	}
	return &InvestmentPlansHandler{path: path, log: log}
}

// investmentPlanData is the top-level JSON structure.
type investmentPlanData struct {
	Plans map[string]*investmentPlan `json:"plans"`
}

// investmentPlan represents a single plan.
type investmentPlan struct {
	Code              string      `json:"code"`
	Name              string      `json:"name,omitempty"`
	TargetPrice       interface{} `json:"target_price,omitempty"`
	StopLoss          interface{} `json:"stop_loss,omitempty"`
	PositionPct       interface{} `json:"position_pct,omitempty"`
	EntryStrategy     string      `json:"entry_strategy,omitempty"`
	Logic             string      `json:"logic,omitempty"`
	HoldPeriod        string      `json:"hold_period,omitempty"`
	ExpectedReturnPct interface{} `json:"expected_return_pct,omitempty"`
	CreatedAt         string      `json:"created_at,omitempty"`
	UpdatedAt         string      `json:"updated_at,omitempty"`
}

// loadPlans reads plans from the JSON file.
func (h *InvestmentPlansHandler) loadPlans() investmentPlanData {
	h.mu.Lock()
	defer h.mu.Unlock()

	data := investmentPlanData{Plans: make(map[string]*investmentPlan)}

	raw, err := os.ReadFile(h.path)
	if err != nil {
		return data
	}
	_ = json.Unmarshal(raw, &data)
	if data.Plans == nil {
		data.Plans = make(map[string]*investmentPlan)
	}
	return data
}

// savePlans writes plans to the JSON file.
func (h *InvestmentPlansHandler) savePlans(data investmentPlanData) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(h.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.path, raw, 0o644)
}

// List handles GET /api/investment-plans — return all plans.
// @Summary      获取投资计划列表
// @Description  返回所有投资计划
// @Tags         investment-plans
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/investment-plans [get]
func (h *InvestmentPlansHandler) List(c *gin.Context) {
	data := loadPlansSafe(h)
	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"plans": data.Plans,
	})
}

func loadPlansSafe(h *InvestmentPlansHandler) investmentPlanData {
	return h.loadPlans()
}

// upsertRequest is the POST body for creating/updating a plan.
type upsertRequest struct {
	Code              string      `json:"code"`
	Name              string      `json:"name,omitempty"`
	TargetPrice       interface{} `json:"target_price,omitempty"`
	StopLoss          interface{} `json:"stop_loss,omitempty"`
	PositionPct       interface{} `json:"position_pct,omitempty"`
	EntryStrategy     string      `json:"entry_strategy,omitempty"`
	Logic             string      `json:"logic,omitempty"`
	HoldPeriod        string      `json:"hold_period,omitempty"`
	ExpectedReturnPct interface{} `json:"expected_return_pct,omitempty"`
}

// Upsert handles POST /api/investment-plans — create or update a plan.
// @Summary      创建或更新投资计划
// @Description  创建新投资计划或更新已有计划
// @Tags         investment-plans
// @Accept       json
// @Produce      json
// @Param        body  body  upsertRequest  true  "投资计划数据"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/investment-plans [post]
func (h *InvestmentPlansHandler) Upsert(c *gin.Context) {
	var req upsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_BODY", "请求体不能为空")
		return
	}

	rawCode := cleanCode(req.Code)
	if rawCode == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "股票代码不能为空")
		return
	}

	code := normalizeStockCode(rawCode)
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "股票代码格式错误")
		return
	}
	// Use 6-digit code as key
	six := code[2:] // strip sh/sz/bj prefix

	data := h.loadPlans()
	now := time.Now().Format(time.RFC3339)
	_, isUpdate := data.Plans[six]

	plan := data.Plans[six]
	if plan == nil {
		plan = &investmentPlan{Code: six, CreatedAt: now}
	}

	// Update fields (keep existing if not provided)
	plan.Code = six
	if req.Name != "" {
		plan.Name = req.Name
	}
	if req.TargetPrice != nil {
		plan.TargetPrice = req.TargetPrice
	}
	if req.StopLoss != nil {
		plan.StopLoss = req.StopLoss
	}
	if req.PositionPct != nil {
		plan.PositionPct = req.PositionPct
	}
	if req.EntryStrategy != "" {
		plan.EntryStrategy = req.EntryStrategy
	}
	if req.Logic != "" {
		plan.Logic = req.Logic
	}
	if req.HoldPeriod != "" {
		plan.HoldPeriod = req.HoldPeriod
	}
	if req.ExpectedReturnPct != nil {
		plan.ExpectedReturnPct = req.ExpectedReturnPct
	}
	plan.UpdatedAt = now

	data.Plans[six] = plan

	if err := h.savePlans(data); err != nil {
		h.log.Error("failed to save investment plans", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "SAVE_ERROR", "保存投资计划失败")
		return
	}

	action := "investment_plan_create"
	if isUpdate {
		action = "investment_plan_update"
	}
	h.log.Info("investment plan upserted", zap.String("action", action), zap.String("code", six))

	c.JSON(http.StatusOK, gin.H{
		"ok":   true,
		"plan": plan,
	})
}

// Delete handles DELETE /api/investment-plans/:code — delete a plan.
// @Summary      删除投资计划
// @Description  根据股票代码删除对应的投资计划
// @Tags         investment-plans
// @Produce      json
// @Param        code  path  string  true  "股票代码"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/investment-plans/{code} [delete]
func (h *InvestmentPlansHandler) Delete(c *gin.Context) {
	rawCode := cleanCode(c.Param("code"))
	if rawCode == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "股票代码格式错误")
		return
	}

	code := normalizeStockCode(rawCode)
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "股票代码格式错误")
		return
	}
	six := code[2:]

	data := h.loadPlans()
	plan, exists := data.Plans[six]
	if !exists {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "未找到该投资计划")
		return
	}

	delete(data.Plans, six)

	if err := h.savePlans(data); err != nil {
		h.log.Error("failed to save investment plans", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "SAVE_ERROR", "删除投资计划失败")
		return
	}

	h.log.Info("investment plan deleted", zap.String("code", six), zap.String("name", plan.Name))

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"deleted": six,
	})
}
