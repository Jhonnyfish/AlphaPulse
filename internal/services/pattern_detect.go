package services

// Pattern detection logic extracted from pattern_scanner.go for reuse in analyze flow.
// These functions return models.PatternSignal instead of the handler-local PatternResult.

import (
	"fmt"

	"alphapulse/internal/models"
)

// DetectPatterns runs K-line, chart, and volume pattern detection on kline data.
// Requires at least 3 klines; returns empty slice otherwise.
func DetectPatterns(klines []models.KlinePoint) []models.PatternSignal {
	if len(klines) < 3 {
		return nil
	}
	var results []models.PatternSignal
	results = append(results, detectKlineSignals(klines)...)
	results = append(results, detectChartSignals(klines)...)
	results = append(results, detectVolumeSignals(klines)...)
	return results
}

// ──────────────────────────────────────────────
// K-line candlestick patterns
// ──────────────────────────────────────────────

func detectKlineSignals(klines []models.KlinePoint) []models.PatternSignal {
	var results []models.PatternSignal
	n := len(klines)
	bars := klines[n-3:]
	lastDate := klines[n-1].Date

	o := [3]float64{bars[0].Open, bars[1].Open, bars[2].Open}
	c := [3]float64{bars[0].Close, bars[1].Close, bars[2].Close}
	h := [3]float64{bars[0].High, bars[1].High, bars[2].High}
	l := [3]float64{bars[0].Low, bars[1].Low, bars[2].Low}

	body := func(i int) float64 { return pabs(c[i] - o[i]) }
	upperShadow := func(i int) float64 { return h[i] - pmax(o[i], c[i]) }
	lowerShadow := func(i int) float64 { return pmin(o[i], c[i]) - l[i] }
	isUp := func(i int) bool { return c[i] > o[i] }
	isDown := func(i int) bool { return c[i] < o[i] }

	// Doji (十字星)
	price0 := (o[2] + c[2]) / 2
	if price0 > 0 && body(2) < price0*0.001 && body(2) > 0 {
		conf := pmin(0.9, 0.5+(price0*0.001-body(2))/(price0*0.001)*0.4)
		results = append(results, models.PatternSignal{
			Pattern: "十字星", Category: "kline", Direction: "neutral",
			Confidence: round2(conf), Date: lastDate,
			Description: "开盘价与收盘价几乎相同，多空双方力量均衡，信号方向待确认",
		})
	}

	// Hammer (锤子线)
	if body(2) > 0 {
		ls := lowerShadow(2)
		us := upperShadow(2)
		b := body(2)
		if ls >= 2.0*b && us < b*0.5 {
			ratio := ls / b
			conf := pmin(0.95, 0.6+(ratio-2.0)*0.1)
			results = append(results, models.PatternSignal{
				Pattern: "锤子线", Category: "kline", Direction: "bullish",
				Confidence: round2(conf), Date: lastDate,
				Description: fmt.Sprintf("下影线长度为实体%.1f倍，上影线极短，底部看涨信号", ratio),
			})
		}
	}

	// Engulfing (吞没形态)
	prevBody := pabs(c[1] - o[1])
	currBody := pabs(c[2] - o[2])
	if prevBody > 0 && currBody > prevBody {
		if c[1] < o[1] && c[2] > o[2] && o[2] <= c[1] && c[2] >= o[1] {
			conf := pmin(0.9, 0.6+(currBody/prevBody-1)*0.3)
			results = append(results, models.PatternSignal{
				Pattern: "吞没形态", Category: "kline", Direction: "bullish",
				Confidence: round2(conf), Date: lastDate,
				Description: "阳线完全吞没前一根阴线，底部反转看涨信号",
			})
		}
		if c[1] > o[1] && c[2] < o[2] && o[2] >= c[1] && c[2] <= o[1] {
			conf := pmin(0.9, 0.6+(currBody/prevBody-1)*0.3)
			results = append(results, models.PatternSignal{
				Pattern: "吞没形态", Category: "kline", Direction: "bearish",
				Confidence: round2(conf), Date: lastDate,
				Description: "阴线完全吞没前一根阳线，顶部反转看跌信号",
			})
		}
	}

	// Morning Star (早晨之星)
	if isDown(0) && body(1) < body(0)*0.3 && isUp(2) {
		midFirst := (o[0] + c[0]) / 2
		if c[2] > midFirst {
			results = append(results, models.PatternSignal{
				Pattern: "早晨之星", Category: "kline", Direction: "bullish",
				Confidence: 0.8, Date: lastDate,
				Description: "三根K线组合：下跌→小实体→阳线收于第一根中点之上，强烈看涨反转",
			})
		}
	}

	// Evening Star (黄昏之星)
	if isUp(0) && body(1) < body(0)*0.3 && isDown(2) {
		midFirst := (o[0] + c[0]) / 2
		if c[2] < midFirst {
			results = append(results, models.PatternSignal{
				Pattern: "黄昏之星", Category: "kline", Direction: "bearish",
				Confidence: 0.8, Date: lastDate,
				Description: "三根K线组合：上涨→小实体→阴线收于第一根中点之下，强烈看跌反转",
			})
		}
	}

	// Three White Soldiers (三白兵)
	if isUp(0) && isUp(1) && isUp(2) && c[0] < c[1] && c[1] < c[2] && o[0] < o[1] && o[1] < o[2] {
		results = append(results, models.PatternSignal{
			Pattern: "三白兵", Category: "kline", Direction: "bullish",
			Confidence: 0.85, Date: lastDate,
			Description: "连续三根阳线，收盘价逐步抬高，强势上涨信号",
		})
	}

	// Three Black Crows (三黑鸦)
	if isDown(0) && isDown(1) && isDown(2) && c[0] > c[1] && c[1] > c[2] && o[0] > o[1] && o[1] > o[2] {
		results = append(results, models.PatternSignal{
			Pattern: "三黑鸦", Category: "kline", Direction: "bearish",
			Confidence: 0.85, Date: lastDate,
			Description: "连续三根阴线，收盘价逐步走低，强势下跌信号",
		})
	}

	return results
}

