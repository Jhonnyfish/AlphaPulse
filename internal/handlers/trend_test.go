package handlers

import (
	"fmt"
	"math"
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPearsonCorrelation(t *testing.T) {
	tests := []struct {
		name     string
		x, y     []float64
		expected float64
	}{
		{
			name:     "perfect positive correlation",
			x:        []float64{1, 2, 3, 4, 5},
			y:        []float64{2, 4, 6, 8, 10},
			expected: 1.0,
		},
		{
			name:     "perfect negative correlation",
			x:        []float64{1, 2, 3, 4, 5},
			y:        []float64{10, 8, 6, 4, 2},
			expected: -1.0,
		},
		{
			name:     "no correlation (insufficient data)",
			x:        []float64{1},
			y:        []float64{2},
			expected: 0,
		},
		{
			name:     "identical series",
			x:        []float64{0.01, -0.02, 0.03, 0.01, -0.01},
			y:        []float64{0.01, -0.02, 0.03, 0.01, -0.01},
			expected: 1.0,
		},
		{
			name:     "zero variance",
			x:        []float64{5, 5, 5, 5, 5},
			y:        []float64{1, 2, 3, 4, 5},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pearsonCorrelation(tt.x, tt.y)
			if tt.expected == 0 {
				assert.Equal(t, 0.0, result)
			} else {
				assert.InDelta(t, tt.expected, result, 0.001)
			}
		})
	}
}

func TestComputeMA(t *testing.T) {
	closes := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}

	ma5 := computeMA(closes, 5)
	// Last 5: 16, 17, 18, 19, 20 → avg = 18
	assert.Equal(t, 18.0, ma5)

	ma10 := computeMA(closes, 10)
	// Last 10: 11..20 → avg = 15.5
	assert.Equal(t, 15.5, ma10)

	// Insufficient data
	ma20 := computeMA(closes, 20)
	assert.Equal(t, 0.0, ma20)
}

func TestComputeRSI(t *testing.T) {
	// All gains → RSI should be 100
	allGains := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}
	rsi := computeRSI(allGains, 14)
	assert.Equal(t, 100.0, rsi)

	// All losses → RSI should be 0
	allLosses := []float64{25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10}
	rsi = computeRSI(allLosses, 14)
	assert.Equal(t, 0.0, rsi)

	// Insufficient data
	short := []float64{10, 11, 12}
	rsi = computeRSI(short, 14)
	assert.Equal(t, 0.0, rsi)
}

func TestComputeVolumeTrend(t *testing.T) {
	// Increasing volumes
	increasing := []float64{100, 100, 100, 100, 100, 150, 150, 150, 150, 150}
	assert.Equal(t, "increasing", computeVolumeTrend(increasing))

	// Decreasing volumes
	decreasing := []float64{150, 150, 150, 150, 150, 100, 100, 100, 100, 100}
	assert.Equal(t, "decreasing", computeVolumeTrend(decreasing))

	// Stable volumes
	stable := []float64{100, 100, 100, 100, 100, 100, 100, 100, 100, 100}
	assert.Equal(t, "stable", computeVolumeTrend(stable))

	// Too few data points
	short := []float64{100, 100}
	assert.Equal(t, "stable", computeVolumeTrend(short))
}

func TestAggregateKlinesToWeekly(t *testing.T) {
	klines := []models.KlinePoint{
		{Date: "2024-01-01", Open: 10, Close: 12, High: 13, Low: 9, Volume: 100, Amount: 1000},
		{Date: "2024-01-02", Open: 12, Close: 14, High: 15, Low: 11, Volume: 200, Amount: 2000},
		{Date: "2024-01-03", Open: 14, Close: 13, High: 16, Low: 12, Volume: 150, Amount: 1500},
	}

	weekly := aggregateKlinesToWeekly(klines)
	require.Len(t, weekly, 1)

	w := weekly[0]
	assert.Equal(t, "2024-01-01", w.Date)
	assert.Equal(t, 10.0, w.Open)
	assert.Equal(t, 13.0, w.Close)
	assert.Equal(t, 16.0, w.High)
	assert.Equal(t, 9.0, w.Low)
	assert.Equal(t, 450.0, w.Volume)
	assert.Equal(t, 4500.0, w.Amount)

	// Empty input
	assert.Nil(t, aggregateKlinesToWeekly(nil))
	assert.Nil(t, aggregateKlinesToWeekly([]models.KlinePoint{}))
}

