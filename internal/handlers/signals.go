package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SignalHandler handles signal-related API endpoints (Module 18).
type SignalHandler struct {
	alpha300 *services.Alpha300Cache
	tencent  *services.TencentService
	east     *services.EastMoneyService
	log      *zap.Logger

	anomaliesCache *cache.Cache[anomaliesPayload]
	signalCalCache *cache.Cache[signalCalendarPayload]
}

// NewSignalHandler creates a new SignalHandler.
func NewSignalHandler(
	alpha300 *services.Alpha300Cache,
	tencent *services.TencentService,
	east *services.EastMoneyService,
	log *zap.Logger,
) *SignalHandler {
	return &SignalHandler{
		alpha300:       alpha300,
		tencent:        tencent,
		east:           east,
		log:            log,
		anomaliesCache: cache.New[anomaliesPayload](),
		signalCalCache: cache.New[signalCalendarPayload](),
	}
}

// ==================== Anomalies ====================

type anomalyItem struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	ChangePct  float64 `json:"change_pct"`
	VolumeRatio float64 `json:"volume_ratio"`
	Reason     string  `json:"reason"`
}

type anomaliesPayload struct {
	Anomalies struct {
		LimitUp     []anomalyItem `json:"limit_up"`
		LimitDown   []anomalyItem `json:"limit_down"`
		VolumeSurge []anomalyItem `json:"volume_surge"`
		BigMove     []anomalyItem `json:"big_move"`
	} `json:"anomalies"`
	Scanned   int    `json:"scanned"`
	FetchedAt string `json:"fetched_at"`
}

