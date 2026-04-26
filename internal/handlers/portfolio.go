package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	apperrors "alphapulse/internal/errors"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PortfolioHandler struct {
	tencent   *services.TencentService
	eastMoney *services.EastMoneyService
	db        *pgxpool.Pool
	logger    *zap.Logger
}

func NewPortfolioHandler(tencent *services.TencentService, eastMoney *services.EastMoneyService, db *pgxpool.Pool, logger *zap.Logger) *PortfolioHandler {
	return &PortfolioHandler{
		tencent:   tencent,
		eastMoney: eastMoney,
		db:        db,
		logger:    logger,
	}
}

// List handles GET /api/portfolio — list all positions enriched with live quotes.
//
// @Summary      获取投资组合列表
// @Description  获取所有持仓列表，附带实时行情数据
// @Tags         portfolio
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/portfolio [get]
func (h *PortfolioHandler) List(c *gin.Context) {
	h.logger.Info("portfolio list requested")
	ctx := c.Request.Context()
	positions, err := h.loadPositions(ctx)
	if err != nil {
		h.logger.Error("failed to load portfolio positions", zap.Error(err))
		writeAppError(c, apperrors.Internal(fmt.Errorf("failed to load portfolio")))
		return
	}

	enriched := make([]models.PortfolioPositionEnriched, 0, len(positions))
	for _, pos := range positions {
		ep := models.PortfolioPositionEnriched{
			PortfolioPosition: pos,
		}
		if quote, err := h.tencent.FetchQuote(ctx, pos.Code); err == nil {
			ep.CurrentPrice = quote.Price
		}
		ep.MarketValue = roundPort(ep.CurrentPrice * float64(pos.Quantity))
		ep.TotalCost = roundPort(pos.CostPrice * float64(pos.Quantity))
		ep.PnL = roundPort(ep.MarketValue - ep.TotalCost)
		if pos.CostPrice > 0 {
			ep.PnLPct = roundPort((ep.CurrentPrice - pos.CostPrice) / pos.CostPrice * 100)
		}
		enriched = append(enriched, ep)
	}

	h.logger.Info("portfolio listed", zap.Int("count", len(enriched)))
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": enriched})
}

type addPositionRequest struct {
	Code      string  `json:"code"`
	CostPrice float64 `json:"cost_price"`
	Quantity  int     `json:"quantity"`
	Notes     string  `json:"notes"`
}

