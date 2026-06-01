package services

import (
	"fmt"
	"math"
	"strings"

	"alphapulse/internal/models"
)

// ──────────────────────────────────────────────
// ATR (Average True Range)
// ──────────────────────────────────────────────

// ComputeATR calculates the Average True Range using Wilder's smoothing.
func ComputeATR(klines []models.KlinePoint, period int) float64 {
	if len(klines) < period+1 {
		return 0
	}

	// Compute True Range series
	trSeries := make([]float64, len(klines)-1)
	for i := 1; i < len(klines); i++ {
		hl := klines[i].High - klines[i].Low
		hc := math.Abs(klines[i].High - klines[i-1].Close)
		lc := math.Abs(klines[i].Low - klines[i-1].Close)
		trSeries[i-1] = math.Max(hl, math.Max(hc, lc))
	}

	// Wilder's smoothing: first ATR = SMA of first `period` TR values
	if len(trSeries) < period {
		return 0
	}

	atr := 0.0
	for i := 0; i < period; i++ {
		atr += trSeries[i]
	}
	atr /= float64(period)

	// Smooth remaining values
	for i := period; i < len(trSeries); i++ {
		atr = (atr*float64(period-1) + trSeries[i]) / float64(period)
	}

	return round4(atr)
}

// ──────────────────────────────────────────────
// Buy Zone Analysis
// ──────────────────────────────────────────────

// AnalyzeBuyZone computes multiple buy zone suggestions from support levels,
// Bollinger bands, and moving averages.
func AnalyzeBuyZone(
	sr models.SupportResistance,
	tech models.TechnicalAnalysis,
	quote models.Quote,
	atr14 float64,
) *models.BuyZoneAnalysis {
	if atr14 <= 0 || quote.Price <= 0 {
		return nil
	}

	price := quote.Price
	var zones []models.BuyZone

	// Zone 1: Support-based
	if sr.Support1 > 0 {
		upper := sr.Support1 + 0.5*atr14
		lower := sr.Support1 - 0.3*atr14
		optimal := sr.Support1 + 0.1*atr14
		stop := lower - 0.5*atr14
		safety := safetyScore(price, lower, upper)

		zones = append(zones, models.BuyZone{
			Method:       "支撑位",
			UpperBound:   round2(upper),
			LowerBound:   round2(lower),
			OptimalEntry: round2(optimal),
			StopLoss:     round2(stop),
			SafetyScore:  round2(safety),
		})
	}

	// Zone 2: Bollinger Band-based
	if tech.BollLower > 0 && tech.BollMid > tech.BollLower {
		upper := tech.BollMid
		lower := tech.BollLower
		optimal := tech.BollLower + 0.3*(tech.BollMid-tech.BollLower)
		stop := tech.BollLower - 0.5*atr14
		safety := safetyScore(price, lower, upper)

		zones = append(zones, models.BuyZone{
			Method:       "布林带",
			UpperBound:   round2(upper),
			LowerBound:   round2(lower),
			OptimalEntry: round2(optimal),
			StopLoss:     round2(stop),
			SafetyScore:  round2(safety),
		})
	}

	// Zone 3: MA20-based
	if tech.MA20 > 0 {
		upper := tech.MA20 + 0.5*atr14
		lower := tech.MA20 - 0.5*atr14
		optimal := tech.MA20 - 0.2*atr14
		stop := tech.MA20 - atr14
		safety := safetyScore(price, lower, upper)

		zones = append(zones, models.BuyZone{
			Method:       "均线MA20",
			UpperBound:   round2(upper),
			LowerBound:   round2(lower),
			OptimalEntry: round2(optimal),
			StopLoss:     round2(stop),
			SafetyScore:  round2(safety),
		})
	}

	if len(zones) == 0 {
		return nil
	}

	// Find optimal zone (highest safety score)
	optimal := &zones[0]
	for i := range zones {
		if zones[i].SafetyScore > optimal.SafetyScore {
			optimal = &zones[i]
		}
	}

	verdict := buildBuyZoneVerdict(price, optimal)

	return &models.BuyZoneAnalysis{
		Zones:   zones,
		Optimal: optimal,
		Verdict: verdict,
	}
}

func safetyScore(price, lower, upper float64) float64 {
	if upper <= lower {
		return 50
	}
	// Price within zone → high score; below zone → moderate; above zone → low
	pct := (price - lower) / (upper - lower)
	if pct < 0 {
		// Price below zone — could be a good buy if not too far
		return math.Max(30+pct*100, 10)
	}
	if pct > 1 {
		// Price above zone — wait for pullback
		return math.Max(70-(pct-1)*50, 10)
	}
	// Price within zone — good
	return 60 + pct*30
}

