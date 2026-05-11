package models

// TrendStage represents the current trend stage analysis.
type TrendStage struct {
	Direction    string  `json:"direction"`     // "上升", "下降", "震荡"
	Stage        string  `json:"stage"`         // "早期", "中期", "末期"
	Strength     float64 `json:"strength"`      // 0-100 trend strength
	Confidence   float64 `json:"confidence"`    // 0-100 confidence in assessment
	Signals      []string `json:"signals"`      // supporting signals
	Description  string  `json:"description"`   // human-readable description
}

// SupportResistance holds key price levels.
type SupportResistance struct {
	// Support levels (price floors)
	Support1      float64 `json:"support1"`       // nearest support
	Support2      float64 `json:"support2"`       // second support
	Support3      float64 `json:"support3"`       // third support
	// Resistance levels (price ceilings)
	Resistance1   float64 `json:"resistance1"`    // nearest resistance
	Resistance2   float64 `json:"resistance2"`    // second resistance
	Resistance3   float64 `json:"resistance3"`    // third resistance
	// Key moving averages as dynamic S/R
	MA5           float64 `json:"ma5"`
	MA10          float64 `json:"ma10"`
	MA20          float64 `json:"ma20"`
	MA60          float64 `json:"ma60"`
	// Bollinger bands
	BollUpper     float64 `json:"boll_upper"`
	BollMid       float64 `json:"boll_mid"`
	BollLower     float64 `json:"boll_lower"`
	// Current price position
	PricePosition string  `json:"price_position"` // "上方", "下方", "附近"
	NearestLevel  float64 `json:"nearest_level"`  // nearest S/R level
	NearestType   string  `json:"nearest_type"`   // "支撑" or "阻力"
	DistancePct   float64 `json:"distance_pct"`   // % distance to nearest level
}

// TrendAnalysis is the full trend analysis response.
type TrendAnalysis struct {
	TrendStage        TrendStage        `json:"trend_stage"`
	SupportResistance SupportResistance `json:"support_resistance"`
	Verdict           string            `json:"verdict"`
}

// MultiTrendStock is a stock in the multi-trend view.
type MultiTrendStock struct {
	Code            string       `json:"code"`
	Name            string       `json:"name"`
	Price           float64      `json:"price"`
	ChangePct       float64      `json:"change_pct"`
	Trend           TrendAnalysis `json:"trend"`
	Daily           interface{}  `json:"daily,omitempty"`
	Weekly          interface{}  `json:"weekly,omitempty"`
	Monthly         interface{}  `json:"monthly,omitempty"`
	OverallStrength float64      `json:"overall_strength,omitempty"`
}

// MultiTrendResponse is the response for GET /api/multi-trend.
type MultiTrendResponse struct {
	OK     bool               `json:"ok"`
	Stocks []MultiTrendStock  `json:"stocks"`
	Cached bool               `json:"cached,omitempty"`
	Error  string             `json:"error,omitempty"`
}

// CorrelationResponse is the response for correlation analysis.
type CorrelationResponse struct {
	OK      bool        `json:"ok"`
	Codes   []string    `json:"codes,omitempty"`
	Names   []string    `json:"names,omitempty"`
	Matrix  interface{} `json:"matrix,omitempty"`
	Message string      `json:"message,omitempty"`
	Cached  bool        `json:"cached,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PeriodIndicators holds technical indicators for a specific time period.
type PeriodIndicators struct {
	ReturnPct   float64 `json:"return_pct"`
	MA5         float64 `json:"ma5"`
	MA10        float64 `json:"ma10"`
	MA20        float64 `json:"ma20"`
	MAAligned   bool    `json:"ma_aligned"`
	RSI         float64 `json:"rsi"`
	VolumeTrend string  `json:"volume_trend"`
	Strength    int     `json:"strength"`
}
