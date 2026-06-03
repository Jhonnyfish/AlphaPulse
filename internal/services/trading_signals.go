package services

import (
	"fmt"
	"math"
	"sort"
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
	holdingCost float64,
) *models.TSuggestionAnalysis {
	if atr14 <= 0 || quote.Price <= 0 || quote.PrevClose <= 0 {
		return nil
	}

	// No position → no T suggestion
	if holdingQty <= 0 {
		return nil
	}

	maxTQty := roundDownLot(holdingQty)
	if maxTQty < 100 {
		return &models.TSuggestionAnalysis{
			Type:         "观望",
			Action:       "底仓不足，暂不做T",
			Reason:       "当前持仓不足1手，无法覆盖A股整手交易和T+0回转约束",
			Confidence:   0,
			SignalScore:  0,
			TQuantity:    0,
			ExecutionTip: "先保持底仓，等持仓达到至少1手后再生成条件单",
			RiskNotes:    []string{"底仓不足100股，不生成做T委托数量"},
		}
	}

	price := quote.Price
	prevClose := quote.PrevClose
	atrPct := atr14 / price
	intradayPct := (price - prevClose) / prevClose

	ind := ComputeIndicators(klines)
	rsi6 := ind.RSI.RSI6
	if rsi6 <= 0 && tech.RSI_14 > 0 {
		rsi6 = tech.RSI_14
	}
	if rsi6 <= 0 {
		rsi6 = 50
	}
	kdjJ := ind.KDJ.J
	if kdjJ == 0 && tech.KDJ_J > 0 {
		kdjJ = tech.KDJ_J
	}
	if kdjJ == 0 {
		kdjJ = 50
	}
	boll := ind.Boll
	if boll.Upper <= boll.Lower && tech.BollUpper > tech.BollLower {
		boll.Upper = tech.BollUpper
		boll.Middle = tech.BollMid
		boll.Lower = tech.BollLower
	}

	triggerPct := clampFloat(0.42*atrPct, 0.006, 0.025)

	// Bollinger Band %B: 0 = lower band, 1 = upper band, 0.5 = middle
	bollPctB := 0.5
	if boll.Upper > boll.Lower {
		bollPctB = (price - boll.Lower) / (boll.Upper - boll.Lower)
	}

	var ptDetails, rtDetails, commonRisks []string
	ptScore := 0.0
	rtScore := 0.0

	if intradayPct < 0 {
		add := clampFloat(math.Abs(intradayPct)/triggerPct, 0, 1.35) * 24
		ptScore += add
		if add >= 10 {
			ptDetails = append(ptDetails, fmt.Sprintf("盘中回落%.2f%%超过触发阈值%.2f%%", math.Abs(intradayPct)*100, triggerPct*100))
		}
	} else if intradayPct > 0 {
		add := clampFloat(intradayPct/triggerPct, 0, 1.35) * 24
		rtScore += add
		if add >= 10 {
			rtDetails = append(rtDetails, fmt.Sprintf("盘中冲高%.2f%%超过触发阈值%.2f%%", intradayPct*100, triggerPct*100))
		}
	}

	if rsi6 < 50 {
		add := clampFloat((50-rsi6)/25, 0, 1) * 18
		ptScore += add
		if rsi6 <= 35 {
			ptDetails = append(ptDetails, fmt.Sprintf("RSI6=%.0f处于短线超卖区", rsi6))
		}
	}
	if rsi6 > 50 {
		add := clampFloat((rsi6-50)/25, 0, 1) * 18
		rtScore += add
		if rsi6 >= 65 {
			rtDetails = append(rtDetails, fmt.Sprintf("RSI6=%.0f处于短线偏热区", rsi6))
		}
	}

	if kdjJ < 55 {
		add := clampFloat((55-kdjJ)/45, 0, 1) * 15
		ptScore += add
		if kdjJ <= 20 {
			ptDetails = append(ptDetails, fmt.Sprintf("KDJ-J=%.0f，短线惯性释放较充分", kdjJ))
		}
	}
	if kdjJ > 45 {
		add := clampFloat((kdjJ-45)/45, 0, 1) * 15
		rtScore += add
		if kdjJ >= 80 {
			rtDetails = append(rtDetails, fmt.Sprintf("KDJ-J=%.0f，短线冲高后回落概率增加", kdjJ))
		}
	}

	if bollPctB < 0.5 {
		add := clampFloat((0.5-bollPctB)/0.45, 0, 1) * 16
		ptScore += add
		if bollPctB <= 0.2 {
			ptDetails = append(ptDetails, fmt.Sprintf("价格靠近布林下沿，BB%%B=%.0f%%", bollPctB*100))
		}
	}
	if bollPctB > 0.5 {
		add := clampFloat((bollPctB-0.5)/0.45, 0, 1) * 16
		rtScore += add
		if bollPctB >= 0.8 {
			rtDetails = append(rtDetails, fmt.Sprintf("价格靠近布林上沿，BB%%B=%.0f%%", bollPctB*100))
		}
	}

	if containsAny(tech.MAArrangement, []string{"多", "短多"}) {
		ptScore += 4
		rtScore -= 5
		rtDetails = append(rtDetails, "均线偏多，反T需防卖飞")
	}
	if containsAny(tech.MAArrangement, []string{"空", "短空"}) {
		rtScore += 3
		ptScore -= 5
		ptDetails = append(ptDetails, "均线偏空，正T只按反抽处理")
	}
	switch tech.MACD_HistTrend {
	case "连续增强":
		if tech.MACD_Hist > 0 {
			// 多头动量加速 → 正T受益（反弹概率大），反T不利
			ptScore += 4
			rtScore -= 3
			ptDetails = append(ptDetails, "MACD柱增强+多头方向，正T胜率提升")
		} else {
			// 空头动量加速 → 正T风险增大，反T顺势卖出更安全
			ptScore -= 4
			rtScore += 3
			rtDetails = append(rtDetails, "MACD柱增强+空头方向，反T顺势而为")
		}
	case "连续减弱":
		if tech.MACD_Hist > 0 {
			// 多头动能趋缓 → 反T（先卖）机会增加
			rtScore += 4
			ptScore -= 3
			rtDetails = append(rtDetails, "MACD柱减弱+多头方向，反T等待回落")
		} else {
			// 空头动能趋缓 → 正T（抄底）机会增加
			ptScore += 4
			rtScore -= 3
			ptDetails = append(ptDetails, "MACD柱减弱+空头方向，正T等待反弹")
		}
	}

	if holdingCost > 0 {
		costGapPct := (price - holdingCost) / holdingCost * 100
		switch {
		case costGapPct >= 5:
			rtScore += 5
			ptScore -= 2
			rtDetails = append(rtDetails, fmt.Sprintf("现价高于成本%.2f%%，可用反T锁定浮盈", costGapPct))
		case costGapPct <= -5:
			ptScore += 4
			rtScore -= 8
			ptDetails = append(ptDetails, fmt.Sprintf("现价低于成本%.2f%%，正T以降低持仓成本为主", math.Abs(costGapPct)))
			commonRisks = append(commonRisks, "亏损位反T容易做丢底仓，卖出后必须按计划买回")
		}
	}

	if quote.LimitDown > 0 && price <= quote.LimitDown*1.015 {
		ptScore -= 18
		commonRisks = append(commonRisks, "价格接近跌停，流动性和继续下探风险较高")
	}
	if quote.LimitUp > 0 && price >= quote.LimitUp*0.985 {
		rtScore -= 18
		commonRisks = append(commonRisks, "价格接近涨停，反T卖出后可能难以买回")
	}
	if atrPct < 0.006 {
		ptScore -= 8
		rtScore -= 8
		commonRisks = append(commonRisks, "ATR波动过低，扣除费用后做T空间有限")
	}
	if atrPct > 0.055 {
		ptScore -= 6
		rtScore -= 6
		commonRisks = append(commonRisks, "ATR波动过高，条件单滑点和止损触发概率上升")
	}

	ptScore = clampFloat(ptScore, 0, 100)
	rtScore = clampFloat(rtScore, 0, 100)

	positiveAligned := intradayPct <= -0.25*triggerPct || bollPctB <= 0.18 || (rsi6 <= 30 && kdjJ <= 25)
	reverseAligned := intradayPct >= 0.25*triggerPct || bollPctB >= 0.82 || (rsi6 >= 70 && kdjJ >= 75)
	doPositiveT := ptScore >= 55 && positiveAligned
	doReverseT := rtScore >= 55 && reverseAligned

	if doPositiveT && doReverseT {
		if ptScore >= rtScore+5 {
			doReverseT = false
		} else if rtScore >= ptScore+5 {
			doPositiveT = false
		} else {
			doPositiveT = false
			doReverseT = false
			commonRisks = append(commonRisks, "正T和反T信号接近，等待方向拉开后再执行")
		}
	}

	if doPositiveT {
		entry := price
		target, stop, profitPct, lossPct, rr := tTradePricePlan(entry, atr14, boll.Middle, true)
		riskNotes := append([]string{}, commonRisks...)
		tQty := suggestTQuantity(holdingQty, ptScore, atrPct, len(riskNotes))
		positionRatio := round2(float64(tQty) / float64(holdingQty) * 100)
		if positionRatio >= 50 {
			riskNotes = append(riskNotes, fmt.Sprintf("T仓占底仓%.0f%%，小底仓波动会更明显", positionRatio))
		}
		confidence := confidenceFromScore(ptScore, riskNotes)
		tQtyStr := fmt.Sprintf("%d股 / 底仓%.0f%%", tQty, positionRatio)

		condBuy := &models.ConditionOrder{
			Direction:     "买入",
			TriggerPrice:  round2(entry),
			TriggerDesc:   fmt.Sprintf("当前价 %.2f，日内跌%.2f%%（RSI6=%.0f J=%.0f BB%%B=%.0f%%）", entry, math.Abs(intradayPct)*100, rsi6, kdjJ, bollPctB*100),
			OrderPrice:    round2(entry),
			OrderType:     "限价委托",
			QuantityRatio: tQtyStr,
			StopPrice:     stop,
			StopDesc:      fmt.Sprintf("跌破 %.2f 止损（约-%.2f%%）", stop, lossPct),
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
			Note:          fmt.Sprintf("若未达目标，14:50前 %.2f 以上择机卖出", round2(entry*1.0015)),
		}

		return &models.TSuggestionAnalysis{
			Type:              "正T",
			Action:            "先买后卖",
			EntryPrice:        round2(entry),
			TargetPrice:       target,
			StopLoss:          stop,
			Reason:            fmt.Sprintf("正T评分%.0f，反T评分%.0f；%s", ptScore, rtScore, signalSummary(ptDetails)),
			Confidence:        confidence,
			SignalScore:       round2(ptScore),
			TQuantity:         tQty,
			PositionRatio:     positionRatio,
			TriggerPct:        round2(triggerPct * 100),
			ExpectedProfitPct: profitPct,
			MaxLossPct:        lossPct,
			RiskReward:        rr,
			ExecutionTip:      "先买入T仓，成交后立即挂目标卖出；临近收盘仍未止盈时按计划处理",
			SignalDetails:     ptDetails,
			RiskNotes:         riskNotes,
			ConditionBuy:      condBuy,
			ConditionSell:     condSell,
		}
	}

	if doReverseT {
		entry := price
		target, stop, profitPct, lossPct, rr := tTradePricePlan(entry, atr14, boll.Middle, false)
		riskNotes := append([]string{}, commonRisks...)
		tQty := suggestTQuantity(holdingQty, rtScore, atrPct, len(riskNotes))
		positionRatio := round2(float64(tQty) / float64(holdingQty) * 100)
		if positionRatio >= 50 {
			riskNotes = append(riskNotes, fmt.Sprintf("T仓占底仓%.0f%%，反T卖出后必须严格买回", positionRatio))
		}
		confidence := confidenceFromScore(rtScore, riskNotes)
		tQtyStr := fmt.Sprintf("%d股 / 底仓%.0f%%", tQty, positionRatio)

		condSell := &models.ConditionOrder{
			Direction:     "卖出",
			TriggerPrice:  round2(entry),
			TriggerDesc:   fmt.Sprintf("当前价 %.2f，日内涨%.2f%%（RSI6=%.0f J=%.0f BB%%B=%.0f%%）", entry, intradayPct*100, rsi6, kdjJ, bollPctB*100),
			OrderPrice:    round2(entry),
			OrderType:     "限价委托",
			QuantityRatio: tQtyStr,
			StopPrice:     stop,
			StopDesc:      fmt.Sprintf("涨超 %.2f 止损买回（约+%.2f%%，防踏空）", stop, lossPct),
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
			Note:          fmt.Sprintf("若未达目标，14:50前 %.2f 以下择机买回", round2(entry*0.9985)),
		}

		return &models.TSuggestionAnalysis{
			Type:              "反T",
			Action:            "先卖后买",
			EntryPrice:        round2(entry),
			TargetPrice:       target,
			StopLoss:          stop,
			Reason:            fmt.Sprintf("反T评分%.0f，正T评分%.0f；%s", rtScore, ptScore, signalSummary(rtDetails)),
			Confidence:        confidence,
			SignalScore:       round2(rtScore),
			TQuantity:         tQty,
			PositionRatio:     positionRatio,
			TriggerPct:        round2(triggerPct * 100),
			ExpectedProfitPct: profitPct,
			MaxLossPct:        lossPct,
			RiskReward:        rr,
			ExecutionTip:      "先卖出T仓，成交后立即挂目标买回；若继续放量突破，按止损价买回防踏空",
			SignalDetails:     rtDetails,
			RiskNotes:         riskNotes,
			ConditionBuy:      condSell,
			ConditionSell:     condBuyBack,
		}
	}

	// ── No active signal → pending conditional orders ──
	if ptScore >= rtScore {
		entry := round2(prevClose * (1 - triggerPct))
		target, stop, profitPct, lossPct, rr := tTradePricePlan(entry, atr14, boll.Middle, true)
		riskNotes := append([]string{}, commonRisks...)
		tQty := suggestTQuantity(holdingQty, ptScore, atrPct, len(riskNotes))
		positionRatio := round2(float64(tQty) / float64(holdingQty) * 100)
		if positionRatio >= 50 {
			riskNotes = append(riskNotes, fmt.Sprintf("T仓占底仓%.0f%%，小底仓建议降低执行频率", positionRatio))
		}
		tQtyStr := fmt.Sprintf("%d股 / 底仓%.0f%%", tQty, positionRatio)
		pendingBuy := &models.ConditionOrder{
			Direction:     "买入",
			TriggerPrice:  entry,
			TriggerDesc:   fmt.Sprintf("股价回落至 %.2f 触发正T买入（昨收-%.2f%%）", entry, triggerPct*100),
			OrderPrice:    entry,
			OrderType:     "限价委托",
			QuantityRatio: tQtyStr,
			StopPrice:     stop,
			StopDesc:      fmt.Sprintf("跌破 %.2f 止损（约-%.2f%%）", stop, lossPct),
			Note:          "待触发",
		}
		pendingSell := &models.ConditionOrder{
			Direction:     "卖出",
			TriggerPrice:  target,
			TriggerDesc:   fmt.Sprintf("买入后反弹至 %.2f 触发卖出止盈（+%.2f%%）", target, profitPct),
			OrderPrice:    target,
			OrderType:     "限价委托",
			QuantityRatio: tQtyStr,
			StopPrice:     stop,
			StopDesc:      fmt.Sprintf("盘中跌破 %.2f 止损卖出", stop),
			Note:          "待触发",
		}
		return &models.TSuggestionAnalysis{
			Type:              "观望",
			Action:            "等待回落触发正T",
			EntryPrice:        entry,
			TargetPrice:       target,
			StopLoss:          stop,
			Reason:            fmt.Sprintf("正T评分%.0f，反T评分%.0f；即时信号不足，等待价格进入回落区间", ptScore, rtScore),
			Confidence:        confidenceFromScore(ptScore, riskNotes),
			SignalScore:       round2(ptScore),
			TQuantity:         tQty,
			PositionRatio:     positionRatio,
			TriggerPct:        round2(triggerPct * 100),
			ExpectedProfitPct: profitPct,
			MaxLossPct:        lossPct,
			RiskReward:        rr,
			ExecutionTip:      "仅预挂首个触发单，成交后再挂止盈/止损单，避免未成交时误挂反向单",
			SignalDetails:     ptDetails,
			RiskNotes:         riskNotes,
			ConditionBuy:      pendingBuy,
			ConditionSell:     pendingSell,
		}
	}

	entry := round2(prevClose * (1 + triggerPct))
	target, stop, profitPct, lossPct, rr := tTradePricePlan(entry, atr14, boll.Middle, false)
	riskNotes := append([]string{}, commonRisks...)
	tQty := suggestTQuantity(holdingQty, rtScore, atrPct, len(riskNotes))
	positionRatio := round2(float64(tQty) / float64(holdingQty) * 100)
	if positionRatio >= 50 {
		riskNotes = append(riskNotes, fmt.Sprintf("T仓占底仓%.0f%%，反T触发后必须严格买回", positionRatio))
	}
	tQtyStr := fmt.Sprintf("%d股 / 底仓%.0f%%", tQty, positionRatio)
	pendingSell := &models.ConditionOrder{
		Direction:     "卖出",
		TriggerPrice:  entry,
		TriggerDesc:   fmt.Sprintf("股价冲高至 %.2f 触发反T卖出（昨收+%.2f%%）", entry, triggerPct*100),
		OrderPrice:    entry,
		OrderType:     "限价委托",
		QuantityRatio: tQtyStr,
		StopPrice:     stop,
		StopDesc:      fmt.Sprintf("涨超 %.2f 止损买回（约+%.2f%%）", stop, lossPct),
		Note:          "待触发",
	}
	pendingBuyBack := &models.ConditionOrder{
		Direction:     "买入",
		TriggerPrice:  target,
		TriggerDesc:   fmt.Sprintf("卖出后回落至 %.2f 触发买回（-%.2f%%）", target, profitPct),
		OrderPrice:    target,
		OrderType:     "限价委托",
		QuantityRatio: tQtyStr,
		StopPrice:     stop,
		StopDesc:      fmt.Sprintf("盘中涨超 %.2f 止损买回", stop),
		Note:          "待触发",
	}

	return &models.TSuggestionAnalysis{
		Type:              "观望",
		Action:            "等待冲高触发反T",
		EntryPrice:        entry,
		TargetPrice:       target,
		StopLoss:          stop,
		Reason:            fmt.Sprintf("反T评分%.0f，正T评分%.0f；即时信号不足，等待价格进入冲高区间", rtScore, ptScore),
		Confidence:        confidenceFromScore(rtScore, riskNotes),
		SignalScore:       round2(rtScore),
		TQuantity:         tQty,
		PositionRatio:     positionRatio,
		TriggerPct:        round2(triggerPct * 100),
		ExpectedProfitPct: profitPct,
		MaxLossPct:        lossPct,
		RiskReward:        rr,
		ExecutionTip:      "仅预挂首个触发单，成交后再挂买回/止损单，避免未成交时误挂反向单",
		SignalDetails:     rtDetails,
		RiskNotes:         riskNotes,
		ConditionBuy:      pendingSell,
		ConditionSell:     pendingBuyBack,
	}
}

