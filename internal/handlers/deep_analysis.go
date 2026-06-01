package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"alphapulse/internal/logger"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DeepAnalysisRequest is the request for deep analysis.
type DeepAnalysisRequest struct {
	Code string `json:"code" binding:"required"`
}

// DeepAnalysisResponse is the response for deep analysis.
type DeepAnalysisResponse struct {
	OK      bool    `json:"ok"`
	Code    string  `json:"code"`
	Status  string  `json:"status,omitempty"`  // "running", "completed", "failed"
	Report  string  `json:"report,omitempty"`
	Error   string  `json:"error,omitempty"`
	PctDone *string `json:"pct_done,omitempty"` // progress indicator
}

// CommodityData holds futures price data for a commodity.
type CommodityData struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	LatestDate string `json:"latest_date"`
	Close     float64 `json:"close"`
	PrevClose float64 `json:"prev_close"`
	Change5D  float64 `json:"change_5d"`
	Change20D float64 `json:"change_20d"`
	Trend     string  `json:"trend"`
}

// IndustryCommodityMap maps industry keywords to relevant commodity futures.
var IndustryCommodityMap = map[string][]struct {
	Code string
	Name string
}{
	"锂电":   {{Code: "LC.GFE", Name: "碳酸锂"}},
	"电解液":  {{Code: "LC.GFE", Name: "碳酸锂"}},
	"锂电池":  {{Code: "LC.GFE", Name: "碳酸锂"}},
	"新能源":  {{Code: "LC.GFE", Name: "碳酸锂"}, {Code: "CU.SHF", Name: "铜"}},
	"光伏":   {{Code: "SI.GFE", Name: "工业硅"}},
	"半导体":  {},
	"白酒":   {},
	"银行":   {},
	"钢铁":   {{Code: "RB.SHF", Name: "螺纹钢"}, {Code: "HC.SHF", Name: "热卷"}},
	"有色":   {{Code: "CU.SHF", Name: "铜"}, {Code: "AL.SHF", Name: "铝"}},
	"铜":     {{Code: "CU.SHF", Name: "铜"}},
	"铝":     {{Code: "AL.SHF", Name: "铝"}},
	"黄金":   {{Code: "AU.SHF", Name: "黄金"}},
	"化工":   {{Code: "SC.INE", Name: "原油"}, {Code: "LC.GFE", Name: "碳酸锂"}},
	"原油":   {{Code: "SC.INE", Name: "原油"}},
	"石油":   {{Code: "SC.INE", Name: "原油"}},
	"煤炭":   {{Code: "ZC.CZC", Name: "动力煤"}},
	"农产品":  {{Code: "CF.CZC", Name: "棉花"}, {Code: "SR.CZC", Name: "白糖"}},
	"养殖":   {{Code: "LH.DCE", Name: "生猪"}},
	"猪肉":   {{Code: "LH.DCE", Name: "生猪"}},
}

// AnalysisResult holds the result of an async deep analysis.
type AnalysisResult struct {
	Code     string
	Status   string    // "running", "completed", "failed"
	Report   string
	Error    string
	StartedAt time.Time
}

// DeepAnalysisHandler handles deep analysis requests.
type DeepAnalysisHandler struct {
	tushareDB  *services.TushareDB
	tushareSvc *services.TushareService
	results    map[string]*AnalysisResult
	mu         sync.RWMutex
}

// NewDeepAnalysisHandler creates a new DeepAnalysisHandler.
func NewDeepAnalysisHandler() *DeepAnalysisHandler {
	return &DeepAnalysisHandler{
		results: make(map[string]*AnalysisResult),
	}
}

// SetTushareDB sets the TushareDB service.
func (h *DeepAnalysisHandler) SetTushareDB(db *services.TushareDB) {
	h.tushareDB = db
}

// SetTushareService sets the TushareService for API calls.
func (h *DeepAnalysisHandler) SetTushareService(svc *services.TushareService) {
	h.tushareSvc = svc
}

