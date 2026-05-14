package services

import (
	"fmt"
	"math"
	"strings"
	"time"

	"alphapulse/internal/models"
)

// ---- Technical Indicators ----

func MovingAverage(values []float64, window int) float64 {
	if len(values) < window || window <= 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values[len(values)-window:] {
		sum += v
	}
	return sum / float64(window)
}

func CalculateMACD(closes []float64) models.MACDResult {
	if len(closes) == 0 {
		return models.MACDResult{Signal: "数据不足", HistTrend: "数据不足", HistLast3: []float64{0, 0, 0}}
	}
	ema12 := closes[0]
	ema26 := closes[0]
	dea := 0.0
	prevDIF := 0.0
	prevDEA := 0.0
	dif := 0.0
	var histList []float64

	for _, close := range closes {
		ema12 = ema12*11.0/13.0 + close*2.0/13.0
		ema26 = ema26*25.0/27.0 + close*2.0/27.0
		prevDIF = dif
		prevDEA = dea
		dif = ema12 - ema26
		dea = dea*8.0/10.0 + dif*2.0/10.0
		histList = append(histList, (dif-dea)*2)
	}

	hist := 0.0
	if len(histList) > 0 {
		hist = histList[len(histList)-1]
	}

	signal := "数据不足"
	if len(closes) >= 35 {
		if prevDIF <= prevDEA && dif > dea {
			signal = "金叉"
		} else if prevDIF >= prevDEA && dif < dea {
			signal = "死叉"
		} else if dif > dea {
			signal = "多头"
		} else {
			signal = "空头"
		}
	}

	last3 := make([]float64, 3)
	if len(histList) >= 3 {
		last3[0] = round2(histList[len(histList)-3])
		last3[1] = round2(histList[len(histList)-2])
		last3[2] = round2(histList[len(histList)-1])
	} else {
		for i, v := range histList {
			last3[3-len(histList)+i] = round2(v)
		}
	}

	histTrend := "震荡"
	if last3[0] < last3[1] && last3[1] < last3[2] {
		histTrend = "连续增强"
	} else if last3[0] > last3[1] && last3[1] > last3[2] {
		histTrend = "连续减弱"
	}

	return models.MACDResult{
		DIF:       round2(dif),
		DEA:       round2(dea),
		Hist:      round2(hist),
		Signal:    signal,
		HistTrend: histTrend,
		HistLast3: last3,
	}
}

func CalculateRSI(closes []float64, period int) float64 {
	if len(closes) <= period || period <= 0 {
		return 0
	}
	changes := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		changes = append(changes, closes[i]-closes[i-1])
	}
	recent := changes[len(changes)-period:]
	gains := 0.0
	losses := 0.0
	for _, c := range recent {
		if c > 0 {
			gains += c
		} else {
			losses += -c
		}
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	if avgLoss == 0 {
		if avgGain > 0 {
			return 100
		}
		return 50
	}
	rs := avgGain / avgLoss
	return round2(100 - 100/(1+rs))
}

func CalculateBollinger(closes []float64, period int) models.BollingerResult {
	if len(closes) < period || period <= 0 {
		return models.BollingerResult{}
	}
	recent := closes[len(closes)-period:]
	mid := 0.0
	for _, v := range recent {
		mid += v
	}
	mid /= float64(period)
	variance := 0.0
	for _, v := range recent {
		d := v - mid
		variance += d * d
	}
	variance /= float64(period)
	std := math.Sqrt(variance)
	upper := mid + 2*std
	lower := mid - 2*std
	bandwidth := 0.0
	if mid != 0 {
		bandwidth = (upper - lower) / mid * 100
	}
	return models.BollingerResult{
		Upper:     round2(upper),
		Mid:       round2(mid),
		Lower:     round2(lower),
		Bandwidth: round2(bandwidth),
	}
}

func CalculateKDJ(closes, highs, lows []float64, period int) models.KDJResult {
	if len(closes) < period || period <= 0 {
		return models.KDJResult{Signal: "数据不足"}
	}
	k := 50.0
	d := 50.0
	prevK := k
	prevD := d
	for i := range closes {
		start := i - period + 1
		if start < 0 {
			start = 0
		}
		periodHigh := math.Inf(-1)
		periodLow := math.Inf(1)
		for j := start; j <= i; j++ {
			if highs[j] > periodHigh {
				periodHigh = highs[j]
			}
			if lows[j] < periodLow {
				periodLow = lows[j]
			}
		}
		denom := periodHigh - periodLow
		rsv := 50.0
		if denom != 0 {
			rsv = (closes[i] - periodLow) / denom * 100
		}
		prevK = k
		prevD = d
		k = (2.0/3.0)*k + (1.0/3.0)*rsv
		d = (2.0/3.0)*d + (1.0/3.0)*k
	}
	j := 3*k - 2*d
	signal := "中性"
	if prevK <= prevD && k > d {
		signal = "金叉"
	} else if prevK >= prevD && k < d {
		signal = "死叉"
	} else if j >= 100 {
		signal = "高位钝化"
	} else if j <= 0 {
		signal = "低位钝化"
	}
	return models.KDJResult{
		K:      round2(k),
		D:      round2(d),
		J:      round2(j),
		Signal: signal,
	}
}

func CalculateOBV(closes, volumes []float64) models.OBVResult {
	if len(closes) < 2 || len(volumes) < 2 {
		return models.OBVResult{Trend: "数据不足"}
	}
	obvList := []float64{0}
	for i := 1; i < len(closes); i++ {
		prev := obvList[len(obvList)-1]
		if closes[i] > closes[i-1] {
			obvList = append(obvList, prev+volumes[i])
		} else if closes[i] < closes[i-1] {
			obvList = append(obvList, prev-volumes[i])
		} else {
			obvList = append(obvList, prev)
		}
	}
	recent := make([]float64, 0, 5)
	if len(obvList) >= 5 {
		for _, v := range obvList[len(obvList)-5:] {
			recent = append(recent, round2(v))
		}
	} else {
		for _, v := range obvList {
			recent = append(recent, round2(v))
		}
	}
	trend := "走平"
	if len(recent) >= 2 {
		if recent[len(recent)-1] > recent[0] {
			trend = "上升"
		} else if recent[len(recent)-1] < recent[0] {
			trend = "下降"
		}
	}
	return models.OBVResult{Recent5D: recent, Trend: trend}
}

