package handlers

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TrendHandler handles /api/multi-trend and /api/correlation endpoints.
type TrendHandler struct {
	eastMoney    *services.EastMoneyService
	tencent      *services.TencentService
	db           *pgxpool.Pool
	logger       *zap.Logger
	trendCache   *cache.Cache[[]models.MultiTrendStock]
	corrCache    *cache.Cache[correlationData]
}

type correlationData struct {
	Codes   []string    `json:"codes"`
	Names   []string    `json:"names"`
	Matrix  [][]float64 `json:"matrix"`
	Message string      `json:"message,omitempty"`
}

func NewTrendHandler(eastMoney *services.EastMoneyService, tencent *services.TencentService, db *pgxpool.Pool, logger *zap.Logger) *TrendHandler {
	return &TrendHandler{
		eastMoney:  eastMoney,
		tencent:    tencent,
		db:         db,
		logger:     logger,
		trendCache: cache.New[[]models.MultiTrendStock](),
		corrCache:  cache.New[correlationData](),
	}
}

// watchlistCodes fetches stock codes from the user's watchlist in PostgreSQL.
func (h *TrendHandler) watchlistCodes(ctx context.Context) ([]string, error) {
	rows, err := h.db.Query(ctx, `SELECT code FROM watchlist ORDER BY added_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			continue
		}
		codes = append(codes, code)
	}
	return codes, nil
}

// fetchKlinesCached fetches kline data with a simple in-memory cache.
func (h *TrendHandler) fetchKlinesCached(ctx context.Context, code string, days int) ([]models.KlinePoint, error) {
	cacheKey := code + ":" + strings.TrimSpace(string(rune('0'+days/100))) + strings.TrimSpace(string(rune('0'+(days/10)%10))) + strings.TrimSpace(string(rune('0'+days%10)))
	if cached, ok := h.trendCache.Get(cacheKey); ok {
		// Return cached klines (this is a different type, use a separate cache)
		_ = cached
	}
	return h.eastMoney.FetchKline(ctx, code, days)
}

// MultiTrend handles GET /api/multi-trend.
// @Summary Multi-period trend analysis
// @Description Multi-period trend analysis for watchlist stocks (daily/weekly/monthly)
// @Tags analysis
// @Produce json
// @Param codes query string false "Comma-separated stock codes (defaults to watchlist)"
// @Success 200 {object} models.MultiTrendResponse
// @Router /api/multi-trend [get]
func (h *TrendHandler) MultiTrend(c *gin.Context) {
	h.logger.Info("multi-trend request")

	// Check cache
	if cached, ok := h.trendCache.Get("all"); ok {
		c.JSON(http.StatusOK, models.MultiTrendResponse{
			OK:     true,
			Stocks: cached,
			Cached: true,
		})
		return
	}

	// Determine stock codes
	codesParam := strings.TrimSpace(c.Query("codes"))
	var codeList []string
	if codesParam != "" {
		for _, c := range strings.Split(codesParam, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				codeList = append(codeList, c)
			}
		}
	} else {
		var err error
		codeList, err = h.watchlistCodes(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, "WATCHLIST_FETCH_FAILED", "failed to fetch watchlist")
			return
		}
	}

	if len(codeList) == 0 {
		c.JSON(http.StatusOK, models.MultiTrendResponse{
			OK:     true,
			Stocks: []models.MultiTrendStock{},
			Cached: false,
		})
		return
	}

	// Concurrently fetch data for each stock
	var mu sync.Mutex
	var wg sync.WaitGroup
	results := make([]models.MultiTrendStock, 0, len(codeList))

	for _, rawCode := range codeList {
		wg.Add(1)
		go func(raw string) {
			defer wg.Done()
			code := services.NormalizeCode(raw)
			stock := h.analyzeMultiTrend(c.Request.Context(), code)
			if stock != nil {
				mu.Lock()
				results = append(results, *stock)
				mu.Unlock()
			}
		}(rawCode)
	}
	wg.Wait()

	// Sort by code for deterministic output
	sort.Slice(results, func(i, j int) bool {
		return results[i].Code < results[j].Code
	})

	// Cache for 5 minutes
	h.trendCache.Set("all", results, 5*time.Minute)

	c.JSON(http.StatusOK, models.MultiTrendResponse{
		OK:     true,
		Stocks: results,
		Cached: false,
	})
}

// analyzeMultiTrend computes multi-period trend indicators for a single stock.
func (h *TrendHandler) analyzeMultiTrend(ctx context.Context, code string) *models.MultiTrendStock {
	klines, err := h.eastMoney.FetchKline(ctx, code, 120)
	if err != nil || len(klines) == 0 {
		h.logger.Error("failed to fetch klines for multi-trend",
			zap.String("code", code),
			zap.Error(err),
		)
		return nil
	}

	// Try to get stock name from quote
	name := code
	if quote, err := h.tencent.FetchQuote(ctx, code); err == nil && quote.Name != "" {
		name = quote.Name
	}

	// Aggregate to weekly and monthly
	weeklyKlines := aggregateKlinesToWeekly(klines)
	monthlyKlines := aggregateKlinesToMonthly(klines)

	dailyIndicators := computePeriodStrength(klines)
	weeklyIndicators := computePeriodStrength(weeklyKlines)
	monthlyIndicators := computePeriodStrength(monthlyKlines)

	overallStrength := (dailyIndicators.Strength + weeklyIndicators.Strength + monthlyIndicators.Strength) / 3

	return &models.MultiTrendStock{
		Code:            code,
		Name:            name,
		Daily:           dailyIndicators,
		Weekly:          weeklyIndicators,
		Monthly:         monthlyIndicators,
		OverallStrength: overallStrength,
	}
}

// Correlation handles GET /api/correlation.
// @Summary Pairwise correlation matrix
// @Description Return pairwise Pearson correlation matrix for watchlist stocks (30-day daily returns)
// @Tags analysis
// @Produce json
// @Success 200 {object} models.CorrelationResponse
// @Router /api/correlation [get]
func (h *TrendHandler) Correlation(c *gin.Context) {
	h.logger.Info("correlation request")

	// Check cache
	if cached, ok := h.corrCache.Get("all"); ok {
		c.JSON(http.StatusOK, models.CorrelationResponse{
			OK:     true,
			Codes:  cached.Codes,
			Names:  cached.Names,
			Matrix: cached.Matrix,
			Message: cached.Message,
			Cached: true,
		})
		return
	}

	// Get watchlist stocks
	stockCodes, err := h.watchlistCodes(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "WATCHLIST_FETCH_FAILED", "failed to fetch watchlist")
		return
	}

	if len(stockCodes) < 2 {
		result := correlationData{
			Codes:   []string{},
			Names:   []string{},
			Matrix:  [][]float64{},
			Message: "监控池不足2只股票，无法计算相关性",
		}
		h.corrCache.Set("all", result, 1*time.Minute)
		c.JSON(http.StatusOK, models.CorrelationResponse{
			OK:     true,
			Codes:  result.Codes,
			Names:  result.Names,
			Matrix: result.Matrix,
			Message: result.Message,
			Cached: false,
		})
		return
	}

	// Fetch klines for each stock concurrently
	type stockData struct {
		code    string
		name    string
		returns []float64
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	var stockDataList []stockData

	for _, rawCode := range stockCodes {
		wg.Add(1)
		go func(raw string) {
			defer wg.Done()
			code := services.NormalizeCode(raw)

			klines, err := h.eastMoney.FetchKline(c.Request.Context(), code, 35)
			if err != nil || len(klines) < 10 {
				return
			}

			closes := make([]float64, 0, len(klines))
			for _, k := range klines {
				if k.Close > 0 {
					closes = append(closes, k.Close)
				}
			}
			if len(closes) < 10 {
				return
			}

			// Compute daily returns
			returns := make([]float64, 0, len(closes)-1)
			for i := 1; i < len(closes); i++ {
				if closes[i-1] > 0 {
					returns = append(returns, (closes[i]-closes[i-1])/closes[i-1])
				}
			}
			if len(returns) < 5 {
				return
			}

			// Get stock name
			name := code
			if quote, err := h.tencent.FetchQuote(c.Request.Context(), code); err == nil && quote.Name != "" {
				name = quote.Name
			}

			mu.Lock()
			stockDataList = append(stockDataList, stockData{
				code:    code,
				name:    name,
				returns: returns,
			})
			mu.Unlock()
		}(rawCode)
	}
	wg.Wait()

	if len(stockDataList) < 2 {
		result := correlationData{
			Codes:   []string{},
			Names:   []string{},
			Matrix:  [][]float64{},
			Message: "有效数据不足2只股票，无法计算相关性",
		}
		h.corrCache.Set("all", result, 1*time.Minute)
		c.JSON(http.StatusOK, models.CorrelationResponse{
			OK:     true,
			Codes:  result.Codes,
			Names:  result.Names,
			Matrix: result.Matrix,
			Message: result.Message,
			Cached: false,
		})
		return
	}

	// Sort by code for deterministic output
	sort.Slice(stockDataList, func(i, j int) bool {
		return stockDataList[i].code < stockDataList[j].code
	})

	// Align return series to same length (use the shortest)
	minLen := len(stockDataList[0].returns)
	for _, sd := range stockDataList {
		if len(sd.returns) < minLen {
			minLen = len(sd.returns)
		}
	}
	for i := range stockDataList {
		stockDataList[i].returns = stockDataList[i].returns[len(stockDataList[i].returns)-minLen:]
	}

	// Compute pairwise correlation matrix
	n := len(stockDataList)
	matrix := make([][]float64, n)
	for i := 0; i < n; i++ {
		matrix[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = 1.0
			} else if j < i {
				matrix[i][j] = matrix[j][i] // symmetric
			} else {
				matrix[i][j] = pearsonCorrelation(stockDataList[i].returns, stockDataList[j].returns)
			}
		}
	}

	codesOut := make([]string, n)
	namesOut := make([]string, n)
	for i, sd := range stockDataList {
		codesOut[i] = sd.code
		namesOut[i] = sd.name
	}

	result := correlationData{
		Codes:  codesOut,
		Names:  namesOut,
		Matrix: matrix,
	}

	// Cache for 5 minutes
	h.corrCache.Set("all", result, 5*time.Minute)

	c.JSON(http.StatusOK, models.CorrelationResponse{
		OK:     true,
		Codes:  result.Codes,
		Names:  result.Names,
		Matrix: result.Matrix,
		Cached: false,
	})
}

// --- Technical indicator helpers ---

// aggregateKlinesToWeekly aggregates daily klines into weekly bars (Mon-Fri).
func aggregateKlinesToWeekly(daily []models.KlinePoint) []models.KlinePoint {
	if len(daily) == 0 {
		return nil
	}

	type weekKey struct {
		year int
		week int
	}
	weeks := make(map[weekKey][]models.KlinePoint)
	for _, k := range daily {
		t, err := time.Parse("2006-01-02", k.Date)
		if err != nil {
			continue
		}
		y, w := t.ISOWeek()
		key := weekKey{year: y, week: w}
		weeks[key] = append(weeks[key], k)
	}

	// Sort week keys
	keys := make([]weekKey, 0, len(weeks))
	for k := range weeks {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].year != keys[j].year {
			return keys[i].year < keys[j].year
		}
		return keys[i].week < keys[j].week
	})

	result := make([]models.KlinePoint, 0, len(keys))
	for _, key := range keys {
		bars := weeks[key]
		if len(bars) == 0 {
			continue
		}
		high := bars[0].High
		low := bars[0].Low
		var volume, amount float64
		for _, b := range bars {
			if b.High > high {
				high = b.High
			}
			if b.Low < low {
				low = b.Low
			}
			volume += b.Volume
			amount += b.Amount
		}
		result = append(result, models.KlinePoint{
			Date:   bars[0].Date,
			Open:   bars[0].Open,
			Close:  bars[len(bars)-1].Close,
			High:   high,
			Low:    low,
			Volume: volume,
			Amount: amount,
		})
	}
	return result
}

// aggregateKlinesToMonthly aggregates daily klines into monthly bars.
func aggregateKlinesToMonthly(daily []models.KlinePoint) []models.KlinePoint {
	if len(daily) == 0 {
		return nil
	}

	months := make(map[string][]models.KlinePoint)
	for _, k := range daily {
		if len(k.Date) >= 7 {
			monthKey := k.Date[:7]
			months[monthKey] = append(months[monthKey], k)
		}
	}

	keys := make([]string, 0, len(months))
	for k := range months {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]models.KlinePoint, 0, len(keys))
	for _, key := range keys {
		bars := months[key]
		if len(bars) == 0 {
			continue
		}
		high := bars[0].High
		low := bars[0].Low
		var volume, amount float64
		for _, b := range bars {
			if b.High > high {
				high = b.High
			}
			if b.Low < low {
				low = b.Low
			}
			volume += b.Volume
			amount += b.Amount
		}
		result = append(result, models.KlinePoint{
			Date:   bars[0].Date,
			Open:   bars[0].Open,
			Close:  bars[len(bars)-1].Close,
			High:   high,
			Low:    low,
			Volume: volume,
			Amount: amount,
		})
	}
	return result
}

// computeMA computes simple moving average for the given period.
func computeMA(closes []float64, period int) float64 {
	if len(closes) < period {
		return 0
	}
	sum := 0.0
	for _, c := range closes[len(closes)-period:] {
		sum += c
	}
	return math.Round(sum/float64(period)*10000) / 10000
}

// computeRSI computes RSI indicator for the given period.
func computeRSI(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 0
	}
	gains := make([]float64, 0, len(closes)-1)
	losses := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		diff := closes[i] - closes[i-1]
		if diff > 0 {
			gains = append(gains, diff)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -diff)
		}
	}
	recentGains := gains[len(gains)-period:]
	recentLosses := losses[len(losses)-period:]
	avgGain := 0.0
	avgLoss := 0.0
	for i := 0; i < period; i++ {
		avgGain += recentGains[i]
		avgLoss += recentLosses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)
	if avgLoss == 0 {
		return 100.0
	}
	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))
	return math.Round(rsi*100) / 100
}

// computeVolumeTrend determines volume trend: increasing / decreasing / stable.
func computeVolumeTrend(volumes []float64) string {
	if len(volumes) < 5 {
		return "stable"
	}
	recent := volumes[len(volumes)-5:]
	avgRecent := 0.0
	for _, v := range recent {
		avgRecent += v
	}
	avgRecent /= float64(len(recent))

	var older []float64
	if len(volumes) >= 10 {
		older = volumes[len(volumes)-10 : len(volumes)-5]
	} else {
		older = volumes[:5]
	}
	avgOlder := 0.0
	for _, v := range older {
		avgOlder += v
	}
	avgOlder /= float64(len(older))

	if avgOlder == 0 {
		return "stable"
	}
	ratio := avgRecent / avgOlder
	if ratio > 1.15 {
		return "increasing"
	} else if ratio < 0.85 {
		return "decreasing"
	}
	return "stable"
}

// computePeriodStrength computes trend indicators for a single period's klines.
func computePeriodStrength(klines []models.KlinePoint) models.PeriodIndicators {
	if len(klines) < 5 {
		return models.PeriodIndicators{
			ReturnPct:   0,
			MA5:         0,
			MA10:        0,
			MA20:        0,
			MAAligned:   false,
			RSI:         0,
			VolumeTrend: "stable",
			Strength:    50,
		}
	}

	closes := make([]float64, len(klines))
	volumes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
		volumes[i] = k.Volume
	}

	// Period return %
	firstClose := klines[0].Close
	lastClose := klines[len(klines)-1].Close
	periodReturn := 0.0
	if firstClose > 0 {
		periodReturn = math.Round((lastClose-firstClose)/firstClose*10000) / 100
	}

	// Moving averages
	ma5 := computeMA(closes, 5)
	ma10 := computeMA(closes, 10)
	ma20 := computeMA(closes, 20)

	// MA alignment (bullish: MA5 > MA10 > MA20)
	maAligned := false
	if ma5 > 0 && ma10 > 0 && ma20 > 0 {
		maAligned = ma5 > ma10 && ma10 > ma20
	}

	// RSI
	rsi := computeRSI(closes, 14)

	// Volume trend
	volTrend := computeVolumeTrend(volumes)

	// Strength score (0-100)
	score := 50.0
	// Return contribution (+/- 20)
	retContrib := periodReturn * 2
	if retContrib > 20 {
		retContrib = 20
	}
	if retContrib < -20 {
		retContrib = -20
	}
	score += retContrib

	// MA alignment
	if maAligned {
		score += 15
	} else if ma5 > 0 && ma10 > 0 && ma20 > 0 {
		if ma5 < ma10 && ma10 < ma20 {
			score -= 10
		}
	}

	// RSI contribution
	if rsi > 70 {
		score += 10
	} else if rsi > 0 && rsi < 30 {
		score -= 10
	}

	// Volume trend
	if volTrend == "increasing" && periodReturn > 0 {
		score += 5
	} else if volTrend == "decreasing" && periodReturn < 0 {
		score -= 5
	}

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return models.PeriodIndicators{
		ReturnPct:   periodReturn,
		MA5:         ma5,
		MA10:        ma10,
		MA20:        ma20,
		MAAligned:   maAligned,
		RSI:         rsi,
		VolumeTrend: volTrend,
		Strength:    int(math.Round(score)),
	}
}

// pearsonCorrelation computes the Pearson correlation coefficient between two series.
func pearsonCorrelation(x, y []float64) float64 {
	n := len(x)
	if n < 2 || n != len(y) {
		return 0
	}
	meanX := 0.0
	meanY := 0.0
	for i := 0; i < n; i++ {
		meanX += x[i]
		meanY += y[i]
	}
	meanX /= float64(n)
	meanY /= float64(n)

	var varX, varY, covXY float64
	for i := 0; i < n; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		varX += dx * dx
		varY += dy * dy
		covXY += dx * dy
	}

	denom := math.Sqrt(varX * varY)
	if denom == 0 {
		return 0
	}
	result := covXY / denom
	return math.Round(result*10000) / 10000
}
