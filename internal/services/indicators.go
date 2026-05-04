package services

import (
	"math"

	"alphapulse/internal/models"
)

// ──────────────────────────────────────────────
// Result types
// ──────────────────────────────────────────────

// MACDResult holds MACD indicator values.
type MACDResult struct {
	DIF      float64 `json:"dif"`
	DEA      float64 `json:"dea"`
	Histogram float64 `json:"histogram"` // DIF - DEA
	Signal   string  `json:"signal"`     // golden_cross / death_cross / neutral
}

// KDJResult holds KDJ indicator values.
type KDJResult struct {
	K      float64 `json:"k"`
	D      float64 `json:"d"`
	J      float64 `json:"j"`
	Signal string  `json:"signal"` // overbought / oversold / neutral
}

// RSIResult holds RSI indicator values for multiple periods.
type RSIResult struct {
	RSI6   float64 `json:"rsi6"`
	RSI12  float64 `json:"rsi12"`
	RSI24  float64 `json:"rsi24"`
	Signal string  `json:"signal"` // overbought / oversold / neutral
}

// BollResult holds Bollinger Band values.
type BollResult struct {
	Upper  float64 `json:"upper"`
	Middle float64 `json:"middle"`
	Lower  float64 `json:"lower"`
	Width  float64 `json:"width"`  // (upper - lower) / middle
	Position float64 `json:"position"` // 0-100, where price is within the band
	Signal string  `json:"signal"` // above_upper / below_lower / within
}

// MAAlignmentResult holds moving average values and alignment.
type MAAlignmentResult struct {
	MA5       float64 `json:"ma5"`
	MA10      float64 `json:"ma10"`
	MA20      float64 `json:"ma20"`
	MA60      float64 `json:"ma60"`
	Aligned   bool    `json:"aligned"`   // true if MA5 > MA10 > MA20 (bullish)
	BearAlign bool    `json:"bear_align"` // true if MA5 < MA10 < MA20
	Score     int     `json:"score"`     // 0-100 alignment score
}

// TechnicalIndicators aggregates all computed indicators.
type TechnicalIndicators struct {
	MACD      MACDResult       `json:"macd"`
	KDJ       KDJResult        `json:"kdj"`
	RSI       RSIResult        `json:"rsi"`
	Boll      BollResult       `json:"boll"`
	MA        MAAlignmentResult `json:"ma"`
	Klines    int              `json:"klines"` // number of klines used
}

// ──────────────────────────────────────────────
// Main entry point
// ──────────────────────────────────────────────

// ComputeIndicators calculates all technical indicators from kline data.
func ComputeIndicators(klines []models.KlinePoint) TechnicalIndicators {
	if len(klines) < 5 {
		return TechnicalIndicators{Klines: len(klines)}
	}

	closes := extractCloses(klines)
	highs := extractHighs(klines)
	lows := extractLows(klines)

	return TechnicalIndicators{
		MACD:   ComputeMACD(closes),
		KDJ:    ComputeKDJ(closes, highs, lows),
		RSI:    ComputeRSI(closes),
		Boll:   ComputeBoll(closes),
		MA:     ComputeMAAlignment(closes),
		Klines: len(klines),
	}
}

// ──────────────────────────────────────────────
// MACD (12/26/9)
// ──────────────────────────────────────────────

// ComputeMACD calculates MACD with standard 12/26/9 parameters.
func ComputeMACD(closes []float64) MACDResult {
	if len(closes) < 26 {
		return MACDResult{}
	}

	ema12 := computeEMA(closes, 12)
	ema26 := computeEMA(closes, 26)

	// DIF = EMA12 - EMA26
	difSeries := make([]float64, len(closes))
	for i := range closes {
		difSeries[i] = ema12[i] - ema26[i]
	}

	// DEA = EMA9 of DIF
	deaSeries := computeEMA(difSeries, 9)

	dif := difSeries[len(difSeries)-1]
	dea := deaSeries[len(deaSeries)-1]
	hist := dif - dea

	// Signal detection
	signal := "neutral"
	if len(difSeries) >= 2 && len(deaSeries) >= 2 {
		prevDif := difSeries[len(difSeries)-2]
		prevDea := deaSeries[len(deaSeries)-2]
		if prevDif <= prevDea && dif > dea {
			signal = "golden_cross"
		} else if prevDif >= prevDea && dif < dea {
			signal = "death_cross"
		}
	}

	return MACDResult{
		DIF:       round4(dif),
		DEA:       round4(dea),
		Histogram: round4(hist),
		Signal:    signal,
	}
}

// ──────────────────────────────────────────────
// KDJ (9/3/3)
// ──────────────────────────────────────────────

// ComputeKDJ calculates KDJ with standard 9/3/3 parameters.
func ComputeKDJ(closes, highs, lows []float64) KDJResult {
	n := len(closes)
	if n < 9 {
		return KDJResult{}
	}

	period := 9
	rsvSeries := make([]float64, n)

	for i := period - 1; i < n; i++ {
		highest := maxFloat(highs[i-period+1 : i+1])
		lowest := minFloat(lows[i-period+1 : i+1])
		if highest == lowest {
			rsvSeries[i] = 50
		} else {
			rsvSeries[i] = (closes[i] - lowest) / (highest - lowest) * 100
		}
	}

	// K = 2/3 * prevK + 1/3 * RSV
	// D = 2/3 * prevD + 1/3 * K
	// J = 3K - 2D
	k, d := 50.0, 50.0
	for i := period - 1; i < n; i++ {
		k = 2.0/3.0*k + 1.0/3.0*rsvSeries[i]
		d = 2.0/3.0*d + 1.0/3.0*k
	}
	j := 3*k - 2*d

	signal := "neutral"
	if k > 80 && d > 80 {
		signal = "overbought"
	} else if k < 20 && d < 20 {
		signal = "oversold"
	}

	return KDJResult{
		K:      round2(k),
		D:      round2(d),
		J:      round2(j),
		Signal: signal,
	}
}

