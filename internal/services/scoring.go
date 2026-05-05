package services

import (
	"context"
	"math"
	"sort"

	"alphapulse/internal/models"

	"go.uber.org/zap"
)

// ──────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────

// DimensionScore represents a single analysis dimension.
type DimensionScore struct {
	Name    string  `json:"name"`
	Score   int     `json:"score"`   // 0-100
	Weight  float64 `json:"weight"`
	Signal  string  `json:"signal"`  // bullish / bearish / neutral
	Details string  `json:"details"`
}

// StockAnalysis is the full 8-dimension analysis result.
type StockAnalysis struct {
	Code       string          `json:"code"`
	Name       string          `json:"name"`
	Dimensions []DimensionScore `json:"dimensions"`
	TotalScore int             `json:"total_score"`
	Signal     string          `json:"signal"`
	Suggestion string          `json:"suggestion"`
}

// AnalysisDeps holds dependencies for the scoring engine.
type AnalysisDeps struct {
	EastMoney *EastMoneyService
	Tencent   *TencentService
	Logger    *zap.Logger
}

// ──────────────────────────────────────────────
// Weights
// ──────────────────────────────────────────────

const (
	weightTechnical = 0.25
	weightFundFlow  = 0.20
	weightVolume    = 0.15
	weightTrend     = 0.15
	weightValuation = 0.10
	weightVolatility = 0.05
	weightSector    = 0.05
	weightSentiment = 0.05
)

// ──────────────────────────────────────────────
// Main analysis function
// ──────────────────────────────────────────────

// AnalyzeStock performs full 8-dimension analysis on a stock.
func AnalyzeStock(ctx context.Context, deps AnalysisDeps, code, name string) StockAnalysis {
	dims := make([]DimensionScore, 0, 8)

	// 1. Fetch kline data (shared by multiple dimensions)
	klines, kerr := deps.EastMoney.FetchKline(ctx, code, 120)

	// 2. Technical Indicators (MACD/KDJ/RSI/Boll/MA)
	dims = append(dims, analyzeTechnical(klines, kerr))

	// 3. Fund Flow
	dims = append(dims, analyzeFundFlow(ctx, deps, code))

	// 4. Volume-Price Coordination
	dims = append(dims, analyzeVolumePrice(klines, kerr))

	// 5. Trend (multi-period)
	dims = append(dims, analyzeTrend(klines, kerr))

	// 6. Valuation
	dims = append(dims, analyzeValuation(ctx, deps, code))

	// 7. Volatility
	dims = append(dims, analyzeVolatility(klines, kerr))

	// 8. Sector
	dims = append(dims, analyzeSector(ctx, deps, code))

	// 9. Sentiment (news-based)
	dims = append(dims, analyzeSentiment(ctx, deps, code))

	// Calculate weighted total
	total := 0.0
	for _, d := range dims {
		total += float64(d.Score) * d.Weight
	}
	totalScore := int(math.Round(total))

	signal := scoreToSignal(totalScore)
	suggestion := generateSuggestion(dims, totalScore, signal)

	return StockAnalysis{
		Code:       code,
		Name:       name,
		Dimensions: dims,
		TotalScore: totalScore,
		Signal:     signal,
		Suggestion: suggestion,
	}
}

// ──────────────────────────────────────────────
// Dimension analyzers
// ──────────────────────────────────────────────

func analyzeTechnical(klines []models.KlinePoint, err error) DimensionScore {
	d := DimensionScore{Name: "technical", Weight: weightTechnical, Signal: "neutral", Details: "技术面分析"}

	if err != nil || len(klines) < 20 {
		d.Score = 50
		d.Details = "K线数据不足，使用默认评分"
		return d
	}

	ind := ComputeIndicators(klines)
	score := 50.0
	signals := make([]string, 0)

	// MACD contribution
	switch ind.MACD.Signal {
	case "golden_cross":
		score += 15
		signals = append(signals, "MACD金叉")
	case "death_cross":
		score -= 15
		signals = append(signals, "MACD死叉")
	default:
		if ind.MACD.Histogram > 0 {
			score += 5
		} else {
			score -= 5
		}
	}

	// KDJ contribution
	switch ind.KDJ.Signal {
	case "oversold":
		score += 10
		signals = append(signals, "KDJ超卖")
	case "overbought":
		score -= 10
		signals = append(signals, "KDJ超买")
	}

	// RSI contribution (use RSI12)
	if ind.RSI.RSI12 > 70 {
		score -= 10
		signals = append(signals, "RSI超买")
	} else if ind.RSI.RSI12 < 30 {
		score += 10
		signals = append(signals, "RSI超卖")
	} else if ind.RSI.RSI12 > 50 {
		score += 5
	}

	// Boll contribution
	switch ind.Boll.Signal {
	case "above_upper":
		score -= 5
		signals = append(signals, "突破布林上轨")
	case "below_lower":
		score += 5
		signals = append(signals, "跌破布林下轨")
	}

	// MA alignment
	score += float64(ind.MA.Score-50) * 0.3

	d.Score = clampScore(score)
	d.Signal = scoreToSignal(d.Score)
	if len(signals) > 0 {
		d.Details = joinStrings(signals)
	} else {
		d.Details = "技术指标中性"
	}
	return d
}

