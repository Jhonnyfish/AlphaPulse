package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDimensions(t *testing.T) {
	analysis := map[string]any{
		"order_flow": map[string]any{
			"score": 75.0,
		},
		"volume_price": map[string]any{
			"score": 60.5,
		},
		"valuation": map[string]any{
			"score": 80,
		},
		"technical": map[string]any{
			"score": "65.5",
		},
		"sector": map[string]any{
			// no score field
			"is_sector_leader": true,
		},
	}

	dims := ParseDimensions(analysis)
	assert.Equal(t, 75.0, dims["order_flow"])
	assert.Equal(t, 60.5, dims["volume_price"])
	assert.Equal(t, 80.0, dims["valuation"])
	assert.Equal(t, 65.5, dims["technical"])
	assert.NotContains(t, dims, "sector")       // no score
	assert.NotContains(t, dims, "volatility")   // not in analysis
}

func TestParseDimensionsEmpty(t *testing.T) {
	dims := ParseDimensions(map[string]any{})
	assert.Empty(t, dims)
}

func TestParseDimensionsNilValues(t *testing.T) {
	analysis := map[string]any{
		"order_flow": nil,
		"volume_price": "invalid",
	}
	dims := ParseDimensions(analysis)
	assert.Empty(t, dims)
}
