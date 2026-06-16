package services

import (
	"fmt"
	"math"
	"strings"

	"alphapulse/internal/models"
)

// ──────────────────────────────────────────────
// T+0 (做T) historical backtest
// ──────────────────────────────────────────────
//
// Walk-forward replay of AnalyzeTSuggestion to measure:
//   - Historical success rate of 正T / 反T signals
//   - Average return per T trade
//   - Cumulative T overlay return vs pure buy-hold
//
// Method: for each day i in the last `daysBack` days, replay the
// signal generator using ONLY klines[0:i] (no look-ahead). "Current
// price" at forecast time = klines[i-1].Close (previous day's close),
// simulating a pre-open analysis. Today's OHLC is used only for
// execution simulation (did the target/stop trigger?).

// TBacktestDay is the per-day result of a single T-signal replay.
type TBacktestDay struct {
	Date        string  `json:"date"`
	SignalType  string  `json:"signal_type"` // "正T" / "反T" / "观望"
	EntryPrice  float64 `json:"entry_price"`
	TargetPrice float64 `json:"target_price"`
	StopLoss    float64 `json:"stop_loss"`
	SignalScore float64 `json:"signal_score"`
	TQuantity   int     `json:"t_quantity"`

	// Actual day OHLC
	ActualHigh  float64 `json:"actual_high"`
	ActualLow   float64 `json:"actual_low"`
	ActualClose float64 `json:"actual_close"`

	// Execution outcome
	Executed   bool    `json:"executed"`
	ExitPrice  float64 `json:"exit_price"`
	ExitReason string  `json:"exit_reason"` // "target" / "stop" / "close" / ""
	ProfitPct  float64 `json:"profit_pct"`
	ProfitAbs  float64 `json:"profit_abs"` // yuan (t_quantity * price_diff)
}

// TBacktestResult is the aggregated output of the T-suggestion backtest.
type TBacktestResult struct {
	Code        string `json:"code"`
	DaysEval    int    `json:"days_evaluated"`
	HoldingQty  int    `json:"holding_qty"`
	HoldingCost float64 `json:"holding_cost"`

	Details []TBacktestDay `json:"details"`

	// Aggregated stats
	TotalTrades int     `json:"total_trades"`
	WinCount    int     `json:"win_count"`
	WinRate     float64 `json:"win_rate"`

	AvgProfitPct  float64 `json:"avg_profit_pct"`
	AvgLossPct    float64 `json:"avg_loss_pct"`
	CumulativePct float64 `json:"cumulative_pct"`   // geometric compounded T return
	MaxDrawdownPct float64 `json:"max_drawdown_pct"` // peak-to-trough of cumulative T profit

	// By-type breakdown
	PositiveTTrades  int     `json:"positive_t_trades"`
	PositiveTWinRate float64 `json:"positive_t_win_rate"`
	ReverseTTrades   int     `json:"reverse_t_trades"`
	ReverseTWinRate  float64 `json:"reverse_t_win_rate"`
	WatchDays        int     `json:"watch_days"`

	// Comparison curves
	EquityCurve    []EquityPoint `json:"equity_curve"`    // buy-hold + T overlay
	BenchmarkCurve []EquityPoint `json:"benchmark_curve"` // pure buy-hold

	TOverlayPct      float64 `json:"t_overlay_pct"`       // incremental return from T
	BuyHoldReturnPct float64 `json:"buy_hold_return_pct"`
	EnhancedReturnPct float64 `json:"enhanced_return_pct"` // buy-hold + T overlay

	Personality *StockPersonality `json:"personality,omitempty"`

	Error string `json:"error,omitempty"`
}