// @Summary      异常检测
// @Description  扫描 Alpha300 候选股中的市场异常(涨停/跌停/放量/大幅波动)
// @Tags         signals
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/anomalies [get]
func (h *SignalHandler) Anomalies(c *gin.Context) {
	if cached, ok := h.anomaliesCache.Get("anomalies"); ok {
		c.JSON(http.StatusOK, mergeMap(gin.H{"ok": true, "cached": true}, structToMap(cached)))
		return
	}

	ctx := c.Request.Context()
	candidates, err := h.alpha300.GetTopN(ctx, 100)
	if err != nil {
		h.log.Warn("fetch alpha300 candidates for anomalies", zap.Error(err))
		c.JSON(http.StatusOK, emptyAnomalies())
		return
	}

	if len(candidates) == 0 {
		c.JSON(http.StatusOK, emptyAnomalies())
		return
	}

	// Build candidate name map and code list
	candidateNames := make(map[string]string, len(candidates))
	codes := make([]string, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, cand := range candidates {
		code := services.NormalizeCode(cand.TsCode)
		if code == "" {
			code = cand.Code
		}
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		codes = append(codes, code)
		candidateNames[code] = cand.Name
	}

	// Fetch quotes in parallel with semaphore
	quotesMap := make(map[string]models.Quote, len(codes))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for _, code := range codes {
		wg.Add(1)
		go func(c string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			q, err := h.tencent.FetchQuote(ctx, c)
			if err != nil {
				return
			}
			if q.Price > 0 {
				mu.Lock()
				quotesMap[c] = q
				mu.Unlock()
			}
		}(code)
	}
	wg.Wait()

	// Classify anomalies
	var limitUp, limitDown, volumeSurge, bigMove []anomalyItem

	for code, q := range quotesMap {
		pct := q.ChangePercent
		price := q.Price
		name := q.Name
		if name == "" {
			name = candidateNames[code]
		}
		if name == "" {
			name = code
		}
		amount := q.Volume * q.Price // approximate

		item := anomalyItem{
			Code:      code,
			Name:      name,
			Price:     price,
			ChangePct: round2(pct),
		}

		// Limit up/down: >= 9.8% or <= -9.8%
		if pct >= 9.8 {
			item.Reason = fmt.Sprintf("涨停 %+.2f%%", pct)
			limitUp = append(limitUp, item)
			continue
		}
		if pct <= -9.8 {
			item.Reason = fmt.Sprintf("跌停 %+.2f%%", pct)
			limitDown = append(limitDown, item)
			continue
		}

		// Volume surge: amount > 5e8 with price move > 2%
		if (amount > 500_000_000) && (pct > 2 || pct < -2) {
				item.Reason = fmt.Sprintf("成交额%.1f亿 涨跌%+.2f%%", amount/1e8, pct)
				volumeSurge = append(volumeSurge, item)
			continue
		}

		// Big move: abs(pct) >= 5
		nearHigh := q.High > 0 && price > 0 && price > q.High*0.98 && pct > 0
		nearLow := q.Low > 0 && price > 0 && price < q.Low*1.02 && pct < 0

		if pct >= 5 {
			item.Reason = fmt.Sprintf("大幅上涨 %+.2f%%", pct)
			if nearHigh {
				item.Reason += " · 冲高"
			}
			bigMove = append(bigMove, item)
			continue
		}
		if pct <= -5 {
			item.Reason = fmt.Sprintf("大幅下跌 %+.2f%%", pct)
			if nearLow {
				item.Reason += " · 探底"
			}
			bigMove = append(bigMove, item)
			continue
		}

		if nearHigh && pct > 3 {
			item.Reason = fmt.Sprintf("快速拉升 %+.2f%% · 接近日高", pct)
			bigMove = append(bigMove, item)
		} else if nearLow && pct < -3 {
			item.Reason = fmt.Sprintf("快速下跌 %+.2f%% · 接近日低", pct)
			bigMove = append(bigMove, item)
		}
	}

	// Sort
	sort.Slice(limitUp, func(i, j int) bool { return limitUp[i].ChangePct > limitUp[j].ChangePct })
	sort.Slice(limitDown, func(i, j int) bool { return limitDown[i].ChangePct < limitDown[j].ChangePct })
	sort.Slice(volumeSurge, func(i, j int) bool { return abs(volumeSurge[i].ChangePct) > abs(volumeSurge[j].ChangePct) })
	sort.Slice(bigMove, func(i, j int) bool { return abs(bigMove[i].ChangePct) > abs(bigMove[j].ChangePct) })

	payload := anomaliesPayload{
		Scanned:   len(codes),
		FetchedAt: time.Now().Format(time.RFC3339),
	}
	payload.Anomalies.LimitUp = limitUp
	payload.Anomalies.LimitDown = limitDown
	payload.Anomalies.VolumeSurge = volumeSurge
	payload.Anomalies.BigMove = bigMove

	h.anomaliesCache.Set("anomalies", payload, 60*time.Second)
	c.JSON(http.StatusOK, mergeMap(gin.H{"ok": true, "cached": false}, structToMap(payload)))
}

// ==================== Signal History ====================

type signalHistoryEntry struct {
	Timestamp string `json:"timestamp"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type signalHistoryPayload struct {
	OK          bool                `json:"ok"`
	Items       []signalHistoryEntry `json:"items"`
	Total       int                 `json:"total"`
	LevelCounts map[string]int      `json:"level_counts"`
}

// @Summary      信号历史
// @Description  获取历史信号记录，支持按级别和代码过滤
// @Tags         signals
// @Produce      json
// @Param        level  query      string  false  "信号级别"
// @Param        code   query      string  false  "股票代码"
// @Param        limit  query      int     false  "结果数量上限" default(100)
// @Success      200  {object}  signalHistoryPayload
// @Router       /api/signal-history [get]
func (h *SignalHandler) SignalHistory(c *gin.Context) {
	level := strings.TrimSpace(c.Query("level"))
	code := strings.TrimSpace(c.Query("code"))
	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := parseInt(v); err == nil && n >= 1 && n <= 500 {
			limit = n
		}
	}

	const signalHistoryPath = "/home/finn/.hermes/scripts/signal_history.json"
	entries := loadSignalHistory(signalHistoryPath)

	// Apply filters
	if level != "" {
		filtered := make([]signalHistoryEntry, 0, len(entries))
		for _, e := range entries {
			if strings.EqualFold(e.Level, level) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}
	if code != "" {
		codeUpper := strings.ToUpper(code)
		filtered := make([]signalHistoryEntry, 0, len(entries))
		for _, e := range entries {
			if strings.Contains(strings.ToUpper(e.Code), codeUpper) ||
				strings.Contains(strings.ToUpper(e.Name), codeUpper) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	// Sort newest first
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp > entries[j].Timestamp
	})

	total := len(entries)
	if total > limit {
		entries = entries[:limit]
	}

	// Compute level counts from all entries (unfiltered)
	allEntries := loadSignalHistory(signalHistoryPath)
	levelCounts := make(map[string]int)
	for _, e := range allEntries {
		lv := e.Level
		if lv == "" {
			lv = "unknown"
		}
		levelCounts[lv]++
	}

	c.JSON(http.StatusOK, signalHistoryPayload{
		OK:          true,
		Items:       entries,
		Total:       total,
		LevelCounts: levelCounts,
	})
}

// ==================== Signal Calendar ====================

type signalEntry struct {
	Date        string  `json:"date"`
	SignalType  string  `json:"signal_type"`
	SignalName  string  `json:"signal_name"`
	SignalIcon  string  `json:"signal_icon"`
	SignalColor string  `json:"signal_color"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type signalCalendarPayload struct {
	Code      string        `json:"code"`
	Name      string        `json:"name"`
	Signals   []signalEntry `json:"signals"`
	Days      int           `json:"days"`
	FetchedAt string        `json:"fetched_at"`
}

// @Summary      信号日历
// @Description  检测历史技术信号(MACD/KDJ/MA交叉, RSI超买超卖, 布林突破)
// @Tags         signals
// @Produce      json
// @Param        code  query      string  true   "股票代码"
// @Param        days  query      int     false  "回溯天数" default(120)
// @Success      200  {object}  signalCalendarPayload
// @Router       /api/signal-calendar [get]
func (h *SignalHandler) SignalCalendar(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "code 参数不能为空"})
		return
	}

	days := 120
	if v := c.Query("days"); v != "" {
		if n, err := parseInt(v); err == nil && n >= 30 && n <= 500 {
			days = n
		}
	}

	code = services.NormalizeCode(code)
	cacheKey := fmt.Sprintf("signal_calendar:%s:%d", code, days)
	if cached, ok := h.signalCalCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, mergeMap(gin.H{"ok": true, "cached": true}, structToMap(cached)))
		return
	}

	ctx := c.Request.Context()

	// Fetch klines with warmup period
	warmup := 70
	klines, err := h.east.FetchKline(ctx, code, days+warmup)
	if err != nil {
		h.log.Warn("fetch klines for signal calendar", zap.String("code", code), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "KLINE_FETCH_FAILED", "failed to fetch kline data")
		return
	}

	if len(klines) < warmup+5 {
		c.JSON(http.StatusOK, gin.H{
			"ok": true, "code": code, "name": code, "signals": []signalEntry{},
			"days": days, "fetched_at": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Get stock name from first quote fetch
	name := code
	if q, err := h.tencent.FetchQuote(ctx, code); err == nil && q.Name != "" {
		name = q.Name
	}

	// Build arrays
	allDates := make([]string, len(klines))
	allCloses := make([]float64, len(klines))
	allHighs := make([]float64, len(klines))
	allLows := make([]float64, len(klines))
	for i, k := range klines {
		allDates[i] = k.Date
		allCloses[i] = k.Close
		allHighs[i] = k.High
		allLows[i] = k.Low
	}

	reportStart := max(0, len(klines)-days)
	var signals []signalEntry

	for i := reportStart; i < len(klines); i++ {
		date := allDates[i]
		price := allCloses[i]

		// MACD cross detection (need >= 35 data points)
		if i >= 35 {
			closesSub := allCloses[:i+1]
			macdPrev := services.CalculateMACD(closesSub[:len(closesSub)-1])
			macdCurr := services.CalculateMACD(closesSub)
			if macdPrev.DIF <= macdPrev.DEA && macdCurr.DIF > macdCurr.DEA {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "macd_golden",
					SignalName: "MACD金叉", SignalIcon: "🔴",
					SignalColor: "bullish", Price: round2(price),
					Description: fmt.Sprintf("DIF(%.3f)上穿DEA(%.3f)", macdCurr.DIF, macdCurr.DEA),
				})
			} else if macdPrev.DIF >= macdPrev.DEA && macdCurr.DIF < macdCurr.DEA {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "macd_death",
					SignalName: "MACD死叉", SignalIcon: "🟢",
					SignalColor: "bearish", Price: round2(price),
					Description: fmt.Sprintf("DIF(%.3f)下穿DEA(%.3f)", macdCurr.DIF, macdCurr.DEA),
				})
			}
		}

		// KDJ cross detection (need >= 9 data points)
		if i >= 9 {
			closesSub := allCloses[:i+1]
			highsSub := allHighs[:i+1]
			lowsSub := allLows[:i+1]
			kdjPrev := services.CalculateKDJ(closesSub[:len(closesSub)-1], highsSub[:len(highsSub)-1], lowsSub[:len(lowsSub)-1], 9)
			kdjCurr := services.CalculateKDJ(closesSub, highsSub, lowsSub, 9)
			if kdjPrev.K <= kdjPrev.D && kdjCurr.K > kdjCurr.D {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "kdj_golden",
					SignalName: "KDJ金叉", SignalIcon: "🟡",
					SignalColor: "bullish", Price: round2(price),
					Description: fmt.Sprintf("K(%.1f)上穿D(%.1f)", kdjCurr.K, kdjCurr.D),
				})
			} else if kdjPrev.K >= kdjPrev.D && kdjCurr.K < kdjCurr.D {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "kdj_death",
					SignalName: "KDJ死叉", SignalIcon: "🔵",
					SignalColor: "bearish", Price: round2(price),
					Description: fmt.Sprintf("K(%.1f)下穿D(%.1f)", kdjCurr.K, kdjCurr.D),
				})
			}
		}

		// MA cross detection (need >= 20 data points)
		if i >= 20 {
			ma := func(arr []float64, period, idx int) float64 {
				if idx < period-1 {
					return 0
				}
				sum := 0.0
				for j := idx - period + 1; j <= idx; j++ {
					sum += arr[j]
				}
				return sum / float64(period)
			}
			ma5Prev := ma(allCloses, 5, i-1)
			ma10Prev := ma(allCloses, 10, i-1)
			ma20Prev := ma(allCloses, 20, i-1)
			ma5Curr := ma(allCloses, 5, i)
			ma10Curr := ma(allCloses, 10, i)
			ma20Curr := ma(allCloses, 20, i)

			if ma5Prev <= ma10Prev && ma5Curr > ma10Curr {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "ma_cross_up",
					SignalName: "MA5上穿MA10", SignalIcon: "📈",
					SignalColor: "bullish", Price: round2(price),
					Description: fmt.Sprintf("MA5(%.2f)上穿MA10(%.2f)", ma5Curr, ma10Curr),
				})
			} else if ma5Prev >= ma10Prev && ma5Curr < ma10Curr {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "ma_cross_down",
					SignalName: "MA5下穿MA10", SignalIcon: "📉",
					SignalColor: "bearish", Price: round2(price),
					Description: fmt.Sprintf("MA5(%.2f)下穿MA10(%.2f)", ma5Curr, ma10Curr),
				})
			}
			if ma10Prev <= ma20Prev && ma10Curr > ma20Curr {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "ma_cross_up",
					SignalName: "MA10上穿MA20", SignalIcon: "📈",
					SignalColor: "bullish", Price: round2(price),
					Description: fmt.Sprintf("MA10(%.2f)上穿MA20(%.2f)", ma10Curr, ma20Curr),
				})
			} else if ma10Prev >= ma20Prev && ma10Curr < ma20Curr {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "ma_cross_down",
					SignalName: "MA10下穿MA20", SignalIcon: "📉",
					SignalColor: "bearish", Price: round2(price),
					Description: fmt.Sprintf("MA10(%.2f)下穿MA20(%.2f)", ma10Curr, ma20Curr),
				})
			}
		}

		// RSI overbought/oversold (need >= 15 data points)
		if i >= 15 {
			closesSub := allCloses[:i+1]
			rsiVal := services.CalculateRSI(closesSub, 14)
			if rsiVal > 70 {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "rsi_overbought",
					SignalName: "RSI超买", SignalIcon: "🔥",
					SignalColor: "warning", Price: round2(price),
					Description: fmt.Sprintf("RSI(14)=%.1f > 70", rsiVal),
				})
			} else if rsiVal < 30 {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "rsi_oversold",
					SignalName: "RSI超卖", SignalIcon: "💧",
					SignalColor: "bullish", Price: round2(price),
					Description: fmt.Sprintf("RSI(14)=%.1f < 30", rsiVal),
				})
			}
		}

		// Bollinger band breakout (need >= 20 data points)
		if i >= 20 {
			closesSub := allCloses[:i+1]
			boll := services.CalculateBollinger(closesSub, 20)
			if boll.Upper > 0 && price > boll.Upper {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "boll_upper",
					SignalName: "突破上轨", SignalIcon: "⬆️",
					SignalColor: "warning", Price: round2(price),
					Description: fmt.Sprintf("收盘价(%.2f) > 布林上轨(%.2f)", price, boll.Upper),
				})
			}
			if boll.Lower > 0 && price < boll.Lower {
				signals = append(signals, signalEntry{
					Date: date, SignalType: "boll_lower",
					SignalName: "跌破下轨", SignalIcon: "⬇️",
					SignalColor: "bullish", Price: round2(price),
					Description: fmt.Sprintf("收盘价(%.2f) < 布林下轨(%.2f)", price, boll.Lower),
				})
			}
		}
	}

	// Sort by date descending
	sort.Slice(signals, func(i, j int) bool {
		return signals[i].Date > signals[j].Date
	})

	payload := signalCalendarPayload{
		Code:      code,
		Name:      name,
		Signals:   signals,
		Days:      days,
		FetchedAt: time.Now().Format(time.RFC3339),
	}

	h.signalCalCache.Set(cacheKey, payload, 5*time.Minute)
	c.JSON(http.StatusOK, mergeMap(gin.H{"ok": true, "cached": false}, structToMap(payload)))
}

// ==================== Helpers ====================

func loadSignalHistory(path string) []signalHistoryEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return []signalHistoryEntry{}
	}
	var entries []signalHistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return []signalHistoryEntry{}
	}
	return entries
}

func emptyAnomalies() gin.H {
	return gin.H{
		"ok": true,
		"anomalies": gin.H{
			"limit_up":     []anomalyItem{},
			"limit_down":   []anomalyItem{},
			"volume_surge": []anomalyItem{},
			"big_move":     []anomalyItem{},
		},
		"scanned":    0,
		"fetched_at": time.Now().Format(time.RFC3339),
	}
}

func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid integer")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func structToMap(v interface{}) map[string]interface{} {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	return m
}

func mergeMap(base map[string]interface{}, overlay map[string]interface{}) map[string]interface{} {
	for k, v := range overlay {
		base[k] = v
	}
	return base
}

// CacheStats returns cache stats for monitoring.
func (h *SignalHandler) CacheStats() map[string]interface{} {
	return map[string]interface{}{
		"anomalies": h.anomaliesCache,
		"signal_calendar": h.signalCalCache,
	}
}
