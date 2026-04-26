package handlers

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

// ReportsHandler handles report-related API endpoints (Module 16).
type ReportsHandler struct {
	db         *pgxpool.Pool
	tencent    *services.TencentService
	eastMoney  *services.EastMoneyService
	analyze    *AnalyzeHandler
	watchlist  *WatchlistHandler
	log        *zap.Logger
	reportsDir string

	dailyBriefCache *cache.Cache[dailyBriefPayload]

	lastGenTime time.Time
	genMu       sync.Mutex
}

// NewReportsHandler creates a new ReportsHandler.
func NewReportsHandler(
	db *pgxpool.Pool,
	tencent *services.TencentService,
	eastMoney *services.EastMoneyService,
	analyze *AnalyzeHandler,
	watchlist *WatchlistHandler,
	log *zap.Logger,
) *ReportsHandler {
	dir := "/home/finn/.uzi-skill/reports"
	os.MkdirAll(dir, 0o755)

	return &ReportsHandler{
		db:              db,
		tencent:         tencent,
		eastMoney:       eastMoney,
		analyze:         analyze,
		watchlist:       watchlist,
		log:             log,
		reportsDir:      dir,
		dailyBriefCache: cache.New[dailyBriefPayload](),
	}
}

// ==================== Data Types ====================

type reportItem struct {
	Filename string `json:"filename"`
	Date     string `json:"date"`
	Size     int64  `json:"size"`
	Mtime    string `json:"mtime"`
	Type     string `json:"type"`
	Preview  string `json:"preview"`
}

type dailyBriefPayload struct {
	Ok          bool             `json:"ok"`
	Market      dailyBriefMarket `json:"market"`
	Sectors     []briefSector    `json:"sectors"`
	Watchlist   briefWatchlist   `json:"watchlist"`
	GeneratedAt string           `json:"generated_at"`
}

type dailyBriefMarket struct {
	Indices []briefIndex `json:"indices"`
	Breadth interface{}  `json:"breadth"`
}

type briefIndex struct {
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
}

type briefSector struct {
	Name      string  `json:"name"`
	ChangePct float64 `json:"change_pct"`
}

type briefWatchlist struct {
	Count  int          `json:"count"`
	Top    []briefStock `json:"top"`
	Bottom []briefStock `json:"bottom"`
}

type briefStock struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
}

// ==================== Helpers ====================

// safePath resolves and validates the filename stays within reportsDir.
func (h *ReportsHandler) safePath(filename string) (string, bool) {
	clean := filepath.Clean(filename)
	if strings.Contains(clean, "..") || filepath.IsAbs(clean) {
		return "", false
	}
	full := filepath.Join(h.reportsDir, clean)
	if !strings.HasPrefix(full, h.reportsDir) {
		return "", false
	}
	return full, true
}

// previewLine reads the first non-header, non-empty line from a file, truncated to 200 chars.
func previewLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if len(line) > 200 {
			return line[:200]
		}
		return line
	}
	return ""
}

// loadWatchlistCodes returns all stock codes in the watchlist.
func (h *ReportsHandler) loadWatchlistCodes(ctx context.Context) ([]string, error) {
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

// classifyReport determines report type from filename.
func classifyReport(filename string) string {
	if strings.HasPrefix(filename, "daily_report_") {
		return "daily_report"
	}
	if strings.Contains(filename, "8dim") {
		return "8dim_analysis"
	}
	return "other"
}

// extractDate extracts a date string from a report filename.
func extractDate(filename string) string {
	base := strings.TrimSuffix(filename, ".md")
	parts := strings.Split(base, "_")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// ==================== API Handlers ====================

// RedirectToAPI redirects GET /reports to GET /api/reports for backward compatibility.
// @Summary      重定向到API报告列表
// @Description  将 /reports 重定向到 /api/reports 以保持向后兼容
// @Tags         reports
// @Router       /reports [get]
func (h *ReportsHandler) RedirectToAPI(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/api/reports")
}

// ListReports returns all reports in the reports directory.
// GET /api/reports
// @Summary      获取报告列表
// @Description  返回 reports 目录下的所有报告文件列表
// @Tags         reports
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/reports [get]
func (h *ReportsHandler) ListReports(c *gin.Context) {
	entries, err := os.ReadDir(h.reportsDir)
	if err != nil {
		h.log.Warn("failed to read reports dir", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"ok": true, "reports": []reportItem{}})
		return
	}

	reports := make([]reportItem, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		reports = append(reports, reportItem{
			Filename: entry.Name(),
			Date:     extractDate(entry.Name()),
			Size:     info.Size(),
			Mtime:    info.ModTime().Format("2006-01-02 15:04:05"),
			Type:     classifyReport(entry.Name()),
			Preview:  previewLine(filepath.Join(h.reportsDir, entry.Name())),
		})
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Mtime > reports[j].Mtime
	})

	c.JSON(http.StatusOK, gin.H{"ok": true, "reports": reports})
}

