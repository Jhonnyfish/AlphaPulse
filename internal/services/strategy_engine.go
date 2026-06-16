package services

import (
	"context"
	"fmt"
	"math"
	"strings"

	"alphapulse/internal/models"
)

// ──────────────────────────────────────────────
// Strategy engine: dynamic position management
// ──────────────────────────────────────────────
//
// Replaces the naive fixed-T+3 backtest with a realistic strategy:
//   - Default to holding a full position, matching the benchmark's market exposure
//   - Reduce exposure only when downside risk is clearly elevated
//   - Restore exposure after risk cools down, with a cooldown to avoid churn
//
// Produces daily actionable signals with reasoning and a full
// walk-forward backtest including benchmark comparison.

// ──────────────────────────
// Core types
// ──────────────────────────

// DailySignal is a single day's trading recommendation.
type DailySignal struct {
	Date        string  `json:"date"`
	Action      string  `json:"action"`             // BUY / SELL / HOLD
	Price       float64 `json:"price"`              // actionable price (today's close)
	PositionPct float64 `json:"position_pct"`       // 0.0 ~ 100.0, current position after this action
	Score       int     `json:"score"`              // composite score that day
	Reason      string  `json:"reason"`             // human-readable rationale
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
	Code           string          `json:"code"`
	Name           string          `json:"name,omitempty"`
	Days           int             `json:"days"`
	DailySignals   []DailySignal   `json:"daily_signals,omitempty"` // last 60 days of signals
	Trades         []StrategyTrade `json:"trades"`
	EquityCurve    []EquityPoint   `json:"equity_curve"`
	BenchmarkCurve []EquityPoint   `json:"benchmark_curve"`

	// Summary stats
	StrategyReturnPct float64 `json:"strategy_return_pct"`
	BuyHoldReturnPct  float64 `json:"buy_hold_return_pct"`
	SharpeRatio       float64 `json:"sharpe_ratio"`
	MaxDrawdownPct    float64 `json:"max_drawdown_pct"`
	WinRate           float64 `json:"win_rate"`
	AvgHoldingDays    float64 `json:"avg_holding_days"`
	TotalTrades       int     `json:"total_trades"`
	SignalEfficiency  float64 `json:"signal_efficiency"` // % of days with active position

	Error string `json:"error,omitempty"`
}

// EquityPoint is one day on the equity curve.
type EquityPoint struct {
	Date       string  `json:"date"`
	Equity     float64 `json:"equity"` // starting from 1.0
	InPosition bool    `json:"in_position"`
}

// ──────────────────────────
// Config
// ──────────────────────────

type strategyConfig struct {
	buyThreshold      float64 // kept for legacy score display and tests
	exitScoreFloor    float64 // kept for legacy score display and tests
	trailingStopPct   float64 // kept for legacy signal wording
	maExitPeriod      int     // moving average period used by risk scoring context
	maxHoldDays       int     // kept for legacy compatibility
	warmupDays        int     // minimum bars to compute indicators
	commissionBuy     float64 // 0.0005
	commissionSell    float64 // 0.0010  (佣金 0.05% + 印花税 0.05%)
	signalLookback    int     // how many recent signals to return
	rebalanceCooldown int     // minimum days between non-emergency rebalances
	rebalanceBand     float64 // ignore small target/current drifts
}

var defaultStrategyConfig = strategyConfig{
	buyThreshold:      60,
	exitScoreFloor:    30,
	trailingStopPct:   0.12,
	maExitPeriod:      20,
	maxHoldDays:       60,
	warmupDays:        20,
	commissionBuy:     0.0005,
	commissionSell:    0.0010,
	signalLookback:    60,
	rebalanceCooldown: 5,
	rebalanceBand:     0.05,
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

	// Score reversal — exit fast when quality deteriorates
	if score < int(cfg.exitScoreFloor) {
		return true, fmt.Sprintf("评分跌至%d, 低于阈值%d", score, int(cfg.exitScoreFloor))
	}

	// MA exit with 3% buffer — only trigger if decisively below MA
	if ma20 > 0 && price < ma20*0.97 {
		return true, fmt.Sprintf("跌破MA%d(%.2f), 跌幅%.1f%%", cfg.maExitPeriod, ma20, (1-price/ma20)*100)
	}

	// Trailing stop — only activates when in profit, protects gains
	if p.entryPrice > 0 && p.highestPrice > p.entryPrice {
		trailPrice := p.highestPrice * (1 - cfg.trailingStopPct)
		if price <= trailPrice {
			return true, fmt.Sprintf("trailing stop: 最高价%.2f, 回撤%.1f%%, 止损价%.2f",
				p.highestPrice, (1-price/p.highestPrice)*100, trailPrice)
		}
	}

	// Time stop — let winners run longer
	if p.daysHeld >= cfg.maxHoldDays {
		return true, fmt.Sprintf("持仓已达%d天, 触及时间止损", cfg.maxHoldDays)
	}

	return false, ""
}

