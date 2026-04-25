package services

import (
	"testing"
	"time"
)

func TestParseKlinePoint(t *testing.T) {
	// Format: date,open,close,high,low,volume,amount
	parts := []string{"2026-04-25", "1845.00", "1850.00", "1860.00", "1835.00", "12345", "228000000"}
	point, err := parseKlinePoint(parts)
	if err != nil {
		t.Fatalf("parseKlinePoint failed: %v", err)
	}

	if point.Date != "2026-04-25" {
		t.Errorf("expected date 2026-04-25, got %s", point.Date)
	}
	if point.Open != 1845.00 {
		t.Errorf("expected open 1845.00, got %.2f", point.Open)
	}
	if point.Close != 1850.00 {
		t.Errorf("expected close 1850.00, got %.2f", point.Close)
	}
	if point.High != 1860.00 {
		t.Errorf("expected high 1860.00, got %.2f", point.High)
	}
	if point.Low != 1835.00 {
		t.Errorf("expected low 1835.00, got %.2f", point.Low)
	}
	if point.Volume != 12345 {
		t.Errorf("expected volume 12345, got %.0f", point.Volume)
	}
	if point.Amount != 228000000 {
		t.Errorf("expected amount 228000000, got %.0f", point.Amount)
	}

	// Validate the parsed data
	if err := point.Validate(); err != nil {
		t.Errorf("parsed kline point failed validation: %v", err)
	}
}

func TestParseKlinePointInvalidFloat(t *testing.T) {
	parts := []string{"2026-04-25", "abc", "1850.00", "1860.00", "1835.00", "12345", "228000000"}
	_, err := parseKlinePoint(parts)
	if err == nil {
		t.Error("expected error for invalid float in open")
	}
}

func TestParseKlinePointNegativeVolume(t *testing.T) {
	parts := []string{"2026-04-25", "100", "100", "100", "100", "-100", "100"}
	point, err := parseKlinePoint(parts)
	if err != nil {
		t.Fatalf("parseKlinePoint failed: %v", err)
	}
	// parseKlinePoint doesn't validate — it just parses.
	// Validation should happen at the caller level via Validate()
	if err := point.Validate(); err == nil {
		t.Error("expected validation error for negative volume")
	}
}

func TestParseEastMoneyTime(t *testing.T) {
	tests := []struct {
		input    string
		hasError bool
	}{
		{"2026-04-25 15:00:03", false},
		{"2026-04-25 15:00", false},
		{"2026-04-25T15:00:03Z", false},
		{"", true},  // No valid layout, returns zero time
		{"invalid", true},
	}

	for _, tt := range tests {
		got := parseEastMoneyTime(tt.input)
		if tt.hasError {
			if !got.IsZero() {
				t.Errorf("parseEastMoneyTime(%q) expected zero time, got %v", tt.input, got)
			}
		} else {
			if got.IsZero() {
				t.Errorf("parseEastMoneyTime(%q) returned zero time unexpectedly", tt.input)
			}
		}
	}
}

func TestParseEastMoneyTimeSpecific(t *testing.T) {
	got := parseEastMoneyTime("2026-04-25 15:00:03")
	expected := time.Date(2026, 4, 25, 15, 0, 3, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("parseEastMoneyTime = %v, want %v", got, expected)
	}
}