// GetReport returns the content of a specific report.
// GET /api/reports/:filename
// @Summary      获取报告详情
// @Description  根据文件名获取报告的完整内容
// @Tags         reports
// @Produce      json
// @Param        filename  path  string  true  "报告文件名"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/reports/{filename} [get]
func (h *ReportsHandler) GetReport(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "filename required"})
		return
	}

	path, ok := h.safePath(filename)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid filename"})
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "report not found"})
			return
		}
		h.log.Warn("failed to read report", zap.String("path", path), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "failed to read report"})
		return
	}

	info, _ := os.Stat(path)
	mtime := ""
	var size int64
	if info != nil {
		mtime = info.ModTime().Format("2006-01-02 15:04:05")
		size = info.Size()
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":       true,
		"filename": filename,
		"content":  string(data),
		"size":     size,
		"mtime":    mtime,
	})
}

// reportFile holds info about a report file on disk.
type reportFile struct {
	name    string
	path    string
	modTime time.Time
	size    int64
}

// DailyReportLatest returns the most recent daily report.
// GET /api/daily-report/latest
// @Summary      获取最新日报
// @Description  返回最近生成的一份日报内容
// @Tags         reports
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/daily-report/latest [get]
func (h *ReportsHandler) DailyReportLatest(c *gin.Context) {
	entries, err := os.ReadDir(h.reportsDir)
	if err != nil || len(entries) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "no reports found"})
		return
	}

	var latest *reportFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "daily_report_") || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		rf := reportFile{
			name:    entry.Name(),
			path:    filepath.Join(h.reportsDir, entry.Name()),
			modTime: info.ModTime(),
			size:    info.Size(),
		}
		if latest == nil || rf.modTime.After(latest.modTime) {
			latest = &rf
		}
	}

	if latest == nil {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "no daily reports found"})
		return
	}

	data, err := os.ReadFile(latest.path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "failed to read report"})
		return
	}

	date := strings.TrimPrefix(latest.name, "daily_report_")
	date = strings.TrimSuffix(date, ".md")

	c.JSON(http.StatusOK, gin.H{
		"ok":       true,
		"filename": latest.name,
		"date":     date,
		"size":     latest.size,
		"mtime":    latest.modTime.Format("2006-01-02 15:04:05"),
		"content":  string(data),
	})
}

// DailyReportList lists all daily reports.
// GET /api/daily-report/list
// @Summary      获取日报列表
// @Description  列出所有已生成的日报
// @Tags         reports
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/daily-report/list [get]
func (h *ReportsHandler) DailyReportList(c *gin.Context) {
	entries, err := os.ReadDir(h.reportsDir)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": true, "reports": []reportItem{}})
		return
	}

	reports := make([]reportItem, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "daily_report_") || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		date := strings.TrimPrefix(entry.Name(), "daily_report_")
		date = strings.TrimSuffix(date, ".md")

		reports = append(reports, reportItem{
			Filename: entry.Name(),
			Date:     date,
			Size:     info.Size(),
			Mtime:    info.ModTime().Format("2006-01-02 15:04:05"),
			Type:     "daily_report",
			Preview:  previewLine(filepath.Join(h.reportsDir, entry.Name())),
		})
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Date > reports[j].Date
	})

	c.JSON(http.StatusOK, gin.H{"ok": true, "reports": reports})
}

