package models

import (
	"fmt"
	"math"
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
