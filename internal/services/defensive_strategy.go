package services

import (
	"fmt"
	"math"

	"alphapulse/internal/models"
)

// BuildDefensiveStrategy turns today's kline state into a discipline-first
// position instruction. It intentionally avoids buy/sell price levels.
func BuildDefensiveStrategy(klines []models.KlinePoint, holdingQty int) *models.DefensiveStrategy {
	if len(klines) < 20 {
		return nil
	}

	alphaScore, _ := ScoreFromKlines(klines)
	riskScore, riskDims := RiskFromKlines(klines)
	state := selectedStockState(klines, riskScore)
	targetPct := selectedStockTargetPct(alphaScore, riskScore, state)
	currentPct := 0
	if holdingQty > 0 {
		currentPct = 100
	}

	action := "WAIT"
	actionLabel := "等待"
	allowIntradayOrders := false
	switch {
	case holdingQty <= 0 && targetPct >= 50:
		action = "BUILD"
		actionLabel = fmt.Sprintf("建立%d%%底仓", targetPct)
		allowIntradayOrders = true
	case holdingQty <= 0:
		action = "WAIT"
		actionLabel = "未持仓观望"
	case targetPct <= 0:
		action = "CLEAR"
		actionLabel = "清仓防守"
		allowIntradayOrders = true
	case targetPct < currentPct:
		action = "REDUCE"
		actionLabel = fmt.Sprintf("降仓至%d%%", targetPct)
		allowIntradayOrders = true
	case targetPct > currentPct:
		action = "RESTORE"
		actionLabel = fmt.Sprintf("恢复至%d%%", targetPct)
		allowIntradayOrders = true
	default:
		action = "HOLD"
		actionLabel = "维持持有"
	}

	return &models.DefensiveStrategy{
		Action:              action,
		ActionLabel:         actionLabel,
		RiskScore:           riskScore,
		AlphaScore:          alphaScore,
		RiskLevel:           defensiveRiskLevel(riskScore),
		CurrentPositionPct:  currentPct,
		TargetPositionPct:   targetPct,
		AllowIntradayOrders: allowIntradayOrders,
		Reason:              defensiveReason(action, riskScore, alphaScore, targetPct, holdingQty, state),
		ExecutionTip:        defensiveExecutionTip(action, targetPct),
		RiskNotes:           defensiveRiskNotes(riskDims, state),
	}
}

// BuildStrategyOptions returns multiple strategy disciplines for the selected
// stock. The balanced profile should normally be treated as the default.
func BuildStrategyOptions(klines []models.KlinePoint, holdingQty int) []models.StrategyOption {
	if len(klines) < 20 {
		return nil
	}
	alphaScore, _ := ScoreFromKlines(klines)
	riskScore, riskDims := RiskFromKlines(klines)
	state := selectedStockState(klines, riskScore)
	baseTarget := selectedStockTargetPct(alphaScore, riskScore, state)
	riskNotes := defensiveRiskNotes(riskDims, state)

	options := []models.StrategyOption{
		strategyOption(
			"conservative",
			"保守防守",
			"少回撤",
			clampTarget(baseTarget-20, 0, 80),
			strategyMinInt(coreFloorPct(alphaScore), 30),
			"你不想承受大波动，优先控制回撤。",
			alphaScore, riskScore, holdingQty, riskNotes, false,
		),
		strategyOption(
			"balanced",
			"均衡持有",
			"默认",
			baseTarget,
			coreFloorPct(alphaScore),
			"默认推荐：兼顾持有收益和回撤控制。",
			alphaScore, riskScore, holdingQty, riskNotes, true,
		),
		strategyOption(
			"aggressive",
			"进攻持有",
			"高收益",
			aggressiveTarget(baseTarget, alphaScore, riskScore, state),
			coreFloorPct(alphaScore),
			"你愿意承受波动，目标是尽量不丢主升段。",
			alphaScore, riskScore, holdingQty, riskNotes, false,
		),
		strategyOption(
			"rebound",
			"反弹恢复",
			"抢修复",
			reboundTarget(baseTarget, state),
			strategyMinInt(coreFloorPct(alphaScore), 40),
			"风险回落或站回MA5时，用于快速恢复部分仓位。",
			alphaScore, riskScore, holdingQty, riskNotes, false,
		),
		strategyOption(
			"exit_weak",
			"弱势退出",
			"止损",
			exitWeakTarget(alphaScore, riskScore, state),
			0,
			"趋势破坏、放量下跌或基本走弱时，优先退出弱股。",
			alphaScore, riskScore, holdingQty, riskNotes, false,
		),
	}

	return options
}

