package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
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

type AnalyzeHandler struct {
	newsSvc            *services.NewsService
	tushareDB          *services.TushareDB        // Primary data source, may be nil
	tencent            *services.TencentService   // Real-time quote fallback
	eastMoney          *services.EastMoneyService // Near-real-time kline fallback
	db                 *pgxpool.Pool              // DB for portfolio lookups, may be nil
	logger             *zap.Logger
	scoreHistory       *ScoreHistoryHandler
	quoteCache         *cache.Cache[models.Quote]
	klineCache         *cache.Cache[[]models.KlinePoint]
	flowCache          *cache.Cache[[]models.MoneyFlowDay]
	sectorsCache       *cache.Cache[[]models.StockSector]
	newsCache          *cache.Cache[[]models.NewsItem]
	announcementsCache *cache.Cache[[]models.Announcement]
}

func NewAnalyzeHandler(newsSvc *services.NewsService, logger *zap.Logger) *AnalyzeHandler {
	return &AnalyzeHandler{
		newsSvc:            newsSvc,
		logger:             logger,
		quoteCache:         cache.New[models.Quote](),
		klineCache:         cache.New[[]models.KlinePoint](),
		flowCache:          cache.New[[]models.MoneyFlowDay](),
		sectorsCache:       cache.New[[]models.StockSector](),
		newsCache:          cache.New[[]models.NewsItem](),
		announcementsCache: cache.New[[]models.Announcement](),
	}
}

// SetScoreHistory sets the score history handler for recording scores.
func (h *AnalyzeHandler) SetScoreHistory(sh *ScoreHistoryHandler) {
	h.scoreHistory = sh
}

// SetTushareDB sets the Tushare local database service as primary data source.
func (h *AnalyzeHandler) SetTushareDB(db *services.TushareDB) {
	h.tushareDB = db
}

// SetDB sets the database pool for portfolio lookups.
func (h *AnalyzeHandler) SetDB(pool *pgxpool.Pool) {
	h.db = pool
}

// SetRealtime sets the real-time data services for live quote/kline fallback.
func (h *AnalyzeHandler) SetRealtime(tencent *services.TencentService, em *services.EastMoneyService) {
	h.tencent = tencent
	h.eastMoney = em
}

// @Summary      8维度综合分析
// @Description  对指定股票进行8维度综合分析，支持批量分析(逗号分隔，最多10只)
// @Tags         analyze
// @Accept       json
// @Produce      json
// @Param        code  query      string  true  "股票代码，多个用逗号分隔"
// @Success      200  {object}  interface{}
// @Router       /analyze [get]
func (h *AnalyzeHandler) Analyze(c *gin.Context) {
	codeParam := strings.TrimSpace(c.Query("code"))
	if codeParam == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}

	h.logger.Info("analyze request",
		zap.String("codes", codeParam),
	)

	codeList := strings.Split(codeParam, ",")
	var cleaned []string
	for _, code := range codeList {
		code = strings.TrimSpace(code)
		if code != "" {
			if err := services.ValidateStockCode(code); err != nil {
				writeError(c, http.StatusBadRequest, "INVALID_CODE_FORMAT", err.Error())
				return
			}
			cleaned = append(cleaned, code)
		}
	}
	if len(cleaned) == 0 {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code cannot be empty")
		return
	}
	if len(cleaned) > 10 {
		writeError(c, http.StatusBadRequest, "TOO_MANY_CODES", "最多支持10只股票批量分析")
		return
	}

	if len(cleaned) == 1 {
		result := h.analyzeSingle(c.Request.Context(), cleaned[0])
		c.JSON(http.StatusOK, result)
		return
	}

	// Batch analysis — concurrent
	results := make([]models.StockAnalysis, len(cleaned))
	var wg sync.WaitGroup
	for i, code := range cleaned {
		wg.Add(1)
		go func(idx int, c string) {
			defer wg.Done()
			results[idx] = h.analyzeSingle(context.Background(), c)
		}(i, code)
	}
	wg.Wait()
	c.JSON(http.StatusOK, results)
}

func (h *AnalyzeHandler) analyzeSingle(ctx context.Context, code string) models.StockAnalysis {
	return h.analyzeSingleWithMode(ctx, code, false)
}