// ---- 8 Analysis Dimensions ----

func AnalyzeOrderFlow(quote models.Quote) models.OrderFlowAnalysis {
	outer := quote.OuterVol
	inner := quote.InnerVol
	total := outer + inner
	ratio := 0.0
	if total > 0 {
		ratio = outer / total * 100
	}
	direction := "数据不足"
	verdict := "内外盘数据不足，暂不判断"
	if ratio > 55 {
		direction = "买方强势"
		verdict = "外盘明显高于内盘，买方力量占优"
	} else if ratio >= 50 {
		direction = "买方略强"
		verdict = "外盘占比>50%，买方力量略占优"
	} else if total > 0 && ratio < 45 {
		direction = "卖方强势"
		verdict = "内盘明显高于外盘，卖方压力较大"
	} else if total > 0 {
		direction = "卖方略强"
		verdict = "内盘略高于外盘，短线抛压略大"
	}
	return models.OrderFlowAnalysis{
		OuterVol:     outer,
		InnerVol:     inner,
		OuterRatio:   round2(ratio),
		NetDirection: direction,
		Verdict:      verdict,
	}
}

func AnalyzeVolumePrice(quote models.Quote, klines []models.KlinePoint, turnoverRate float64) models.VolumePriceAnalysis {
	todayVolume := quote.Volume
	recent := make([]models.KlinePoint, 0, 5)
	if len(klines) >= 6 {
		recent = klines[len(klines)-6 : len(klines)-1]
	} else if len(klines) > 1 {
		recent = klines[:len(klines)-1]
	}
	avgVolume := 0.0
	if len(recent) > 0 {
		sum := 0.0
		for _, k := range recent {
			sum += k.Volume
		}
		avgVolume = sum / float64(len(recent))
	}
	volumeRatio := 0.0
	if avgVolume > 0 {
		volumeRatio = todayVolume / avgVolume
	}
	changePct := quote.ChangePercent
	turnover := turnoverRate

	turnoverLevel := "低迷"
	if turnover >= 7 {
		turnoverLevel = "过热"
	} else if turnover >= 3 {
		turnoverLevel = "活跃"
	} else if turnover >= 1 {
		turnoverLevel = "正常"
	}

	harmony := "量价平稳"
	verdict := "量价变化不极端，走势偏平稳"
	if changePct > 0 && volumeRatio >= 1.1 {
		harmony = "量价齐升"
		verdict = "上涨放量，量价配合健康"
	} else if changePct > 0 && volumeRatio < 0.8 {
		harmony = "缩量上涨"
		verdict = "上涨但量能不足，持续性需要观察"
	} else if changePct < 0 && volumeRatio >= 1.1 {
		harmony = "放量下跌"
		verdict = "下跌放量，短线承压明显"
	} else if changePct < 0 && volumeRatio < 0.8 {
		harmony = "缩量下跌"
		verdict = "下跌缩量，抛压暂未扩大"
	}

	return models.VolumePriceAnalysis{
		TodayChangePct:     round2(changePct),
		TodayVolume:        todayVolume,
		AvgVolume5D:        round2(avgVolume),
		VolumeRatio:        round2(volumeRatio),
		Turnover:           round2(turnover),
		TurnoverLevel:      turnoverLevel,
		PriceVolumeHarmony: harmony,
		Verdict:            verdict,
	}
}

func levelPE(pe float64) string {
	if pe <= 0 {
		return "亏损或无效"
	}
	if pe < 15 {
		return "偏低"
	}
	if pe < 30 {
		return "合理"
	}
	if pe < 60 {
		return "偏高"
	}
	return "很高"
}

func levelPB(pb float64) string {
	if pb <= 0 {
		return "无效"
	}
	if pb < 1 {
		return "偏低"
	}
	if pb < 3 {
		return "合理"
	}
	if pb < 6 {
		return "偏高"
	}
	return "很高"
}

func AnalyzeValuation(quote models.Quote) models.ValuationAnalysis {
	enhanced := AnalyzeValuationEnhanced(quote)
	return models.ValuationAnalysis{
		PE:      enhanced.PE,
		PELevel: enhanced.PELevel,
		PB:      enhanced.PB,
		PBLevel: enhanced.PBLevel,
		TotalMV: enhanced.TotalMV,
		MVLevel: enhanced.MVLevel,
		Verdict: enhanced.Verdict,
	}
}

func AnalyzeVolatility(quote models.Quote) models.VolatilityAnalysis {
	amplitude := quote.Amplitude
	price := quote.Price
	limitUp := quote.LimitUp
	limitDown := quote.LimitDown

	level := "正常波动"
	if amplitude < 3 {
		level = "低波动"
	} else if amplitude >= 6 {
		level = "高波动"
	}

	distanceUp := 0.0
	if price > 0 && limitUp > 0 {
		distanceUp = (limitUp - price) / price * 100
	}
	distanceDown := 0.0
	if price > 0 && limitDown > 0 {
		distanceDown = (price - limitDown) / price * 100
	}

	verdict := "波动处于可控区间"
	if level == "高波动" {
		verdict = "振幅偏大，短线波动风险较高"
	} else if distanceUp < 2 && distanceUp > 0 {
		verdict = "距离涨停较近，短线情绪较强"
	} else if distanceDown < 2 && distanceDown > 0 {
		verdict = "距离跌停较近，短线风险较高"
	}

	return models.VolatilityAnalysis{
		Amplitude:           round2(amplitude),
		AmplitudeLevel:      level,
		DistanceToLimitUp:   round2(distanceUp),
		DistanceToLimitDown: round2(distanceDown),
		Verdict:             verdict,
	}
}

