package services

import (
	"testing"
)

func TestValidateStockCode(t *testing.T) {
	tests := []struct {
		code    string
		wantErr bool
	}{
		{"600519", false}, // 贵州茅台 - Shanghai main
		{"000001", false}, // 平安银行 - Shenzhen main
		{"300750", false}, // 宁德时代 - ChiNext
		{"002594", false}, // 比亚迪 - SME
		{"688001", false}, // 华兴源创 - STAR
		{"830799", false}, // Beijing
		{"900901", false}, // Shanghai B
		{"200002", false}, // Shenzhen B
		{"", true},        // empty
		{"6005", true},    // too short
		{"6005190", true}, // too long
		{"abcdef", true},  // non-numeric
		{"60051a", true},  // mixed
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := ValidateStockCode(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStockCode(%q) error = %v, wantErr %v", tt.code, err, tt.wantErr)
			}
		})
	}
}

func TestIsShanghai(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{"600519", true},
		{"601398", true},
		{"603986", true},
		{"605588", true},
		{"688001", true},
		{"900901", true},
		{"000001", false},
		{"002594", false},
		{"300750", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := IsShanghai(tt.code)
			if got != tt.expected {
				t.Errorf("IsShanghai(%s) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestEastMoneySecID(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"600519", "1.600519"},
		{"000001", "0.000001"},
		{"300750", "0.300750"},
		{"688001", "1.688001"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := EastMoneySecID(tt.code)
			if got != tt.expected {
				t.Errorf("EastMoneySecID(%s) = %s, want %s", tt.code, got, tt.expected)
			}
		})
	}
}

func TestTencentSymbol(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"600519", "sh600519"},
		{"000001", "sz000001"},
		{"300750", "sz300750"},
		{"688001", "sh688001"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := TencentSymbol(tt.code)
			if got != tt.expected {
				t.Errorf("TencentSymbol(%s) = %s, want %s", tt.code, got, tt.expected)
			}
		})
	}
}
