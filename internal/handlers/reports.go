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
	"alphapulse/internal/models"
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
	deepseek   *services.DeepSeekClient

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
	deepseek *services.DeepSeekClient,
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
		deepseek:        deepseek,
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
// Alpha300 top 10 are auto-synced into the watchlist by the scheduler.
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

	// Use analyzeSingle for each stock (parallel, max 4 concurrent)
	type analysisResult struct {
		analysis models.StockAnalysis
		err      error
	}
	results := make([]analysisResult, len(codes))
	var wg sync.WaitGroup
	errCount := 0
	sem := make(chan struct{}, 4)

	for i, code := range codes {
		wg.Add(1)
		go func(idx int, stockCode string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			analysis := h.analyze.analyzeSingle(ctx, stockCode)
			results[idx] = analysisResult{analysis: analysis}
		}(i, code)
	}
	wg.Wait()

	// Filter successful results
	validResults := make([]models.StockAnalysis, 0, len(results))
	for _, r := range results {
		if r.err == nil && r.analysis.Summary.OverallScore > 0 {
			validResults = append(validResults, r.analysis)
		} else {
			errCount++
		}
	}

	if len(validResults) == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": "no valid analysis results"})
		return
	}

	// Sort by overall score descending
	sort.Slice(validResults, func(i, j int) bool {
		return validResults[i].Summary.OverallScore > validResults[j].Summary.OverallScore
	})

	// Generate markdown
	now := time.Now()
	dateDisplay := now.Format("2006年01月02日")
	var lines []string

	lines = append(lines, fmt.Sprintf("# 📊 AlphaPulse 每日研报 — %s", dateDisplay))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("> 生成时间: %s  ", now.Format("2006-01-02 15:04:05")))
	lines = append(lines, fmt.Sprintf("> 分析股票数: %d", len(validResults)))
	lines = append(lines, "")

	// Market overview
	overview, ovErr := h.eastMoney.FetchOverview(ctx)
	if ovErr == nil && len(overview.Indices) > 0 {
		lines = append(lines, "## 🏛️ 大盘概况")
		lines = append(lines, "")
		lines = append(lines, "| 指数 | 最新价 | 涨跌幅 |")
		lines = append(lines, "|------|--------|--------|")
		for _, idx := range overview.Indices {
			arrow := "⚪"
			if idx.ChangePercent > 0 {
				arrow = "🔴"
			} else if idx.ChangePercent < 0 {
				arrow = "🟢"
			}
			lines = append(lines, fmt.Sprintf("| %s | %.2f | %s %.2f%% |", idx.Name, idx.Price, arrow, idx.ChangePercent))
		}
		lines = append(lines, "")
	}

	// Sector top movers
	sectors, secErr := h.eastMoney.FetchSectors(ctx)
	if secErr == nil && len(sectors) > 0 {
		sort.Slice(sectors, func(i, j int) bool { return sectors[i].ChangePercent > sectors[j].ChangePercent })
		topN := 5
		if len(sectors) < topN {
			topN = len(sectors)
		}
		lines = append(lines, "## 🔥 板块轮动")
		lines = append(lines, "")
		lines = append(lines, "**领涨板块:**")
		for i := 0; i < topN; i++ {
			s := sectors[i]
			lines = append(lines, fmt.Sprintf("- %s (%.2f%%)", s.Name, s.ChangePercent))
		}
		if len(sectors) > 5 {
			lines = append(lines, "")
			lines = append(lines, "**领跌板块:**")
			bottomN := len(sectors) - 5
			if bottomN > 5 {
				bottomN = 5
			}
			for i := 0; i < bottomN; i++ {
				s := sectors[len(sectors)-1-i]
				lines = append(lines, fmt.Sprintf("- %s (%.2f%%)", s.Name, s.ChangePercent))
			}
		}
		lines = append(lines, "")
	}

	// Enhanced score ranking table with 8-dimension scores
	lines = append(lines, "## 📈 综合评分排名")
	lines = append(lines, "")
	lines = append(lines, "| 排名 | 股票 | 代码 | 现价 | 涨跌幅 | 综合评分 | 信号 | 技术面 | 资金面 | 量价 | 估值 |")
	lines = append(lines, "|------|------|------|------|--------|----------|------|--------|--------|------|------|")

	for i, a := range validResults {
		arrow := "⚪"
		if a.Quote.ChangePercent > 0 {
			arrow = "🔴"
		} else if a.Quote.ChangePercent < 0 {
			arrow = "🟢"
		}
		sigLabel := "中性"
		if a.Summary.OverallSignal == "bullish" {
			sigLabel = "偏多"
		} else if a.Summary.OverallSignal == "bearish" {
			sigLabel = "偏空"
		}

		// Get dimension scores
		techScore := a.Technical.MACD_Signal
		if techScore == "" {
			techScore = "-"
		}
		fundScore := a.MoneyFlow.TodayMainDirection
		if fundScore == "" {
			fundScore = "-"
		}
		vpScore := a.VolumePrice.PriceVolumeHarmony
		if vpScore == "" {
			vpScore = "-"
		}
		valScore := a.Valuation.PELevel
		if valScore == "" {
			valScore = "-"
		}

		lines = append(lines, fmt.Sprintf("| %d | %s | %s | %.2f | %s %.2f%% | %d | %s | %s | %s | %s | %s |",
			i+1, a.Name, a.Code, a.Quote.Price, arrow, a.Quote.ChangePercent,
			a.Summary.OverallScore, sigLabel, techScore, fundScore, vpScore, valScore))
	}
	lines = append(lines, "")

	// Summary section
	if len(validResults) > 0 {
		top := validResults[0]
		bottom := validResults[len(validResults)-1]
		bullCount := 0
		bearCount := 0
		for _, a := range validResults {
			if a.Summary.OverallSignal == "bullish" {
				bullCount++
			} else if a.Summary.OverallSignal == "bearish" {
				bearCount++
			}
		}
		lines = append(lines, "## 💡 总结")
		lines = append(lines, "")
		topSig := "中性"
		if top.Summary.OverallSignal == "bullish" {
			topSig = "偏多"
		}
		botSig := "中性"
		if bottom.Summary.OverallSignal == "bearish" {
			botSig = "偏空"
		}
		lines = append(lines, fmt.Sprintf("- **最看好**: %s (%s) — 评分 %d，信号: %s", top.Name, top.Code, top.Summary.OverallScore, topSig))
		lines = append(lines, fmt.Sprintf("- **需关注**: %s (%s) — 评分 %d，信号: %s", bottom.Name, bottom.Code, bottom.Summary.OverallScore, botSig))
		if bearCount > 0 {
			var bearNames []string
			for _, a := range validResults {
				if a.Summary.OverallSignal == "bearish" {
					bearNames = append(bearNames, a.Name)
				}
			}
			lines = append(lines, fmt.Sprintf("- **弱势股 (%d只)**: %s", bearCount, strings.Join(bearNames, ", ")))
		}
	}

	// Individual stock details (top 3)
	lines = append(lines, "")
	lines = append(lines, "## 📋 重点个股分析")
	lines = append(lines, "")

	topN := 3
	if len(validResults) < topN {
		topN = len(validResults)
	}
	for i := 0; i < topN; i++ {
		a := validResults[i]
		lines = append(lines, fmt.Sprintf("### %d. %s (%s)", i+1, a.Name, a.Code))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- **现价**: %.2f | **涨跌幅**: %.2f%%", a.Quote.Price, a.Quote.ChangePercent))
		lines = append(lines, fmt.Sprintf("- **综合评分**: %d | **信号**: %s", a.Summary.OverallScore, a.Summary.OverallSignal))
		lines = append(lines, "")

		// Technical indicators
		lines = append(lines, "**技术指标:**")
		if a.Technical.MACD_Signal != "" {
			lines = append(lines, fmt.Sprintf("- MACD: %s (DIF: %.2f, DEA: %.2f)", a.Technical.MACD_Signal, a.Technical.MACD_DIF, a.Technical.MACD_DEA))
		}
		if a.Technical.KDJ_Signal != "" {
			lines = append(lines, fmt.Sprintf("- KDJ: %s (K: %.2f, D: %.2f, J: %.2f)", a.Technical.KDJ_Signal, a.Technical.KDJ_K, a.Technical.KDJ_D, a.Technical.KDJ_J))
		}
		if a.Technical.RSI_Level != "" {
			lines = append(lines, fmt.Sprintf("- RSI: %s (%.2f)", a.Technical.RSI_Level, a.Technical.RSI_14))
		}
		if a.Technical.MAArrangement != "" {
			lines = append(lines, fmt.Sprintf("- 均线: %s", a.Technical.MAArrangement))
		}
		lines = append(lines, "")

		// Money flow
		if a.MoneyFlow.TodayMainDirection != "" && a.MoneyFlow.TodayMainDirection != "数据不足" {
			lines = append(lines, "**资金流向:**")
			lines = append(lines, fmt.Sprintf("- 主力方向: %s (净额: %.2f万)", a.MoneyFlow.TodayMainDirection, a.MoneyFlow.TodayMainNet))
			if a.MoneyFlow.MainConsecutiveDays > 0 {
				lines = append(lines, fmt.Sprintf("- 连续%s: %d天", a.MoneyFlow.MainConsecutiveDirection, a.MoneyFlow.MainConsecutiveDays))
			}
			lines = append(lines, "")
		}

		// Volume analysis
		if a.VolumePrice.PriceVolumeHarmony != "" {
			lines = append(lines, "**量价分析:**")
			lines = append(lines, fmt.Sprintf("- 量价关系: %s", a.VolumePrice.PriceVolumeHarmony))
			if a.VolumePrice.VolumeRatio > 0 {
				lines = append(lines, fmt.Sprintf("- 量比: %.2f", a.VolumePrice.VolumeRatio))
			}
			lines = append(lines, "")
		}

		// Strengths and risks
		if len(a.Summary.Strengths) > 0 {
			lines = append(lines, "**优势:**")
			for _, s := range a.Summary.Strengths {
				lines = append(lines, fmt.Sprintf("- %s", s))
			}
		}
		if len(a.Summary.Risks) > 0 {
			lines = append(lines, "**风险:**")
			for _, r := range a.Summary.Risks {
				lines = append(lines, fmt.Sprintf("- %s", r))
			}
		}
		lines = append(lines, "")
	}

	lines = append(lines, "")
	lines = append(lines, "---")
	lines = append(lines, "*本报告由 AlphaPulse 八维分析系统自动生成，仅供参考，不构成投资建议。*")

	// AI analysis via DeepSeek (if configured)
	if h.deepseek != nil && h.deepseek.Enabled() {
		// Build a comprehensive prompt with all analysis data
		var analysisData []string
		for _, a := range validResults {
			analysisData = append(analysisData, fmt.Sprintf("%s(%s): 评分%d, 信号%s, 技术面%s, 资金%s, 量价%s, 估值%s",
				a.Name, a.Code, a.Summary.OverallScore, a.Summary.OverallSignal,
				a.Technical.MACD_Signal, a.MoneyFlow.TodayMainDirection,
				a.VolumePrice.PriceVolumeHarmony, a.Valuation.PELevel))
		}

		aiPrompt := fmt.Sprintf(`以下是今日A股市场数据和自选股八维分析结果，请用中文写一段专业的市场分析（400字以内），包括：
1) 大盘走势判断
2) 板块轮动特征
3) 自选股重点关注（结合技术形态、资金动向、量价配合）
4) 风险提示

自选股分析数据:
%s

请用专业、客观的语言分析，避免泛泛而谈。`, strings.Join(analysisData, "\n"))

		aiCtx, aiCancel := context.WithTimeout(ctx, 90*time.Second)
		aiAnalysis, aiErr := h.deepseek.Chat(aiCtx, "你是一位资深A股分析师，具备10年以上投研经验。分析要专业、客观、有深度，关注量价配合、资金流向、技术形态等核心指标。", aiPrompt)
		aiCancel()
		if aiErr == nil && aiAnalysis != "" {
			lines = append(lines, "")
			lines = append(lines, "## 🤖 AI 分析")
			lines = append(lines, "")
			lines = append(lines, aiAnalysis)
		} else if aiErr != nil {
			h.log.Warn("daily report: deepseek analysis failed", zap.Error(aiErr))
		}
	}

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