func TestAggregateKlinesToMonthly(t *testing.T) {
	klines := []models.KlinePoint{
		{Date: "2024-01-15", Open: 10, Close: 12, High: 13, Low: 9, Volume: 100, Amount: 1000},
		{Date: "2024-01-20", Open: 12, Close: 14, High: 15, Low: 11, Volume: 200, Amount: 2000},
		{Date: "2024-02-01", Open: 14, Close: 16, High: 17, Low: 13, Volume: 300, Amount: 3000},
	}

	monthly := aggregateKlinesToMonthly(klines)
	require.Len(t, monthly, 2)

	assert.Equal(t, "2024-01-15", monthly[0].Date)
	assert.Equal(t, 10.0, monthly[0].Open)
	assert.Equal(t, 14.0, monthly[0].Close)
	assert.Equal(t, 15.0, monthly[0].High)
	assert.Equal(t, 9.0, monthly[0].Low)
	assert.Equal(t, 300.0, monthly[0].Volume)

	assert.Equal(t, "2024-02-01", monthly[1].Date)
	assert.Equal(t, 14.0, monthly[1].Open)
	assert.Equal(t, 16.0, monthly[1].Close)

	// Empty input
	assert.Nil(t, aggregateKlinesToMonthly(nil))
}

func TestComputePeriodStrength(t *testing.T) {
	// Strong uptrend with aligned MAs
	uptrend := make([]models.KlinePoint, 30)
	for i := range uptrend {
		price := 10.0 + float64(i)*0.5
		vol := 1000.0 * math.Pow(1.05, float64(i)) // exponential growth > 15% over 5 bars
		uptrend[i] = models.KlinePoint{
			Date:   "2024-01-" + fmt.Sprintf("%02d", i+1),
			Open:   price,
			Close:  price + 0.2,
			High:   price + 0.5,
			Low:    price - 0.1,
			Volume: vol,
			Amount: 10000,
		}
	}
	result := computePeriodStrength(uptrend)
	assert.Greater(t, result.Strength, 50, "uptrend should have strength > 50")
	assert.True(t, result.ReturnPct > 0, "uptrend should have positive return")

	// Insufficient data → default values
	short := []models.KlinePoint{
		{Date: "2024-01-01", Open: 10, Close: 11, High: 12, Low: 9, Volume: 100},
	}
	result = computePeriodStrength(short)
	assert.Equal(t, 50, result.Strength)
	assert.Equal(t, "stable", result.VolumeTrend)
}

func TestPearsonCorrelationSymmetry(t *testing.T) {
	x := []float64{0.01, -0.02, 0.03, 0.01, -0.01, 0.02, 0.005, -0.015}
	y := []float64{0.02, -0.01, 0.025, 0.015, -0.005, 0.01, 0.01, -0.01}

	corrXY := pearsonCorrelation(x, y)
	corrYX := pearsonCorrelation(y, x)
	assert.InDelta(t, corrXY, corrYX, 0.0001, "correlation should be symmetric")
}

func TestPearsonCorrelationRange(t *testing.T) {
	// Correlation should always be in [-1, 1]
	x := []float64{1.5, 2.3, 3.1, 4.7, 5.2, 6.8, 7.1, 8.9}
	y := []float64{8.1, 7.3, 6.5, 5.2, 4.8, 3.1, 2.5, 1.3}

	corr := pearsonCorrelation(x, y)
	assert.True(t, corr >= -1.0 && corr <= 1.0, "correlation should be in [-1, 1], got %f", corr)
}

func TestComputePeriodStrengthClamping(t *testing.T) {
	// Extreme downtrend should still clamp to >= 0
	downtrend := make([]models.KlinePoint, 20)
	for i := range downtrend {
		price := 100.0 - float64(i)*5
		downtrend[i] = models.KlinePoint{
			Date:   "2024-01-01",
			Open:   price,
			Close:  price - 2,
			High:   price + 1,
			Low:    price - 3,
			Volume: 100,
			Amount: 1000,
		}
	}
	result := computePeriodStrength(downtrend)
	assert.True(t, result.Strength >= 0, "strength should be >= 0, got %d", result.Strength)
	assert.True(t, result.Strength <= 100, "strength should be <= 100, got %d", result.Strength)
}

func TestRSIMidRange(t *testing.T) {
	// Oscillating series: up, down, up, down repeatedly
	oscillating := make([]float64, 20)
	for i := range oscillating {
		if i%2 == 0 {
			oscillating[i] = 10.0 + 0.1
		} else {
			oscillating[i] = 10.0 - 0.1
		}
	}
	rsi := computeRSI(oscillating, 14)
	// With oscillating gains/losses, RSI should be between 40 and 60
	assert.True(t, rsi > 30 && rsi < 70, "oscillating series should have RSI near 50, got %f", rsi)
}

func TestComputeMADifferentPeriods(t *testing.T) {
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = 10 + float64(i)*0.5
	}

	ma5 := computeMA(closes, 5)
	ma10 := computeMA(closes, 10)
	ma20 := computeMA(closes, 20)

	// In uptrend, shorter MA should be > longer MA
	assert.True(t, ma5 > ma10, "MA5 should be > MA10 in uptrend")
	assert.True(t, ma10 > ma20, "MA10 should be > MA20 in uptrend")

	_ = math.Abs(ma5) // ensure no panic
}
