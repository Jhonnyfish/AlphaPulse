package models

import "time"

type Quote struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Open          float64 `json:"open"`
	PrevClose     float64 `json:"prev_close"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	UpdatedAt     string  `json:"updated_at"`
}

type KlinePoint struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volume float64 `json:"volume"`
	Amount float64 `json:"amount"`
}

type Sector struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
}

type OverviewIndex struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	AdvanceCount  int     `json:"advance_count"`
	DeclineCount  int     `json:"decline_count"`
	FlatCount     int     `json:"flat_count"`
}

type MarketOverview struct {
	AdvanceCount int             `json:"advance_count"`
	DeclineCount int             `json:"decline_count"`
	FlatCount    int             `json:"flat_count"`
	Indices      []OverviewIndex `json:"indices"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type TopMover struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Volume        float64 `json:"volume"`   // 成交量（手）
	Amount        float64 `json:"amount"`   // 成交额（元）
	Amplitude     float64 `json:"amplitude"` // 振幅%
}

type NewsItem struct {
	Code        string    `json:"code,omitempty"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary,omitempty"`
	Source      string    `json:"source,omitempty"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
}

// MarketSession represents the current trading session status
type MarketSession struct {
	Session         string `json:"session"`           // 盘前/集合竞价/交易中/午间休市/已收盘/休市
	SessionEN       string `json:"session_en"`        // pre_market/call_auction/trading/lunch_break/closed
	IsTrading       bool   `json:"is_trading"`        // true if trading or call_auction
	RefreshInterval int    `json:"refresh_interval"`  // recommended refresh interval in seconds
	NextSession     string `json:"next_session"`      // info about next session
	ServerTime      string `json:"server_time"`       // server time ISO format
	Weekday         int    `json:"weekday"`           // 0=Mon, 6=Sun
}

// TrendStock represents a single stock/index in the market trends view
type TrendStock struct {
	Code       string       `json:"code"`
	Name       string       `json:"name"`
	Price      *float64     `json:"price,omitempty"`
	Change1D   *float64     `json:"change_1d"`
	Change5D   *float64     `json:"change_5d"`
	Change20D  *float64     `json:"change_20d"`
	Change30D  *float64     `json:"change_30d"`
	KlineData  []KlinePoint `json:"kline_data"`
}

// TrendPortfolio represents portfolio-level aggregated trend data
type TrendPortfolio struct {
	Change1D      *float64           `json:"change_1d"`
	Change5D      *float64           `json:"change_5d"`
	Change20D     *float64           `json:"change_20d"`
	Change30D     *float64           `json:"change_30d"`
	DailyReturns  []DailyReturn      `json:"daily_returns"`
}

// DailyReturn represents one day's portfolio vs benchmark return
type DailyReturn struct {
	Date       string   `json:"date"`
	Portfolio  *float64 `json:"portfolio"`
	Benchmark  float64  `json:"benchmark"`
}

// MarketTrends is the full response for the trends API
type MarketTrends struct {
	Indices         []TrendStock  `json:"indices"`
	WatchlistStocks []TrendStock  `json:"watchlist_stocks"`
	Portfolio       TrendPortfolio `json:"portfolio"`
	FetchedAt       string        `json:"fetched_at"`
}
