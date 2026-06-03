package services

import (
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectPatternsBullFlag(t *testing.T) {
	klines := make([]models.KlinePoint, 0, 20)
	for i := 0; i < 13; i++ {
		close := 10.0 + float64(i)*0.18
		klines = append(klines, models.KlinePoint{
			Date:   "2026-01-01",
			Open:   close - 0.08,
			Close:  close,
			High:   close + 0.08,
			Low:    close - 0.10,
			Volume: 1000,
		})
	}
	for i := 0; i < 7; i++ {
		close := 12.1 + float64(i%2)*0.08
		klines = append(klines, models.KlinePoint{
			Date:   "2026-01-02",
			Open:   close - 0.03,
			Close:  close,
			High:   close + 0.10,
			Low:    close - 0.10,
			Volume: 900,
		})
	}

	patterns := DetectPatterns(klines)

	require.NotEmpty(t, patterns)
	assert.True(t, hasPatternSignal(patterns, "上升旗形"))
}

func TestAnalyzePatternSignalsSummary(t *testing.T) {
	patterns := []models.PatternSignal{
		{Pattern: "锤子线", Category: "kline", Direction: "bullish", Confidence: 0.8},
		{Pattern: "放量突破", Category: "volume", Direction: "bullish", Confidence: 0.75},
	}

	result := AnalyzePatternSignals(patterns)

	require.NotNil(t, result)
	assert.Equal(t, "bullish", result.NetBias)
	assert.Equal(t, 2, result.BullishCount)
	assert.True(t, result.ScoreImpact > 0)
	require.NotNil(t, result.Primary)
	assert.Equal(t, "锤子线", result.Primary.Pattern)
}

func TestAnalyzeShortTermScoreBullish(t *testing.T) {
	patterns := AnalyzePatternSignals([]models.PatternSignal{
		{Pattern: "放量突破", Category: "volume", Direction: "bullish", Confidence: 0.8},
	})
	forecast := &models.IntradayForecast{CurrentZone: "lower", Bias: "bullish"}

	result := AnalyzeShortTermScore(
		models.Quote{ChangePercent: 2.5},
		models.TechnicalAnalysis{MAArrangement: "多头排列", MACD_Signal: "多头", RSI_14: 62},
		models.VolumePriceAnalysis{PriceVolumeHarmony: "量价齐升", VolumeRatio: 1.8, TodayChangePct: 2.5},
		models.MoneyFlowAnalysis{TodayMainDirection: "流入", MainConsecutiveDirection: "流入", MainConsecutiveDays: 3},
		models.TrendAnalysis{TrendStage: models.TrendStage{Direction: "上升", Strength: 70}},
		patterns,
		forecast,
	)

	require.NotNil(t, result)
	assert.True(t, result.Score >= 68)
	assert.Contains(t, []string{"A", "B"}, result.Grade)
	assert.NotEmpty(t, result.Components)
	assert.NotEmpty(t, result.Reasons)
}

func hasPatternSignal(patterns []models.PatternSignal, name string) bool {
	for _, p := range patterns {
		if p.Pattern == name {
			return true
		}
	}
	return false
}