// ──────────────────────────
// Main engine
// ──────────────────────────

type strategyKlineFetcher interface {
	FetchKline(ctx context.Context, code string, days int) ([]models.KlinePoint, error)
}

// RunStrategyBacktest runs a strategy backtest with dynamic position management.
// Returns daily signals, trades, equity curves, and comparison metrics.
func RunStrategyBacktest(
	ctx context.Context,
	eastMoney *EastMoneyService,
	code string,
	days int,
) StrategyResult {
	return RunStrategyBacktestWithStrategy(ctx, eastMoney, code, days, "balanced")
}

// RunStrategyBacktestWithStrategy runs the selected discipline profile.
func RunStrategyBacktestWithStrategy(
	ctx context.Context,
	fetcher strategyKlineFetcher,
	code string,
	days int,
	strategyID string,
) StrategyResult {
	code = NormalizeCode(code)
	cfg := defaultStrategyConfig

	needCount := days + cfg.warmupDays*2
	if needCount < 120 {
		needCount = 120
	}

	klines, err := fetcher.FetchKline(ctx, code, needCount)
	if err != nil {
		return StrategyResult{Code: code, Days: days, Error: err.Error()}
	}
	if len(klines) < days+cfg.warmupDays {
		return StrategyResult{
			Code:  code,
			Days:  days,
			Error: fmt.Sprintf("K线数据不足: 需要至少%d条, 当前%d条", days+cfg.warmupDays, len(klines)),
		}
	}
	return runStrategyWithProfile(ctx, code, days, klines, strategyID)
}

// runStrategy is the core backtest logic, separated so tests can inject klines directly.
func runStrategy(_ context.Context, code string, days int, klines []models.KlinePoint) StrategyResult {
	return runStrategyWithProfile(context.Background(), code, days, klines, "balanced")
}

