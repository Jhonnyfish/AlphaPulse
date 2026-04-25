package services

import (
	"context"
	"math"

	"alphapulse/internal/models"
)

// BacktestTrade records a single simulated trade.
type BacktestTrade struct {
	SignalDate  string  `json:"signal_date"`
	SellDate    string  `json:"sell_date"`
	BuyPrice    float64 `json:"buy_price"`
	SellPrice   float64 `json:"sell_price"`
	HoldingDays int     `json:"holding_days"`
	Score       int     `json:"score"`
	ReturnPct   float64 `json:"return_pct"`
}

// BacktestResult is the response for a single-stock backtest run.
type BacktestResult struct {
	Code         string          `json:"code"`
	Days         int             `json:"days"`
	SignalCount  int             `json:"signal_count"`
	WinRate      float64         `json:"win_rate"`
	AvgReturnPct float64         `json:"avg_return_pct"`
	MaxDrawdown  float64         `json:"max_drawdown_pct"`
	Trades       []BacktestTrade `json:"trades"`
	Error        string          `json:"error,omitempty"`
}

// ScoreFromKlines computes a simplified 8-dimension score from kline data only.
// Base score = 50, adjusted by 8 technical dimensions.
func ScoreFromKlines(klines []models.KlinePoint) (int, map[string]interface{}) {
	closes := make([]float64, 0, len(klines))
	volumes := make([]float64, 0, len(klines))
	for _, k := range klines {
		if k.Close > 0 {
			closes = append(closes, k.Close)
			volumes = append(volumes, k.Volume)
		}
	}
	if len(closes) < 20 {
		return 0, map[string]interface{}{"error": "数据不足"}
	}

	macd := CalculateMACD(closes)
	rsi := CalculateRSI(closes, 14)
	boll := CalculateBollinger(closes, 20)
	ma5 := MovingAverage(closes, 5)
	ma10 := MovingAverage(closes, 10)
	ma20 := MovingAverage(closes, 20)

	last := closes[len(closes)-1]
	prev := closes[len(closes)-2]
	prev3 := closes[0]
	if len(closes) >= 4 {
		prev3 = closes[len(closes)-4]
	}

	avgVol5 := 0.0
	n := min(5, len(volumes))
	if n > 0 {
		for _, v := range volumes[len(volumes)-n:] {
			avgVol5 += v
		}
		avgVol5 /= float64(n)
	}
	volRatio := 0.0
	if avgVol5 > 0 {
		volRatio = volumes[len(volumes)-1] / avgVol5
	}

	dims := map[string]interface{}{}
	maArrangement := 0
	if ma5 > ma10 && ma10 > ma20 {
		maArrangement = 1
	}
	maBreakout := 0
	if last > ma20 && prev <= ma20 {
		maBreakout = 1
	}
	macdSignal := 0
	if macd.Hist > 0 {
		macdSignal = 1
	}
	macdHistTrend := 0
	if macd.HistTrend == "连续增强" {
		macdHistTrend = 1
	}
	rsiState := 0
	if rsi >= 45 && rsi <= 75 {
		rsiState = 1
	}
	volumeRatioDim := 0
	if volRatio >= 1.0 && volRatio <= 2.5 {
		volumeRatioDim = 1
	}
	bollPosition := 0
	if boll.Mid > 0 && last >= boll.Mid {
		bollPosition = 1
	}
	momentum3d := 0
	if last > prev3 {
		momentum3d = 1
	}

	dims["ma_arrangement"] = maArrangement
	dims["ma_breakout"] = maBreakout
	dims["macd_signal"] = macdSignal
	dims["macd_hist_trend"] = macdHistTrend
	dims["rsi_state"] = rsiState
	dims["volume_ratio"] = volumeRatioDim
	dims["boll_position"] = bollPosition
	dims["momentum_3d"] = momentum3d
	dims["vol_ratio"] = math.Round(volRatio*100) / 100

	score := 50
	score += maArrangement * 10
	score += maBreakout * 8
	score += macdSignal * 10
	score += macdHistTrend * 8
	score += rsiState * 8
	score += volumeRatioDim * 8
	score += bollPosition * 8
	score += momentum3d * 10

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score, dims
}

// RunBacktest runs a simplified backtest for a single stock.
// It slides through klines, scores each point, and simulates trades when score >= 70.
func RunBacktest(ctx context.Context, eastMoney *EastMoneyService, code string, days int) BacktestResult {
	code = NormalizeCode(code)
	needCount := days + 80
	if needCount < 120 {
		needCount = 120
	}

	klines, err := eastMoney.FetchKline(ctx, code, needCount)
	if err != nil {
		return BacktestResult{
			Code: code,
			Days: days,
			Error: err.Error(),
		}
	}
	if len(klines) < 25 {
		return BacktestResult{
			Code:  code,
			Days:  days,
			Error: "K线数据不足",
		}
	}

	startIdx := 20
	if len(klines)-days > startIdx {
		startIdx = len(klines) - days
	}

	var trades []BacktestTrade
	for i := startIdx; i < len(klines)-3; i++ {
		score, _ := ScoreFromKlines(klines[:i+1])
		if score >= 70 {
			buy := klines[i]
			sell := klines[i+3]
			buyPrice := buy.Close
			sellPrice := sell.Close
			retPct := 0.0
			if buyPrice > 0 {
				retPct = (sellPrice - buyPrice) / buyPrice * 100
			}
			trades = append(trades, BacktestTrade{
				SignalDate:  buy.Date,
				SellDate:    sell.Date,
				BuyPrice:    math.Round(buyPrice*100) / 100,
				SellPrice:   math.Round(sellPrice*100) / 100,
				HoldingDays: 3,
				Score:       score,
				ReturnPct:   math.Round(retPct*100) / 100,
			})
		}
	}

	signalCount := len(trades)
	wins := 0
	totalReturn := 0.0
	for _, t := range trades {
		totalReturn += t.ReturnPct
		if t.ReturnPct > 0 {
			wins++
		}
	}

	winRate := 0.0
	avgReturn := 0.0
	if signalCount > 0 {
		winRate = float64(wins) / float64(signalCount) * 100
		avgReturn = totalReturn / float64(signalCount)
	}

	// Build equity curve for max drawdown
	equity := make([]float64, 0, signalCount+1)
	equity = append(equity, 1.0)
	for _, t := range trades {
		last := equity[len(equity)-1]
		equity = append(equity, last*(1+t.ReturnPct/100))
	}
	maxDD := maxDrawdown(equity)

	return BacktestResult{
		Code:         code,
		Days:         days,
		SignalCount:  signalCount,
		WinRate:      math.Round(winRate*100) / 100,
		AvgReturnPct: math.Round(avgReturn*100) / 100,
		MaxDrawdown:  math.Round(maxDD*100) / 100,
		Trades:       trades,
	}
}

func maxDrawdown(equity []float64) float64 {
	if len(equity) == 0 {
		return 0.0
	}
	peak := equity[0]
	maxDD := 0.0
	for _, v := range equity {
		if v > peak {
			peak = v
		}
		if peak > 0 {
			dd := (peak - v) / peak * 100
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	return maxDD
}
