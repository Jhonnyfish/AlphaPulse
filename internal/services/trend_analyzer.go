package services

import (
	"fmt"
	"math"

	"alphapulse/internal/models"
)

// TrendAnalyzer analyzes trend stages and support/resistance levels.
type TrendAnalyzer struct{}

// NewTrendAnalyzer creates a new TrendAnalyzer.
func NewTrendAnalyzer() *TrendAnalyzer {
	return &TrendAnalyzer{}
}

// AnalyzeTrend performs full trend analysis on a stock.
func (ta *TrendAnalyzer) AnalyzeTrend(
	tech models.TechnicalAnalysis,
	vp models.VolumePriceAnalysis,
	price float64,
) models.TrendAnalysis {
	stage := ta.analyzeTrendStage(tech, vp, price)
	sr := ta.calculateSupportResistance(tech, price)
	verdict := ta.buildVerdict(stage, sr, price)

	return models.TrendAnalysis{
		TrendStage:        stage,
		SupportResistance: sr,
		Verdict:           verdict,
	}
}

// analyzeTrendStage determines the current trend direction and stage.
func (ta *TrendAnalyzer) analyzeTrendStage(
	tech models.TechnicalAnalysis,
	vp models.VolumePriceAnalysis,
	price float64,
) models.TrendStage {
	var signals []string
	score := 0.0
	maxScore := 0.0

	// 1. MA Arrangement Analysis (weight: 30)
	maxScore += 30
	switch tech.MAArrangement {
	case "多头排列":
		score += 30
		signals = append(signals, "均线多头排列")
	case "空头排列":
		score -= 30
		signals = append(signals, "均线空头排列")
	case "交叉":
		score += 0
		signals = append(signals, "均线交叉")
	}

	// 2. MACD Signal Analysis (weight: 25)
	maxScore += 25
	switch tech.MACD_Signal {
	case "金叉":
		score += 25
		signals = append(signals, "MACD金叉")
	case "死叉":
		score -= 25
		signals = append(signals, "MACD死叉")
	case "多头":
		score += 15
		signals = append(signals, "MACD多头")
	case "空头":
		score -= 15
		signals = append(signals, "MACD空头")
	}

	// 3. MACD Histogram Trend (weight: 15)
	maxScore += 15
	switch tech.MACD_HistTrend {
	case "连续增强":
		score += 15
		signals = append(signals, "MACD柱状连续增强")
	case "连续减弱":
		score -= 5
		signals = append(signals, "MACD柱状连续减弱")
	case "转强":
		score += 10
		signals = append(signals, "MACD柱状转强")
	case "转弱":
		score -= 10
		signals = append(signals, "MACD柱状转弱")
	}

	// 4. RSI Level (weight: 15)
	maxScore += 15
	switch tech.RSI_Level {
	case "超买":
		score -= 10
		signals = append(signals, "RSI超买")
	case "偏强":
		score += 10
		signals = append(signals, "RSI偏强")
	case "超卖":
		score += 10
		signals = append(signals, "RSI超卖")
	case "偏弱":
		score -= 10
		signals = append(signals, "RSI偏弱")
	default:
		score += 0
	}

	// 5. Volume Confirmation (weight: 15)
	maxScore += 15
	switch vp.PriceVolumeHarmony {
	case "量价齐升":
		score += 15
		signals = append(signals, "量价齐升确认")
	case "放量下跌":
		score -= 15
		signals = append(signals, "放量下跌")
	case "缩量上涨":
		score += 5
		signals = append(signals, "缩量上涨")
	case "缩量下跌":
		score -= 5
		signals = append(signals, "缩量下跌")
	}

	// Normalize score to 0-100
	normalizedScore := ((score / maxScore) * 50) + 50
	if normalizedScore > 100 {
		normalizedScore = 100
	}
	if normalizedScore < 0 {
		normalizedScore = 0
	}

	// Determine direction
	direction := "震荡"
	if normalizedScore >= 65 {
		direction = "上升"
	} else if normalizedScore <= 35 {
		direction = "下降"
	}

	// Determine stage based on momentum and distance from MAs
	stage := ta.determineStage(tech, price, normalizedScore, direction)

	// Calculate confidence based on signal consistency
	confidence := ta.calculateConfidence(tech, vp)

	return models.TrendStage{
		Direction:   direction,
		Stage:       stage,
		Strength:    math.Round(normalizedScore*10) / 10,
		Confidence:  math.Round(confidence*10) / 10,
		Signals:     signals,
		Description: ta.buildStageDescription(direction, stage, normalizedScore),
	}
}

