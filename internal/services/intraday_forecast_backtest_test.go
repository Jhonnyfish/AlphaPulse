package services

import (
	"math"
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBacktestIntradayForecast_SyntheticTrend runs the backtest on a synthetic
// gentle uptrend and checks that:
//   - the function returns a result
//   - the number of evaluated days matches the request
//   - all percentage metrics are within [0, 1]
//   - the reliability grade is one of the documented values
//   - per-day details are populated and self-consistent
//   - wide-band (±1σ) hit rate ≥ central hit rate (by definition)
func TestBacktestIntradayForecast_SyntheticTrend(t *testing.T) {
	const totalBars = 200
	const daysBack = 60

	closes := newKlineBuilder(50.0).
		moveTo(80, totalBars-1).
		build()
	require.Len(t, closes, totalBars)

	klines := make([]models.KlinePoint, totalBars)
	for i, c := range closes {
		klines[i] = models.KlinePoint{
			Date:   "2026-01-01",
			Open:   c,
			Close:  c,
			High:   c + 0.5,
			Low:    c - 0.5,
			Volume: 1000,
		}
	}

	acc := BacktestIntradayForecast(klines, daysBack)
	require.NotNil(t, acc)
	assert.Equal(t, daysBack, acc.DaysEvaluated)
	assert.Len(t, acc.Details, daysBack)

	assert.GreaterOrEqual(t, acc.BothInRange, 0.0)
	assert.LessOrEqual(t, acc.BothInRange, 1.0)
	assert.GreaterOrEqual(t, acc.HighInRange, 0.0)
	assert.LessOrEqual(t, acc.HighInRange, 1.0)
	assert.GreaterOrEqual(t, acc.LowInRange, 0.0)
	assert.LessOrEqual(t, acc.LowInRange, 1.0)

	// Wide band must be ≥ central band (it's a strict superset).
	assert.GreaterOrEqual(t, acc.BothInWideRange, acc.BothInRange)
	assert.GreaterOrEqual(t, acc.HighInWideRange, acc.HighInRange)
	assert.GreaterOrEqual(t, acc.LowInWideRange, acc.LowInRange)

	// σ must be positive and wide width must be > central width.
	assert.Greater(t, acc.AvgSigma, 0.0)
	assert.Greater(t, acc.AvgWideWidth, acc.AvgRangeWidth)

	assert.Contains(t, []string{"high_confidence", "moderate", "low_confidence"}, acc.Reliability)
	assert.Greater(t, acc.AvgRangeWidth, 0.0)
	assert.Greater(t, acc.AvgRangeWidthPct, 0.0)

	// Sanity-check a single day.
	d := acc.Details[0]
	assert.GreaterOrEqual(t, d.ActualHigh, d.ActualLow)
	assert.GreaterOrEqual(t, d.PredictedHigh, d.PredictedLow)
	assert.GreaterOrEqual(t, d.PredictedHighUp, d.PredictedHigh)
	assert.LessOrEqual(t, d.PredictedLowDown, d.PredictedLow)
	assert.Equal(t, d.HighInRange && d.LowInRange, d.BothInRange)
	assert.Equal(t, d.HighInWideRange && d.LowInWideRange, d.BothInWideRange)
}

// TestBacktestIntradayForecast_InsufficientHistory returns nil when there
// are not enough warmup bars.
func TestBacktestIntradayForecast_InsufficientHistory(t *testing.T) {
	klines := make([]models.KlinePoint, 20)
	acc := BacktestIntradayForecast(klines, 60)
	assert.Nil(t, acc)
}

// TestBacktestIntradayForecast_AdaptiveClamp verifies that when fewer than
// `daysBack` evaluable days are available, the function clamps the window
// down instead of returning nil.
func TestBacktestIntradayForecast_AdaptiveClamp(t *testing.T) {
	// 80 klines total → warmup=30, so 50 evaluable days.
	// Requesting 120 should clamp to 50.
	const totalBars = 80
	closes := newKlineBuilder(50.0).moveTo(80, totalBars-1).build()
	require.Len(t, closes, totalBars)

	klines := make([]models.KlinePoint, totalBars)
	for i, c := range closes {
		klines[i] = models.KlinePoint{
			Date: "2026-01-01", Open: c, Close: c,
			High: c + 0.5, Low: c - 0.5, Volume: 1000,
		}
	}

	acc := BacktestIntradayForecast(klines, 120)
	require.NotNil(t, acc)
	assert.Equal(t, 50, acc.DaysEvaluated, "should clamp to available evaluable days")
	assert.Len(t, acc.Details, 50)
}

// ──────────────────────────────────────────────
// Confidence interval (σ) tests
// ──────────────────────────────────────────────

func TestQuantile(t *testing.T) {
	// Empty / single
	assert.Equal(t, 0.0, quantile(nil, 0.5))
	assert.Equal(t, 42.0, quantile([]float64{42}, 0.5))

	// Uniform [1..5]
	vals := []float64{5, 1, 4, 2, 3} // unsorted on purpose
	assert.Equal(t, 1.0, quantile(vals, 0.0))
	assert.Equal(t, 5.0, quantile(vals, 1.0))
	assert.Equal(t, 3.0, quantile(vals, 0.5))

	// 70th percentile of [10,20,30,40,50,60,70,80,90,100] via linear interp.
	// rank = 0.7 * 9 = 6.3 → between idx 6 (70) and idx 7 (80) → 70 + 0.3*10 = 73.
	decade := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	assert.InDelta(t, 73.0, quantile(decade, 0.70), 1e-9)
	assert.InDelta(t, 91.0, quantile(decade, 0.90), 1e-9)
}

func TestStddev(t *testing.T) {
	// Identical values → 0.
	assert.InDelta(t, 0.0, stddev([]float64{5, 5, 5, 5}), 1e-9)
	// [1, 2, 3] → mean=2, variance=1, σ=1.
	assert.InDelta(t, 1.0, stddev([]float64{1, 2, 3}), 1e-9)
	// < 2 samples → 0.
	assert.Equal(t, 0.0, stddev([]float64{1}))
	assert.Equal(t, 0.0, stddev(nil))
}

func TestAnalyzeIntradayForecast_ConfidenceInterval(t *testing.T) {
	// Build 40 klines around price 50 with low, stable volatility.
	klines := make([]models.KlinePoint, 40)
	for i := range klines {
		c := 50.0
		klines[i] = models.KlinePoint{
			Date:   "2026-01-01",
			Open:   c, Close: c,
			High:   c + 0.5, Low: c - 0.5,
			Volume: 1000,
		}
	}
	quote := models.Quote{Price: 50, PrevClose: 50}
	atr := ComputeATR(klines, 14)
	require.Greater(t, atr, 0.0)

	fc := AnalyzeIntradayForecast(klines, quote, atr, nil, 0, 0)
	require.NotNil(t, fc)

	// Wide band must extend at least as far as central (with constant
	// input, all excursions are equal → P83 == P95 → wide == central).
	assert.GreaterOrEqual(t, fc.PredictedHighUp, fc.PredictedHigh)
	assert.LessOrEqual(t, fc.PredictedLowDown, fc.PredictedLow)

	// σ is the distance from central to wide and is non-negative.
	assert.GreaterOrEqual(t, fc.SigmaHigh, 0.0)
	assert.GreaterOrEqual(t, fc.SigmaLow, 0.0)

	// On symmetric synthetic data σ_high ≈ σ_low.
	assert.InDelta(t, fc.SigmaHigh, fc.SigmaLow, 0.05)
}

// ──────────────────────────────────────────────
// Pattern-weighted bias test
// ──────────────────────────────────────────────

func TestAnalyzeIntradayForecast_PatternConfidenceWeighted(t *testing.T) {
	klines := make([]models.KlinePoint, 40)
	for i := range klines {
		c := 50.0 + math.Sin(float64(i)/3)*0.5
		klines[i] = models.KlinePoint{
			Date:   "2026-01-01",
			Open:   c, Close: c,
			High:   c + 0.5, Low: c - 0.5,
			Volume: 1000,
		}
	}
	quote := models.Quote{Price: 50, PrevClose: 50}
	atr := ComputeATR(klines, 14)

	// Three forecasts: baseline, low-conf bullish (0.5), high-conf bullish (0.95).
	fcA := AnalyzeIntradayForecast(klines, quote, atr, nil, 0, 0)
	fcB := AnalyzeIntradayForecast(klines, quote, atr, []models.PatternSignal{
		{Pattern: "x", Direction: "bullish", Confidence: 0.5},
	}, 0, 0)
	fcC := AnalyzeIntradayForecast(klines, quote, atr, []models.PatternSignal{
		{Pattern: "x", Direction: "bullish", Confidence: 0.95},
	}, 0, 0)

	require.NotNil(t, fcA)
	require.NotNil(t, fcB)
	require.NotNil(t, fcC)

	// BiasStrength is the confidence-weighted score, monotonic in confidence.
	assert.InDelta(t, 0.0, fcA.BiasStrength, 1e-9)
	assert.InDelta(t, 0.5, fcB.BiasStrength, 1e-9)
	assert.InDelta(t, 0.95, fcC.BiasStrength, 1e-9)

	// Bias direction is correctly classified.
	assert.Equal(t, "neutral", fcA.Bias)
	assert.Equal(t, "bullish", fcB.Bias)
	assert.Equal(t, "bullish", fcC.Bias)

	// The actual high prediction must be ≥ baseline (bias never reduces it).
	// Note: in low-volatility scenarios the Bollinger clamp can equalize the
	// final values, so we assert non-strict ordering here.
	assert.GreaterOrEqual(t, fcB.PredictedHigh, fcA.PredictedHigh)
	assert.GreaterOrEqual(t, fcC.PredictedHigh, fcA.PredictedHigh)
}

// ──────────────────────────────────────────────
// Empirical-quantile hit-rate test
// ──────────────────────────────────────────────
// Run the backtest on synthetic data with a known excursion distribution and
// verify that the empirical central hit rate is close to the 70% target the
// P70 quantile promises, and the wide hit rate is close to 90%.
//
// We build 200 bars where daily high = prev_close × (1 + uniform[0, 0.04])
// and daily low = prev_close × (1 - uniform[0, 0.04]). The empirical P70 of
// U[0, 0.04] is 0.028 and P90 is 0.036, so the central band should hit ~70%
// and the wide band ~90%. We allow ±15 pp slack since the test sample is
// finite (60 evaluable days).
func TestBacktestIntradayForecast_QuantileHitRate(t *testing.T) {
	const totalBars = 200
	const daysBack = 60

	rng := newDeterministicRng(42)
	klines := make([]models.KlinePoint, totalBars)
	c := 50.0
	for i := 0; i < totalBars; i++ {
		upExc := rng.float64() * 0.04   // U[0, 0.04]
		downExc := rng.float64() * 0.04 // U[0, 0.04]
		high := c * (1 + upExc)
		low := c * (1 - downExc)
		nextC := (high + low) / 2 // arbitrary
		klines[i] = models.KlinePoint{
			Date: "2026-01-01", Open: c, Close: nextC,
			High: high, Low: low, Volume: 1000,
		}
		c = nextC
	}

	acc := BacktestIntradayForecast(klines, daysBack)
	require.NotNil(t, acc)
	require.GreaterOrEqual(t, acc.DaysEvaluated, daysBack-5,
		"should evaluate most of the requested window")

	// Central band targets 70%, allow [55%, 85%] window.
	assert.GreaterOrEqual(t, acc.BothInRange, 0.55,
		"central hit rate too low for P70-targeted forecast: %.2f", acc.BothInRange)
	assert.LessOrEqual(t, acc.BothInRange, 0.90,
		"central hit rate suspiciously high — band may be over-wide: %.2f", acc.BothInRange)

	// Wide band targets 90%, allow [75%, 100%].
	assert.GreaterOrEqual(t, acc.BothInWideRange, 0.75,
		"wide hit rate too low for P90-targeted forecast: %.2f", acc.BothInWideRange)

	// Wide must be ≥ central (it's a strict superset).
	assert.GreaterOrEqual(t, acc.BothInWideRange, acc.BothInRange)
}

// deterministicRng is a tiny linear-congruential generator so the test is
// reproducible without taking a dependency on math/rand seeding.
type deterministicRng struct{ state uint64 }

func newDeterministicRng(seed uint64) *deterministicRng {
	return &deterministicRng{state: seed | 1} // |1 ensures non-zero
}

func (r *deterministicRng) float64() float64 {
	// LCG with Numerical Recipes constants.
	r.state = r.state*6364136223846793005 + 1442695040888963407
	// Use top 32 bits for the mantissa so the low-bit patterns don't bias
	// the resulting float.
	return float64(r.state>>33) / float64(1<<31)
}