// ──────────────────────────────────────────────
// Chart patterns (double top/bottom, triangles, rectangles)
// ──────────────────────────────────────────────

type Pivot struct {
	Idx int
	Val float64
}

func detectChartSignals(klines []models.KlinePoint) []models.PatternSignal {
	var results []models.PatternSignal
	n := len(klines)
	lastDate := klines[n-1].Date

	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	for i, k := range klines {
		closes[i] = k.Close
		highs[i] = k.High
		lows[i] = k.Low
	}

	priceRange := psmax(highs) - psmin(lows)
	if priceRange <= 0 {
		return results
	}
	tolerance := priceRange * 0.03

	window := pimax(3, n/15)
	lowPivots, highPivots := findPivots(closes, window)

	// Double Bottom / Double Top — strict Wikipedia/Bulkowski criteria
	// (prior trend, time gap, recency, neckline depth, neckline-break confirmation,
	//  not-yet-played-out, optional volume confirmation).
	if sig := DetectDoubleBottomStrict(klines, closes, highPivots, lowPivots, tolerance); sig != nil {
		results = append(results, *sig)
	}
	if sig := DetectDoubleTopStrict(klines, closes, highPivots, lowPivots, tolerance); sig != nil {
		results = append(results, *sig)
	}

	// Ascending Triangle (上升三角形)
	if len(highPivots) >= 2 && len(lowPivots) >= 2 {
		recentHighs := plastN(highPivots, 3)
		recentLows := plastN(lowPivots, 3)
		highVals := ppivotVals(recentHighs)
		lowVals := ppivotVals(recentLows)
		highSpread := psmax(highVals) - psmin(highVals)

		if highSpread < tolerance && len(lowVals) >= 2 {
			rising := true
			for j := 1; j < len(lowVals); j++ {
				if lowVals[j] < lowVals[j-1]-tolerance*0.2 {
					rising = false
					break
				}
			}
			if rising && lowVals[len(lowVals)-1] > lowVals[0]+tolerance*0.5 {
				conf := pmin(0.85, 0.6+(1-highSpread/tolerance)*0.25)
				results = append(results, models.PatternSignal{
					Pattern: "上升三角形", Category: "chart", Direction: "bullish",
					Confidence: round2(conf), Date: lastDate,
					Description: fmt.Sprintf("阻力位约%.2f（水平），支撑位逐步抬升，看涨整理形态", psmax(highVals)),
				})
			}
		}
	}

	// Descending Triangle (下降三角形)
	if len(lowPivots) >= 2 && len(highPivots) >= 2 {
		recentLows := plastN(lowPivots, 3)
		recentHighs := plastN(highPivots, 3)
		lowVals := ppivotVals(recentLows)
		highVals := ppivotVals(recentHighs)
		lowSpread := psmax(lowVals) - psmin(lowVals)

		if lowSpread < tolerance && len(highVals) >= 2 {
			declining := true
			for j := 1; j < len(highVals); j++ {
				if highVals[j] > highVals[j-1]+tolerance*0.2 {
					declining = false
					break
				}
			}
			if declining && highVals[len(highVals)-1] < highVals[0]-tolerance*0.5 {
				conf := pmin(0.85, 0.6+(1-lowSpread/tolerance)*0.25)
				results = append(results, models.PatternSignal{
					Pattern: "下降三角形", Category: "chart", Direction: "bearish",
					Confidence: round2(conf), Date: lastDate,
					Description: fmt.Sprintf("支撑位约%.2f（水平），阻力位逐步下降，看跌整理形态", psmin(lowVals)),
				})
			}
		}
	}

	// Symmetrical Triangle (对称三角形)
	if len(lowPivots) >= 3 && len(highPivots) >= 3 {
		recentLows := plastN(lowPivots, 3)
		recentHighs := plastN(highPivots, 3)
		lowVals := ppivotVals(recentLows)
		highVals := ppivotVals(recentHighs)
		risingLows := lowVals[1] >= lowVals[0]-tolerance*0.2 && lowVals[2] >= lowVals[1]-tolerance*0.2
		fallingHighs := highVals[1] <= highVals[0]+tolerance*0.2 && highVals[2] <= highVals[1]+tolerance*0.2
		converging := highVals[2]-lowVals[2] < (highVals[0]-lowVals[0])*0.75
		if risingLows && fallingHighs && converging {
			results = append(results, models.PatternSignal{
				Pattern: "对称三角形", Category: "chart", Direction: "neutral",
				Confidence: 0.72, Date: lastDate,
				Description: fmt.Sprintf("高点下降、低点抬升，区间收敛至%.2f-%.2f，等待放量突破方向", lowVals[2], highVals[2]),
			})
		}
	}

	// Head and shoulders (top & bottom) — strict Wikipedia/Bulkowski criteria.
	if sig := DetectHeadShouldersTopStrict(klines, closes, highPivots, lowPivots, tolerance); sig != nil {
		results = append(results, *sig)
	}
	if sig := DetectHeadShouldersBottomStrict(klines, closes, highPivots, lowPivots, tolerance); sig != nil {
		results = append(results, *sig)
	}

	// Flag patterns: sharp pole followed by tight consolidation.
	if n >= 18 {
		poleStart := n - 18
		poleEnd := n - 7
		poleMove := 0.0
		if closes[poleStart] > 0 {
			poleMove = (closes[poleEnd] - closes[poleStart]) / closes[poleStart]
		}
		flagHigh := psmax(highs[n-7:])
		flagLow := psmin(lows[n-7:])
		flagRangePct := 0.0
		if closes[n-1] > 0 {
			flagRangePct = (flagHigh - flagLow) / closes[n-1]
		}
		if poleMove > 0.08 && flagRangePct < 0.06 {
			results = append(results, models.PatternSignal{
				Pattern: "上升旗形", Category: "chart", Direction: "bullish",
				Confidence: 0.7, Date: lastDate,
				Description: fmt.Sprintf("前段上涨%.1f%%后窄幅整理，若放量突破旗形上沿则延续概率较高", poleMove*100),
			})
		}
		if poleMove < -0.08 && flagRangePct < 0.06 {
			results = append(results, models.PatternSignal{
				Pattern: "下降旗形", Category: "chart", Direction: "bearish",
				Confidence: 0.7, Date: lastDate,
				Description: fmt.Sprintf("前段下跌%.1f%%后窄幅整理，若跌破旗形下沿则延续概率较高", pabs(poleMove)*100),
			})
		}
	}

	// Rectangle/Box (箱体整理)
	if len(lowPivots) >= 2 && len(highPivots) >= 2 {
		recentLows := plastN(lowPivots, 3)
		recentHighs := plastN(highPivots, 3)
		lowVals := ppivotVals(recentLows)
		highVals := ppivotVals(recentHighs)
		lowSpread := psmax(lowVals) - psmin(lowVals)
		highSpread := psmax(highVals) - psmin(highVals)

		if lowSpread < tolerance && highSpread < tolerance {
			avgRange := psmax(highVals) - psmin(lowVals)
			if avgRange > tolerance {
				conf := pmin(0.8, 0.55+(1-(lowSpread+highSpread)/(2*tolerance))*0.25)
				results = append(results, models.PatternSignal{
					Pattern: "箱体整理", Category: "chart", Direction: "neutral",
					Confidence: round2(conf), Date: lastDate,
					Description: fmt.Sprintf("价格在%.2f-%.2f区间震荡，关注突破方向", psmin(lowVals), psmax(highVals)),
				})
			}
		}
	}

	return results
}

