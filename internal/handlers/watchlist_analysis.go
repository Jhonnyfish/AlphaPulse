package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// WatchlistAnalysisHandler provides watchlist analysis endpoints.
type WatchlistAnalysisHandler struct {
	db        *pgxpool.Pool
	tencent   *services.TencentService
	eastMoney *services.EastMoneyService
	tushareDB *services.TushareDB
	analyze   *AnalyzeHandler
	log       *zap.Logger
	scoring   *services.ScoringEngine

	heatmapCache *cache.Cache[[]HeatmapItem]
	sectorsCache *cache.Cache[SectorsResponse]
	rankingCache *cache.Cache[RankingResponse]
}

// NewWatchlistAnalysisHandler creates a new WatchlistAnalysisHandler.
func NewWatchlistAnalysisHandler(
	db *pgxpool.Pool,
	tencent *services.TencentService,
	eastMoney *services.EastMoneyService,
	analyze *AnalyzeHandler,
	log *zap.Logger,
) *WatchlistAnalysisHandler {
	return &WatchlistAnalysisHandler{
		db:           db,
		tencent:      tencent,
		eastMoney:    eastMoney,
		analyze:      analyze,
		log:          log,
		scoring:      services.NewScoringEngine(),
		heatmapCache: cache.New[[]HeatmapItem](),
		sectorsCache: cache.New[SectorsResponse](),
		rankingCache: cache.New[RankingResponse](),
	}
}

// SetTushareDB sets the Tushare local database service as primary data source.
func (h *WatchlistAnalysisHandler) SetTushareDB(db *services.TushareDB) {
	h.tushareDB = db
}

// InvalidateRankingCache clears the ranking cache so the next request re-analyzes.
// Called when the watchlist changes (add/remove/sync).
func (h *WatchlistAnalysisHandler) InvalidateRankingCache() {
	h.rankingCache.Delete("all")
	h.log.Info("ranking cache invalidated due to watchlist change")
}

// PreComputeRanking computes the ranking and stores it in cache.
// Intended to be called by the scheduler after daily data sync, so the
// ranking page serves cached results instantly without user-triggered computation.
func (h *WatchlistAnalysisHandler) PreComputeRanking() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	start := time.Now()
	h.log.Info("pre-compute ranking: starting...")

	codes, err := h.loadWatchlistCodes(ctx)
	if err != nil {
		h.log.Warn("pre-compute ranking: load watchlist failed", zap.Error(err))
		return
	}
	if len(codes) == 0 {
		h.log.Info("pre-compute ranking: empty watchlist, skip")
		return
	}

	// Analyze each stock concurrently (limit to 8 workers)
	items := make([]RankingItem, len(codes))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, code := range codes {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, cd string) {
			defer wg.Done()
			defer func() { <-sem }()
			items[idx] = h.analyzeForRanking(ctx, cd)
		}(i, code)
	}
	wg.Wait()

	// Sort by overall_score descending, filter out errors
	valid := make([]RankingItem, 0, len(items))
	for _, item := range items {
		if item.Error == "" {
			valid = append(valid, item)
		}
	}
	sort.Slice(valid, func(i, j int) bool {
		return valid[i].OverallScore > valid[j].OverallScore
	})
	for i := range valid {
		valid[i].Rank = i + 1
	}

	// Calculate sector rankings
	calculateSectorRankings(valid)
	h.enrichWithTrends(ctx, valid)

	// Build summary
	var avgScore float64
	var best, worst *RankingBest
	if len(valid) > 0 {
		total := 0
		for _, item := range valid {
			total += item.OverallScore
		}
		avgScore = float64(total) / float64(len(valid))
		best = &RankingBest{Code: valid[0].Code, Name: valid[0].Name, Score: valid[0].OverallScore}
		worst = &RankingBest{Code: valid[len(valid)-1].Code, Name: valid[len(valid)-1].Name, Score: valid[len(valid)-1].OverallScore}
	}

	resp := RankingResponse{
		OK:    true,
		Items: valid,
		Summary: RankingSummary{
			AvgScore: avgScore,
			Best:     best,
			Worst:    worst,
			Count:    len(valid),
		},
		FetchedAt: time.Now().Format(time.RFC3339),
	}

	h.rankingCache.Set("all", resp, 12*time.Hour)
	h.log.Info("pre-compute ranking: done",
		zap.Int("stocks", len(valid)),
		zap.Duration("elapsed", time.Since(start)))
}

// ---- Heatmap ----

// HeatmapItem is a single stock in the heatmap view.
type HeatmapItem struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	ChangePct  float64 `json:"change_pct"`
	Volume     float64 `json:"volume"`
	Amount     float64 `json:"amount"`
}

// HeatmapResponse is the response for GET /api/watchlist-heatmap.
type HeatmapResponse struct {
	OK    bool           `json:"ok"`
	Items []HeatmapItem  `json:"items"`
	Error string         `json:"error,omitempty"`
}

