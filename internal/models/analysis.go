package models

import "time"

// MoneyFlowDay represents one day of money flow data from EastMoney
type MoneyFlowDay struct {
	Date         string  `json:"date"`
	MainNet      float64 `json:"main_net"`
	SmallNet     float64 `json:"small_net"`
	MiddleNet    float64 `json:"middle_net"`
	BigNet       float64 `json:"big_net"`
	HugeNet      float64 `json:"huge_net"`
	MainNetPct   float64 `json:"main_net_pct"`
	SmallNetPct  float64 `json:"small_net_pct"`
	MiddleNetPct float64 `json:"middle_net_pct"`
	BigNetPct    float64 `json:"big_net_pct"`
	HugeNetPct   float64 `json:"huge_net_pct"`
}

// StockSector represents a sector/industry a stock belongs to
type StockSector struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type SectorMember struct {
	Name      string  `json:"name"`
	Code      string  `json:"code"`
	ChangePct float64 `json:"change_pct"`
	PE        float64 `json:"pe"`
	PB        float64 `json:"pb"`
	Amount    float64 `json:"amount"`
}

type SectorCompareResult struct {
	Code        string         `json:"code"`
	SectorName  string         `json:"sector_name"`
	BoardCode   string         `json:"board_code"`
	Top5        []SectorMember `json:"top5"`
	CurrentRank int            `json:"current_rank"`
	TotalCount  int            `json:"total_count"`
}

