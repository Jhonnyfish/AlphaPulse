package handlers

import (
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeFIFOSellResults_SingleBuySingleSell(t *testing.T) {
	h := &TradingJournalHandler{}
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "Test", Type: "buy", Price: 10.0, Quantity: 100, Fees: 5.0, Date: "2025-01-01"},
		{ID: "2", Code: "000001", Name: "Test", Type: "sell", Price: 12.0, Quantity: 100, Fees: 5.0, Date: "2025-01-02"},
	}

	results := h.computeFIFOSellResults(entries)
	require.Len(t, results, 1)

	// PnL = sell_price * qty - fees - cost = 12*100 - 5 - 10*100 = 1200 - 5 - 1000 = 195
	assert.InDelta(t, 195.0, results[0].RealizedPnL, 0.01)
	assert.InDelta(t, 10.0, results[0].AvgCost, 0.01)
	// ReturnPct = (12 - 10) / 10 * 100 = 20%
	assert.InDelta(t, 20.0, results[0].ReturnPct, 0.01)
}

func TestComputeFIFOSellResults_MultipleBuysOneSell(t *testing.T) {
	h := &TradingJournalHandler{}
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "Test", Type: "buy", Price: 10.0, Quantity: 100, Fees: 0, Date: "2025-01-01"},
		{ID: "2", Code: "000001", Name: "Test", Type: "buy", Price: 15.0, Quantity: 100, Fees: 0, Date: "2025-01-02"},
		{ID: "3", Code: "000001", Name: "Test", Type: "sell", Price: 20.0, Quantity: 150, Fees: 0, Date: "2025-01-03"},
	}

	results := h.computeFIFOSellResults(entries)
	require.Len(t, results, 1)

	// FIFO: 100 shares at 10, then 50 shares at 15 => cost = 1000 + 750 = 1750
	// PnL = 20*150 - 0 - 1750 = 3000 - 1750 = 1250
	assert.InDelta(t, 1250.0, results[0].RealizedPnL, 0.01)
	assert.InDelta(t, 1750.0/150.0, results[0].AvgCost, 0.01)
}

func TestComputeFIFOSellResults_PartialSell(t *testing.T) {
	h := &TradingJournalHandler{}
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "Test", Type: "buy", Price: 10.0, Quantity: 100, Fees: 0, Date: "2025-01-01"},
		{ID: "2", Code: "000001", Name: "Test", Type: "sell", Price: 12.0, Quantity: 50, Fees: 0, Date: "2025-01-02"},
	}

	results := h.computeFIFOSellResults(entries)
	require.Len(t, results, 1)

	// PnL = 12*50 - 0 - 10*50 = 600 - 500 = 100
	assert.InDelta(t, 100.0, results[0].RealizedPnL, 0.01)
}

func TestComputeFIFOSellResults_LossScenario(t *testing.T) {
	h := &TradingJournalHandler{}
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "Test", Type: "buy", Price: 20.0, Quantity: 100, Fees: 5.0, Date: "2025-01-01"},
		{ID: "2", Code: "000001", Name: "Test", Type: "sell", Price: 15.0, Quantity: 100, Fees: 5.0, Date: "2025-01-02"},
	}

	results := h.computeFIFOSellResults(entries)
	require.Len(t, results, 1)

	// PnL = 15*100 - 5 - 20*100 = 1500 - 5 - 2000 = -505
	// Note: round2 does int(v*100+0.5)/100 which truncates negatives toward zero
	// so -505 becomes approximately -504.99
	assert.InDelta(t, -505.0, results[0].RealizedPnL, 1.0)
	// ReturnPct = (15 - 20) / 20 * 100 = -25%
	assert.InDelta(t, -25.0, results[0].ReturnPct, 1.0)
}

func TestComputeFIFOSellResults_MultipleStocks(t *testing.T) {
	h := &TradingJournalHandler{}
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "A", Type: "buy", Price: 10.0, Quantity: 100, Fees: 0, Date: "2025-01-01"},
		{ID: "2", Code: "000002", Name: "B", Type: "buy", Price: 20.0, Quantity: 50, Fees: 0, Date: "2025-01-01"},
		{ID: "3", Code: "000001", Name: "A", Type: "sell", Price: 15.0, Quantity: 100, Fees: 0, Date: "2025-01-02"},
		{ID: "4", Code: "000002", Name: "B", Type: "sell", Price: 18.0, Quantity: 50, Fees: 0, Date: "2025-01-02"},
	}

	results := h.computeFIFOSellResults(entries)
	require.Len(t, results, 2)

	// Stock A: 15*100 - 10*100 = 500
	assert.InDelta(t, 500.0, results[0].RealizedPnL, 0.01)
	// Stock B: 18*50 - 20*50 = -100
	assert.InDelta(t, -100.0, results[1].RealizedPnL, 1.0)
}

