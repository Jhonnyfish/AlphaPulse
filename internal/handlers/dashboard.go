package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const defaultReportsDir = "/home/finn/.uzi-skill/reports"

// DashboardHandler handles GET /api/dashboard-summary — composite dashboard data.
type DashboardHandler struct {
	db            *pgxpool.Pool
	tencentSvc    *services.TencentService
	eastMoneySvc  *services.EastMoneyService
	watchlistH    *WatchlistHandler
	log           *zap.Logger
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(db *pgxpool.Pool, tencentSvc *services.TencentService, eastMoneySvc *services.EastMoneyService, watchlistH *WatchlistHandler, log *zap.Logger) *DashboardHandler {
	return &DashboardHandler{
		db:           db,
		tencentSvc:   tencentSvc,
		eastMoneySvc: eastMoneySvc,
		watchlistH:   watchlistH,
		log:          log,
	}
}

// DashboardSummary handles GET /api/dashboard-summary
// Returns: indices, watchlist stats, top gainers/losers, active alerts, recent activity, last report date.
//
//	@Summary		Dashboard summary
//	@Description	Returns composite dashboard data: market indices, watchlist stats, top movers, alerts, activity, reports
//	@Tags			dashboard
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]interface{}
//	@Router			/api/dashboard-summary [get]
func (h *DashboardHandler) DashboardSummary(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	result := gin.H{"ok": true}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// 1. Market indices
	wg.Add(1)
	go func() {
		defer wg.Done()
		indices := h.fetchIndices(ctx)
		mu.Lock()
		result["indices"] = indices
		mu.Unlock()
	}()

	// 2. Watchlist summary + top movers
	wg.Add(1)
	go func() {
		defer wg.Done()
		wlSummary, topGainers, topLosers := h.fetchWatchlistSummary(ctx)
		mu.Lock()
		result["watchlist"] = wlSummary
		result["top_gainers"] = topGainers
		result["top_losers"] = topLosers
		mu.Unlock()
	}()

	// 3. Active alerts count
	wg.Add(1)
	go func() {
		defer wg.Done()
		count := h.countActiveAlerts(ctx)
		mu.Lock()
		result["active_alerts"] = count
		mu.Unlock()
	}()

	// 4. Recent activity (last 5)
	wg.Add(1)
	go func() {
		defer wg.Done()
		entries := h.recentActivity()
		mu.Lock()
		result["recent_activity"] = entries
		mu.Unlock()
	}()

	// 5. Last report date
	wg.Add(1)
	go func() {
		defer wg.Done()
		date := h.lastReportDate()
		mu.Lock()
		result["last_report_date"] = date
		mu.Unlock()
	}()

	// 6. Market overview (advance/decline/flat + indices)
	wg.Add(1)
	go func() {
		defer wg.Done()
		overview := h.fetchMarketOverview(ctx)
		mu.Lock()
		result["market_overview"] = overview
		mu.Unlock()
	}()

	// 7. Sectors (top 20)
	wg.Add(1)
	go func() {
		defer wg.Done()
		sectors := h.fetchTopSectors(ctx)
		mu.Lock()
		result["sectors"] = sectors
		mu.Unlock()
	}()

	// 8. Recent signals (last 7 days)
	wg.Add(1)
	go func() {
		defer wg.Done()
		signals := h.fetchRecentSignals(ctx, 7)
		mu.Lock()
		result["signals"] = signals
		mu.Unlock()
	}()

	wg.Wait()

	c.JSON(http.StatusOK, result)
}

// fetchIndices fetches major market index quotes.
func (h *DashboardHandler) fetchIndices(ctx context.Context) []gin.H {
	type indexDef struct {
		code string
		name string
	}
	defs := []indexDef{
		{"sh000001", "上证指数"},
		{"sz399001", "深证成指"},
		{"sz399006", "创业板指"},
	}

	indices := make([][2]string, len(defs))
	for i, d := range defs {
		indices[i] = [2]string{d.code, d.name}
	}

	quotes, err := h.tencentSvc.FetchIndexQuotes(ctx, indices)
	if err != nil {
		h.log.Warn("dashboard: failed to fetch index quotes", zap.Error(err))
		// Return stub data on error
		result := make([]gin.H, len(defs))
		for i, d := range defs {
			result[i] = gin.H{
				"code": d.code, "name": d.name,
				"price": nil, "change": nil, "change_pct": nil,
				"prev_close": nil, "open": nil, "high": nil, "low": nil,
				"volume": nil, "amount": nil,
			}
		}
		return result
	}

	result := make([]gin.H, 0, len(quotes))
	for _, q := range quotes {
		result = append(result, gin.H{
			"code":        q.Code,
			"name":        q.Name,
			"price":       q.Price,
			"change":      q.Change,
			"change_pct":  q.ChangePercent,
			"prev_close":  q.PrevClose,
			"open":        0.0,
			"high":        0.0,
			"low":         0.0,
			"volume":      q.Volume,
			"amount":      q.Amount,
		})
	}
	return result
}

