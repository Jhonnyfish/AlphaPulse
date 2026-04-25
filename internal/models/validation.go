package models

import (
	"fmt"
	"math"
	"strings"
)

// Validate checks that quote data is within reasonable bounds.
// Returns error describing the first validation failure, or nil if valid.
func (q Quote) Validate() error {
	if q.Code == "" {
		return fmt.Errorf("quote validation: code is empty")
	}
	if q.Name == "" {
		return fmt.Errorf("quote validation: name is empty for %s", q.Code)
	}

	// Skip price validation if market hasn't opened (all zeros)
	if q.Price == 0 && q.Open == 0 && q.PrevClose == 0 {
		return nil
	}

	if q.Price < 0 {
		return fmt.Errorf("quote validation: price %.4f is negative for %s", q.Price, q.Code)
	}
	if q.Open < 0 {
		return fmt.Errorf("quote validation: open %.4f is negative for %s", q.Open, q.Code)
	}
	if q.High < 0 {
		return fmt.Errorf("quote validation: high %.4f is negative for %s", q.High, q.Code)
	}
	if q.Low < 0 {
		return fmt.Errorf("quote validation: low %.4f is negative for %s", q.Low, q.Code)
	}
	if q.PrevClose < 0 {
		return fmt.Errorf("quote validation: prev_close %.4f is negative for %s", q.PrevClose, q.Code)
	}

	// A-share daily limit: ±22% (covers ±20% limit boards + slight overshoot)
	if q.ChangePercent != 0 && math.Abs(q.ChangePercent) > 22.0 {
		return fmt.Errorf("quote validation: change_percent %.2f%% exceeds ±22%% for %s", q.ChangePercent, q.Code)
	}

	// High should be >= Low when both are non-zero
	if q.High > 0 && q.Low > 0 && q.High < q.Low {
		return fmt.Errorf("quote validation: high %.4f < low %.4f for %s", q.High, q.Low, q.Code)
	}

	return nil
}

// Validate checks K-line data point.
func (k KlinePoint) Validate() error {
	if k.Date == "" {
		return fmt.Errorf("kline validation: date is empty")
	}
	if k.Close < 0 {
		return fmt.Errorf("kline validation: close %.4f is negative", k.Close)
	}
	if k.Volume < 0 {
		return fmt.Errorf("kline validation: volume %.0f is negative", k.Volume)
	}
	if k.High > 0 && k.Low > 0 && k.High < k.Low {
		return fmt.Errorf("kline validation: high %.4f < low %.4f", k.High, k.Low)
	}
	return nil
}

// Validate checks that sector data is within reasonable bounds.
func (s Sector) Validate() error {
	s.Code = strings.TrimSpace(s.Code)
	s.Name = strings.TrimSpace(s.Name)
	if s.Code == "" {
		return fmt.Errorf("sector validation: code is empty")
	}
	if s.Name == "" {
		return fmt.Errorf("sector validation: name is empty for code %s", s.Code)
	}
	if s.Price < 0 {
		return fmt.Errorf("sector validation: price %.4f is negative for %s", s.Price, s.Code)
	}
	if math.Abs(s.ChangePercent) > 22.0 {
		return fmt.Errorf("sector validation: change_percent %.2f%% exceeds ±22%% for %s", s.ChangePercent, s.Code)
	}
	return nil
}

// Validate checks that overview index data is within reasonable bounds.
func (o OverviewIndex) Validate() error {
	o.Code = strings.TrimSpace(o.Code)
	o.Name = strings.TrimSpace(o.Name)
	if o.Code == "" {
		return fmt.Errorf("overview index validation: code is empty")
	}
	if o.Name == "" {
		return fmt.Errorf("overview index validation: name is empty for code %s", o.Code)
	}
	if o.Price < 0 {
		return fmt.Errorf("overview index validation: price %.4f is negative for %s", o.Price, o.Code)
	}
	if math.Abs(o.ChangePercent) > 22.0 {
		return fmt.Errorf("overview index validation: change_percent %.2f%% exceeds ±22%% for %s", o.ChangePercent, o.Code)
	}
	if o.AdvanceCount < 0 {
		return fmt.Errorf("overview index validation: advance_count %d is negative for %s", o.AdvanceCount, o.Code)
	}
	if o.DeclineCount < 0 {
		return fmt.Errorf("overview index validation: decline_count %d is negative for %s", o.DeclineCount, o.Code)
	}
	if o.FlatCount < 0 {
		return fmt.Errorf("overview index validation: flat_count %d is negative for %s", o.FlatCount, o.Code)
	}
	return nil
}
