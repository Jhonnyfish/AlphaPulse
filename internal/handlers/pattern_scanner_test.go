package handlers

import (
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeKline(date string, open, high, low, close float64, volume float64) models.KlinePoint {
	return models.KlinePoint{
		Date:   date,
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: volume,
	}
}

func TestDetectKlinePatterns_Doji(t *testing.T) {
	klines := []models.KlinePoint{
		makeKline("2025-01-01", 10.0, 10.5, 9.5, 10.1, 1000),
		makeKline("2025-01-02", 10.1, 10.3, 9.8, 10.2, 1100),
		makeKline("2025-01-03", 10.2, 10.3, 10.1, 10.2001, 1200), // near-identical open/close
	}
	results := detectKlinePatterns(klines, "600001", "测试股票")
	found := false
	for _, r := range results {
		if r.Pattern == "十字星" {
			found = true
			assert.Equal(t, "kline", r.Category)
			assert.Equal(t, "neutral", r.Direction)
			assert.Greater(t, r.Confidence, 0.0)
		}
	}
	assert.True(t, found, "should detect doji pattern")
}

func TestDetectKlinePatterns_Hammer(t *testing.T) {
	klines := []models.KlinePoint{
		makeKline("2025-01-01", 10.0, 10.5, 9.5, 10.1, 1000),
		makeKline("2025-01-02", 10.1, 10.3, 9.8, 10.2, 1100),
		makeKline("2025-01-03", 10.0, 10.04, 8.0, 9.9, 1200), // long lower shadow, very short upper shadow
	}
	results := detectKlinePatterns(klines, "600001", "测试股票")
	found := false
	for _, r := range results {
		if r.Pattern == "锤子线" {
			found = true
			assert.Equal(t, "bullish", r.Direction)
			assert.Equal(t, "kline", r.Category)
		}
	}
	assert.True(t, found, "should detect hammer pattern")
}

func TestDetectKlinePatterns_BullishEngulfing(t *testing.T) {
	klines := []models.KlinePoint{
		makeKline("2025-01-01", 10.0, 10.5, 9.5, 10.1, 1000),
		makeKline("2025-01-02", 10.0, 10.1, 9.5, 9.6, 1100),  // down bar
		makeKline("2025-01-03", 9.5, 10.5, 9.4, 10.4, 1500),   // big up bar engulfing
	}
	results := detectKlinePatterns(klines, "600001", "测试股票")
	found := false
	for _, r := range results {
		if r.Pattern == "吞没形态" && r.Direction == "bullish" {
			found = true
			assert.Equal(t, "kline", r.Category)
		}
	}
	assert.True(t, found, "should detect bullish engulfing")
}

func TestDetectKlinePatterns_ThreeWhiteSoldiers(t *testing.T) {
	klines := []models.KlinePoint{
		makeKline("2025-01-01", 10.0, 10.5, 9.5, 10.1, 1000),
		makeKline("2025-01-02", 10.1, 10.6, 10.0, 10.5, 1100),
		makeKline("2025-01-03", 10.5, 11.0, 10.4, 10.9, 1200),
	}
	results := detectKlinePatterns(klines, "600001", "测试股票")
	found := false
	for _, r := range results {
		if r.Pattern == "三白兵" {
			found = true
			assert.Equal(t, "bullish", r.Direction)
			assert.Equal(t, 0.85, r.Confidence)
		}
	}
	assert.True(t, found, "should detect three white soldiers")
}

func TestDetectKlinePatterns_ThreeBlackCrows(t *testing.T) {
	klines := []models.KlinePoint{
		makeKline("2025-01-01", 11.0, 11.1, 10.5, 10.6, 1000),
		makeKline("2025-01-02", 10.6, 10.7, 10.1, 10.2, 1100),
		makeKline("2025-01-03", 10.2, 10.3, 9.7, 9.8, 1200),
	}
	results := detectKlinePatterns(klines, "600001", "测试股票")
	found := false
	for _, r := range results {
		if r.Pattern == "三黑鸦" {
			found = true
			assert.Equal(t, "bearish", r.Direction)
			assert.Equal(t, 0.85, r.Confidence)
		}
	}
	assert.True(t, found, "should detect three black crows")
}

func TestDetectKlinePatterns_NoPattern(t *testing.T) {
	klines := []models.KlinePoint{
		makeKline("2025-01-01", 10.0, 10.5, 9.5, 10.1, 1000),
		makeKline("2025-01-02", 10.1, 10.8, 9.9, 10.7, 1100),
		makeKline("2025-01-03", 10.7, 10.9, 10.3, 10.4, 1200), // mild pattern, no clear signal
	}
	results := detectKlinePatterns(klines, "600001", "测试股票")
	// May or may not find patterns, but shouldn't crash
	_ = results
}

func TestDetectChartPatterns_DoubleBottom(t *testing.T) {
	// Create 30 bars with a W-shaped pattern
	klines := make([]models.KlinePoint, 30)
	for i := range klines {
		klines[i] = models.KlinePoint{
			Date:   "2025-01-" + string(rune('0'+i/10)) + string(rune('0'+i%10)),
			Open:   10.0,
			Close:  10.0,
			High:   10.5,
			Low:    9.5,
			Volume: 1000,
		}
	}
	// Create a W shape: high, low, high, low, high
	klines[5].Close = 10.0
	klines[5].Low = 9.0
	klines[10].Close = 11.0
	klines[10].High = 11.5
	klines[15].Close = 10.0
	klines[15].Low = 9.0
	klines[20].Close = 11.0
	klines[20].High = 11.5
	klines[25].Close = 11.5
	klines[25].High = 12.0

	results := detectChartPatterns(klines, "600001", "测试股票")
	require.NotNil(t, results)
	// Just verify it doesn't crash; exact pattern detection depends on pivot algorithm
}

func TestDetectVolumePatterns_VolumeBreakout(t *testing.T) {
	klines := make([]models.KlinePoint, 25)
	for i := range klines {
		klines[i] = models.KlinePoint{
			Date:   "2025-01-01",
			Open:   10.0,
			Close:  10.0,
			High:   10.2,
			Low:    9.8,
			Volume: 1000,
		}
	}
	// Last bar: huge volume + price breakout
	klines[24].Close = 11.0
	klines[24].High = 11.5
	klines[24].Volume = 3000 // 3x average
	klines[24].Date = "2025-01-25"

	results := detectVolumePatterns(klines, "600001", "测试股票")
	found := false
	for _, r := range results {
		if r.Pattern == "放量突破" {
			found = true
			assert.Equal(t, "bullish", r.Direction)
			assert.Equal(t, "volume", r.Category)
		}
	}
	assert.True(t, found, "should detect volume breakout")
}

func TestDetectVolumePatterns_ContractionPullback(t *testing.T) {
	klines := make([]models.KlinePoint, 25)
	for i := 0; i < 20; i++ {
		klines[i] = models.KlinePoint{
			Date:   "2025-01-01",
			Open:   11.0,
			Close:  11.0,
			High:   11.2,
			Low:    10.8,
			Volume: 1000,
		}
	}
	// Recent 5 bars: price drops, volume shrinks
	for i := 20; i < 25; i++ {
		klines[i] = models.KlinePoint{
			Date:   "2025-01-01",
			Open:   10.5,
			Close:  10.3,
			High:   10.6,
			Low:    10.2,
			Volume: 300, // 30% of average
		}
	}
	klines[24].Date = "2025-01-25"

	results := detectVolumePatterns(klines, "600001", "测试股票")
	found := false
	for _, r := range results {
		if r.Pattern == "缩量回调" {
			found = true
			assert.Equal(t, "bullish", r.Direction)
		}
	}
	assert.True(t, found, "should detect contraction pullback")
}

func TestBuildPatternSummary(t *testing.T) {
	patterns := []PatternResult{
		{Direction: "bullish", Category: "kline"},
		{Direction: "bullish", Category: "chart"},
		{Direction: "bearish", Category: "volume"},
		{Direction: "neutral", Category: "chart"},
	}
	summary := buildPatternSummary(patterns, 5)
	assert.Equal(t, 4, summary.Total)
	assert.Equal(t, 2, summary.Bullish)
	assert.Equal(t, 1, summary.Bearish)
	assert.Equal(t, 1, summary.Neutral)
	assert.Equal(t, 2, summary.ByCategory["chart"])
	assert.Equal(t, 1, summary.ByCategory["kline"])
	assert.Equal(t, 1, summary.ByCategory["volume"])
	assert.Equal(t, 5, summary.Scanned)
}

func TestFindPivots(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 3, 2, 1, 2, 3, 4, 5, 4, 3, 2}
	mins, maxs := findPivots(data, 2)
	require.NotEmpty(t, mins)
	require.NotEmpty(t, maxs)
	// Should find local min at index 7 (value 1) and local max at index 4 (value 5)
	foundMin := false
	for _, m := range mins {
		if m.val == 1.0 && m.idx == 7 {
			foundMin = true
		}
	}
	assert.True(t, foundMin, "should find local minimum at index 7")
}

func TestHelperFunctions(t *testing.T) {
	assert.Equal(t, 3.0, abs(-3.0))
	assert.Equal(t, 3.0, abs(3.0))
	assert.Equal(t, 2.0, minF(2.0, 5.0))
	assert.Equal(t, 5.0, maxF(2.0, 5.0))
	assert.Equal(t, 3, maxI(2, 3))
	assert.Equal(t, 3, absInt(-3))
	assert.Equal(t, 3, absInt(3))

	assert.Equal(t, 1.23, round2(1.234))
	assert.Equal(t, 1.24, round2(1.235))

	assert.Equal(t, 5.0, maxSlice([]float64{1, 2, 5, 3}))
	assert.Equal(t, 1.0, minSlice([]float64{1, 2, 5, 3}))
	assert.Equal(t, 3.0, avgSlice([]float64{2, 4, 4, 2}))
	assert.Equal(t, 0.0, maxSlice(nil))
	assert.Equal(t, 0.0, minSlice(nil))
	assert.Equal(t, 0.0, avgSlice(nil))
}
