package handlers

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"alphapulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// TradingJournalHandler handles trading journal endpoints.
type TradingJournalHandler struct {
	db *pgxpool.Pool
}

// NewTradingJournalHandler creates a new TradingJournalHandler.
func NewTradingJournalHandler(db *pgxpool.Pool) *TradingJournalHandler {
	return &TradingJournalHandler{db: db}
}

type createTradeRequest struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Price         float64 `json:"price"`
	Quantity      int     `json:"quantity"`
	Fees          float64 `json:"fees"`
	Date          string  `json:"date"`
	Notes         string  `json:"notes"`
	StrategyLabel string  `json:"strategy_label"`
}

// List handles GET /api/trading-journal — list trades with filters.
func (h *TradingJournalHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	code := strings.TrimSpace(c.Query("code"))
	tradeType := strings.TrimSpace(c.Query("type"))
	dateFrom := strings.TrimSpace(c.Query("date_from"))
	dateTo := strings.TrimSpace(c.Query("date_to"))

	query := `SELECT id, code, name, type, price, quantity, fees, date::text, notes, strategy_label, created_at
		FROM trading_journal WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if code != "" {
		query += ` AND code = $` + strconv.Itoa(argIdx)
		args = append(args, code)
		argIdx++
	}
	if tradeType != "" && (tradeType == "buy" || tradeType == "sell") {
		query += ` AND type = $` + strconv.Itoa(argIdx)
		args = append(args, tradeType)
		argIdx++
	}
	if dateFrom != "" {
		query += ` AND date >= $` + strconv.Itoa(argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		query += ` AND date <= $` + strconv.Itoa(argIdx)
		args = append(args, dateTo)
		argIdx++
	}

	query += ` ORDER BY date ASC, created_at ASC`

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		zap.L().Error("query trading journal", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "JOURNAL_QUERY_FAILED", "failed to query trading journal")
		return
	}
	defer rows.Close()

	entries := make([]models.TradingJournalEntry, 0)
	for rows.Next() {
		var e models.TradingJournalEntry
		if err := rows.Scan(&e.ID, &e.Code, &e.Name, &e.Type, &e.Price, &e.Quantity, &e.Fees, &e.Date, &e.Notes, &e.StrategyLabel, &e.CreatedAt); err != nil {
			zap.L().Error("scan trading journal row", zap.Error(err))
			writeError(c, http.StatusInternalServerError, "JOURNAL_SCAN_FAILED", "failed to scan trading journal row")
			return
		}
		entries = append(entries, e)
	}

	summary, positions := h.computeSummaryAndPositions(ctx, entries)

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": models.TradingJournalListResponse{
			Items:     entries,
			Summary:   summary,
			Positions: positions,
		},
	})
}

// Create handles POST /api/trading-journal — create a trade record.
func (h *TradingJournalHandler) Create(c *gin.Context) {
	var req createTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}
	tradeType := strings.TrimSpace(req.Type)
	if tradeType != "buy" && tradeType != "sell" {
		writeError(c, http.StatusBadRequest, "INVALID_TYPE", "type must be buy or sell")
		return
	}
	if req.Price <= 0 {
		writeError(c, http.StatusBadRequest, "INVALID_PRICE", "price must be greater than 0")
		return
	}
	if req.Quantity <= 0 {
		writeError(c, http.StatusBadRequest, "INVALID_QUANTITY", "quantity must be greater than 0")
		return
	}

	ctx := c.Request.Context()
	id := uuid.New().String()
	now := time.Now()

	_, err := h.db.Exec(ctx,
		`INSERT INTO trading_journal (id, code, name, type, price, quantity, fees, date, notes, strategy_label, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		id, code, req.Name, tradeType, req.Price, req.Quantity, req.Fees, req.Date, req.Notes, req.StrategyLabel, now,
	)
	if err != nil {
		zap.L().Error("insert trading journal", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "JOURNAL_INSERT_FAILED", "failed to create trade record")
		return
	}

	entry := models.TradingJournalEntry{
		ID:            id,
		Code:          code,
		Name:          req.Name,
		Type:          tradeType,
		Price:         req.Price,
		Quantity:      req.Quantity,
		Fees:          req.Fees,
		Date:          req.Date,
		Notes:         req.Notes,
		StrategyLabel: req.StrategyLabel,
		CreatedAt:     now,
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "data": entry})
}

