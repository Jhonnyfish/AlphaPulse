package handlers

import (
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
)

// generateTestAnalysis creates a StockAnalysis for testing.
func generateTestAnalysis(score int, signal string, strengths, risks []string) models.StockAnalysis {
	return models.StockAnalysis{
		Code: "000001",
		Name: "测试股票",
		Summary: models.AnalysisSummary{
			OverallScore:  score,
			OverallSignal: signal,
			Strengths:     strengths,
			Risks:         risks,
		},
		Sentiment: models.SentimentAnalysis{
			KeyEvents: []string{},
		},
	}
}

func TestAlertPriority(t *testing.T) {
	assert.Equal(t, 0, alertPriority(AlertWarning))
	assert.Equal(t, 1, alertPriority(AlertOpportunity))
	assert.Equal(t, 2, alertPriority(AlertInfo))
	assert.Equal(t, 3, alertPriority(AlertType("unknown")))
}

func TestContainsNotablePositive(t *testing.T) {
	assert.True(t, containsNotablePositive("强势突破"))
	assert.True(t, containsNotablePositive("涨停板"))
	assert.True(t, containsNotablePositive("金叉信号"))
	assert.True(t, containsNotablePositive("主力买入"))
	assert.True(t, containsNotablePositive("资金流入明显"))
	assert.True(t, containsNotablePositive("业绩预增"))
	assert.False(t, containsNotablePositive("普通走势"))
	assert.False(t, containsNotablePositive(""))
}

func TestContainsNotableNegative(t *testing.T) {
	assert.True(t, containsNotableNegative("弱势下跌"))
	assert.True(t, containsNotableNegative("跌停板"))
	assert.True(t, containsNotableNegative("死叉信号"))
	assert.True(t, containsNotableNegative("主力卖出"))
	assert.True(t, containsNotableNegative("资金流出"))
	assert.True(t, containsNotableNegative("业绩预减"))
	assert.True(t, containsNotableNegative("风险提示"))
	assert.False(t, containsNotableNegative("正常波动"))
	assert.False(t, containsNotableNegative(""))
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "1", itoa(1))
	assert.Equal(t, "42", itoa(42))
	assert.Equal(t, "100", itoa(100))
	assert.Equal(t, "-5", itoa(-5))
	assert.Equal(t, "99999", itoa(99999))
}

func TestGenerateAlertsForStock(t *testing.T) {
	// Test with high score (opportunity alert)
	highScoreAnalysis := generateTestAnalysis(80, "买入", []string{"强势突破"}, []string{})
	alerts := generateAlertsForStock(highScoreAnalysis)
	assert.NotEmpty(t, alerts)
	foundOpportunity := false
	for _, a := range alerts {
		if a.Type == AlertOpportunity {
			foundOpportunity = true
			assert.Equal(t, 80, a.Score)
		}
	}
	assert.True(t, foundOpportunity, "should generate opportunity alert for high score")

	// Test with low score (warning alert)
	lowScoreAnalysis := generateTestAnalysis(30, "卖出", []string{}, []string{"弱势下跌"})
	alerts = generateAlertsForStock(lowScoreAnalysis)
	assert.NotEmpty(t, alerts)
	foundWarning := false
	for _, a := range alerts {
		if a.Type == AlertWarning {
			foundWarning = true
			assert.Equal(t, 30, a.Score)
		}
	}
	assert.True(t, foundWarning, "should generate warning alert for low score")

	// Test with medium score (no high/low alerts)
	mediumAnalysis := generateTestAnalysis(55, "持有", []string{"普通走势"}, []string{"正常波动"})
	alerts = generateAlertsForStock(mediumAnalysis)
	for _, a := range alerts {
		assert.NotEqual(t, AlertOpportunity, a.Type, "medium score should not have opportunity alerts")
		assert.NotEqual(t, AlertWarning, a.Type, "medium score should not have warning alerts")
	}
}
