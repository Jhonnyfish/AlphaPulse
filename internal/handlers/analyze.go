package handlers

import (
	"context"
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
	logger             *zap.Logger
	quoteCache         *cache.Cache[models.Quote]
	klineCache       *cache.Cache[[]models.KlinePoint]
	flowCache        *cache.Cache[[]models.MoneyFlowDay]
	sectorsCache     *cache.Cache[[]models.StockSector]
	newsCache        *cache.Cache[[]models.NewsItem]
	announcementsCache *cache.Cache[[]models.Announcement]
}

func NewAnalyzeHandler(eastMoney *services.EastMoneyService, tencent *services.TencentService, logger *zap.Logger) *AnalyzeHandler {
	return &AnalyzeHandler{
		eastMoney:          eastMoney,
		tencent:            tencent,
		logger:             logger,
		quoteCache:         cache.New[models.Quote](),
		klineCache:         cache.New[[]models.KlinePoint](),
		flowCache:          cache.New[[]models.MoneyFlowDay](),
		sectorsCache:       cache.New[[]models.StockSector](),
		newsCache:          cache.New[[]models.NewsItem](),
		announcementsCache: cache.New[[]models.Announcement](),
	}
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
	code = services.NormalizeCode(code)
	errs := make(map[string]string)

	h.logger.Info("analyzing single stock", zap.String("code", code))

	// Fetch quote
	quote, quoteErr := h.fetchQuote(ctx, code)
	if quoteErr != nil {
		errs["quote"] = quoteErr.Error()
	}

	// Fetch klines
	klines, klineErr := h.fetchKlines(ctx, code)
	if klineErr != nil {
		errs["klines"] = klineErr.Error()
	}

	// Fetch money flow
	flows, flowErr := h.fetchFlow(ctx, code)
	if flowErr != nil {
		errs["money_flow"] = flowErr.Error()
	}

	// Fetch sectors
	sectors, sectorErr := h.fetchSectors(ctx, code)
	if sectorErr != nil {
		errs["sectors"] = sectorErr.Error()
	}
	sectorNames := make([]string, 0, len(sectors))
	for _, s := range sectors {
		sectorNames = append(sectorNames, s.Name)
	}

	// Fetch news
	news, newsErr := h.fetchNews(ctx, code)
	if newsErr != nil {
		errs["news"] = newsErr.Error()
	}

	// Fetch announcements
	anns, annErr := h.fetchAnnouncements(ctx, code)
	if annErr != nil {
		errs["announcements"] = annErr.Error()
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
		Sector:      services.AnalyzeSector(quote, sectorNames),
		Sentiment:   services.AnalyzeSentiment(news, anns),
		DataSources: map[string]string{
			"quote":         "tencent",
			"klines":        "eastmoney",
			"money_flow":    "eastmoney",
			"sector":        "eastmoney",
			"sentiment":     "eastmoney",
		},
		Errors:    errs,
		FetchedAt: time.Now(),
	}

	analysis.Summary = services.BuildSummary(&analysis)
	return analysis
}

func (h *AnalyzeHandler) fetchQuote(ctx context.Context, code string) (models.Quote, error) {
	if cached, ok := h.quoteCache.Get(code); ok {
		return cached, nil
	}
	quote, err := h.tencent.FetchQuote(ctx, code)
	if err != nil {
		return models.Quote{}, err
	}
	h.quoteCache.Set(code, quote, 5*time.Second)
	return quote, nil
}

func (h *AnalyzeHandler) fetchKlines(ctx context.Context, code string) ([]models.KlinePoint, error) {
	if cached, ok := h.klineCache.Get(code); ok {
		return cached, nil
	}
	klines, err := h.eastMoney.FetchKline(ctx, code, 60)
	if err != nil {
		return nil, err
	}
	h.klineCache.Set(code, klines, 60*time.Second)
	return klines, nil
}

func (h *AnalyzeHandler) fetchFlow(ctx context.Context, code string) ([]models.MoneyFlowDay, error) {
	if cached, ok := h.flowCache.Get(code); ok {
		return cached, nil
	}
	flows, err := h.eastMoney.FetchMoneyFlow(ctx, code, 5)
	if err != nil {
		return nil, err
	}
	h.flowCache.Set(code, flows, 300*time.Second)
	return flows, nil
}

func (h *AnalyzeHandler) fetchSectors(ctx context.Context, code string) ([]models.StockSector, error) {
	if cached, ok := h.sectorsCache.Get(code); ok {
		return cached, nil
	}
	sectors, err := h.eastMoney.FetchStockSectors(ctx, code)
	if err != nil {
		return nil, err
	}
	h.sectorsCache.Set(code, sectors, 600*time.Second)
	return sectors, nil
}

func (h *AnalyzeHandler) fetchNews(ctx context.Context, code string) ([]models.NewsItem, error) {
	if cached, ok := h.newsCache.Get(code); ok {
		return cached, nil
	}
	news, err := h.eastMoney.FetchStockNews(ctx, code, 10)
	if err != nil {
		return nil, err
	}
	h.newsCache.Set(code, news, 300*time.Second)
	return news, nil
}

func (h *AnalyzeHandler) fetchAnnouncements(ctx context.Context, code string) ([]models.Announcement, error) {
	if cached, ok := h.announcementsCache.Get(code); ok {
		return cached, nil
	}
	anns, err := h.eastMoney.FetchStockAnnouncements(ctx, code, 10)
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
