package services

import (
	"context"
	"fmt"
	"math"

	"alphapulse/internal/models"
)

// ──────────────────────────────────────────────
// Strategy engine: dynamic position management
// ──────────────────────────────────────────────
//
// Replaces the naive fixed-T+3 backtest with a realistic strategy:
//   - BUY  when composite score ≥ 70 and trend aligned
//   - SELL when trailing stop hit OR MA20 broken OR score < 40 OR time stop (30d)
//   - Position sizing: 33% at 70-79, 66% at 80-89, 100% at 90+
//
// Produces daily actionable signals with reasoning and a full
// walk-forward backtest including benchmark comparison.

// ──────────────────────────
// Core types
// ──────────────────────────

// DailySignal is a single day's trading recommendation.
type DailySignal struct {
	Date        string  `json:"date"`
	Action      string  `json:"action"`       // BUY / SELL / HOLD
	Price       float64 `json:"price"`         // actionable price (today's close)
	PositionPct float64 `json:"position_pct"`  // 0.0 ~ 100.0, current position after this action
	Score       int     `json:"score"`         // composite score that day
	Reason      string  `json:"reason"`        // human-readable rationale
	TradeID     string  `json:"trade_id,omitempty"` // set on BUY + SELL days
}

// StrategyTrade is a completed round-trip.
type StrategyTrade struct {
	TradeID     string  `json:"trade_id"`
	BuyDate     string  `json:"buy_date"`
	BuyPrice    float64 `json:"buy_price"`
	BuyScore    int     `json:"buy_score"`
	SellDate    string  `json:"sell_date"`
	SellPrice   float64 `json:"sell_price"`
	SellReason  string  `json:"sell_reason"`
	HoldingDays int     `json:"holding_days"`
	ReturnPct   float64 `json:"return_pct"`
	PositionPct float64 `json:"position_pct"` // % of capital deployed
}

// StrategyResult is the full output of a strategy run.
type StrategyResult struct {
	Code          string           `json:"code"`
	Name          string           `json:"name,omitempty"`
	Days          int              `json:"days"`
	DailySignals  []DailySignal    `json:"daily_signals,omitempty"` // last 60 days of signals
	Trades        []StrategyTrade  `json:"trades"`
	EquityCurve   []EquityPoint    `json:"equity_curve"`
	BenchmarkCurve []EquityPoint   `json:"benchmark_curve"`

	// Summary stats
	StrategyReturnPct  float64 `json:"strategy_return_pct"`
	BuyHoldReturnPct   float64 `json:"buy_hold_return_pct"`
	SharpeRatio        float64 `json:"sharpe_ratio"`
	MaxDrawdownPct     float64 `json:"max_drawdown_pct"`
	WinRate            float64 `json:"win_rate"`
	AvgHoldingDays     float64 `json:"avg_holding_days"`
	TotalTrades        int     `json:"total_trades"`
	SignalEfficiency   float64 `json:"signal_efficiency"` // % of days with active position

	Error string `json:"error,omitempty"`
}

// EquityPoint is one day on the equity curve.
type EquityPoint struct {
	Date      string  `json:"date"`
	Equity    float64 `json:"equity"`    // starting from 1.0
	InPosition bool   `json:"in_position"`
}

// ──────────────────────────
// Config
// ──────────────────────────

type strategyConfig struct {
	buyThreshold     float64 // minimum composite score to enter (70)
	exitScoreFloor   float64 // sell if score drops below this (40)
	trailingStopPct  float64 // sell if price < highest since entry * (1 - this) (0.07 = 7%)
	maExitPeriod     int     // sell if price < MA(N) (20)
	maxHoldDays      int     // force sell after N days (30)
	warmupDays       int     // minimum bars to compute indicators (20)
	commissionBuy    float64 // 0.0005
	commissionSell   float64 // 0.0010  (佣金 0.05% + 印花税 0.05%)
	signalLookback   int     // how many recent signals to return (60)
}

var defaultStrategyConfig = strategyConfig{
	buyThreshold:    70,
	exitScoreFloor:  40,
	trailingStopPct: 0.07,
	maExitPeriod:    20,
	maxHoldDays:     30,
	warmupDays:      20,
	commissionBuy:   0.0005,
	commissionSell:  0.0010,
	signalLookback:  60,
}

// ──────────────────────────
// Position state
// ──────────────────────────