// ──────────────────────────────────────────────
// Strict reversal-pattern detectors (Wikipedia / Bulkowski criteria)
//
// Common rules applied to double top/bottom and head & shoulders:
//   1. Prior opposing trend (the pattern is a reversal, not a continuation).
//   2. Time gap between the relevant pivots (not too close → consolidation,
//      not too far → stale).
//   3. Recency: last pivot must be within ~25 bars (otherwise the pattern
//      is no longer actionable).
//   4. Neckline depth ≥ 5% of price (real reversal, not noise).
//   5. Confirmation: current close must be ≥3% beyond the neckline.
//      Without neckline break, it is just consolidation.
//   6. Not yet at measured target (peak/trough height projected from
//      neckline); otherwise the pattern has already played out.
//   7. Optional volume confirmation raises confidence.
// ──────────────────────────────────────────────

// DetectDoubleTopStrict searches for a confirmed M-top.
// Iterates newest-to-oldest so the most recent valid pattern wins.
func DetectDoubleTopStrict(
	klines []models.KlinePoint,
	closes []float64,
	highPivots, lowPivots []Pivot,
	tolerance float64,
) *models.PatternSignal {
	n := len(klines)
	if n < 25 || len(highPivots) < 2 || len(closes) != n {
		return nil
	}

	for i := len(highPivots) - 1; i >= 1; i-- {
		idx2, val2 := highPivots[i].Idx, highPivots[i].Val
		// Recency: peak2 must be within last 25 bars.
		if n-1-idx2 > 25 {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			idx1, val1 := highPivots[j].Idx, highPivots[j].Val
			gap := idx2 - idx1
			if gap < 5 || gap > 30 {
				continue
			}
			// Peak values within tolerance.
			if pabs(val1-val2) > tolerance {
				continue
			}
			// Find the lowest trough between the two peaks (neckline).
			troughVal, hasTrough := 0.0, false
			for _, lp := range lowPivots {
				if lp.Idx > idx1 && lp.Idx < idx2 {
					if !hasTrough || lp.Val < troughVal {
						troughVal = lp.Val
						hasTrough = true
					}
				}
			}
			if !hasTrough {
				continue
			}
			peakAvg := (val1 + val2) / 2
			// Neckline depth ≥ 5% of peak.
			if peakAvg <= 0 || peakAvg-troughVal < peakAvg*0.05 {
				continue
			}
			// Prior uptrend: at least 5% rise over 20 bars ending at idx1.
			if !hasPriorUptrend(closes, idx1, 20, 0.05) {
				continue
			}
			// Confirmation: current close must be ≥3% below neckline.
			currentClose := closes[n-1]
			if currentClose > troughVal*0.97 {
				continue
			}
			// Not yet at measured target (peak height projected below neckline).
			target := troughVal - (peakAvg - troughVal)
			if currentClose < target*0.95 {
				continue
			}

			peak1Vol := avgVolAround(klines, idx1)
			peak2Vol := avgVolAround(klines, idx2)
			volumeConfirm := peak1Vol > 0 && peak2Vol > 0 && peak1Vol > peak2Vol*1.1

			baseConf := 0.65 + (1-pabs(val1-val2)/tolerance)*0.2
			if volumeConfirm {
				baseConf += 0.1
			}
			conf := pmin(0.92, baseConf)

			desc := fmt.Sprintf("M顶形态，两高点约%.2f（间隔%d根），颈线%.2f已跌破确认，看跌反转",
				peakAvg, gap, troughVal)
			if !volumeConfirm {
				desc = fmt.Sprintf("M顶形态，两高点约%.2f（间隔%d根），颈线%.2f已跌破，第二峰量能未明显萎缩",
					peakAvg, gap, troughVal)
			}
			return &models.PatternSignal{
				Pattern: "双顶", Category: "chart", Direction: "bearish",
				Confidence: round2(conf),
				Date:       klines[n-1].Date,
				Description: desc,
			}
		}
	}
	return nil
}