// Announcement represents a stock announcement
type Announcement struct {
	Title       string    `json:"title"`
	Date        string    `json:"date"`
	URL         string    `json:"url"`
	ArtCode     string    `json:"art_code"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
}

// MACDResult holds MACD indicator values
type MACDResult struct {
	DIF       float64   `json:"dif"`
	DEA       float64   `json:"dea"`
	Hist      float64   `json:"hist"`
	Signal    string    `json:"signal"`
	HistTrend string    `json:"hist_trend"`
	HistLast3 []float64 `json:"hist_last3"`
}

// BollingerResult holds Bollinger Bands values
type BollingerResult struct {
	Upper     float64 `json:"upper"`
	Mid       float64 `json:"mid"`
	Lower     float64 `json:"lower"`
	Bandwidth float64 `json:"bandwidth"`
}

// KDJResult holds KDJ indicator values
type KDJResult struct {
	K      float64 `json:"k"`
	D      float64 `json:"d"`
	J      float64 `json:"j"`
	Signal string  `json:"signal"`
}

// OBVResult holds OBV indicator values
type OBVResult struct {
	Recent5D []float64 `json:"recent_5d"`
	Trend    string    `json:"trend"`
}

// ---- 8 Analysis Dimensions ----

// OrderFlowAnalysis removed - Tencent inner/outer volume data not available via Tushare

type VolumePriceAnalysis struct {
	TodayChangePct     float64 `json:"today_change_pct"`
	TodayVolume        float64 `json:"today_volume"`
	AvgVolume5D        float64 `json:"avg_volume_5d"`
	VolumeRatio        float64 `json:"volume_ratio"`
	Turnover           float64 `json:"turnover"`
	TurnoverLevel      string  `json:"turnover_level"`
	PriceVolumeHarmony string  `json:"price_volume_harmony"`
	Verdict            string  `json:"verdict"`
}

type ValuationAnalysis struct {
	PE      float64 `json:"pe"`
	PELevel string  `json:"pe_level"`
	PB      float64 `json:"pb"`
	PBLevel string  `json:"pb_level"`
	TotalMV float64 `json:"total_mv"`
	MVLevel string  `json:"mv_level"`
	Verdict string  `json:"verdict"`
}

type VolatilityAnalysis struct {
	Amplitude           float64 `json:"amplitude"`
	AmplitudeLevel      string  `json:"amplitude_level"`
	DistanceToLimitUp   float64 `json:"distance_to_limit_up"`
	DistanceToLimitDown float64 `json:"distance_to_limit_down"`
	Verdict             string  `json:"verdict"`
}

type MoneyFlowAnalysis struct {
	TodayMainNet             float64 `json:"today_main_net"`
	TodayMainDirection       string  `json:"today_main_direction"`
	TodayHugeNet             float64 `json:"today_huge_net"`
	TodayBigNet              float64 `json:"today_big_net"`
	InstitutionVsHotMoney    string  `json:"institution_vs_hotmoney"`
	MainConsecutiveDays      int     `json:"main_consecutive_days"`
	MainConsecutiveDirection string  `json:"main_consecutive_direction"`
	RetailBehavior           string  `json:"retail_behavior"`
	Verdict                  string  `json:"verdict"`
}

type TechnicalAnalysis struct {
	MA5            float64   `json:"ma5"`
	MA10           float64   `json:"ma10"`
	MA20           float64   `json:"ma20"`
	MA60           float64   `json:"ma60"`
	MAArrangement  string    `json:"ma_arrangement"`
	MACD_DIF       float64   `json:"macd_dif"`
	MACD_DEA       float64   `json:"macd_dea"`
	MACD_Hist      float64   `json:"macd_hist"`
	MACD_Signal    string    `json:"macd_signal"`
	MACD_HistLast3 []float64 `json:"macd_hist_last3"`
	MACD_HistTrend string    `json:"macd_hist_trend"`
	KDJ_K          float64   `json:"kdj_k"`
	KDJ_D          float64   `json:"kdj_d"`
	KDJ_J          float64   `json:"kdj_j"`
	KDJ_Signal     string    `json:"kdj_signal"`
	OBV_5D         []float64 `json:"obv_5d"`
	OBV_Trend      string    `json:"obv_trend"`
	RSI_14         float64   `json:"rsi_14"`
	RSI_Level      string    `json:"rsi_level"`
	BollUpper      float64   `json:"boll_upper"`
	BollMid        float64   `json:"boll_mid"`
	BollLower      float64   `json:"boll_lower"`
	BollBandwidth  float64   `json:"boll_bandwidth"`
	BollPosition   string    `json:"boll_position"`
	// Multi-period confirmation (weekly)
	WeeklyMACD     string  `json:"weekly_macd"`
	WeeklyRSI      float64 `json:"weekly_rsi"`
	WeeklyRSILevel string  `json:"weekly_rsi_level"`
	WeeklyMA       string  `json:"weekly_ma"`
	PeriodAlign    string  `json:"period_align"` // 日周共振/日强周弱/日弱周强/日周背离
	Verdict        string  `json:"verdict"`
}

type SectorAnalysis struct {
	Sectors        []string `json:"sectors"`
	PrimarySector  string   `json:"primary_sector"`
	IsSectorLeader bool     `json:"is_sector_leader"`
	// Relative strength (P1)
	SectorPctChg5D float64 `json:"sector_pct_chg_5d"`
	StockPctChg5D  float64 `json:"stock_pct_chg_5d"`
	RelStrength    float64 `json:"rel_strength"`
	RelStrengthTag string  `json:"rel_strength_tag"` // 强于板块/弱于板块/同步
	Verdict        string  `json:"verdict"`
}

type SentimentAnalysis struct {
	NewsCount         int      `json:"news_count"`
	AnnouncementCount int      `json:"announcement_count"`
	KeyEvents         []string `json:"key_events"`
	SentimentScore    float64  `json:"sentiment_score"`
	SentimentLabel    string   `json:"sentiment_label"`
	Verdict           string   `json:"verdict"`
}

type AnalysisSummary struct {
	OverallScore  int      `json:"overall_score"`
	OverallSignal string   `json:"overall_signal"`
	Strengths     []string `json:"strengths"`
	Risks         []string `json:"risks"`
	Suggestion    string   `json:"suggestion"`
}

// ---- Extended Analysis Dimensions ----

type FundamentalsAnalysis struct {
	ROE                  float64 `json:"roe"`
	ROELevel             string  `json:"roe_level"`
	GrossMargin          float64 `json:"gross_margin"`
	GrossMarginLevel     string  `json:"gross_margin_level"`
	NetMargin            float64 `json:"net_margin"`
	NetMarginLevel       string  `json:"net_margin_level"`
	DebtRatio            float64 `json:"debt_ratio"`
	DebtRatioLevel       string  `json:"debt_ratio_level"`
	RevenueGrowth        float64 `json:"revenue_growth"`
	RevenueGrowthLevel   string  `json:"revenue_growth_level"`
	NetProfitGrowth      float64 `json:"net_profit_growth"`
	NetProfitGrowthLevel string  `json:"net_profit_growth_level"`
	EPSTrend             string  `json:"eps_trend"`
	Score                int     `json:"score"`
	Verdict              string  `json:"verdict"`
}

type NorthboundAnalysis struct {
	LatestNetFlow  float64 `json:"latest_net_flow"`
	Trend5D        float64 `json:"trend_5d"`
	FlowDirection  string  `json:"flow_direction"`
	StockNetAmount float64 `json:"stock_net_amount"`
	StockAction    string  `json:"stock_action"`
	Signal         string  `json:"signal"`
	Verdict        string  `json:"verdict"`
}

type MarginAnalysis struct {
	LatestMarginBalance float64 `json:"latest_margin_balance"`
	MarginBalanceTrend  string  `json:"margin_balance_trend"`
	MarginBuyingTrend   string  `json:"margin_buying_trend"`
	ShortSellingTrend   string  `json:"short_selling_trend"`
	Signal              string  `json:"signal"`
	SentimentScore      float64 `json:"sentiment_score"`
	Verdict             string  `json:"verdict"`
}

// StockAnalysis is the full response for /api/analyze
type StockAnalysis struct {
	Code             string               `json:"code"`
	Name             string               `json:"name"`
	Version          string               `json:"version"`
	Quote            Quote                `json:"quote"`
	VolumePrice      VolumePriceAnalysis  `json:"volume_price"`
	Valuation        ValuationAnalysis    `json:"valuation"`
	Volatility       VolatilityAnalysis   `json:"volatility"`
	MoneyFlow        MoneyFlowAnalysis    `json:"money_flow"`
	Technical        TechnicalAnalysis    `json:"technical"`
	Sector           SectorAnalysis       `json:"sector"`
	Sentiment        SentimentAnalysis    `json:"sentiment"`
	Fundamentals     FundamentalsAnalysis `json:"fundamentals"`
	Northbound       NorthboundAnalysis   `json:"northbound"`
	MarginDetail     MarginAnalysis       `json:"margin_detail"`
	TrendAnalysis    TrendAnalysis        `json:"trend_analysis"`
	BuyZone          *BuyZoneAnalysis     `json:"buy_zone,omitempty"`
	TSuggestion      *TSuggestionAnalysis `json:"t_suggestion,omitempty"`
	IntradayForecast *IntradayForecast    `json:"intraday_forecast,omitempty"`
	PatternAnalysis  *PatternAnalysis     `json:"pattern_analysis,omitempty"`
	ShortTermScore   *ShortTermScore      `json:"short_term_score,omitempty"`
	Holding          *HoldingInfo         `json:"holding,omitempty"`
	Summary          AnalysisSummary      `json:"summary"`
	DataSources      map[string]string    `json:"data_sources"`
	Errors           map[string]string    `json:"errors"`
	FetchedAt        time.Time            `json:"fetched_at"`
}

// ---- Trading signal models ----

// BuyZone represents a single buy price zone.
type BuyZone struct {
	Method       string  `json:"method"`
	UpperBound   float64 `json:"upper_bound"`
	LowerBound   float64 `json:"lower_bound"`
	OptimalEntry float64 `json:"optimal_entry"`
	StopLoss     float64 `json:"stop_loss"`
	SafetyScore  float64 `json:"safety_score"`
}

// BuyZoneAnalysis holds multiple buy zone suggestions.
type BuyZoneAnalysis struct {
	Zones   []BuyZone `json:"zones"`
	Optimal *BuyZone  `json:"optimal"`
	Verdict string    `json:"verdict"`
}

// TSuggestionAnalysis holds a T+0 round-trip trading suggestion.
type TSuggestionAnalysis struct {
	Type              string   `json:"type"`
	Action            string   `json:"action"`
	EntryPrice        float64  `json:"entry_price"`
	TargetPrice       float64  `json:"target_price"`
	StopLoss          float64  `json:"stop_loss"`
	Reason            string   `json:"reason"`
	Confidence        float64  `json:"confidence"`
	SignalScore       float64  `json:"signal_score"`
	TQuantity         int      `json:"t_quantity"`
	PositionRatio     float64  `json:"position_ratio"`
	TriggerPct        float64  `json:"trigger_pct"`
	ExpectedProfitPct float64  `json:"expected_profit_pct"`
	MaxLossPct        float64  `json:"max_loss_pct"`
	RiskReward        float64  `json:"risk_reward"`
	ExecutionTip      string   `json:"execution_tip,omitempty"`
	SignalDetails     []string `json:"signal_details,omitempty"`
	RiskNotes         []string `json:"risk_notes,omitempty"`
	// Condition order fields
	ConditionBuy  *ConditionOrder `json:"condition_buy,omitempty"`
	ConditionSell *ConditionOrder `json:"condition_sell,omitempty"`
}

// ConditionOrder represents a specific conditional order for T+0 trading.
type ConditionOrder struct {
	Direction     string  `json:"direction"`      // "买入" / "卖出"
	TriggerPrice  float64 `json:"trigger_price"`  // 触发价
	TriggerDesc   string  `json:"trigger_desc"`   // 触发条件描述
	OrderPrice    float64 `json:"order_price"`    // 委托价
	OrderType     string  `json:"order_type"`     // "限价委托"
	QuantityRatio string  `json:"quantity_ratio"` // "底仓的1/3" / "底仓的1/2"
	StopPrice     float64 `json:"stop_price"`     // 止损/止盈触发价
	StopDesc      string  `json:"stop_desc"`      // 止损/止盈描述
	Note          string  `json:"note"`           // 操作备注
}

// IntradayForecast holds predicted daily high/low and current zone.
type IntradayForecast struct {
	PredictedHigh float64 `json:"predicted_high"`
	PredictedLow  float64 `json:"predicted_low"`
	// ±1σ confidence bands derived from recent TR std-dev. ~68% of trading
	// days should see the actual high fall in [PredictedHighDown, PredictedHighUp]
	// (and analogously for the low). Tight bands → high-confidence forecast;
	// wide bands → low-confidence, treat the central number skeptically.
	PredictedHighUp   float64 `json:"predicted_high_up,omitempty"`
	PredictedHighDown float64 `json:"predicted_high_down,omitempty"`
	PredictedLowUp    float64 `json:"predicted_low_up,omitempty"`
	PredictedLowDown  float64 `json:"predicted_low_down,omitempty"`
	SigmaHigh         float64 `json:"sigma_high,omitempty"`
	SigmaLow          float64 `json:"sigma_low,omitempty"`
	// Median high/low — P50 of the historical up/down excursion distribution.
	// ~50% of trading days will reach this level. Use these for "high fill
	// rate" condition orders when you must transact today at a tolerable
	// price. Contrast with PredictedHigh (P83, only 17% of days reach it).
	PredictedHighMedian float64 `json:"predicted_high_median,omitempty"`
	PredictedLowMedian  float64 `json:"predicted_low_median,omitempty"`
	CurrentZone         string  `json:"current_zone"`
	ZonePct             float64 `json:"zone_pct"`
	Bias                string  `json:"bias,omitempty"`        // "bullish" / "bearish" / "neutral"
	BiasReason          string  `json:"bias_reason,omitempty"` // e.g. "锤子线+放量突破"
	BiasStrength        float64 `json:"bias_strength,omitempty"` // sum of pattern confidences, capped at 2.5
	SupportLevel        float64 `json:"support_level,omitempty"`
	ResistLevel         float64 `json:"resist_level,omitempty"`
}

// PatternSignal represents a detected K-line, chart, or volume pattern.
type PatternSignal struct {
	Pattern     string  `json:"pattern"`
	Category    string  `json:"category"`   // "kline" / "chart" / "volume"
	Direction   string  `json:"direction"`  // "bullish" / "bearish" / "neutral"
	Confidence  float64 `json:"confidence"` // 0-1
	Date        string  `json:"date"`
	Description string  `json:"description"`
}

// PatternAnalysis summarizes detected K-line and chart patterns.
type PatternAnalysis struct {
	Signals       []PatternSignal `json:"signals"`
	KlineSignals  []PatternSignal `json:"kline_signals,omitempty"`
	ChartSignals  []PatternSignal `json:"chart_signals,omitempty"`
	VolumeSignals []PatternSignal `json:"volume_signals,omitempty"`
	Primary       *PatternSignal  `json:"primary,omitempty"`
	BullishCount  int             `json:"bullish_count"`
	BearishCount  int             `json:"bearish_count"`
	NeutralCount  int             `json:"neutral_count"`
	NetBias       string          `json:"net_bias"`
	ScoreImpact   float64         `json:"score_impact"`
	Verdict       string          `json:"verdict"`
}

// ScoreComponent explains one component of the short-term score.
type ScoreComponent struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
	Reason string  `json:"reason"`
}

// ShortTermScore holds a 1-5 day tactical score.
type ShortTermScore struct {
	Score      float64          `json:"score"`
	Grade      string           `json:"grade"`
	Signal     string           `json:"signal"`
	Components []ScoreComponent `json:"components"`
	Reasons    []string         `json:"reasons,omitempty"`
	Risks      []string         `json:"risks,omitempty"`
	Verdict    string           `json:"verdict"`
}

// HoldingInfo holds the user's portfolio position for the analyzed stock.
type HoldingInfo struct {
	Quantity    int     `json:"quantity"`
	CostPrice   float64 `json:"cost_price"`
	MarketValue float64 `json:"market_value"`
	PnL         float64 `json:"pnl"`
	PnLPct      float64 `json:"pnl_pct"`
}