// StockPersonality describes a stock's behavioral characteristics relevant to
// T+0 trading. It answers: is this stock suitable for T? Which T direction?
// What are its volatility and trend patterns?
type StockPersonality struct {
	// ── Volatility ──
	ATRPct        float64 `json:"atr_pct"`        // 14-day ATR / avg price %
	AvgDailyRange float64 `json:"avg_daily_range"` // mean (high-low)/prevClose %
	RangeCV       float64 `json:"range_cv"`        // coefficient of variation of daily range
	AvgGapPct     float64 `json:"avg_gap_pct"`     // mean |open-prevClose|/prevClose %

	// ── T Suitability ──
	TSuitability    float64 `json:"t_suitability"`    // 0-100 composite score
	TSpacePct       float64 `json:"t_space_pct"`      // avg range - round-trip commission %
	CommissionRatio float64 `json:"commission_ratio"`  // commission / avg range (lower is better)
	RecommendedT    string  `json:"recommended_t"`    // "正T" / "反T" / "均可" / "不适合"

	// ── Trend Character ──
	TrendBias        string  `json:"trend_bias"`         // "上涨" / "下跌" / "震荡"
	TrendStrength    float64 `json:"trend_strength"`     // 0-1 (0=oscillating, 1=strong trend)
	MeanReversionPct float64 `json:"mean_reversion_pct"` // % of days where close reverses from open direction
	UpDayPct         float64 `json:"up_day_pct"`         // % of up days

	// ── T History ──
	PositiveTAdvantage float64 `json:"positive_t_advantage"` // positive-T win rate - reverse-T win rate
	BestSignalWindow   string  `json:"best_signal_window"`   // "连续下跌后" / "连续上涨后" / "无明显规律"
	AvgSignalInterval  float64 `json:"avg_signal_interval"`  // avg days between active signals

	// ── Volume ──
	AvgVolume float64 `json:"avg_volume"`
	VolumeCV  float64 `json:"volume_cv"` // coefficient of variation of volume

	// ── Summary ──
	Tags    []string `json:"tags"`
	Summary string   `json:"summary"`
}

// BacktestTSuggestion runs a walk-forward replay of AnalyzeTSuggestion for each
// of the last `daysBack` trading days. No look-ahead: every input to
// AnalyzeTSuggestion is built from klines[:i]; today's OHLC is used only for
// execution simulation.
//
// holdingQty is the assumed base position (e.g. 1000 shares).
// holdingCost is auto-computed from the 20-day average close if <= 0.
func BacktestTSuggestion(klines []models.KlinePoint, daysBack, holdingQty int, holdingCost float64) *TBacktestResult {
	const warmup = 30

	if daysBack <= 0 {
		daysBack = 30
	}
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

	if holdingQty <= 0 {
		holdingQty = 1000
	}
	// Auto-compute holdingCost from 20-day average close of data before the backtest window.
	if holdingCost <= 0 {
		costSliceEnd := len(klines) - daysBack
		if costSliceEnd > 20 {
			holdingCost = movingAverageFromKlines(klines[:costSliceEnd], 20)
		}
		if holdingCost <= 0 {
			holdingCost = klines[0].Close // fallback
		}
	}

	start := len(klines) - daysBack
	details := make([]TBacktestDay, 0, daysBack)

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

		// Simulate at market open: current price = today's open.
		// This gives a non-zero intradayPct so the 24-point
		// intraday momentum component actually contributes to scoring.
		// All indicators still come from hist[:i] — no look-ahead.
		openPrice := today.Open
		if openPrice <= 0 {
			openPrice = prevClose // fallback
		}
		quote := models.Quote{
			Price:     openPrice,
			PrevClose: prevClose,
		}
		tech := models.TechnicalAnalysis{}

		signal := AnalyzeTSuggestion(quote, tech, hist, atr14, holdingQty, holdingCost)
		if signal == nil {
			details = append(details, TBacktestDay{
				Date:        today.Date,
				SignalType:  "观望",
				ActualHigh:  today.High,
				ActualLow:   today.Low,
				ActualClose: today.Close,
			})
			continue
		}

		day := TBacktestDay{
			Date:        today.Date,
			SignalType:  signal.Type,
			EntryPrice:  signal.EntryPrice,
			TargetPrice: signal.TargetPrice,
			StopLoss:    signal.StopLoss,
			SignalScore: signal.SignalScore,
			TQuantity:   signal.TQuantity,
			ActualHigh:  today.High,
			ActualLow:   today.Low,
			ActualClose: today.Close,
		}

		switch signal.Type {
		case "正T":
			simulatePositiveT(&day, signal)
		case "反T":
			simulateReverseT(&day, signal)
		default: // "观望"
			simulatePending(&day, signal)
		}

		details = append(details, day)
	}

	if len(details) == 0 {
		return nil
	}
	result := summarizeTBacktest(details, holdingQty)
	result.Personality = analyzeStockPersonality(klines[start:], details, result)
	return result
}

// ──────────────────────────────────────────────
// Execution simulation helpers
// ──────────────────────────────────────────────