func strategyOption(
	id, name, style string,
	targetPct, minPct int,
	expectedUse string,
	alphaScore, riskScore, holdingQty int,
	riskNotes []string,
	recommended bool,
) models.StrategyOption {
	targetPct = clampTarget(targetPct, 0, 100)
	action, label := optionAction(targetPct, holdingQty)
	return models.StrategyOption{
		ID:                id,
		Name:              name,
		Style:             style,
		Action:            action,
		ActionLabel:       label,
		TargetPositionPct: targetPct,
		MinPositionPct:    minPct,
		ExpectedUse:       expectedUse,
		Reason:            fmt.Sprintf("评分%d、风险分%d，目标仓位%d%%。", alphaScore, riskScore, targetPct),
		ExecutionTip:      optionExecutionTip(action, targetPct),
		RiskNotes:         riskNotes,
		Recommended:       recommended,
	}
}

func optionAction(targetPct int, holdingQty int) (string, string) {
	if holdingQty <= 0 {
		if targetPct >= 50 {
			return "BUILD", fmt.Sprintf("建%d%%", targetPct)
		}
		return "WAIT", "等待"
	}
	if targetPct <= 0 {
		return "CLEAR", "清仓"
	}
	if targetPct < 100 {
		return "REDUCE", fmt.Sprintf("仓位%d%%", targetPct)
	}
	return "HOLD", "满仓持有"
}

func optionExecutionTip(action string, targetPct int) string {
	switch action {
	case "BUILD":
		return fmt.Sprintf("只建到%d%%，未成交不追价。", targetPct)
	case "CLEAR":
		return "执行退出后，当天不因反弹买回。"
	case "REDUCE":
		return fmt.Sprintf("仓位靠近%d%%即可，不追求精确。", targetPct)
	default:
		return "继续持有，不做日内预测挂单。"
	}
}

func aggressiveTarget(baseTarget int, alphaScore, riskScore int, state stockStrategyState) int {
	target := baseTarget + 20
	if alphaScore >= 70 && riskScore < 75 {
		target = strategyMaxInt(target, 90)
	}
	if state.strongTrend {
		target = 100
	}
	if state.breakdown || state.panicBreak {
		target = strategyMinInt(target, 50)
	}
	return clampTarget(target, 0, 100)
}

func reboundTarget(baseTarget int, state stockStrategyState) int {
	if state.rebound {
		if state.ma20 > 0 && state.close > state.ma20 {
			return strategyMaxInt(baseTarget, 80)
		}
		return strategyMaxInt(baseTarget, 60)
	}
	return clampTarget(baseTarget-10, 0, 80)
}

func exitWeakTarget(alphaScore, riskScore int, state stockStrategyState) int {
	if state.breakdown || state.panicBreak || (riskScore >= 75 && alphaScore < 65) {
		return 0
	}
	if riskScore >= 65 || alphaScore < 55 {
		return 20
	}
	return 50
}

func clampTarget(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return roundToStep(v, 10)
}

type stockStrategyState struct {
	close                float64
	ma5                  float64
	ma20                 float64
	ma60                 float64
	momentum3Pct         float64
	momentum5Pct         float64
	momentum20Pct        float64
	riskDrop             int
	strongTrend          bool
	uptrend              bool
	constructivePullback bool
	rebound              bool
	breakdown            bool
	panicBreak           bool
}