// DetectDoubleBottomStrict searches for a confirmed W-bottom (mirror of M-top).
func DetectDoubleBottomStrict(
	klines []models.KlinePoint,
	closes []float64,
	highPivots, lowPivots []Pivot,
	tolerance float64,
) *models.PatternSignal {
	n := len(klines)
	if n < 25 || len(lowPivots) < 2 || len(closes) != n {
		return nil
	}

	for i := len(lowPivots) - 1; i >= 1; i-- {
		idx2, val2 := lowPivots[i].Idx, lowPivots[i].Val
		if n-1-idx2 > 25 {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			idx1, val1 := lowPivots[j].Idx, lowPivots[j].Val
			gap := idx2 - idx1
			if gap < 5 || gap > 30 {
				continue
			}
			if pabs(val1-val2) > tolerance {
				continue
			}
			peakVal, hasPeak := 0.0, false
			for _, hp := range highPivots {
				if hp.Idx > idx1 && hp.Idx < idx2 {
					if !hasPeak || hp.Val > peakVal {
						peakVal = hp.Val
						hasPeak = true
					}
				}
			}
			if !hasPeak {
				continue
			}
			troughAvg := (val1 + val2) / 2
			if troughAvg <= 0 || peakVal-troughAvg < troughAvg*0.05 {
				continue
			}
			if !hasPriorDowntrend(closes, idx1, 20, 0.05) {
				continue
			}
			currentClose := closes[n-1]
			if currentClose < peakVal*1.03 {
				continue
			}
			target := peakVal + (peakVal - troughAvg)
			if currentClose > target*1.05 {
				continue
			}

			// W-bottom: second trough should form on higher volume (selling
			// exhaustion) or the breakout above neckline should bring volume.
			trough1Vol := avgVolAround(klines, idx1)
			trough2Vol := avgVolAround(klines, idx2)
			volumeConfirm := trough1Vol > 0 && trough2Vol > 0 && trough2Vol > trough1Vol*1.1

			baseConf := 0.65 + (1-pabs(val1-val2)/tolerance)*0.2
			if volumeConfirm {
				baseConf += 0.1
			}
			conf := pmin(0.92, baseConf)

			desc := fmt.Sprintf("W底形态，两低点约%.2f（间隔%d根），颈线%.2f已突破确认，看涨反转",
				troughAvg, gap, peakVal)
			if !volumeConfirm {
				desc = fmt.Sprintf("W底形态，两低点约%.2f（间隔%d根），颈线%.2f已突破，第二底量能未放大",
					troughAvg, gap, peakVal)
			}
			return &models.PatternSignal{
				Pattern: "双底", Category: "chart", Direction: "bullish",
				Confidence: round2(conf),
				Date:       klines[n-1].Date,
				Description: desc,
			}
		}
	}
	return nil
}