// Delete handles DELETE /api/trading-journal/:id — delete a trade record.
func (h *TradingJournalHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ID", "id is required")
		return
	}

	tag, err := h.db.Exec(c.Request.Context(),
		`DELETE FROM trading_journal WHERE id = $1`, id,
	)
	if err != nil {
		zap.L().Error("delete trading journal", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "JOURNAL_DELETE_FAILED", "failed to delete trade record")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "trade record not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "deleted"})
}

// Stats handles GET /api/trading-journal/stats — comprehensive stats.
func (h *TradingJournalHandler) Stats(c *gin.Context) {
	ctx := c.Request.Context()

	entries, err := h.loadAllEntries(ctx)
	if err != nil {
		zap.L().Error("load journal entries for stats", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "JOURNAL_LOAD_FAILED", "failed to load trading journal")
		return
	}

	sellResults := h.computeFIFOSellResults(entries)

	// Summary stats
	totalPnL := 0.0
	maxProfit := 0.0
	wins, losses := 0, 0
	for _, sr := range sellResults {
		totalPnL += sr.RealizedPnL
		if sr.RealizedPnL > maxProfit {
			maxProfit = sr.RealizedPnL
		}
		if sr.RealizedPnL > 0 {
			wins++
		} else if sr.RealizedPnL < 0 {
			losses++
		}
	}

	totalSells := len(sellResults)
	winRate := 0.0
	if totalSells > 0 {
		winRate = round2(float64(wins) / float64(totalSells) * 100)
	}

	// Monthly trades
	monthSet := make(map[string]bool)
	for _, e := range entries {
		monthSet[e.Date[:7]] = true
	}
	numMonths := len(monthSet)
	if numMonths == 0 {
		numMonths = 1
	}
	avgMonthlyTrades := round2(float64(len(entries)) / float64(numMonths))

	// Monthly PnL
	monthPnLMap := make(map[string]float64)
	for _, sr := range sellResults {
		m := sr.Date[:7]
		monthPnLMap[m] += sr.RealizedPnL
	}
	monthlyPnL := make([]models.MonthlyPnL, 0, len(monthPnLMap))
	for m, pnl := range monthPnLMap {
		monthlyPnL = append(monthlyPnL, models.MonthlyPnL{Month: m, PnL: round2(pnl)})
	}
	sort.Slice(monthlyPnL, func(i, j int) bool { return monthlyPnL[i].Month < monthlyPnL[j].Month })

	// Cumulative PnL
	cumulativePnL := make([]models.CumulativePnL, 0, len(monthlyPnL))
	cumPnL := 0.0
	for _, mp := range monthlyPnL {
		cumPnL += mp.PnL
		cumulativePnL = append(cumulativePnL, models.CumulativePnL{Month: mp.Month, CumPnL: round2(cumPnL)})
	}

	// Return distribution
	rd := models.ReturnDistribution{}
	for _, sr := range sellResults {
		pct := sr.ReturnPct
		switch {
		case pct < -10:
			rd.LessThanNeg10++
		case pct >= -10 && pct < -5:
			rd.Neg10ToNeg5++
		case pct >= -5 && pct < 0:
			rd.Neg5To0++
		case pct >= 0 && pct < 5:
			rd.ZeroTo5++
		case pct >= 5 && pct < 10:
			rd.FiveTo10++
		default:
			rd.GreaterThan10++
		}
	}

	// Activity
	byWeekday := make(map[string]int)
	byHour := make(map[string]int)
	for _, e := range entries {
		t, err := time.Parse("2006-01-02", e.Date)
		if err == nil {
			byWeekday[t.Weekday().String()]++
		}
		if e.CreatedAt.Hour() >= 0 {
			byHour[strconv.Itoa(e.CreatedAt.Hour())]++
		}
	}

	// Per stock stats
	type stockAgg struct {
		code     string
		name     string
		totalPnL float64
		trades   int
		wins     int
		losses   int
	}
	stockMap := make(map[string]*stockAgg)
	for _, sr := range sellResults {
		sa, ok := stockMap[sr.Code]
		if !ok {
			sa = &stockAgg{code: sr.Code, name: sr.Name}
			stockMap[sr.Code] = sa
		}
		sa.totalPnL += sr.RealizedPnL
		sa.trades++
		if sr.RealizedPnL > 0 {
			sa.wins++
		} else if sr.RealizedPnL < 0 {
			sa.losses++
		}
	}

	perStock := make([]models.PerStockWinLoss, 0, len(stockMap))
	for _, sa := range stockMap {
		wr := 0.0
		if sa.trades > 0 {
			wr = round2(float64(sa.wins) / float64(sa.trades) * 100)
		}
		perStock = append(perStock, models.PerStockWinLoss{
			Code: sa.code, Name: sa.name, Trades: sa.trades,
			Wins: sa.wins, Losses: sa.losses, WinRate: wr, TotalPnL: round2(sa.totalPnL),
		})
	}
	sort.Slice(perStock, func(i, j int) bool { return perStock[i].TotalPnL > perStock[j].TotalPnL })

	topPerformers := perStock
	if len(topPerformers) > 5 {
		topPerformers = topPerformers[:5]
	}
	bottomPerformers := make([]models.PerStockWinLoss, 0)
	for i := len(perStock) - 1; i >= 0 && len(bottomPerformers) < 5; i-- {
		bottomPerformers = append(bottomPerformers, perStock[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": models.TradingJournalStats{
			Summary: models.StatsSummary{
				TotalRealizedPnL: round2(totalPnL),
				WinRate:          winRate,
				AvgMonthlyTrades: avgMonthlyTrades,
				MaxSingleProfit:  round2(maxProfit),
			},
			MonthlyPnL:         monthlyPnL,
			CumulativePnL:      cumulativePnL,
			ReturnDistribution: rd,
			Activity:           models.TradingActivity{ByWeekday: byWeekday, ByHour: byHour},
			TopPerformers:      topPerformers,
			BottomPerformers:   bottomPerformers,
			WinLoss: models.WinLossOverall{
				TotalTrades: totalSells,
				Wins:        wins,
				Losses:      losses,
				WinRate:     winRate,
				PerStock:    perStock,
			},
			SellResults: sellResults,
		},
	})
}

