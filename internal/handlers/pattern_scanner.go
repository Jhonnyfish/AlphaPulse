package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/logger"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PatternResult represents a single detected pattern.
type PatternResult struct {
	Pattern    string  `json:"pattern"`
	Category   string  `json:"category"`   // kline, chart, volume
	Direction  string  `json:"direction"`   // bullish, bearish, neutral
	Confidence float64 `json:"confidence"`  // 0.0 - 1.0
	Date       string  `json:"date"`
	Description string `json:"description"`
	Code       string  `json:"code,omitempty"`
	Name       string  `json:"name,omitempty"`
}

// PatternScannerResponse is the full response for /api/pattern-scanner.
type PatternScannerResponse struct {
	OK       bool                  `json:"ok"`
	Patterns []PatternResult       `json:"patterns"`
	Summary  PatternScannerSummary `json:"summary"`
	Cached   bool                  `json:"cached"`
}

// PatternScannerSummary provides aggregate counts.
type PatternScannerSummary struct {
	Total        int            `json:"total"`
	Bullish      int            `json:"bullish"`
	Bearish      int            `json:"bearish"`
	Neutral      int            `json:"neutral"`
	ByCategory   map[string]int `json:"by_category"`
	Scanned      int            `json:"scanned"`
}

// PatternScannerHandler handles pattern scanning requests.
type PatternScannerHandler struct {
	eastMoney  *services.EastMoneyService
	tencent    *services.TencentService
	db         *pgxpool.Pool
	scanCache  *cache.Cache[PatternScannerResponse]
}

// NewPatternScannerHandler creates a new PatternScannerHandler.
func NewPatternScannerHandler(eastMoney *services.EastMoneyService, tencent *services.TencentService, db *pgxpool.Pool) *PatternScannerHandler {
	return &PatternScannerHandler{
		eastMoney: eastMoney,
		tencent:   tencent,
		db:        db,
		scanCache: cache.New[PatternScannerResponse](),
	}
}

// Scan handles GET /api/pattern-scanner
// @Summary 形态扫描
// @Description 扫描自选股的技术形态，包括K线形态、图形形态和量价异动
// @Tags pattern-scanner
// @Accept json
// @Produce json
// @Success 200 {object} PatternScannerResponse
// @Router /api/pattern-scanner [get]
func (h *PatternScannerHandler) Scan(c *gin.Context) {
	start := time.Now()
	cacheKey := "pattern_scanner_all"

	// Check cache
	if cached, ok := h.scanCache.Get(cacheKey); ok {
		cached.Cached = true
		c.JSON(http.StatusOK, cached)
		return
	}

	// Load watchlist codes from DB
	codes := h.loadWatchlistCodes(c)
	if len(codes) == 0 {
		c.JSON(http.StatusOK, PatternScannerResponse{
			OK:       true,
			Patterns: []PatternResult{},
			Summary:  PatternScannerSummary{ByCategory: map[string]int{}},
			Cached:   false,
		})
		return
	}

	// Scan each stock concurrently
	var (
		mu          sync.Mutex
		allPatterns []PatternResult
		scannedCount int
		wg          sync.WaitGroup
	)

	semaphore := make(chan struct{}, 5) // limit concurrency
	for code := range codes {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(stockCode string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			patterns, err := h.scanStock(c.Request.Context(), stockCode)
			if err != nil {
				logger.Warn("pattern scan failed",
					zap.String("code", stockCode),
					zap.Error(err))
				return
			}

			mu.Lock()
			allPatterns = append(allPatterns, patterns...)
			scannedCount++
			mu.Unlock()
		}(code)
	}
	wg.Wait()

	// Sort by confidence descending
	sort.Slice(allPatterns, func(i, j int) bool {
		return allPatterns[i].Confidence > allPatterns[j].Confidence
	})

	// Build summary
	summary := buildPatternSummary(allPatterns, scannedCount)

	response := PatternScannerResponse{
		OK:       true,
		Patterns: allPatterns,
		Summary:  summary,
		Cached:   false,
	}

	// Cache for 5 minutes
	h.scanCache.Set(cacheKey, response, 5*time.Minute)

	logger.Info("pattern scanner completed",
		zap.Int("scanned", scannedCount),
		zap.Int("patterns", len(allPatterns)),
		zap.Duration("latency", time.Since(start)))

	c.JSON(http.StatusOK, response)
}

