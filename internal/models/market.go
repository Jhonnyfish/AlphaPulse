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
