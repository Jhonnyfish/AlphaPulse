package models

import "time"

// PortfolioPosition represents a single portfolio holding.
type PortfolioPosition struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CostPrice float64   `json:"cost_price"`
	Quantity  int       `json:"quantity"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PortfolioPositionEnriched is a position with live market data.
type PortfolioPositionEnriched struct {
	PortfolioPosition
	CurrentPrice float64 `json:"current_price"`
	MarketValue  float64 `json:"market_value"`
	TotalCost    float64 `json:"total_cost"`
	PnL          float64 `json:"pnl"`
	PnLPct       float64 `json:"pnl_pct"`
}

// --- Analytics models ---

// PortfolioAnalytics holds the full analytics payload.
type PortfolioAnalytics struct {
	ReturnCurve   []ReturnCurvePoint `json:"return_curve"`
	WinLoss       WinLossStats       `json:"win_loss"`
	MaxDrawdown   MaxDrawdownStats   `json:"max_drawdown"`
	Contributions []ContributionItem `json:"contributions"`
}

// ReturnCurvePoint is a single point on the cumulative return curve.
type ReturnCurvePoint struct {
	Date      string  `json:"date"`
	Value     float64 `json:"value"`
	ReturnPct float64 `json:"return_pct"`
}

// WinLossStats summarises winning vs losing positions.
type WinLossStats struct {
	WinCount   int     `json:"win_count"`
	LossCount  int     `json:"loss_count"`
	WinRate    float64 `json:"win_rate"`
	AvgGainPct float64 `json:"avg_gain_pct"`
	AvgLossPct float64 `json:"avg_loss_pct"`
}

// MaxDrawdownStats stores the maximum drawdown value and dates.
type MaxDrawdownStats struct {
	Value      float64 `json:"value"`
	PeakDate   string  `json:"peak_date"`
	TroughDate string  `json:"trough_date"`
}

// ContributionItem is a per-stock PnL contribution entry.
type ContributionItem struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	PnL        float64 `json:"pnl"`
	PnLPct     float64 `json:"pnl_pct"`
	ContribPct float64 `json:"contrib_pct"`
}

// --- Risk models ---

// PortfolioRisk holds the full risk analysis payload.
type PortfolioRisk struct {
	Stocks          []RiskStockDetail `json:"stocks"`
	Metrics         *RiskMetrics      `json:"metrics"`
	Recommendations []string          `json:"recommendations"`
	Message         string            `json:"message,omitempty"`
}

// RiskStockDetail contains per-stock risk metrics.
type RiskStockDetail struct {
	Code       string   `json:"code"`
	Name       string   `json:"name"`
	Sector     string   `json:"sector"`
	Sectors    []string `json:"sectors"`
	Volatility float64  `json:"volatility"`
	Beta       float64  `json:"beta"`
	RiskLevel  string   `json:"risk_level"`
	Weight     float64  `json:"weight"`
}

// RiskMetrics contains portfolio-level risk metrics.
type RiskMetrics struct {
	SectorDistribution   []SectorDistItem `json:"sector_distribution"`
	AvgCorrelation       float64          `json:"avg_correlation"`
	AnnualizedVolatility float64          `json:"annualized_volatility"`
	MaxDrawdown          float64          `json:"max_drawdown"`
	Beta                 float64          `json:"beta"`
	DiversificationScore int              `json:"diversification_score"`
	RiskLevel            string           `json:"risk_level"`
}

// SectorDistItem is one entry in the sector distribution.
type SectorDistItem struct {
	Sector string  `json:"sector"`
	Count  int     `json:"count"`
	Pct    float64 `json:"pct"`
}