func buildBuyZoneVerdict(price float64, zone *models.BuyZone) string {
	if zone == nil {
		return "数据不足，无法计算买入区间"
	}
	if price >= zone.LowerBound && price <= zone.UpperBound {
		return fmt.Sprintf("当前价格 %.2f 处于 [%s] 买入区间内，可考虑分批建仓，止损 %.2f",
			price, zone.Method, zone.StopLoss)
	}
	if price > zone.UpperBound {
		return fmt.Sprintf("当前价格 %.2f 偏高，建议等待回调至 %.2f 附近再考虑买入",
			price, zone.LowerBound)
	}
	return fmt.Sprintf("当前价格 %.2f 已低于 [%s] 买入区间下沿，建议观望等待企稳信号",
		price, zone.Method)
}

// ──────────────────────────────────────────────
// T+0 Suggestion (做T建议) — with condition orders
// ──────────────────────────────────────────────

// AnalyzeTSuggestion generates a T+0 round-trip trading suggestion with
// specific conditional orders (条件单).
//
// Multi-layer algorithm:
//  1. ATR-based volatility band for price targets (Keltner-style)
//  2. Bollinger Bands %B for mean-reversion reference
//  3. RSI6 + KDJ-J for momentum confirmation
//  4. Intraday change % for direction detection
//  5. Cost-basis integration when holdingQty > 0
//
// Only returns a suggestion when the user holds the stock (holdingQty > 0).
func AnalyzeTSuggestion(
	quote models.Quote,
	tech models.TechnicalAnalysis,
	klines []models.KlinePoint,
	atr14 float64,
	holdingQty int,
) *models.TSuggestionAnalysis {
	if atr14 <= 0 || quote.Price <= 0 || quote.PrevClose <= 0 {
		return nil
	}

	// No position → no T suggestion
	if holdingQty <= 0 {
		return nil
	}

	price := quote.Price
	prevClose := quote.PrevClose
	priceRange := atr14 / price
	intradayPct := (price - prevClose) / prevClose

	ind := ComputeIndicators(klines)
	rsi6 := ind.RSI.RSI6
	kdjJ := ind.KDJ.J
	boll := ind.Boll

	// T trade size: 1/3 of base position, rounded to lots of 100
	tQty := holdingQty / 3
	tQty = (tQty / 100) * 100
	if tQty < 100 {
		tQty = 100
	}
	tQtyStr := fmt.Sprintf("%d股", tQty)

	// Bollinger Band %B: 0 = lower band, 1 = upper band, 0.5 = middle
	bollPctB := 0.5
	if boll.Upper > boll.Lower {
		bollPctB = (price - boll.Lower) / (boll.Upper - boll.Lower)
	}

	// ── Scoring (each signal 0-1 point) ──
	// Positive T (正T): buy dip, sell bounce
	ptScore := 0
	if intradayPct <= -0.5*priceRange {
		ptScore++
	}
	if rsi6 < 35 {
		ptScore++
	}
	if kdjJ < 20 {
		ptScore++
	}
	if bollPctB < 0.2 {
		ptScore++
	}

	// Reverse T (反T): sell rally, buy back dip
	rtScore := 0
	if intradayPct >= 0.5*priceRange {
		rtScore++
	}
	if rsi6 > 65 {
		rtScore++
	}
	if kdjJ > 80 {
		rtScore++
	}
	if bollPctB > 0.8 {
		rtScore++
	}

	// Need >= 2 signals AND direction alignment
	doPositiveT := ptScore >= 2 && intradayPct < 0
	doReverseT := rtScore >= 2 && intradayPct > 0

	if doPositiveT && doReverseT {
		if ptScore >= rtScore {
			doReverseT = false
		} else {
			doPositiveT = false
		}
	}

	// ── Price targets (Keltner-style: 0.5 ATR profit / 0.3 ATR stop) ──
	// Reward/risk ratio = 0.5/0.3 ≈ 1.67:1
	// ATR 0.5 ≈ 3-4x transaction costs (0.15%) → profitable after fees.

	if doPositiveT {
		entry := price
		target := round2(entry + 0.5*atr14)
		stop := round2(entry - 0.3*atr14)
		confidence := math.Min(round2(float64(ptScore)*25+25), 100)

		// Use Bollinger middle as target if it's between entry and ATR target
		if boll.Middle > entry && boll.Middle <= target {
			target = round2(boll.Middle)
		} else if boll.Middle > target && boll.Middle-entry <= 1.0*atr14 {
			target = round2(boll.Middle)
		}
		// Minimum 0.3% to cover transaction costs
		if target <= entry*1.003 {
			target = round2(entry * 1.003)
		}

		profitPct := (target/entry - 1) * 100

		condBuy := &models.ConditionOrder{
			Direction:     "买入",
			TriggerPrice:  round2(entry),
			TriggerDesc:   fmt.Sprintf("当前价 %.2f，日内跌%.2f%%（RSI6=%.0f J=%.0f BB%%B=%.0f%%）", entry, intradayPct*100, rsi6, kdjJ, bollPctB*100),
			OrderPrice:    round2(entry),
			OrderType:     "限价委托",
			QuantityRatio: tQtyStr,
			StopPrice:     stop,
			StopDesc:      fmt.Sprintf("跌破 %.2f 止损（-0.3ATR）", stop),
			Note:          "买入后立即挂卖出条件单",
		}
		condSell := &models.ConditionOrder{
			Direction:     "卖出",
			TriggerPrice:  target,
			TriggerDesc:   fmt.Sprintf("反弹至 %.2f 触发卖出（+%.2f%%）", target, profitPct),
			OrderPrice:    target,
			OrderType:     "限价委托",
			QuantityRatio: tQtyStr,
			StopPrice:     stop,
			StopDesc:      fmt.Sprintf("盘中跌破 %.2f 止损卖出", stop),
			Note:          fmt.Sprintf("若未达目标，收盘前 %.2f 以上择机卖出", round2(entry*1.001)),
		}

		return &models.TSuggestionAnalysis{
			Type:          "正T",
			Action:        "先买后卖",
			EntryPrice:    round2(entry),
			TargetPrice:   target,
			StopLoss:      stop,
			Reason:        fmt.Sprintf("日内跌%.2f%%，RSI6=%.0f，J=%.0f，BB%%B=%.0f%%，超卖信号%d项", intradayPct*100, rsi6, kdjJ, bollPctB*100, ptScore),
			Confidence:    confidence,
			ConditionBuy:  condBuy,
			ConditionSell: condSell,
		}
	}

	if doReverseT {
		entry := price
		target := round2(entry - 0.5*atr14)
		stop := round2(entry + 0.3*atr14)
		confidence := math.Min(round2(float64(rtScore)*25+25), 100)

		if boll.Middle < entry && boll.Middle >= target {
			target = round2(boll.Middle)
		} else if boll.Middle < target && entry-boll.Middle <= 1.0*atr14 {
			target = round2(boll.Middle)
		}
		if target >= entry*0.997 {
			target = round2(entry * 0.997)
		}

		profitPct := (1 - target/entry) * 100

		condSell := &models.ConditionOrder{
			Direction:     "卖出",
			TriggerPrice:  round2(entry),
			TriggerDesc:   fmt.Sprintf("当前价 %.2f，日内涨%.2f%%（RSI6=%.0f J=%.0f BB%%B=%.0f%%）", entry, intradayPct*100, rsi6, kdjJ, bollPctB*100),
			OrderPrice:    round2(entry),
			OrderType:     "限价委托",
			QuantityRatio: tQtyStr,
			StopPrice:     stop,
			StopDesc:      fmt.Sprintf("涨超 %.2f 止损买回（+0.3ATR，防踏空）", stop),
			Note:          "卖出后立即挂买入条件单",
		}
		condBuyBack := &models.ConditionOrder{
			Direction:     "买入",
			TriggerPrice:  target,
			TriggerDesc:   fmt.Sprintf("回落至 %.2f 触发买回（-%.2f%%）", target, profitPct),
			OrderPrice:    target,
			OrderType:     "限价委托",
			QuantityRatio: tQtyStr,
			StopPrice:     stop,
			StopDesc:      fmt.Sprintf("盘中涨超 %.2f 止损买回", stop),
			Note:          fmt.Sprintf("若未达目标，收盘前 %.2f 以下择机买回", round2(entry*0.999)),
		}

		return &models.TSuggestionAnalysis{
			Type:          "反T",
			Action:        "先卖后买",
			EntryPrice:    round2(entry),
			TargetPrice:   target,
			StopLoss:      stop,
			Reason:        fmt.Sprintf("日内涨%.2f%%，RSI6=%.0f，J=%.0f，BB%%B=%.0f%%，超买信号%d项", intradayPct*100, rsi6, kdjJ, bollPctB*100, rtScore),
			Confidence:    confidence,
			ConditionBuy:  condSell,
			ConditionSell: condBuyBack,
		}
	}

	// ── No active signal → pending conditional orders ──
	buyTrigger := round2(prevClose * (1 - 0.5*priceRange))
	buyStop := round2(buyTrigger * (1 - 0.3*priceRange))
	buyTargetRaw := math.Max(math.Max(buyTrigger*1.01, prevClose*1.002), price*1.003)
	// Sell target must be above both buy trigger and current market price
	buyTarget := round2(buyTargetRaw)

	pendingBuy := &models.ConditionOrder{
		Direction:     "买入",
		TriggerPrice:  buyTrigger,
		TriggerDesc:   fmt.Sprintf("股价跌至 %.2f 触发买入（昨收-%.2f%%）", buyTrigger, 0.5*priceRange*100),
		OrderPrice:    buyTrigger,
		OrderType:     "限价委托",
		QuantityRatio: tQtyStr,
		StopPrice:     buyStop,
		StopDesc:      fmt.Sprintf("跌破 %.2f 止损", buyStop),
		Note:          "待触发",
	}
	pendingSell := &models.ConditionOrder{
		Direction:     "卖出",
		TriggerPrice:  buyTarget,
		TriggerDesc:   fmt.Sprintf("买入后反弹至 %.2f 触发卖出止盈（+%.2f%%）", buyTarget, (buyTarget/buyTrigger-1)*100),
		OrderPrice:    buyTarget,
		OrderType:     "限价委托",
		QuantityRatio: tQtyStr,
		StopPrice:     buyStop,
		StopDesc:      fmt.Sprintf("盘中跌破 %.2f 止损卖出", buyStop),
		Note:          "待触发",
	}

	return &models.TSuggestionAnalysis{
		Type:          "观望",
		Action:        "暂无操作（可预挂条件单）",
		EntryPrice:    buyTrigger,
		TargetPrice:   buyTarget,
		StopLoss:      buyStop,
		Reason:        fmt.Sprintf("日内涨跌%.2f%%，做T阈值±%.2f%%（ATR=%.2f），RSI6=%.0f，J=%.0f", intradayPct*100, 0.5*priceRange*100, atr14, rsi6, kdjJ),
		ConditionBuy:  pendingBuy,
		ConditionSell: pendingSell,
	}
}