// DetectHeadShouldersTopStrict searches for a confirmed head-and-shoulders top.
// Peaks are in time order: left shoulder (LS), head (H), right shoulder (RS).
func DetectHeadShouldersTopStrict(
	klines []models.KlinePoint,
	closes []float64,
	highPivots, lowPivots []Pivot,
	tolerance float64,
) *models.PatternSignal {
	n := len(klines)
	if n < 30 || len(highPivots) < 3 || len(closes) != n {
		return nil
	}

	// Iterate RS (latest peak) from newest to oldest.
	for i := len(highPivots) - 1; i >= 2; i-- {
		right := highPivots[i]
		if n-1-right.Idx > 30 {
			continue
		}
		for j := i - 1; j >= 1; j-- {
			head := highPivots[j]
			if head.Idx >= right.Idx {
				continue
			}
			for k := j - 1; k >= 0; k-- {
				left := highPivots[k]
				if left.Idx >= head.Idx {
					continue
				}
				gap1 := head.Idx - left.Idx
				gap2 := right.Idx - head.Idx
				if gap1 < 3 || gap2 < 3 || gap1 > 20 || gap2 > 20 {
					continue
				}
				// Head must be clearly higher than both shoulders.
				if head.Val <= pmax(left.Val, right.Val)+tolerance*0.5 {
					continue
				}
				// Shoulders roughly symmetric (within 8% of average).
				shoulderAvg := (left.Val + right.Val) / 2
				if shoulderAvg <= 0 || pabs(left.Val-right.Val)/shoulderAvg > 0.08 {
					continue
				}
				// Two neckline troughs: between LS-H and H-RS.
				t1, t1OK := 0.0, false
				for _, lp := range lowPivots {
					if lp.Idx > left.Idx && lp.Idx < head.Idx {
						if !t1OK || lp.Val < t1 {
							t1 = lp.Val
							t1OK = true
						}
					}
				}
				t2, t2OK := 0.0, false
				for _, lp := range lowPivots {
					if lp.Idx > head.Idx && lp.Idx < right.Idx {
						if !t2OK || lp.Val < t2 {
							t2 = lp.Val
							t2OK = true
						}
					}
				}
				if !t1OK || !t2OK {
					continue
				}
				neckline := (t1 + t2) / 2
				// Neckline endpoints within 3% (near-horizontal).
				if neckline <= 0 || pabs(t1-t2) > neckline*0.03 {
					continue
				}
				if !hasPriorUptrend(closes, left.Idx, 20, 0.05) {
					continue
				}
				currentClose := closes[n-1]
				if currentClose > neckline*0.97 {
					continue
				}
				target := neckline - (head.Val - neckline)
				if currentClose < target*0.95 {
					continue
				}

				lsVol := avgVolAround(klines, left.Idx)
				rsVol := avgVolAround(klines, right.Idx)
				volumeConfirm := lsVol > 0 && rsVol > 0 && lsVol > rsVol*1.1

				conf := 0.7
				if volumeConfirm {
					conf += 0.1
				}
				conf = pmin(0.9, conf)

				desc := fmt.Sprintf("头肩顶：左肩%.2f、头部%.2f、右肩%.2f，颈线%.2f已跌破，看跌反转",
					left.Val, head.Val, right.Val, neckline)
				if !volumeConfirm {
					desc = fmt.Sprintf("头肩顶：左肩%.2f、头部%.2f、右肩%.2f，颈线%.2f已跌破（右肩量能未萎缩）",
						left.Val, head.Val, right.Val, neckline)
				}
				return &models.PatternSignal{
					Pattern: "头肩顶", Category: "chart", Direction: "bearish",
					Confidence: round2(conf),
					Date:       klines[n-1].Date,
					Description: desc,
				}
			}
		}
	}
	return nil
}

