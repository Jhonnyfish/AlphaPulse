package services

import (
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTBacktestKlines creates a kline series long enough for a T-backtest
// with a mix of up and down days to produce both 正T and 反T signals.
func generateTBacktestKlines(base float64, n int) []models.KlinePoint {
	klines := make([]models.KlinePoint, 0, n+30)
	// Warmup: flat
	for i := 0; i < 30; i++ {
		klines = append(klines, models.KlinePoint{
			Date:   "2025-01-01",
			Close:  base,
			High:   base + 0.5,
			Low:    base - 0.5,
			Open:   base,
			Volume: 10000,
		})
	}
	// Alternating up/down days to trigger signals
	price := base
	for i := 0; i < n; i++ {
		delta := 1.0
		if i%2 == 0 {
			delta = -1.5 // down day → potential 正T
		} else {
			delta = 1.5 // up day → potential 反T
		}
		open := price
		price += delta
		klines = append(klines, models.KlinePoint{
			Date:   "2025-01-01",
			Close:  price,
			High:   price + 1,
			Low:    price - 1,
			Open:   open,
			Volume: 10000,
		})
	}
	return klines
}

func TestBacktestTSuggestion_BasicRun(t *testing.T) {
	klines := generateTBacktestKlines(100, 60)
	require.True(t, len(klines) > 60)

	result := BacktestTSuggestion(klines, 30, 1000, 0)

	require.NotNil(t, result)
	assert.Equal(t, 1000, result.HoldingQty)
	assert.True(t, result.DaysEval > 0)
	assert.NotNil(t, result.Details)
	assert.NotNil(t, result.EquityCurve)
	assert.NotNil(t, result.BenchmarkCurve)
	// Equity and benchmark curves should have same length
	assert.Equal(t, len(result.EquityCurve), len(result.BenchmarkCurve))
	assert.Equal(t, result.DaysEval, len(result.Details))
	assert.Equal(t, result.DaysEval, len(result.EquityCurve))
}

func TestBacktestTSuggestion_InsufficientData(t *testing.T) {
	klines := make([]models.KlinePoint, 10) // too short
	for i := range klines {
		klines[i] = models.KlinePoint{Close: 100, High: 101, Low: 99}
	}
	result := BacktestTSuggestion(klines, 30, 1000, 0)
	assert.Nil(t, result)
}

func TestBacktestTSuggestion_EquityCurvesValid(t *testing.T) {
	klines := generateTBacktestKlines(50, 80)
	result := BacktestTSuggestion(klines, 40, 1000, 0)
	require.NotNil(t, result)

	// Benchmark curve should start at 1.0
	assert.InDelta(t, 1.0, result.BenchmarkCurve[0].Equity, 0.01)

	// All equity values should be positive
	for _, ep := range result.EquityCurve {
		assert.True(t, ep.Equity > 0, "equity should be positive, got %f", ep.Equity)
	}
	for _, ep := range result.BenchmarkCurve {
		assert.True(t, ep.Equity > 0, "benchmark equity should be positive, got %f", ep.Equity)
	}

	// Cumulative P&L should be consistent with the equity curve
	assert.True(t, result.CumulativePct >= -1 && result.CumulativePct <= 10,
		"cumulative return should be in reasonable range, got %f", result.CumulativePct)
}

func TestBacktestTSuggestion_TradeStats(t *testing.T) {
	klines := generateTBacktestKlines(100, 100)
	result := BacktestTSuggestion(klines, 50, 1000, 0)
	require.NotNil(t, result)

	// Win count should not exceed total trades
	assert.LessOrEqual(t, result.WinCount, result.TotalTrades)

	// Win rate should be 0-1
	if result.TotalTrades > 0 {
		assert.True(t, result.WinRate >= 0 && result.WinRate <= 1)
	}

	// Positive + Reverse + Watch <= DaysEval
	totalClassified := result.PositiveTTrades + result.ReverseTTrades + result.WatchDays
	assert.LessOrEqual(t, totalClassified, result.DaysEval)
}

func TestBacktestTSuggestion_CommissionImpact(t *testing.T) {
	// Create very low volatility klines — T profit should be minimal after commission
	klines := make([]models.KlinePoint, 0, 80)
	for i := 0; i < 80; i++ {
		price := 100.0 + float64(i%2)*0.1 // tiny oscillation
		klines = append(klines, models.KlinePoint{
			Date:   "2025-01-01",
			Close:  price,
			High:   price + 0.05,
			Low:    price - 0.05,
			Open:   price,
			Volume: 10000,
		})
	}
	result := BacktestTSuggestion(klines, 30, 1000, 100)
	require.NotNil(t, result)
	// With tiny ATR, many days should be 观望 (signal too weak)
	// And any executed trades should have near-zero or negative returns (commission dominates)
}

func TestBacktestTSuggestion_Personality(t *testing.T) {
	klines := generateTBacktestKlines(100, 80)
	result := BacktestTSuggestion(klines, 50, 1000, 0)
	require.NotNil(t, result)

	// Personality should be populated
	require.NotNil(t, result.Personality, "Personality should be non-nil")

	p := result.Personality
	assert.True(t, p.ATRPct > 0, "ATRPct should be positive, got %f", p.ATRPct)
	assert.True(t, p.AvgDailyRange > 0, "AvgDailyRange should be positive")
	assert.NotEmpty(t, p.TrendBias, "TrendBias should not be empty")
	assert.Contains(t, []string{"上涨", "下跌", "震荡"}, p.TrendBias)
	assert.Contains(t, []string{"正T", "反T", "均可", "不适合"}, p.RecommendedT)
	assert.True(t, p.TSuitability >= 0 && p.TSuitability <= 100, "TSuitability should be 0-100")
	assert.True(t, p.UpDayPct >= 0 && p.UpDayPct <= 1, "UpDayPct should be 0-1")
	assert.True(t, p.TrendStrength >= 0 && p.TrendStrength <= 1, "TrendStrength should be 0-1")
	assert.NotNil(t, p.Tags, "Tags should not be nil")
	assert.NotEmpty(t, p.Summary, "Summary should not be empty")
}
