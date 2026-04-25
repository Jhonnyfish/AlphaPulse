package services

import (
	"math"
	"strings"

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

func AnalyzeVolumePrice(quote models.Quote, klines []models.KlinePoint) models.VolumePriceAnalysis {
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
	turnover := quote.Turnover

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
	pe := quote.PE
	pb := quote.PB
	totalMV := quote.TotalMV
	peLevel := levelPE(pe)
	pbLevel := levelPB(pb)
	mvLevel := "小盘股"
	if totalMV >= 1000 {
		mvLevel = "大盘股"
	} else if totalMV >= 300 {
		mvLevel = "中大盘股"
	} else if totalMV >= 50 {
		mvLevel = "中小盘股"
	}

	verdict := "估值指标分化，需要结合行业比较"
	if (peLevel == "偏高" || peLevel == "很高") && (pbLevel == "偏高" || pbLevel == "很高") {
		verdict = "PE/PB均偏高，估值不便宜"
	} else if (peLevel == "偏低" || peLevel == "合理") && (pbLevel == "偏低" || pbLevel == "合理") {
		verdict = "PE/PB处于相对合理区间"
	}

	return models.ValuationAnalysis{
		PE:      round2(pe),
		PELevel: peLevel,
		PB:      round2(pb),
		PBLevel: pbLevel,
		TotalMV: round2(totalMV),
		MVLevel: mvLevel,
		Verdict: verdict,
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

	verdict := "今日主力资金持平，方向不明"
	if main > 0 && huge > 0 {
		verdict = "今日主力净流入，超大单同步流入，机构进场迹象较强"
	} else if main > 0 {
		verdict = "今日主力净流入，资金面偏积极"
	} else if main < 0 {
		verdict = "今日主力净流出，资金面承压"
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
	}

	macd := CalculateMACD(closes)
	kdj := CalculateKDJ(closes, highs, lows, 9)
	obv := CalculateOBV(closes, volumes)
	rsi := CalculateRSI(closes, 14)
	rsiLevel := "数据不足"
	if rsi >= 80 {
		rsiLevel = "超买"
	} else if rsi >= 60 {
		rsiLevel = "中性偏强"
	} else if rsi >= 40 {
		rsiLevel = "中性"
	} else if rsi > 0 {
		rsiLevel = "中性偏弱"
	}

	boll := CalculateBollinger(closes, 20)
	bollPosition := "数据不足"
	if latest > 0 {
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

	var parts []string
	if arrangement == "多头排列" {
		parts = append(parts, "均线多头排列")
	} else if arrangement == "空头排列" {
		parts = append(parts, "均线空头排列")
	}
	if macd.Signal == "金叉" || macd.Signal == "多头" {
		parts = append(parts, "MACD"+macd.Signal)
	} else if macd.Signal == "死叉" || macd.Signal == "空头" {
		parts = append(parts, "MACD"+macd.Signal)
	}
	if kdj.Signal == "金叉" || kdj.Signal == "死叉" {
		parts = append(parts, "KDJ"+kdj.Signal)
	}
	if obv.Trend == "上升" || obv.Trend == "下降" {
		parts = append(parts, "OBV"+obv.Trend)
	}
	if rsiLevel != "数据不足" {
		parts = append(parts, "RSI"+rsiLevel)
	}

	verdict := "技术指标数据不足"
	if len(parts) > 0 {
		verdict = strings.Join(parts, "，")
		if arrangement == "多头排列" {
			verdict += "，技术面偏多"
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
		Verdict:        verdict,
	}
}

func AnalyzeSector(quote models.Quote, sectors []string) models.SectorAnalysis {
	primary := ""
	if len(sectors) > 0 {
		primary = sectors[0]
	}
	totalMV := quote.TotalMV
	name := quote.Name
	isLeader := totalMV >= 500 || strings.HasPrefix(name, "中国") || strings.Contains(name, "龙头")

	verdict := "板块数据不足，暂不判断"
	if primary != "" && isLeader {
		verdict = "所属" + primary + "板块，具备较强行业地位"
	} else if primary != "" {
		verdict = "所属" + primary + "板块，需结合板块强弱观察"
	}

	return models.SectorAnalysis{
		Sectors:        sectors,
		PrimarySector:  primary,
		IsSectorLeader: isLeader,
		Verdict:        verdict,
	}
}

var positiveWords = []string{"预增", "增长", "利好", "突破", "新高", "涨停", "获利", "盈利", "超预期", "翻倍"}
var negativeWords = []string{"预减", "下降", "利空", "跌停", "亏损", "风险", "违规", "处罚", "下跌", "暴跌"}

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