// Heatmap returns all watchlist stocks with real-time price data for heatmap rendering.
// @Summary      获取自选股热力图数据
// @Description  返回所有自选股的实时价格数据，用于热力图渲染
// @Tags         watchlist-analysis
// @Produce      json
// @Success      200  {object}  HeatmapResponse
// @Router       /api/watchlist-heatmap [get]
func (h *WatchlistAnalysisHandler) Heatmap(c *gin.Context) {
	if cached, ok := h.heatmapCache.Get("all"); ok {
		c.JSON(http.StatusOK, HeatmapResponse{OK: true, Items: cached})
		return
	}

	codes, err := h.loadWatchlistCodes(c.Request.Context())
	if err != nil {
		h.log.Warn("heatmap: load watchlist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, HeatmapResponse{OK: false, Error: err.Error()})
		return
	}

	if len(codes) == 0 {
		c.JSON(http.StatusOK, HeatmapResponse{OK: true, Items: []HeatmapItem{}})
		return
	}

	items := make([]HeatmapItem, len(codes))
	var wg sync.WaitGroup
	for i, code := range codes {
		wg.Add(1)
		go func(idx int, cd string) {
			defer wg.Done()
			var quote models.Quote
			var err error
			if h.tushareDB != nil {
				quote, err = h.tushareDB.FetchQuoteFromDB(context.Background(), cd)
			}
			if err != nil || quote.Price <= 0 {
				quote, err = h.tencent.FetchQuote(context.Background(), cd)
			}
			if err != nil {
				h.log.Debug("heatmap: fetch quote failed", zap.String("code", cd), zap.Error(err))
				items[idx] = HeatmapItem{Code: services.StockCode6(cd), Name: cd}
				return
			}
			items[idx] = HeatmapItem{
				Code:      services.StockCode6(cd),
				Name:      quote.Name,
				Price:     quote.Price,
				ChangePct: quote.ChangePercent,
				Volume:    quote.Volume,
				Amount:    quote.Turnover,
			}
		}(i, code)
	}
	wg.Wait()

	h.heatmapCache.Set("all", items, 10*time.Second)
	c.JSON(http.StatusOK, HeatmapResponse{OK: true, Items: items})
}

// ---- Sectors ----

// SectorGroup is a group of stocks belonging to the same sector.
type SectorGroup struct {
	Name   string   `json:"name"`
	Count  int      `json:"count"`
	Stocks []string `json:"stocks"`
}

// SectorsResponse is the response for GET /api/watchlist-sectors.
type SectorsResponse struct {
	OK      bool          `json:"ok"`
	Sectors []SectorGroup `json:"sectors"`
	Total   int           `json:"total"`
	Error   string        `json:"error,omitempty"`
}

// sectorCleanRe matches the industry code prefix like "C39".
var sectorCleanRe = regexp.MustCompile(`^[A-Z]\d{2}`)

// sectorSuffixes to strip from sector names.
var sectorSuffixes = []string{
	"制造业", "供应业", "服务业", "零售业", "开采业",
	"加工业", "制品业", "生产业",
}

// simplifySector cleans sector names like "C39计算机、通信和其他电子设备制造业" → "通信电子".
func simplifySector(raw string) string {
	if !sectorCleanRe.MatchString(raw) {
		return raw
	}
	clean := raw[3:]
	for _, suffix := range sectorSuffixes {
		if strings.HasSuffix(clean, suffix) {
			clean = clean[:len(clean)-len(suffix)]
			break
		}
	}
	clean = strings.NewReplacer("、", "", "和", "", "其他", "").Replace(clean)
	if clean == "" {
		if len(raw) > 9 {
			return raw[3:9]
		}
		return raw[3:]
	}
	runes := []rune(clean)
	if len(runes) > 8 {
		return string(runes[:8])
	}
	return clean
}

// Sectors returns sector distribution of watchlist stocks.
// @Summary      获取自选股板块分布
// @Description  返回自选股的行业板块分布统计
// @Tags         watchlist-analysis
// @Produce      json
// @Success      200  {object}  SectorsResponse
// @Router       /api/watchlist-sectors [get]
func (h *WatchlistAnalysisHandler) Sectors(c *gin.Context) {
	if cached, ok := h.sectorsCache.Get("all"); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	codes, err := h.loadWatchlistCodes(c.Request.Context())
	if err != nil {
		h.log.Warn("sectors: load watchlist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, SectorsResponse{OK: false, Error: err.Error()})
		return
	}

	if len(codes) == 0 {
		resp := SectorsResponse{OK: true, Sectors: []SectorGroup{}, Total: 0}
		c.JSON(http.StatusOK, resp)
		return
	}

	// Fetch sectors for each stock concurrently
	type codeSectors struct {
		code    string
		sectors []string
	}
	results := make([]codeSectors, len(codes))
	var wg sync.WaitGroup
	for i, code := range codes {
		wg.Add(1)
		go func(idx int, cd string) {
			defer wg.Done()
			// Try TushareDB first
			if h.tushareDB != nil {
				if industry, err := h.tushareDB.FetchIndustryFromDB(context.Background(), cd); err == nil && industry != "" {
					results[idx] = codeSectors{code: cd, sectors: []string{industry}}
					return
				}
			}
			// Fallback to EastMoney
			sectors, err := h.eastMoney.FetchStockSectors(context.Background(), cd)
			if err != nil {
				h.log.Debug("sectors: fetch failed", zap.String("code", cd), zap.Error(err))
				results[idx] = codeSectors{code: cd, sectors: []string{"未分类"}}
				return
			}
			names := make([]string, 0, len(sectors))
			for _, s := range sectors {
				names = append(names, s.Name)
			}
			if len(names) == 0 {
				names = []string{"未分类"}
			}
			results[idx] = codeSectors{code: cd, sectors: names}
		}(i, code)
	}
	wg.Wait()

	// Group by primary sector (first sector for each stock)
	sectorMap := make(map[string][]string)
	for _, r := range results {
		sector := r.sectors[0]
		sector = simplifySector(sector)
		sectorMap[sector] = append(sectorMap[sector], services.StockCode6(r.code))
	}

	sectors := make([]SectorGroup, 0, len(sectorMap))
	for name, stocks := range sectorMap {
		sort.Strings(stocks)
		sectors = append(sectors, SectorGroup{
			Name:   name,
			Count:  len(stocks),
			Stocks: stocks,
		})
	}
	sort.Slice(sectors, func(i, j int) bool {
		return sectors[i].Count > sectors[j].Count
	})

	resp := SectorsResponse{OK: true, Sectors: sectors, Total: len(codes)}
	h.sectorsCache.Set("all", resp, 60*time.Second)
	c.JSON(http.StatusOK, resp)
}

// ---- Ranking ----

// RankingItem is a single stock in the ranking view.
type RankingItem struct {
	Code            string             `json:"code"`
	Name            string             `json:"name"`
	OverallScore    int                `json:"overall_score"`
	OverallSignal   string             `json:"overall_signal"`
	DimensionScores map[string]float64 `json:"dimension_scores"`
	ChangePct       float64            `json:"change_pct"`
	Price           float64            `json:"price"`
	Strengths       []string           `json:"strengths"`
	Risks           []string           `json:"risks"`
	Rank            int                `json:"rank"`
	// Enhanced scoring fields
	WeightedScore   float64                    `json:"weighted_score"`
	PeriodScores    models.MultiPeriodScore    `json:"period_scores"`
	Confidence      models.Confidence          `json:"confidence"`
	DimContributions map[string]float64        `json:"dim_contributions"`
	// Sector ranking
	Sector          string `json:"sector"`
	SectorRank      int    `json:"sector_rank"`
	SectorTotal     int    `json:"sector_total"`
	// Strategy used
	Strategy        string `json:"strategy"`
	// Score trend from score_history
	ScoreTrend    string  `json:"score_trend"`
	ScoreChange7D float64 `json:"score_change_7d,omitempty"`
	Error         string  `json:"error,omitempty"`
}

// RankingSummary is summary statistics for the ranking.
type RankingSummary struct {
	AvgScore float64       `json:"avg_score"`
	Best     *RankingBest  `json:"best"`
	Worst    *RankingBest  `json:"worst"`
	Count    int           `json:"count"`
}

// RankingBest represents the best/worst stock.
type RankingBest struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

// RankingResponse is the response for GET /api/watchlist-ranking.
type RankingResponse struct {
	OK        bool           `json:"ok"`
	Items     []RankingItem  `json:"items"`
	Summary   RankingSummary `json:"summary"`
	FetchedAt string         `json:"fetched_at"`
	Error     string         `json:"error,omitempty"`
}

// Ranking returns a ranked analysis of all watchlist stocks.
// @Summary      获取自选股排名分析
// @Description  对所有自选股进行多维度分析并返回排名结果
// @Tags         watchlist-analysis
// @Produce      json
// @Param        strategy query string false "排名策略: momentum/value/balanced/default"
// @Success      200  {object}  RankingResponse
// @Router       /api/watchlist-ranking [get]
func (h *WatchlistAnalysisHandler) Ranking(c *gin.Context) {
	strategy := models.ScoringStrategy(c.DefaultQuery("strategy", "default"))
	cacheKey := "all_" + string(strategy)

	if cached, ok := h.rankingCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	codes, err := h.loadWatchlistCodes(c.Request.Context())
	if err != nil {
		h.log.Warn("ranking: load watchlist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, RankingResponse{OK: false, Error: err.Error()})
		return
	}

	if len(codes) == 0 {
		resp := RankingResponse{
			OK:        true,
			Items:     []RankingItem{},
			Summary:   RankingSummary{AvgScore: 0, Count: 0},
			FetchedAt: time.Now().Format(time.RFC3339),
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	// Analyze each stock concurrently (limit to 16 workers)
	items := make([]RankingItem, len(codes))
	sem := make(chan struct{}, 16)
	var wg sync.WaitGroup
	for i, code := range codes {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, cd string) {
			defer wg.Done()
			defer func() { <-sem }()
			items[idx] = h.analyzeForRanking(c.Request.Context(), cd)
		}(i, code)
	}
	wg.Wait()

	// Sort by overall_score descending, filter out errors
	valid := make([]RankingItem, 0, len(items))
	for _, item := range items {
		if item.Error == "" {
			valid = append(valid, item)
		}
	}
	sort.Slice(valid, func(i, j int) bool {
		return valid[i].OverallScore > valid[j].OverallScore
	})
	for i := range valid {
		valid[i].Rank = i + 1
	}

	calculateSectorRankings(valid)
	h.enrichWithTrends(c.Request.Context(), valid)
	// Build summary
	var avgScore float64
	var best, worst *RankingBest
	if len(valid) > 0 {
		total := 0
		for _, item := range valid {
			total += item.OverallScore
		}
		avgScore = float64(total) / float64(len(valid))
		best = &RankingBest{
			Code:  valid[0].Code,
			Name:  valid[0].Name,
			Score: valid[0].OverallScore,
		}
		worst = &RankingBest{
			Code:  valid[len(valid)-1].Code,
			Name:  valid[len(valid)-1].Name,
			Score: valid[len(valid)-1].OverallScore,
		}
	}

	resp := RankingResponse{
		OK:    true,
		Items: valid,
		Summary: RankingSummary{
			AvgScore: avgScore,
			Best:     best,
			Worst:    worst,
			Count:    len(valid),
		},
		FetchedAt: time.Now().Format(time.RFC3339),
	}

	h.rankingCache.Set(cacheKey, resp, 180*time.Second)
	c.JSON(http.StatusOK, resp)
}

// streamMessage types for NDJSON streaming.
type streamBasic struct {
	Type   string        `json:"type"`
	Stocks []streamStock `json:"stocks"`
}
type streamStock struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
type streamResult struct {
	Type      string      `json:"type"`
	Item      RankingItem `json:"item"`
	Completed int         `json:"completed"`
	Total     int         `json:"total"`
}
type streamSummary struct {
	Type      string         `json:"type"`
	Summary   RankingSummary `json:"summary"`
	FetchedAt string         `json:"fetched_at"`
}
type streamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// RankingStream streams analysis results progressively as NDJSON.
// GET /api/watchlist-ranking/stream
func (h *WatchlistAnalysisHandler) RankingStream(c *gin.Context) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	c.Writer.Header().Set("Content-Type", "application/x-ndjson")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
	c.Writer.WriteHeader(http.StatusOK)

	writeLine := func(v interface{}) {
		data, _ := json.Marshal(v)
		c.Writer.Write(append(data, '\n'))
		flusher.Flush()
	}

	// 1. Send basic stock info immediately
	codes, err := h.loadWatchlistCodes(c.Request.Context())
	if err != nil {
		writeLine(streamError{Type: "error", Message: err.Error()})
		return
	}

	if len(codes) == 0 {
		writeLine(streamSummary{
			Type:      "summary",
			Summary:   RankingSummary{AvgScore: 0, Count: 0},
			FetchedAt: time.Now().Format(time.RFC3339),
		})
		return
	}

	// Load names for basic info
	stocks := make([]streamStock, len(codes))
	for i, code := range codes {
		stocks[i] = streamStock{Code: code, Name: code}
	}
	// Quick name lookup from DB
	nameRows, nameErr := h.db.Query(c.Request.Context(),
		`SELECT code, COALESCE(name, '') FROM watchlist`)
	if nameErr == nil {
		defer nameRows.Close()
		nameMap := make(map[string]string)
		for nameRows.Next() {
			var cd, nm string
			if nameRows.Scan(&cd, &nm) == nil {
				nameMap[cd] = nm
			}
		}
		for i := range stocks {
			if nm, ok := nameMap[stocks[i].Code]; ok && nm != "" {
				stocks[i].Name = nm
			}
		}
	}

	writeLine(streamBasic{Type: "basic", Stocks: stocks})

	// 2. Analyze and stream results
	total := len(codes)
	type indexedResult struct {
		idx  int
		item RankingItem
	}
	resultCh := make(chan indexedResult, total)
	sem := make(chan struct{}, 16)
	var wg sync.WaitGroup

	for i, code := range codes {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, cd string) {
			defer wg.Done()
			defer func() { <-sem }()
			item := h.analyzeForRanking(c.Request.Context(), cd)
			resultCh <- indexedResult{idx: idx, item: item}
		}(i, code)
	}

	// Close channel when all goroutines finish
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results and stream as they arrive
	completed := 0
	allItems := make([]RankingItem, total)
	for r := range resultCh {
		allItems[r.idx] = r.item
		completed++
		writeLine(streamResult{
			Type:      "result",
			Item:      r.item,
			Completed: completed,
			Total:     total,
		})
	}

	// 3. Build and send summary
	valid := make([]RankingItem, 0, total)
	for _, item := range allItems {
		if item.Error == "" {
			valid = append(valid, item)
		}
	}
	sort.Slice(valid, func(i, j int) bool {
		return valid[i].OverallScore > valid[j].OverallScore
	})
	for i := range valid {
		valid[i].Rank = i + 1
	}

		calculateSectorRankings(valid)
		h.enrichWithTrends(c.Request.Context(), valid)

	var avgScore float64
	var best, worst *RankingBest
	if len(valid) > 0 {
		totalScore := 0
		for _, item := range valid {
			totalScore += item.OverallScore
		}
		avgScore = float64(totalScore) / float64(len(valid))
		best = &RankingBest{Code: valid[0].Code, Name: valid[0].Name, Score: valid[0].OverallScore}
		worst = &RankingBest{Code: valid[len(valid)-1].Code, Name: valid[len(valid)-1].Name, Score: valid[len(valid)-1].OverallScore}
	}

	// Cache the full result so the regular endpoint uses it
	h.rankingCache.Set("all", RankingResponse{
		OK:    true,
		Items: valid,
		Summary: RankingSummary{
			AvgScore: avgScore,
			Best:     best,
			Worst:    worst,
			Count:    len(valid),
		},
		FetchedAt: time.Now().Format(time.RFC3339),
	}, 180*time.Second)

	writeLine(streamSummary{
		Type: "summary",
		Summary: RankingSummary{
			AvgScore: avgScore,
			Best:     best,
			Worst:    worst,
			Count:    len(valid),
		},
		FetchedAt: time.Now().Format(time.RFC3339),
	})
}

func (h *WatchlistAnalysisHandler) analyzeForRanking(ctx context.Context, code string) RankingItem {
	return h.analyzeForRankingWithStrategy(ctx, code, models.StrategyDefault)
}

func (h *WatchlistAnalysisHandler) analyzeForRankingWithStrategy(ctx context.Context, code string, strategy models.ScoringStrategy) RankingItem {
	analysis := h.analyze.analyzeSingleWithMode(ctx, code, true)
	if len(analysis.Errors) > 0 {
		// Check if critical data failed
		if _, ok := analysis.Errors["quote"]; ok {
			return RankingItem{
				Code:  services.StockCode6(code),
				Name:  code,
				Error: "quote fetch failed",
			}
		}
	}

	// Compute dimension scores (0-100)
	dimScoresMap := make(map[string]float64)
	dimScoresMap["order_flow"] = scoreDimension(analysis.OrderFlow.Verdict, analysis.OrderFlow.OuterRatio > 55)
	dimScoresMap["volume_price"] = scoreDimensionVP(analysis.VolumePrice)
	dimScoresMap["valuation"] = scoreDimensionValuation(analysis.Valuation)
	dimScoresMap["volatility"] = scoreDimension(analysis.Volatility.Verdict, false)
	dimScoresMap["money_flow"] = scoreDimensionMF(analysis.MoneyFlow)
	dimScoresMap["technical"] = scoreDimensionTech(analysis.Technical)
	dimScoresMap["sector"] = scoreDimension(analysis.Sector.Verdict, analysis.Sector.IsSectorLeader)
	dimScoresMap["sentiment"] = scoreDimension(analysis.Sentiment.Verdict, analysis.Sentiment.SentimentScore > 0)
	dimScoresMap["fundamentals"] = scoreDimensionFundamentals(analysis.Fundamentals)
	dimScoresMap["northbound"] = scoreDimensionNorthbound(analysis.Northbound)
	dimScoresMap["margin"] = scoreDimensionMargin(analysis.MarginDetail)

	// Build dimension scores struct for scoring engine
	dimScores := services.DimensionScores{
		OrderFlow:    dimScoresMap["order_flow"],
		VolumePrice:  dimScoresMap["volume_price"],
		Valuation:    dimScoresMap["valuation"],
		Volatility:   dimScoresMap["volatility"],
		MoneyFlow:    dimScoresMap["money_flow"],
		Technical:    dimScoresMap["technical"],
		Sector:       dimScoresMap["sector"],
		Sentiment:    dimScoresMap["sentiment"],
		Fundamentals: dimScoresMap["fundamentals"],
		Northbound:   dimScoresMap["northbound"],
		Margin:       dimScoresMap["margin"],
	}

	// Compute enhanced summary with strategy
	enhanced := h.scoring.ComputeEnhancedSummaryWithStrategy(&analysis, dimScores, strategy)

	return RankingItem{
		Code:             services.StockCode6(code),
		Name:             analysis.Name,
		OverallScore:     enhanced.OverallScore,
		OverallSignal:    enhanced.OverallSignal,
		DimensionScores:  dimScoresMap,
		ChangePct:        analysis.Quote.ChangePercent,
		Price:            analysis.Quote.Price,
		Strengths:        safeStrings(enhanced.Strengths),
		Risks:            safeStrings(enhanced.Risks),
		WeightedScore:    enhanced.WeightedScore,
		PeriodScores:     enhanced.PeriodScores,
		Confidence:       enhanced.Confidence,
		DimContributions: enhanced.DimContributions,
		Sector:           analysis.Sector.PrimarySector,
		Strategy:         string(strategy),
	}
}

// calculateSectorRankings calculates each stock's rank within its sector.
func calculateSectorRankings(items []RankingItem) {
	// Group stocks by sector
	sectorGroups := make(map[string][]int) // sector -> indices in items
	for i, item := range items {
		if item.Sector != "" {
			sectorGroups[item.Sector] = append(sectorGroups[item.Sector], i)
		}
	}

	// Calculate rank within each sector
	for _, indices := range sectorGroups {
		// Sort indices by score descending
		sort.Slice(indices, func(a, b int) bool {
			return items[indices[a]].OverallScore > items[indices[b]].OverallScore
		})
		total := len(indices)
		for rank, idx := range indices {
			items[idx].SectorRank = rank + 1
			items[idx].SectorTotal = total
		}
	}
}

// enrichWithTrends batch-queries score_history to add trend data to each ranking item.
func (h *WatchlistAnalysisHandler) enrichWithTrends(ctx context.Context, items []RankingItem) {
	if len(items) == 0 {
		return
	}

	codes := make([]string, 0, len(items))
	for _, item := range items {
		if item.Code != "" {
			codes = append(codes, item.Code)
		}
	}

	rows, err := h.db.Query(ctx,
		`WITH latest AS (
			SELECT DISTINCT ON (code) code, score
			FROM score_history
			WHERE code = ANY($1)
			ORDER BY code, recorded_at DESC
		),
		old AS (
			SELECT DISTINCT ON (code) code, score
			FROM score_history
			WHERE code = ANY($1)
			  AND recorded_at <= NOW() - INTERVAL '6 days'
			ORDER BY code, recorded_at DESC
		)
		SELECT l.code, l.score, o.score
		FROM latest l
		LEFT JOIN old o ON l.code = o.code`, codes)
	if err != nil {
		h.log.Warn("enrichWithTrends: query failed", zap.Error(err))
		return
	}
	defer rows.Close()

	type trendData struct {
		current float64
		prev7d  float64
		hasOld  bool
	}
	trends := make(map[string]trendData, len(codes))
	for rows.Next() {
		var code string
		var current float64
		var prev7d *float64
		if err := rows.Scan(&code, &current, &prev7d); err != nil {
			continue
		}
		td := trendData{current: current}
		if prev7d != nil {
			td.prev7d = *prev7d
			td.hasOld = true
		}
		trends[code] = td
	}

	for i := range items {
		td, ok := trends[items[i].Code]
		if !ok || !td.hasOld {
			continue
		}
		change := td.current - td.prev7d
		items[i].ScoreChange7D = change
		switch {
		case change > 3:
			items[i].ScoreTrend = "rising"
		case change < -3:
			items[i].ScoreTrend = "falling"
		default:
			items[i].ScoreTrend = "stable"
		}
	}
}

// scoreDimension maps a verdict string to a 0-100 score.
func scoreDimension(verdict string, isPositive bool) float64 {
	switch {
	// Strong positive
	case strings.Contains(verdict, "强势") || strings.Contains(verdict, "优秀") || strings.Contains(verdict, "强烈"):
		return 85
	// Bullish signals
	case strings.Contains(verdict, "偏多") || strings.Contains(verdict, "良好") || strings.Contains(verdict, "积极"):
		return 70
	case strings.Contains(verdict, "多头") || strings.Contains(verdict, "金叉"):
		return 75
	case strings.Contains(verdict, "占优") || strings.Contains(verdict, "流入") || strings.Contains(verdict, "较强"):
		return 65
	case strings.Contains(verdict, "上方") || strings.Contains(verdict, "上升"):
		return 60
	// Mild positive
	case strings.Contains(verdict, "略强") || strings.Contains(verdict, "偏强") || strings.Contains(verdict, "观察"):
		return 55
	// Neutral
	case strings.Contains(verdict, "中性") || strings.Contains(verdict, "均衡") || strings.Contains(verdict, "正常") || strings.Contains(verdict, "可控") || strings.Contains(verdict, "持平"):
		return 50
	case strings.Contains(verdict, "数据不足"):
		return 50
	// Mild negative
	case strings.Contains(verdict, "不足") || strings.Contains(verdict, "偏弱") || strings.Contains(verdict, "谨慎"):
		return 40
	case strings.Contains(verdict, "偏空") || strings.Contains(verdict, "较低"):
		return 35
	case strings.Contains(verdict, "偏高") || strings.Contains(verdict, "不便宜"):
		return 35
	// Bearish signals
	case strings.Contains(verdict, "弱势") || strings.Contains(verdict, "危险") || strings.Contains(verdict, "极低"):
		return 20
	case strings.Contains(verdict, "空头") || strings.Contains(verdict, "死叉"):
		return 25
	case strings.Contains(verdict, "超买"):
		return 30
	case strings.Contains(verdict, "超卖"):
		return 70
	case strings.Contains(verdict, "流出") || strings.Contains(verdict, "承压"):
		return 35
	case strings.Contains(verdict, "下跌") || strings.Contains(verdict, "回调"):
		return 35
	default:
		if isPositive {
			return 60
		}
		return 50
	}
}

// scoreDimensionTech scores technical analysis with MACD/KDJ/RSI/MA details.
func scoreDimensionTech(tech models.TechnicalAnalysis) float64 {
	score := 50.0

	// MA arrangement
	switch tech.MAArrangement {
	case "多头排列":
		score += 15
	case "短多排列":
		score += 10
	case "空头排列":
		score -= 15
	case "短空排列":
		score -= 10
	}

	// MACD
	switch tech.MACD_Signal {
	case "金叉", "多头":
		score += 8
	case "死叉", "空头":
		score -= 8
	}

	// KDJ
	switch tech.KDJ_Signal {
	case "超买":
		score -= 5
	case "超卖":
		score += 5
	case "金叉":
		score += 5
	case "死叉":
		score -= 5
	}

	// RSI
	if tech.RSI_14 > 70 {
		score -= 5
	} else if tech.RSI_14 < 30 && tech.RSI_14 > 0 {
		score += 5
	} else if tech.RSI_14 > 50 {
		score += 3
	}

	// Boll position
	switch tech.BollPosition {
	case "上轨上方":
		score -= 3 // overbought risk
	case "下轨下方":
		score += 3 // oversold bounce
	}

	return clampScoreRank(score)
}

// scoreDimensionVP scores volume-price analysis.
func scoreDimensionVP(vp models.VolumePriceAnalysis) float64 {
	score := 50.0
	switch vp.PriceVolumeHarmony {
	case "量价齐升":
		score += 20
	case "缩量上涨":
		score += 5
	case "放量下跌":
		score -= 20
	case "缩量下跌":
		score -= 5
	}
	if vp.VolumeRatio > 1.5 {
		score += 5
	} else if vp.VolumeRatio < 0.5 {
		score -= 5
	}
	return clampScoreRank(score)
}

// scoreDimensionMF scores money flow analysis.
func scoreDimensionMF(mf models.MoneyFlowAnalysis) float64 {
	score := 50.0
	switch mf.TodayMainDirection {
	case "流入":
		score += 15
	case "流出":
		score -= 15
	}
	if mf.TodayMainNet > 0 {
		score += 5
	} else if mf.TodayMainNet < 0 {
		score -= 5
	}
	return clampScoreRank(score)
}

// scoreDimensionValuation scores valuation analysis.
func scoreDimensionValuation(vl models.ValuationAnalysis) float64 {
	score := 50.0
	switch vl.PELevel {
	case "偏低":
		score += 15
	case "合理":
		score += 5
	case "偏高":
		score -= 10
	case "很高":
		score -= 15
	}
	switch vl.PBLevel {
	case "偏低":
		score += 10
	case "合理":
		score += 3
	case "偏高":
		score -= 8
	case "很高":
		score -= 12
	}
	return clampScoreRank(score)
}

func clampScoreRank(v float64) float64 {
	if v > 100 {
		return 100
	}
	if v < 0 {
		return 0
	}
	return v
}

// P0 dimension scoring functions

func scoreDimensionFundamentals(fund models.FundamentalsAnalysis) float64 {
	if fund.Score == 0 && fund.Verdict == "暂无财务数据" {
		return 50 // neutral when no data
	}
	return clampScoreRank(float64(fund.Score))
}

func scoreDimensionNorthbound(nb models.NorthboundAnalysis) float64 {
	switch nb.Signal {
	case "北向大幅买入":
		return 85
	case "北向小幅买入":
		return 65
	case "北向大幅卖出":
		return 15
	case "北向小幅卖出":
		return 35
	case "北向成交活跃":
		return 70
	case "北向成交一般":
		return 55
	default:
		return 50
	}
}

func scoreDimensionMargin(mg models.MarginAnalysis) float64 {
	switch mg.Signal {
	case "融资看多":
		return 75
	case "融资偏多":
		return 60
	case "融资看空":
		return 25
	case "融资偏空":
		return 40
	default:
		return 50
	}
}

// safeStrings returns an empty slice instead of nil.
func safeStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ---- Groups CRUD ----

// WatchlistGroup is a named group for organizing watchlist stocks.
type WatchlistGroup struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// GroupsData is the response for groups endpoints.
type GroupsData struct {
	Groups      []WatchlistGroup    `json:"groups"`
	Assignments map[string]string   `json:"assignments"` // code → group_id
}

// GroupsResponse wraps GroupsData.
type GroupsResponse struct {
	OK   bool       `json:"ok"`
	Data GroupsData `json:"data"`
}

// GetGroups returns all watchlist groups and assignments.
// @Summary      获取自选股分组列表
// @Description  返回所有自选股分组及股票分配关系
// @Tags         watchlist-analysis
// @Produce      json
// @Success      200  {object}  GroupsResponse
// @Router       /api/watchlist-groups [get]
func (h *WatchlistAnalysisHandler) GetGroups(c *gin.Context) {
	groups, err := h.loadGroups(c.Request.Context())
	if err != nil {
		h.log.Warn("get groups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	assignments, err := h.loadAssignments(c.Request.Context())
	if err != nil {
		h.log.Warn("get assignments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, GroupsResponse{
		OK: true,
		Data: GroupsData{
			Groups:      groups,
			Assignments: assignments,
		},
	})
}

type createGroupRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// CreateGroup creates a new watchlist group.
// @Summary      创建自选股分组
// @Description  创建一个新的自选股分组
// @Tags         watchlist-analysis
// @Accept       json
// @Produce      json
// @Param        body  body  createGroupRequest  true  "分组数据"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/watchlist-groups [post]
func (h *WatchlistAnalysisHandler) CreateGroup(c *gin.Context) {
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid request body"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing group name"})
		return
	}
	color := strings.TrimSpace(req.Color)
	if color == "" {
		color = "#3b82f6"
	}

	// Generate next group ID
	groupID, err := h.nextGroupID(c.Request.Context())
	if err != nil {
		h.log.Warn("create group: next id", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	_, err = h.db.Exec(c.Request.Context(),
		`INSERT INTO watchlist_groups (id, name, color) VALUES ($1, $2, $3)`,
		groupID, name, color,
	)
	if err != nil {
		h.log.Warn("create group: insert", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	h.log.Info("created watchlist group", zap.String("id", groupID), zap.String("name", name))
	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"group": WatchlistGroup{ID: groupID, Name: name, Color: color},
	})
}

type updateGroupRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

// UpdateGroup updates a watchlist group's name or color.
// @Summary      更新自选股分组
// @Description  更新指定分组的名称或颜色
// @Tags         watchlist-analysis
// @Accept       json
// @Produce      json
// @Param        id    path  string              true  "分组ID"
// @Param        body  body  updateGroupRequest   true  "更新数据"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/watchlist-groups/{id} [put]
func (h *WatchlistAnalysisHandler) UpdateGroup(c *gin.Context) {
	groupID := c.Param("id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing group id"})
		return
	}

	var req updateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid request body"})
		return
	}

	// Check group exists
	var group WatchlistGroup
	err := h.db.QueryRow(c.Request.Context(),
		`SELECT id, name, color FROM watchlist_groups WHERE id = $1`, groupID,
	).Scan(&group.ID, &group.Name, &group.Color)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "group not found"})
		return
	}

	if req.Name != nil {
		group.Name = strings.TrimSpace(*req.Name)
	}
	if req.Color != nil {
		group.Color = strings.TrimSpace(*req.Color)
	}

	_, err = h.db.Exec(c.Request.Context(),
		`UPDATE watchlist_groups SET name = $1, color = $2 WHERE id = $3`,
		group.Name, group.Color, group.ID,
	)
	if err != nil {
		h.log.Warn("update group", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	h.log.Info("updated watchlist group", zap.String("id", groupID))
	c.JSON(http.StatusOK, gin.H{"ok": true, "group": group})
}

// DeleteGroup deletes a watchlist group and unassigns all its stocks.
// @Summary      删除自选股分组
// @Description  删除指定分组并解除其下所有股票的分组分配
// @Tags         watchlist-analysis
// @Produce      json
// @Param        id  path  string  true  "分组ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/watchlist-groups/{id} [delete]
func (h *WatchlistAnalysisHandler) DeleteGroup(c *gin.Context) {
	groupID := c.Param("id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing group id"})
		return
	}

	// Check group exists
	var name string
	err := h.db.QueryRow(c.Request.Context(),
		`SELECT name FROM watchlist_groups WHERE id = $1`, groupID,
	).Scan(&name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "group not found"})
		return
	}

	// Delete assignments first, then the group
	tx, err := h.db.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	defer tx.Rollback(c.Request.Context())

	_, _ = tx.Exec(c.Request.Context(),
		`DELETE FROM watchlist_group_assignments WHERE group_id = $1`, groupID,
	)
	_, err = tx.Exec(c.Request.Context(),
		`DELETE FROM watchlist_groups WHERE id = $1`, groupID,
	)
	if err != nil {
		h.log.Warn("delete group", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	h.log.Info("deleted watchlist group", zap.String("id", groupID), zap.String("name", name))
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "deleted group " + name})
}

