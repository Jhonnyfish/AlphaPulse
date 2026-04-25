package models

import (
	"testing"
)

func TestQuoteValidate(t *testing.T) {
	tests := []struct {
		name    string
		quote   Quote
		wantErr bool
	}{
		{
			name: "valid quote",
			quote: Quote{
				Code: "600519", Name: "贵州茅台",
				Price: 1850.00, Open: 1845.00, PrevClose: 1840.00,
				High: 1860.00, Low: 1835.00, ChangePercent: 0.54,
			},
			wantErr: false,
		},
		{
			name: "zero prices (market not open)",
			quote: Quote{
				Code: "600519", Name: "贵州茅台",
				Price: 0, Open: 0, PrevClose: 0,
			},
			wantErr: false,
		},
		{
			name: "empty code",
			quote: Quote{
				Code: "", Name: "贵州茅台", Price: 100,
			},
			wantErr: true,
		},
		{
			name: "empty name",
			quote: Quote{
				Code: "600519", Name: "", Price: 100,
			},
			wantErr: true,
		},
		{
			name: "negative price",
			quote: Quote{
				Code: "600519", Name: "贵州茅台",
				Price: -100, PrevClose: 100,
			},
			wantErr: true,
		},
		{
			name: "change exceeds limit",
			quote: Quote{
				Code: "600519", Name: "贵州茅台",
				Price: 100, PrevClose: 100, ChangePercent: 25.0,
			},
			wantErr: true,
		},
		{
			name: "change within limit",
			quote: Quote{
				Code: "600519", Name: "贵州茅台",
				Price: 120, PrevClose: 100, ChangePercent: 20.0,
			},
			wantErr: false,
		},
		{
			name: "high < low",
			quote: Quote{
				Code: "600519", Name: "贵州茅台",
				Price: 100, High: 90, Low: 110, PrevClose: 100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.quote.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKlinePointValidate(t *testing.T) {
	tests := []struct {
		name    string
		kline   KlinePoint
		wantErr bool
	}{
		{
			name:    "valid kline",
			kline:   KlinePoint{Date: "2026-04-25", Open: 100, Close: 105, High: 110, Low: 95, Volume: 1000},
			wantErr: false,
		},
		{
			name:    "empty date",
			kline:   KlinePoint{Date: "", Close: 100},
			wantErr: true,
		},
		{
			name:    "negative close",
			kline:   KlinePoint{Date: "2026-04-25", Close: -1},
			wantErr: true,
		},
		{
			name:    "negative volume",
			kline:   KlinePoint{Date: "2026-04-25", Close: 100, Volume: -100},
			wantErr: true,
		},
		{
			name:    "high < low",
			kline:   KlinePoint{Date: "2026-04-25", Close: 100, High: 90, Low: 110},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.kline.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