// analyzeSingleWithMode runs 8-dimension analysis. If fast=true, news/announcements skip external API fallback.
func (h *AnalyzeHandler) analyzeSingleWithMode(ctx context.Context, code string, fast bool) models.StockAnalysis {
	code = services.NormalizeCode(code)
	errs := make(map[string]string)

	h.logger.Info("analyzing single stock", zap.String("code", code))

	// Fetch all data sources concurrently
	var (
		quote                                                   models.Quote
		klines                                                  []models.KlinePoint
		flows                                                   []models.MoneyFlowDay
		sectors                                                 []models.StockSector
		news                                                    []models.NewsItem
		anns                                                    []models.Announcement
		fins                                                    []services.FinancialData
		hsgt                                                    []services.HsgtData
		top10                                                   []services.HsgtTop10Data
		hkHold                                                  []services.HkHoldData
		marginD                                                 []services.MarginDetailData
		quoteErr, klineErr, flowErr, sectorErr, newsErr, annErr error
	)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		quote, quoteErr = h.fetchQuote(ctx, code)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		klines, klineErr = h.fetchKlines(ctx, code)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		flows, flowErr = h.fetchFlow(ctx, code)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sectors, sectorErr = h.fetchSectors(ctx, code)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		news, newsErr = h.fetchNewsWithMode(ctx, code, fast)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		anns, annErr = h.fetchAnnouncementsWithMode(ctx, code, fast)
	}()

	wg.Wait()

	// Fetch P0 data (financials, northbound, margin) - these use tushareDB directly
	var sectorPerf *services.SectorPerformance
	var turnoverRate float64
	if h.tushareDB != nil {
		fins, _ = h.tushareDB.FetchFinancials(ctx, code, 8)
		top10, _ = h.tushareDB.FetchHsgtTop10ByCode(ctx, code, 10)
		hkHold, _ = h.tushareDB.FetchHkHoldByCode(ctx, code, 5)
		marginD, _ = h.tushareDB.FetchMarginDetailHistory(ctx, code, 10)
		hsgt, _ = h.tushareDB.FetchHsgtHistory(ctx, 10)
		if sp, err := h.tushareDB.FetchSectorPerformance(ctx, code); err == nil {
			sectorPerf = &sp
		}
		if basic, err := h.tushareDB.FetchDailyBasic(ctx, code); err == nil {
			turnoverRate = basic.TurnoverRate
		}
	}

	if quoteErr != nil {
		errs["quote"] = quoteErr.Error()
	}
	if klineErr != nil {
		errs["klines"] = klineErr.Error()
	}
	if flowErr != nil {
		errs["money_flow"] = flowErr.Error()
	}
	if sectorErr != nil {
		errs["sectors"] = sectorErr.Error()
	}
	if newsErr != nil {
		errs["news"] = newsErr.Error()
	}
	if annErr != nil {
		errs["announcements"] = annErr.Error()
	}

	sectorNames := make([]string, 0, len(sectors))
	for _, s := range sectors {
		sectorNames = append(sectorNames, s.Name)
	}

	// Run 8 analysis dimensions
	analysis := models.StockAnalysis{
		Code:         services.StockCode6(code),
		Name:         quote.Name,
		Version:      "3.0",
		Quote:        quote,
		VolumePrice:  services.AnalyzeVolumePrice(quote, klines, turnoverRate),
		Valuation:    services.AnalyzeValuation(quote),
		Volatility:   services.AnalyzeVolatility(quote),
		MoneyFlow:    services.AnalyzeMoneyFlow(flows),
		Technical:    services.AnalyzeTechnical(klines),
		Sector:       services.AnalyzeSector(quote, sectorNames, sectorPerf),
		Sentiment:    services.AnalyzeSentiment(news, anns),
		Fundamentals: services.AnalyzeFundamentals(fins),
		Northbound:   services.AnalyzeNorthbound(hsgt, top10, hkHold),
		MarginDetail: services.AnalyzeMarginDetail(marginD),
		DataSources: func() map[string]string {
			ds := map[string]string{
				"quote":        "tencent/tushare",
				"klines":       "tushare/eastmoney",
				"money_flow":   "tushare",
				"sector":       "tushare",
				"sentiment":    "db/eastmoney",
				"fundamentals": "tushare/fina_indicator",
				"margin":       "tushare/margin_detail",
			}
			if len(top10) > 0 {
				ds["northbound"] = "tushare/hsgt_top10"
			} else if len(hkHold) > 0 {
				ds["northbound"] = "tushare/hk_hold"
			} else {
				ds["northbound"] = "tushare/hsgt"
			}
			return ds
		}(),
		Errors:    errs,
		FetchedAt: time.Now(),
	}

	analysis.Summary = services.BuildSummary(&analysis)

	// Run trend analysis
	trendAnalyzer := services.NewTrendAnalyzer()
	analysis.TrendAnalysis = trendAnalyzer.AnalyzeTrend(analysis.Technical, analysis.VolumePrice, analysis.Quote.Price)

	// Trading signals: buy zone, T-suggestion, intraday forecast, patterns, short-term score
	atr14 := services.ComputeATR(klines, 14)
	patterns := services.DetectPatterns(klines)
	analysis.PatternAnalysis = services.AnalyzePatternSignals(patterns)

	// Look up portfolio holding for this stock
	holdingQty := 0
	holdingCost := 0.0
	if h.db != nil {
		var qty int
		var cost float64
		err := h.db.QueryRow(ctx,
			"SELECT quantity, cost_price FROM portfolio WHERE code = $1 LIMIT 1", code).Scan(&qty, &cost)
		if err == nil && qty > 0 {
			holdingQty = qty
			holdingCost = cost
			pnlPct := 0.0
			if cost > 0 {
				pnlPct = (analysis.Quote.Price - cost) / cost * 100
			}
			analysis.Holding = &models.HoldingInfo{
				Quantity:    qty,
				CostPrice:   cost,
				MarketValue: float64(qty) * analysis.Quote.Price,
				PnL:         float64(qty) * (analysis.Quote.Price - cost),
				PnLPct:      pnlPct,
			}
		}
	}

	if atr14 > 0 {
		analysis.BuyZone = services.AnalyzeBuyZone(
			analysis.TrendAnalysis.SupportResistance,
			analysis.Technical,
			analysis.Quote,
			atr14,
		)
		analysis.TSuggestion = services.AnalyzeTSuggestion(
			analysis.Quote,
			analysis.Technical,
			klines,
			atr14,
			holdingQty,
			holdingCost,
		)
		// Extract support/resistance for forecast
		support := 0.0
		resistance := 0.0
		sr := analysis.TrendAnalysis.SupportResistance
		if sr.Support1 > 0 {
			support = sr.Support1
		}
		if sr.Resistance1 > 0 {
			resistance = sr.Resistance1
		}
		analysis.IntradayForecast = services.AnalyzeIntradayForecast(
			klines,
			analysis.Quote,
			atr14,
			patterns,
			support,
			resistance,
		)
	}
	analysis.ShortTermScore = services.AnalyzeShortTermScore(
		analysis.Quote,
		analysis.Technical,
		analysis.VolumePrice,
		analysis.MoneyFlow,
		analysis.TrendAnalysis,
		analysis.PatternAnalysis,
		analysis.IntradayForecast,
	)

	// Record score history (best-effort)
	if h.scoreHistory != nil {
		dimScores := map[string]float64{
			"volume_price": scoreDimensionFromVerdict(analysis.VolumePrice.Verdict),
			"valuation":    scoreDimensionFromVerdict(analysis.Valuation.Verdict),
			"volatility":   scoreDimensionFromVerdict(analysis.Volatility.Verdict),
			"money_flow":   scoreDimensionFromVerdict(analysis.MoneyFlow.Verdict),
			"technical":    scoreDimensionFromVerdict(analysis.Technical.Verdict),
			"sector":       scoreDimensionFromVerdict(analysis.Sector.Verdict),
			"sentiment":    scoreDimensionFromVerdict(analysis.Sentiment.Verdict),
			"fundamentals": scoreDimensionFromVerdict(analysis.Fundamentals.Verdict),
			"northbound":   scoreDimensionFromVerdict(analysis.Northbound.Verdict),
			"margin":       scoreDimensionFromVerdict(analysis.MarginDetail.Verdict),
		}
		go h.scoreHistory.RecordScore(code, float64(analysis.Summary.OverallScore), dimScores)
	}

	return analysis
}