// ──────────────────────────────────────────────
// RSI (6/12/24)
// ──────────────────────────────────────────────

// ComputeRSI calculates RSI for 6, 12, and 24 periods.
func ComputeRSI(closes []float64) RSIResult {
	rsi6 := computeRSIPeriod(closes, 6)
	rsi12 := computeRSIPeriod(closes, 12)
	rsi24 := computeRSIPeriod(closes, 24)

	signal := "neutral"
	// Use RSI14 equivalent (average of 12 and 24) for signal
	rsi14 := (rsi12 + rsi24) / 2
	if rsi14 > 70 {
		signal = "overbought"
	} else if rsi14 < 30 {
		signal = "oversold"
	}

	return RSIResult{
		RSI6:   round2(rsi6),
		RSI12:  round2(rsi12),
		RSI24:  round2(rsi24),
		Signal: signal,
	}
}

func computeRSIPeriod(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 50 // default neutral
	}

	gains := make([]float64, 0, len(closes)-1)
	losses := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		diff := closes[i] - closes[i-1]
		if diff > 0 {
			gains = append(gains, diff)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -diff)
		}
	}

	// Wilder's smoothing (exponential)
	avgGain := 0.0
	avgLoss := 0.0
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	for i := period; i < len(gains); i++ {
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)
	}

	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// ──────────────────────────────────────────────
// Bollinger Bands (20, 2)
// ──────────────────────────────────────────────

// ComputeBoll calculates Bollinger Bands with 20-period SMA and 2 standard deviations.
func ComputeBoll(closes []float64) BollResult {
	if len(closes) < 20 {
		return BollResult{}
	}

	period := 20
	recent := closes[len(closes)-period:]

	// Middle = SMA20
	middle := 0.0
	for _, v := range recent {
		middle += v
	}
	middle /= float64(period)

	// Standard deviation
	variance := 0.0
	for _, v := range recent {
		diff := v - middle
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(period))

	upper := middle + 2*stddev
	lower := middle - 2*stddev

	width := 0.0
	if middle > 0 {
		width = (upper - lower) / middle
	}

	currentPrice := closes[len(closes)-1]
	position := 50.0
	if upper != lower {
		position = (currentPrice - lower) / (upper - lower) * 100
	}

	signal := "within"
	if currentPrice > upper {
		signal = "above_upper"
	} else if currentPrice < lower {
		signal = "below_lower"
	}

	return BollResult{
		Upper:    round4(upper),
		Middle:   round4(middle),
		Lower:    round4(lower),
		Width:    round4(width),
		Position: round2(position),
		Signal:   signal,
	}
}

// ──────────────────────────────────────────────
// MA Alignment (5/10/20/60)
// ──────────────────────────────────────────────

// ComputeMAAlignment calculates moving averages and checks bullish/bearish alignment.
func ComputeMAAlignment(closes []float64) MAAlignmentResult {
	ma5 := computeSMA(closes, 5)
	ma10 := computeSMA(closes, 10)
	ma20 := computeSMA(closes, 20)
	ma60 := computeSMA(closes, 60)

	bullish := ma5 > 0 && ma10 > 0 && ma20 > 0 && ma5 > ma10 && ma10 > ma20
	bearish := ma5 > 0 && ma10 > 0 && ma20 > 0 && ma5 < ma10 && ma10 < ma20

	score := 50 // neutral
	if bullish {
		score = 80
		if ma60 > 0 && ma20 > ma60 {
			score = 90 // strong bullish
		}
	} else if bearish {
		score = 20
		if ma60 > 0 && ma20 < ma60 {
			score = 10 // strong bearish
		}
	} else {
		// Partial alignment
		if ma5 > ma10 && ma10 > 0 {
			score = 60
		} else if ma5 < ma10 && ma10 > 0 {
			score = 40
		}
	}

	return MAAlignmentResult{
		MA5:       round4(ma5),
		MA10:      round4(ma10),
		MA20:      round4(ma20),
		MA60:      round4(ma60),
		Aligned:   bullish,
		BearAlign: bearish,
		Score:     score,
	}
}

// ──────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────

func computeEMA(data []float64, period int) []float64 {
	if len(data) < period {
		return make([]float64, len(data))
	}

	result := make([]float64, len(data))
	multiplier := 2.0 / float64(period+1)

	// Initialize with SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i]
	}
	result[period-1] = sum / float64(period)

	// Calculate EMA
	for i := period; i < len(data); i++ {
		result[i] = (data[i]-result[i-1])*multiplier + result[i-1]
	}

	return result
}

func computeSMA(data []float64, period int) float64 {
	if len(data) < period {
		return 0
	}
	sum := 0.0
	for _, v := range data[len(data)-period:] {
		sum += v
	}
	return sum / float64(period)
}

func extractCloses(klines []models.KlinePoint) []float64 {
	result := make([]float64, len(klines))
	for i, k := range klines {
		result[i] = k.Close
	}
	return result
}

func extractHighs(klines []models.KlinePoint) []float64 {
	result := make([]float64, len(klines))
	for i, k := range klines {
		result[i] = k.High
	}
	return result
}

func extractLows(klines []models.KlinePoint) []float64 {
	result := make([]float64, len(klines))
	for i, k := range klines {
		result[i] = k.Low
	}
	return result
}

func maxFloat(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	m := data[0]
	for _, v := range data[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func minFloat(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	m := data[0]
	for _, v := range data[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// round2 and round4 are defined in eastmoney.go
