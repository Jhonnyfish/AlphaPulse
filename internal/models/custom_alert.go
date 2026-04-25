package models

import "time"

// CustomAlert represents a user-defined price alert.
type CustomAlert struct {
	ID         string    `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Threshold  float64   `json:"threshold"`
	Enabled    bool      `json:"enabled"`
	Triggered  bool      `json:"triggered"`
	CreatedAt  time.Time `json:"created_at"`
}

// TriggeredAlert extends CustomAlert with current market data.
type TriggeredAlert struct {
	CustomAlert
	CurrentPrice     float64 `json:"current_price"`
	CurrentChangePct float64 `json:"current_change_pct"`
	StockName        string  `json:"stock_name"`
}

// CustomAlertListResponse is the response for GET /api/custom-alerts.
type CustomAlertListResponse struct {
	OK     bool           `json:"ok"`
	Alerts []CustomAlert  `json:"alerts"`
}

// CustomAlertResponse wraps a single alert.
type CustomAlertResponse struct {
	OK    bool         `json:"ok"`
	Alert CustomAlert  `json:"alert"`
}

// CustomAlertCheckResponse is the response for GET /api/custom-alerts/check.
type CustomAlertCheckResponse struct {
	OK        bool             `json:"ok"`
	Triggered []TriggeredAlert `json:"triggered"`
	Checked   int              `json:"checked"`
}
