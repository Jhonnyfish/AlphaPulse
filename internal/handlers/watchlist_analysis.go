package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"alphapulse/internal/cache"
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
	analyze   *AnalyzeHandler
	log       *zap.Logger

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
		heatmapCache: cache.New[[]HeatmapItem](),
		sectorsCache: cache.New[SectorsResponse](),
		rankingCache: cache.New[RankingResponse](),
	}
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
			quote, err := h.tencent.FetchQuote(context.Background(), cd)
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
	Error           string             `json:"error,omitempty"`
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
func (h *WatchlistAnalysisHandler) Ranking(c *gin.Context) {
	if cached, ok := h.rankingCache.Get("all"); ok {
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

	h.rankingCache.Set("all", resp, 180*time.Second)
	c.JSON(http.StatusOK, resp)
}

func (h *WatchlistAnalysisHandler) analyzeForRanking(ctx context.Context, code string) RankingItem {
	analysis := h.analyze.analyzeSingle(ctx, code)
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

	dimScores := make(map[string]float64)
	dimScores["order_flow"] = scoreDimension(analysis.OrderFlow.Verdict, analysis.OrderFlow.OuterRatio > 0.55)
	dimScores["volume_price"] = scoreDimension(analysis.VolumePrice.Verdict, analysis.VolumePrice.VolumeRatio > 1.2)
	dimScores["valuation"] = scoreDimension(analysis.Valuation.Verdict, false)
	dimScores["volatility"] = scoreDimension(analysis.Volatility.Verdict, false)
	dimScores["money_flow"] = scoreDimension(analysis.MoneyFlow.Verdict, analysis.MoneyFlow.TodayMainNet > 0)
	dimScores["technical"] = scoreDimension(analysis.Technical.Verdict, false)
	dimScores["sector"] = scoreDimension(analysis.Sector.Verdict, analysis.Sector.IsSectorLeader)
	dimScores["sentiment"] = scoreDimension(analysis.Sentiment.Verdict, analysis.Sentiment.SentimentScore > 0)

	return RankingItem{
		Code:            services.StockCode6(code),
		Name:            analysis.Name,
		OverallScore:    analysis.Summary.OverallScore,
		OverallSignal:   analysis.Summary.OverallSignal,
		DimensionScores: dimScores,
		ChangePct:       analysis.Quote.ChangePercent,
		Price:           analysis.Quote.Price,
		Strengths:       analysis.Summary.Strengths,
		Risks:           analysis.Summary.Risks,
	}
}

// scoreDimension maps a verdict string to a 0-100 score.
func scoreDimension(verdict string, isPositive bool) float64 {
	switch {
	case strings.Contains(verdict, "强势") || strings.Contains(verdict, "优秀") || strings.Contains(verdict, "强烈"):
		return 85
	case strings.Contains(verdict, "偏多") || strings.Contains(verdict, "良好") || strings.Contains(verdict, "积极"):
		return 70
	case strings.Contains(verdict, "中性") || strings.Contains(verdict, "均衡") || strings.Contains(verdict, "正常"):
		return 50
	case strings.Contains(verdict, "偏空") || strings.Contains(verdict, "谨慎") || strings.Contains(verdict, "较低"):
		return 35
	case strings.Contains(verdict, "弱势") || strings.Contains(verdict, "危险") || strings.Contains(verdict, "极低"):
		return 20
	default:
		if isPositive {
			return 60
		}
		return 50
	}
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