const (
	tCommissionBuy  = 0.0005 // 佣金
	tCommissionSell = 0.0010 // 佣金 + 印花税
)

// simulatePositiveT simulates a 正T (buy first, sell later) trade.
// Entry = buy at entry price. Target = sell at higher price. Stop = sell at lower price.
func simulatePositiveT(day *TBacktestDay, signal *models.TSuggestionAnalysis) {
	entry := day.EntryPrice
	target := day.TargetPrice
	stop := day.StopLoss
	qty := day.TQuantity

	if entry <= 0 || qty <= 0 {
		day.Executed = false
		return
	}

	// Buy happens at entry (prevClose). Now simulate intraday outcome:
	// Priority: check stop first (conservative — if both hit, assume worst case).
	var exitPrice float64
	var exitReason string

	if day.ActualLow <= stop {
		// Stop-loss triggered
		exitPrice = stop
		exitReason = "stop"
	} else if day.ActualHigh >= target {
		// Target hit
		exitPrice = target
		exitReason = "target"
	} else {
		// Neither target nor stop — exit at close (14:50 rule)
		exitPrice = day.ActualClose
		exitReason = "close"
	}

	// P&L: (sell - buy) * qty, minus commissions
	grossProfit := (exitPrice - entry) * float64(qty)
	commission := entry*float64(qty)*tCommissionBuy + exitPrice*float64(qty)*tCommissionSell
	netProfit := grossProfit - commission

	day.Executed = true
	day.ExitPrice = round2(exitPrice)
	day.ExitReason = exitReason
	day.ProfitAbs = round2(netProfit)
	if entry > 0 && qty > 0 {
		day.ProfitPct = round4(netProfit / (entry * float64(qty)))
	}
}

// simulateReverseT simulates a 反T (sell first, buy back later) trade.
// Entry = sell at entry price. Target = buy back at lower price. Stop = buy back at higher price.
func simulateReverseT(day *TBacktestDay, signal *models.TSuggestionAnalysis) {
	entry := day.EntryPrice
	target := day.TargetPrice
	stop := day.StopLoss
	qty := day.TQuantity

	if entry <= 0 || qty <= 0 {
		day.Executed = false
		return
	}

	// Sell happens at entry (prevClose). Now simulate intraday outcome:
	var exitPrice float64
	var exitReason string

	if day.ActualHigh >= stop {
		// Stop-loss triggered (price went up, must buy back at higher price)
		exitPrice = stop
		exitReason = "stop"
	} else if day.ActualLow <= target {
		// Target hit (price dropped, buy back at lower price)
		exitPrice = target
		exitReason = "target"
	} else {
		// Neither — buy back at close
		exitPrice = day.ActualClose
		exitReason = "close"
	}

	// P&L: (sell_price - buy_back_price) * qty, minus commissions
	grossProfit := (entry - exitPrice) * float64(qty)
	commission := entry*float64(qty)*tCommissionSell + exitPrice*float64(qty)*tCommissionBuy
	netProfit := grossProfit - commission

	day.Executed = true
	day.ExitPrice = round2(exitPrice)
	day.ExitReason = exitReason
	day.ProfitAbs = round2(netProfit)
	if entry > 0 && qty > 0 {
		day.ProfitPct = round4(netProfit / (entry * float64(qty)))
	}
}

// simulatePending handles "观望" signals with pending conditional orders.
// Checks whether the pending trigger price was reached during the day.
func simulatePending(day *TBacktestDay, signal *models.TSuggestionAnalysis) {
	// Determine which pending order we have based on Action field
	isPositive := false
	switch {
	case signal.ConditionBuy != nil && signal.ConditionBuy.Direction == "买入" &&
		signal.ConditionSell != nil && signal.ConditionSell.Direction == "卖出":
		// Positive-T pending: buy trigger at lower price, sell at higher price
		isPositive = true
	case signal.ConditionSell != nil && signal.ConditionSell.Direction == "卖出" &&
		signal.ConditionBuy != nil && signal.ConditionBuy.Direction == "买入":
		// Reverse-T pending: sell trigger at higher price, buy back at lower price
		isPositive = false
	default:
		// Can't determine pending type — no simulation
		return
	}

	triggerPrice := signal.EntryPrice
	if triggerPrice <= 0 || day.TQuantity <= 0 {
		return
	}

	if isPositive {
		// Pending buy trigger: price needs to drop to triggerPrice
		if day.ActualLow <= triggerPrice {
			// Trigger hit — simulate the resulting positive T
			// Buy at trigger, use signal's target/stop
			day.EntryPrice = triggerPrice
			simulatePositiveT(day, signal)
		}
	} else {
		// Pending sell trigger: price needs to rise to triggerPrice
		if day.ActualHigh >= triggerPrice {
			// Trigger hit — simulate the resulting reverse T
			day.EntryPrice = triggerPrice
			simulateReverseT(day, signal)
		}
	}
}