// ──────────────────────────────────────────────
// Intraday Forecast (日内高低点预测)
// ──────────────────────────────────────────────

// AnalyzeIntradayForecast predicts the daily high/low range and current zone.
func AnalyzeIntradayForecast(
	klines []models.KlinePoint,
	quote models.Quote,
	atr14 float64,
) *models.IntradayForecast {
	if atr14 <= 0 || quote.PrevClose <= 0 || len(klines) < 5 {
		return nil
	}

	prevClose := quote.PrevClose

	// Calculate average intraday amplitude over last 5 days
	avgAmplitude := 0.0
	count := 0
	start := len(klines) - 5
	if start < 0 {
		start = 0
	}
	for i := start; i < len(klines); i++ {
		if klines[i].Close > 0 {
			amplitude := (klines[i].High - klines[i].Low) / klines[i].Close
			avgAmplitude += amplitude
			count++
		}
	}
	if count == 0 {
		return nil
	}
	avgAmplitude /= float64(count)

	// Volatility coefficient: ratio of average amplitude to ATR-based expected range
	atrPct := atr14 / prevClose
	volCoef := 1.0
	if atrPct > 0 {
		volCoef = avgAmplitude / atrPct
	}

	predictedHigh := prevClose * (1 + atrPct*volCoef)
	predictedLow := prevClose * (1 - atrPct*volCoef)

	// Current position within predicted range
	zonePct := 50.0
	spread := predictedHigh - predictedLow
	if spread > 0 {
		zonePct = (quote.Price - predictedLow) / spread * 100
	}
	zonePct = math.Max(0, math.Min(100, zonePct))

	currentZone := "middle"
	if zonePct < 30 {
		currentZone = "lower"
	} else if zonePct > 70 {
		currentZone = "upper"
	}

	return &models.IntradayForecast{
		PredictedHigh: round2(predictedHigh),
		PredictedLow:  round2(predictedLow),
		CurrentZone:   currentZone,
		ZonePct:       round2(zonePct),
	}
}