// loadWatchlistCodes loads all stock codes from the watchlist table.
func (h *PatternScannerHandler) loadWatchlistCodes(c *gin.Context) map[string]bool {
	codes := make(map[string]bool)
	rows, err := h.db.Query(c.Request.Context(), "SELECT code FROM watchlist")
	if err != nil {
		return codes
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err == nil {
			codes[code] = true
		}
	}
	return codes
}

// scanStock fetches klines and runs all pattern detectors for a single stock.
func (h *PatternScannerHandler) scanStock(ctx context.Context, code string) ([]PatternResult, error) {
	klines, err := h.eastMoney.FetchKline(ctx, code, 60)
	if err != nil {
		return nil, fmt.Errorf("fetch klines: %w", err)
	}
	if len(klines) < 10 {
		return nil, nil // not enough data
	}

	// Try to get stock name from quote
	var stockName string
	if quote, err := h.tencent.FetchQuote(ctx, code); err == nil {
		stockName = quote.Name
	}

	var allPatterns []PatternResult

	// K-line patterns (need at least 3 bars)
	if len(klines) >= 3 {
		allPatterns = append(allPatterns, detectKlinePatterns(klines, code, stockName)...)
	}

	// Chart patterns (need at least 20 bars)
	if len(klines) >= 20 {
		allPatterns = append(allPatterns, detectChartPatterns(klines, code, stockName)...)
	}

	// Volume patterns (need at least 20 bars)
	if len(klines) >= 20 {
		allPatterns = append(allPatterns, detectVolumePatterns(klines, code, stockName)...)
	}

	return allPatterns, nil
}

// ==================== K-line Pattern Detection ====================