func tTradePricePlan(entry, atr14, bollMiddle float64, positive bool) (target, stop, profitPct, lossPct, riskReward float64) {
	profitMove := math.Max(0.45*atr14, entry*0.0045)
	stopMove := math.Max(0.30*atr14, entry*0.003)

	if positive {
		target = entry + profitMove
		minTarget := entry * 1.0045
		if bollMiddle > minTarget && bollMiddle < target {
			target = bollMiddle
		}
		if target < minTarget {
			target = minTarget
		}
		stop = entry - stopMove
		profitPct = (target/entry - 1) * 100
		lossPct = (1 - stop/entry) * 100
	} else {
		target = entry - profitMove
		maxTarget := entry * 0.9955
		if bollMiddle > target && bollMiddle < maxTarget {
			target = bollMiddle
		}
		if target > maxTarget {
			target = maxTarget
		}
		stop = entry + stopMove
		profitPct = (1 - target/entry) * 100
		lossPct = (stop/entry - 1) * 100
	}

	if lossPct > 0 {
		riskReward = profitPct / lossPct
	}
	return round2(target), round2(stop), round2(profitPct), round2(lossPct), round2(riskReward)
}

func suggestTQuantity(holdingQty int, score, atrPct float64, riskCount int) int {
	maxQty := roundDownLot(holdingQty)
	if maxQty < 100 {
		return 0
	}

	ratio := 0.20
	if score >= 78 {
		ratio = 1.0 / 3.0
	} else if score >= 65 {
		ratio = 0.25
	}
	if atrPct > 0.035 {
		ratio -= 0.05
	}
	if riskCount >= 2 {
		ratio -= 0.05
	}
	ratio = clampFloat(ratio, 0.15, 0.35)

	qty := roundDownLot(int(float64(holdingQty) * ratio))
	if qty < 100 {
		qty = 100
	}
	if qty > maxQty {
		qty = maxQty
	}
	return qty
}

