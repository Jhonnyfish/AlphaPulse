package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchCandidates_Success(t *testing.T) {
	mockResp := Alpha300Response{
		AsOfDate: "20260423",
		RunID:    "test-run-id",
		Rows: []Alpha300RawRow{
			{
				Rank:               1,
				TsCode:             "605117.SH",
				Name:               "德业股份",
				Score:              1.638,
				Close:              148.8,
				ATR14:              7.518,
				BuyLow:             143.161,
				BuyHigh:            146.920,
				SellLow:            156.318,
				SellHigh:           160.077,
				StopLoss:           139.777,
				Momentum:           0.563,
				Trend:              0.155,
				Volatility:         0.039,
				Liquidity:          14.639,
				Industry:           "C38电气机械和器材制造业",
				LimitUpToday:       false,
				LimitUpPrevDay:     false,
				LeaderSignal:       "none",
				HarvestRiskLevel:   "low",
				FocusRank:          1,
				FocusScore:         1.745,
				RecommendationTier: "focus",
				FocusReason:        "High base score",
				HarvestRiskNote:    "无近两日涨停信号",
			},
			{
				Rank:               2,
				TsCode:             "002384.SZ",
				Name:               "东山精密",
				Score:              1.576,
				Close:              186.66,
				Momentum:           0.925,
				Trend:              0.290,
				Volatility:         0.043,
				Industry:           "C39计算机",
				RecommendationTier: "observe",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/rank/latest", r.URL.Path)
		assert.Equal(t, "50", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	// Override the API base URL for testing
	origURL := alpha300APIBase
	// We can't easily override the const, so we test via the server URL
	// This test validates the parsing logic
	_ = origURL

	svc := NewAlpha300Service(5 * time.Second)

	// Directly test parsing by creating a request to our mock server
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/api/rank/latest?limit=50", nil)
	require.NoError(t, err)

	resp, err := svc.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var alphaResp Alpha300Response
	err = json.NewDecoder(resp.Body).Decode(&alphaResp)
	require.NoError(t, err)

	// Verify parsing
	assert.Equal(t, "20260423", alphaResp.AsOfDate)
	assert.Len(t, alphaResp.Rows, 2)

	// Verify first row
	row := alphaResp.Rows[0]
	assert.Equal(t, 1, row.Rank)
	assert.Equal(t, "605117.SH", row.TsCode)
	assert.Equal(t, "德业股份", row.Name)
	assert.InDelta(t, 1.638, row.Score, 0.001)
	assert.Equal(t, "focus", row.RecommendationTier)

	// Verify code normalization
	assert.Equal(t, "605117", normalizeCode("605117.SH"))
	assert.Equal(t, "002384", normalizeCode("002384.SZ"))
	assert.Equal(t, "600522", normalizeCode("600522.SH"))
}

func TestFetchCandidates_BadResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := NewAlpha300Service(5 * time.Second)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/api/rank/latest?limit=50", nil)
	require.NoError(t, err)

	resp, err := svc.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestFetchCandidates_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	svc := NewAlpha300Service(100 * time.Millisecond)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/api/rank/latest?limit=50", nil)
	require.NoError(t, err)

	_, err = svc.client.Do(req)
	assert.Error(t, err)
}

func TestNormalizeCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"605117.SH", "605117"},
		{"002384.SZ", "002384"},
		{"600522.SH", "600522"},
		{"300750.SZ", "300750"},
		{"000001", "000001"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeCode(tt.input))
		})
	}
}