// determineStage identifies if the trend is early, mid, or late.
func (ta *TrendAnalyzer) determineStage(
	tech models.TechnicalAnalysis,
	price float64,
	score float64,
	direction string,
) string {
	if direction == "震荡" {
		return "震荡"
	}

	// Calculate distance from key MAs
	ma20Dist := 0.0
	if tech.MA20 > 0 {
		ma20Dist = ((price - tech.MA20) / tech.MA20) * 100
	}

	// Check MACD histogram momentum
	isMomentumDecreasing := tech.MACD_HistTrend == "连续减弱" || tech.MACD_HistTrend == "转弱"

	if direction == "上升" {
		// Early uptrend: just crossed above MAs, MACD turning positive
		if ma20Dist < 5 && tech.MACD_Signal == "金叉" {
			return "早期"
		}
		// Late uptrend: far from MAs, momentum decreasing, RSI overbought
		if (ma20Dist > 15 || tech.RSI_Level == "超买") && isMomentumDecreasing {
			return "末期"
		}
		// Mid uptrend: moderate distance, momentum intact
		return "中期"
	}

	if direction == "下降" {
		// Early downtrend: just crossed below MAs
		if ma20Dist > -5 && tech.MACD_Signal == "死叉" {
			return "早期"
		}
		// Late downtrend: far below MAs, RSI oversold
		if (ma20Dist < -15 || tech.RSI_Level == "超卖") && !isMomentumDecreasing {
			return "末期"
		}
		return "中期"
	}

	return "震荡"
}

// calculateConfidence measures how consistent the signals are.
func (ta *TrendAnalyzer) calculateConfidence(
	tech models.TechnicalAnalysis,
	vp models.VolumePriceAnalysis,
) float64 {
	confidence := 50.0 // base confidence

	// MA alignment adds confidence
	if tech.MAArrangement == "多头排列" || tech.MAArrangement == "空头排列" {
		confidence += 20
	}

	// MACD confirmation
	if (tech.MACD_Signal == "金叉" || tech.MACD_Signal == "多头") &&
		tech.MACD_HistTrend == "连续增强" {
		confidence += 15
	}
	if (tech.MACD_Signal == "死叉" || tech.MACD_Signal == "空头") &&
		tech.MACD_HistTrend == "连续减弱" {
		confidence += 15
	}

	// Volume confirmation
	if vp.PriceVolumeHarmony == "量价齐升" || vp.PriceVolumeHarmony == "放量下跌" {
		confidence += 15
	}

	// Period alignment
	if tech.PeriodAlign == "日周共振看多" || tech.PeriodAlign == "日周共振看空" {
		confidence += 10
	}

	if confidence > 100 {
		confidence = 100
	}

	return confidence
}

