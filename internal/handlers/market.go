package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
)

type MarketHandler struct {
	eastMoney     *services.EastMoneyService
	tencent       *services.TencentService
	quoteCache    *cache.Cache[models.Quote]
	klineCache    *cache.Cache[[]models.KlinePoint]
	sectorsCache  *cache.Cache[[]models.Sector]
	overviewCache *cache.Cache[models.MarketOverview]
	newsCache     *cache.Cache[[]models.NewsItem]
}

func NewMarketHandler(eastMoney *services.EastMoneyService, tencent *services.TencentService) *MarketHandler {
	return &MarketHandler{
		eastMoney:     eastMoney,
		tencent:       tencent,
		quoteCache:    cache.New[models.Quote](),
		klineCache:    cache.New[[]models.KlinePoint](),
		sectorsCache:  cache.New[[]models.Sector](),
		overviewCache: cache.New[models.MarketOverview](),
		newsCache:     cache.New[[]models.NewsItem](),
	}
}

func (h *MarketHandler) Quote(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}

	if cached, ok := h.quoteCache.Get(code); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	quote, err := h.tencent.FetchQuote(c.Request.Context(), code)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "QUOTE_FETCH_FAILED", "failed to fetch quote")
		return
	}
	h.quoteCache.Set(code, quote, 3*time.Second)

	c.JSON(http.StatusOK, quote)
}

func (h *MarketHandler) Kline(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}

	days := 60
	if rawDays := strings.TrimSpace(c.Query("days")); rawDays != "" {
		parsedDays, err := strconv.Atoi(rawDays)
		if err != nil || parsedDays < 1 {
			writeError(c, http.StatusBadRequest, "INVALID_DAYS", "days must be a positive integer")
			return
		}
		days = parsedDays
	}

	cacheKey := fmt.Sprintf("%s:%d", code, days)
	if cached, ok := h.klineCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	points, err := h.eastMoney.FetchKline(c.Request.Context(), code, days)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "KLINE_FETCH_FAILED", "failed to fetch kline data")
		return
	}
	h.klineCache.Set(cacheKey, points, 30*time.Second)

	c.JSON(http.StatusOK, points)
}

func (h *MarketHandler) Sectors(c *gin.Context) {
	if cached, ok := h.sectorsCache.Get("sectors"); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	sectors, err := h.eastMoney.FetchSectors(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "SECTORS_FETCH_FAILED", "failed to fetch sector data")
		return
	}
	h.sectorsCache.Set("sectors", sectors, 30*time.Second)

	c.JSON(http.StatusOK, sectors)
}

func (h *MarketHandler) Overview(c *gin.Context) {
	if cached, ok := h.overviewCache.Get("overview"); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	overview, err := h.eastMoney.FetchOverview(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "OVERVIEW_FETCH_FAILED", "failed to fetch market overview")
		return
	}
	h.overviewCache.Set("overview", overview, 10*time.Second)

	c.JSON(http.StatusOK, overview)
}

func (h *MarketHandler) News(c *gin.Context) {
	limit := 20
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 1 {
			writeError(c, http.StatusBadRequest, "INVALID_LIMIT", "limit must be a positive integer")
			return
		}
		limit = parsedLimit
	}

	cacheKey := fmt.Sprintf("news:%d", limit)
	if cached, ok := h.newsCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	items, err := h.eastMoney.FetchNews(c.Request.Context(), limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "NEWS_FETCH_FAILED", "failed to fetch news")
		return
	}
	h.newsCache.Set(cacheKey, items, 60*time.Second)

	c.JSON(http.StatusOK, items)
}

func (h *MarketHandler) CacheStats() map[string]cache.Sizer {
	return map[string]cache.Sizer{
		"quote":    h.quoteCache,
		"kline":    h.klineCache,
		"sectors":  h.sectorsCache,
		"overview": h.overviewCache,
		"news":     h.newsCache,
	}
}