func (h *AnalyzeHandler) fetchQuote(ctx context.Context, code string) (models.Quote, error) {
	if cached, ok := h.quoteCache.Get(code); ok {
		return cached, nil
	}

	// Try real-time Tencent quote first (during trading hours this is live)
	if h.tencent != nil {
		quote, err := h.tencent.FetchQuote(ctx, code)
		if err == nil && quote.Price > 0 {
			h.quoteCache.Set(code, quote, 5*time.Second)
			return quote, nil
		}
		h.logger.Warn("tencent quote failed, falling back", zap.String("code", code), zap.Error(err))
	}

	// Fallback: TushareDB (end-of-day data)
	if h.tushareDB != nil {
		quote, err := h.tushareDB.FetchQuoteFromDB(ctx, code)
		if err == nil && quote.Price > 0 {
			h.quoteCache.Set(code, quote, 5*time.Second)
			return quote, nil
		}
		h.logger.Warn("tushare quote failed", zap.String("code", code), zap.Error(err))
	}

	return models.Quote{}, fmt.Errorf("quote not available for %s", code)
}

func (h *AnalyzeHandler) fetchKlines(ctx context.Context, code string) ([]models.KlinePoint, error) {
	return h.fetchKlinesN(ctx, code, 60)
}

// fetchKlinesN loads `days` daily klines, using the same source precedence
// as fetchKlines but bypassing the cache (since the cache key is per-code
// without a days dimension).
func (h *AnalyzeHandler) fetchKlinesN(ctx context.Context, code string, days int) ([]models.KlinePoint, error) {
	if h.tushareDB != nil {
		klines, err := h.tushareDB.FetchKline(ctx, code, days)
		if err == nil && len(klines) > 0 {
			return klines, nil
		}
		h.logger.Warn("tushare kline failed, falling back",
			zap.String("code", code), zap.Int("days", days), zap.Error(err))
	}
	if h.eastMoney != nil {
		klines, err := h.eastMoney.FetchKline(ctx, code, days)
		if err == nil && len(klines) > 0 {
			return klines, nil
		}
		h.logger.Warn("eastmoney kline failed",
			zap.String("code", code), zap.Int("days", days), zap.Error(err))
	}
	return nil, fmt.Errorf("klines not available for %s", code)
}