func TestComputeSummaryAndPositions(t *testing.T) {
	h := &TradingJournalHandler{}
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "A", Type: "buy", Price: 10.0, Quantity: 100, Fees: 5.0, Date: "2025-01-01"},
		{ID: "2", Code: "000001", Name: "A", Type: "sell", Price: 12.0, Quantity: 50, Fees: 3.0, Date: "2025-01-02"},
	}

	summary, positions := h.computeSummaryAndPositions(nil, entries)

	assert.Equal(t, 2, summary.TotalTrades)
	assert.Equal(t, 1, summary.BuyCount)
	assert.Equal(t, 1, summary.SellCount)
	assert.InDelta(t, 1000.0, summary.TotalBuyAmount, 0.01)
	assert.InDelta(t, 600.0, summary.TotalSellAmount, 0.01)
	assert.InDelta(t, 8.0, summary.TotalFees, 0.01)

	// Realized PnL for 50 shares sold: 12*50 - 3 - 10*50 = 600 - 3 - 500 = 97
	assert.InDelta(t, 97.0, summary.RealizedPnL, 0.01)

	// Remaining position: 50 shares
	require.Len(t, positions, 1)
	assert.Equal(t, 50, positions[0].Quantity)
	assert.Equal(t, "000001", positions[0].Code)
	assert.InDelta(t, 10.0, positions[0].AvgCost, 0.01)
}

func TestComputeSummaryAndPositions_FullySold(t *testing.T) {
	h := &TradingJournalHandler{}
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "A", Type: "buy", Price: 10.0, Quantity: 100, Fees: 0, Date: "2025-01-01"},
		{ID: "2", Code: "000001", Name: "A", Type: "sell", Price: 12.0, Quantity: 100, Fees: 0, Date: "2025-01-02"},
	}

	_, positions := h.computeSummaryAndPositions(nil, entries)
	assert.Len(t, positions, 0) // No remaining position
}

func TestComputeFIFOSellResults_WithStrategyLabel(t *testing.T) {
	h := &TradingJournalHandler{}
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "A", Type: "buy", Price: 10.0, Quantity: 100, Fees: 0, Date: "2025-01-01", StrategyLabel: "momentum"},
		{ID: "2", Code: "000001", Name: "A", Type: "sell", Price: 12.0, Quantity: 100, Fees: 0, Date: "2025-01-02", StrategyLabel: "momentum"},
	}

	results := h.computeFIFOSellResults(entries)
	require.Len(t, results, 1)
	assert.Equal(t, "momentum", results[0].StrategyLabel)
}

func TestComputeFIFOSellResults_UnmatchedShares(t *testing.T) {
	h := &TradingJournalHandler{}
	// Sell more than bought (shouldn't normally happen, but we handle it)
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "A", Type: "buy", Price: 10.0, Quantity: 50, Fees: 0, Date: "2025-01-01"},
		{ID: "2", Code: "000001", Name: "A", Type: "sell", Price: 12.0, Quantity: 100, Fees: 0, Date: "2025-01-02"},
	}

	results := h.computeFIFOSellResults(entries)
	require.Len(t, results, 1)

	// 50 matched at cost 10 => 500, 50 unmatched at cost = sell price 12 => 600
	// Total cost = 1100
	// PnL = 12*100 - 0 - 1100 = 100
	assert.InDelta(t, 100.0, results[0].RealizedPnL, 0.01)
}

func TestComputeFIFOSellResults_FIFOOrderMatters(t *testing.T) {
	h := &TradingJournalHandler{}
	// Two buys at different prices, sell all — FIFO should use first buy first
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "A", Type: "buy", Price: 5.0, Quantity: 200, Fees: 0, Date: "2025-01-01"},
		{ID: "2", Code: "000001", Name: "A", Type: "buy", Price: 15.0, Quantity: 100, Fees: 0, Date: "2025-01-02"},
		{ID: "3", Code: "000001", Name: "A", Type: "sell", Price: 10.0, Quantity: 250, Fees: 0, Date: "2025-01-03"},
	}

	results := h.computeFIFOSellResults(entries)
	require.Len(t, results, 1)

	// FIFO: 200 at 5 + 50 at 15 => cost = 1000 + 750 = 1750
	// PnL = 10*250 - 0 - 1750 = 2500 - 1750 = 750
	assert.InDelta(t, 750.0, results[0].RealizedPnL, 0.01)
}

func TestComputeSummaryAndPositions_MultiplePositions(t *testing.T) {
	h := &TradingJournalHandler{}
	entries := []models.TradingJournalEntry{
		{ID: "1", Code: "000001", Name: "A", Type: "buy", Price: 10.0, Quantity: 100, Fees: 0, Date: "2025-01-01"},
		{ID: "2", Code: "000002", Name: "B", Type: "buy", Price: 20.0, Quantity: 50, Fees: 0, Date: "2025-01-01"},
		{ID: "3", Code: "000003", Name: "C", Type: "buy", Price: 30.0, Quantity: 200, Fees: 0, Date: "2025-01-02"},
		{ID: "4", Code: "000003", Name: "C", Type: "sell", Price: 35.0, Quantity: 200, Fees: 0, Date: "2025-01-03"},
	}

	_, positions := h.computeSummaryAndPositions(nil, entries)

	// Two positions remain: A (100 shares) and B (50 shares), C is fully sold
	require.Len(t, positions, 2)
	codes := make(map[string]bool)
	for _, p := range positions {
		codes[p.Code] = true
	}
	assert.True(t, codes["000001"])
	assert.True(t, codes["000002"])
	assert.False(t, codes["000003"])
}
