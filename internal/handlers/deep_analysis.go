package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
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
	OK     bool   `json:"ok"`
	Code   string `json:"code"`
	Report string `json:"report,omitempty"`
	Error  string `json:"error,omitempty"`
}

// CommodityData holds futures price data for a commodity.
type CommodityData struct {
	Code      string  `json:"code"`       // e.g. "LC.GFE"
	Name      string  `json:"name"`       // e.g. "碳酸锂"
	LatestDate string `json:"latest_date"` // e.g. "20260512"
	Close     float64 `json:"close"`      // latest close price
	PrevClose float64 `json:"prev_close"` // previous close
	Change5D  float64 `json:"change_5d"`  // 5-day change %
	Change20D float64 `json:"change_20d"` // 20-day change %
	Trend     string  `json:"trend"`      // "上涨"/"下跌"/"震荡"
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

// DeepAnalysisHandler handles deep analysis requests.
type DeepAnalysisHandler struct {
	tushareDB  *services.TushareDB
	tushareSvc *services.TushareService
}

// NewDeepAnalysisHandler creates a new DeepAnalysisHandler.
func NewDeepAnalysisHandler() *DeepAnalysisHandler {
	return &DeepAnalysisHandler{}
}

// SetTushareDB sets the TushareDB service.
func (h *DeepAnalysisHandler) SetTushareDB(db *services.TushareDB) {
	h.tushareDB = db
}

// SetTushareService sets the TushareService for API calls.
func (h *DeepAnalysisHandler) SetTushareService(svc *services.TushareService) {
	h.tushareSvc = svc
}

// Analyze handles POST /api/deep-analysis
// @Summary 触发深度分析
// @Description 调用Hermes Agent对股票进行深度分析
// @Tags deep-analysis
// @Accept json
// @Produce json
// @Param request body DeepAnalysisRequest true "股票代码"
// @Success 200 {object} DeepAnalysisResponse
// @Router /api/deep-analysis [post]
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

	// Fetch stock data from DB directly
	stockData, industry := h.fetchStockDataWithIndustry(code)

	// Fetch related commodity futures data based on industry
	commodityData := h.fetchRelatedCommodities(industry)

	// Build prompt with data context
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

		prompt += "\n\n请基于以上数据，结合你的研究，生成完整的深度分析报告。"
	} else {
		prompt = fmt.Sprintf("对%s进行深度分析", code)
	}

	logger.Info("triggering deep analysis",
		zap.String("code", code),
		zap.Bool("has_data", stockData != ""),
		zap.Int("commodity_count", len(commodityData)))

	// Call Hermes Agent CLI — deep analysis needs time for web research
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	hermesPath := "/home/finn/.local/bin/hermes"
	cmd := exec.CommandContext(ctx, hermesPath, "chat", "-q", prompt, "--skills", "stock-deep-analysis", "-Q", "--yolo")
	output, err := cmd.CombinedOutput()

	if err != nil {
		logger.Error("hermes agent failed",
			zap.String("code", code),
			zap.Error(err),
			zap.String("output", string(output)))
		c.JSON(http.StatusInternalServerError, DeepAnalysisResponse{
			OK:    false,
			Code:  code,
			Error: "analysis failed: " + err.Error(),
		})
		return
	}

	report := strings.TrimSpace(string(output))
	if report == "" {
		c.JSON(http.StatusInternalServerError, DeepAnalysisResponse{
			OK:    false,
			Code:  code,
			Error: "empty report",
		})
		return
	}

	logger.Info("deep analysis completed",
		zap.String("code", code),
		zap.Int("report_length", len(report)))

	c.JSON(http.StatusOK, DeepAnalysisResponse{
		OK:     true,
		Code:   code,
		Report: report,
	})
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

	// Fetch industry
	industry, _ := h.tushareDB.FetchIndustryFromDB(ctx, code)

	// Format as structured data
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

	// Find matching commodities for this industry
	var commodities []struct {
		Code string
		Name string
	}
	for keyword, commodityList := range IndustryCommodityMap {
		if strings.Contains(industry, keyword) {
			commodities = append(commodities, commodityList...)
			break // use first match
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
	startDate := time.Now().AddDate(0, -1, 0).Format("20060102") // 1 month ago

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

	// Parse results — items are sorted by trade_date desc
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

	// Calculate 5-day and 20-day changes
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

	// Determine trend
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
