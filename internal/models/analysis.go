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
	URL         string    `json:"url"`
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

type OrderFlowAnalysis struct {
	OuterVol     float64 `json:"outer_vol"`
	InnerVol     float64 `json:"inner_vol"`
	OuterRatio   float64 `json:"outer_ratio"`
	NetDirection string  `json:"net_direction"`
	Verdict      string  `json:"verdict"`
}

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
	TodayMainNet            float64 `json:"today_main_net"`
	TodayMainDirection      string  `json:"today_main_direction"`
	TodayHugeNet            float64 `json:"today_huge_net"`
	TodayBigNet             float64 `json:"today_big_net"`
	InstitutionVsHotMoney   string  `json:"institution_vs_hotmoney"`
	MainConsecutiveDays     int     `json:"main_consecutive_days"`
	MainConsecutiveDirection string  `json:"main_consecutive_direction"`
	RetailBehavior          string  `json:"retail_behavior"`
	Verdict                 string  `json:"verdict"`
}

type TechnicalAnalysis struct {
	MA5           float64   `json:"ma5"`
	MA10          float64   `json:"ma10"`
	MA20          float64   `json:"ma20"`
	MA60          float64   `json:"ma60"`
	MAArrangement string    `json:"ma_arrangement"`
	MACD_DIF      float64   `json:"macd_dif"`
	MACD_DEA      float64   `json:"macd_dea"`
	MACD_Hist     float64   `json:"macd_hist"`
	MACD_Signal   string    `json:"macd_signal"`
	MACD_HistLast3 []float64 `json:"macd_hist_last3"`
	MACD_HistTrend string   `json:"macd_hist_trend"`
	KDJ_K         float64   `json:"kdj_k"`
	KDJ_D         float64   `json:"kdj_d"`
	KDJ_J         float64   `json:"kdj_j"`
	KDJ_Signal    string    `json:"kdj_signal"`
	OBV_5D        []float64 `json:"obv_5d"`
	OBV_Trend     string    `json:"obv_trend"`
	RSI_14        float64   `json:"rsi_14"`
	RSI_Level     string    `json:"rsi_level"`
	BollUpper     float64   `json:"boll_upper"`
	BollMid       float64   `json:"boll_mid"`
	BollLower     float64   `json:"boll_lower"`
	BollBandwidth float64   `json:"boll_bandwidth"`
	BollPosition  string    `json:"boll_position"`
	Verdict       string    `json:"verdict"`
}

type SectorAnalysis struct {
	Sectors        []string `json:"sectors"`
	PrimarySector  string   `json:"primary_sector"`
	IsSectorLeader bool     `json:"is_sector_leader"`
	Verdict        string   `json:"verdict"`
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

// StockAnalysis is the full response for /api/analyze
type StockAnalysis struct {
	Code        string              `json:"code"`
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Quote       Quote               `json:"quote"`
	OrderFlow   OrderFlowAnalysis   `json:"order_flow"`
	VolumePrice VolumePriceAnalysis `json:"volume_price"`
	Valuation   ValuationAnalysis   `json:"valuation"`
	Volatility  VolatilityAnalysis  `json:"volatility"`
	MoneyFlow   MoneyFlowAnalysis   `json:"money_flow"`
	Technical   TechnicalAnalysis   `json:"technical"`
	Sector      SectorAnalysis      `json:"sector"`
	Sentiment   SentimentAnalysis   `json:"sentiment"`
	Summary     AnalysisSummary     `json:"summary"`
	DataSources map[string]string   `json:"data_sources"`
	Errors      map[string]string   `json:"errors"`
	FetchedAt   time.Time           `json:"fetched_at"`
}
