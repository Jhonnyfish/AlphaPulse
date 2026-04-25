package handlers

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
)

type AnalyzeHandler struct {
	eastMoney        *services.EastMoneyService
	tencent          *services.TencentService
	quoteCache       *cache.Cache[models.Quote]
	klineCache       *cache.Cache[[]models.KlinePoint]
	flowCache        *cache.Cache[[]models.MoneyFlowDay]
	sectorsCache     *cache.Cache[[]models.StockSector]
	newsCache        *cache.Cache[[]models.NewsItem]
	announcementsCache *cache.Cache[[]models.Announcement]
}

func NewAnalyzeHandler(eastMoney *services.EastMoneyService, tencent *services.TencentService) *AnalyzeHandler {
	return &AnalyzeHandler{
		eastMoney:          eastMoney,
		tencent:            tencent,
		quoteCache:         cache.New[models.Quote](),
		klineCache:         cache.New[[]models.KlinePoint](),
		flowCache:          cache.New[[]models.MoneyFlowDay](),
		sectorsCache:       cache.New[[]models.StockSector](),
		newsCache:          cache.New[[]models.NewsItem](),
		announcementsCache: cache.New[[]models.Announcement](),
	}
}

func (h *AnalyzeHandler) Analyze(c *gin.Context) {
	codeParam := strings.TrimSpace(c.Query("code"))
	if codeParam == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}

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