// calculateSupportResistance computes key price levels.
func (ta *TrendAnalyzer) calculateSupportResistance(
	tech models.TechnicalAnalysis,
	price float64,
) models.SupportResistance {
	sr := models.SupportResistance{
		MA5:       math.Round(tech.MA5*100) / 100,
		MA10:      math.Round(tech.MA10*100) / 100,
		MA20:      math.Round(tech.MA20*100) / 100,
		MA60:      math.Round(tech.MA60*100) / 100,
		BollUpper: math.Round(tech.BollUpper*100) / 100,
		BollMid:   math.Round(tech.BollMid*100) / 100,
		BollLower: math.Round(tech.BollLower*100) / 100,
	}

	// Collect all potential levels
	levels := []struct {
		price float64
		label string
	}{
		{tech.MA5, "MA5"},
		{tech.MA10, "MA10"},
		{tech.MA20, "MA20"},
		{tech.MA60, "MA60"},
		{tech.BollUpper, "Boll上轨"},
		{tech.BollMid, "Boll中轨"},
		{tech.BollLower, "Boll下轨"},
	}

	// Separate into support (below price) and resistance (above price)
	var supports, resistances []float64
	for _, l := range levels {
		if l.price <= 0 {
			continue
		}
		if l.price < price {
			supports = append(supports, l.price)
		} else if l.price > price {
			resistances = append(resistances, l.price)
		}
	}

	// Sort supports descending (nearest first)
	for i := 0; i < len(supports); i++ {
		for j := i + 1; j < len(supports); j++ {
			if supports[j] > supports[i] {
				supports[i], supports[j] = supports[j], supports[i]
			}
		}
	}

	// Sort resistances ascending (nearest first)
	for i := 0; i < len(resistances); i++ {
		for j := i + 1; j < len(resistances); j++ {
			if resistances[j] < resistances[i] {
				resistances[i], resistances[j] = resistances[j], resistances[i]
			}
		}
	}

	// Assign support levels
	if len(supports) > 0 {
		sr.Support1 = math.Round(supports[0]*100) / 100
	}
	if len(supports) > 1 {
		sr.Support2 = math.Round(supports[1]*100) / 100
	}
	if len(supports) > 2 {
		sr.Support3 = math.Round(supports[2]*100) / 100
	}

	// Assign resistance levels
	if len(resistances) > 0 {
		sr.Resistance1 = math.Round(resistances[0]*100) / 100
	}
	if len(resistances) > 1 {
		sr.Resistance2 = math.Round(resistances[1]*100) / 100
	}
	if len(resistances) > 2 {
		sr.Resistance3 = math.Round(resistances[2]*100) / 100
	}

	// Determine nearest S/R level
	nearestSupport := 0.0
	nearestResistance := 0.0
	if len(supports) > 0 {
		nearestSupport = supports[0]
	}
	if len(resistances) > 0 {
		nearestResistance = resistances[0]
	}

	if nearestSupport > 0 && nearestResistance > 0 {
		supportDist := ((price - nearestSupport) / price) * 100
		resistanceDist := ((nearestResistance - price) / price) * 100

		if supportDist < resistanceDist {
			sr.NearestLevel = math.Round(nearestSupport*100) / 100
			sr.NearestType = "支撑"
			sr.DistancePct = math.Round(supportDist*100) / 100
			sr.PricePosition = "上方"
		} else {
			sr.NearestLevel = math.Round(nearestResistance*100) / 100
			sr.NearestType = "阻力"
			sr.DistancePct = math.Round(resistanceDist*100) / 100
			sr.PricePosition = "下方"
		}
	} else if nearestSupport > 0 {
		sr.NearestLevel = math.Round(nearestSupport*100) / 100
		sr.NearestType = "支撑"
		sr.DistancePct = math.Round(((price-nearestSupport)/price)*100*100) / 100
		sr.PricePosition = "上方"
	} else if nearestResistance > 0 {
		sr.NearestLevel = math.Round(nearestResistance*100) / 100
		sr.NearestType = "阻力"
		sr.DistancePct = math.Round(((nearestResistance-price)/price)*100*100) / 100
		sr.PricePosition = "下方"
	}

	return sr
}

// buildStageDescription creates a human-readable trend description.
func (ta *TrendAnalyzer) buildStageDescription(direction, stage string, score float64) string {
	if direction == "震荡" {
		return "当前处于震荡整理阶段，趋势不明朗"
	}

	strengthDesc := "中等"
	if score >= 75 {
		strengthDesc = "强势"
	} else if score <= 35 {
		strengthDesc = "弱势"
	}

	switch stage {
	case "早期":
		return direction + "趋势早期，" + strengthDesc + "，可关注趋势确认信号"
	case "中期":
		return direction + "趋势中期，" + strengthDesc + "，趋势延续概率较高"
	case "末期":
		return direction + "趋势末期，" + strengthDesc + "，注意趋势反转风险"
	default:
		return direction + "趋势，" + strengthDesc
	}
}

// buildVerdict creates the final trend verdict.
func (ta *TrendAnalyzer) buildVerdict(
	stage models.TrendStage,
	sr models.SupportResistance,
	price float64,
) string {
	if stage.Direction == "震荡" {
		return "震荡整理，建议观望等待方向明确"
	}

	verdict := stage.Direction + "趋势" + stage.Stage

	if stage.Stage == "早期" && stage.Direction == "上升" {
		verdict += "，可关注突破确认，目标阻力位" + formatPrice(sr.Resistance1)
	} else if stage.Stage == "中期" && stage.Direction == "上升" {
		verdict += "，趋势延续，止损参考支撑位" + formatPrice(sr.Support1)
	} else if stage.Stage == "末期" && stage.Direction == "上升" {
		verdict += "，注意追高风险，关注阻力位" + formatPrice(sr.Resistance1)
	} else if stage.Stage == "早期" && stage.Direction == "下降" {
		verdict += "，注意风险控制，支撑位" + formatPrice(sr.Support1)
	} else if stage.Stage == "中期" && stage.Direction == "下降" {
		verdict += "，建议观望，等待企稳信号"
	} else if stage.Stage == "末期" && stage.Direction == "下降" {
		verdict += "，关注超跌反弹机会"
	}

	return verdict
}

func formatPrice(price float64) string {
	if price <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.2f", price)
}