// Analyze handles POST /api/deep-analysis — starts async analysis, returns immediately.
func (h *DeepAnalysisHandler) Analyze(c *gin.Context) {
	var req DeepAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, DeepAnalysisResponse{
			OK:    false,
			Error: "invalid request: " + err.Error(),
		})
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" || len(code) > 6 {
		c.JSON(http.StatusBadRequest, DeepAnalysisResponse{
			OK:    false,
			Error: "invalid stock code",
		})
		return
	}

	// Check if already running or completed
	h.mu.RLock()
	if result, exists := h.results[code]; exists {
		if result.Status == "running" {
			h.mu.RUnlock()
			elapsed := time.Since(result.StartedAt)
			pct := "分析中..."
			if elapsed > 2*time.Minute {
				pct = "分析中 (已运行" + fmt.Sprintf("%.0f", elapsed.Minutes()) + "分钟)"
			}
			c.JSON(http.StatusOK, DeepAnalysisResponse{
				OK:     true,
				Code:   code,
				Status: "running",
				PctDone: &pct,
			})
			return
		}
		h.mu.RUnlock()
	} else {
		h.mu.RUnlock()
	}

	// Start analysis in background (uses context.Background(), NOT request context)
	h.mu.Lock()
	for k := range h.results {
		// Clean up completed results older than 1 hour
		if h.results[k].Status != "running" && time.Since(h.results[k].StartedAt) > 1*time.Hour {
			delete(h.results, k)
		}
	}
	h.results[code] = &AnalysisResult{
		Code:      code,
		Status:    "running",
		StartedAt: time.Now(),
	}
	h.mu.Unlock()

	go h.runAnalysis(code)

	c.JSON(http.StatusOK, DeepAnalysisResponse{
		OK:     true,
		Code:   code,
		Status: "running",
	})
}

// Status handles GET /api/deep-analysis/status/:code — polls for result.
func (h *DeepAnalysisHandler) Status(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, DeepAnalysisResponse{OK: false, Error: "code required"})
		return
	}

	h.mu.RLock()
	result, exists := h.results[code]
	h.mu.RUnlock()

	if !exists {
		// Fallback: check persistent cache file
		cacheDir := filepath.Join(os.Getenv("HOME"), ".alphapulse", "deep_reports")
		cacheFile := filepath.Join(cacheDir, code+".md")
		if data, err := os.ReadFile(cacheFile); err == nil {
			report := string(data)
			if len(report) > 200 {
				c.JSON(http.StatusOK, DeepAnalysisResponse{
					OK:     true,
					Code:   code,
					Status: "completed",
					Report: report,
				})
				return
			}
		}
		c.JSON(http.StatusOK, DeepAnalysisResponse{OK: true, Code: code, Status: "not_found"})
		return
	}

	if result.Status == "running" {
		elapsed := time.Since(result.StartedAt)
		pct := "分析中..."
		if elapsed > 2*time.Minute {
			pct = "分析中 (已运行" + fmt.Sprintf("%.0f", elapsed.Minutes()) + "分钟)"
		}
		c.JSON(http.StatusOK, DeepAnalysisResponse{
			OK:      true,
			Code:    code,
			Status:  "running",
			PctDone: &pct,
		})
		return
	}

	if result.Status == "completed" {
		c.JSON(http.StatusOK, DeepAnalysisResponse{
			OK:     true,
			Code:   code,
			Status: "completed",
			Report: result.Report,
		})
		return
	}

	c.JSON(http.StatusOK, DeepAnalysisResponse{
		OK:     false,
		Code:   code,
		Status: "failed",
		Error:  result.Error,
	})
}