func runStrategyWithProfile(_ context.Context, code string, days int, klines []models.KlinePoint, strategyID string) StrategyResult {
	cfg := defaultStrategyConfig
	strategyID = normalizeStrategyID(strategyID)
	strategyName := strategyDisplayName(strategyID)

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
		trades           []StrategyTrade
		allSignals       []DailySignal
		equityCurve      []EquityPoint
		benchCurve       []EquityPoint
		tradeCounter     int
		cashEquity       float64 = 1.0
		shares           float64
		entryDate        string
		entryPrice       float64
		entryScore       int
		highestPrice     float64
		daysHeld         int
		lastTargetPct    float64 = 1.0
		lastRebalanceIdx int     = -999
	)

	benchEntry := klines[startIdx].Close
	if benchEntry <= 0 {
		benchEntry = 1
	}

	for i := startIdx; i < len(klines); i++ {
		day := klines[i]
		hist := klines[:i+1] // data available before today's close
		score, _ := ScoreFromKlines(hist)
		riskScore, _ := RiskFromKlines(hist)
		ma20 := movingAverageFromKlines(hist, 20)

		// ----- Today's signal -----
		signal := DailySignal{
			Date:  day.Date,
			Price: round2(day.Close),
			Score: score,
		}

		if shares > 0 {
			daysHeld++
			if day.High > highestPrice {
				highestPrice = day.High
			}
		}

		equityBefore := cashEquity + shares*day.Close
		if equityBefore <= 0 {
			equityBefore = 1
		}
		currentPct := 0.0
		if shares > 0 {
			currentPct = shares * day.Close / equityBefore
		}

		targetPct := strategyTargetPosition(strategyID, hist, riskScore, score, lastTargetPct)
		emergencyReduce := targetPct < lastTargetPct && riskScore >= 75
		cooldownOK := i-lastRebalanceIdx >= cfg.rebalanceCooldown
		if !cooldownOK && !emergencyReduce {
			targetPct = lastTargetPct
		}

		deltaPct := targetPct - currentPct
		if math.Abs(deltaPct) < cfg.rebalanceBand {
			targetPct = currentPct
			deltaPct = 0
		}

		if deltaPct > 0 {
			buyValue := equityBefore*targetPct - shares*day.Close
			maxBuyValue := cashEquity / (1 + cfg.commissionBuy)
			if buyValue > maxBuyValue {
				buyValue = maxBuyValue
			}
			if buyValue > 0 {
				oldShares := shares
				buyShares := buyValue / day.Close
				shares += buyShares
				cashEquity -= buyValue * (1 + cfg.commissionBuy)
				if oldShares <= 0 {
					entryDate = day.Date
					entryPrice = day.Close * (1 + cfg.commissionBuy)
					entryScore = score
					highestPrice = day.High
					daysHeld = 0
				} else if entryPrice > 0 {
					entryPrice = (entryPrice*oldShares + day.Close*(1+cfg.commissionBuy)*buyShares) / shares
				}
				lastTargetPct = targetPct
				lastRebalanceIdx = i

				signal.Action = "BUY"
				signal.TradeID = fmt.Sprintf("R%d", len(allSignals)+1)
				signal.Reason = fmt.Sprintf("%s恢复仓位: 风险分%d降温, 评分%d, 目标仓位%.0f%%, MA20=%.2f",
					strategyName, riskScore, score, targetPct*100, round2(ma20))
			}
		} else if deltaPct < 0 {
			sellValue := shares*day.Close - equityBefore*targetPct
			currentValue := shares * day.Close
			if sellValue > currentValue {
				sellValue = currentValue
			}
			if sellValue > 0 && day.Close > 0 {
				soldPct := 0.0
				if equityBefore > 0 {
					soldPct = sellValue / equityBefore
				}
				sellShares := sellValue / day.Close
				shares -= sellShares
				cashEquity += sellValue * (1 - cfg.commissionSell)
				if shares < 1e-9 {
					shares = 0
				}

				returnPct := 0.0
				if entryPrice > 0 {
					returnPct = (day.Close*(1-cfg.commissionSell)/entryPrice - 1) * 100
				}
				soldHoldingDays := daysHeld
				tradeCounter++
				tradeID := fmt.Sprintf("T%d", tradeCounter)
				reason := fmt.Sprintf("风险分%d升高, 降至目标仓位%.0f%%", riskScore, targetPct*100)
				trades = append(trades, StrategyTrade{
					TradeID:     tradeID,
					BuyDate:     entryDate,
					BuyPrice:    round2(entryPrice),
					BuyScore:    entryScore,
					SellDate:    day.Date,
					SellPrice:   round2(day.Close),
					SellReason:  reason,
					HoldingDays: soldHoldingDays,
					ReturnPct:   round2(returnPct),
					PositionPct: round2(soldPct * 100),
				})

				if shares == 0 {
					entryDate = ""
					entryPrice = 0
					entryScore = 0
					highestPrice = 0
					daysHeld = 0
				}
				lastTargetPct = targetPct
				lastRebalanceIdx = i

				signal.Action = "SELL"
				signal.TradeID = tradeID
				signal.Reason = fmt.Sprintf("%s降仓: %s (本次降仓%.0f%%, 持仓%d天, 收益%.2f%%)",
					strategyName, reason, soldPct*100, soldHoldingDays, returnPct)
			}
		} else {
			signal.Action = "HOLD"
			if shares > 0 {
				pnlPct := 0.0
				if entryPrice > 0 {
					pnlPct = (day.Close/entryPrice - 1) * 100
				}
				signal.Reason = fmt.Sprintf("%s持有: 风险分%d, 评分%d, 持仓%.0f%%, 持仓%d天, 浮盈%.2f%%",
					strategyName, riskScore, score, currentPct*100, daysHeld, pnlPct)
			} else {
				signal.Reason = fmt.Sprintf("%s空仓: 风险分%d仍高, 等待风险降温", strategyName, riskScore)
			}
		}

		equityAfter := cashEquity + shares*day.Close
		positionPct := 0.0
		if equityAfter > 0 && shares > 0 {
			positionPct = shares * day.Close / equityAfter
		}
		signal.PositionPct = round2(positionPct * 100)

		allSignals = append(allSignals, signal)

		// Equity curve
		equityCurve = append(equityCurve, EquityPoint{
			Date:       day.Date,
			Equity:     round4(equityAfter),
			InPosition: shares > 0,
		})

		// Benchmark (buy-and-hold from start)
		benchEq := day.Close / benchEntry
		benchCurve = append(benchCurve, EquityPoint{
			Date:   day.Date,
			Equity: round4(benchEq),
		})
	}

	// Force-close any open position at last price
	if shares > 0 {
		lastDay := klines[len(klines)-1]
		positionValue := shares * lastDay.Close
		sellPriceNet := lastDay.Close * (1 - cfg.commissionSell)
		returnPct := 0.0
		if entryPrice > 0 {
			returnPct = (sellPriceNet/entryPrice - 1) * 100
		}
		cashEquity += positionValue * (1 - cfg.commissionSell)

		tradeCounter++
		trades = append(trades, StrategyTrade{
			TradeID:     fmt.Sprintf("T%d", tradeCounter),
			BuyDate:     entryDate,
			BuyPrice:    round2(entryPrice),
			BuyScore:    entryScore,
			SellDate:    lastDay.Date,
			SellPrice:   round2(lastDay.Close),
			SellReason:  "回测结束，强制平仓",
			HoldingDays: daysHeld,
			ReturnPct:   round2(returnPct),
			PositionPct: round2(lastTargetPct * 100),
		})
		shares = 0
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
		Code:              code,
		Name:              strategyName,
		Days:              days,
		DailySignals:      signals,
		Trades:            trades,
		EquityCurve:       equityCurve,
		BenchmarkCurve:    benchCurve,
		StrategyReturnPct: round2(stratReturn),
		BuyHoldReturnPct:  round2(buyHoldReturn),
		SharpeRatio:       round2(sharpe),
		MaxDrawdownPct:    round2(maxDD),
		WinRate:           round2(winRate),
		AvgHoldingDays:    round2(avgHold),
		TotalTrades:       len(trades),
		SignalEfficiency:  round2(sigEff),
	}
}