func analyzeFundFlow(ctx context.Context, deps AnalysisDeps, code string) DimensionScore {
	d := DimensionScore{Name: "fund_flow", Weight: weightFundFlow, Signal: "neutral", Details: "资金流向分析"}

	flows, err := deps.EastMoney.FetchMoneyFlow(ctx, code, 10)
	if err != nil || len(flows) == 0 {
		d.Score = 50
		d.Details = "资金流数据不可用"
		return d
	}

	score := 50.0
	signals := make([]string, 0)

	// Calculate net flow for recent 5 days
	recent := flows
	if len(recent) > 5 {
		recent = flows[len(flows)-5:]
	}

	totalNet := 0.0
	positiveDays := 0
	for _, f := range recent {
		net := f.MainNet
		totalNet += net
		if net > 0 {
			positiveDays++
		}
	}

	// Net inflow contribution
	if totalNet > 0 {
		score += math.Min(20, totalNet/1e6*2) // scale by millions
		signals = append(signals, "主力净流入")
	} else {
		score += math.Max(-20, totalNet/1e6*2)
		signals = append(signals, "主力净流出")
	}

	// Consecutive days
	if positiveDays >= 4 {
		score += 10
		signals = append(signals, "连续流入")
	} else if positiveDays <= 1 {
		score -= 10
		signals = append(signals, "连续流出")
	}

	d.Score = clampScore(score)
	d.Signal = scoreToSignal(d.Score)
	if len(signals) > 0 {
		d.Details = joinStrings(signals)
	}
	return d
}

func analyzeVolumePrice(klines []models.KlinePoint, err error) DimensionScore {
	d := DimensionScore{Name: "volume_price", Weight: weightVolume, Signal: "neutral", Details: "量价分析"}

	if err != nil || len(klines) < 10 {
		d.Score = 50
		d.Details = "数据不足"
		return d
	}

	va := AnalyzeVolume(klines)
	d.Score = va.Score
	d.Signal = va.Signal

	switch va.Coordination {
	case "strong":
		d.Details = "量价齐升，强势信号"
	case "divergence":
		d.Details = "量价背离，注意反转风险"
	case "weak":
		d.Details = "放量下跌，趋势转弱"
	default:
		if va.Trend5D == "increasing" {
			d.Details = "成交量放大"
		} else if va.Trend5D == "decreasing" {
			d.Details = "成交量萎缩"
		} else {
			d.Details = "量能平稳"
		}
	}
	return d
}

func analyzeTrend(klines []models.KlinePoint, err error) DimensionScore {
	d := DimensionScore{Name: "trend", Weight: weightTrend, Signal: "neutral", Details: "趋势分析"}

	if err != nil || len(klines) < 20 {
		d.Score = 50
		d.Details = "数据不足"
		return d
	}

	closes := extractCloses(klines)
	score := 50.0
	signals := make([]string, 0)

	// Short-term trend (5 days)
	if len(closes) >= 5 {
		change5d := (closes[len(closes)-1] - closes[len(closes)-5]) / closes[len(closes)-5] * 100
		score += change5d * 2
		if change5d > 3 {
			signals = append(signals, "5日强势上涨")
		} else if change5d < -3 {
			signals = append(signals, "5日明显下跌")
		}
	}

	// Medium-term trend (20 days)
	if len(closes) >= 20 {
		change20d := (closes[len(closes)-1] - closes[len(closes)-20]) / closes[len(closes)-20] * 100
		score += change20d * 0.5
		if change20d > 10 {
			signals = append(signals, "20日强势")
		} else if change20d < -10 {
			signals = append(signals, "20日弱势")
		}
	}

	// MA alignment
	ma := ComputeMAAlignment(closes)
	score += float64(ma.Score-50) * 0.3
	if ma.Aligned {
		signals = append(signals, "均线多头排列")
	} else if ma.BearAlign {
		signals = append(signals, "均线空头排列")
	}

	d.Score = clampScore(score)
	d.Signal = scoreToSignal(d.Score)
	if len(signals) > 0 {
		d.Details = joinStrings(signals)
	} else {
		d.Details = "趋势中性"
	}
	return d
}