// Calendar handles GET /api/trading-journal/calendar — daily P&L heatmap.
func (h *TradingJournalHandler) Calendar(c *gin.Context) {
	ctx := c.Request.Context()

	monthsParam := c.DefaultQuery("months", "12")
	months, err := strconv.Atoi(monthsParam)
	if err != nil || months < 1 {
		months = 12
	}

	entries, err := h.loadAllEntries(ctx)
	if err != nil {
		zap.L().Error("load journal entries for calendar", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "JOURNAL_LOAD_FAILED", "failed to load trading journal")
		return
	}

	sellResults := h.computeFIFOSellResults(entries)

	cutoff := time.Now().AddDate(0, -months, 0)
	dailyMap := make(map[string]*models.TradingCalendarDay)
	for _, sr := range sellResults {
		t, err := time.Parse("2006-01-02", sr.Date)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			continue
		}
		d, ok := dailyMap[sr.Date]
		if !ok {
			d = &models.TradingCalendarDay{Date: sr.Date}
			dailyMap[sr.Date] = d
		}
		d.PnL += sr.RealizedPnL
		d.Trades++
	}

	daily := make([]models.TradingCalendarDay, 0, len(dailyMap))
	for _, d := range dailyMap {
		d.PnL = round2(d.PnL)
		daily = append(daily, *d)
	}
	sort.Slice(daily, func(i, j int) bool { return daily[i].Date < daily[j].Date })

	// Summary
	totalPnL := 0.0
	bestDay := 0.0
	worstDay := 0.0
	profitableDays, losingDays := 0, 0
	for _, d := range daily {
		totalPnL += d.PnL
		if d.PnL > bestDay {
			bestDay = d.PnL
		}
		if d.PnL < worstDay {
			worstDay = d.PnL
		}
		if d.PnL > 0 {
			profitableDays++
		} else if d.PnL < 0 {
			losingDays++
		}
	}

	avgDailyPnL := 0.0
	if len(daily) > 0 {
		avgDailyPnL = round2(totalPnL / float64(len(daily)))
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": models.TradingJournalCalendarResponse{
			Daily: daily,
			Summary: models.TradingCalendarSummary{
				TotalDaysTraded:  len(daily),
				TotalRealizedPnL: round2(totalPnL),
				BestDay:          round2(bestDay),
				WorstDay:         round2(worstDay),
				AvgDailyPnL:      avgDailyPnL,
				ProfitableDays:   profitableDays,
				LosingDays:       losingDays,
			},
		},
	})
}

