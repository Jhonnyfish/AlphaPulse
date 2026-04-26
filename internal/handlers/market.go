package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
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

type MarketHandler struct {
	eastMoney           *services.EastMoneyService
	tencent             *services.TencentService
	db                  *pgxpool.Pool
	quoteCache          *cache.Cache[models.Quote]
	klineCache          *cache.Cache[[]models.KlinePoint]
	sectorsCache        *cache.Cache[[]models.Sector]
	overviewCache       *cache.Cache[models.MarketOverview]
	newsCache           *cache.Cache[[]models.NewsItem]
	announcementsCache  *cache.Cache[[]models.Announcement]
	searchCache         *cache.Cache[[]models.SearchSuggestion]
	topMoversCache      *cache.Cache[[]models.TopMover]
	trendsCache         *cache.Cache[models.MarketTrends]
	marketOverviewCache *cache.Cache[models.MarketOverviewResponse]
	hotConceptsCache    *cache.Cache[[]models.HotConcept]
	conceptStocksCache  *cache.Cache[[]conceptStockItem]
	breadthCache        *cache.Cache[models.MarketBreadthDetail]
	sentimentCache      *cache.Cache[models.MarketSentimentResponse]
}

func NewMarketHandler(eastMoney *services.EastMoneyService, tencent *services.TencentService, db *pgxpool.Pool) *MarketHandler {
	return &MarketHandler{
		eastMoney:           eastMoney,
		tencent:             tencent,
		db:                  db,
		quoteCache:          cache.New[models.Quote](),
		klineCache:          cache.New[[]models.KlinePoint](),
		sectorsCache:        cache.New[[]models.Sector](),
		overviewCache:       cache.New[models.MarketOverview](),
		newsCache:           cache.New[[]models.NewsItem](),
		announcementsCache:  cache.New[[]models.Announcement](),
		searchCache:         cache.New[[]models.SearchSuggestion](),
		topMoversCache:      cache.New[[]models.TopMover](),
		trendsCache:         cache.New[models.MarketTrends](),
		marketOverviewCache: cache.New[models.MarketOverviewResponse](),
		hotConceptsCache:    cache.New[[]models.HotConcept](),
		conceptStocksCache:  cache.New[[]conceptStockItem](),
		breadthCache:        cache.New[models.MarketBreadthDetail](),
		sentimentCache:      cache.New[models.MarketSentimentResponse](),
	}
}

type conceptStockItem struct {
	models.SectorMember
	InWatchlist bool `json:"in_watchlist"`
}

type conceptOverlapItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// @Summary      获取股票实时行情
// @Description  根据股票代码获取实时行情数据
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Param        code query string true "股票代码"
// @Success      200 {object} models.Quote
// @Failure      400 {object} map[string]interface{}
// @Router       /api/market/quote [get]
func (h *MarketHandler) Quote(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}
	if err := services.ValidateStockCode(code); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_CODE_FORMAT", err.Error())
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

// @Summary      获取K线数据
// @Description  根据股票代码获取日K线历史数据
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Param        code query string true "股票代码"
// @Param        days query int false "天数" default(60)
// @Success      200 {array} models.KlinePoint
// @Failure      400 {object} map[string]interface{}
// @Router       /api/market/kline [get]
func (h *MarketHandler) Kline(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}
	if err := services.ValidateStockCode(code); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_CODE_FORMAT", err.Error())
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

// @Summary      获取板块数据
// @Description  获取所有板块行情及涨跌排行
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.Sector
// @Router       /api/market/sectors [get]
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

// @Summary      市场概览（旧版）
// @Description  获取市场基础概览数据
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} models.MarketOverview
// @Router       /api/market/overview [get]
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

// @Summary      获取市场新闻
// @Description  获取最新的市场新闻资讯
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "返回条数" default(20)
// @Success      200 {array} models.NewsItem
// @Router       /api/market/news [get]
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