// ──────────────────────────
// Helpers
// ──────────────────────────

func defensiveTargetPosition(riskScore int, alphaScore int, currentTarget float64) float64 {
	risk := float64(riskScore)

	// Reduce exposure quickly as downside risk rises.
	if risk >= 75 {
		return 0
	}
	if risk >= 65 {
		return math.Min(currentTarget, 0.20)
	}
	if risk >= 50 {
		return math.Min(currentTarget, 0.50)
	}
	if risk >= 35 {
		return math.Min(currentTarget, 0.70)
	}

	// Restore exposure more slowly than it was reduced.
	if risk < 25 && alphaScore >= 50 {
		return 1.0
	}
	if risk < 35 && alphaScore >= 50 {
		return math.Max(currentTarget, 0.70)
	}
	if risk < 45 && alphaScore >= 55 {
		return math.Max(currentTarget, 0.50)
	}
	return currentTarget
}

func strategyTargetPosition(strategyID string, klines []models.KlinePoint, riskScore int, alphaScore int, currentTarget float64) float64 {
	state := selectedStockState(klines, riskScore)
	baseTarget := selectedStockTargetPct(alphaScore, riskScore, state)

	switch normalizeStrategyID(strategyID) {
	case "conservative":
		return float64(clampTarget(baseTarget-20, 0, 80)) / 100
	case "aggressive":
		return float64(aggressiveTarget(baseTarget, alphaScore, riskScore, state)) / 100
	case "rebound":
		return float64(reboundTarget(baseTarget, state)) / 100
	case "exit_weak":
		return float64(exitWeakTarget(alphaScore, riskScore, state)) / 100
	case "balanced":
		return float64(baseTarget) / 100
	default:
		return defensiveTargetPosition(riskScore, alphaScore, currentTarget)
	}
}

func normalizeStrategyID(id string) string {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "conservative", "balanced", "aggressive", "rebound", "exit_weak":
		return strings.ToLower(strings.TrimSpace(id))
	default:
		return "balanced"
	}
}

func strategyDisplayName(id string) string {
	switch normalizeStrategyID(id) {
	case "conservative":
		return "保守防守"
	case "aggressive":
		return "进攻持有"
	case "rebound":
		return "反弹恢复"
	case "exit_weak":
		return "弱势退出"
	default:
		return "均衡持有"
	}
}

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