func detectKlinePatterns(klines []models.KlinePoint, code, name string) []PatternResult {
	var results []PatternResult
	n := len(klines)
	bars := klines[n-3:]
	lastDate := klines[n-1].Date

	// Extract OHLC for last 3 bars
	o := [3]float64{bars[0].Open, bars[1].Open, bars[2].Open}
	c := [3]float64{bars[0].Close, bars[1].Close, bars[2].Close}
	high := [3]float64{bars[0].High, bars[1].High, bars[2].High}
	low := [3]float64{bars[0].Low, bars[1].Low, bars[2].Low}

	body := func(i int) float64 { return abs(c[i] - o[i]) }
	upperShadow := func(i int) float64 { return high[i] - maxF(o[i], c[i]) }
	lowerShadow := func(i int) float64 { return minF(o[i], c[i]) - low[i] }
	isUp := func(i int) bool { return c[i] > o[i] }
	isDown := func(i int) bool { return c[i] < o[i] }

	// --- Doji (十字星) ---
	price0 := (o[2] + c[2]) / 2
	if price0 > 0 && body(2) < price0*0.001 && body(2) > 0 {
		conf := minF(0.9, 0.5+(price0*0.001-body(2))/(price0*0.001)*0.4)
		results = append(results, PatternResult{
			Pattern:    "十字星",
			Category:   "kline",
			Direction:  "neutral",
			Confidence: round2(conf),
			Date:       lastDate,
			Description: "开盘价与收盘价几乎相同，多空双方力量均衡，信号方向待确认",
			Code:       code,
			Name:       name,
		})
	}

	// --- Hammer (锤子线) ---
	if body(2) > 0 {
		ls := lowerShadow(2)
		us := upperShadow(2)
		b := body(2)
		if ls >= 2.0*b && us < b*0.5 {
			ratio := ls / b
			conf := minF(0.95, 0.6+(ratio-2.0)*0.1)
			results = append(results, PatternResult{
				Pattern:    "锤子线",
				Category:   "kline",
				Direction:  "bullish",
				Confidence: round2(conf),
				Date:       lastDate,
				Description: fmt.Sprintf("下影线长度为实体%.1f倍，上影线极短，底部看涨信号", ratio),
				Code:       code,
				Name:       name,
			})
		}
	}

	// --- Engulfing (吞没形态) ---
	prevBodyAbs := abs(c[1] - o[1])
	currBodyAbs := abs(c[2] - o[2])
	if prevBodyAbs > 0 && currBodyAbs > prevBodyAbs {
		// Bullish engulfing
		if c[1] < o[1] && c[2] > o[2] && o[2] <= c[1] && c[2] >= o[1] {
			conf := minF(0.9, 0.6+(currBodyAbs/prevBodyAbs-1)*0.3)
			results = append(results, PatternResult{
				Pattern:    "吞没形态",
				Category:   "kline",
				Direction:  "bullish",
				Confidence: round2(conf),
				Date:       lastDate,
				Description: "阳线完全吞没前一根阴线，底部反转看涨信号",
				Code:       code,
				Name:       name,
			})
		}
		// Bearish engulfing
		if c[1] > o[1] && c[2] < o[2] && o[2] >= c[1] && c[2] <= o[1] {
			conf := minF(0.9, 0.6+(currBodyAbs/prevBodyAbs-1)*0.3)
			results = append(results, PatternResult{
				Pattern:    "吞没形态",
				Category:   "kline",
				Direction:  "bearish",
				Confidence: round2(conf),
				Date:       lastDate,
				Description: "阴线完全吞没前一根阳线，顶部反转看跌信号",
				Code:       code,
				Name:       name,
			})
		}
	}

	// --- Morning Star (早晨之星) ---
	if isDown(0) && body(1) < body(0)*0.3 && isUp(2) {
		midFirst := (o[0] + c[0]) / 2
		if c[2] > midFirst {
			results = append(results, PatternResult{
				Pattern:    "早晨之星",
				Category:   "kline",
				Direction:  "bullish",
				Confidence: 0.8,
				Date:       lastDate,
				Description: "三根K线组合：下跌→小实体→阳线收于第一根中点之上，强烈看涨反转",
				Code:       code,
				Name:       name,
			})
		}
	}

	// --- Evening Star (黄昏之星) ---
	if isUp(0) && body(1) < body(0)*0.3 && isDown(2) {
		midFirst := (o[0] + c[0]) / 2
		if c[2] < midFirst {
			results = append(results, PatternResult{
				Pattern:    "黄昏之星",
				Category:   "kline",
				Direction:  "bearish",
				Confidence: 0.8,
				Date:       lastDate,
				Description: "三根K线组合：上涨→小实体→阴线收于第一根中点之下，强烈看跌反转",
				Code:       code,
				Name:       name,
			})
		}
	}

	// --- Three White Soldiers (三白兵) ---
	if isUp(0) && isUp(1) && isUp(2) && c[0] < c[1] && c[1] < c[2] && o[0] < o[1] && o[1] < o[2] {
		results = append(results, PatternResult{
			Pattern:    "三白兵",
			Category:   "kline",
			Direction:  "bullish",
			Confidence: 0.85,
			Date:       lastDate,
			Description: "连续三根阳线，收盘价逐步抬高，强势上涨信号",
			Code:       code,
			Name:       name,
		})
	}

	// --- Three Black Crows (三黑鸦) ---
	if isDown(0) && isDown(1) && isDown(2) && c[0] > c[1] && c[1] > c[2] && o[0] > o[1] && o[1] > o[2] {
		results = append(results, PatternResult{
			Pattern:    "三黑鸦",
			Category:   "kline",
			Direction:  "bearish",
			Confidence: 0.85,
			Date:       lastDate,
			Description: "连续三根阴线，收盘价逐步走低，强势下跌信号",
			Code:       code,
			Name:       name,
		})
	}

	return results
}

// ==================== Chart Pattern Detection ====================