// fetchWatchlistSummary returns watchlist stats, top gainers, and top losers.
func (h *DashboardHandler) fetchWatchlistSummary(ctx context.Context) (summary gin.H, gainers []gin.H, losers []gin.H) {
	rows, err := h.db.Query(ctx,
		`SELECT code, COALESCE(name, '') FROM watchlist ORDER BY added_at DESC`)
	if err != nil {
		h.log.Warn("dashboard: failed to query watchlist", zap.Error(err))
		return gin.H{"total": 0, "avg_change_pct": nil}, nil, nil
	}
	defer rows.Close()

	type wlStock struct {
		code string
		name string
	}
	var stocks []wlStock
	for rows.Next() {
		var s wlStock
		if err := rows.Scan(&s.code, &s.name); err != nil {
			continue
		}
		stocks = append(stocks, s)
	}

	if len(stocks) == 0 {
		return gin.H{"total": 0, "avg_change_pct": nil}, nil, nil
	}

	// Fetch quotes for watchlist stocks (limit to first 30 for performance)
	limit := len(stocks)
	if limit > 30 {
		limit = 30
	}

	type stockQuote struct {
		code        string
		name        string
		price       float64
		changePct   float64
		hasQuote    bool
	}

	items := make([]stockQuote, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex

	sem := make(chan struct{}, 5) // limit concurrency
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			q, err := h.tencentSvc.FetchQuote(ctx, stocks[idx].code)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				items[idx] = stockQuote{code: stocks[idx].code, name: stocks[idx].name}
			} else {
				items[idx] = stockQuote{
					code:      stocks[idx].code,
					name:      stocks[idx].name,
					price:     q.Price,
					changePct: q.ChangePercent,
					hasQuote:  true,
				}
			}
		}(i)
	}
	wg.Wait()

	// Calculate stats
	totalChange := 0.0
	changeCount := 0
	var valid []stockQuote
	for _, it := range items {
		if it.hasQuote {
			totalChange += it.changePct
			changeCount++
			valid = append(valid, it)
		}
	}

	avgChange := 0.0
	if changeCount > 0 {
		avgChange = totalChange / float64(changeCount)
	}

	summary = gin.H{
		"total":           len(stocks),
		"avg_change_pct":  avgChange,
	}

	// Sort for top gainers/losers
	sort.Slice(valid, func(i, j int) bool {
		return valid[i].changePct > valid[j].changePct
	})

	gainers = make([]gin.H, 0, 3)
	for i := 0; i < 3 && i < len(valid); i++ {
		gainers = append(gainers, gin.H{
			"code":        valid[i].code,
			"name":        valid[i].name,
			"price":       valid[i].price,
			"change_pct":  valid[i].changePct,
		})
	}

	losers = make([]gin.H, 0, 3)
	for i := len(valid) - 1; i >= 0 && len(losers) < 3; i-- {
		losers = append(losers, gin.H{
			"code":        valid[i].code,
			"name":        valid[i].name,
			"price":       valid[i].price,
			"change_pct":  valid[i].changePct,
		})
	}

	return summary, gainers, losers
}

