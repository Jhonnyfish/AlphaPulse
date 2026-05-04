package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimplifySector(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"C39计算机、通信和其他电子设备制造业", "计算机通信电子设"},   // 8 rune limit after removing suffix + punctuation
		{"K70房地产业", "房地产业"},                                    // no suffix matched, 4 chars kept
		{"I65软件和信息技术服务业", "软件信息技术"},                   // "服务业" stripped
		{"未分类", "未分类"},
		{"银行", "银行"},
		{"C27医药制造业", "医药"},                                    // "制造业" stripped
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := simplifySector(tt.input)
			assert.Equal(t, tt.expected, result, "simplifySector(%q)", tt.input)
		})
	}
}

func TestScoreDimension(t *testing.T) {
	tests := []struct {
		verdict    string
		isPositive bool
		minScore   float64
		maxScore   float64
	}{
		{"强势买入信号", false, 80, 90},
		{"偏多格局", false, 65, 75},
		{"中性均衡", false, 45, 55},
		{"偏空谨慎", false, 30, 40},
		{"弱势危险", false, 15, 25},
		{"未知状态", true, 55, 65},
		{"未知状态", false, 45, 55},
	}
	for _, tt := range tests {
		t.Run(tt.verdict, func(t *testing.T) {
			score := scoreDimension(tt.verdict, tt.isPositive)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}
