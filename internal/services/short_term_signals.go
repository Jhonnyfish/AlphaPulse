package services

import (
	"fmt"
	"sort"

	"alphapulse/internal/models"
)

// AnalyzePatternSignals summarizes K-line, chart, and volume pattern signals.
func AnalyzePatternSignals(patterns []models.PatternSignal) *models.PatternAnalysis {
	if len(patterns) == 0 {
		return &models.PatternAnalysis{
			Signals:     []models.PatternSignal{},
			NetBias:     "neutral",
			ScoreImpact: 0,
			Verdict:     "暂无明确K线或图形形态信号",
		}
	}

	signals := append([]models.PatternSignal{}, patterns...)
	sort.SliceStable(signals, func(i, j int) bool {
		return signals[i].Confidence > signals[j].Confidence
	})

	a := &models.PatternAnalysis{Signals: signals}
	var impact float64
	for _, p := range signals {
		switch p.Category {
		case "kline":
			a.KlineSignals = append(a.KlineSignals, p)
		case "chart":
			a.ChartSignals = append(a.ChartSignals, p)
		case "volume":
			a.VolumeSignals = append(a.VolumeSignals, p)
		}

		weight := 1.0
		if p.Category == "chart" {
			weight = 1.2
		}
		switch p.Direction {
		case "bullish":
			a.BullishCount++
			impact += p.Confidence * weight
		case "bearish":
			a.BearishCount++
			impact -= p.Confidence * weight
		default:
			a.NeutralCount++
		}
	}

	if len(signals) > 0 {
		primary := signals[0]
		a.Primary = &primary
	}

	a.ScoreImpact = round2(clampFloat(impact*12, -25, 25))
	switch {
	case a.ScoreImpact >= 12:
		a.NetBias = "bullish"
		a.Verdict = fmt.Sprintf("形态信号偏多，核心信号为%s，短线有上攻或反弹倾向", a.Primary.Pattern)
	case a.ScoreImpact <= -12:
		a.NetBias = "bearish"
		a.Verdict = fmt.Sprintf("形态信号偏空，核心信号为%s，短线需防回落", a.Primary.Pattern)
	case a.BullishCount > 0 && a.BearishCount > 0:
		a.NetBias = "mixed"
		a.Verdict = "多空形态并存，等待放量突破或跌破后再确认方向"
	default:
		a.NetBias = "neutral"
		if a.Primary != nil {
			a.Verdict = fmt.Sprintf("检测到%s，但方向性不足，适合作为辅助观察信号", a.Primary.Pattern)
		} else {
			a.Verdict = "暂无明确方向性形态"
		}
	}

	return a
}

// AnalyzeShortTermScore computes a tactical 1-5 day score from existing dimensions.
func AnalyzeShortTermScore(
	quote models.Quote,
	tech models.TechnicalAnalysis,
	volume models.VolumePriceAnalysis,
	money models.MoneyFlowAnalysis,
	trend models.TrendAnalysis,
	patterns *models.PatternAnalysis,
	forecast *models.IntradayForecast,
) *models.ShortTermScore {
	components := []models.ScoreComponent{
		scoreTechnicalMomentum(quote, tech, trend),
		scoreVolumePrice(volume),
		scoreMoneyFlow(money),
		scorePatternComponent(patterns),
		scoreIntradayPosition(forecast),
	}

	totalWeight := 0.0
	score := 0.0
	var reasons, risks []string
	for _, c := range components {
		totalWeight += c.Weight
		score += c.Score * c.Weight
		if c.Score >= 65 {
			reasons = append(reasons, c.Reason)
		} else if c.Score <= 42 {
			risks = append(risks, c.Reason)
		}
	}
	if totalWeight > 0 {
		score /= totalWeight
	}
	score = round2(clampFloat(score, 0, 100))

	grade := "C"
	signal := "观望"
	switch {
	case score >= 80:
		grade = "A"
		signal = "强势"
	case score >= 68:
		grade = "B"
		signal = "偏强"
	case score >= 55:
		grade = "C+"
		signal = "中性偏强"
	case score >= 45:
		grade = "C"
		signal = "中性"
	case score >= 35:
		grade = "D"
		signal = "偏弱"
	default:
		grade = "E"
		signal = "弱势"
	}

	verdict := fmt.Sprintf("短线评分%.0f，评级%s，%s", score, grade, signal)
	if len(reasons) > 0 {
		verdict += "；优势：" + joinStrings(reasons, "；")
	}
	if len(risks) > 0 {
		verdict += "；风险：" + joinStrings(risks, "；")
	}

	return &models.ShortTermScore{
		Score:      score,
		Grade:      grade,
		Signal:     signal,
		Components: components,
		Reasons:    trimStrings(reasons, 4),
		Risks:      trimStrings(risks, 4),
		Verdict:    verdict,
	}
}

