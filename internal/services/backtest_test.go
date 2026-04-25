package services

import (
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScoreFromKlines_InsufficientData(t *testing.T) {
	klines := makeKlines(10, 100, 0.5)
	score, dims := ScoreFromKlines(klines)
	assert.Equal(t, 0, score)
	assert.Equal(t, "数据不足", dims["error"])
}

func TestScoreFromKlines_BullishPattern(t *testing.T) {
	// Create an uptrend: prices rising steadily
	klines := makeUptrendKlines(30, 100.0, 1.5, 500000)
	score, dims := ScoreFromKlines(klines)
	require.GreaterOrEqual(t, score, 50, "uptrend should score >= 50, got %d", score)
	t.Logf("Uptrend score: %d, dims: %v", score, dims)
}

func TestScoreFromKlines_BearishPattern(t *testing.T) {
	// Create a downtrend: prices falling steadily
	klines := makeDowntrendKlines(30, 200.0, 1.5, 500000)
	score, dims := ScoreFromKlines(klines)
	require.LessOrEqual(t, score, 70, "downtrend should score <= 70, got %d", score)
	t.Logf("Downtrend score: %d, dims: %v", score, dims)
}

func TestScoreFromKlines_ScoreRange(t *testing.T) {
	klines := makeKlines(30, 100.0, 0.5)
	score, _ := ScoreFromKlines(klines)
	assert.GreaterOrEqual(t, score, 0)
	assert.LessOrEqual(t, score, 100)
}

func TestMaxDrawdown(t *testing.T) {
	tests := []struct {
		name     string
		equity   []float64
		expected float64
	}{
		{"empty", nil, 0.0},
		{"no drawdown", []float64{1.0, 1.1, 1.2, 1.3}, 0.0},
		{"simple", []float64{1.0, 1.2, 0.9, 1.1}, 25.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dd := maxDrawdown(tt.equity)
			assert.InDelta(t, tt.expected, dd, 0.01)
		})
	}
}

// makeKlines creates N klines with random-ish data around a base price.
func makeKlines(n int, basePrice, amplitude float64) []models.KlinePoint {
	klines := make([]models.KlinePoint, n)
	for i := 0; i < n; i++ {
		p := basePrice + amplitude*float64(i%7-3)
		klines[i] = models.KlinePoint{
			Date:   "2026-01-" + padInt(i+1),
			Open:   p - 0.5,
			Close:  p,
			High:   p + 1.0,
			Low:    p - 1.0,
			Volume: 100000,
		}
	}
	return klines
}

// makeUptrendKlines creates N klines in a steady uptrend.
func makeUptrendKlines(n int, startPrice, step float64, baseVol float64) []models.KlinePoint {
	klines := make([]models.KlinePoint, n)
	for i := 0; i < n; i++ {
		p := startPrice + step*float64(i)
		vol := baseVol + float64(i)*10000
		klines[i] = models.KlinePoint{
			Date:   "2026-01-" + padInt(i+1),
			Open:   p - step*0.3,
			Close:  p,
			High:   p + step*0.5,
			Low:    p - step*0.5,
			Volume: vol,
		}
	}
	return klines
}

// makeDowntrendKlines creates N klines in a steady downtrend.
func makeDowntrendKlines(n int, startPrice, step float64, baseVol float64) []models.KlinePoint {
	klines := make([]models.KlinePoint, n)
	for i := 0; i < n; i++ {
		p := startPrice - step*float64(i)
		if p < 1 {
			p = 1
		}
		vol := baseVol - float64(i)*5000
		if vol < 10000 {
			vol = 10000
		}
		klines[i] = models.KlinePoint{
			Date:   "2026-01-" + padInt(i+1),
			Open:   p + step*0.3,
			Close:  p,
			High:   p + step*0.5,
			Low:    p - step*0.5,
			Volume: vol,
		}
	}
	return klines
}

func padInt(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