func (h *AnalyzeHandler) fetchFlow(ctx context.Context, code string) ([]models.MoneyFlowDay, error) {
	if cached, ok := h.flowCache.Get(code); ok {
		return cached, nil
	}

	// Try TushareDB first (primary local data source)
	if h.tushareDB != nil {
		flows, err := h.tushareDB.FetchMoneyFlow(ctx, code, 10)
		if err == nil && len(flows) > 0 {
			h.flowCache.Set(code, flows, 60*time.Second)
			return flows, nil
		}
		h.logger.Warn("tushare moneyflow failed", zap.String("code", code), zap.Error(err))
	}

	return nil, fmt.Errorf("money flow not available from TushareDB for %s", code)
}

func (h *AnalyzeHandler) fetchSectors(ctx context.Context, code string) ([]models.StockSector, error) {
	if cached, ok := h.sectorsCache.Get(code); ok {
		return cached, nil
	}

	// Try TushareDB first (industry from stock_basic)
	if h.tushareDB != nil {
		industry, err := h.tushareDB.FetchIndustryFromDB(ctx, code)
		if err == nil && industry != "" {
			sectors := []models.StockSector{{Name: industry}}
			h.sectorsCache.Set(code, sectors, 600*time.Second)
			return sectors, nil
		}
		h.logger.Warn("tushare industry failed", zap.String("code", code), zap.Error(err))
	}

	return nil, fmt.Errorf("sectors not available from TushareDB for %s", code)
}

func (h *AnalyzeHandler) fetchNews(ctx context.Context, code string) ([]models.NewsItem, error) {
	return h.fetchNewsWithMode(ctx, code, false)
}

func (h *AnalyzeHandler) fetchAnnouncements(ctx context.Context, code string) ([]models.Announcement, error) {
	return h.fetchAnnouncementsWithMode(ctx, code, false)
}

// fetchNewsWithMode fetches news. If fast=true, skips external API fallback (DB only).
func (h *AnalyzeHandler) fetchNewsWithMode(ctx context.Context, code string, fast bool) ([]models.NewsItem, error) {
	if cached, ok := h.newsCache.Get(code); ok {
		return cached, nil
	}
	// Always use DB-only mode since news is pre-synced
	news, err := h.newsSvc.GetStockNewsDBOnly(ctx, code, 10)
	if err != nil {
		return nil, err
	}
	h.newsCache.Set(code, news, 300*time.Second)
	return news, nil
}