// StrategyEval handles GET /api/trade-strategy-eval — strategy evaluation.
func (h *TradingJournalHandler) StrategyEval(c *gin.Context) {
	ctx := c.Request.Context()

	entries, err := h.loadAllEntries(ctx)
	if err != nil {
		zap.L().Error("load journal entries for strategy eval", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "JOURNAL_LOAD_FAILED", "failed to load trading journal")
		return
	}

	sellResults := h.computeFIFOSellResults(entries)

	type stratAgg struct {
		label       string
		pnls        []float64
		returnPcts  []float64
		wins        int
		losses      int
		cumPnL      float64
		peakCumPnL  float64
		maxDrawdown float64
	}
	stratMap := make(map[string]*stratAgg)

	for _, sr := range sellResults {
		label := sr.StrategyLabel
		if label == "" {
			label = "(none)"
		}
		sa, ok := stratMap[label]
		if !ok {
			sa = &stratAgg{label: label}
			stratMap[label] = sa
		}
		sa.pnls = append(sa.pnls, sr.RealizedPnL)
		sa.returnPcts = append(sa.returnPcts, sr.ReturnPct)
		if sr.RealizedPnL > 0 {
			sa.wins++
		} else if sr.RealizedPnL < 0 {
			sa.losses++
		}
		sa.cumPnL += sr.RealizedPnL
		if sa.cumPnL > sa.peakCumPnL {
			sa.peakCumPnL = sa.cumPnL
		}
		dd := sa.peakCumPnL - sa.cumPnL
		if dd > sa.maxDrawdown {
			sa.maxDrawdown = dd
		}
	}

	strategies := make([]models.StrategyEval, 0, len(stratMap))
	for _, sa := range stratMap {
		count := len(sa.pnls)
		totalPnL := 0.0
		totalReturn := 0.0
		maxProfit := 0.0
		maxLoss := 0.0
		for i, pnl := range sa.pnls {
			totalPnL += pnl
			totalReturn += sa.returnPcts[i]
			if pnl > maxProfit {
				maxProfit = pnl
			}
			if pnl < maxLoss {
				maxLoss = pnl
			}
		}
		avgPnL := 0.0
		avgReturn := 0.0
		if count > 0 {
			avgPnL = round2(totalPnL / float64(count))
			avgReturn = round2(totalReturn / float64(count))
		}
		wr := 0.0
		if count > 0 {
			wr = round2(float64(sa.wins) / float64(count) * 100)
		}
		strategies = append(strategies, models.StrategyEval{
			StrategyLabel: sa.label,
			TradeCount:    count,
			Wins:          sa.wins,
			Losses:        sa.losses,
			WinRate:       wr,
			TotalPnL:      round2(totalPnL),
			AvgPnL:        avgPnL,
			AvgReturnPct:  avgReturn,
			MaxProfit:     round2(maxProfit),
			MaxLoss:       round2(maxLoss),
			MaxDrawdown:   round2(sa.maxDrawdown),
		})
	}
	sort.Slice(strategies, func(i, j int) bool { return strategies[i].TotalPnL > strategies[j].TotalPnL })

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": models.StrategyEvalOverall{Strategies: strategies},
	})
}

// --- Internal helpers ---