type positionState struct {
	active          bool
	entryPrice      float64
	entryDate       string
	entryScore      int
	positionPct     float64 // fraction of capital deployed (0.33, 0.66, 1.0)
	deployedCapital float64 // absolute capital deployed at entry time
	highestPrice    float64 // since entry
	daysHeld        int
}

func (p *positionState) reset() {
	*p = positionState{}
}

func (p *positionState) enter(date string, price float64, score int, posPct float64, deployed float64) {
	p.active = true
	p.entryPrice = price
	p.entryDate = date
	p.entryScore = score
	p.positionPct = posPct
	p.deployedCapital = deployed
	p.highestPrice = price
	p.daysHeld = 0
}

// advance updates daily state (highest price, day count).
func (p *positionState) advance(todayHigh float64) {
	p.daysHeld++
	if todayHigh > p.highestPrice {
		p.highestPrice = todayHigh
	}
}

// shouldSell returns (shouldSell bool, reason string).
func (p *positionState) shouldSell(price float64, score int, ma20 float64, cfg strategyConfig) (bool, string) {
	if !p.active {
		return false, ""
	}

	// Trailing stop
	trailPrice := p.highestPrice * (1 - cfg.trailingStopPct)
	if price <= trailPrice {
		return true, fmt.Sprintf("trailing stop: 最高价%.2f, 回撤%.1f%%, 止损价%.2f",
			p.highestPrice, (1-price/p.highestPrice)*100, trailPrice)
	}

	// MA exit
	if ma20 > 0 && price < ma20 {
		return true, fmt.Sprintf("跌破MA%d(%.2f)", cfg.maExitPeriod, ma20)
	}

	// Score reversal
	if score < int(cfg.exitScoreFloor) {
		return true, fmt.Sprintf("评分跌至%d, 低于阈值%d", score, int(cfg.exitScoreFloor))
	}

	// Time stop
	if p.daysHeld >= cfg.maxHoldDays {
		return true, fmt.Sprintf("持仓已达%d天, 触及时间止损", cfg.maxHoldDays)
	}

	return false, ""
}

// ──────────────────────────
// Main engine
// ──────────────────────────

// RunStrategyBacktest runs a strategy backtest with dynamic position management.
// Returns daily signals, trades, equity curves, and comparison metrics.
func RunStrategyBacktest(
	ctx context.Context,
	eastMoney *EastMoneyService,
	code string,
	days int,
) StrategyResult {
	code = NormalizeCode(code)
	cfg := defaultStrategyConfig

	needCount := days + cfg.warmupDays*2
	if needCount < 120 {
		needCount = 120
	}

	klines, err := eastMoney.FetchKline(ctx, code, needCount)
	if err != nil {
		return StrategyResult{Code: code, Days: days, Error: err.Error()}
	}
	return runStrategy(ctx, code, days, klines)
}