// DetectHeadShouldersBottomStrict — mirror of head-and-shoulders top.
func DetectHeadShouldersBottomStrict(
	klines []models.KlinePoint,
	closes []float64,
	highPivots, lowPivots []Pivot,
	tolerance float64,
) *models.PatternSignal {
	n := len(klines)
	if n < 30 || len(lowPivots) < 3 || len(closes) != n {
		return nil
	}

	for i := len(lowPivots) - 1; i >= 2; i-- {
		right := lowPivots[i]
		if n-1-right.Idx > 30 {
			continue
		}
		for j := i - 1; j >= 1; j-- {
			head := lowPivots[j]
			if head.Idx >= right.Idx {
				continue
			}
			for k := j - 1; k >= 0; k-- {
				left := lowPivots[k]
				if left.Idx >= head.Idx {
					continue
				}
				gap1 := head.Idx - left.Idx
				gap2 := right.Idx - head.Idx
				if gap1 < 3 || gap2 < 3 || gap1 > 20 || gap2 > 20 {
					continue
				}
				// Head must be clearly lower than both shoulders.
				if head.Val >= pmin(left.Val, right.Val)-tolerance*0.5 {
					continue
				}
				shoulderAvg := (left.Val + right.Val) / 2
				if shoulderAvg <= 0 || pabs(left.Val-right.Val)/shoulderAvg > 0.08 {
					continue
				}
				// Two neckline peaks.
				p1, p1OK := 0.0, false
				for _, hp := range highPivots {
					if hp.Idx > left.Idx && hp.Idx < head.Idx {
						if !p1OK || hp.Val > p1 {
							p1 = hp.Val
							p1OK = true
						}
					}
				}
				p2, p2OK := 0.0, false
				for _, hp := range highPivots {
					if hp.Idx > head.Idx && hp.Idx < right.Idx {
						if !p2OK || hp.Val > p2 {
							p2 = hp.Val
							p2OK = true
						}
					}
				}
				if !p1OK || !p2OK {
					continue
				}
				neckline := (p1 + p2) / 2
				if neckline <= 0 || pabs(p1-p2) > neckline*0.03 {
					continue
				}
				if !hasPriorDowntrend(closes, left.Idx, 20, 0.05) {
					continue
				}
				currentClose := closes[n-1]
				if currentClose < neckline*1.03 {
					continue
				}
				target := neckline + (neckline - head.Val)
				if currentClose > target*1.05 {
					continue
				}

				// Inverse H&S: right-side rally on increasing volume.
				lsVol := avgVolAround(klines, left.Idx)
				rsVol := avgVolAround(klines, right.Idx)
				volumeConfirm := lsVol > 0 && rsVol > 0 && rsVol > lsVol*1.1

				conf := 0.7
				if volumeConfirm {
					conf += 0.1
				}
				conf = pmin(0.9, conf)

				desc := fmt.Sprintf("头肩底：左肩%.2f、头部%.2f、右肩%.2f，颈线%.2f已突破，看涨反转",
					left.Val, head.Val, right.Val, neckline)
				if !volumeConfirm {
					desc = fmt.Sprintf("头肩底：左肩%.2f、头部%.2f、右肩%.2f，颈线%.2f已突破（右肩量能未放大）",
						left.Val, head.Val, right.Val, neckline)
				}
				return &models.PatternSignal{
					Pattern: "头肩底", Category: "chart", Direction: "bullish",
					Confidence: round2(conf),
					Date:       klines[n-1].Date,
					Description: desc,
				}
			}
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// Helpers shared by the strict detectors
// ──────────────────────────────────────────────

// hasPriorUptrend returns true if closes[idx] is at least (1+gainPct) × closes
// at max(0, idx-lookback).
func hasPriorUptrend(closes []float64, idx, lookback int, gainPct float64) bool {
	if idx <= 0 || idx >= len(closes) {
		return false
	}
	start := pimax(0, idx-lookback)
	if closes[start] <= 0 {
		return false
	}
	return closes[idx] >= closes[start]*(1+gainPct)
}

func hasPriorDowntrend(closes []float64, idx, lookback int, dropPct float64) bool {
	if idx <= 0 || idx >= len(closes) {
		return false
	}
	start := pimax(0, idx-lookback)
	if closes[start] <= 0 {
		return false
	}
	return closes[idx] <= closes[start]*(1-dropPct)
}

// avgVolAround returns the mean volume of the 3 bars centered on idx.
func avgVolAround(klines []models.KlinePoint, idx int) float64 {
	n := len(klines)
	if idx < 0 || idx >= n {
		return 0
	}
	start := pimax(0, idx-1)
	end := pimin(n, idx+2)
	if start >= end {
		return 0
	}
	sum, count := 0.0, 0
	for k := start; k < end; k++ {
		sum += klines[k].Volume
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func pimin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ──────────────────────────────────────────────
// Volume patterns
// ──────────────────────────────────────────────

func detectVolumeSignals(klines []models.KlinePoint) []models.PatternSignal {
	var results []models.PatternSignal
	n := len(klines)
	if n < 20 {
		return results
	}
	lastDate := klines[n-1].Date

	closes := make([]float64, n)
	highs := make([]float64, n)
	volumes := make([]float64, n)
	for i, k := range klines {
		closes[i] = k.Close
		highs[i] = k.High
		volumes[i] = k.Volume
	}

	avgVol20 := psavg(volumes[n-20:])
	if avgVol20 <= 0 {
		return results
	}

	recentHighMax := psmax(highs[n-20 : n-1])
	latestVol := volumes[n-1]
	latestClose := closes[n-1]

	// Volume Breakout (放量突破)
	if latestVol > 2*avgVol20 && latestClose > recentHighMax {
		volRatio := latestVol / avgVol20
		conf := pmin(0.95, 0.6+(volRatio-2)*0.1)
		results = append(results, models.PatternSignal{
			Pattern: "放量突破", Category: "volume", Direction: "bullish",
			Confidence: round2(conf), Date: lastDate,
			Description: fmt.Sprintf("成交量为20日均量的%.1f倍，价格突破近期高点%.2f，看涨", volRatio, recentHighMax),
		})
	}

	// Contraction Pullback (缩量回调)
	if n >= 5 {
		peakClose := psmax(closes[pimax(0, n-10) : n-1])
		pullback := 0.0
		if peakClose > 0 {
			pullback = (peakClose - latestClose) / peakClose
		}
		recentAvgVol := psavg(volumes[n-5:])
		if pullback > 0.02 && recentAvgVol < avgVol20*0.5 {
			volRatio := recentAvgVol / avgVol20
			conf := pmin(0.85, 0.55+(0.5-volRatio)*0.6)
			results = append(results, models.PatternSignal{
				Pattern: "缩量回调", Category: "volume", Direction: "bullish",
				Confidence: round2(conf), Date: lastDate,
				Description: fmt.Sprintf("价格从高点回调%.1f%%，但成交量降至20日均量的%.0f%%，回调无量看涨", pullback*100, volRatio*100),
			})
		}
	}

	// Volume-Price Divergence (量价背离)
	if n >= 10 {
		fhClose := psavg(closes[n-10 : n-5])
		shClose := psavg(closes[n-5:])
		fhVol := psavg(volumes[n-10 : n-5])
		shVol := psavg(volumes[n-5:])

		if shClose > fhClose*1.01 && shVol < fhVol*0.8 {
			conf := pmin(0.85, 0.6+(1-shVol/fhVol)*0.5)
			results = append(results, models.PatternSignal{
				Pattern: "量价背离", Category: "volume", Direction: "bearish",
				Confidence: round2(conf), Date: lastDate,
				Description: "价格创近期新高但成交量递减，上涨动能减弱，看跌背离",
			})
		}
		if shClose < fhClose*0.99 && shVol < fhVol*0.7 {
			conf := pmin(0.8, 0.55+(1-shVol/fhVol)*0.5)
			results = append(results, models.PatternSignal{
				Pattern: "量价背离", Category: "volume", Direction: "bullish",
				Confidence: round2(conf), Date: lastDate,
				Description: "价格创新低但成交量萎缩，卖压衰竭，底部看涨背离",
			})
		}
	}

	return results
}

// ──────────────────────────────────────────────
// Pivot detection + math helpers
// ──────────────────────────────────────────────

func findPivots(data []float64, window int) (mins, maxs []Pivot) {
	for i := window; i < len(data)-window; i++ {
		isMin, isMax := true, true
		for j := 1; j <= window; j++ {
			if data[i] > data[i-j] || data[i] > data[i+j] {
				isMin = false
			}
			if data[i] < data[i-j] || data[i] < data[i+j] {
				isMax = false
			}
		}
		if isMin {
			mins = append(mins, Pivot{Idx: i, Val: data[i]})
		}
		if isMax {
			maxs = append(maxs, Pivot{Idx: i, Val: data[i]})
		}
	}
	return
}

func pabs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func pabsInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func pmin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func pmax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func pimax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func psmax(data []float64) float64 {
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

func psmin(data []float64) float64 {
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

func psavg(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func ppivotVals(pivots []Pivot) []float64 {
	vals := make([]float64, len(pivots))
	for i, p := range pivots {
		vals[i] = p.Val
	}
	return vals
}

func plastN(pivots []Pivot, n int) []Pivot {
	if len(pivots) <= n {
		return pivots
	}
	return pivots[len(pivots)-n:]
}