func roundDownLot(qty int) int {
	if qty <= 0 {
		return 0
	}
	return qty / 100 * 100
}

func confidenceFromScore(score float64, risks []string) float64 {
	confidence := score - math.Min(float64(len(risks))*4, 16)
	return round2(clampFloat(confidence, 0, 95))
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func signalSummary(details []string) string {
	if len(details) == 0 {
		return "信号未形成共振"
	}
	if len(details) > 3 {
		details = details[:3]
	}
	return strings.Join(details, "；")
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	if len(ss) > 3 {
		ss = ss[:3]
	}
	return strings.Join(ss, sep)
}

// ──────────────────────────────────────────────
// Intraday Forecast (日内高低点预测)
// ──────────────────────────────────────────────
//
// The forecast uses empirical quantiles of past high/low excursions rather
// than a parametric ATR-based formula. For each past day i we compute:
//
//   up_exc[i]   = (high[i] - close[i-1]) / close[i-1]
//   down_exc[i] = (close[i-1] - low[i])  / close[i-1]
//
// Today's predicted high is then close[-1] × (1 + P70(up_exc)), predicted
// low is close[-1] × (1 - P70(down_exc)). Wide band uses P90. This is
// non-parametric, automatically adapts to any volatility regime, and by
// construction targets ~70% central / ~90% wide hit rates.
//
// Pattern bias is applied as an additive excursion shift (capped at
// 0.4 × ATR / prevClose per side) so high-confidence bullish patterns
// push the high prediction up without affecting the low.

// excursionLookback is how many recent days of excursions feed the
// percentile estimate. 60 is enough for stable quantiles while staying
// responsive to volatility regime changes.
const excursionLookback = 60

// excursionMinSample is the minimum sample size needed before we trust
// the empirical quantiles. Below this we return nil (caller will fall
// back to "forecast unavailable" rather than emit garbage).
const excursionMinSample = 20

// AnalyzeIntradayForecast predicts today's daily high/low using empirical
// quantile estimation of past (high, low) excursions from prev_close.
func AnalyzeIntradayForecast(
	klines []models.KlinePoint,
	quote models.Quote,
	atr14 float64,
	patterns []models.PatternSignal,
	support float64,
	resistance float64,
) *models.IntradayForecast {
	if quote.PrevClose <= 0 || len(klines) < excursionMinSample+1 {
		return nil
	}
	prevClose := quote.PrevClose

	upPCentral, upPWide, upPMedian, downPCentral, downPWide, downPMedian, ok := computeExcursionStats(klines)
	if !ok {
		return nil
	}

	// Base predictions: empirical percentiles of historical excursions.
	// Per-side hit rate = percentile value, so joint hit rate ≈ p². To target
	// ~70% joint central hit and ~90% joint wide hit we need per-side rates
	// of √0.70 ≈ 0.837 and √0.90 ≈ 0.949 — i.e. P83 / P95.
	predictedHigh := prevClose * (1 + upPCentral)
	predictedLow := prevClose * (1 - downPCentral)
	wideHigh := prevClose * (1 + upPWide)
	wideLow := prevClose * (1 - downPWide)
	medianHigh := prevClose * (1 + upPMedian)
	medianLow := prevClose * (1 - downPMedian)

	// --- Pattern bias (confidence-weighted, expressed as % of prevClose) ---
	// Same magnitude as the old ATR-based shift, but applied to the
	// percentile-based prediction. Capped to prevent runaway.
	var bullScore, bearScore float64
	var bullReasons, bearReasons []string
	for _, p := range patterns {
		switch p.Direction {
		case "bullish":
			bullScore += p.Confidence
			bullReasons = append(bullReasons, p.Pattern)
		case "bearish":
			bearScore += p.Confidence
			bearReasons = append(bearReasons, p.Pattern)
		}
	}
	bullScore = math.Min(bullScore, 2.5)
	bearScore = math.Min(bearScore, 2.5)
	hasBullish := bullScore > 0
	hasBearish := bearScore > 0

	atrPct := 0.02
	if atr14 > 0 {
		atrPct = atr14 / prevClose
		if atrPct <= 0 {
			atrPct = 0.02
		}
	}
	bullShift := math.Min(bullScore*0.15*atrPct, 0.4*atrPct)
	bearShift := math.Min(bearScore*0.15*atrPct, 0.4*atrPct)

	predictedHigh += prevClose * bullShift
	predictedLow -= prevClose * bearShift
	wideHigh += prevClose * bullShift
	wideLow -= prevClose * bearShift

	// σ displayed in the UI = distance from central to wide band. This is
	// consistent with how the backtest grades "wide" hits and gives σ a
	// concrete interpretation rather than the misleading "±1 std-dev".
	sigmaHigh := math.Max(0, wideHigh-predictedHigh)
	sigmaLow := math.Max(0, predictedLow-wideLow)

	// Minimum range floor: at least 0.3% of price, so the visual band
	// doesn't collapse to a line on extremely low-volatility stocks.
	minSpread := 0.003 * prevClose
	if predictedHigh-predictedLow < minSpread {
		mid := (predictedHigh + predictedLow) / 2
		predictedHigh = mid + minSpread/2
		predictedLow = mid - minSpread/2
	}

	// Bias metadata for the UI.
	var biasReasons []string
	biasReasons = append(biasReasons, bullReasons...)
	biasReasons = append(biasReasons, bearReasons...)

	bias := "neutral"
	biasReason := ""
	biasStrength := 0.0
	if hasBullish && !hasBearish {
		bias = "bullish"
		biasReason = joinStrings(biasReasons, "+")
		biasStrength = bullScore
	} else if hasBearish && !hasBullish {
		bias = "bearish"
		biasReason = joinStrings(biasReasons, "+")
		biasStrength = bearScore
	} else if hasBullish && hasBearish {
		bias = "mixed"
		biasReason = joinStrings(biasReasons, " / ")
		biasStrength = math.Abs(bullScore - bearScore)
	}

	// Current position within predicted range.
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
		PredictedHigh:       round2(predictedHigh),
		PredictedLow:        round2(predictedLow),
		PredictedHighUp:     round2(wideHigh),
		PredictedHighDown:   round2(predictedHigh - sigmaHigh),
		PredictedLowUp:      round2(predictedLow + sigmaLow),
		PredictedLowDown:    round2(wideLow),
		PredictedHighMedian: round2(medianHigh),
		PredictedLowMedian:  round2(medianLow),
		SigmaHigh:           round2(sigmaHigh),
		SigmaLow:            round2(sigmaLow),
		CurrentZone:         currentZone,
		ZonePct:             round2(zonePct),
		Bias:                bias,
		BiasReason:          biasReason,
		BiasStrength:        round2(biasStrength),
		SupportLevel:        round2(support),
		ResistLevel:         round2(resistance),
	}
}