// ──────────────────────────────────────────────
// Aggregation
// ──────────────────────────────────────────────

func summarizeTBacktest(details []TBacktestDay, holdingQty int) *TBacktestResult {
	n := len(details)
	result := &TBacktestResult{
		DaysEval:   n,
		HoldingQty: holdingQty,
		Details:    details,
	}

	// Per-trade stats
	var totalTrades, winCount int
	var profitSum, lossSum float64
	var posTrades, posWins, revTrades, revWins, watchDays int
	cumProduct := 1.0 // geometric compounding

	// Equity curve construction
	startPrice := details[0].ActualClose
	if startPrice <= 0 {
		startPrice = 1
	}
	initialEquity := float64(holdingQty) * startPrice
	cumTProfit := 0.0

	equityCurve := make([]EquityPoint, 0, n)
	benchmarkCurve := make([]EquityPoint, 0, n)

	// Track cumulative T profit for max drawdown
	var cumTProfits []float64

	for _, d := range details {
		// Benchmark: pure buy-hold (normalized to 1.0)
		benchmarkEquity := d.ActualClose / startPrice
		benchmarkCurve = append(benchmarkCurve, EquityPoint{
			Date:       d.Date,
			Equity:     round4(benchmarkEquity),
			InPosition: true,
		})

		if d.Executed {
			totalTrades++
			cumTProfit += d.ProfitAbs
			if d.ProfitAbs > 0 {
				winCount++
				profitSum += d.ProfitPct
			} else {
				lossSum += d.ProfitPct
			}
			// Geometric compounding of per-trade return
			cumProduct *= (1 + d.ProfitPct)

			switch d.SignalType {
			case "正T":
				posTrades++
				if d.ProfitAbs > 0 {
					posWins++
				}
			case "反T":
				revTrades++
				if d.ProfitAbs > 0 {
					revWins++
				}
			}
		} else if d.SignalType == "观望" {
			watchDays++
		}

		// Enhanced equity: benchmark + cumulative T profit (normalized)
		enhancedEquity := (float64(holdingQty)*d.ActualClose + cumTProfit) / initialEquity
		equityCurve = append(equityCurve, EquityPoint{
			Date:       d.Date,
			Equity:     round4(enhancedEquity),
			InPosition: d.Executed,
		})

		cumTProfits = append(cumTProfits, cumTProfit)
	}

	result.TotalTrades = totalTrades
	result.WinCount = winCount
	result.WatchDays = watchDays

	if totalTrades > 0 {
		result.WinRate = round4(float64(winCount) / float64(totalTrades))
	}
	if winCount > 0 {
		result.AvgProfitPct = round4(profitSum / float64(winCount))
	}
	losses := totalTrades - winCount
	if losses > 0 {
		result.AvgLossPct = round4(lossSum / float64(losses))
	}
	result.CumulativePct = round4(cumProduct - 1)

	// Max drawdown of cumulative T profit
	result.MaxDrawdownPct = round4(tProfitMaxDrawdown(cumTProfits, initialEquity))

	// By-type win rates
	result.PositiveTTrades = posTrades
	if posTrades > 0 {
		result.PositiveTWinRate = round4(float64(posWins) / float64(posTrades))
	}
	result.ReverseTTrades = revTrades
	if revTrades > 0 {
		result.ReverseTWinRate = round4(float64(revWins) / float64(revTrades))
	}

	// Comparison
	endPrice := details[n-1].ActualClose
	if startPrice > 0 && endPrice > 0 {
		result.BuyHoldReturnPct = round4(endPrice/startPrice - 1)
	}

	endEnhanced := equityCurve[n-1].Equity
	endBenchmark := benchmarkCurve[n-1].Equity
	result.EnhancedReturnPct = round4(endEnhanced - 1)
	if endBenchmark > 0 {
		result.TOverlayPct = round4(endEnhanced - endBenchmark)
	}

	result.EquityCurve = equityCurve
	result.BenchmarkCurve = benchmarkCurve

	// HoldingCost: use the first day's close as representative
	result.HoldingCost = round2(startPrice)

	// Sanitize: ensure no NaN/Inf values that would break JSON serialization.
	sanitizeFloat(&result.WinRate)
	sanitizeFloat(&result.AvgProfitPct)
	sanitizeFloat(&result.AvgLossPct)
	sanitizeFloat(&result.CumulativePct)
	sanitizeFloat(&result.MaxDrawdownPct)
	sanitizeFloat(&result.PositiveTWinRate)
	sanitizeFloat(&result.ReverseTWinRate)
	sanitizeFloat(&result.BuyHoldReturnPct)
	sanitizeFloat(&result.EnhancedReturnPct)
	sanitizeFloat(&result.TOverlayPct)
	for i := range result.Details {
		sanitizeFloat(&result.Details[i].ProfitPct)
		sanitizeFloat(&result.Details[i].ProfitAbs)
	}

	return result
}