// @Summary      获取个股公告
// @Description  根据股票代码获取相关公告列表
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Param        code query string true "股票代码"
// @Param        limit query int false "返回条数" default(10)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /api/announcements [get]
func (h *MarketHandler) Announcements(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "code is required")
		return
	}
	if err := services.ValidateStockCode(code); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_CODE_FORMAT", err.Error())
		return
	}

	limit := 10
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 1 || parsedLimit > 50 {
			writeError(c, http.StatusBadRequest, "INVALID_LIMIT", "limit must be between 1 and 50")
			return
		}
		limit = parsedLimit
	}

	cacheKey := fmt.Sprintf("announcements:%s:%d", code, limit)
	if cached, ok := h.announcementsCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{
			"code":   services.StockCode6(code),
			"items":  cached,
			"source": "eastmoney",
			"cached": true,
		})
		return
	}

	items, err := h.eastMoney.FetchStockAnnouncements(c.Request.Context(), code, limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "ANNOUNCEMENTS_FETCH_FAILED", "failed to fetch announcements")
		return
	}

	h.announcementsCache.Set(cacheKey, items, 5*time.Minute)
	c.JSON(http.StatusOK, gin.H{
		"code":   services.StockCode6(code),
		"items":  items,
		"source": "eastmoney",
		"cached": false,
	})
}

// @Summary      搜索股票
// @Description  根据关键词搜索股票，返回匹配的股票列表
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Param        q query string true "搜索关键词"
// @Success      200 {array} models.SearchSuggestion
// @Failure      400 {object} map[string]interface{}
// @Router       /api/market/search [get]
func (h *MarketHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		writeError(c, http.StatusBadRequest, "INVALID_QUERY", "q is required")
		return
	}
	if len(query) < 1 {
		writeError(c, http.StatusBadRequest, "INVALID_QUERY", "query too short")
		return
	}

	cacheKey := "search:" + query
	if cached, ok := h.searchCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	suggestions, err := h.eastMoney.SearchStocks(c.Request.Context(), query, 10)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "SEARCH_FAILED", "failed to search stocks")
		return
	}
	h.searchCache.Set(cacheKey, suggestions, 5*time.Minute)

	c.JSON(http.StatusOK, suggestions)
}

// @Summary      获取涨跌幅排行
// @Description  获取当日涨幅或跌幅前N只股票
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Param        sort query string false "排序方式: desc=涨幅 asc=跌幅" default(desc)
// @Param        limit query int false "返回条数" default(20)
// @Success      200 {array} models.TopMover
// @Router       /api/market/top-movers [get]
func (h *MarketHandler) TopMovers(c *gin.Context) {
	sortOrder := strings.TrimSpace(c.DefaultQuery("sort", "desc")) // desc=gainers, asc=losers
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	limit := 20
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 1 {
			writeError(c, http.StatusBadRequest, "INVALID_LIMIT", "limit must be a positive integer")
			return
		}
		limit = parsedLimit
	}

	cacheKey := fmt.Sprintf("topmovers:%s:%d", sortOrder, limit)
	if cached, ok := h.topMoversCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	movers, err := h.eastMoney.FetchTopMovers(c.Request.Context(), sortOrder, limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "TOP_MOVERS_FETCH_FAILED", "failed to fetch top movers")
		return
	}
	h.topMoversCache.Set(cacheKey, movers, 30*time.Second)

	c.JSON(http.StatusOK, movers)
}

// ==================== Market Session ====================

// chinaTZ returns the China/CST timezone (UTC+8).
func chinaTZ() *time.Location {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return loc
}

