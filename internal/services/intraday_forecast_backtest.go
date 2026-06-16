package services

import (
	"math"
	"sort"

	"alphapulse/internal/models"
)

// ──────────────────────────────────────────────
// Intraday-forecast historical accuracy backtest
// ──────────────────────────────────────────────
//
// Goal: measure how often AnalyzeIntradayForecast's [PredictedLow, PredictedHigh]
// actually covers the day's [Low, High]. Without this backtest the prediction
// is a single point estimate with no accuracy backing — using it to make sell
// decisions is gambling.
//
// Method:
//   For each day i in the last `daysBack` trading days, replay the forecast
//   using ONLY data available before market open on day i (klines[0:i]).
//   "Current price" at forecast time is set to klines[i-1].Close (the previous
//   day's close), simulating a pre-open prediction. The actual klines[i].High
//   and klines[i].Low are then compared against the prediction.
//
// Avoiding look-ahead: every input is built from klines[0:i]; today's OHLC
// never enters ATR, indicators, patterns, S/R, or quote.

// IntradayForecastDay is the per-day result of a single backtest prediction.
type IntradayForecastDay struct {
	Date             string  `json:"date"`
	PrevClose        float64 `json:"prev_close"`
	PredictedHigh    float64 `json:"predicted_high"`
	PredictedLow     float64 `json:"predicted_low"`
	PredictedHighUp  float64 `json:"predicted_high_up"`  // +1σ
	PredictedLowDown float64 `json:"predicted_low_down"` // -1σ
	// P50 of historical excursion distribution — the median high/low.
	// Used for "high-fill-rate" condition orders.
	PredictedHighMedian float64 `json:"predicted_high_median,omitempty"`
	PredictedLowMedian  float64 `json:"predicted_low_median,omitempty"`
	PredictedMid        float64 `json:"predicted_mid"`
	ActualHigh          float64 `json:"actual_high"`
	ActualLow           float64 `json:"actual_low"`
	ActualClose         float64 `json:"actual_close"`
	HighInRange         bool    `json:"high_in_range"` // actual_high ≤ predicted_high
	LowInRange          bool    `json:"low_in_range"`  // actual_low ≥ predicted_low
	BothInRange         bool    `json:"both_in_range"`
	HighInWideRange     bool    `json:"high_in_wide_range"` // actual_high ≤ predicted_high_up
	LowInWideRange      bool    `json:"low_in_wide_range"`  // actual_low ≥ predicted_low_down
	BothInWideRange     bool    `json:"both_in_wide_range"`
	Bias                string  `json:"bias"`
	BiasStrength        float64 `json:"bias_strength"`
	Zone                string  `json:"zone"`
}

// OrderLevelStat is the empirical backtest result for a single condition-order
// price level (e.g. "sell at predicted_high"). Frontend renders this side-by-side
// with the theoretical fill rate to show calibration.
type OrderLevelStat struct {
	Side               string  `json:"side"`                 // "sell" / "buy"
	Tag                string  `json:"tag"`                  // "上限" / "预测高" / "中位" / "预测低" / "下限"
	TheoreticalFillPct float64 `json:"theoretical_fill_pct"` // expected fill rate (5%/17%/50% from percentile construction)
	EmpiricalFillPct   float64 `json:"empirical_fill_pct"`   // actual fill rate over the backtest window
	Fills              int     `json:"fills"`                // number of days the order filled
	SampleSize         int     `json:"sample_size"`          // days considered
	AvgFillPrice       float64 `json:"avg_fill_price"`       // avg price when filled ≈ limit price
	AvgPnlPct          float64 `json:"avg_pnl_pct"`          // avg execution improvement % vs close when filled
	WinRate            float64 `json:"win_rate"`             // % of filled days with positive P&L
	CumulativePnlPct   float64 `json:"cumulative_pnl_pct"`   // geometric compounded improvement over filled days
}