// sanitizeFloat replaces NaN/Inf with 0 so json.Marshal won't fail.
func sanitizeFloat(v *float64) {
	if math.IsNaN(*v) || math.IsInf(*v, 0) {
		*v = 0
	}
}

// mean returns the arithmetic mean of values.
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// tProfitMaxDrawdown computes the peak-to-trough drawdown of the cumulative
// T profit series, normalized by the initial equity.
func tProfitMaxDrawdown(cumProfits []float64, initialEquity float64) float64 {
	if len(cumProfits) == 0 || initialEquity <= 0 {
		return 0
	}
	var peak float64
	var maxDD float64
	for _, p := range cumProfits {
		if p > peak {
			peak = p
		}
		dd := (peak - p) / initialEquity
		if dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD
}

// ──────────────────────────────────────────────
// Stock personality analysis
// ──────────────────────────────────────────────

// analyzeStockPersonality computes the stock's behavioral profile for T+0 trading.
// klines is the raw kline data for the backtest window; details are the per-day
// backtest results; result provides aggregated T stats.
func analyzeStockPersonality(klines []models.KlinePoint, details []TBacktestDay, result *TBacktestResult) *StockPersonality {
	if len(klines) == 0 || len(details) == 0 {
		return nil
	}

	p := &StockPersonality{}

	// ── Volatility ──
	// Compute ATR% from the klines in the window
	atr14 := ComputeATR(klines, 14)
	avgClose := 0.0
	for _, k := range klines {
		avgClose += k.Close
	}
	avgClose /= float64(len(klines))
	if avgClose > 0 {
		p.ATRPct = round4(atr14 / avgClose * 100)
	}

	// Daily range stats: (high - low) / prevClose * 100
	dailyRanges := make([]float64, 0, len(klines)-1)
	gapPcts := make([]float64, 0, len(klines)-1)
	for i := 1; i < len(klines); i++ {
		prevClose := klines[i-1].Close
		if prevClose <= 0 {
			continue
		}
		rng := (klines[i].High - klines[i].Low) / prevClose * 100
		dailyRanges = append(dailyRanges, rng)
		gap := math.Abs(klines[i].Open-prevClose) / prevClose * 100
		gapPcts = append(gapPcts, gap)
	}
	if len(dailyRanges) > 0 {
		p.AvgDailyRange = round4(mean(dailyRanges))
		p.RangeCV = round4(stddev(dailyRanges) / mean(dailyRanges))
	}
	if len(gapPcts) > 0 {
		p.AvgGapPct = round4(mean(gapPcts))
	}

	// ── T Suitability ──
	const roundTripCommission = 0.15 // buy 0.05% + sell 0.10% (%)
	p.TSpacePct = round4(p.AvgDailyRange - roundTripCommission)
	if p.AvgDailyRange > 0 {
		p.CommissionRatio = round4(roundTripCommission / p.AvgDailyRange)
	}

	// Composite suitability score (0-100)
	score := 0.0
	if p.TSpacePct > 0.5 {
		score += 15
	}
	if p.TSpacePct > 1.0 {
		score += 15
	}
	if p.TSpacePct > 2.0 {
		score += 10
	}
	if result.WinRate > 0.4 {
		score += 10
	}
	if result.WinRate > 0.5 {
		score += 15
	}
	if result.CumulativePct > 0 {
		score += 15
	}
	if p.RangeCV > 0 && p.RangeCV < 0.5 {
		score += 10
	}
	if p.RangeCV > 0 && p.RangeCV < 0.8 {
		score += 5
	}
	if result.TotalTrades >= 5 {
		score += 5 // enough sample size
	}
	p.TSuitability = round4(clampFloat(score, 0, 100))

	// Recommended T direction
	p.PositiveTAdvantage = round4(result.PositiveTWinRate - result.ReverseTWinRate)
	switch {
	case p.TSuitability < 30:
		p.RecommendedT = "不适合"
	case p.PositiveTAdvantage > 0.1:
		p.RecommendedT = "正T"
	case p.PositiveTAdvantage < -0.1:
		p.RecommendedT = "反T"
	default:
		p.RecommendedT = "均可"
	}

	// ── Trend Character ──
	upDays := 0
	meanReversions := 0
	directionDays := 0
	for i := 1; i < len(klines); i++ {
		if klines[i-1].Close <= 0 {
			continue
		}
		if klines[i].Close > klines[i-1].Close {
			upDays++
		}
		// Mean reversion: close reverses from open direction
		openDir := klines[i].Open - klines[i-1].Close
		closeDir := klines[i].Close - klines[i].Open
		if (openDir > 0 && closeDir < 0) || (openDir < 0 && closeDir > 0) {
			meanReversions++
		}
		if openDir != 0 {
			directionDays++
		}
	}
	n := len(klines) - 1
	if n > 0 {
		p.UpDayPct = round4(float64(upDays) / float64(n))
	}
	if directionDays > 0 {
		p.MeanReversionPct = round4(float64(meanReversions) / float64(directionDays))
	}
	switch {
	case p.UpDayPct > 0.6:
		p.TrendBias = "上涨"
	case p.UpDayPct < 0.4:
		p.TrendBias = "下跌"
	default:
		p.TrendBias = "震荡"
	}
	p.TrendStrength = round4(math.Abs(p.UpDayPct-0.5) * 2)

	// ── T History ──
	activeSignals := result.PositiveTTrades + result.ReverseTTrades
	if activeSignals > 0 {
		p.AvgSignalInterval = round4(float64(result.DaysEval) / float64(activeSignals))
	}
	p.BestSignalWindow = detectBestSignalWindow(details)

	// ── Volume ──
	volumes := make([]float64, 0, len(klines))
	for _, k := range klines {
		if k.Volume > 0 {
			volumes = append(volumes, k.Volume)
		}
	}
	if len(volumes) > 0 {
		p.AvgVolume = round2(mean(volumes))
		p.VolumeCV = round4(stddev(volumes) / mean(volumes))
	}

	// ── Tags ──
	p.Tags = buildPersonalityTags(p)
	p.Summary = buildPersonalitySummary(p, result)

	// Sanitize
	sanitizeFloat(&p.ATRPct)
	sanitizeFloat(&p.AvgDailyRange)
	sanitizeFloat(&p.RangeCV)
	sanitizeFloat(&p.AvgGapPct)
	sanitizeFloat(&p.TSpacePct)
	sanitizeFloat(&p.CommissionRatio)
	sanitizeFloat(&p.TSuitability)
	sanitizeFloat(&p.TrendStrength)
	sanitizeFloat(&p.MeanReversionPct)
	sanitizeFloat(&p.UpDayPct)
	sanitizeFloat(&p.PositiveTAdvantage)
	sanitizeFloat(&p.AvgSignalInterval)
	sanitizeFloat(&p.VolumeCV)

	return p
}

// detectBestSignalWindow checks whether T trades after consecutive up/down days
// have higher win rates.
func detectBestSignalWindow(details []TBacktestDay) string {
	// Track: after N consecutive down days, what's the T win rate?
	var afterDownWins, afterDownTotal int
	var afterUpWins, afterUpTotal int

	consecDir := 0 // negative = down days, positive = up days
	for i, d := range details {
		if i > 0 {
			if details[i-1].ActualClose > 0 {
				if d.ActualClose > details[i-1].ActualClose {
					if consecDir < 0 {
						consecDir = 0
					}
					consecDir++
				} else {
					if consecDir > 0 {
						consecDir = 0
					}
					consecDir--
				}
			}
		}

		if !d.Executed {
			continue
		}
		// After 2+ consecutive down days → positive T opportunity
		if consecDir <= -2 {
			afterDownTotal++
			if d.ProfitAbs > 0 {
				afterDownWins++
			}
		}
		// After 2+ consecutive up days → reverse T opportunity
		if consecDir >= 2 {
			afterUpTotal++
			if d.ProfitAbs > 0 {
				afterUpWins++
			}
		}
	}

	afterDownRate := 0.0
	if afterDownTotal > 0 {
		afterDownRate = float64(afterDownWins) / float64(afterDownTotal)
	}
	afterUpRate := 0.0
	if afterUpTotal > 0 {
		afterUpRate = float64(afterUpWins) / float64(afterUpTotal)
	}

	// Need enough samples to be meaningful
	minSamples := 3
	if afterDownTotal >= minSamples && afterDownRate > 0.6 && afterDownRate > afterUpRate+0.1 {
		return "连续下跌后"
	}
	if afterUpTotal >= minSamples && afterUpRate > 0.6 && afterUpRate > afterDownRate+0.1 {
		return "连续上涨后"
	}
	return "无明显规律"
}

// buildPersonalityTags generates descriptive tags based on personality metrics.
func buildPersonalityTags(p *StockPersonality) []string {
	var tags []string

	// Volatility
	if p.ATRPct > 2.5 {
		tags = append(tags, "高波动")
	} else if p.ATRPct < 1.0 {
		tags = append(tags, "低波动")
	}

	// T suitability
	if p.TSuitability >= 60 {
		tags = append(tags, "适合做T")
	} else if p.TSuitability < 35 {
		tags = append(tags, "不适合做T")
	}

	// T direction advantage
	if p.PositiveTAdvantage > 0.1 {
		tags = append(tags, "正T优势")
	} else if p.PositiveTAdvantage < -0.1 {
		tags = append(tags, "反T优势")
	}

	// Trend type
	if p.TrendStrength > 0.4 {
		tags = append(tags, "趋势型")
	} else {
		tags = append(tags, "震荡型")
	}

	// Range stability
	if p.RangeCV > 0 && p.RangeCV < 0.5 {
		tags = append(tags, "振幅稳定")
	} else if p.RangeCV > 0.8 {
		tags = append(tags, "振幅不稳")
	}

	// Mean reversion
	if p.MeanReversionPct > 0.55 {
		tags = append(tags, "均值回归强")
	}

	// Volume
	if p.VolumeCV > 0 && p.VolumeCV < 0.4 {
		tags = append(tags, "成交稳定")
	} else if p.VolumeCV > 1.0 {
		tags = append(tags, "成交不稳")
	}

	return tags
}

// buildPersonalitySummary creates a one-line Chinese summary of the stock personality.
func buildPersonalitySummary(p *StockPersonality, result *TBacktestResult) string {
	parts := []string{}

	// Trend + volatility type
	trendType := "震荡型"
	if p.TrendStrength > 0.4 {
		trendType = "趋势型"
	}
	volType := ""
	if p.ATRPct > 2.5 {
		volType = "高波动"
	} else if p.ATRPct < 1.0 {
		volType = "低波动"
	}
	if volType != "" {
		parts = append(parts, volType+trendType)
	} else {
		parts = append(parts, trendType)
	}

	// T space
	if p.TSpacePct > 0 {
		parts = append(parts, fmt.Sprintf("T空间%.1f%%", p.TSpacePct))
	}

	// T direction + win rate
	if p.PositiveTAdvantage > 0.1 && result.PositiveTTrades > 0 {
		parts = append(parts, fmt.Sprintf("正T胜率更高(%.0f%%)", result.PositiveTWinRate*100))
	} else if p.PositiveTAdvantage < -0.1 && result.ReverseTTrades > 0 {
		parts = append(parts, fmt.Sprintf("反T胜率更高(%.0f%%)", result.ReverseTWinRate*100))
	}

	// Suitability
	if p.TSuitability >= 60 {
		parts = append(parts, "适合做T")
	} else if p.TSuitability < 35 {
		parts = append(parts, "不建议做T")
	}

	return strings.Join(parts, "，")
}