func selectedStockState(klines []models.KlinePoint, riskScore int) stockStrategyState {
	closes := make([]float64, 0, len(klines))
	for _, k := range klines {
		if k.Close > 0 {
			closes = append(closes, k.Close)
		}
	}
	last := closes[len(closes)-1]
	ma5 := MovingAverage(closes, 5)
	ma20 := MovingAverage(closes, 20)
	ma60 := 0.0
	if len(closes) >= 60 {
		ma60 = MovingAverage(closes, 60)
	}
	mom3 := momentumPct(closes, 3)
	mom5 := momentumPct(closes, 5)
	mom20 := momentumPct(closes, 20)
	prevRisk := riskScore
	if len(klines) > 20 {
		prevRisk, _ = RiskFromKlines(klines[:len(klines)-1])
	}
	riskDrop := prevRisk - riskScore

	uptrend := ma20 > 0 && last > ma20
	if ma60 > 0 {
		uptrend = uptrend && ma20 >= ma60*0.98
	}
	strongTrend := uptrend && ma5 > ma20 && mom20 > 3
	constructivePullback := ma60 > 0 && last > ma60 && last < ma20 && mom3 > 0
	rebound := riskDrop >= 8 && mom3 > 0 && ma5 > 0 && last > ma5
	if !rebound && riskScore <= 50 && mom3 > 1.5 && ma5 > 0 && last > ma5 {
		rebound = true
	}
	breakdown := ma60 > 0 && last < ma60 && ma20 < ma60 && mom20 < -5
	panicBreak := riskScore >= 85 || (ma60 > 0 && last < ma60*0.94)

	return stockStrategyState{
		close:                last,
		ma5:                  ma5,
		ma20:                 ma20,
		ma60:                 ma60,
		momentum3Pct:         mom3,
		momentum5Pct:         mom5,
		momentum20Pct:        mom20,
		riskDrop:             riskDrop,
		strongTrend:          strongTrend,
		uptrend:              uptrend,
		constructivePullback: constructivePullback,
		rebound:              rebound,
		breakdown:            breakdown,
		panicBreak:           panicBreak,
	}
}

func selectedStockTargetPct(alphaScore, riskScore int, state stockStrategyState) int {
	target := baseTargetPct(alphaScore)
	if state.strongTrend {
		target += 15
	} else if state.uptrend {
		target += 10
	} else if state.constructivePullback {
		target += 5
	} else if state.breakdown {
		target -= 30
	} else if state.ma20 > 0 && state.close < state.ma20 {
		target -= 15
	}

	switch {
	case riskScore >= 80:
		target -= 35
	case riskScore >= 65:
		target -= 25
	case riskScore >= 50:
		target -= 15
	case riskScore >= 35:
		target -= 5
	}

	if state.rebound {
		if state.ma20 > 0 && state.close > state.ma20 {
			target = strategyMaxInt(target, 70)
		} else {
			target = strategyMaxInt(target, 50)
		}
	}
	if state.strongTrend {
		target = strategyMaxInt(target, 90)
	}

	floor := coreFloorPct(alphaScore)
	if state.panicBreak {
		floor = strategyMinInt(floor, 20)
	}
	if state.breakdown && alphaScore < 65 {
		floor = 0
	}
	target = strategyMaxInt(target, floor)
	target = strategyMinInt(target, 100)
	target = strategyMaxInt(target, 0)

	return roundToStep(target, 10)
}

func baseTargetPct(alphaScore int) int {
	switch {
	case alphaScore >= 85:
		return 90
	case alphaScore >= 75:
		return 80
	case alphaScore >= 65:
		return 70
	case alphaScore >= 55:
		return 50
	default:
		return 30
	}
}

func coreFloorPct(alphaScore int) int {
	switch {
	case alphaScore >= 80:
		return 60
	case alphaScore >= 70:
		return 50
	case alphaScore >= 60:
		return 40
	case alphaScore >= 50:
		return 20
	default:
		return 0
	}
}

