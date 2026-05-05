package services

import (
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMovingAverage(t *testing.T) {
	assert.Equal(t, 0.0, MovingAverage(nil, 5))
	assert.Equal(t, 0.0, MovingAverage([]float64{1, 2, 3}, 5))
	assert.InDelta(t, 3.0, MovingAverage([]float64{1, 2, 3, 4, 5}, 5), 0.01)
	assert.InDelta(t, 5.0, MovingAverage([]float64{2, 3, 4, 5, 6}, 3), 0.01)
}

func TestCalculateMACD(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := CalculateMACD(nil)
		assert.Equal(t, "数据不足", result.Signal)
		assert.Equal(t, "数据不足", result.HistTrend)
	})

	t.Run("sufficient data", func(t *testing.T) {
		// Generate 40 data points
		closes := make([]float64, 40)
		for i := range closes {
			closes[i] = 100 + float64(i)*0.5
		}
		result := CalculateMACD(closes)
		assert.NotZero(t, result.DIF)
		assert.NotZero(t, result.DEA)
		assert.Len(t, result.HistLast3, 3)
		// Uptrend should be 多头 or 金叉
		assert.Contains(t, []string{"多头", "金叉", "数据不足"}, result.Signal)
	})
}

func TestCalculateRSI(t *testing.T) {
	assert.Equal(t, 0.0, CalculateRSI(nil, 14))
	assert.Equal(t, 0.0, CalculateRSI([]float64{1, 2, 3}, 14))

	closes := []float64{44, 44.34, 44.09, 43.61, 44.33, 44.83, 45.10, 45.42, 45.84, 46.08, 45.89, 46.03, 45.61, 46.28, 46.28, 46.00, 46.03}
	rsi := CalculateRSI(closes, 14)
	assert.True(t, rsi >= 0 && rsi <= 100, "RSI should be between 0 and 100, got %f", rsi)
}

func TestCalculateBollinger(t *testing.T) {
	t.Run("insufficient data", func(t *testing.T) {
		result := CalculateBollinger([]float64{1, 2, 3}, 20)
		assert.Zero(t, result.Upper)
	})

	t.Run("sufficient data", func(t *testing.T) {
		closes := make([]float64, 20)
		for i := range closes {
			closes[i] = 100 + float64(i%3)
		}
		result := CalculateBollinger(closes, 20)
		assert.True(t, result.Upper > result.Mid)
		assert.True(t, result.Mid > result.Lower)
		assert.True(t, result.Bandwidth > 0)
	})
}

func TestCalculateKDJ(t *testing.T) {
	t.Run("insufficient data", func(t *testing.T) {
		result := CalculateKDJ([]float64{1}, []float64{2}, []float64{0.5}, 9)
		assert.Equal(t, "数据不足", result.Signal)
	})

	t.Run("sufficient data", func(t *testing.T) {
		n := 15
		closes := make([]float64, n)
		highs := make([]float64, n)
		lows := make([]float64, n)
		for i := range closes {
			closes[i] = 100 + float64(i)
			highs[i] = closes[i] + 1
			lows[i] = closes[i] - 1
		}
		result := CalculateKDJ(closes, highs, lows, 9)
		assert.NotZero(t, result.K)
		assert.NotZero(t, result.D)
		assert.NotEmpty(t, result.Signal)
	})
}

func TestCalculateOBV(t *testing.T) {
	t.Run("insufficient data", func(t *testing.T) {
		result := CalculateOBV([]float64{1}, []float64{100})
		assert.Equal(t, "数据不足", result.Trend)
	})

	t.Run("uptrend", func(t *testing.T) {
		closes := []float64{10, 11, 12, 13, 14, 15}
		volumes := []float64{100, 200, 150, 300, 250, 200}
		result := CalculateOBV(closes, volumes)
		assert.Equal(t, "上升", result.Trend)
		assert.Len(t, result.Recent5D, 5)
	})
}

func TestAnalyzeOrderFlow(t *testing.T) {
	t.Run("strong buy", func(t *testing.T) {
		quote := models.Quote{OuterVol: 6000, InnerVol: 4000}
		result := AnalyzeOrderFlow(quote)
		assert.Equal(t, "买方强势", result.NetDirection)
		assert.InDelta(t, 60.0, result.OuterRatio, 0.01)
	})

	t.Run("no data", func(t *testing.T) {
		quote := models.Quote{}
		result := AnalyzeOrderFlow(quote)
		assert.Equal(t, "数据不足", result.NetDirection)
	})
}

func TestAnalyzeVolumePrice(t *testing.T) {
	quote := models.Quote{Volume: 10000, ChangePercent: 2.5, Turnover: 5.0}
	klines := []models.KlinePoint{
		{Volume: 8000}, {Volume: 9000}, {Volume: 7000}, {Volume: 8500}, {Volume: 7500},
	}
	result := AnalyzeVolumePrice(quote, klines)
	assert.Equal(t, "量价齐升", result.PriceVolumeHarmony)
	assert.Equal(t, "活跃", result.TurnoverLevel)
}

func TestAnalyzeValuation(t *testing.T) {
	quote := models.Quote{PE: 12, PB: 0.8, TotalMV: 600}
	result := AnalyzeValuation(quote)
	assert.Equal(t, "偏低", result.PELevel)
	assert.Equal(t, "偏低", result.PBLevel)
	assert.Equal(t, "中大盘股", result.MVLevel)
}

