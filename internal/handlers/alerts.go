package handlers

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// AlertType classifies the alert category.
type AlertType string

const (
	AlertWarning   AlertType = "warning"
	AlertOpportunity AlertType = "opportunity"
	AlertInfo      AlertType = "info"
)

// Alert represents a single smart alert for a stock.
type Alert struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Type      AlertType `json:"type"`
	Title     string   `json:"title"`
	Message   string   `json:"message"`
	Score     int      `json:"score"`
	Signal    string   `json:"signal"`
	Details   []string `json:"details,omitempty"`
	Timestamp string   `json:"timestamp"`
}

// AlertsResponse is the response for GET /api/alerts.
type AlertsResponse struct {
	OK    bool    `json:"ok"`
	Alerts []Alert `json:"alerts"`
	Count  int     `json:"count"`
	Error  string  `json:"error,omitempty"`
}

// AlertsHandler provides the smart alerts endpoint.
type AlertsHandler struct {
	db        *pgxpool.Pool
	analyze   *AnalyzeHandler
	log       *zap.Logger
	alertsCache *cache.Cache[[]Alert]
}

// NewAlertsHandler creates a new AlertsHandler.
func NewAlertsHandler(
	db *pgxpool.Pool,
	analyze *AnalyzeHandler,
	log *zap.Logger,
) *AlertsHandler {
	return &AlertsHandler{
		db:          db,
		analyze:     analyze,
		log:         log,
		alertsCache: cache.New[[]Alert](),
	}
}

// Alerts returns smart alerts for all watchlist stocks.
func (h *AlertsHandler) Alerts(c *gin.Context) {
	if cached, ok := h.alertsCache.Get("all"); ok {
		c.JSON(http.StatusOK, AlertsResponse{
			OK:     true,
			Alerts: cached,
			Count:  len(cached),
		})
		return
	}

	codes, err := h.loadWatchlistCodes(c.Request.Context())
	if err != nil {
		h.log.Warn("alerts: load watchlist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, AlertsResponse{OK: false, Error: err.Error()})
		return
	}

	if len(codes) == 0 {
		c.JSON(http.StatusOK, AlertsResponse{OK: true, Alerts: []Alert{}, Count: 0})
		return
	}

	// Analyze each stock concurrently (limit to 8 workers)
	analyses := make([]models.StockAnalysis, len(codes))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, code := range codes {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, cd string) {
			defer wg.Done()
			defer func() { <-sem }()
			analyses[idx] = h.analyze.analyzeSingle(context.Background(), cd)
		}(i, code)
	}
	wg.Wait()

	// Generate alerts from analysis results
	var allAlerts []Alert
	for _, analysis := range analyses {
		alerts := generateAlertsForStock(analysis)
		allAlerts = append(allAlerts, alerts...)
	}

	// Sort: warnings first, then opportunities, then info
	sort.Slice(allAlerts, func(i, j int) bool {
		pi := alertPriority(allAlerts[i].Type)
		pj := alertPriority(allAlerts[j].Type)
		if pi != pj {
			return pi < pj
		}
		// Within same type, sort by score descending for opportunities, ascending for warnings
		if allAlerts[i].Type == AlertWarning {
			return allAlerts[i].Score < allAlerts[j].Score
		}
		return allAlerts[i].Score > allAlerts[j].Score
	})

	h.alertsCache.Set("all", allAlerts, 60*time.Second)

	c.JSON(http.StatusOK, AlertsResponse{
		OK:     true,
		Alerts: allAlerts,
		Count:  len(allAlerts),
	})
}