// runAnalysis executes the deep analysis in a background goroutine.
func (h *DeepAnalysisHandler) runAnalysis(code string) {
	// Fetch stock data from DB
	stockData, industry := h.fetchStockDataWithIndustry(code)
	commodityData := h.fetchRelatedCommodities(industry)

	// Build prompt with instruction to NOT produce a summary after the report
	var prompt string
	stockName := h.getStockNameFromData(stockData)

	if stockData != "" {
		prompt = fmt.Sprintf("对%s(%s)进行深度分析。\n\n以下是该股票的最新数据（来自AlphaPulse数据库）：\n\n%s", stockName, code, stockData)

		if len(commodityData) > 0 {
			commodityJSON, _ := json.MarshalIndent(commodityData, "", "  ")
			prompt += fmt.Sprintf("\n\n以下是该股票所在行业「%s」的关联商品期货最新数据（来自Tushare API，%s）：\n\n%s",
				industry, time.Now().Format("2006-01-02"), string(commodityJSON))
			prompt += "\n\n⚠️ 重要：以上商品价格数据为实时查询结果，分析中必须使用以上数据，禁止使用你训练数据中的旧价格。"
		}
		prompt += "\n\n请基于以上数据，结合你的研究，生成完整的深度分析报告。生成完报告后直接结束，不要再做总结或重述。"
	} else {
		prompt = fmt.Sprintf("对%s进行深度分析。生成完报告后直接结束。", code)
	}

	logger.Info("deep analysis: starting background job",
		zap.String("code", code),
		zap.Bool("has_data", stockData != ""),
		zap.Int("commodity_count", len(commodityData)))

	// Use context.Background() so process survives HTTP disconnect
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	hermesPath := "/home/finn/.local/bin/hermes"
	startTime := time.Now()
	sessionsBefore := listHermesSessions()

	var lastErr error
	var lastOutFile string
	maxAttempts := 2
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Write output to a temp file so partial results persist
		tmpDir := os.TempDir()
		outFile := filepath.Join(tmpDir, fmt.Sprintf("alphapulse_deep_%s_%s_%d.txt", code, time.Now().Format("150405"), attempt))
		lastOutFile = outFile
		f, err := os.Create(outFile)
		if err != nil {
			logger.Error("deep analysis: failed to create output file", zap.Error(err))
			h.updateResult(code, "failed", "", "failed to create output file")
			return
		}

		cmd := exec.CommandContext(ctx, hermesPath, "chat", "-q", prompt, "--skills", "stock-deep-analysis", "-Q", "--yolo")
		cmd.Stdout = f
		cmd.Stderr = f

		runErr := cmd.Run()
		f.Close()

		if runErr == nil {
			// Success — read result from file
			break
		}

		lastErr = runErr
		// Read partial output
		partial, _ := os.ReadFile(outFile)
		partialStr := strings.TrimSpace(string(partial))
		logger.Warn("deep analysis: hermes attempt failed",
			zap.String("code", code),
			zap.Int("attempt", attempt),
			zap.Error(runErr),
			zap.Int("partial_output_len", len(partialStr)))

		// If we have substantial partial output, use it
		if len(partialStr) > 200 {
			h.updateResult(code, "completed", partialStr, "")
			logger.Info("deep analysis: saved partial result despite error",
				zap.String("code", code),
				zap.Int("length", len(partialStr)),
				zap.Duration("elapsed", time.Since(startTime)))
			return
		}

		if attempt < maxAttempts {
			logger.Info("deep analysis: retrying in 5s...", zap.String("code", code))
			time.Sleep(5 * time.Second)
		}
	}

	if lastErr != nil {
		h.updateResult(code, "failed", "", fmt.Sprintf("analysis error after %.0f min (%d attempts): %v", time.Since(startTime).Minutes(), maxAttempts, lastErr))
		return
	}

	// Try to extract the full report from the Hermes session file.
	// The -Q flag only outputs the last message; the full report may be
	// an earlier message. Search for the newest session file.
	report := extractReportFromSession(sessionsBefore, code)

	// Fallback: use stdout if session parsing failed
	if report == "" {
		output, _ := os.ReadFile(lastOutFile)
		report = strings.TrimSpace(string(output))
	}

	if report == "" {
		h.updateResult(code, "failed", "", "empty report")
		return
	}

	h.updateResult(code, "completed", report, "")

	// Also persist to cache file for cross-restart persistence
	h.cacheReport(code, report)

	logger.Info("deep analysis: completed",
		zap.String("code", code),
		zap.Int("report_length", len(report)),
		zap.Duration("elapsed", time.Since(startTime)))
}

// updateResult thread-safely updates the analysis result.
func (h *DeepAnalysisHandler) updateResult(code, status, report, err string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if result, exists := h.results[code]; exists {
		result.Status = status
		result.Report = report
		result.Error = err
	}
}

