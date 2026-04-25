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

func TestSectorValidate(t *testing.T) {
	tests := []struct {
		name    string
		sector  Sector
		wantErr bool
	}{
		{
			name: "valid sector",
			sector: Sector{
				Code: "BK0477", Name: "白酒",
				Price: 1234.56, Change: 12.34, ChangePercent: 1.01,
			},
			wantErr: false,
		},
		{
			name: "empty code",
			sector: Sector{
				Code: "", Name: "白酒",
				Price: 100, ChangePercent: 1.0,
			},
			wantErr: true,
		},
		{
			name: "empty name",
			sector: Sector{
				Code: "BK0477", Name: "",
				Price: 100, ChangePercent: 1.0,
			},
			wantErr: true,
		},
		{
			name: "negative price",
			sector: Sector{
				Code: "BK0477", Name: "白酒",
				Price: -10, ChangePercent: 1.0,
			},
			wantErr: true,
		},
		{
			name: "zero price is valid",
			sector: Sector{
				Code: "BK0477", Name: "白酒",
				Price: 0, ChangePercent: 0,
			},
			wantErr: false,
		},
		{
			name: "change exceeds limit",
			sector: Sector{
				Code: "BK0477", Name: "白酒",
				Price: 100, ChangePercent: 25.0,
			},
			wantErr: true,
		},
		{
			name: "change within limit",
			sector: Sector{
				Code: "BK0477", Name: "白酒",
				Price: 120, ChangePercent: 20.0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sector.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOverviewIndexValidate(t *testing.T) {
	tests := []struct {
		name    string
		index   OverviewIndex
		wantErr bool
	}{
		{
			name: "valid index",
			index: OverviewIndex{
				Code: "000001", Name: "上证指数",
				Price: 3200.50, Change: 15.30, ChangePercent: 0.48,
				AdvanceCount: 2500, DeclineCount: 1800, FlatCount: 200,
			},
			wantErr: false,
		},
		{
			name: "empty code",
			index: OverviewIndex{
				Code: "", Name: "上证指数",
				Price: 3200, ChangePercent: 0.5,
			},
			wantErr: true,
		},
		{
			name: "empty name",
			index: OverviewIndex{
				Code: "000001", Name: "",
				Price: 3200, ChangePercent: 0.5,
			},
			wantErr: true,
		},
		{
			name: "negative price",
			index: OverviewIndex{
				Code: "000001", Name: "上证指数",
				Price: -100, ChangePercent: 0.5,
			},
			wantErr: true,
		},
		{
			name: "change exceeds limit",
			index: OverviewIndex{
				Code: "000001", Name: "上证指数",
				Price: 3200, ChangePercent: 25.0,
			},
			wantErr: true,
		},
		{
			name: "negative advance count",
			index: OverviewIndex{
				Code: "000001", Name: "上证指数",
				Price: 3200, ChangePercent: 0.5,
				AdvanceCount: -1, DeclineCount: 100, FlatCount: 50,
			},
			wantErr: true,
		},
		{
			name: "negative decline count",
			index: OverviewIndex{
				Code: "000001", Name: "上证指数",
				Price: 3200, ChangePercent: 0.5,
				AdvanceCount: 100, DeclineCount: -1, FlatCount: 50,
			},
			wantErr: true,
		},
		{
			name: "negative flat count",
			index: OverviewIndex{
				Code: "000001", Name: "上证指数",
				Price: 3200, ChangePercent: 0.5,
				AdvanceCount: 100, DeclineCount: 50, FlatCount: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.index.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