func TestAnalyzeVolatility(t *testing.T) {
	quote := models.Quote{Amplitude: 7.5, Price: 100, LimitUp: 110, LimitDown: 90}
	result := AnalyzeVolatility(quote)
	assert.Equal(t, "高波动", result.AmplitudeLevel)
	assert.Contains(t, result.Verdict, "振幅偏大")
}

func TestAnalyzeMoneyFlow(t *testing.T) {
	t.Run("inflow", func(t *testing.T) {
		flows := []models.MoneyFlowDay{
			{MainNet: 100, HugeNet: 50, BigNet: 50, SmallNet: -30},
			{MainNet: 200, HugeNet: 80, BigNet: 120, SmallNet: -40},
		}
		result := AnalyzeMoneyFlow(flows)
		assert.Equal(t, "流入", result.TodayMainDirection)
		assert.Equal(t, 2, result.MainConsecutiveDays)
		assert.Equal(t, "大单主导", result.InstitutionVsHotMoney)
	})

	t.Run("empty", func(t *testing.T) {
		result := AnalyzeMoneyFlow(nil)
		assert.Equal(t, "数据不足", result.TodayMainDirection)
	})
}

func TestAnalyzeTechnical(t *testing.T) {
	// Generate uptrend klines
	klines := make([]models.KlinePoint, 60)
	for i := range klines {
		price := 100.0 + float64(i)*0.5
		klines[i] = models.KlinePoint{
			Close:  price,
			High:   price + 1,
			Low:    price - 1,
			Volume: 10000 + float64(i)*100,
		}
	}
	result := AnalyzeTechnical(klines)
	assert.NotEmpty(t, result.MAArrangement)
	assert.NotEmpty(t, result.MACD_Signal)
	assert.NotEmpty(t, result.Verdict)
}

func TestAnalyzeSector(t *testing.T) {
	t.Run("leader", func(t *testing.T) {
		quote := models.Quote{Name: "中国平安", TotalMV: 800}
		result := AnalyzeSector(quote, []string{"保险", "金融"})
		assert.True(t, result.IsSectorLeader)
		assert.Equal(t, "保险", result.PrimarySector)
	})

	t.Run("no data", func(t *testing.T) {
		result := AnalyzeSector(models.Quote{}, nil)
		assert.False(t, result.IsSectorLeader)
		assert.Contains(t, result.Verdict, "数据不足")
	})
}

func TestAnalyzeSentiment(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		news := []models.NewsItem{
			{Title: "公司业绩增长超预期"},
			{Title: "利好消息刺激涨停"},
		}
		result := AnalyzeSentiment(news, nil)
		assert.Equal(t, "正面", result.SentimentLabel)
		assert.True(t, result.SentimentScore > 0)
	})

	t.Run("negative", func(t *testing.T) {
		anns := []models.Announcement{
			{Title: "公司亏损风险提示"},
			{Title: "利空消息导致下跌"},
		}
		result := AnalyzeSentiment(nil, anns)
		assert.Equal(t, "负面", result.SentimentLabel)
	})

	t.Run("neutral", func(t *testing.T) {
		news := []models.NewsItem{{Title: "公司召开股东大会"}}
		result := AnalyzeSentiment(news, nil)
		assert.Equal(t, "中性", result.SentimentLabel)
	})
}

func TestBuildSummary(t *testing.T) {
	t.Run("bullish", func(t *testing.T) {
		analysis := &models.StockAnalysis{
			OrderFlow:   models.OrderFlowAnalysis{NetDirection: "买方强势"},
			VolumePrice: models.VolumePriceAnalysis{PriceVolumeHarmony: "量价齐升"},
			Valuation:   models.ValuationAnalysis{PELevel: "合理", PBLevel: "合理"},
			MoneyFlow:   models.MoneyFlowAnalysis{TodayMainDirection: "流入"},
			Technical:   models.TechnicalAnalysis{MAArrangement: "多头排列", MACD_Signal: "金叉"},
			Sector:      models.SectorAnalysis{IsSectorLeader: true},
			Sentiment:   models.SentimentAnalysis{SentimentLabel: "正面", KeyEvents: []string{"利好"}},
		}
		summary := BuildSummary(analysis)
		require.True(t, summary.OverallScore >= 75, "expected bullish score >= 75, got %d", summary.OverallScore)
		assert.Equal(t, "看多", summary.OverallSignal)
		assert.NotEmpty(t, summary.Strengths)
	})

	t.Run("bearish", func(t *testing.T) {
		analysis := &models.StockAnalysis{
			OrderFlow:   models.OrderFlowAnalysis{NetDirection: "卖方强势"},
			VolumePrice: models.VolumePriceAnalysis{PriceVolumeHarmony: "放量下跌"},
			Valuation:   models.ValuationAnalysis{PELevel: "很高", PBLevel: "很高"},
			MoneyFlow:   models.MoneyFlowAnalysis{TodayMainDirection: "流出"},
			Technical:   models.TechnicalAnalysis{MAArrangement: "空头排列", MACD_Signal: "死叉"},
			Sentiment:   models.SentimentAnalysis{SentimentLabel: "负面", KeyEvents: []string{"利空"}},
		}
		summary := BuildSummary(analysis)
		require.True(t, summary.OverallScore <= 40, "expected bearish score <= 40, got %d", summary.OverallScore)
		assert.Equal(t, "偏空", summary.OverallSignal)
		assert.NotEmpty(t, summary.Risks)
	})
}