// fetchAnnouncementsWithMode fetches announcements. If fast=true, skips external API fallback (DB only).
func (h *AnalyzeHandler) fetchAnnouncementsWithMode(ctx context.Context, code string, fast bool) ([]models.Announcement, error) {
	if cached, ok := h.announcementsCache.Get(code); ok {
		return cached, nil
	}
	// Always use DB-only mode since announcements are pre-synced
	anns, err := h.newsSvc.GetStockAnnouncementsDBOnly(ctx, code, 10)
	if err != nil {
		return nil, err
	}
	h.announcementsCache.Set(code, anns, 300*time.Second)
	return anns, nil
}

// ---- StockInfo endpoint ----

// StockInfoResponse is the response for GET /stockinfo.
type StockInfoResponse struct {
	Code          string                `json:"code"`
	Name          string                `json:"name,omitempty"`
	Quote         *models.Quote         `json:"quote,omitempty"`
	Flow          []models.MoneyFlowDay `json:"flow,omitempty"`
	News          []models.NewsItem     `json:"news,omitempty"`
	Announcements []models.Announcement `json:"announcements,omitempty"`
	Sectors       []models.StockSector  `json:"sectors,omitempty"`
	Cached        bool                  `json:"cached"`
	CacheDetail   map[string]bool       `json:"cache_detail"`
	Errors        map[string]string     `json:"errors,omitempty"`
}

// @Summary      个股详情
// @Description  获取个股综合信息(行情/资金流向/新闻/公告/板块)
// @Tags         analyze
// @Accept       json
// @Produce      json
// @Param        code  query      string  true  "股票代码"
// @Success      200  {object}  StockInfoResponse
// @Router       /stockinfo [get]
func (h *AnalyzeHandler) StockInfo(c *gin.Context) {
	codeParam := strings.TrimSpace(c.Query("code"))
	if codeParam == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}

	h.logger.Info("stock info request",
		zap.String("code", codeParam),
	)

	if err := services.ValidateStockCode(codeParam); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_CODE_FORMAT", err.Error())
		return
	}

	code := services.NormalizeCode(codeParam)
	ctx := c.Request.Context()
	errs := make(map[string]string)
	cacheDetail := make(map[string]bool)

	// Fetch all concurrently
	var (
		quote         *models.Quote
		flows         []models.MoneyFlowDay
		news          []models.NewsItem
		announcements []models.Announcement
		sectors       []models.StockSector
		quoteName     string
	)

	var wg sync.WaitGroup

	// Quote
	wg.Add(1)
	go func() {
		defer wg.Done()
		q, err := h.fetchQuote(ctx, code)
		if err != nil {
			errs["quote"] = err.Error()
			cacheDetail["quote"] = false
			return
		}
		quote = &q
		quoteName = q.Name
		cacheDetail["quote"] = true // if we got here without error, it was fetched (possibly from cache)
	}()

	// Flow (5 days)
	wg.Add(1)
	go func() {
		defer wg.Done()
		f, err := h.fetchFlow(ctx, code)
		if err != nil {
			errs["flow"] = err.Error()
			cacheDetail["flow"] = false
			return
		}
		flows = f
		cacheDetail["flow"] = true
	}()

	// News (10 items)
	wg.Add(1)
	go func() {
		defer wg.Done()
		n, err := h.fetchNews(ctx, code)
		if err != nil {
			errs["news"] = err.Error()
			cacheDetail["news"] = false
			return
		}
		news = n
		cacheDetail["news"] = true
	}()

	// Announcements (10 items)
	wg.Add(1)
	go func() {
		defer wg.Done()
		a, err := h.fetchAnnouncements(ctx, code)
		if err != nil {
			errs["announcements"] = err.Error()
			cacheDetail["announcements"] = false
			return
		}
		announcements = a
		cacheDetail["announcements"] = true
	}()

	// Sectors
	wg.Add(1)
	go func() {
		defer wg.Done()
		s, err := h.fetchSectors(ctx, code)
		if err != nil {
			errs["sectors"] = err.Error()
			cacheDetail["sectors"] = false
			return
		}
		sectors = s
		cacheDetail["sectors"] = true
	}()

	wg.Wait()

	allCached := true
	for _, v := range cacheDetail {
		if !v {
			allCached = false
			break
		}
	}

	resp := StockInfoResponse{
		Code:          services.StockCode6(code),
		Name:          quoteName,
		Quote:         quote,
		Flow:          flows,
		News:          news,
		Announcements: announcements,
		Sectors:       sectors,
		Cached:        allCached,
		CacheDetail:   cacheDetail,
	}
	if len(errs) > 0 {
		resp.Errors = errs
	}

	c.JSON(http.StatusOK, resp)
}