// computeExcursionStats returns percentile (P70, P90) and std-dev of recent
// up/down excursions from prev_close. Returns ok=false when there isn't
// enough history to compute a reliable estimate.
//
// Up excursion on day i:   (high[i] - close[i-1]) / close[i-1]
// Down excursion on day i: (close[i-1] - low[i])  / close[i-1]
//
// Both are signed positive — up excursion measures how far above the
// previous close the day's high reached, down measures how far below.
func computeExcursionStats(klines []models.KlinePoint) (
	upPCentral, upPWide, upPMedian float64,
	downPCentral, downPWide, downPMedian float64,
	ok bool,
) {
	n := len(klines)
	if n < excursionMinSample+1 {
		return
	}

	start := n - excursionLookback
	if start < 1 {
		start = 1
	}

	ups := make([]float64, 0, n-start)
	downs := make([]float64, 0, n-start)
	for i := start; i < n; i++ {
		prev := klines[i-1].Close
		if prev <= 0 {
			continue
		}
		ups = append(ups, (klines[i].High-prev)/prev)
		downs = append(downs, (prev-klines[i].Low)/prev)
	}
	if len(ups) < excursionMinSample {
		return
	}

	upPMedian = quantile(ups, 0.50)
	upPCentral = quantile(ups, 0.83)
	upPWide = quantile(ups, 0.95)
	downPMedian = quantile(downs, 0.50)
	downPCentral = quantile(downs, 0.83)
	downPWide = quantile(downs, 0.95)
	ok = true
	return
}

// quantile returns the p-th quantile of values using linear interpolation
// between closest ranks. p must be in [0, 1]; values may be unsorted.
// (A separate percentile() in perf_tracker.go expects pre-sorted input and
// a [0, 100] percentile scale — kept distinct to avoid collision.)
func quantile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if p <= 0 {
		return minFloat(values)
	}
	if p >= 1 {
		return maxFloat(values)
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	rank := p * float64(len(sorted)-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sorted[lo]
	}
	frac := rank - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}

// stddev returns the sample standard deviation (Bessel-corrected, n-1).
func stddev(values []float64) float64 {
	n := len(values)
	if n < 2 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(n)
	var sq float64
	for _, v := range values {
		d := v - mean
		sq += d * d
	}
	return math.Sqrt(sq / float64(n-1))
}