func detectChartPatterns(klines []models.KlinePoint, code, name string) []PatternResult {
	var results []PatternResult
	n := len(klines)
	lastDate := klines[n-1].Date

	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	for i, k := range klines {
		closes[i] = k.Close
		highs[i] = k.High
		lows[i] = k.Low
	}

	priceRange := maxSlice(highs) - minSlice(lows)
	if priceRange <= 0 {
		return results
	}
	tolerance := priceRange * 0.03 // 3% tolerance

	// Find pivots
	window := maxI(3, n/15)
	lowPivots, highPivots := findPivots(closes, window)

	// --- Double Bottom (双底) ---
	if len(lowPivots) >= 2 {
		for i := 0; i < len(lowPivots)-1; i++ {
			for j := i + 1; j < len(lowPivots); j++ {
				idx1, val1 := lowPivots[i].idx, lowPivots[i].val
				idx2, val2 := lowPivots[j].idx, lowPivots[j].val
				if abs(val1-val2) < tolerance && absInt(idx2-idx1) >= 5 {
					// Check for a peak between them
					peakVal := 0.0
					found := false
					for _, hp := range highPivots {
						if hp.idx > idx1 && hp.idx < idx2 {
							if hp.val > peakVal {
								peakVal = hp.val
							}
							found = true
						}
					}
					if found && peakVal > val1+tolerance {
						conf := minF(0.9, 0.6+(1-abs(val1-val2)/tolerance)*0.3)
						results = append(results, PatternResult{
							Pattern:    "双底",
							Category:   "chart",
							Direction:  "bullish",
							Confidence: round2(conf),
							Date:       lastDate,
							Description: fmt.Sprintf("W底形态，两个低点约%.2f，颈线%.2f，看涨反转", val1, peakVal),
							Code:       code,
							Name:       name,
						})
						break
					}
				}
			}
			if hasPattern(results, "双底") {
				break
			}
		}
	}

	// --- Double Top (双顶) ---
	if len(highPivots) >= 2 {
		for i := 0; i < len(highPivots)-1; i++ {
			for j := i + 1; j < len(highPivots); j++ {
				idx1, val1 := highPivots[i].idx, highPivots[i].val
				idx2, val2 := highPivots[j].idx, highPivots[j].val
				if abs(val1-val2) < tolerance && absInt(idx2-idx1) >= 5 {
					troughVal := val1 * 2 // large initial value
					found := false
					for _, lp := range lowPivots {
						if lp.idx > idx1 && lp.idx < idx2 {
							if lp.val < troughVal {
								troughVal = lp.val
							}
							found = true
						}
					}
					if found && troughVal < val1-tolerance {
						conf := minF(0.9, 0.6+(1-abs(val1-val2)/tolerance)*0.3)
						results = append(results, PatternResult{
							Pattern:    "双顶",
							Category:   "chart",
							Direction:  "bearish",
							Confidence: round2(conf),
							Date:       lastDate,
							Description: fmt.Sprintf("M顶形态，两个高点约%.2f，颈线%.2f，看跌反转", val1, troughVal),
							Code:       code,
							Name:       name,
						})
						break
					}
				}
			}
			if hasPattern(results, "双顶") {
				break
			}
		}
	}

	// --- Ascending Triangle (上升三角形) ---
	if len(highPivots) >= 2 && len(lowPivots) >= 2 {
		recentHighs := lastN(highPivots, 3)
		recentLows := lastN(lowPivots, 3)
		highVals := pivotVals(recentHighs)
		lowVals := pivotVals(recentLows)
		highSpread := maxSlice(highVals) - minSlice(highVals)

		if highSpread < tolerance && len(lowVals) >= 2 {
			rising := true
			for j := 1; j < len(lowVals); j++ {
				if lowVals[j] < lowVals[j-1]-tolerance*0.2 {
					rising = false
					break
				}
			}
			if rising && lowVals[len(lowVals)-1] > lowVals[0]+tolerance*0.5 {
				conf := minF(0.85, 0.6+(1-highSpread/tolerance)*0.25)
				results = append(results, PatternResult{
					Pattern:    "上升三角形",
					Category:   "chart",
					Direction:  "bullish",
					Confidence: round2(conf),
					Date:       lastDate,
					Description: fmt.Sprintf("阻力位约%.2f（水平），支撑位逐步抬升，看涨整理形态", maxSlice(highVals)),
					Code:       code,
					Name:       name,
				})
			}
		}
	}

	// --- Descending Triangle (下降三角形) ---
	if len(lowPivots) >= 2 && len(highPivots) >= 2 {
		recentLows := lastN(lowPivots, 3)
		recentHighs := lastN(highPivots, 3)
		lowVals := pivotVals(recentLows)
		highVals := pivotVals(recentHighs)
		lowSpread := maxSlice(lowVals) - minSlice(lowVals)

		if lowSpread < tolerance && len(highVals) >= 2 {
			declining := true
			for j := 1; j < len(highVals); j++ {
				if highVals[j] > highVals[j-1]+tolerance*0.2 {
					declining = false
					break
				}
			}
			if declining && highVals[len(highVals)-1] < highVals[0]-tolerance*0.5 {
				conf := minF(0.85, 0.6+(1-lowSpread/tolerance)*0.25)
				results = append(results, PatternResult{
					Pattern:    "下降三角形",
					Category:   "chart",
					Direction:  "bearish",
					Confidence: round2(conf),
					Date:       lastDate,
					Description: fmt.Sprintf("支撑位约%.2f（水平），阻力位逐步下降，看跌整理形态", minSlice(lowVals)),
					Code:       code,
					Name:       name,
				})
			}
		}
	}

	// --- Rectangle/Box (箱体整理) ---
	if len(lowPivots) >= 2 && len(highPivots) >= 2 {
		recentLows := lastN(lowPivots, 3)
		recentHighs := lastN(highPivots, 3)
		lowVals := pivotVals(recentLows)
		highVals := pivotVals(recentHighs)
		lowSpread := maxSlice(lowVals) - minSlice(lowVals)
		highSpread := maxSlice(highVals) - minSlice(highVals)

		if lowSpread < tolerance && highSpread < tolerance {
			avgRange := maxSlice(highVals) - minSlice(lowVals)
			if avgRange > tolerance {
				conf := minF(0.8, 0.55+(1-(lowSpread+highSpread)/(2*tolerance))*0.25)
				results = append(results, PatternResult{
					Pattern:    "箱体整理",
					Category:   "chart",
					Direction:  "neutral",
					Confidence: round2(conf),
					Date:       lastDate,
					Description: fmt.Sprintf("价格在%.2f-%.2f区间震荡，关注突破方向", minSlice(lowVals), maxSlice(highVals)),
					Code:       code,
					Name:       name,
				})
			}
		}
	}

	return results
}