type assignGroupRequest struct {
	Code    string  `json:"code"`
	GroupID *string `json:"group_id"` // null to unassign
}

// AssignStock assigns or unassigns a stock to/from a group.
// @Summary      分配股票到分组
// @Description  将股票分配到指定分组，或取消分配（group_id为空时）
// @Tags         watchlist-analysis
// @Accept       json
// @Produce      json
// @Param        body  body  assignGroupRequest  true  "分配数据"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/watchlist-groups/assign [post]
func (h *WatchlistAnalysisHandler) AssignStock(c *gin.Context) {
	var req assignGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid request body"})
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing stock code"})
		return
	}

	if req.GroupID == nil || *req.GroupID == "" {
		// Unassign
		_, _ = h.db.Exec(c.Request.Context(),
			`DELETE FROM watchlist_group_assignments WHERE code = $1`, code,
		)
		h.log.Info("unassigned stock from group", zap.String("code", code))
	} else {
		// Validate group exists
		var groupName string
		err := h.db.QueryRow(c.Request.Context(),
			`SELECT name FROM watchlist_groups WHERE id = $1`, *req.GroupID,
		).Scan(&groupName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "group not found"})
			return
		}
		// Upsert assignment
		_, err = h.db.Exec(c.Request.Context(),
			`INSERT INTO watchlist_group_assignments (code, group_id)
			 VALUES ($1, $2)
			 ON CONFLICT (code) DO UPDATE SET group_id = $2`,
			code, *req.GroupID,
		)
		if err != nil {
			h.log.Warn("assign stock", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
			return
		}
		h.log.Info("assigned stock to group", zap.String("code", code), zap.String("group", groupName))
	}

	// Return current assignments
	assignments, _ := h.loadAssignments(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"ok": true, "assignments": assignments})
}