// IntradayForecastAccuracy aggregates hit-rate metrics over a backtest window.
type IntradayForecastAccuracy struct {
	DaysEvaluated int     `json:"days_evaluated"`
	BothInRange   float64 `json:"both_in_range_pct"` // % of days where actual range ⊆ predicted range
	HighInRange   float64 `json:"high_in_range_pct"` // % of days where actual_high ≤ predicted_high
	LowInRange    float64 `json:"low_in_range_pct"`  // % of days where actual_low ≥ predicted_low

	// Wide-band hit rates (predicted_high_up / predicted_low_down, ~84/16 percentile).
	// A central-band miss with a wide-band hit is the typical case — it means
	// volatility was slightly higher than expected, not that the model is broken.
	BothInWideRange float64 `json:"both_in_wide_range_pct"`
	HighInWideRange float64 `json:"high_in_wide_range_pct"`
	LowInWideRange  float64 `json:"low_in_wide_range_pct"`

	// When the prediction missed (actual broke out of the range), how far did
	// it break out on average? Positive = broke beyond the bound (under-predicted
	// volatility). Negative = stopped short of the bound (over-predicted).
	AvgHighMissPct float64 `json:"avg_high_miss_pct"` // E[(actual_high - predicted_high) / predicted_high] over days that missed high
	AvgLowMissPct  float64 `json:"avg_low_miss_pct"`  // E[(predicted_low - actual_low) / predicted_low] over days that missed low

	// Average predicted range width (absolute and as % of mid).
	AvgRangeWidth    float64 `json:"avg_range_width"`
	AvgRangeWidthPct float64 `json:"avg_range_width_pct"`
	AvgWideWidth     float64 `json:"avg_wide_width"`
	AvgWideWidthPct  float64 `json:"avg_wide_width_pct"`

	// Average σ (uncertainty) — tells you how tightly the forecast clusters.
	AvgSigma float64 `json:"avg_sigma"`

	// Recommendation based on BothInRange hit rate.
	//   ≥ 70% : "high_confidence" — usable for sell timing
	//   ≥ 55% : "moderate"        — use as one input, not sole trigger
	//   <  55% : "low_confidence"  — do NOT use for sell decisions
	Reliability string `json:"reliability"`

	// Per-level order execution stats — empirical fill rates and P&L for
	// each of the 6 condition-order price tiers (3 sell + 3 buy).
	OrderStats []OrderLevelStat `json:"order_stats,omitempty"`

	Details []IntradayForecastDay `json:"details,omitempty"`
}

// BacktestIntradayForecast runs AnalyzeIntradayForecast for each of the last
// `daysBack` days in `klines` and returns accuracy metrics.
//
// Requires at least warmup+1 klines to evaluate a single day. If fewer than
// `daysBack` evaluable days are available, the window is clamped down to
// whatever the data supports (always ≥ 1 evaluable day when len(klines) >
// warmup). Returns nil only when even a single-day evaluation is impossible.
func BacktestIntradayForecast(klines []models.KlinePoint, daysBack int) *IntradayForecastAccuracy {
	const warmup = 30

	if daysBack <= 0 {
		daysBack = 30
	}
	// Clamp the window to whatever the data actually supports rather than
	// refusing outright — a 90-day report is strictly more useful than no
	// report at all.
	if len(klines) <= warmup {
		return nil
	}
	available := len(klines) - warmup
	if available < daysBack {
		daysBack = available
	}
	if daysBack < 1 {
		return nil
	}

	// Evaluate the most recent `daysBack` days.
	start := len(klines) - daysBack
	details := make([]IntradayForecastDay, 0, daysBack)

	for i := start; i < len(klines); i++ {
		hist := klines[:i] // strictly before day i — no look-ahead
		if len(hist) < warmup {
			continue
		}
		today := klines[i]

		prevClose := hist[len(hist)-1].Close
		if prevClose <= 0 {
			continue
		}

		atr14 := ComputeATR(hist, 14)
		if atr14 <= 0 {
			continue
		}
		patterns := DetectPatterns(hist)
		support, resistance := computeSupportResistanceFromHist(hist)

		// Pre-open simulation: current price = previous close.
		quote := models.Quote{
			Price:     prevClose,
			PrevClose: prevClose,
		}
		fc := AnalyzeIntradayForecast(hist, quote, atr14, patterns, support, resistance)
		if fc == nil {
			continue
		}

		highHit := today.High <= fc.PredictedHigh
		lowHit := today.Low >= fc.PredictedLow
		highWide := today.High <= fc.PredictedHighUp
		lowWide := today.Low >= fc.PredictedLowDown
		details = append(details, IntradayForecastDay{
			Date:                today.Date,
			PrevClose:           round2(prevClose),
			PredictedHigh:       fc.PredictedHigh,
			PredictedLow:        fc.PredictedLow,
			PredictedHighUp:     fc.PredictedHighUp,
			PredictedLowDown:    fc.PredictedLowDown,
			PredictedHighMedian: fc.PredictedHighMedian,
			PredictedLowMedian:  fc.PredictedLowMedian,
			PredictedMid:        round2((fc.PredictedHigh + fc.PredictedLow) / 2),
			ActualHigh:          today.High,
			ActualLow:           today.Low,
			ActualClose:         today.Close,
			HighInRange:         highHit,
			LowInRange:          lowHit,
			BothInRange:         highHit && lowHit,
			HighInWideRange:     highWide,
			LowInWideRange:      lowWide,
			BothInWideRange:     highWide && lowWide,
			Bias:                fc.Bias,
			BiasStrength:        fc.BiasStrength,
			Zone:                fc.CurrentZone,
		})
	}

	if len(details) == 0 {
		return nil
	}
	return summarizeAccuracy(details)
}