func analyzeValuation(_ context.Context, _ AnalysisDeps, _ string) DimensionScore {
	// Placeholder — PE/PB percentile requires historical data
	return DimensionScore{
		Name:    "valuation",
		Score:   50,
		Weight:  weightValuation,
		Signal:  "neutral",
		Details: "估值分析（待实现历史分位数）",
	}
}

func analyzeVolatility(klines []models.KlinePoint, err error) DimensionScore {
	d := DimensionScore{Name: "volatility", Weight: weightVolatility, Signal: "neutral", Details: "波动率分析"}

	if err != nil || len(klines) < 20 {
		d.Score = 50
		d.Details = "数据不足"
		return d
	}

	closes := extractCloses(klines)

	// Calculate ATR-like volatility
	sum := 0.0
	for i := 1; i < len(closes); i++ {
		change := math.Abs(closes[i]-closes[i-1]) / closes[i-1]
		sum += change
	}
	avgVol := sum / float64(len(closes)-1) * 100

	score := 50.0
	if avgVol < 1 {
		score = 70 // low volatility is generally favorable
		d.Details = "低波动，走势平稳"
	} else if avgVol < 2 {
		score = 60
		d.Details = "正常波动"
	} else if avgVol < 3 {
		score = 45
		d.Details = "波动偏大"
	} else {
		score = 30
		d.Details = "高波动，风险较高"
	}

	d.Score = clampScore(score)
	d.Signal = scoreToSignal(d.Score)
	return d
}

func analyzeSector(_ context.Context, _ AnalysisDeps, _ string) DimensionScore {
	// Placeholder — needs sector performance comparison
	return DimensionScore{
		Name:    "sector",
		Score:   50,
		Weight:  weightSector,
		Signal:  "neutral",
		Details: "板块分析（待实现板块强弱对比）",
	}
}

func analyzeSentiment(ctx context.Context, deps AnalysisDeps, code string) DimensionScore {
	d := DimensionScore{Name: "sentiment", Weight: weightSentiment, Signal: "neutral", Details: "消息面分析"}

	news, err := deps.EastMoney.FetchStockNews(ctx, code, 10)
	if err != nil || len(news) == 0 {
		d.Score = 50
		d.Details = "暂无相关新闻"
		return d
	}

	// Simple keyword-based sentiment
	positive := 0
	negative := 0
	for _, n := range news {
		title := n.Title
		for _, kw := range []string{"利好", "增长", "突破", "新高", "涨停", "大涨", "翻倍"} {
			if contains(title, kw) {
				positive++
				break
			}
		}
		for _, kw := range []string{"利空", "下跌", "跌停", "暴跌", "亏损", "减持", "违规"} {
			if contains(title, kw) {
				negative++
				break
			}
		}
	}

	score := 50.0 + float64(positive-negative)*5
	d.Score = clampScore(score)
	d.Signal = scoreToSignal(d.Score)
	d.Details = "近期新闻情绪中性"
	if positive > negative {
		d.Details = "近期新闻偏正面"
	} else if negative > positive {
		d.Details = "近期新闻偏负面"
	}
	return d
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func scoreToSignal(score int) string {
	if score >= 70 {
		return "bullish"
	}
	if score <= 30 {
		return "bearish"
	}
	return "neutral"
}

func clampScore(v float64) int {
	if v > 100 {
		return 100
	}
	if v < 0 {
		return 0
	}
	return int(math.Round(v))
}

func joinStrings(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += "，" + p
	}
	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func generateSuggestion(dims []DimensionScore, total int, signal string) string {
	// Find strongest and weakest dimensions
	sort.Slice(dims, func(i, j int) bool { return dims[i].Score > dims[j].Score })
	best := dims[0]
	worst := dims[len(dims)-1]

	switch signal {
	case "bullish":
		return "综合偏多，" + best.Name + "评分最高(" + itoa(best.Score) + ")，可关注买入机会"
	case "bearish":
		return "综合偏空，" + worst.Name + "评分最低(" + itoa(worst.Score) + ")，建议观望或减仓"
	default:
		return "综合中性，建议观望等待明确信号"
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
