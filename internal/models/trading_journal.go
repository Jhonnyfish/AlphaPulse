package models

import "time"

// TradingJournalEntry represents a single trade record.
type TradingJournalEntry struct {
	ID            string    `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Price         float64   `json:"price"`
	Quantity      int       `json:"quantity"`
	Fees          float64   `json:"fees"`
	Date          string    `json:"date"`
	Notes         string    `json:"notes,omitempty"`
	StrategyLabel string    `json:"strategy_label,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// TradingJournalSummary holds aggregate trade statistics.
type TradingJournalSummary struct {
	TotalTrades    int     `json:"total_trades"`
	BuyCount       int     `json:"buy_count"`
	SellCount      int     `json:"sell_count"`
	TotalBuyAmount float64 `json:"total_buy_amount"`
	TotalSellAmount float64 `json:"total_sell_amount"`
	TotalFees      float64 `json:"total_fees"`
	RealizedPnL    float64 `json:"realized_pnl"`
}

// TradingJournalPosition represents a current holding computed from buy-sell history.
type TradingJournalPosition struct {
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Quantity     int     `json:"quantity"`
	AvgCost      float64 `json:"avg_cost"`
	TotalCost    float64 `json:"total_cost"`
	TotalBuys    int     `json:"total_buys"`
	TotalSells   int     `json:"total_sells"`
	RealizedPnL  float64 `json:"realized_pnl"`
}

// TradingJournalListResponse is the response for GET /api/trading-journal.
type TradingJournalListResponse struct {
	Items     []TradingJournalEntry    `json:"items"`
	Summary   TradingJournalSummary    `json:"summary"`
	Positions []TradingJournalPosition `json:"positions"`
}

// --- Stats models ---

// TradingJournalStats is the full stats payload for GET /api/trading-journal/stats.
type TradingJournalStats struct {
	Summary             StatsSummary        `json:"summary"`
	MonthlyPnL          []MonthlyPnL        `json:"monthly_pnl"`
	CumulativePnL       []CumulativePnL     `json:"cumulative_pnl"`
	ReturnDistribution  ReturnDistribution  `json:"return_distribution"`
	Activity            TradingActivity     `json:"activity"`
	TopPerformers       []PerStockWinLoss   `json:"top_performers"`
	BottomPerformers    []PerStockWinLoss   `json:"bottom_performers"`
	WinLoss             WinLossOverall      `json:"win_loss"`
	SellResults         []SellResult        `json:"sell_results"`
}

// StatsSummary holds high-level stats.
type StatsSummary struct {
	TotalRealizedPnL float64 `json:"total_realized_pnl"`
	WinRate          float64 `json:"win_rate"`
	AvgMonthlyTrades float64 `json:"avg_monthly_trades"`
	MaxSingleProfit  float64 `json:"max_single_profit"`
}

// MonthlyPnL holds PnL for a calendar month.
type MonthlyPnL struct {
	Month string  `json:"month"`
	PnL   float64 `json:"pnl"`
}

// CumulativePnL holds running cumulative PnL by month.
type CumulativePnL struct {
	Month     string  `json:"month"`
	CumPnL    float64 `json:"cum_pnl"`
}

// ReturnDistribution groups sell results into buckets.
type ReturnDistribution struct {
	LessThanNeg10 int `json:"less_than_neg10"`
	Neg10ToNeg5   int `json:"neg10_to_neg5"`
	Neg5To0       int `json:"neg5_to_0"`
	ZeroTo5       int `json:"zero_to_5"`
	FiveTo10      int `json:"5_to_10"`
	GreaterThan10 int `json:"greater_than_10"`
}

// TradingActivity holds activity breakdown.
type TradingActivity struct {
	ByWeekday map[string]int `json:"by_weekday"`
	ByHour    map[string]int `json:"by_hour"`
}

// WinLossOverall holds overall and per-stock win/loss stats.
type WinLossOverall struct {
	TotalTrades int              `json:"total_trades"`
	Wins        int              `json:"wins"`
	Losses      int              `json:"losses"`
	WinRate     float64          `json:"win_rate"`
	PerStock    []PerStockWinLoss `json:"per_stock"`
}

// PerStockWinLoss holds win/loss stats for a single stock.
type PerStockWinLoss struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Trades   int     `json:"trades"`
	Wins     int     `json:"wins"`
	Losses   int     `json:"losses"`
	WinRate  float64 `json:"win_rate"`
	TotalPnL float64 `json:"total_pnl"`
}

// SellResult holds the result of a single sell trade.
type SellResult struct {
	ID            string  `json:"id"`
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Date          string  `json:"date"`
	SellPrice     float64 `json:"sell_price"`
	SellQty       int     `json:"sell_qty"`
	AvgCost       float64 `json:"avg_cost"`
	RealizedPnL   float64 `json:"realized_pnl"`
	ReturnPct     float64 `json:"return_pct"`
	StrategyLabel string  `json:"strategy_label,omitempty"`
}

// --- Calendar models ---

// TradingJournalCalendarResponse is the response for GET /api/trading-journal/calendar.
type TradingJournalCalendarResponse struct {
	Daily   []TradingCalendarDay  `json:"daily"`
	Summary TradingCalendarSummary `json:"summary"`
}

// TradingCalendarDay holds daily P&L data for the calendar.
type TradingCalendarDay struct {
	Date  string  `json:"date"`
	PnL   float64 `json:"pnl"`
	Trades int    `json:"trades"`
}

// TradingCalendarSummary holds calendar summary.
type TradingCalendarSummary struct {
	TotalDaysTraded int     `json:"total_days_traded"`
	TotalRealizedPnL float64 `json:"total_realized_pnl"`
	BestDay         float64 `json:"best_day"`
	WorstDay        float64 `json:"worst_day"`
	AvgDailyPnL     float64 `json:"avg_daily_pnl"`
	ProfitableDays  int     `json:"profitable_days"`
	LosingDays      int     `json:"losing_days"`
}

// --- Strategy Eval models ---

// StrategyEval is per-strategy evaluation.
type StrategyEval struct {
	StrategyLabel  string  `json:"strategy_label"`
	TradeCount     int     `json:"trade_count"`
	Wins           int     `json:"wins"`
	Losses         int     `json:"losses"`
	WinRate        float64 `json:"win_rate"`
	TotalPnL       float64 `json:"total_pnl"`
	AvgPnL         float64 `json:"avg_pnl"`
	AvgReturnPct   float64 `json:"avg_return_pct"`
	MaxProfit      float64 `json:"max_profit"`
	MaxLoss        float64 `json:"max_loss"`
	MaxDrawdown    float64 `json:"max_drawdown"`
}

// StrategyEvalOverall wraps the strategy eval list.
type StrategyEvalOverall struct {
	Strategies []StrategyEval `json:"strategies"`
}