func summarizeAccuracy(details []IntradayForecastDay) *IntradayForecastAccuracy {
	n := len(details)
	acc := &IntradayForecastAccuracy{
		DaysEvaluated: n,
		Details:       details,
	}

	var bothHit, highHit, lowHit int
	var bothWideHit, highWideHit, lowWideHit int
	var highMissSum, lowMissSum float64
	highMissCount, lowMissCount := 0, 0
	var widthSum, widthPctSum, wideSum, widePctSum, sigmaSum float64

	for _, d := range details {
		if d.BothInRange {
			bothHit++
		}
		if d.BothInWideRange {
			bothWideHit++
		}
		if d.HighInRange {
			highHit++
		} else {
			if d.PredictedHigh > 0 {
				highMissSum += (d.ActualHigh - d.PredictedHigh) / d.PredictedHigh
				highMissCount++
			}
		}
		if d.HighInWideRange {
			highWideHit++
		}
		if d.LowInRange {
			lowHit++
		} else {
			if d.PredictedLow > 0 {
				lowMissSum += (d.PredictedLow - d.ActualLow) / d.PredictedLow
				lowMissCount++
			}
		}
		if d.LowInWideRange {
			lowWideHit++
		}
		width := d.PredictedHigh - d.PredictedLow
		widthSum += width
		wide := d.PredictedHighUp - d.PredictedLowDown
		wideSum += wide
		if d.PredictedMid > 0 {
			widthPctSum += width / d.PredictedMid
			widePctSum += wide / d.PredictedMid
		}
		// σ ≈ (wide - width) / 2 on each side.
		sigmaSum += (wide - width) / 2
	}

	acc.BothInRange = pct(bothHit, n)
	acc.HighInRange = pct(highHit, n)
	acc.LowInRange = pct(lowHit, n)
	acc.BothInWideRange = pct(bothWideHit, n)
	acc.HighInWideRange = pct(highWideHit, n)
	acc.LowInWideRange = pct(lowWideHit, n)
	if highMissCount > 0 {
		acc.AvgHighMissPct = round4(highMissSum / float64(highMissCount))
	}
	if lowMissCount > 0 {
		acc.AvgLowMissPct = round4(lowMissSum / float64(lowMissCount))
	}
	acc.AvgRangeWidth = round2(widthSum / float64(n))
	acc.AvgRangeWidthPct = round4(widthPctSum / float64(n))
	acc.AvgWideWidth = round2(wideSum / float64(n))
	acc.AvgWideWidthPct = round4(widePctSum / float64(n))
	acc.AvgSigma = round2(sigmaSum / float64(n))

	switch {
	case acc.BothInRange >= 0.70:
		acc.Reliability = "high_confidence"
	case acc.BothInRange >= 0.55:
		acc.Reliability = "moderate"
	default:
		acc.Reliability = "low_confidence"
	}

	// ──────────────────────────────────────────────
	// Per-level order execution stats
	// ──────────────────────────────────────────────
	// For each of the 6 condition-order price tiers (3 sell + 3 buy),
	// compute empirical fill rate and avg P&L over the backtest window.
	//
	// Fill semantics:
	//   Sell at level L → filled if actual_high ≥ L (price reached the level)
	//   Buy  at level L → filled if actual_low  ≤ L (price dropped to level)
	//
	// P&L semantics (execution improvement vs closing price, no position tracking):
	//   Sell: P&L = (fill_price - actual_close) / actual_close
	//         Positive = sold above the day's close (better than waiting).
	//   Buy:  P&L = (actual_close - fill_price) / fill_price
	//         Positive = bought below the day's close (better than waiting).

	levels := []struct {
		side               string
		tag                string
		theoreticalFillPct float64
		levelFn            func(IntradayForecastDay) float64
		filledFn           func(IntradayForecastDay, float64) bool
	}{
		// Sell levels (descending by aggressiveness)
		{
			side: "sell", tag: "上限", theoreticalFillPct: 0.05,
			levelFn:  func(d IntradayForecastDay) float64 { return d.PredictedHighUp },
			filledFn: func(d IntradayForecastDay, lv float64) bool { return d.ActualHigh >= lv },
		},
		{
			side: "sell", tag: "预测高", theoreticalFillPct: 0.17,
			levelFn:  func(d IntradayForecastDay) float64 { return d.PredictedHigh },
			filledFn: func(d IntradayForecastDay, lv float64) bool { return d.ActualHigh >= lv },
		},
		{
			side: "sell", tag: "中位", theoreticalFillPct: 0.50,
			levelFn:  func(d IntradayForecastDay) float64 { return d.PredictedHighMedian },
			filledFn: func(d IntradayForecastDay, lv float64) bool { return d.ActualHigh >= lv },
		},
		// Buy levels
		{
			side: "buy", tag: "下限", theoreticalFillPct: 0.05,
			levelFn:  func(d IntradayForecastDay) float64 { return d.PredictedLowDown },
			filledFn: func(d IntradayForecastDay, lv float64) bool { return d.ActualLow <= lv },
		},
		{
			side: "buy", tag: "预测低", theoreticalFillPct: 0.17,
			levelFn:  func(d IntradayForecastDay) float64 { return d.PredictedLow },
			filledFn: func(d IntradayForecastDay, lv float64) bool { return d.ActualLow <= lv },
		},
		{
			side: "buy", tag: "中位", theoreticalFillPct: 0.50,
			levelFn:  func(d IntradayForecastDay) float64 { return d.PredictedLowMedian },
			filledFn: func(d IntradayForecastDay, lv float64) bool { return d.ActualLow <= lv },
		},
	}

	var orderStats []OrderLevelStat
	for _, lv := range levels {
		stat := OrderLevelStat{
			Side:               lv.side,
			Tag:                lv.tag,
			TheoreticalFillPct: lv.theoreticalFillPct,
		}
		var fills int
		var priceSum, pnlSum float64
		var wins int
		cumProduct := 1.0 // geometric compounding

		for _, d := range details {
			level := lv.levelFn(d)
			if level <= 0 || d.PrevClose <= 0 {
				continue
			}
			stat.SampleSize++

			if !lv.filledFn(d, level) {
				continue
			}
			var pnl float64
			if lv.side == "sell" {
				if d.ActualClose > 0 {
					pnl = (level - d.ActualClose) / d.ActualClose
				}
			} else {
				if d.ActualClose > 0 {
					pnl = (d.ActualClose - level) / level
				}
			}
			fills++
			priceSum += level
			pnlSum += pnl
			if pnl > 0 {
				wins++
			}
			cumProduct *= (1 + pnl)
		}

		stat.Fills = fills
		if stat.SampleSize > 0 {
			stat.EmpiricalFillPct = round4(float64(fills) / float64(stat.SampleSize))
		}
		if fills > 0 {
			stat.AvgFillPrice = round2(priceSum / float64(fills))
			stat.AvgPnlPct = round4(pnlSum / float64(fills))
			stat.WinRate = round4(float64(wins) / float64(fills))
			stat.CumulativePnlPct = round4(cumProduct - 1)
		}
		orderStats = append(orderStats, stat)
	}
	acc.OrderStats = orderStats

	return acc
}