// ==================== Volume Pattern Detection ====================

func detectVolumePatterns(klines []models.KlinePoint, code, name string) []PatternResult {
	var results []PatternResult
	n := len(klines)
	lastDate := klines[n-1].Date

	closes := make([]float64, n)
	highs := make([]float64, n)
	volumes := make([]float64, n)
	for i, k := range klines {
		closes[i] = k.Close
		highs[i] = k.High
		volumes[i] = k.Volume
	}

	// 20-day average volume
	vol20 := volumes[n-20:]
	avgVol20 := avgSlice(vol20)
	if avgVol20 <= 0 {
		return results
	}

	recentHighMax := maxSlice(highs[n-20 : n-1])
	latestVol := volumes[n-1]
	latestClose := closes[n-1]

	// --- Volume Breakout (放量突破) ---
	if latestVol > 2*avgVol20 && latestClose > recentHighMax {
		volRatio := latestVol / avgVol20
		conf := minF(0.95, 0.6+(volRatio-2)*0.1)
		results = append(results, PatternResult{
			Pattern:    "放量突破",
			Category:   "volume",
			Direction:  "bullish",
			Confidence: round2(conf),
			Date:       lastDate,
			Description: fmt.Sprintf("成交量为20日均量的%.1f倍，价格突破近期高点%.2f，看涨", volRatio, recentHighMax),
			Code:       code,
			Name:       name,
		})
	}

	// --- Contraction Pullback (缩量回调) ---
	if n >= 5 {
		peakClose := maxSlice(closes[n-10 : n-1])
		pullback := 0.0
		if peakClose > 0 {
			pullback = (peakClose - latestClose) / peakClose
		}
		recentVol5 := volumes[n-5:]
		recentAvgVol := avgSlice(recentVol5)
		if pullback > 0.02 && recentAvgVol < avgVol20*0.5 {
			volRatio := recentAvgVol / avgVol20
			conf := minF(0.85, 0.55+(0.5-volRatio)*0.6)
			results = append(results, PatternResult{
				Pattern:    "缩量回调",
				Category:   "volume",
				Direction:  "bullish",
				Confidence: round2(conf),
				Date:       lastDate,
				Description: fmt.Sprintf("价格从高点回调%.1f%%，但成交量降至20日均量的%.0f%%，回调无量看涨", pullback*100, volRatio*100),
				Code:       code,
				Name:       name,
			})
		}
	}

	// --- Volume-Price Divergence (量价背离) ---
	if n >= 10 {
		firstHalfClose := avgSlice(closes[n-10 : n-5])
		secondHalfClose := avgSlice(closes[n-5:])
		firstHalfVol := avgSlice(volumes[n-10 : n-5])
		secondHalfVol := avgSlice(volumes[n-5:])

		// Bearish divergence: price rising but volume declining
		if secondHalfClose > firstHalfClose*1.01 && secondHalfVol < firstHalfVol*0.8 {
			conf := minF(0.85, 0.6+(1-secondHalfVol/firstHalfVol)*0.5)
			results = append(results, PatternResult{
				Pattern:    "量价背离",
				Category:   "volume",
				Direction:  "bearish",
				Confidence: round2(conf),
				Date:       lastDate,
				Description: "价格创近期新高但成交量递减，上涨动能减弱，看跌背离",
				Code:       code,
				Name:       name,
			})
		}

		// Bullish divergence: price declining but volume declining (selling exhaustion)
		if secondHalfClose < firstHalfClose*0.99 && secondHalfVol < firstHalfVol*0.7 {
			conf := minF(0.8, 0.55+(1-secondHalfVol/firstHalfVol)*0.5)
			results = append(results, PatternResult{
				Pattern:    "量价背离",
				Category:   "volume",
				Direction:  "bullish",
				Confidence: round2(conf),
				Date:       lastDate,
				Description: "价格创新低但成交量萎缩，卖压衰竭，底部看涨背离",
				Code:       code,
				Name:       name,
			})
		}
	}

	return results
}