func scoreTechnicalMomentum(quote models.Quote, tech models.TechnicalAnalysis, trend models.TrendAnalysis) models.ScoreComponent {
	score := 50.0
	var reason string
	switch {
	case containsAny(tech.MAArrangement, []string{"多头", "短多"}):
		score += 14
		reason = "均线偏多"
	case containsAny(tech.MAArrangement, []string{"空头", "短空"}):
		score -= 14
		reason = "均线偏空"
	default:
		reason = "均线震荡"
	}
	if containsAny(tech.MACD_Signal, []string{"金叉", "多头"}) {
		score += 8
		reason += "，MACD支持"
	} else if containsAny(tech.MACD_Signal, []string{"死叉", "空头"}) {
		score -= 8
		reason += "，MACD压制"
	}
	if tech.RSI_14 >= 55 && tech.RSI_14 <= 72 {
		score += 6
	} else if tech.RSI_14 > 78 {
		score -= 8
		reason += "，RSI过热"
	} else if tech.RSI_14 < 35 && tech.RSI_14 > 0 {
		score -= 5
		reason += "，RSI偏弱"
	}
	if trend.TrendStage.Direction == "上升" {
		score += trend.TrendStage.Strength * 0.08
	} else if trend.TrendStage.Direction == "下降" {
		score -= trend.TrendStage.Strength * 0.08
	}
	if quote.ChangePercent > 0 {
		score += clampFloat(quote.ChangePercent, 0, 5)
	} else {
		score += clampFloat(quote.ChangePercent, -5, 0)
	}
	return models.ScoreComponent{Name: "技术动量", Score: round2(clampFloat(score, 0, 100)), Weight: 0.28, Reason: reason}
}

func scoreVolumePrice(volume models.VolumePriceAnalysis) models.ScoreComponent {
	score := 50.0
	reason := "量价中性"
	if containsAny(volume.PriceVolumeHarmony, []string{"量价齐升", "价涨量增"}) {
		score += 18
		reason = "量价齐升"
	} else if containsAny(volume.PriceVolumeHarmony, []string{"下跌放量", "价跌量增"}) {
		score -= 18
		reason = "下跌放量"
	}
	if volume.VolumeRatio >= 1.5 && volume.TodayChangePct > 0 {
		score += 8
		reason += "，放量上涨"
	} else if volume.VolumeRatio >= 1.5 && volume.TodayChangePct < 0 {
		score -= 8
		reason += "，放量下跌"
	} else if volume.VolumeRatio < 0.7 {
		score -= 3
		reason += "，量能不足"
	}
	return models.ScoreComponent{Name: "量价配合", Score: round2(clampFloat(score, 0, 100)), Weight: 0.20, Reason: reason}
}

func scoreMoneyFlow(money models.MoneyFlowAnalysis) models.ScoreComponent {
	score := 50.0
	reason := "资金方向中性"
	if money.TodayMainDirection == "流入" {
		score += 14
		reason = "主力净流入"
	} else if money.TodayMainDirection == "流出" {
		score -= 14
		reason = "主力净流出"
	}
	if money.MainConsecutiveDirection == "流入" && money.MainConsecutiveDays >= 2 {
		score += float64(minInt(money.MainConsecutiveDays, 5)) * 3
		reason += fmt.Sprintf("，连续%d天流入", money.MainConsecutiveDays)
	} else if money.MainConsecutiveDirection == "流出" && money.MainConsecutiveDays >= 2 {
		score -= float64(minInt(money.MainConsecutiveDays, 5)) * 3
		reason += fmt.Sprintf("，连续%d天流出", money.MainConsecutiveDays)
	}
	return models.ScoreComponent{Name: "资金流", Score: round2(clampFloat(score, 0, 100)), Weight: 0.20, Reason: reason}
}

func scorePatternComponent(patterns *models.PatternAnalysis) models.ScoreComponent {
	score := 50.0
	reason := "暂无明确形态"
	if patterns != nil {
		score += patterns.ScoreImpact
		if patterns.Primary != nil {
			reason = fmt.Sprintf("核心形态%s", patterns.Primary.Pattern)
		} else {
			reason = patterns.Verdict
		}
	}
	return models.ScoreComponent{Name: "形态信号", Score: round2(clampFloat(score, 0, 100)), Weight: 0.18, Reason: reason}
}

func scoreIntradayPosition(forecast *models.IntradayForecast) models.ScoreComponent {
	score := 50.0
	reason := "日内位置中性"
	if forecast != nil {
		switch forecast.CurrentZone {
		case "lower":
			score += 8
			reason = "价格处于日内预测低位"
		case "upper":
			score -= 8
			reason = "价格处于日内预测高位"
		}
		if forecast.Bias == "bullish" {
			score += 8
			reason += "，预测偏多"
		} else if forecast.Bias == "bearish" {
			score -= 8
			reason += "，预测偏空"
		}
	}
	return models.ScoreComponent{Name: "日内位置", Score: round2(clampFloat(score, 0, 100)), Weight: 0.14, Reason: reason}
}

func trimStrings(ss []string, n int) []string {
	if len(ss) <= n {
		return ss
	}
	return ss[:n]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
