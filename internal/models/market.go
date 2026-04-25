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
	Volume        float64 `json:"volume"`
	Turnover      float64 `json:"turnover"`
	PE            float64 `json:"pe"`
	PB            float64 `json:"pb"`
	TotalMV       float64 `json:"total_mv"`
	Amplitude     float64 `json:"amplitude"`
	LimitUp       float64 `json:"limit_up"`
	LimitDown     float64 `json:"limit_down"`
	OuterVol      float64 `json:"outer_vol"`
	InnerVol      float64 `json:"inner_vol"`
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
	Indices         []TrendStock   `json:"indices"`
	WatchlistStocks []TrendStock   `json:"watchlist_stocks"`
	Portfolio       TrendPortfolio `json:"portfolio"`
	FetchedAt       string         `json:"fetched_at"`
}

// IndexQuote represents a major market index quote from Tencent API
type IndexQuote struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	PrevClose     float64 `json:"prev_close"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_pct"`
	Volume        int64   `json:"volume"`
	Amount        float64 `json:"amount"`
}

// MarketBreadth represents market-wide advance/decline statistics
type MarketBreadth struct {
	UpCount        int     `json:"up_count"`
	DownCount      int     `json:"down_count"`
	FlatCount      int     `json:"flat_count"`
	LimitUp        int     `json:"limit_up"`
	LimitDown      int     `json:"limit_down"`
	Sentiment      string  `json:"sentiment"`
	SentimentRatio float64 `json:"sentiment_ratio"`
}

// BreadthDistributionItem represents a single bucket in the advance/decline distribution
type BreadthDistributionItem struct {
	Range string `json:"range"`
	Count int    `json:"count"`
}

// VolumeStats represents trading volume broken by direction
type VolumeStats struct {
	UpVolume   int64 `json:"up_volume"`
	DownVolume int64 `json:"down_volume"`
	FlatVolume int64 `json:"flat_volume"`
}

// MarketBreadthDetail is the full response for /api/market-breadth
type MarketBreadthDetail struct {
	Advancing      int                        `json:"advancing"`
	Declining      int                        `json:"declining"`
	Flat           int                        `json:"flat"`
	LimitUp        int                        `json:"limit_up"`
	LimitDown      int                        `json:"limit_down"`
	ADRatio        float64                    `json:"ad_ratio"`
	BreadthThrust  float64                    `json:"breadth_thrust"`
	LimitRatio     float64                    `json:"limit_ratio"`
	Total          int                        `json:"total"`
	VolumeStats    VolumeStats                `json:"volume_stats"`
	Distribution   []BreadthDistributionItem  `json:"distribution"`
	Timestamp      string                     `json:"timestamp"`
}

// SectorVolume represents a sector's volume and change data
type SectorVolume struct {
	Name       string  `json:"name"`
	Volume     int64   `json:"volume"`
	ChangePct  float64 `json:"change_pct"`
}

// MarketSentimentResponse is the full response for /api/market-sentiment
type MarketSentimentResponse struct {
	OK              bool            `json:"ok"`
	FearGreedIndex  int             `json:"fear_greed_index"`
	FearGreedLabel  string          `json:"fear_greed_label"`
	UpCount         int             `json:"up_count"`
	DownCount       int             `json:"down_count"`
	FlatCount       int             `json:"flat_count"`
	TotalCount      int             `json:"total_count"`
	LimitUp         int             `json:"limit_up"`
	LimitDown       int             `json:"limit_down"`
	VolumeToday     int64           `json:"volume_today"`
	VolumeAvg5D     int64           `json:"volume_avg_5d"`
	SectorVolumes   []SectorVolume  `json:"sector_volumes"`
	Temperature     int             `json:"temperature"`
	ServerTime      string          `json:"server_time"`
}

// MarketOverviewResponse is the full response for /api/market-overview
type MarketOverviewResponse struct {
	OK      bool           `json:"ok"`
	Indices []IndexQuote   `json:"indices"`
	Market  MarketBreadth  `json:"market"`
}

// HotConcept represents a hot concept sector
type HotConcept struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	ChangePct  float64 `json:"change_pct"`
	Change     float64 `json:"change"`
	RiseCount  int     `json:"rise_count"`
	FallCount  int     `json:"fall_count"`
	LeaderStock string `json:"leader_stock"`
}

// SectorRotationItem represents a sector in the rotation analysis
type SectorRotationItem struct {
	Code           string   `json:"code"`
	Name           string   `json:"name"`
	ChangePct      float64  `json:"change_pct"`
	Price          float64  `json:"price"`
	RisingCount    int      `json:"rising_count"`
	FallingCount   int      `json:"falling_count"`
	BreadthRatio   float64  `json:"breadth_ratio"`
	NetFlow        float64  `json:"net_flow"`
	StrengthScore  float64  `json:"strength_score"`
	WatchlistMatch bool     `json:"watchlist_match"`
	WatchlistStocks []string `json:"watchlist_stocks"`
}

// SectorRotationSummary is the summary for sector rotation analysis
type SectorRotationSummary struct {
	AvgBreadth   float64 `json:"avg_breadth"`
	TotalNetFlow float64 `json:"total_net_flow"`
	StrongCount  int     `json:"strong_count"`
	WeakCount    int     `json:"weak_count"`
}

// SectorRotationResponse is the full response for /api/sector-rotation
type SectorRotationResponse struct {
	OK      bool                   `json:"ok"`
	Sectors []SectorRotationItem   `json:"sectors"`
	Summary SectorRotationSummary  `json:"summary"`
	Cached  bool                   `json:"cached,omitempty"`
}

// SectorRotationSnapshot represents a historical snapshot of sector rotation
type SectorRotationSnapshot struct {
	Timestamp string               `json:"timestamp"`
	Sectors   []SectorRotationItem `json:"sectors"`
}

// SectorRotationHistoryResponse is the full response for /api/sector-rotation-history
type SectorRotationHistoryResponse struct {
	OK        bool                      `json:"ok"`
	Snapshots []SectorRotationSnapshot  `json:"snapshots"`
	Total     int                       `json:"total"`
}