// fetchStockDataWithIndustry fetches stock data and industry from DB.
func (h *DeepAnalysisHandler) fetchStockDataWithIndustry(code string) (string, string) {
	if h.tushareDB == nil {
		return "", ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	quote, err := h.tushareDB.FetchQuoteFromDB(ctx, code)
	if err != nil {
		logger.Warn("failed to fetch quote for deep analysis", zap.String("code", code), zap.Error(err))
		return "", ""
	}

	industry, _ := h.tushareDB.FetchIndustryFromDB(ctx, code)

	data := map[string]interface{}{
		"code":           quote.Code,
		"name":           quote.Name,
		"price":          quote.Price,
		"change":         quote.Change,
		"change_percent": quote.ChangePercent,
		"open":           quote.Open,
		"high":           quote.High,
		"low":            quote.Low,
		"prev_close":     quote.PrevClose,
		"volume":         quote.Volume,
		"turnover":       quote.Turnover,
		"pe":             quote.PE,
		"pb":             quote.PB,
		"total_mv":       quote.TotalMV,
		"amplitude":      quote.Amplitude,
		"industry":       industry,
		"updated_at":     quote.UpdatedAt,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", industry
	}

	return string(jsonData), industry
}

// fetchRelatedCommodities fetches commodity futures data based on industry.
func (h *DeepAnalysisHandler) fetchRelatedCommodities(industry string) []CommodityData {
	if h.tushareSvc == nil || industry == "" {
		return nil
	}

	var commodities []struct {
		Code string
		Name string
	}
	for keyword, commodityList := range IndustryCommodityMap {
		if strings.Contains(industry, keyword) {
			commodities = append(commodities, commodityList...)
			break
		}
	}
	if len(commodities) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var results []CommodityData
	for _, comm := range commodities {
		cd, err := h.fetchCommodityPrice(ctx, comm.Code, comm.Name)
		if err != nil {
			logger.Warn("failed to fetch commodity price",
				zap.String("code", comm.Code),
				zap.Error(err))
			continue
		}
		results = append(results, cd)
	}
	return results
}

// fetchCommodityPrice fetches the latest futures price from Tushare API.
func (h *DeepAnalysisHandler) fetchCommodityPrice(ctx context.Context, tsCode, name string) (CommodityData, error) {
	endDate := time.Now().Format("20060102")
	startDate := time.Now().AddDate(0, -1, 0).Format("20060102")

	resp, err := h.tushareSvc.Query(ctx, "fut_daily", map[string]string{
		"ts_code":    tsCode,
		"start_date": startDate,
		"end_date":   endDate,
	}, "trade_date,close,open,high,low,pct_chg")
	if err != nil {
		return CommodityData{}, fmt.Errorf("tushare query failed: %w", err)
	}

	if len(resp.Data.Items) == 0 {
		return CommodityData{}, fmt.Errorf("no data returned for %s", tsCode)
	}

	items := resp.Data.Items
	latest := items[0]

	cd := CommodityData{
		Code:       tsCode,
		Name:       name,
		LatestDate: fmt.Sprintf("%v", latest[0]),
	}

	if v, ok := latest[1].(float64); ok {
		cd.Close = v
	}
	if len(items) > 1 {
		if v, ok := items[0][1].(float64); ok {
			cd.PrevClose = v
		}
	}

	if len(items) >= 5 {
		if v, ok := items[4][1].(float64); ok && v > 0 {
			cd.Change5D = (cd.Close - v) / v * 100
		}
	}
	if len(items) >= 20 {
		if v, ok := items[19][1].(float64); ok && v > 0 {
			cd.Change20D = (cd.Close - v) / v * 100
		}
	}

	if cd.Change5D > 3 {
		cd.Trend = "上涨"
	} else if cd.Change5D < -3 {
		cd.Trend = "下跌"
	} else {
		cd.Trend = "震荡"
	}

	return cd, nil
}

// getStockNameFromData extracts stock name from JSON data string.
func (h *DeepAnalysisHandler) getStockNameFromData(data string) string {
	var result struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return ""
	}
	return result.Name
}

// listHermesSessions returns a set of session file paths that exist before analysis starts.
func listHermesSessions() map[string]bool {
	sessionsDir := filepath.Join(os.Getenv("HOME"), ".hermes", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			seen[entry.Name()] = true
		}
	}
	return seen
}

// extractReportFromSession finds the newest session file created after the given
// set of existing sessions, and extracts the longest assistant message as the report.
func extractReportFromSession(before map[string]bool, code string) string {
	if before == nil {
		before = make(map[string]bool)
	}
	sessionsDir := filepath.Join(os.Getenv("HOME"), ".hermes", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return ""
	}

	// Find the newest session file that wasn't there before
	var newestFile string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if before[name] {
			continue // was already there before analysis started
		}
		if newestFile == "" || entry.Name() > newestFile {
			newestFile = name
		}
	}
	if newestFile == "" {
		return ""
	}

	fullPath := filepath.Join(sessionsDir, newestFile)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		logger.Warn("failed to read session file", zap.String("file", fullPath), zap.Error(err))
		return ""
	}

	var session struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(data, &session); err != nil {
		logger.Warn("failed to parse session file", zap.String("file", fullPath), zap.Error(err))
		return ""
	}

	// Find the longest assistant message (the actual report, not summaries)
	var longest string
	for _, msg := range session.Messages {
		if msg.Role == "assistant" && len(msg.Content) > len(longest) {
			longest = msg.Content
		}
	}

	if len(longest) < 200 {
		logger.Warn("report too short from session",
			zap.String("code", code),
			zap.Int("length", len(longest)))
		return ""
	}

	logger.Info("extracted report from session",
		zap.String("code", code),
		zap.String("session_file", newestFile),
		zap.Int("report_length", len(longest)))
	return longest
}

// cacheReport persists the report to a file for cross-restart durability.
func (h *DeepAnalysisHandler) cacheReport(code, report string) {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".alphapulse", "deep_reports")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		logger.Warn("failed to create cache dir", zap.Error(err))
		return
	}
	cacheFile := filepath.Join(cacheDir, code+".md")
	if err := os.WriteFile(cacheFile, []byte(report), 0644); err != nil {
		logger.Warn("failed to cache report", zap.Error(err))
	}
}