// ==================== Pivot Detection ====================

type pivot struct {
	idx int
	val float64
}

func findPivots(data []float64, window int) (mins, maxs []pivot) {
	for i := window; i < len(data)-window; i++ {
		isMin := true
		isMax := true
		for j := 1; j <= window; j++ {
			if data[i] > data[i-j] || data[i] > data[i+j] {
				isMin = false
			}
			if data[i] < data[i-j] || data[i] < data[i+j] {
				isMax = false
			}
		}
		if isMin {
			mins = append(mins, pivot{idx: i, val: data[i]})
		}
		if isMax {
			maxs = append(maxs, pivot{idx: i, val: data[i]})
		}
	}
	return
}

// ==================== Summary Builder ====================

func buildPatternSummary(patterns []PatternResult, scanned int) PatternScannerSummary {
	summary := PatternScannerSummary{
		Total:      len(patterns),
		ByCategory: make(map[string]int),
		Scanned:    scanned,
	}
	for _, p := range patterns {
		switch p.Direction {
		case "bullish":
			summary.Bullish++
		case "bearish":
			summary.Bearish++
		default:
			summary.Neutral++
		}
		summary.ByCategory[p.Category]++
	}
	return summary
}

// ==================== Helper Functions ====================

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func maxSlice(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	m := data[0]
	for _, v := range data[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func minSlice(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	m := data[0]
	for _, v := range data[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func avgSlice(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func pivotVals(pivots []pivot) []float64 {
	vals := make([]float64, len(pivots))
	for i, p := range pivots {
		vals[i] = p.val
	}
	return vals
}

func lastN(pivots []pivot, n int) []pivot {
	if len(pivots) <= n {
		return pivots
	}
	return pivots[len(pivots)-n:]
}

func hasPattern(results []PatternResult, name string) bool {
	for _, r := range results {
		if r.Pattern == name {
			return true
		}
	}
	return false
}