// Add handles POST /api/portfolio — add a new position.
//
// @Summary      添加持仓
// @Description  添加新的股票持仓
// @Tags         portfolio
// @Accept       json
// @Produce      json
// @Param        body  body      addPositionRequest  true  "持仓信息"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/portfolio [post]
func (h *PortfolioHandler) Add(c *gin.Context) {
	h.logger.Info("portfolio add requested")
	var req addPositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAppError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	code := strings.TrimSpace(req.Code)
	if len(code) < 6 {
		writeAppError(c, apperrors.BadRequest("股票代码格式错误"))
		return
	}
	if req.CostPrice <= 0 {
		writeAppError(c, apperrors.BadRequest("成本价必须大于 0"))
		return
	}
	if req.Quantity <= 0 {
		writeAppError(c, apperrors.BadRequest("持仓量必须大于 0"))
		return
	}

	ctx := c.Request.Context()

	// Resolve stock name
	name := ""
	if quote, err := h.tencent.FetchQuote(ctx, code); err == nil {
		name = quote.Name
	}

	now := time.Now()
	id := uuid.New().String()

	_, err := h.db.Exec(ctx,
		`INSERT INTO portfolio (id, code, name, cost_price, quantity, notes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, code, name, req.CostPrice, req.Quantity, req.Notes, now, now,
	)
	if err != nil {
		h.logger.Error("failed to insert portfolio position", zap.Error(err))
		writeAppError(c, apperrors.Internal(fmt.Errorf("failed to add position")))
		return
	}

	position := models.PortfolioPosition{
		ID:        id,
		Code:      code,
		Name:      name,
		CostPrice: req.CostPrice,
		Quantity:  req.Quantity,
		Notes:     req.Notes,
		CreatedAt: now,
		UpdatedAt: now,
	}

	h.logger.Info("portfolio position added", zap.String("code", code))
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": position})
}

type updatePositionRequest struct {
	Code      *string  `json:"code"`
	CostPrice *float64 `json:"cost_price"`
	Quantity  *int     `json:"quantity"`
	Notes     *string  `json:"notes"`
}

// Update handles PUT /api/portfolio/:id — update an existing position.
//
// @Summary      更新持仓
// @Description  根据 ID 更新持仓信息
// @Tags         portfolio
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "持仓 ID"
// @Param        body  body      updatePositionRequest true  "更新信息"
// @Success      200   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Router       /api/portfolio/{id} [put]
func (h *PortfolioHandler) Update(c *gin.Context) {
	h.logger.Info("portfolio update requested")
	id := c.Param("id")
	if id == "" {
		writeAppError(c, apperrors.BadRequest("id is required"))
		return
	}

	var req updatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAppError(c, apperrors.BadRequest("invalid request body"))
		return
	}

	ctx := c.Request.Context()

	// Fetch existing position
	var pos models.PortfolioPosition
	err := h.db.QueryRow(ctx,
		`SELECT id, code, name, cost_price, quantity, notes, created_at, updated_at
		 FROM portfolio WHERE id = $1`, id,
	).Scan(&pos.ID, &pos.Code, &pos.Name, &pos.CostPrice, &pos.Quantity, &pos.Notes, &pos.CreatedAt, &pos.UpdatedAt)
	if err != nil {
		writeAppError(c, apperrors.NotFound("持仓"))
		return
	}

	// Apply updates
	if req.CostPrice != nil {
		if *req.CostPrice <= 0 {
			writeAppError(c, apperrors.BadRequest("成本价必须大于 0"))
			return
		}
		pos.CostPrice = *req.CostPrice
	}
	if req.Quantity != nil {
		if *req.Quantity <= 0 {
			writeAppError(c, apperrors.BadRequest("持仓量必须大于 0"))
			return
		}
		pos.Quantity = *req.Quantity
	}
	if req.Code != nil {
		code := strings.TrimSpace(*req.Code)
		if len(code) >= 6 {
			pos.Code = code
			if quote, err := h.tencent.FetchQuote(ctx, code); err == nil {
				pos.Name = quote.Name
			}
		}
	}
	if req.Notes != nil {
		pos.Notes = *req.Notes
	}
	pos.UpdatedAt = time.Now()

	_, err = h.db.Exec(ctx,
		`UPDATE portfolio SET code=$1, name=$2, cost_price=$3, quantity=$4, notes=$5, updated_at=$6
		 WHERE id=$7`,
		pos.Code, pos.Name, pos.CostPrice, pos.Quantity, pos.Notes, pos.UpdatedAt, id,
	)
	if err != nil {
		h.logger.Error("failed to update portfolio position", zap.Error(err))
		writeAppError(c, apperrors.Internal(fmt.Errorf("failed to update position")))
		return
	}

	h.logger.Info("portfolio position updated", zap.String("id", id))
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": pos})
}

// Delete handles DELETE /api/portfolio/:id — delete a position.
//
// @Summary      删除持仓
// @Description  根据 ID 删除持仓
// @Tags         portfolio
// @Produce      json
// @Param        id  path      string  true  "持仓 ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/portfolio/{id} [delete]
func (h *PortfolioHandler) Delete(c *gin.Context) {
	h.logger.Info("portfolio delete requested")
	id := c.Param("id")
	if id == "" {
		writeAppError(c, apperrors.BadRequest("id is required"))
		return
	}

	tag, err := h.db.Exec(c.Request.Context(),
		`DELETE FROM portfolio WHERE id = $1`, id,
	)
	if err != nil {
		h.logger.Error("failed to delete portfolio position", zap.Error(err))
		writeAppError(c, apperrors.Internal(fmt.Errorf("failed to delete position")))
		return
	}
	if tag.RowsAffected() == 0 {
		writeAppError(c, apperrors.NotFound("持仓"))
		return
	}

	h.logger.Info("portfolio position deleted", zap.String("id", id))
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "已删除"})
}

// Analytics handles GET /api/portfolio/analytics — portfolio analytics.
//
// @Summary      投资组合分析
// @Description  获取投资组合的收益率曲线、盈亏统计、贡献度等分析数据
// @Tags         portfolio
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/portfolio/analytics [get]
func (h *PortfolioHandler) Analytics(c *gin.Context) {
	h.logger.Info("portfolio analytics requested")
	ctx := c.Request.Context()
	positions, err := h.loadPositions(ctx)
	if err != nil {
		h.logger.Error("failed to load portfolio for analytics", zap.Error(err))
		writeAppError(c, apperrors.Internal(fmt.Errorf("failed to load portfolio")))
		return
	}

	empty := models.PortfolioAnalytics{
		ReturnCurve:   []models.ReturnCurvePoint{},
		WinLoss:       models.WinLossStats{},
		MaxDrawdown:   models.MaxDrawdownStats{},
		Contributions: []models.ContributionItem{},
	}

	if len(positions) == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": empty})
		return
	}

	// Compute per-position PnL
	type posStat struct {
		code      string
		name      string
		pnl       float64
		pnlPct    float64
		totalCost float64
		marketVal float64
	}

	stats := make([]posStat, 0, len(positions))
	for _, pos := range positions {
		currentPrice := 0.0
		if quote, err := h.tencent.FetchQuote(ctx, pos.Code); err == nil {
			currentPrice = quote.Price
		}
		totalCost := pos.CostPrice * float64(pos.Quantity)
		marketVal := currentPrice * float64(pos.Quantity)
		pnl := marketVal - totalCost
		pnlPct := 0.0
		if pos.CostPrice > 0 {
			pnlPct = (currentPrice - pos.CostPrice) / pos.CostPrice * 100
		}
		stats = append(stats, posStat{
			code: pos.Code, name: pos.Name,
			pnl: roundPort(pnl), pnlPct: roundPort(pnlPct),
			totalCost: roundPort(totalCost), marketVal: roundPort(marketVal),
		})
	}

	// Win/Loss
	winners, losers := 0, 0
	gainSum, lossSum := 0.0, 0.0
	for _, s := range stats {
		if s.pnl > 0 {
			winners++
			gainSum += s.pnlPct
		} else if s.pnl < 0 {
			losers++
			lossSum += s.pnlPct
		}
	}
	total := len(stats)
	winRate := 0.0
	if total > 0 {
		winRate = roundPort(float64(winners) / float64(total) * 100)
	}
	avgGain := 0.0
	if winners > 0 {
		avgGain = roundPort(gainSum / float64(winners))
	}
	avgLoss := 0.0
	if losers > 0 {
		avgLoss = roundPort(lossSum / float64(losers))
	}

	// Contributions
	totalPnl := 0.0
	for _, s := range stats {
		totalPnl += s.pnl
	}
	contributions := make([]models.ContributionItem, 0, len(stats))
	for _, s := range stats {
		contribPct := 0.0
		if totalPnl != 0 {
			contribPct = roundPort(s.pnl / totalPnl * 100)
		}
		contributions = append(contributions, models.ContributionItem{
			Code: s.code, Name: s.name,
			PnL: s.pnl, PnLPct: s.pnlPct, ContribPct: contribPct,
		})
	}
	sort.Slice(contributions, func(i, j int) bool {
		return math.Abs(contributions[i].PnL) > math.Abs(contributions[j].PnL)
	})

	// Return curve from kline data
	returnCurve := []models.ReturnCurvePoint{}
	maxDrawdown := models.MaxDrawdownStats{}

	klineMap := make(map[string][]models.KlinePoint)
	for _, pos := range positions {
		if _, exists := klineMap[pos.Code]; exists {
			continue
		}
		if klines, err := h.eastMoney.FetchKline(ctx, pos.Code, 60); err == nil && len(klines) > 0 {
			klineMap[pos.Code] = klines
		}
	}

	if len(klineMap) > 0 {
		// Find common dates
		dateSet := make(map[string]struct{})
		for _, klines := range klineMap {
			for _, k := range klines {
				dateSet[k.Date] = struct{}{}
			}
		}
		dates := make([]string, 0, len(dateSet))
		for d := range dateSet {
			dates = append(dates, d)
		}
		sort.Strings(dates)

		// Build code->date->close lookup
		codeDateClose := make(map[string]map[string]float64)
		for code, klines := range klineMap {
			m := make(map[string]float64)
			for _, k := range klines {
				m[k.Date] = k.Close
			}
			codeDateClose[code] = m
		}

		// Compute portfolio value per date
		type dateVal struct {
			date  string
			value float64
		}
		var dateValues []dateVal
		for _, date := range dates {
			totalValue := 0.0
			hasAll := true
			for _, pos := range positions {
				close, ok := codeDateClose[pos.Code][date]
				if !ok {
					hasAll = false
					break
				}
				totalValue += close * float64(pos.Quantity)
			}
			if hasAll && totalValue > 0 {
				dateValues = append(dateValues, dateVal{date: date, value: totalValue})
			}
		}

		if len(dateValues) > 0 {
			initialValue := dateValues[0].value
			returnCurve = make([]models.ReturnCurvePoint, 0, len(dateValues))
			for _, dv := range dateValues {
				retPct := 0.0
				if initialValue > 0 {
					retPct = roundPort4((dv.value - initialValue) / initialValue * 100)
				}
				returnCurve = append(returnCurve, models.ReturnCurvePoint{
					Date: dv.date, Value: roundPort(dv.value), ReturnPct: retPct,
				})
			}

			// Max drawdown
			peakValue := dateValues[0].value
			peakDate := dateValues[0].date
			maxDD := 0.0
			ddPeakDate, ddTroughDate := "", ""
			for _, dv := range dateValues {
				if dv.value > peakValue {
					peakValue = dv.value
					peakDate = dv.date
				}
				dd := (peakValue - dv.value) / peakValue * 100
				if dd > maxDD {
					maxDD = dd
					ddPeakDate = peakDate
					ddTroughDate = dv.date
				}
			}
			maxDrawdown = models.MaxDrawdownStats{
				Value: roundPort(maxDD), PeakDate: ddPeakDate, TroughDate: ddTroughDate,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": models.PortfolioAnalytics{
			ReturnCurve:   returnCurve,
			WinLoss:       models.WinLossStats{WinCount: winners, LossCount: losers, WinRate: winRate, AvgGainPct: avgGain, AvgLossPct: avgLoss},
			MaxDrawdown:   maxDrawdown,
			Contributions: contributions,
		},
	})
}

// Risk handles GET /api/portfolio/risk — portfolio risk analysis.
//
// @Summary      投资组合风险分析
// @Description  获取持仓风险评估、行业分布及建议
// @Tags         portfolio
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/portfolio/risk [get]
func (h *PortfolioHandler) Risk(c *gin.Context) {
	h.logger.Info("portfolio risk requested")
	ctx := c.Request.Context()
	positions, err := h.loadPositions(ctx)
	if err != nil {
		h.logger.Error("failed to load portfolio for risk analysis", zap.Error(err))
		writeAppError(c, apperrors.Internal(fmt.Errorf("failed to load portfolio")))
		return
	}

	if len(positions) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": models.PortfolioRisk{
				Stocks:          []models.RiskStockDetail{},
				Metrics:         nil,
				Recommendations: []string{},
				Message:         "暂无持仓数据",
			},
		})
		return
	}

	// Compute total market value for weight calculation
	totalMarketValue := 0.0
	type stockInfo struct {
		pos         models.PortfolioPosition
		currentPrice float64
		marketValue  float64
	}
	stocks := make([]stockInfo, 0, len(positions))
	for _, pos := range positions {
		currentPrice := 0.0
		if quote, err := h.tencent.FetchQuote(ctx, pos.Code); err == nil {
			currentPrice = quote.Price
		}
		mv := currentPrice * float64(pos.Quantity)
		totalMarketValue += mv
		stocks = append(stocks, stockInfo{pos: pos, currentPrice: currentPrice, marketValue: mv})
	}

	riskStocks := make([]models.RiskStockDetail, 0, len(stocks))
	sectorCount := make(map[string]int)
	for _, s := range stocks {
		weight := 0.0
		if totalMarketValue > 0 {
			weight = roundPort(s.marketValue / totalMarketValue * 100)
		}
		riskLevel := "中"
		if weight > 30 {
			riskLevel = "高"
		} else if weight < 10 {
			riskLevel = "低"
		}
		riskStocks = append(riskStocks, models.RiskStockDetail{
			Code: s.pos.Code, Name: s.pos.Name,
			Sector: "", Sectors: []string{},
			Volatility: 0, Beta: 0,
			RiskLevel: riskLevel, Weight: weight,
		})
		sectorCount["未知"]++
	}

	sectorDist := make([]models.SectorDistItem, 0)
	for sector, count := range sectorCount {
		pct := 0.0
		if len(positions) > 0 {
			pct = roundPort(float64(count) / float64(len(positions)) * 100)
		}
		sectorDist = append(sectorDist, models.SectorDistItem{Sector: sector, Count: count, Pct: pct})
	}

	recommendations := []string{}
	if len(positions) < 3 {
		recommendations = append(recommendations, "持仓过于集中，建议增加股票数量以分散风险")
	}
	if len(sectorCount) <= 1 && len(positions) > 1 {
		recommendations = append(recommendations, "行业集中度较高，建议配置不同行业的股票")
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": models.PortfolioRisk{
			Stocks: riskStocks,
			Metrics: &models.RiskMetrics{
				SectorDistribution:   sectorDist,
				AvgCorrelation:       0,
				AnnualizedVolatility: 0,
				MaxDrawdown:          0,
				Beta:                 0,
				DiversificationScore: len(sectorCount),
				RiskLevel:            "中",
			},
			Recommendations: recommendations,
		},
	})
}

// --- helpers ---

func (h *PortfolioHandler) loadPositions(ctx context.Context) ([]models.PortfolioPosition, error) {
	rows, err := h.db.Query(ctx,
		`SELECT id, code, name, cost_price, quantity, notes, created_at, updated_at
		 FROM portfolio ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []models.PortfolioPosition
	for rows.Next() {
		var p models.PortfolioPosition
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.CostPrice, &p.Quantity, &p.Notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		positions = append(positions, p)
	}
	return positions, nil
}

func roundPort(v float64) float64 {
	return math.Round(v*100) / 100
}

func roundPort4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
