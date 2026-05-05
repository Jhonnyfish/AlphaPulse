package models

// PeriodIndicators holds technical indicators for a single time period (daily/weekly/monthly).
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

// MultiTrendStock holds multi-period trend data for a single stock.
type MultiTrendStock struct {
	Code            string            `json:"code"`
	Name            string            `json:"name"`
	Daily           PeriodIndicators  `json:"daily"`
	Weekly          PeriodIndicators  `json:"weekly"`
	Monthly         PeriodIndicators  `json:"monthly"`
	OverallStrength int               `json:"overall_strength"`
}

// MultiTrendResponse is the response for GET /api/multi-trend.
type MultiTrendResponse struct {
	OK     bool               `json:"ok"`
	Stocks []MultiTrendStock  `json:"stocks"`
	Cached bool               `json:"cached"`
}

// CorrelationResponse is the response for GET /api/correlation.
type CorrelationResponse struct {
	OK      bool        `json:"ok"`
	Codes   []string    `json:"codes"`
	Names   []string    `json:"names"`
	Matrix  [][]float64 `json:"matrix"`
	Message string      `json:"message,omitempty"`
	Cached  bool        `json:"cached"`
}