// DailyReportGenerate generates a new daily report from watchlist analysis.
// POST /api/daily-report/generate
// @Summary      生成日报
// @Description  基于自选股分析生成新的日报（有10分钟冷却时间）
// @Tags         reports
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/daily-report/generate [post]
func (h *ReportsHandler) DailyReportGenerate(c *gin.Context) {
	// 10-minute cooldown
	h.genMu.Lock()
	if time.Since(h.lastGenTime) < 10*time.Minute {
		remaining := int((10*time.Minute - time.Since(h.lastGenTime)).Seconds())
		h.genMu.Unlock()
		c.JSON(http.StatusOK, gin.H{
			"ok":                 false,
			"error":              fmt.Sprintf("please wait %d seconds", remaining),
			"cooldown_remaining": remaining,
		})
		return
	}
	h.genMu.Unlock()

	ctx := c.Request.Context()

	// Load watchlist
	codes, err := h.loadWatchlistCodes(ctx)
	if err != nil {
		h.log.Warn("daily report: failed to load watchlist", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": "failed to load watchlist"})
		return
	}
	if len(codes) == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": "watchlist is empty"})
		return
	}

	// Parallel analysis
	type analysisResult struct {
		code  string
		name  string
		price float64
		chg   float64
		score float64
		sig   string
	}
	results := make([]analysisResult, 0, len(codes))
	var mu sync.Mutex
	var wg sync.WaitGroup
	errCount := 0

	sem := make(chan struct{}, 8)
	for _, code := range codes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			quote, qErr := h.tencent.FetchQuote(ctx, code)
			if qErr != nil {
				h.log.Warn("daily report: quote failed", zap.String("code", code), zap.Error(qErr))
				mu.Lock()
				errCount++
				mu.Unlock()
				return
			}

			// Simple scoring
			score := 50.0
			if quote.ChangePercent > 0 {
				score += quote.ChangePercent * 2
			} else {
				score += quote.ChangePercent * 1.5
			}
			if score > 100 {
				score = 100
			}
			if score < 0 {
				score = 0
			}

			signal := "neutral"
			if score >= 70 {
				signal = "bullish"
			} else if score <= 40 {
				signal = "bearish"
			}

			mu.Lock()
			results = append(results, analysisResult{
				code:  code,
				name:  quote.Name,
				price: quote.Price,
				chg:   quote.ChangePercent,
				score: score,
				sig:   signal,
			})
			mu.Unlock()
		}(code)
	}
	wg.Wait()

	// Sort by score desc
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Generate markdown
	now := time.Now()
	dateDisplay := now.Format("2006年01月02日")
	var lines []string

	lines = append(lines, fmt.Sprintf("# 📊 AlphaPulse Daily Report — %s", dateDisplay))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("> Generated: %s  ", now.Format("2006-01-02 15:04:05")))
	lines = append(lines, fmt.Sprintf("> Stocks analyzed: %d", len(results)))
	lines = append(lines, "")

	// Ranking table
	lines = append(lines, "## 📈 Score Ranking")
	lines = append(lines, "")
	lines = append(lines, "| # | Stock | Code | Price | Change | Score | Signal |")
	lines = append(lines, "|---|-------|------|-------|--------|-------|--------|")

	for i, r := range results {
		arrow := "⚪"
		if r.chg > 0 {
			arrow = "🔴"
		} else if r.chg < 0 {
			arrow = "🟢"
		}

		lines = append(lines, fmt.Sprintf("| %d | %s | %s | %.2f | %s %.2f%% | %.1f | %s |",
			i+1, r.name, r.code, r.price, arrow, r.chg, r.score, r.sig))
	}
	lines = append(lines, "")

	// Summary
	if len(results) > 0 {
		top := results[0]
		bottom := results[len(results)-1]
		lines = append(lines, "## 💡 Summary")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- **Top pick**: %s (%s) — score %.1f", top.name, top.code, top.score))
		lines = append(lines, fmt.Sprintf("- **Watch out**: %s (%s) — score %.1f", bottom.name, bottom.code, bottom.score))
	}

	lines = append(lines, "")
	lines = append(lines, "---")
	lines = append(lines, "*Auto-generated by AlphaPulse analysis system. For reference only.*")

	mdContent := strings.Join(lines, "\n")

	// Save to file
	todayStr := now.Format("20060102")
	filename := fmt.Sprintf("daily_report_%s.md", todayStr)
	path := filepath.Join(h.reportsDir, filename)

	if err := os.WriteFile(path, []byte(mdContent), 0o644); err != nil {
		h.log.Warn("daily report: failed to write file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "failed to save report"})
		return
	}

	h.genMu.Lock()
	h.lastGenTime = time.Now()
	h.genMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"ok":              true,
		"filename":        filename,
		"date":            todayStr,
		"stocks_analyzed": len(results),
		"errors_count":    errCount,
		"content":         mdContent,
	})
}