// Session returns the current market session status.
// @Summary      获取市场交易时段
// @Description  获取当前市场交易时段状态，包括盘前、交易中、午休、已收盘等
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /api/market/session [get]
// Pure time-based logic — no external API calls needed.
func (h *MarketHandler) Session(c *gin.Context) {
	now := time.Now().In(chinaTZ())
	weekday := now.Weekday() // 0=Sun, 6=Sat
	minutes := now.Hour()*60 + now.Minute()

	var session, sessionEN, nextInfo string
	var refreshInterval int

	switch {
	case weekday == time.Saturday || weekday == time.Sunday:
		session = "休市"
		sessionEN = "closed"
		refreshInterval = 600
		if weekday == time.Saturday {
			nextInfo = "周一 09:30 开盘"
		} else {
			nextInfo = "明日 09:30 开盘"
		}
	case minutes < 9*60+25:
		session = "盘前"
		sessionEN = "pre_market"
		refreshInterval = 120
		nextInfo = "09:25 集合竞价"
	case minutes < 9*60+30:
		session = "集合竞价"
		sessionEN = "call_auction"
		refreshInterval = 10
		nextInfo = "09:30 开盘"
	case minutes <= 11*60+30:
		session = "交易中"
		sessionEN = "trading"
		refreshInterval = 30
		nextInfo = "11:30 午间休市"
	case minutes < 13*60:
		session = "午间休市"
		sessionEN = "lunch_break"
		refreshInterval = 120
		nextInfo = "13:00 下午开盘"
	case minutes <= 15*60:
		session = "交易中"
		sessionEN = "trading"
		refreshInterval = 30
		nextInfo = "15:00 收盘"
	default:
		session = "已收盘"
		sessionEN = "closed"
		refreshInterval = 600
		if weekday == time.Friday {
			nextInfo = "周一 09:30 开盘"
		} else {
			nextInfo = "明日 09:30 开盘"
		}
	}

	isTrading := sessionEN == "trading" || sessionEN == "call_auction"

	result := models.MarketSession{
		Session:         session,
		SessionEN:       sessionEN,
		IsTrading:       isTrading,
		RefreshInterval: refreshInterval,
		NextSession:     nextInfo,
		ServerTime:      now.Format(time.RFC3339),
		Weekday:         int(weekday),
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":   true,
		"data": result,
	})
}

// ==================== Market Trends ====================

var trendIndexCodes = []string{"sh000001", "sz399001", "sz399006"}
var trendIndexNames = map[string]string{
	"sh000001": "上证指数",
	"sz399001": "深证成指",
	"sz399006": "创业板指",
}

const trendKlineLimit = 35 // 30 trading days + safety margin

func calcKlineChange(klines []models.KlinePoint, days int) *float64 {
	if len(klines) < days+1 {
		return nil
	}
	current := klines[len(klines)-1].Close
	base := klines[len(klines)-1-days].Close
	if base == 0 {
		return nil
	}
	v := round2((current - base) / base * 100)
	return &v
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100 // banker's rounding simplified
}

func round2Neg(v float64) float64 {
	// handle negative correctly
	return float64(int(v*100+0.5)) / 100
}

// @Summary      多周期趋势对比
// @Description  获取指数和自选股的多周期涨跌幅对比及组合收益曲线
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} models.MarketTrends
// @Router       /api/market/trends [get]
func (h *MarketHandler) Trends(c *gin.Context) {
	if cached, ok := h.trendsCache.Get("trends:all"); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	ctx := c.Request.Context()

	// Load watchlist codes from DB
	wlCodes := h.loadWatchlistCodes(ctx)

	type result struct {
		index  []models.TrendStock
		stocks []models.TrendStock
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	var indices []models.TrendStock
	var watchlistStocks []models.TrendStock

	// Fetch index klines
	for _, code := range trendIndexCodes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			ts := h.fetchTrendStock(ctx, code, trendIndexNames[code])
			mu.Lock()
			indices = append(indices, ts)
			mu.Unlock()
		}(code)
	}

	// Fetch watchlist stock klines
	for _, code := range wlCodes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			ts := h.fetchTrendStock(ctx, code, "")
			mu.Lock()
			watchlistStocks = append(watchlistStocks, ts)
			mu.Unlock()
		}(code)
	}

	wg.Wait()

	// Sort indices in defined order
	idxOrder := make(map[string]int)
	for i, c := range trendIndexCodes {
		idxOrder[c] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return idxOrder[indices[i].Code] < idxOrder[indices[j].Code]
	})

	// Compute equal-weighted portfolio returns
	portfolio := models.TrendPortfolio{
		DailyReturns: []models.DailyReturn{},
	}
	for _, period := range []int{1, 5, 20, 30} {
		val := portfolioChange(watchlistStocks, period)
		switch period {
		case 1:
			portfolio.Change1D = val
		case 5:
			portfolio.Change5D = val
		case 20:
			portfolio.Change20D = val
		case 30:
			portfolio.Change30D = val
		}
	}

	// Compute daily portfolio vs benchmark cumulative returns for chart
	if len(indices) > 0 {
		benchIdx := indices[0] // sh000001
		if len(benchIdx.KlineData) > 0 {
			benchBase := benchIdx.KlineData[0].Close
			if benchBase > 0 {
				// Build per-stock daily close lookup by date
				stockDailyMap := make(map[string]map[string]float64)
				for _, s := range watchlistStocks {
					m := make(map[string]float64)
					for _, k := range s.KlineData {
						m[k.Date] = k.Close
					}
					stockDailyMap[s.Code] = m
				}

				dailyReturns := make([]models.DailyReturn, 0, len(benchIdx.KlineData))
				for _, bk := range benchIdx.KlineData {
					dt := bk.Date
					benchRet := round2((bk.Close - benchBase) / benchBase * 100)

					stockRets := make([]float64, 0)
					for _, s := range watchlistStocks {
						sd := stockDailyMap[s.Code]
						closes := make([]float64, 0, len(s.KlineData))
						for _, k := range s.KlineData {
							closes = append(closes, k.Close)
						}
						if len(closes) > 0 {
							if _, ok := sd[dt]; ok {
								baseClose := closes[0]
								if baseClose > 0 {
									stockRets = append(stockRets, round2((sd[dt]-baseClose)/baseClose*100))
								}
							}
						}
					}

					dr := models.DailyReturn{
						Date:      dt,
						Benchmark: benchRet,
					}
					if len(stockRets) > 0 {
						sum := 0.0
						for _, r := range stockRets {
							sum += r
						}
						portRet := round2(sum / float64(len(stockRets)))
						dr.Portfolio = &portRet
					}
					dailyReturns = append(dailyReturns, dr)
				}
				portfolio.DailyReturns = dailyReturns
			}
		}
	}

	now := time.Now().In(chinaTZ())
	trends := models.MarketTrends{
		Indices:         indices,
		WatchlistStocks: watchlistStocks,
		Portfolio:       portfolio,
		FetchedAt:       now.Format("2006-01-02 15:04:05"),
	}

	h.trendsCache.Set("trends:all", trends, 60*time.Second)

	c.JSON(http.StatusOK, trends)
}