// loadWatchlistCodes returns all stock codes in the user's watchlist.
func (h *AlertsHandler) loadWatchlistCodes(ctx context.Context) ([]string, error) {
	rows, err := h.db.Query(ctx, `SELECT code FROM watchlist ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

// alertPriority returns a numeric priority for sorting (lower = higher priority).
func alertPriority(t AlertType) int {
	switch t {
	case AlertWarning:
		return 0
	case AlertOpportunity:
		return 1
	case AlertInfo:
		return 2
	default:
		return 3
	}
}

// generateAlertsForStock creates alerts based on a single stock's analysis.
func generateAlertsForStock(analysis models.StockAnalysis) []Alert {
	var alerts []Alert
	summary := analysis.Summary
	score := summary.OverallScore
	signal := summary.OverallSignal
	strengths := summary.Strengths
	risks := summary.Risks
	sentiment := analysis.Sentiment
	keyEvents := sentiment.KeyEvents
	code := analysis.Code
	name := analysis.Name
	ts := time.Now().Format(time.RFC3339)

	// High score >= 75: opportunity alert
	if score >= 75 {
		details := make([]string, 0, len(strengths))
		details = append(details, strengths...)
		alerts = append(alerts, Alert{
			Code:      code,
			Name:      name,
			Type:      AlertOpportunity,
			Title:     "高评分机会",
			Message:   name + " 综合评分 " + itoa(score) + "，信号: " + signal,
			Score:     score,
			Signal:    signal,
			Details:   details,
			Timestamp: ts,
		})
	}

	// Low score <= 40: warning alert
	if score <= 40 {
		details := make([]string, 0, len(risks))
		details = append(details, risks...)
		alerts = append(alerts, Alert{
			Code:      code,
			Name:      name,
			Type:      AlertWarning,
			Title:     "低评分风险警告",
			Message:   name + " 综合评分仅 " + itoa(score) + "，信号: " + signal,
			Score:     score,
			Signal:    signal,
			Details:   details,
			Timestamp: ts,
		})
	}

	// Medium score: check for notable keywords in strengths/risks
	if score > 40 && score < 75 {
		for _, s := range strengths {
			if containsNotablePositive(s) {
				alerts = append(alerts, Alert{
					Code:      code,
					Name:      name,
					Type:      AlertInfo,
					Title:     "值得关注",
					Message:   name + ": " + s,
					Score:     score,
					Signal:    signal,
					Timestamp: ts,
				})
				break // one info alert per stock from strengths
			}
		}
		for _, r := range risks {
			if containsNotableNegative(r) {
				alerts = append(alerts, Alert{
					Code:      code,
					Name:      name,
					Type:      AlertInfo,
					Title:     "风险提示",
					Message:   name + ": " + r,
					Score:     score,
					Signal:    signal,
					Timestamp: ts,
				})
				break // one info alert per stock from risks
			}
		}
	}

	// Check sentiment key events for positive/negative news alerts
	for _, event := range keyEvents {
		if containsNotablePositive(event) {
			alerts = append(alerts, Alert{
				Code:      code,
				Name:      name,
				Type:      AlertOpportunity,
				Title:     "利好消息",
				Message:   name + ": " + event,
				Score:     score,
				Signal:    signal,
				Timestamp: ts,
			})
		} else if containsNotableNegative(event) {
			alerts = append(alerts, Alert{
				Code:      code,
				Name:      name,
				Type:      AlertWarning,
				Title:     "利空消息",
				Message:   name + ": " + event,
				Score:     score,
				Signal:    signal,
				Timestamp: ts,
			})
		}
	}

	return alerts
}

// containsNotablePositive checks if text contains positive keywords.
func containsNotablePositive(text string) bool {
	positives := []string{
		"强势", "突破", "涨停", "利好", "增长", "上涨", "金叉", "放量",
		"新高", "主力买入", "资金流入", "业绩预增", "大单买入",
	}
	lower := strings.ToLower(text)
	for _, kw := range positives {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// containsNotableNegative checks if text contains negative keywords.
func containsNotableNegative(text string) bool {
	negatives := []string{
		"弱势", "下跌", "跌停", "利空", "亏损", "下降", "死叉", "缩量",
		"新低", "主力卖出", "资金流出", "业绩预减", "大单卖出", "风险",
	}
	lower := strings.ToLower(text)
	for _, kw := range negatives {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// itoa is a simple int-to-string helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