// scoreDimensionFromVerdict converts a verdict string to a 0-100 score for history recording.
func scoreDimensionFromVerdict(verdict string) float64 {
	if verdict == "" {
		return 50
	}
	switch {
	case contains(verdict, "强势") || contains(verdict, "优秀") || contains(verdict, "强烈"):
		return 85
	case contains(verdict, "偏多") || contains(verdict, "良好") || contains(verdict, "积极"):
		return 70
	case contains(verdict, "多头") || contains(verdict, "金叉"):
		return 75
	case contains(verdict, "占优") || contains(verdict, "流入") || contains(verdict, "较强"):
		return 65
	case contains(verdict, "上方") || contains(verdict, "上升"):
		return 60
	case contains(verdict, "略强") || contains(verdict, "偏强"):
		return 55
	case contains(verdict, "中性") || contains(verdict, "均衡") || contains(verdict, "正常") || contains(verdict, "持平"):
		return 50
	case contains(verdict, "不足") || contains(verdict, "偏弱") || contains(verdict, "谨慎"):
		return 40
	case contains(verdict, "偏空") || contains(verdict, "较低"):
		return 35
	case contains(verdict, "弱势") || contains(verdict, "危险"):
		return 20
	case contains(verdict, "空头") || contains(verdict, "死叉"):
		return 25
	case contains(verdict, "超买"):
		return 30
	case contains(verdict, "超卖"):
		return 70
	default:
		return 50
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// IntradayForecastAccuracyResponse wraps the backtest result with metadata.
type IntradayForecastAccuracyResponse struct {
	OK       bool                              `json:"ok"`
	Code     string                            `json:"code"`
	Days     int                               `json:"days"`
	Accuracy *services.IntradayForecastAccuracy `json:"accuracy"`
}

// IntradayForecastAccuracy runs a historical accuracy backtest for the
// intraday high/low prediction on the given stock.
//
// @Summary      日内预测命中率回测
// @Description  对最近 N 个交易日逐日回放 AnalyzeIntradayForecast，对比预测区间和实际高低点，输出命中率/偏差/可靠性等级
// @Tags         analyze
// @Produce      json
// @Param        code  query   string  true   "股票代码"
// @Param        days  query   int     false  "回测交易日数（默认 120，最大 250）"  default(120)
// @Success      200   {object}  IntradayForecastAccuracyResponse
// @Router       /analyze/intraday-forecast-accuracy [get]
func (h *AnalyzeHandler) IntradayForecastAccuracy(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing code"})
		return
	}

	days := 120
	if d := strings.TrimSpace(c.Query("days")); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}
	if days > 500 {
		days = 500
	}

	// Fetch with headroom: BacktestIntradayForecast needs `days` evaluable days
	// plus a 30-bar warmup, but data sources often return fewer rows than
	// requested (new listings, suspensions, DB gaps). Asking for `days + 60`
	// gives the backtest room to clamp down to whatever's actually available
	// instead of failing the whole request.
	klines, err := h.fetchKlinesN(c.Request.Context(), code, days+60)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": err.Error()})
		return
	}

	acc := services.BacktestIntradayForecast(klines, days)
	if acc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ok":    false,
			"error": "insufficient history for backtest (need > 30 bars)",
		})
		return
	}

	// Optional from/to date filter (YYYY-MM-DD). Both inclusive. Filters the
	// returned `details` array without changing the aggregate metrics (which
	// are computed over the full window). When both bounds are given the
	// frontend table view can show just the requested slice.
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))
	if from != "" || to != "" {
		acc.Details = filterDetailsByDate(acc.Details, from, to)
	}

	c.JSON(http.StatusOK, IntradayForecastAccuracyResponse{
		OK:       true,
		Code:     code,
		Days:     days,
		Accuracy: acc,
	})
}

// filterDetailsByDate returns a copy of details containing only entries with
// Date in [from, to] (inclusive). Empty bounds are treated as unbounded.
// Date comparison is lexicographic on the YYYY-MM-DD format, which sorts
// identically to chronological order — no parsing needed.
func filterDetailsByDate(details []services.IntradayForecastDay, from, to string) []services.IntradayForecastDay {
	if len(details) == 0 && from == "" && to == "" {
		return details
	}
	out := make([]services.IntradayForecastDay, 0, len(details))
	for _, d := range details {
		if from != "" && d.Date < from {
			continue
		}
		if to != "" && d.Date > to {
			continue
		}
		out = append(out, d)
	}
	return out
}