func defensiveRiskLevel(riskScore int) string {
	switch {
	case riskScore >= 75:
		return "极端风险"
	case riskScore >= 65:
		return "高风险"
	case riskScore >= 50:
		return "中风险"
	case riskScore >= 35:
		return "轻度风险"
	default:
		return "低风险"
	}
}

func defensiveReason(action string, riskScore, alphaScore, targetPct, holdingQty int, state stockStrategyState) string {
	if holdingQty <= 0 {
		if action == "BUILD" {
			return fmt.Sprintf("所选股票趋势/反弹条件达标，评分%d、风险分%d，允许建立%d%%策略底仓。", alphaScore, riskScore, targetPct)
		}
		return fmt.Sprintf("当前未持仓。评分%d、风险分%d，趋势或反弹确认不足，先不追买。", alphaScore, riskScore)
	}
	switch action {
	case "CLEAR":
		return fmt.Sprintf("趋势破坏且风险分%d进入极端区间，目标仓位降至%d%%，优先保住本金。", riskScore, targetPct)
	case "REDUCE":
		return fmt.Sprintf("风险分%d偏高，目标仓位降至%d%%；强股保留核心仓位，弱势破位才清仓。", riskScore, targetPct)
	case "RESTORE":
		if state.rebound {
			return fmt.Sprintf("风险分回落%d分且短线站回MA5，反弹恢复条件触发，目标仓位恢复至%d%%。", state.riskDrop, targetPct)
		}
		return fmt.Sprintf("趋势重新转强，目标仓位恢复至%d%%。", targetPct)
	default:
		return fmt.Sprintf("评分%d、风险分%d，目标仓位%d%%；继续持有核心仓位，不做日内情绪化挂单。", alphaScore, riskScore, targetPct)
	}
}

func defensiveExecutionTip(action string, targetPct int) string {
	switch action {
	case "CLEAR":
		return "按纪律清仓或降至最低防守仓位；执行后当天不再根据预测低点买回。"
	case "REDUCE":
		return fmt.Sprintf("只做一次性降仓到%d%%附近；不要因盘中反弹撤销策略动作。", targetPct)
	case "RESTORE":
		return fmt.Sprintf("只恢复到%d%%附近，不满仓追高；若次日重新跌破MA5，撤回新增仓位。", targetPct)
	case "BUILD":
		return fmt.Sprintf("只建立%d%%底仓，优先用分批成交；未成交不追价。", targetPct)
	case "HOLD":
		return "不挂日内卖单，不做T；仅保留既定止损/风控规则。"
	default:
		return "未持仓时不挂预测低买单；等趋势或反弹确认后再建仓。"
	}
}

func defensiveRiskNotes(dims map[string]interface{}, state stockStrategyState) []string {
	var notes []string
	addBool := func(key, note string) {
		if v, ok := dims[key].(bool); ok && v {
			notes = append(notes, note)
		}
	}
	addBool("break_ma20", "跌破MA20")
	addBool("deep_break_ma20", "显著跌破MA20")
	addBool("short_trend_weak", "短期均线转弱")
	addBool("long_trend_weak", "中期趋势转弱")
	addBool("volume_selloff", "放量下跌")
	if state.strongTrend {
		notes = append(notes, "强趋势")
	} else if state.uptrend {
		notes = append(notes, "趋势仍在")
	}
	if state.rebound {
		notes = append(notes, "反弹恢复")
	}
	if state.constructivePullback {
		notes = append(notes, "良性回踩")
	}
	if len(notes) == 0 {
		notes = append(notes, "未触发主要防守风险项")
	}
	return notes
}

func momentumPct(closes []float64, days int) float64 {
	if len(closes) <= days || days <= 0 {
		return 0
	}
	base := closes[len(closes)-1-days]
	if base <= 0 {
		return 0
	}
	return (closes[len(closes)-1]/base - 1) * 100
}

func roundToStep(v, step int) int {
	if step <= 0 {
		return v
	}
	return int(math.Round(float64(v)/float64(step))) * step
}

func strategyMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func strategyMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