func AnalyzeMoneyFlow(flows []models.MoneyFlowDay) models.MoneyFlowAnalysis {
	if len(flows) == 0 {
		return models.MoneyFlowAnalysis{
			TodayMainDirection:       "数据不足",
			InstitutionVsHotMoney:    "数据不足",
			MainConsecutiveDirection: "数据不足",
			RetailBehavior:           "数据不足",
			Verdict:                  "资金流向数据不足，暂不判断",
		}
	}
	today := flows[len(flows)-1]
	main := today.MainNet
	huge := today.HugeNet
	big := today.BigNet
	small := today.SmallNet

	direction := "持平"
	if main > 0 {
		direction = "流入"
	} else if main < 0 {
		direction = "流出"
	}

	// 5-day trend analysis
	trend5d := "平稳"
	netSum5d := 0.0
	if len(flows) >= 5 {
		recent5 := flows[len(flows)-5:]
		for _, f := range recent5 {
			netSum5d += f.MainNet
		}
		if netSum5d > 0 {
			trend5d = "净流入"
		} else if netSum5d < 0 {
			trend5d = "净流出"
		}
	}

	sign := 0
	if main > 0 {
		sign = 1
	} else if main < 0 {
		sign = -1
	}
	consecutive := 0
	if sign != 0 {
		for i := len(flows) - 1; i >= 0; i-- {
			v := flows[i].MainNet
			if (v > 0 && sign > 0) || (v < 0 && sign < 0) {
				consecutive++
			} else {
				break
			}
		}
	}

	dominant := "大单主导"
	if math.Abs(huge) >= math.Abs(big) {
		dominant = "机构主导"
	}
	retail := "散户持平"
	if small < 0 {
		retail = "散户流出"
	} else if small > 0 {
		retail = "散户流入"
	}

	// Enhanced verdict
	verdict := "今日主力资金持平，方向不明"
	if main > 0 && huge > 0 {
		verdict = "今日主力净流入，超大单同步流入，机构进场迹象较强"
		if consecutive >= 3 {
			verdict += fmt.Sprintf("，连续%d日流入", consecutive)
		}
	} else if main > 0 {
		verdict = "今日主力净流入，资金面偏积极"
		if consecutive >= 3 {
			verdict += fmt.Sprintf("，连续%d日流入", consecutive)
		}
	} else if main < 0 && huge < 0 {
		verdict = "今日主力净流出，超大单同步流出，机构减仓迹象明显"
		if consecutive >= 3 {
			verdict += fmt.Sprintf("，连续%d日流出", consecutive)
		}
	} else if main < 0 {
		verdict = "今日主力净流出，资金面承压"
		if consecutive >= 3 {
			verdict += fmt.Sprintf("，连续%d日流出", consecutive)
		}
	}

	// Add 5-day trend context
	if trend5d == "净流入" && direction == "流出" {
		verdict += "；5日资金仍为净流入趋势，短期回调"
	} else if trend5d == "净流出" && direction == "流入" {
		verdict += "；5日资金仍为净流出趋势，短期反弹"
	}

	return models.MoneyFlowAnalysis{
		TodayMainNet:             round2(main),
		TodayMainDirection:       direction,
		TodayHugeNet:             round2(huge),
		TodayBigNet:              round2(big),
		InstitutionVsHotMoney:    dominant,
		MainConsecutiveDays:      consecutive,
		MainConsecutiveDirection: direction,
		RetailBehavior:           retail,
		Verdict:                  verdict,
	}
}