// runStrategy is the core backtest logic, separated so tests can inject klines directly.
func runStrategy(_ context.Context, code string, days int, klines []models.KlinePoint) StrategyResult {
	cfg := defaultStrategyConfig

	if len(klines) < cfg.warmupDays+1 {
		return StrategyResult{Code: code, Days: days, Error: "K线数据不足"}
	}

	// Clamp start index to the evaluation window
	startIdx := cfg.warmupDays
	if len(klines)-days > startIdx {
		startIdx = len(klines) - days
	}

	// ---- Walk-forward simulation ----
	var (
		pos           positionState
		trades        []StrategyTrade
		allSignals    []DailySignal
		equityCurve   []EquityPoint
		benchCurve    []EquityPoint
		tradeCounter  int
		cashEquity    float64 = 1.0
		positionValue float64 = 0.0
	)

	benchEntry := klines[startIdx].Close
	if benchEntry <= 0 {
		benchEntry = 1
	}

	for i := startIdx; i < len(klines); i++ {
		day := klines[i]
		hist := klines[:i+1] // data available before today's close
		score, _ := ScoreFromKlines(hist)
		ma20 := movingAverageFromKlines(hist, 20)

		// ----- Today's signal -----
		signal := DailySignal{
			Date:  day.Date,
			Price: round2(day.Close),
			Score: score,
		}

		// Advance position state
		if pos.active {
			pos.advance(day.High)
		}

		if pos.active {
			shouldSell, sellReason := pos.shouldSell(day.Close, score, ma20, cfg)
			if shouldSell {
				// ---- SELL ----
				sellPriceNet := day.Close * (1 - cfg.commissionSell)
				returnPct := 0.0
				if pos.entryPrice > 0 {
					returnPct = (sellPriceNet/pos.entryPrice - 1) * 100
				}
				// Return position to cash: cashEquity += positionValue (already market-valued)
				cashEquity = cashEquity + positionValue
				positionValue = 0

				tradeCounter++
				tradeID := fmt.Sprintf("T%d", tradeCounter)
				trades = append(trades, StrategyTrade{
					TradeID:     tradeID,
					BuyDate:     pos.entryDate,
					BuyPrice:    pos.entryPrice,
					BuyScore:    pos.entryScore,
					SellDate:    day.Date,
					SellPrice:   round2(day.Close),
					SellReason:  sellReason,
					HoldingDays: pos.daysHeld,
					ReturnPct:   round2(returnPct),
					PositionPct: round2(pos.positionPct * 100),
				})

				signal.Action = "SELL"
				signal.PositionPct = 0
				signal.Reason = fmt.Sprintf("卖出: %s (持仓%d天, 收益%.2f%%)", sellReason, pos.daysHeld, returnPct)
				signal.TradeID = tradeID
				pos.reset()
			} else {
				// ---- HOLD (in position) ----
				// Market value = deployed * currentPrice / entryPrice
				positionValue = pos.deployedCapital * (day.Close / pos.entryPrice)

				signal.Action = "HOLD"
				signal.PositionPct = round2(pos.positionPct * 100)
				pnlPct := 0.0
				if pos.entryPrice > 0 {
					pnlPct = (day.Close/pos.entryPrice - 1) * 100
				}
				signal.Reason = fmt.Sprintf("持仓中 (入场%.2f, 持仓%d天, 浮盈%.2f%%, 止损%.2f)",
					pos.entryPrice, pos.daysHeld, pnlPct, round2(pos.highestPrice*(1-cfg.trailingStopPct)))
			}
		} else {
			// ---- Not in position ----
			positionValue = 0

			// Check buy conditions
			trendOK := ma20 > 0 && day.Close > ma20*0.98 // margin: might be slightly below
			canBuy := score >= int(cfg.buyThreshold) && trendOK

			if canBuy {
				// ---- BUY ----
				posPct := 0.33
				if score >= 90 {
					posPct = 1.0
				} else if score >= 80 {
					posPct = 0.66
				}
				buyPrice := day.Close * (1 + cfg.commissionBuy)
				tradeCounter++
				tradeID := fmt.Sprintf("T%d", tradeCounter)
				// Deploy capital: record deployed amount BEFORE reducing cash
				deployed := cashEquity * posPct
				pos.enter(day.Date, buyPrice, score, posPct, deployed)
				positionValue = deployed
				cashEquity = cashEquity - deployed

				signal.Action = "BUY"
				signal.PositionPct = round2(posPct * 100)
				signal.TradeID = tradeID
				signal.Reason = fmt.Sprintf("买入: 评分%d, 仓位%.0f%%, MA20=%.2f",
					score, posPct*100, round2(ma20))
			} else {
				// ---- HOLD (cash) ----
				signal.Action = "HOLD"
				signal.PositionPct = 0
				if score < int(cfg.buyThreshold) {
					signal.Reason = fmt.Sprintf("观望: 评分%d未达买入阈值%d", score, int(cfg.buyThreshold))
				} else if !trendOK {
					signal.Reason = fmt.Sprintf("观望: 评分%d达标但MA20(%.2f)趋势不满足", score, round2(ma20))
				} else {
					signal.Reason = "观望"
				}
			}
		}

		allSignals = append(allSignals, signal)

		// Equity curve
		totalEquity := cashEquity + positionValue
		equityCurve = append(equityCurve, EquityPoint{
			Date:      day.Date,
			Equity:    round4(totalEquity),
			InPosition: pos.active,
		})

		// Benchmark (buy-and-hold from start)
		benchEq := day.Close / benchEntry
		benchCurve = append(benchCurve, EquityPoint{
			Date:   day.Date,
			Equity: round4(benchEq),
		})
	}

	// Force-close any open position at last price
	if pos.active {
		lastDay := klines[len(klines)-1]
		sellPriceNet := lastDay.Close * (1 - cfg.commissionSell)
		returnPct := (sellPriceNet/pos.entryPrice - 1) * 100
		cashEquity = cashEquity + positionValue
		positionValue = 0

		tradeCounter++
		trades = append(trades, StrategyTrade{
			TradeID:     fmt.Sprintf("T%d", tradeCounter),
			BuyDate:     pos.entryDate,
			BuyPrice:    pos.entryPrice,
			BuyScore:    pos.entryScore,
			SellDate:    lastDay.Date,
			SellPrice:   round2(lastDay.Close),
			SellReason:  "回测结束，强制平仓",
			HoldingDays: pos.daysHeld,
			ReturnPct:   round2(returnPct),
			PositionPct: round2(pos.positionPct * 100),
		})
		pos.reset()
	}

	// ---- Aggregate stats ----
	stratReturn := (cashEquity - 1.0) * 100
	buyHoldReturn := 0.0
	if benchEntry > 0 && len(klines) > startIdx {
		buyHoldReturn = (klines[len(klines)-1].Close/benchEntry - 1) * 100
	}

	wins := 0
	totHoldingDays := 0
	for _, t := range trades {
		if t.ReturnPct > 0 {
			wins++
		}
		totHoldingDays += t.HoldingDays
	}
	winRate := 0.0
	avgHold := 0.0
	if len(trades) > 0 {
		winRate = float64(wins) / float64(len(trades)) * 100
		avgHold = float64(totHoldingDays) / float64(len(trades))
	}

	// Signal efficiency: % of days with active position
	activeDays := 0
	for _, ep := range equityCurve {
		if ep.InPosition {
			activeDays++
		}
	}
	sigEff := 0.0
	if len(equityCurve) > 0 {
		sigEff = float64(activeDays) / float64(len(equityCurve)) * 100
	}

	// Sharpe ratio (annualized, assuming daily returns, risk-free = 0)
	sharpe := computeSharpe(equityCurve)

	// Max drawdown
	maxDD := maxDrawdownFromEquity(equityCurve)

	// Truncate signals to last N days
	signals := allSignals
	if len(signals) > cfg.signalLookback {
		signals = signals[len(signals)-cfg.signalLookback:]
	}

	return StrategyResult{
		Code:               code,
		Days:               days,
		DailySignals:       signals,
		Trades:             trades,
		EquityCurve:        equityCurve,
		BenchmarkCurve:     benchCurve,
		StrategyReturnPct:  round2(stratReturn),
		BuyHoldReturnPct:   round2(buyHoldReturn),
		SharpeRatio:        round2(sharpe),
		MaxDrawdownPct:     round2(maxDD),
		WinRate:            round2(winRate),
		AvgHoldingDays:     round2(avgHold),
		TotalTrades:        len(trades),
		SignalEfficiency:   round2(sigEff),
	}
}