// computeSupportResistanceFromHist replicates the logic of
// (*TrendAnalyzer).calculateSupportResistance using only kline history.
// Returns (nearest_support, nearest_resistance).
func computeSupportResistanceFromHist(hist []models.KlinePoint) (support, resistance float64) {
	if len(hist) == 0 {
		return 0, 0
	}
	ind := ComputeIndicators(hist)
	price := hist[len(hist)-1].Close
	if price <= 0 {
		return 0, 0
	}

	levels := []float64{
		ind.MA.MA5, ind.MA.MA10, ind.MA.MA20, ind.MA.MA60,
		ind.Boll.Upper, ind.Boll.Middle, ind.Boll.Lower,
	}

	var supports, resistances []float64
	for _, l := range levels {
		if l <= 0 {
			continue
		}
		if l < price {
			supports = append(supports, l)
		} else if l > price {
			resistances = append(resistances, l)
		}
	}

	if len(supports) > 0 {
		sort.Float64s(supports)
		support = supports[len(supports)-1] // nearest to price (largest below)
	}
	if len(resistances) > 0 {
		sort.Float64s(resistances)
		resistance = resistances[0] // nearest to price (smallest above)
	}
	support = math.Round(support*100) / 100
	resistance = math.Round(resistance*100) / 100
	return
}

func pct(hit, total int) float64 {
	if total == 0 {
		return 0
	}
	return round4(float64(hit) / float64(total))
}