// ---- Helpers ----

// loadWatchlistCodes returns all stock codes in the user's watchlist.
// Alpha300 top 10 are auto-synced into the watchlist by the scheduler.
func (h *WatchlistAnalysisHandler) loadWatchlistCodes(ctx context.Context) ([]string, error) {
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

// loadGroups returns all watchlist groups.
func (h *WatchlistAnalysisHandler) loadGroups(ctx context.Context) ([]WatchlistGroup, error) {
	rows, err := h.db.Query(ctx,
		`SELECT id, name, color FROM watchlist_groups ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]WatchlistGroup, 0)
	for rows.Next() {
		var g WatchlistGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Color); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// loadAssignments returns a map of code → group_id.
func (h *WatchlistAnalysisHandler) loadAssignments(ctx context.Context) (map[string]string, error) {
	rows, err := h.db.Query(ctx,
		`SELECT code, group_id FROM watchlist_group_assignments`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assignments := make(map[string]string)
	for rows.Next() {
		var code, groupID string
		if err := rows.Scan(&code, &groupID); err != nil {
			return nil, err
		}
		assignments[code] = groupID
	}
	return assignments, rows.Err()
}

// nextGroupID generates the next group ID like g1, g2, ...
func (h *WatchlistAnalysisHandler) nextGroupID(ctx context.Context) (string, error) {
	rows, err := h.db.Query(ctx, `SELECT id FROM watchlist_groups`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		existing[id] = true
	}

	for i := 1; ; i++ {
		id := fmt.Sprintf("g%d", i)
		if !existing[id] {
			return id, nil
		}
	}
}