// stripExchangePrefix converts "sh000001" -> "000001", "sz399001" -> "399001".
// For plain 6-digit codes, returns as-is.
func stripExchangePrefix(code string) string {
	if len(code) > 2 && (code[:2] == "sh" || code[:2] == "sz") {
		return code[2:]
	}
	return code
}

func (h *MarketHandler) fetchTrendStock(ctx context.Context, code string, name string) models.TrendStock {
	klineCode := stripExchangePrefix(code)
	klines, err := h.eastMoney.FetchKline(ctx, klineCode, trendKlineLimit)
	if err != nil {
		if name == "" {
			name = code
		}
		return models.TrendStock{
			Code:      code,
			Name:      name,
			KlineData: []models.KlinePoint{},
		}
	}

	ts := models.TrendStock{
		Code:      code,
		Name:      name,
		KlineData: klines,
		Change1D:  calcKlineChange(klines, 1),
		Change5D:  calcKlineChange(klines, 5),
		Change20D: calcKlineChange(klines, 20),
		Change30D: calcKlineChange(klines, 30),
	}

	if len(klines) > 0 {
		p := klines[len(klines)-1].Close
		ts.Price = &p
	}

	// For stocks (not indices), try to get name from quote if not provided
	if name == "" {
		if q, err := h.tencent.FetchQuote(ctx, code); err == nil && q.Name != "" {
			ts.Name = q.Name
		} else {
			ts.Name = code
		}
	}

	// Keep only last 30 klines in response
	if len(ts.KlineData) > 30 {
		ts.KlineData = ts.KlineData[len(ts.KlineData)-30:]
	}

	return ts
}