// loadAllEntries fetches all journal entries ordered by date.
func (h *TradingJournalHandler) loadAllEntries(ctx context.Context) ([]models.TradingJournalEntry, error) {
	rows, err := h.db.Query(ctx,
		`SELECT id, code, name, type, price, quantity, fees, date::text, notes, strategy_label, created_at
		 FROM trading_journal ORDER BY date ASC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]models.TradingJournalEntry, 0)
	for rows.Next() {
		var e models.TradingJournalEntry
		if err := rows.Scan(&e.ID, &e.Code, &e.Name, &e.Type, &e.Price, &e.Quantity, &e.Fees, &e.Date, &e.Notes, &e.StrategyLabel, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// computeFIFOSellResults computes realized PnL for each sell using FIFO matching.
func (h *TradingJournalHandler) computeFIFOSellResults(entries []models.TradingJournalEntry) []models.SellResult {
	// Build buy lots per stock
	type lot struct {
		remaining int
		price     float64
	}
	buyLots := make(map[string][]*lot)

	results := make([]models.SellResult, 0)

	for _, e := range entries {
		if e.Type == "buy" {
			buyLots[e.Code] = append(buyLots[e.Code], &lot{
				remaining: e.Quantity,
				price:     e.Price,
			})
		} else if e.Type == "sell" {
			lots := buyLots[e.Code]
			remaining := e.Quantity
			cost := 0.0
			matchedQty := 0

			for _, l := range lots {
				if remaining <= 0 {
					break
				}
				take := l.remaining
				if take > remaining {
					take = remaining
				}
				cost += float64(take) * l.price
				matchedQty += take
				l.remaining -= take
				remaining -= take
			}

			// Unmatched shares: assume cost = sell price
			if remaining > 0 {
				cost += float64(remaining) * e.Price
			}

			pnl := e.Price*float64(e.Quantity) - e.Fees - cost
			avgCost := 0.0
			if e.Quantity > 0 {
				avgCost = cost / float64(e.Quantity)
			}
			retPct := 0.0
			if avgCost > 0 {
				retPct = (e.Price - avgCost) / avgCost * 100
			}

			results = append(results, models.SellResult{
				ID:            e.ID,
				Code:          e.Code,
				Name:          e.Name,
				Date:          e.Date,
				SellPrice:     e.Price,
				SellQty:       e.Quantity,
				AvgCost:       round2(avgCost),
				RealizedPnL:   round2(pnl),
				ReturnPct:     round2(retPct),
				StrategyLabel: e.StrategyLabel,
			})
		}
	}

	return results
}

// computeSummaryAndPositions builds summary and positions from entries.
func (h *TradingJournalHandler) computeSummaryAndPositions(_ context.Context, entries []models.TradingJournalEntry) (models.TradingJournalSummary, []models.TradingJournalPosition) {
	sellResults := h.computeFIFOSellResults(entries)

	totalBuyAmount, totalSellAmount, totalFees := 0.0, 0.0, 0.0
	buyCount, sellCount := 0, 0
	realizedPnL := 0.0
	for _, e := range entries {
		totalFees += e.Fees
		if e.Type == "buy" {
			totalBuyAmount += e.Price * float64(e.Quantity)
			buyCount++
		} else {
			totalSellAmount += e.Price * float64(e.Quantity)
			sellCount++
		}
	}
	for _, sr := range sellResults {
		realizedPnL += sr.RealizedPnL
	}

	summary := models.TradingJournalSummary{
		TotalTrades:     len(entries),
		BuyCount:        buyCount,
		SellCount:       sellCount,
		TotalBuyAmount:  round2(totalBuyAmount),
		TotalSellAmount: round2(totalSellAmount),
		TotalFees:       round2(totalFees),
		RealizedPnL:     round2(realizedPnL),
	}

	// Compute positions (current holdings)
	type posAcc struct {
		code       string
		name       string
		buyQty     int
		sellQty    int
		totalCost  float64
		totalBuys  int
		totalSells int
		realized   float64
	}
	posMap := make(map[string]*posAcc)

	for _, e := range entries {
		p, ok := posMap[e.Code]
		if !ok {
			p = &posAcc{code: e.Code, name: e.Name}
			posMap[e.Code] = p
		}
		if e.Type == "buy" {
			p.buyQty += e.Quantity
			p.totalCost += e.Price * float64(e.Quantity)
			p.totalBuys++
		} else {
			p.sellQty += e.Quantity
			p.totalSells++
		}
	}

	// Compute per-stock realized PnL
	stockPnL := make(map[string]float64)
	for _, sr := range sellResults {
		stockPnL[sr.Code] += sr.RealizedPnL
	}

	positions := make([]models.TradingJournalPosition, 0)
	for _, p := range posMap {
		netQty := p.buyQty - p.sellQty
		if netQty <= 0 {
			continue
		}
		avgCost := 0.0
		if p.buyQty > 0 {
			avgCost = p.totalCost / float64(p.buyQty)
		}
		positions = append(positions, models.TradingJournalPosition{
			Code:        p.code,
			Name:        p.name,
			Quantity:    netQty,
			AvgCost:     round2(avgCost),
			TotalCost:   round2(avgCost * float64(netQty)),
			TotalBuys:   p.totalBuys,
			TotalSells:  p.totalSells,
			RealizedPnL: round2(stockPnL[p.code]),
		})
	}

	return summary, positions
}


