package models

import (
	"encoding/json"
	"time"
)

// Strategy represents a custom or builtin strategy.
type Strategy struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	Type          string          `json:"type"`
	Scoring       json.RawMessage `json:"scoring"`
	Dimensions    json.RawMessage `json:"dimensions"`
	Filters       json.RawMessage `json:"filters"`
	MaxCandidates int             `json:"max_candidates"`
	IsActive      bool            `json:"is_active"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// StrategyListResponse is the response for GET /api/strategies.
type StrategyListResponse struct {
	OK               bool       `json:"ok"`
	Strategies       []Strategy `json:"strategies"`
	ActiveStrategies []string   `json:"active_strategies"`
}

// StrategyResponse wraps a single strategy.
type StrategyResponse struct {
	OK       bool     `json:"ok"`
	Strategy Strategy `json:"strategy"`
	Message  string   `json:"message,omitempty"`
}

// StrategyActionResponse is for activate/deactivate/delete.
type StrategyActionResponse struct {
	OK               bool     `json:"ok"`
	Message          string   `json:"message"`
	ActiveStrategies []string `json:"active_strategies,omitempty"`
}