func AnalyzeTechnical(klines []models.KlinePoint) models.TechnicalAnalysis {
	var closes, highs, lows, volumes []float64
	for _, k := range klines {
		if k.Close > 0 {
			closes = append(closes, k.Close)
			highs = append(highs, k.High)
			lows = append(lows, k.Low)
			volumes = append(volumes, k.Volume)
		}
	}
	latest := 0.0
	if len(closes) > 0 {
		latest = closes[len(closes)-1]
	}
	ma5 := MovingAverage(closes, 5)
	ma10 := MovingAverage(closes, 10)
	ma20 := MovingAverage(closes, 20)
	ma60 := MovingAverage(closes, 60)

	arrangement := "纠缠/震荡"
	if ma5 > 0 && ma10 > 0 && ma20 > 0 && ma60 > 0 {
		if ma5 > ma10 && ma10 > ma20 && ma20 > ma60 {
			arrangement = "多头排列"
		} else if ma5 < ma10 && ma10 < ma20 && ma20 < ma60 {
			arrangement = "空头排列"
		}
	} else if ma5 > 0 && ma10 > 0 && ma20 > 0 {
		if ma5 > ma10 && ma10 > ma20 {
			arrangement = "短多排列"
		} else if ma5 < ma10 && ma10 < ma20 {
			arrangement = "短空排列"
		}
	}

	// Use new indicators engine for enhanced calculations
	ind := ComputeIndicators(klines)

	// MACD — prefer new engine, fallback to old
	macd := CalculateMACD(closes)
	if ind.MACD.Signal != "" {
		switch ind.MACD.Signal {
		case "golden_cross":
			macd.Signal = "金叉"
		case "death_cross":
			macd.Signal = "死叉"
		default:
			if ind.MACD.Histogram > 0 {
				macd.Signal = "多头"
			} else if ind.MACD.Histogram < 0 {
				macd.Signal = "空头"
			}
		}
		macd.DIF = ind.MACD.DIF
		macd.DEA = ind.MACD.DEA
		macd.Hist = ind.MACD.Histogram
	}

	// KDJ — use new engine
	kdj := CalculateKDJ(closes, highs, lows, 9)
	if ind.KDJ.Signal != "" {
		switch ind.KDJ.Signal {
		case "overbought":
			kdj.Signal = "超买"
		case "oversold":
			kdj.Signal = "超卖"
		}
		kdj.K = ind.KDJ.K
		kdj.D = ind.KDJ.D
		kdj.J = ind.KDJ.J
	}

	// RSI — use new engine (multi-period)
	rsi := CalculateRSI(closes, 14)
	rsiLevel := "数据不足"
	if ind.RSI.RSI12 > 0 {
		rsi = ind.RSI.RSI12
		if rsi >= 80 {
			rsiLevel = "超买"
		} else if rsi >= 60 {
			rsiLevel = "中性偏强"
		} else if rsi >= 40 {
			rsiLevel = "中性"
		} else if rsi >= 20 {
			rsiLevel = "中性偏弱"
		} else {
			rsiLevel = "超卖"
		}
	} else if rsi > 0 {
		if rsi >= 80 {
			rsiLevel = "超买"
		} else if rsi >= 60 {
			rsiLevel = "中性偏强"
		} else if rsi >= 40 {
			rsiLevel = "中性"
		} else {
			rsiLevel = "中性偏弱"
		}
	}

	// Boll — use new engine
	boll := CalculateBollinger(closes, 20)
	// OBV
	obv := CalculateOBV(closes, volumes)
	bollPosition := "数据不足"
	if ind.Boll.Upper > 0 {
		boll.Upper = ind.Boll.Upper
		boll.Mid = ind.Boll.Middle
		boll.Lower = ind.Boll.Lower
		boll.Bandwidth = ind.Boll.Width
		switch ind.Boll.Signal {
		case "above_upper":
			bollPosition = "上轨上方"
		case "below_lower":
			bollPosition = "下轨下方"
		default:
			if latest >= ind.Boll.Middle {
				bollPosition = "中轨上方"
			} else {
				bollPosition = "中轨下方"
			}
		}
	} else if latest > 0 {
		if boll.Upper > 0 && latest > boll.Upper {
			bollPosition = "上轨上方"
		} else if boll.Mid > 0 && latest >= boll.Mid {
			bollPosition = "中轨上方"
		} else if boll.Lower > 0 && latest < boll.Lower {
			bollPosition = "下轨下方"
		} else if boll.Lower > 0 {
			bollPosition = "中轨下方"
		}
	}

	// Build verdict
	var parts []string
	if arrangement == "多头排列" || arrangement == "短多排列" {
		parts = append(parts, "均线"+arrangement)
	} else if arrangement == "空头排列" || arrangement == "短空排列" {
		parts = append(parts, "均线"+arrangement)
	}
	if macd.Signal == "金叉" || macd.Signal == "多头" {
		parts = append(parts, "MACD"+macd.Signal)
	} else if macd.Signal == "死叉" || macd.Signal == "空头" {
		parts = append(parts, "MACD"+macd.Signal)
	}
	if kdj.Signal == "超买" {
		parts = append(parts, "KDJ超买")
	} else if kdj.Signal == "超卖" {
		parts = append(parts, "KDJ超卖")
	} else if kdj.Signal == "金叉" || kdj.Signal == "死叉" {
		parts = append(parts, "KDJ"+kdj.Signal)
	}
	if rsiLevel != "数据不足" {
		parts = append(parts, "RSI"+rsiLevel)
	}
	if bollPosition != "数据不足" {
		parts = append(parts, "布林"+bollPosition)
	}

	verdict := "技术指标数据不足"
	if len(parts) > 0 {
		verdict = strings.Join(parts, "，")
		if arrangement == "多头排列" {
			verdict += "，技术面偏多"
		} else if arrangement == "空头排列" {
			verdict += "，技术面偏空"
		}
	}

	// Weekly multi-period analysis
	weeklyKlines := AggregateToWeekly(klines)
	var weeklyMACD, weeklyMA, weeklyRSILevel, periodAlign string
	var weeklyRSI float64
	if len(weeklyKlines) >= 10 {
		var wCloses []float64
		for _, wk := range weeklyKlines {
			if wk.Close > 0 {
				wCloses = append(wCloses, wk.Close)
			}
		}
		if len(wCloses) >= 10 {
			wMACD := CalculateMACD(wCloses)
			weeklyRSI = CalculateRSI(wCloses, 14)

			// Weekly MACD signal
			if wMACD.DIF > wMACD.DEA && wMACD.Hist > 0 {
				weeklyMACD = "多头"
			} else if wMACD.DIF < wMACD.DEA && wMACD.Hist < 0 {
				weeklyMACD = "空头"
			} else {
				weeklyMACD = "中性"
			}

			// Weekly RSI level
			switch {
			case weeklyRSI > 70:
				weeklyRSILevel = "超买"
			case weeklyRSI > 55:
				weeklyRSILevel = "中性偏强"
			case weeklyRSI > 45:
				weeklyRSILevel = "中性"
			case weeklyRSI > 30:
				weeklyRSILevel = "中性偏弱"
			default:
				weeklyRSILevel = "超卖"
			}

			// Weekly MA trend
			wMA5 := MovingAverage(wCloses, 5)
			wMA10 := MovingAverage(wCloses, 10)
			if wMA5 > wMA10 {
				weeklyMA = "周线多头"
			} else if wMA5 < wMA10 {
				weeklyMA = "周线空头"
			} else {
				weeklyMA = "周线震荡"
			}

			// Period alignment (日周共振判断)
			dailyBullish := macd.Signal == "金叉" || macd.Signal == "多头"
			weeklyBullish := weeklyMACD == "多头"
			dailyBearish := macd.Signal == "死叉" || macd.Signal == "空头"
			weeklyBearish := weeklyMACD == "空头"

			if dailyBullish && weeklyBullish {
				periodAlign = "日周共振看多"
			} else if dailyBearish && weeklyBearish {
				periodAlign = "日周共振看空"
			} else if dailyBullish && weeklyBearish {
				periodAlign = "日强周弱"
			} else if dailyBearish && weeklyBullish {
				periodAlign = "日弱周强"
			} else {
				periodAlign = "日周方向一致"
			}
		}
	}

	return models.TechnicalAnalysis{
		MA5:            round2(ma5),
		MA10:           round2(ma10),
		MA20:           round2(ma20),
		MA60:           round2(ma60),
		MAArrangement:  arrangement,
		MACD_DIF:       macd.DIF,
		MACD_DEA:       macd.DEA,
		MACD_Hist:      macd.Hist,
		MACD_Signal:    macd.Signal,
		MACD_HistLast3: macd.HistLast3,
		MACD_HistTrend: macd.HistTrend,
		KDJ_K:          kdj.K,
		KDJ_D:          kdj.D,
		KDJ_J:          kdj.J,
		KDJ_Signal:     kdj.Signal,
		OBV_5D:         obv.Recent5D,
		OBV_Trend:      obv.Trend,
		RSI_14:         rsi,
		RSI_Level:      rsiLevel,
		BollUpper:      boll.Upper,
		BollMid:        boll.Mid,
		BollLower:      boll.Lower,
		BollBandwidth:  boll.Bandwidth,
		BollPosition:   bollPosition,
		// Weekly multi-period confirmation
		WeeklyMACD:     weeklyMACD,
		WeeklyRSI:      round2(weeklyRSI),
		WeeklyRSILevel: weeklyRSILevel,
		WeeklyMA:       weeklyMA,
		PeriodAlign:    periodAlign,
		Verdict:        verdict,
	}
}