// ──────────────────────────
// Helpers
// ──────────────────────────

func movingAverageFromKlines(klines []models.KlinePoint, window int) float64 {
	if len(klines) < window || window <= 0 {
		return 0
	}
	subset := klines[len(klines)-window:]
	sum := 0.0
	for _, k := range subset {
		sum += k.Close
	}
	return sum / float64(window)
}

func maxDrawdownFromEquity(curve []EquityPoint) float64 {
	if len(curve) == 0 {
		return 0
	}
	peak := curve[0].Equity
	maxDD := 0.0
	for _, p := range curve {
		if p.Equity > peak {
			peak = p.Equity
		}
		if peak > 0 {
			dd := (peak - p.Equity) / peak * 100
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	return maxDD
}

// computeSharpe annualized from daily equity points.
func computeSharpe(curve []EquityPoint) float64 {
	if len(curve) < 2 {
		return 0
	}
	returns := make([]float64, 0, len(curve)-1)
	for i := 1; i < len(curve); i++ {
		if curve[i-1].Equity > 0 {
			// Daily log return
			r := math.Log(curve[i].Equity / curve[i-1].Equity)
			returns = append(returns, r)
		}
	}
	if len(returns) < 2 {
		return 0
	}
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		d := r - mean
		variance += d * d
	}
	variance /= float64(len(returns) - 1)

	std := math.Sqrt(variance)
	if std <= 0 {
		return 0
	}
	// Annualize: sqrt(252) ≈ 15.87
	return (mean / std) * math.Sqrt(252)
}