// GenerateDailyReportAuto is called by the scheduler (no HTTP context).
// It reuses the same analysis logic as DailyReportGenerate.
func (h *ReportsHandler) GenerateDailyReportAuto() {
	// Cooldown check
	h.genMu.Lock()
	if time.Since(h.lastGenTime) < 10*time.Minute {
		h.genMu.Unlock()
		h.log.Info("daily report auto: skipped (cooldown)")
		return
	}
	h.genMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	codes, err := h.loadWatchlistCodes(ctx)
	if err != nil {
		h.log.Warn("daily report auto: failed to load watchlist", zap.Error(err))
		return
	}
	if len(codes) == 0 {
		h.log.Warn("daily report auto: watchlist is empty")
		return
	}

	// Use analyzeSingle for each stock (parallel, max 4 concurrent)
	type analysisResult struct {
		analysis models.StockAnalysis
		err      error
	}
	results := make([]analysisResult, len(codes))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)

	for i, code := range codes {
		wg.Add(1)
		go func(idx int, stockCode string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			analysis := h.analyze.analyzeSingle(ctx, stockCode)
			results[idx] = analysisResult{analysis: analysis}
		}(i, code)
	}
	wg.Wait()

	// Filter successful results
	validResults := make([]models.StockAnalysis, 0, len(results))
	for _, r := range results {
		if r.err == nil && r.analysis.Summary.OverallScore > 0 {
			validResults = append(validResults, r.analysis)
		}
	}

	if len(validResults) == 0 {
		h.log.Warn("daily report auto: no valid analysis results")
		return
	}

	// Sort by overall score descending
	sort.Slice(validResults, func(i, j int) bool {
		return validResults[i].Summary.OverallScore > validResults[j].Summary.OverallScore
	})

	// Generate markdown report
	now := time.Now()
	dateDisplay := now.Format("2006年01月02日")
	var lines []string

	lines = append(lines, fmt.Sprintf("# 📊 AlphaPulse 每日研报 — %s", dateDisplay))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("> 生成时间: %s  ", now.Format("2006-01-02 15:04:05")))
	lines = append(lines, fmt.Sprintf("> 分析股票数: %d", len(validResults)))
	lines = append(lines, "")

	// Market overview
	overview, ovErr := h.eastMoney.FetchOverview(ctx)
	if ovErr == nil && len(overview.Indices) > 0 {
		lines = append(lines, "## 🏛️ 大盘概况")
		lines = append(lines, "")
		lines = append(lines, "| 指数 | 最新价 | 涨跌幅 |")
		lines = append(lines, "|------|--------|--------|")
		for _, idx := range overview.Indices {
			arrow := "⚪"
			if idx.ChangePercent > 0 {
				arrow = "🔴"
			} else if idx.ChangePercent < 0 {
				arrow = "🟢"
			}
			lines = append(lines, fmt.Sprintf("| %s | %.2f | %s %.2f%% |", idx.Name, idx.Price, arrow, idx.ChangePercent))
		}
		lines = append(lines, "")
	}

	// Sector top movers
	sectors, secErr := h.eastMoney.FetchSectors(ctx)
	if secErr == nil && len(sectors) > 0 {
		sort.Slice(sectors, func(i, j int) bool { return sectors[i].ChangePercent > sectors[j].ChangePercent })
		topN := 5
		if len(sectors) < topN {
			topN = len(sectors)
		}
		lines = append(lines, "## 🔥 板块轮动")
		lines = append(lines, "")
		lines = append(lines, "**领涨板块:**")
		for i := 0; i < topN; i++ {
			s := sectors[i]
			lines = append(lines, fmt.Sprintf("- %s (%.2f%%)", s.Name, s.ChangePercent))
		}
		if len(sectors) > 5 {
			lines = append(lines, "")
			lines = append(lines, "**领跌板块:**")
			bottomN := len(sectors) - 5
			if bottomN > 5 {
				bottomN = 5
			}
			for i := 0; i < bottomN; i++ {
				s := sectors[len(sectors)-1-i]
				lines = append(lines, fmt.Sprintf("- %s (%.2f%%)", s.Name, s.ChangePercent))
			}
		}
		lines = append(lines, "")
	}

	// Enhanced score ranking table with 8-dimension scores
	lines = append(lines, "## 📈 综合评分排名")
	lines = append(lines, "")
	lines = append(lines, "| 排名 | 股票 | 代码 | 现价 | 涨跌幅 | 综合评分 | 信号 | 技术面 | 资金面 | 量价 | 估值 |")
	lines = append(lines, "|------|------|------|------|--------|----------|------|--------|--------|------|------|")

	for i, a := range validResults {
		arrow := "⚪"
		if a.Quote.ChangePercent > 0 {
			arrow = "🔴"
		} else if a.Quote.ChangePercent < 0 {
			arrow = "🟢"
		}
		sigLabel := "中性"
		if a.Summary.OverallSignal == "bullish" {
			sigLabel = "偏多"
		} else if a.Summary.OverallSignal == "bearish" {
			sigLabel = "偏空"
		}

		// Get dimension scores
		techScore := a.Technical.MACD_Signal
		if techScore == "" {
			techScore = "-"
		}
		fundScore := a.MoneyFlow.TodayMainDirection
		if fundScore == "" {
			fundScore = "-"
		}
		vpScore := a.VolumePrice.PriceVolumeHarmony
		if vpScore == "" {
			vpScore = "-"
		}
		valScore := a.Valuation.PELevel
		if valScore == "" {
			valScore = "-"
		}

		lines = append(lines, fmt.Sprintf("| %d | %s | %s | %.2f | %s %.2f%% | %d | %s | %s | %s | %s | %s |",
			i+1, a.Name, a.Code, a.Quote.Price, arrow, a.Quote.ChangePercent,
			a.Summary.OverallScore, sigLabel, techScore, fundScore, vpScore, valScore))
	}
	lines = append(lines, "")

	// Summary section
	if len(validResults) > 0 {
		top := validResults[0]
		bottom := validResults[len(validResults)-1]
		bullCount := 0
		bearCount := 0
		for _, a := range validResults {
			if a.Summary.OverallSignal == "bullish" {
				bullCount++
			} else if a.Summary.OverallSignal == "bearish" {
				bearCount++
			}
		}
		lines = append(lines, "## 💡 总结")
		lines = append(lines, "")
		topSig := "中性"
		if top.Summary.OverallSignal == "bullish" {
			topSig = "偏多"
		}
		botSig := "中性"
		if bottom.Summary.OverallSignal == "bearish" {
			botSig = "偏空"
		}
		lines = append(lines, fmt.Sprintf("- **最看好**: %s (%s) — 评分 %d，信号: %s", top.Name, top.Code, top.Summary.OverallScore, topSig))
		lines = append(lines, fmt.Sprintf("- **需关注**: %s (%s) — 评分 %d，信号: %s", bottom.Name, bottom.Code, bottom.Summary.OverallScore, botSig))
		if bearCount > 0 {
			var bearNames []string
			for _, a := range validResults {
				if a.Summary.OverallSignal == "bearish" {
					bearNames = append(bearNames, a.Name)
				}
			}
			lines = append(lines, fmt.Sprintf("- **弱势股 (%d只)**: %s", bearCount, strings.Join(bearNames, ", ")))
		}
	}

	// Individual stock details (top 3 + bottom 1)
	lines = append(lines, "")
	lines = append(lines, "## 📋 重点个股分析")
	lines = append(lines, "")

	// Show top 3 stocks with detailed analysis
	topN := 3
	if len(validResults) < topN {
		topN = len(validResults)
	}
	for i := 0; i < topN; i++ {
		a := validResults[i]
		lines = append(lines, fmt.Sprintf("### %d. %s (%s)", i+1, a.Name, a.Code))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- **现价**: %.2f | **涨跌幅**: %.2f%%", a.Quote.Price, a.Quote.ChangePercent))
		lines = append(lines, fmt.Sprintf("- **综合评分**: %d | **信号**: %s", a.Summary.OverallScore, a.Summary.OverallSignal))
		lines = append(lines, "")

		// Technical indicators
		lines = append(lines, "**技术指标:**")
		if a.Technical.MACD_Signal != "" {
			lines = append(lines, fmt.Sprintf("- MACD: %s (DIF: %.2f, DEA: %.2f)", a.Technical.MACD_Signal, a.Technical.MACD_DIF, a.Technical.MACD_DEA))
		}
		if a.Technical.KDJ_Signal != "" {
			lines = append(lines, fmt.Sprintf("- KDJ: %s (K: %.2f, D: %.2f, J: %.2f)", a.Technical.KDJ_Signal, a.Technical.KDJ_K, a.Technical.KDJ_D, a.Technical.KDJ_J))
		}
		if a.Technical.RSI_Level != "" {
			lines = append(lines, fmt.Sprintf("- RSI: %s (%.2f)", a.Technical.RSI_Level, a.Technical.RSI_14))
		}
		if a.Technical.MAArrangement != "" {
			lines = append(lines, fmt.Sprintf("- 均线: %s", a.Technical.MAArrangement))
		}
		lines = append(lines, "")

		// Money flow
		if a.MoneyFlow.TodayMainDirection != "" && a.MoneyFlow.TodayMainDirection != "数据不足" {
			lines = append(lines, "**资金流向:**")
			lines = append(lines, fmt.Sprintf("- 主力方向: %s (净额: %.2f万)", a.MoneyFlow.TodayMainDirection, a.MoneyFlow.TodayMainNet))
			if a.MoneyFlow.MainConsecutiveDays > 0 {
				lines = append(lines, fmt.Sprintf("- 连续%s: %d天", a.MoneyFlow.MainConsecutiveDirection, a.MoneyFlow.MainConsecutiveDays))
			}
			lines = append(lines, "")
		}

		// Volume analysis
		if a.VolumePrice.PriceVolumeHarmony != "" {
			lines = append(lines, "**量价分析:**")
			lines = append(lines, fmt.Sprintf("- 量价关系: %s", a.VolumePrice.PriceVolumeHarmony))
			if a.VolumePrice.VolumeRatio > 0 {
				lines = append(lines, fmt.Sprintf("- 量比: %.2f", a.VolumePrice.VolumeRatio))
			}
			lines = append(lines, "")
		}

		// Strengths and risks
		if len(a.Summary.Strengths) > 0 {
			lines = append(lines, "**优势:**")
			for _, s := range a.Summary.Strengths {
				lines = append(lines, fmt.Sprintf("- %s", s))
			}
		}
		if len(a.Summary.Risks) > 0 {
			lines = append(lines, "**风险:**")
			for _, r := range a.Summary.Risks {
				lines = append(lines, fmt.Sprintf("- %s", r))
			}
		}
		lines = append(lines, "")
	}

	// AI analysis
	if h.deepseek != nil && h.deepseek.Enabled() {
		// Build a comprehensive prompt with all analysis data
		var analysisData []string
		for _, a := range validResults {
			analysisData = append(analysisData, fmt.Sprintf("%s(%s): 评分%d, 信号%s, 技术面%s, 资金%s, 量价%s, 估值%s",
				a.Name, a.Code, a.Summary.OverallScore, a.Summary.OverallSignal,
				a.Technical.MACD_Signal, a.MoneyFlow.TodayMainDirection,
				a.VolumePrice.PriceVolumeHarmony, a.Valuation.PELevel))
		}

		aiPrompt := fmt.Sprintf(`以下是今日A股市场数据和自选股八维分析结果，请用中文写一段专业的市场分析（400字以内），包括：
1) 大盘走势判断
2) 板块轮动特征
3) 自选股重点关注（结合技术形态、资金动向、量价配合）
4) 风险提示

自选股分析数据:
%s

请用专业、客观的语言分析，避免泛泛而谈。`, strings.Join(analysisData, "\n"))

		aiCtx, aiCancel := context.WithTimeout(ctx, 90*time.Second)
		aiAnalysis, aiErr := h.deepseek.Chat(aiCtx, "你是一位资深A股分析师，具备10年以上投研经验。分析要专业、客观、有深度，关注量价配合、资金流向、技术形态等核心指标。", aiPrompt)
		aiCancel()
		if aiErr == nil && aiAnalysis != "" {
			lines = append(lines, "## 🤖 AI 分析")
			lines = append(lines, "")
			lines = append(lines, aiAnalysis)
		} else if aiErr != nil {
			h.log.Warn("daily report auto: deepseek failed", zap.Error(aiErr))
		}
	}

	lines = append(lines, "", "---", "*本报告由 AlphaPulse 八维分析系统自动生成，仅供参考，不构成投资建议。*")
	mdContent := strings.Join(lines, "\n")

	filename := fmt.Sprintf("daily_report_%s.md", now.Format("20060102"))
	path := filepath.Join(h.reportsDir, filename)
	if err := os.WriteFile(path, []byte(mdContent), 0o644); err != nil {
		h.log.Warn("daily report auto: failed to write file", zap.Error(err))
		return
	}

	h.genMu.Lock()
	h.lastGenTime = time.Now()
	h.genMu.Unlock()

	h.log.Info("daily report auto: generated", zap.String("filename", filename), zap.Int("stocks", len(validResults)))
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

	// Helper: run a fetch function with a per-task timeout
	runWithTimeout := func(timeout time.Duration, fn func(context.Context)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			taskCtx, taskCancel := context.WithTimeout(ctx, timeout)
			defer taskCancel()
			fn(taskCtx)
		}()
	}

	// Fetch indices via overview (5s timeout)
	runWithTimeout(5*time.Second, func(tCtx context.Context) {
		overview, err := h.eastMoney.FetchOverview(tCtx)
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
	})

	// Fetch breadth (5s timeout)
	runWithTimeout(5*time.Second, func(tCtx context.Context) {
		b, err := h.eastMoney.FetchMarketBreadth(tCtx)
		if err != nil {
			h.log.Warn("daily-brief: breadth fetch failed", zap.Error(err))
			return
		}
		mu.Lock()
		breadth = b
		mu.Unlock()
	})

	// Fetch sectors (5s timeout)
	runWithTimeout(5*time.Second, func(tCtx context.Context) {
		s, err := h.eastMoney.FetchSectors(tCtx)
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
	})

	// Fetch watchlist summary (8s timeout — needs Alpha300 + quotes)
	runWithTimeout(8*time.Second, func(tCtx context.Context) {
		codes, err := h.loadWatchlistCodes(tCtx)
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
				q, err := h.tencent.FetchQuote(tCtx, code)
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
	})

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

	h.dailyBriefCache.Set("daily-brief", payload, 300*time.Second)
	c.JSON(http.StatusOK, payload)
}