// countActiveAlerts returns the count of enabled custom alerts.
func (h *DashboardHandler) countActiveAlerts(ctx context.Context) int {
	var count int
	err := h.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM custom_alerts WHERE COALESCE(enabled, true) = true`).Scan(&count)
	if err != nil {
		h.log.Warn("dashboard: failed to count alerts", zap.Error(err))
		return 0
	}
	return count
}

// activityLogEntry matches the activity log JSON format.
type dashboardActivityEntry struct {
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	Timestamp string `json:"timestamp"`
}

// recentActivity returns the last 5 activity log entries.
func (h *DashboardHandler) recentActivity() []dashboardActivityEntry {
	logPath := os.Getenv("ACTIVITY_LOG_PATH")
	if logPath == "" {
		logPath = defaultActivityLogPath
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		return []dashboardActivityEntry{}
	}

	var entries []dashboardActivityEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return []dashboardActivityEntry{}
	}

	// Last 5, newest first
	if len(entries) > 5 {
		entries = entries[len(entries)-5:]
	}
	// Reverse
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries
}

// lastReportDate returns the date of the most recent daily report.
func (h *DashboardHandler) lastReportDate() string {
	dir := os.Getenv("REPORTS_DIR")
	if dir == "" {
		dir = defaultReportsDir
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var dates []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "daily_report_") && strings.HasSuffix(name, ".md") {
			date := strings.TrimPrefix(name, "daily_report_")
			date = strings.TrimSuffix(date, ".md")
			dates = append(dates, date)
		}
	}

	if len(dates) == 0 {
		return ""
	}

	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	return dates[0]
}

// fetchMarketOverview returns market overview data: indices with advance/decline/flat counts.
func (h *DashboardHandler) fetchMarketOverview(ctx context.Context) gin.H {
	// Define the 6 major indices (same as MarketHandler.MarketOverview)
	indices := [][2]string{
		{"sh000001", "上证指数"},
		{"sz399001", "深证成指"},
		{"sz399006", "创业板指"},
		{"sh000688", "科创50"},
		{"sh000300", "沪深300"},
		{"sh000905", "中证500"},
	}

	type indexResult struct {
		quotes []models.IndexQuote
		err    error
	}
	type breadthResult struct {
		breadth models.MarketBreadth
		err     error
	}

	idxCh := make(chan indexResult, 1)
	brCh := make(chan breadthResult, 1)

	go func() {
		quotes, err := h.tencentSvc.FetchIndexQuotes(ctx, indices)
		idxCh <- indexResult{quotes: quotes, err: err}
	}()
	go func() {
		breadth, err := h.eastMoneySvc.FetchMarketBreadth(ctx)
		brCh <- breadthResult{breadth: breadth, err: err}
	}()

	ir := <-idxCh
	br := <-brCh

	var indexQuotes []models.IndexQuote
	if ir.err != nil {
		h.log.Warn("dashboard: failed to fetch index quotes for overview", zap.Error(ir.err))
		indexQuotes = []models.IndexQuote{}
	} else {
		indexQuotes = ir.quotes
	}

	breadth := br.breadth
	if br.err != nil {
		h.log.Warn("dashboard: failed to fetch market breadth", zap.Error(br.err))
		breadth = models.MarketBreadth{
			UpCount: 0, DownCount: 0, FlatCount: 0,
			Sentiment: "中性", SentimentRatio: 50,
		}
	}

	return gin.H{
		"ok":           true,
		"indices":      indexQuotes,
		"market":       breadth,
		"updated_at":   time.Now().Format(time.RFC3339),
	}
}

// fetchTopSectors returns the top 20 sectors by absolute change.
func (h *DashboardHandler) fetchTopSectors(ctx context.Context) []models.Sector {
	sectors, err := h.eastMoneySvc.FetchSectors(ctx)
	if err != nil {
		h.log.Warn("dashboard: failed to fetch sectors", zap.Error(err))
		return []models.Sector{}
	}

	// Limit to top 20
	if len(sectors) > 20 {
		sectors = sectors[:20]
	}
	return sectors
}

// fetchRecentSignals returns recent signal history entries (last N days).
func (h *DashboardHandler) fetchRecentSignals(ctx context.Context, days int) []signalHistoryEntry {
	const signalHistoryPath = "/home/finn/.hermes/scripts/signal_history.json"
	entries := loadSignalHistory(signalHistoryPath)

	// Filter by days
	if days > 0 {
		cutoff := time.Now().AddDate(0, 0, -days)
		filtered := make([]signalHistoryEntry, 0, len(entries))
		for _, e := range entries {
			t, err := time.Parse(time.RFC3339, e.Timestamp)
			if err != nil {
				// Try other formats
				t, err = time.Parse("2006-01-02 15:04:05", e.Timestamp)
			}
			if err == nil && t.After(cutoff) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	// Sort newest first
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp > entries[j].Timestamp
	})

	// Limit to 50 entries for the dashboard
	if len(entries) > 50 {
		entries = entries[:50]
	}

	return entries
}