// AggregateToWeekly converts daily K-lines to weekly K-lines.
func AggregateToWeekly(klines []models.KlinePoint) []models.KlinePoint {
	if len(klines) == 0 {
		return nil
	}
	var weekly []models.KlinePoint
	var weekOpen, weekHigh, weekLow, weekVol float64
	var weekClose float64
	var weekDate string
	weekStarted := false

	for _, k := range klines {
		if k.Date == "" || k.Close <= 0 {
			continue
		}
		// Parse date to determine week (YYYY-MM-DD format)
		yearWeek := k.Date[:4] // simplified: group by year prefix for now
		if len(k.Date) >= 8 {
			// Use year + week number approximation
			t, err := time.Parse("2006-01-02", k.Date)
			if err == nil {
				year, week := t.ISOWeek()
				yearWeek = fmt.Sprintf("%d-W%02d", year, week)
			}
		}

		if !weekStarted || yearWeek != weekDate {
			// Save previous week
			if weekStarted {
				weekly = append(weekly, models.KlinePoint{
					Date:   weekDate,
					Open:   weekOpen,
					High:   weekHigh,
					Low:    weekLow,
					Close:  weekClose,
					Volume: weekVol,
				})
			}
			// Start new week
			weekDate = yearWeek
			weekOpen = k.Open
			weekHigh = k.High
			weekLow = k.Low
			weekClose = k.Close
			weekVol = k.Volume
			weekStarted = true
		} else {
			// Accumulate within the week
			if k.High > weekHigh {
				weekHigh = k.High
			}
			if k.Low < weekLow {
				weekLow = k.Low
			}
			weekClose = k.Close
			weekVol += k.Volume
		}
	}
	// Save last week
	if weekStarted {
		weekly = append(weekly, models.KlinePoint{
			Date:   weekDate,
			Open:   weekOpen,
			High:   weekHigh,
			Low:    weekLow,
			Close:  weekClose,
			Volume: weekVol,
		})
	}
	return weekly
}

func AnalyzeSector(quote models.Quote, sectors []string, sectorPerf *SectorPerformance) models.SectorAnalysis {
	primary := ""
	if len(sectors) > 0 {
		primary = sectors[0]
	}
	totalMV := quote.TotalMV
	name := quote.Name
	isLeader := totalMV >= 500 || strings.HasPrefix(name, "中国") || strings.Contains(name, "龙头")

	verdict := "板块数据不足，暂不判断"
	result := models.SectorAnalysis{
		Sectors:        sectors,
		PrimarySector:  primary,
		IsSectorLeader: isLeader,
	}

	if sectorPerf != nil && sectorPerf.Industry != "" {
		result.SectorPctChg5D = sectorPerf.AvgPctChg5D
		result.StockPctChg5D = sectorPerf.StockPctChg5D
		result.RelStrength = sectorPerf.RelStrength
		result.RelStrengthTag = sectorPerf.Trend

		if isLeader && sectorPerf.Trend == "强于板块" {
			verdict = "所属" + primary + "板块龙头，强于板块表现"
		} else if isLeader && sectorPerf.Trend == "弱于板块" {
			verdict = "所属" + primary + "板块龙头，但近期弱于板块"
		} else if sectorPerf.Trend == "强于板块" {
			verdict = "所属" + primary + "板块，近期强于板块平均"
		} else if sectorPerf.Trend == "弱于板块" {
			verdict = "所属" + primary + "板块，近期弱于板块平均"
		} else if primary != "" {
			verdict = "所属" + primary + "板块，表现与板块同步"
		}
	} else if primary != "" && isLeader {
		verdict = "所属" + primary + "板块，具备较强行业地位"
	} else if primary != "" {
		verdict = "所属" + primary + "板块，需结合板块强弱观察"
	}

	result.Verdict = verdict
	return result
}

var positiveWords = []string{
	// 财务表现 (financial performance)
	"预增", "增长", "盈利", "超预期", "翻倍", "大增", "扭亏", "减亏", "高增长",
	"业绩预增", "净利增", "营收增", "毛利提升", "订单增长", "产能释放", "创新高",
	// 技术面/动量 (technical/momentum)
	"利好", "突破", "新高", "涨停", "大涨", "强势", "反弹", "放量", "金叉",
	"多头", "主力加仓", "机构买入", "北向买入", "融资买入", "底部放量",
	// 公司行为 (corporate actions)
	"回购", "增持", "分红", "送转", "高送转", "股权激励", "战略合作", "中标",
	"签约", "获批", "专利", "技术突破",
	// 行业/政策 (industry/policy)
	"龙头", "创新药", "新能源", "人工智能", "芯片", "国产替代", "政策利好",
	"减税降费", "扩产", "并购", "重组",
}

var negativeWords = []string{
	// 财务恶化 (financial deterioration)
	"预减", "下降", "亏损", "暴跌", "大亏", "业绩下滑", "净利降", "营收降",
	"商誉减值", "计提", "坏账", "现金流恶化", "负债率上升", "毛利率下降",
	// 技术面/动量 (technical/momentum)
	"利空", "跌停", "下跌", "破位", "死叉", "空头", "缩量", "放量下跌",
	"主力出逃", "机构卖出", "北向卖出", "融资卖出", "连续跌停",
	// 公司风险 (corporate risks)
	"风险", "违规", "处罚", "立案", "调查", "退市", "ST", "*ST", "暂停上市",
	"减持", "质押", "爆雷", "造假", "实控人", "被强制", "冻结",
	// 行业/政策 (industry/policy)
	"制裁", "限制", "环保处罚", "产能过剩", "需求萎缩", "行业寒冬",
	"监管趋严", "反垄断", "关税",
}

