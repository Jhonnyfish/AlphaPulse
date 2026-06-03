package services

import (
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeTSuggestionPositiveT(t *testing.T) {
	klines := flatThenMoveKlines(100, []float64{99, 98, 97, 96})
	quote := models.Quote{
		Price:     96,
		PrevClose: 100,
		LimitUp:   110,
		LimitDown: 90,
	}

	result := AnalyzeTSuggestion(quote, models.TechnicalAnalysis{}, klines, 2, 900, 100)

	require.NotNil(t, result)
	assert.Equal(t, "正T", result.Type)
	assert.Equal(t, "先买后卖", result.Action)
	assert.True(t, result.Confidence > 55)
	assert.True(t, result.TQuantity >= 100)
	assert.True(t, result.TargetPrice > result.EntryPrice)
	assert.True(t, result.StopLoss < result.EntryPrice)
	assert.True(t, result.RiskReward > 1)
	require.NotNil(t, result.ConditionBuy)
	require.NotNil(t, result.ConditionSell)
	assert.Equal(t, "买入", result.ConditionBuy.Direction)
	assert.Equal(t, "卖出", result.ConditionSell.Direction)
}

func TestAnalyzeTSuggestionReverseT(t *testing.T) {
	klines := flatThenMoveKlines(100, []float64{101, 102, 103, 104})
	quote := models.Quote{
		Price:     104,
		PrevClose: 100,
		LimitUp:   110,
		LimitDown: 90,
	}

	result := AnalyzeTSuggestion(quote, models.TechnicalAnalysis{}, klines, 2, 900, 98)

	require.NotNil(t, result)
	assert.Equal(t, "反T", result.Type)
	assert.Equal(t, "先卖后买", result.Action)
	assert.True(t, result.Confidence > 55)
	assert.True(t, result.TQuantity >= 100)
	assert.True(t, result.TargetPrice < result.EntryPrice)
	assert.True(t, result.StopLoss > result.EntryPrice)
	assert.True(t, result.RiskReward > 1)
	require.NotNil(t, result.ConditionBuy)
	require.NotNil(t, result.ConditionSell)
	assert.Equal(t, "卖出", result.ConditionBuy.Direction)
	assert.Equal(t, "买入", result.ConditionSell.Direction)
}

func TestAnalyzeTSuggestionRequiresPosition(t *testing.T) {
	quote := models.Quote{Price: 100, PrevClose: 100}

	result := AnalyzeTSuggestion(quote, models.TechnicalAnalysis{}, flatThenMoveKlines(100, nil), 2, 0, 0)

	assert.Nil(t, result)
}

func TestAnalyzeTSuggestionSmallPositionNoOrder(t *testing.T) {
	quote := models.Quote{Price: 100, PrevClose: 100}

	result := AnalyzeTSuggestion(quote, models.TechnicalAnalysis{}, flatThenMoveKlines(100, nil), 2, 80, 100)

	require.NotNil(t, result)
	assert.Equal(t, "观望", result.Type)
	assert.Zero(t, result.TQuantity)
	assert.Nil(t, result.ConditionBuy)
	assert.Nil(t, result.ConditionSell)
	assert.NotEmpty(t, result.RiskNotes)
}

func flatThenMoveKlines(base float64, tail []float64) []models.KlinePoint {
	klines := make([]models.KlinePoint, 0, 30)
	flatCount := 30 - len(tail)
	for i := 0; i < flatCount; i++ {
		klines = append(klines, models.KlinePoint{
			Close:  base,
			High:   base + 1,
			Low:    base - 1,
			Volume: 10000,
		})
	}
	for _, close := range tail {
		klines = append(klines, models.KlinePoint{
			Close:  close,
			High:   close + 1,
			Low:    close - 1,
			Volume: 10000,
		})
	}
	return klines
}
