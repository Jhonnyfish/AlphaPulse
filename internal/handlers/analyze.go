package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
)

type AnalyzeHandler struct {
	eastMoney          *services.EastMoneyService
	tencent            *services.TencentService
	newsSvc            *services.NewsService
	tushareDB          *services.TushareDB          // Primary data source, may be nil
	logger             *zap.Logger
	scoreHistory       *ScoreHistoryHandler
	quoteCache         *cache.Cache[models.Quote]
	klineCache       *cache.Cache[[]models.KlinePoint]
	flowCache        *cache.Cache[[]models.MoneyFlowDay]
	sectorsCache     *cache.Cache[[]models.StockSector]
	newsCache        *cache.Cache[[]models.NewsItem]
	announcementsCache *cache.Cache[[]models.Announcement]
}

func NewAnalyzeHandler(eastMoney *services.EastMoneyService, tencent *services.TencentService, newsSvc *services.NewsService, logger *zap.Logger) *AnalyzeHandler {
	return &AnalyzeHandler{
		eastMoney:          eastMoney,
		tencent:            tencent,
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
		quote    models.Quote
		klines   []models.KlinePoint
		flows    []models.MoneyFlowDay
		sectors  []models.StockSector
		news     []models.NewsItem
		anns     []models.Announcement
		fins     []services.FinancialData
		hsgt     []services.HsgtData
		top10    []services.HsgtTop10Data
		marginD  []services.MarginDetailData
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
	if h.tushareDB != nil {
		fins, _ = h.tushareDB.FetchFinancials(ctx, code, 8)
		top10, _ = h.tushareDB.FetchHsgtTop10ByCode(ctx, code, 10)
		marginD, _ = h.tushareDB.FetchMarginDetailHistory(ctx, code, 10)
		hsgt, _ = h.tushareDB.FetchHsgtHistory(ctx, 10)
		if sp, err := h.tushareDB.FetchSectorPerformance(ctx, code); err == nil {
			sectorPerf = &sp
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
		Code:    services.StockCode6(code),
		Name:    quote.Name,
		Version: "3.0",
		Quote:   quote,
		OrderFlow:   services.AnalyzeOrderFlow(quote),
		VolumePrice: services.AnalyzeVolumePrice(quote, klines),
		Valuation:   services.AnalyzeValuation(quote),
		Volatility:  services.AnalyzeVolatility(quote),
		MoneyFlow:   services.AnalyzeMoneyFlow(flows),
		Technical:   services.AnalyzeTechnical(klines),
		Sector:      services.AnalyzeSector(quote, sectorNames, sectorPerf),
		Sentiment:   services.AnalyzeSentiment(news, anns),
		Fundamentals: services.AnalyzeFundamentals(fins),
		Northbound:  services.AnalyzeNorthbound(hsgt, top10),
		MarginDetail: services.AnalyzeMarginDetail(marginD),
		DataSources: map[string]string{
			"quote":         "tushare",
			"klines":        "tushare",
			"money_flow":    "tushare",
			"sector":        "tushare",
			"sentiment":     "db/eastmoney",
			"fundamentals":  "tushare/fina_indicator",
			"northbound":    "tushare/hsgt",
			"margin":        "tushare/margin_detail",
		},
		Errors:    errs,
		FetchedAt: time.Now(),
	}

	analysis.Summary = services.BuildSummary(&analysis)

	// Run trend analysis
	trendAnalyzer := services.NewTrendAnalyzer()
	analysis.TrendAnalysis = trendAnalyzer.AnalyzeTrend(analysis.Technical, analysis.VolumePrice, analysis.Quote.Price)

	// Record score history (best-effort)
	if h.scoreHistory != nil {
		dimScores := map[string]float64{
			"order_flow":   scoreDimensionFromVerdict(analysis.OrderFlow.Verdict),
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

	// Try TushareDB first (primary local data source)
	if h.tushareDB != nil {
		quote, err := h.tushareDB.FetchQuoteFromDB(ctx, code)
		if err == nil && quote.Price > 0 {
			h.quoteCache.Set(code, quote, 5*time.Second)
			return quote, nil
		}
		h.logger.Warn("tushare quote failed", zap.String("code", code), zap.Error(err))
	}

	return models.Quote{}, fmt.Errorf("quote not available from TushareDB for %s", code)
}

func (h *AnalyzeHandler) fetchKlines(ctx context.Context, code string) ([]models.KlinePoint, error) {
	if cached, ok := h.klineCache.Get(code); ok {
		return cached, nil
	}

	// Try TushareDB first (primary local data source)
	if h.tushareDB != nil {
		klines, err := h.tushareDB.FetchKline(ctx, code, 60)
		if err == nil && len(klines) > 0 {
			h.klineCache.Set(code, klines, 60*time.Second)
			return klines, nil
		}
		h.logger.Warn("tushare kline failed", zap.String("code", code), zap.Error(err))
	}

	return nil, fmt.Errorf("klines not available from TushareDB for %s", code)
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