func AnalyzeSentiment(news []models.NewsItem, announcements []models.Announcement) models.SentimentAnalysis {
	positive := 0
	negative := 0
	var keyEvents []string
	seen := make(map[string]bool)

	type titled interface{ GetTitle() string }

	for _, item := range news {
		title := item.Title
		posHit := containsAny(title, positiveWords)
		negHit := containsAny(title, negativeWords)
		if posHit {
			positive++
		}
		if negHit {
			negative++
		}
		if (posHit || negHit) && title != "" && !seen[title] {
			keyEvents = append(keyEvents, title)
			seen[title] = true
		}
	}
	for _, item := range announcements {
		title := item.Title
		posHit := containsAny(title, positiveWords)
		negHit := containsAny(title, negativeWords)
		if posHit {
			positive++
		}
		if negHit {
			negative++
		}
		if (posHit || negHit) && title != "" && !seen[title] {
			keyEvents = append(keyEvents, title)
			seen[title] = true
		}
	}

	total := len(news) + len(announcements)
	score := 0.0
	if total > 0 {
		score = float64(positive-negative) / float64(total)
	}
	if score > 1 {
		score = 1
	} else if score < -1 {
		score = -1
	}

	label := "中性"
	verdict := "消息面整体中性"
	if score > 0.2 {
		label = "正面"
		verdict = "正面关键词较多，消息面偏正面"
	} else if score < -0.2 {
		label = "负面"
		verdict = "负面关键词较多，消息面偏负面"
	}

	if len(keyEvents) > 5 {
		keyEvents = keyEvents[:5]
	}

	return models.SentimentAnalysis{
		NewsCount:         len(news),
		AnnouncementCount: len(announcements),
		KeyEvents:         keyEvents,
		SentimentScore:    round2(score),
		SentimentLabel:    label,
		Verdict:           verdict,
	}
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// ==================== P0 Analysis Functions ====================

// AnalyzeFundamentals evaluates financial health from financial statement data.
func AnalyzeFundamentals(financials []FinancialData) models.FundamentalsAnalysis {
	a := models.FundamentalsAnalysis{}
	if len(financials) == 0 {
		a.Verdict = "暂无财务数据"
		return a
	}

	latest := financials[0]
	a.ROE = latest.ROE
	a.GrossMargin = latest.GrossMargin
	a.NetMargin = latest.NetMargin
	a.DebtRatio = latest.DebtRatio
	// EPS not in struct, skip

	// ROE level
	switch {
	case latest.ROE > 15:
		a.ROELevel = "优秀"
	case latest.ROE > 10:
		a.ROELevel = "良好"
	case latest.ROE > 5:
		a.ROELevel = "一般"
	default:
		a.ROELevel = "较差"
	}

	// Gross margin level
	switch {
	case latest.GrossMargin > 40:
		a.GrossMarginLevel = "高毛利"
	case latest.GrossMargin > 25:
		a.GrossMarginLevel = "中毛利"
	default:
		a.GrossMarginLevel = "低毛利"
	}

	// Net margin level
	switch {
	case latest.NetMargin > 15:
		a.NetMarginLevel = "高净利"
	case latest.NetMargin > 8:
		a.NetMarginLevel = "中净利"
	default:
		a.NetMarginLevel = "低净利"
	}

	// Debt ratio level
	switch {
	case latest.DebtRatio < 40:
		a.DebtRatioLevel = "低负债"
	case latest.DebtRatio < 60:
		a.DebtRatioLevel = "中等负债"
	default:
		a.DebtRatioLevel = "高负债"
	}

	// Revenue growth
	a.RevenueGrowth = latest.RevenueYoY
	switch {
	case latest.RevenueYoY > 20:
		a.RevenueGrowthLevel = "高增长"
	case latest.RevenueYoY > 10:
		a.RevenueGrowthLevel = "中增长"
	case latest.RevenueYoY > 0:
		a.RevenueGrowthLevel = "低增长"
	default:
		a.RevenueGrowthLevel = "负增长"
	}

	// Net profit growth
	a.NetProfitGrowth = latest.NetProfitYoY
	switch {
	case latest.NetProfitYoY > 20:
		a.NetProfitGrowthLevel = "高增长"
	case latest.NetProfitYoY > 10:
		a.NetProfitGrowthLevel = "中增长"
	case latest.NetProfitYoY > 0:
		a.NetProfitGrowthLevel = "低增长"
	default:
		a.NetProfitGrowthLevel = "负增长"
	}

	// EPS trend (compare latest 2 periods)
	if len(financials) >= 2 {
		prev := financials[1]
		if latest.EPS > prev.EPS*1.05 {
			a.EPSTrend = "上升"
		} else if latest.EPS < prev.EPS*0.95 {
			a.EPSTrend = "下降"
		} else {
			a.EPSTrend = "平稳"
		}
	} else {
		a.EPSTrend = "数据不足"
	}

	// Overall score (0-100)
	score := 50
	if latest.ROE > 15 { score += 15 } else if latest.ROE > 10 { score += 10 } else if latest.ROE > 5 { score += 5 } else { score -= 10 }
	if latest.GrossMargin > 40 { score += 10 } else if latest.GrossMargin > 25 { score += 5 } else { score -= 5 }
	if latest.NetMargin > 15 { score += 10 } else if latest.NetMargin > 8 { score += 5 } else { score -= 5 }
	if latest.DebtRatio < 40 { score += 5 } else if latest.DebtRatio < 60 { score += 0 } else { score -= 10 }
	if latest.RevenueYoY > 20 { score += 10 } else if latest.RevenueYoY > 10 { score += 5 } else if latest.RevenueYoY < 0 { score -= 10 }
	if latest.NetProfitYoY > 20 { score += 10 } else if latest.NetProfitYoY > 10 { score += 5 } else if latest.NetProfitYoY < 0 { score -= 10 }
	if a.EPSTrend == "上升" { score += 5 } else if a.EPSTrend == "下降" { score -= 5 }
	if score > 100 { score = 100 }
	if score < 0 { score = 0 }
	a.Score = score

	// Verdict
	switch {
	case score >= 75:
		a.Verdict = "基本面优秀，盈利能力强，成长性好"
	case score >= 60:
		a.Verdict = "基本面良好，整体稳健"
	case score >= 40:
		a.Verdict = "基本面一般，有改善空间"
	default:
		a.Verdict = "基本面较弱，需关注风险"
	}
	return a
}

// AnalyzeNorthbound evaluates northbound capital flow signals.
// All monetary inputs are expected in 万元.
func AnalyzeNorthbound(hsgtData []HsgtData, top10Data []HsgtTop10Data) models.NorthboundAnalysis {
	a := models.NorthboundAnalysis{}

	// Market-level northbound flow (values in 万元 from Tushare API)
	if len(hsgtData) > 0 {
		a.LatestNetFlow = hsgtData[0].NorthMoney
		limit := 5
		if len(hsgtData) < limit {
			limit = len(hsgtData)
		}
		sum := 0.0
		for i := 0; i < limit; i++ {
			sum += hsgtData[i].NorthMoney
		}
		a.Trend5D = sum

		// Thresholds in 万元: ±100亿(=±1,000,000万) for significant 5-day flow
		if sum > 1000000 {
			a.FlowDirection = "持续流入"
		} else if sum < -1000000 {
			a.FlowDirection = "持续流出"
		} else if sum > 0 {
			a.FlowDirection = "小幅流入"
		} else if sum < 0 {
			a.FlowDirection = "小幅流出"
		} else {
			a.FlowDirection = "持平"
		}
	} else {
		a.FlowDirection = "无数据"
	}

	// Stock-level northbound activity (values in 万元 after conversion from 元÷10000)
	if len(top10Data) > 0 {
		// After Aug 2024, net_amount is typically null; use total amount as activity indicator
		totalNet := 0.0
		totalAmount := 0.0
		for _, d := range top10Data {
			totalNet += d.NetAmount
			totalAmount += d.Amount
		}
		a.StockNetAmount = totalNet

		hasNetData := totalNet != 0
		if hasNetData {
			// Thresholds in 万元: ±5000万 for significant action
			if totalNet > 5000 {
				a.StockAction = "大幅买入"
				a.Signal = "北向大幅买入"
			} else if totalNet > 1000 {
				a.StockAction = "小幅买入"
				a.Signal = "北向小幅买入"
			} else if totalNet < -5000 {
				a.StockAction = "大幅卖出"
				a.Signal = "北向大幅卖出"
			} else if totalNet < -1000 {
				a.StockAction = "小幅卖出"
				a.Signal = "北向小幅卖出"
			} else {
				a.StockAction = "无明显方向"
				a.Signal = "无明显信号"
			}
		} else if totalAmount > 0 {
			// No net_amount data (post Aug 2024); infer from presence in top10 + total amount
			a.StockNetAmount = totalAmount
			if totalAmount > 500000 {
				a.StockAction = "成交活跃"
				a.Signal = "北向成交活跃"
			} else {
				a.StockAction = "成交一般"
				a.Signal = "北向成交一般"
			}
		} else {
			a.StockAction = "未进入十大成交"
			a.Signal = "无明显信号"
		}
	} else {
		a.StockAction = "未进入十大成交"
		a.Signal = "无明显信号"
	}

	switch a.Signal {
	case "北向大幅买入":
		a.Verdict = "北向资金大幅买入，外资看好"
	case "北向小幅买入":
		a.Verdict = "北向资金小幅流入，偏正面"
	case "北向大幅卖出":
		a.Verdict = "北向资金大幅卖出，需警惕"
	case "北向小幅卖出":
		a.Verdict = "北向资金小幅流出，偏负面"
	case "北向成交活跃":
		a.Verdict = "北向资金成交活跃，关注度高"
	default:
		a.Verdict = "北向资金无明显信号"
	}
	return a
}

// AnalyzeMarginDetail evaluates margin trading signals for a stock.
func AnalyzeMarginDetail(marginData []MarginDetailData) models.MarginAnalysis {
	a := models.MarginAnalysis{}
	if len(marginData) == 0 {
		a.Verdict = "暂无融资融券数据"
		return a
	}

	latest := marginData[0]
	a.LatestMarginBalance = latest.Rzye

	// Trend analysis (compare latest vs 5 days ago)
	if len(marginData) >= 5 {
		old := marginData[4]
		if latest.Rzye > old.Rzye*1.02 {
			a.MarginBalanceTrend = "融资余额增加"
		} else if latest.Rzye < old.Rzye*0.98 {
			a.MarginBalanceTrend = "融资余额减少"
		} else {
			a.MarginBalanceTrend = "融资余额平稳"
		}

		// Margin buying trend
		avgBuy := 0.0
		for i, d := range marginData {
			if i >= 5 { break }
			avgBuy += d.Rzmre
		}
		avgBuy /= float64(len(marginData))
		if latest.Rzmre > avgBuy*1.2 {
			a.MarginBuyingTrend = "融资买入活跃"
		} else if latest.Rzmre < avgBuy*0.8 {
			a.MarginBuyingTrend = "融资买入萎缩"
		} else {
			a.MarginBuyingTrend = "融资买入正常"
		}

		// Short selling trend
		if latest.Rqye > old.Rqye*1.1 {
			a.ShortSellingTrend = "融券余额增加"
		} else if latest.Rqye < old.Rqye*0.9 {
			a.ShortSellingTrend = "融券余额减少"
		} else {
			a.ShortSellingTrend = "融券余额平稳"
		}
	} else {
		a.MarginBalanceTrend = "数据不足"
		a.MarginBuyingTrend = "数据不足"
		a.ShortSellingTrend = "数据不足"
	}

	// Signal
	score := 0.0
	if a.MarginBalanceTrend == "融资余额增加" { score += 0.3 } else if a.MarginBalanceTrend == "融资余额减少" { score -= 0.3 }
	if a.MarginBuyingTrend == "融资买入活跃" { score += 0.2 } else if a.MarginBuyingTrend == "融资买入萎缩" { score -= 0.2 }
	if a.ShortSellingTrend == "融券余额增加" { score -= 0.2 } else if a.ShortSellingTrend == "融券余额减少" { score += 0.1 }
	a.SentimentScore = score

	switch {
	case score > 0.3:
		a.Signal = "融资看多"
		a.Verdict = "融资余额增加、买入活跃，杠杆资金看多"
	case score > 0:
		a.Signal = "融资偏多"
		a.Verdict = "融资端偏正面"
	case score < -0.3:
		a.Signal = "融资看空"
		a.Verdict = "融资余额减少或融券增加，杠杆资金看空"
	case score < 0:
		a.Signal = "融资偏空"
		a.Verdict = "融资端偏负面"
	default:
		a.Signal = "中性"
		a.Verdict = "融资融券无明显信号"
	}
	return a
}

func BuildSummary(a *models.StockAnalysis) models.AnalysisSummary {
	score := 50
	var strengths, risks []string

	// Order flow
	of := a.OrderFlow
	if of.NetDirection == "买方强势" || of.NetDirection == "买方略强" {
		score += 5
		strengths = append(strengths, "买方力量占优")
	} else if of.NetDirection == "卖方强势" || of.NetDirection == "卖方略强" {
		score -= 5
		risks = append(risks, "内盘抛压较大")
	}

	// Volume price
	vp := a.VolumePrice
	if vp.PriceVolumeHarmony == "量价齐升" {
		score += 10
		strengths = append(strengths, "量价齐升")
	} else if vp.PriceVolumeHarmony == "放量下跌" {
		score -= 10
		risks = append(risks, "放量下跌")
	}

	// Valuation
	vl := a.Valuation
	if vl.PELevel == "偏高" || vl.PELevel == "很高" || vl.PBLevel == "偏高" || vl.PBLevel == "很高" {
		score -= 8
		risks = append(risks, "估值偏高")
	} else if (vl.PELevel == "偏低" || vl.PELevel == "合理") && (vl.PBLevel == "偏低" || vl.PBLevel == "合理") {
		score += 5
		strengths = append(strengths, "估值相对合理")
	}

	// Volatility
	vlt := a.Volatility
	if vlt.AmplitudeLevel == "高波动" {
		score -= 5
		risks = append(risks, "振幅偏大")
	}

	// Money flow
	mf := a.MoneyFlow
	if mf.TodayMainDirection == "流入" {
		score += 12
		strengths = append(strengths, "主力流入")
	} else if mf.TodayMainDirection == "流出" {
		score -= 12
		risks = append(risks, "主力流出")
	}

	// Technical
	tech := a.Technical
	if tech.MAArrangement == "多头排列" {
		score += 10
		strengths = append(strengths, "均线多头排列")
	} else if tech.MAArrangement == "空头排列" {
		score -= 10
		risks = append(risks, "均线空头排列")
	}
	if tech.MACD_Signal == "金叉" {
		score += 5
		strengths = append(strengths, "MACD金叉")
	} else if tech.MACD_Signal == "死叉" {
		score -= 5
		risks = append(risks, "MACD死叉")
	}
	// Multi-period alignment
	switch tech.PeriodAlign {
	case "日周共振看多":
		score += 8
		strengths = append(strengths, "日周共振看多")
	case "日周共振看空":
		score -= 8
		risks = append(risks, "日周共振看空")
	case "日强周弱":
		score += 2
	case "日弱周强":
		score -= 2
	}

	// Sector
	sec := a.Sector
	if sec.IsSectorLeader {
		score += 4
		strengths = append(strengths, "行业地位较强")
	}

	// Sentiment
	sent := a.Sentiment
	if sent.SentimentLabel == "正面" {
		score += 7
		if len(sent.KeyEvents) > 0 {
			strengths = append(strengths, sent.KeyEvents[0])
		} else {
			strengths = append(strengths, "消息面正面")
		}
	} else if sent.SentimentLabel == "负面" {
		score -= 7
		if len(sent.KeyEvents) > 0 {
			risks = append(risks, sent.KeyEvents[0])
		} else {
			risks = append(risks, "消息面负面")
		}
	}

	// Fundamentals (P0)
	fund := a.Fundamentals
	if fund.Score >= 75 {
		score += 10
		strengths = append(strengths, "基本面优秀")
	} else if fund.Score >= 60 {
		score += 5
		strengths = append(strengths, "基本面良好")
	} else if fund.Score < 40 && fund.Score > 0 {
		score -= 8
		risks = append(risks, "基本面较弱")
	}
	if fund.RevenueGrowth > 20 {
		strengths = append(strengths, "营收高增长")
	}
	if fund.NetProfitGrowth < 0 && fund.NetProfitGrowth != 0 {
		risks = append(risks, "净利润负增长")
	}

	// Northbound (P0)
	nb := a.Northbound
	if nb.Signal == "北向大幅买入" {
		score += 8
		strengths = append(strengths, "北向资金大幅买入")
	} else if nb.Signal == "北向小幅买入" {
		score += 4
	} else if nb.Signal == "北向大幅卖出" {
		score -= 8
		risks = append(risks, "北向资金大幅卖出")
	} else if nb.Signal == "北向小幅卖出" {
		score -= 4
	}
	if nb.FlowDirection == "持续流入" {
		score += 3
		strengths = append(strengths, "北向资金持续流入")
	} else if nb.FlowDirection == "持续流出" {
		score -= 3
		risks = append(risks, "北向资金持续流出")
	}

	// Margin (P0)
	mg := a.MarginDetail
	if mg.Signal == "融资看多" {
		score += 5
		strengths = append(strengths, "融资余额增加")
	} else if mg.Signal == "融资偏多" {
		score += 2
	} else if mg.Signal == "融资看空" {
		score -= 5
		risks = append(risks, "融资余额减少")
	} else if mg.Signal == "融资偏空" {
		score -= 2
	}

	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	signal := "中性"
	suggestion := "多空信号均衡，建议等待更明确的量价和资金方向"
	if score >= 75 {
		signal = "看多"
		suggestion = "多项指标偏强，可继续关注趋势延续，但避免情绪化追高"
	} else if score >= 60 {
		signal = "偏多"
		suggestion = "短期偏多但仍需控制估值和波动风险，适合持有不宜追高"
	} else if score <= 40 {
		signal = "偏空"
		suggestion = "资金或技术面偏弱，宜等待风险释放和趋势修复"
	}

	return models.AnalysisSummary{
		OverallScore:  score,
		OverallSignal: signal,
		Strengths:     dedupe(strengths)[:min(6, len(dedupe(strengths)))],
		Risks:         dedupe(risks)[:min(6, len(dedupe(risks)))],
		Suggestion:    suggestion,
	}
}

func dedupe(values []string) []string {
	var result []string
	seen := make(map[string]bool)
	for _, v := range values {
		if v != "" && !seen[v] {
			result = append(result, v)
			seen[v] = true
		}
	}
	return result
}

// round2 is defined in eastmoney.go