func (h *MarketHandler) loadWatchlistCodes(ctx context.Context) []string {
	rows, err := h.db.Query(ctx, `SELECT code FROM watchlist ORDER BY added_at DESC`)
	if err != nil {
		return nil
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
	return codes
}

func portfolioChange(stocks []models.TrendStock, period int) *float64 {
	var vals []float64
	for _, s := range stocks {
		var v *float64
		switch period {
		case 1:
			v = s.Change1D
		case 5:
			v = s.Change5D
		case 20:
			v = s.Change20D
		case 30:
			v = s.Change30D
		}
		if v != nil {
			vals = append(vals, *v)
		}
	}
	if len(vals) == 0 {
		return nil
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	r := round2(sum / float64(len(vals)))
	return &r
}

// @Summary      市场全景
// @Description  获取主要指数实时行情与市场涨跌家数统计
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} models.MarketOverviewResponse
// @Router       /api/market/market-overview [get]
func (h *MarketHandler) MarketOverview(c *gin.Context) {
	if cached, ok := h.marketOverviewCache.Get("market_overview"); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	// Define the 6 major indices (same as Python version)
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
		quotes, err := h.tencent.FetchIndexQuotes(c.Request.Context(), indices)
		idxCh <- indexResult{quotes: quotes, err: err}
	}()
	go func() {
		breadth, err := h.eastMoney.FetchMarketBreadth(c.Request.Context())
		brCh <- breadthResult{breadth: breadth, err: err}
	}()

	ir := <-idxCh
	br := <-brCh

	var indexQuotes []models.IndexQuote
	if ir.err != nil {
		indexQuotes = []models.IndexQuote{}
	} else {
		indexQuotes = ir.quotes
	}

	breadth := br.breadth
	if br.err != nil {
		breadth = models.MarketBreadth{
			UpCount: 0, DownCount: 0, FlatCount: 0,
			Sentiment: "中性", SentimentRatio: 50,
		}
	}

	resp := models.MarketOverviewResponse{
		OK:      true,
		Indices: indexQuotes,
		Market:  breadth,
	}
	h.marketOverviewCache.Set("market_overview", resp, 30*time.Second)
	c.JSON(http.StatusOK, resp)
}

// @Summary      获取热门概念板块
// @Description  获取当日最热门的概念板块排行（东方财富数据）
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /api/market/hot-concepts [get]
func (h *MarketHandler) HotConcepts(c *gin.Context) {
	if cached, ok := h.hotConceptsCache.Get("hot_concepts"); ok {
		c.JSON(http.StatusOK, gin.H{"ok": true, "concepts": cached})
		return
	}

	concepts, err := h.eastMoney.FetchHotConcepts(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "HOT_CONCEPTS_FETCH_FAILED", "failed to fetch hot concepts")
		return
	}
	if concepts == nil {
		concepts = []models.HotConcept{}
	}

	h.hotConceptsCache.Set("hot_concepts", concepts, 5*time.Minute)
	c.JSON(http.StatusOK, gin.H{"ok": true, "concepts": concepts})
}

// @Summary      市场宽度
// @Description  获取涨跌家数、涨停跌停、AD比率、成交量等市场宽度指标
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /api/market/breadth [get]
func (h *MarketHandler) MarketBreadth(c *gin.Context) {
	if cached, ok := h.breadthCache.Get("breadth"); ok {
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": cached, "cached": true})
		return
	}

	breadth, err := h.eastMoney.FetchMarketBreadthDetail(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "BREADTH_FETCH_FAILED", "failed to fetch market breadth data")
		return
	}

	h.breadthCache.Set("breadth", breadth, 5*time.Minute)
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": breadth, "cached": false})
}

// @Summary      市场情绪
// @Description  获取恐贪指数、涨跌家数、板块成交额、市场温度等情绪指标
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /api/market/sentiment [get]
func (h *MarketHandler) MarketSentiment(c *gin.Context) {
	if cached, ok := h.sentimentCache.Get("sentiment"); ok {
		c.JSON(http.StatusOK, gin.H{
			"ok":               cached.OK,
			"fear_greed_index": cached.FearGreedIndex,
			"fear_greed_label": cached.FearGreedLabel,
			"up_count":         cached.UpCount,
			"down_count":       cached.DownCount,
			"flat_count":       cached.FlatCount,
			"total_count":      cached.TotalCount,
			"limit_up":         cached.LimitUp,
			"limit_down":       cached.LimitDown,
			"volume_today":     cached.VolumeToday,
			"volume_avg_5d":    cached.VolumeAvg5D,
			"sector_volumes":   cached.SectorVolumes,
			"temperature":      cached.Temperature,
			"server_time":      cached.ServerTime,
			"cached":           true,
		})
		return
	}

	data, err := h.eastMoney.FetchMarketSentimentData(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "SENTIMENT_FETCH_FAILED", "failed to fetch market sentiment data")
		return
	}

	h.sentimentCache.Set("sentiment", data, 5*time.Minute)
	c.JSON(http.StatusOK, data)
}