// DailyBrief returns an aggregate daily market brief.
// GET /api/daily-brief
// @Summary      获取每日市场简报
// @Description  返回包含大盘指数、涨跌家数、板块、自选股的综合市场简报
// @Tags         reports
// @Produce      json
// @Success      200  {object}  dailyBriefPayload
// @Router       /api/daily-brief [get]
func (h *ReportsHandler) DailyBrief(c *gin.Context) {
	// Check cache
	if cached, ok := h.dailyBriefCache.Get("daily-brief"); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	var (
		indices []briefIndex
		breadth interface{}
		sectors []briefSector
		wlTop   []briefStock
		wlBot   []briefStock
		wlCount int
	)

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Fetch indices via overview
	wg.Add(1)
	go func() {
		defer wg.Done()
		overview, err := h.eastMoney.FetchOverview(ctx)
		if err != nil {
			h.log.Warn("daily-brief: indices fetch failed", zap.Error(err))
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for _, idx := range overview.Indices {
			indices = append(indices, briefIndex{
				Name:      idx.Name,
				Price:     idx.Price,
				ChangePct: idx.ChangePercent,
			})
		}
	}()

	// Fetch breadth
	wg.Add(1)
	go func() {
		defer wg.Done()
		b, err := h.eastMoney.FetchMarketBreadth(ctx)
		if err != nil {
			h.log.Warn("daily-brief: breadth fetch failed", zap.Error(err))
			return
		}
		mu.Lock()
		breadth = b
		mu.Unlock()
	}()

	// Fetch sectors
	wg.Add(1)
	go func() {
		defer wg.Done()
		s, err := h.eastMoney.FetchSectors(ctx)
		if err != nil {
			h.log.Warn("daily-brief: sectors fetch failed", zap.Error(err))
			return
		}
		mu.Lock()
		for _, sec := range s {
			sectors = append(sectors, briefSector{
				Name:      sec.Name,
				ChangePct: sec.ChangePercent,
			})
		}
		mu.Unlock()
	}()

	// Fetch watchlist summary
	wg.Add(1)
	go func() {
		defer wg.Done()
		codes, err := h.loadWatchlistCodes(ctx)
		if err != nil || len(codes) == 0 {
			return
		}
		wlCount = len(codes)

		type wlResult struct {
			stock briefStock
		}
		var results []wlResult
		var rMu sync.Mutex
		var wG sync.WaitGroup

		limit := 20
		if len(codes) < limit {
			limit = len(codes)
		}
		for _, code := range codes[:limit] {
			wG.Add(1)
			go func(code string) {
				defer wG.Done()
				q, err := h.tencent.FetchQuote(ctx, code)
				if err != nil {
					return
				}
				rMu.Lock()
				results = append(results, wlResult{
					stock: briefStock{
						Code:      code,
						Name:      q.Name,
						Price:     q.Price,
						ChangePct: q.ChangePercent,
					},
				})
				rMu.Unlock()
			}(code)
		}
		wG.Wait()

		sort.Slice(results, func(i, j int) bool {
			return results[i].stock.ChangePct > results[j].stock.ChangePct
		})

		mu.Lock()
		topN := 5
		if len(results) < topN {
			topN = len(results)
		}
		for _, r := range results[:topN] {
			wlTop = append(wlTop, r.stock)
		}
		botN := 5
		if len(results) < botN {
			botN = len(results)
		}
		start := len(results) - botN
		if start < 0 {
			start = 0
		}
		for _, r := range results[start:] {
			wlBot = append(wlBot, r.stock)
		}
		mu.Unlock()
	}()

	wg.Wait()

	payload := dailyBriefPayload{
		Ok: true,
		Market: dailyBriefMarket{
			Indices: indices,
			Breadth: breadth,
		},
		Sectors:     sectors,
		Watchlist:   briefWatchlist{Count: wlCount, Top: wlTop, Bottom: wlBot},
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	h.dailyBriefCache.Set("daily-brief", payload, 60*time.Second)
	c.JSON(http.StatusOK, payload)
}