// @Summary      获取概念板块成分股
// @Description  根据概念板块代码获取其成分股票列表，标注是否在自选中
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Param        code path string true "概念板块代码"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /api/market/hot-concepts/{code}/stocks [get]
func (h *MarketHandler) HotConceptStocks(c *gin.Context) {
	conceptCode := strings.TrimSpace(c.Param("code"))
	if conceptCode == "" {
		writeError(c, http.StatusBadRequest, "INVALID_CODE", "concept code is required")
		return
	}

	cacheKey := "concept_stocks:" + conceptCode
	if cached, ok := h.conceptStocksCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{"ok": true, "concept_code": conceptCode, "stocks": cached, "cached": true})
		return
	}

	ctx := c.Request.Context()
	members, err := h.eastMoney.FetchSectorMembers(ctx, conceptCode, 200)
	if err != nil {
		logger.L().Warn("fetch concept stocks", zap.String("concept", conceptCode), zap.Error(err))
		writeError(c, http.StatusInternalServerError, "CONCEPT_STOCKS_FAILED", "failed to fetch concept stocks")
		return
	}

	// Get watchlist set from DB
	wlSet := make(map[string]bool)
	rows, err := h.db.Query(ctx, "SELECT code FROM watchlist")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var code string
			if err := rows.Scan(&code); err == nil {
				wlSet[code] = true
			}
		}
	}

	stocks := make([]conceptStockItem, 0, len(members))
	for _, m := range members {
		stocks = append(stocks, conceptStockItem{
			SectorMember: m,
			InWatchlist:  wlSet[m.Code],
		})
	}

	h.conceptStocksCache.Set(cacheKey, stocks, 5*time.Minute)
	c.JSON(http.StatusOK, gin.H{"ok": true, "concept_code": conceptCode, "stocks": stocks, "cached": false})
}

// @Summary      自选股与热门概念交叉
// @Description  检查自选股出现在哪些热门概念板块中
// @Tags         market
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /api/watchlist-concept-overlap [get]
func (h *MarketHandler) WatchlistConceptOverlap(c *gin.Context) {
	ctx := c.Request.Context()

	// Get hot concepts
	var concepts []models.HotConcept
	if cached, ok := h.hotConceptsCache.Get("hot_concepts"); ok {
		concepts = cached
	} else {
		var err error
		concepts, err = h.eastMoney.FetchHotConcepts(ctx)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "HOT_CONCEPTS_FAILED", "failed to fetch hot concepts")
			return
		}
	}

	// Get watchlist codes
	wlCodes := make(map[string]bool)
	rows, err := h.db.Query(ctx, "SELECT code FROM watchlist")
	if err != nil {
		writeError(c, http.StatusInternalServerError, "WATCHLIST_QUERY_FAILED", "failed to query watchlist")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err == nil {
			wlCodes[code] = true
		}
	}

	// For each concept, fetch members and check overlap
	overlap := make(map[string][]conceptOverlapItem)
	for _, concept := range concepts {
		if concept.Code == "" {
			continue
		}
		members, err := h.eastMoney.FetchSectorMembers(ctx, concept.Code, 200)
		if err != nil {
			continue
		}
		for _, m := range members {
			if wlCodes[m.Code] {
				overlap[m.Code] = append(overlap[m.Code], conceptOverlapItem{
					Code: concept.Code,
					Name: concept.Name,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "overlap": overlap})
}

func (h *MarketHandler) CacheStats() map[string]cache.Sizer {
	return map[string]cache.Sizer{
		"quote":           h.quoteCache,
		"kline":           h.klineCache,
		"sectors":         h.sectorsCache,
		"overview":        h.overviewCache,
		"news":            h.newsCache,
		"announcements":   h.announcementsCache,
		"search":          h.searchCache,
		"top_movers":      h.topMoversCache,
		"trends":          h.trendsCache,
		"market_overview": h.marketOverviewCache,
		"hot_concepts":    h.hotConceptsCache,
		"breadth":         h.breadthCache,
		"sentiment":       h.sentimentCache,
	}
}
