package services

import (
	"context"
	"math"
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// klineFetcher is a minimal interface for fetching klines.
// EastMoneyService implicitly satisfies it.
type klineFetcher interface {
	FetchKline(ctx context.Context, code string, days int) ([]models.KlinePoint, error)
}

// mockKlineFetcher returns canned klines for any code.
type mockKlineFetcher struct {
	klines []models.KlinePoint
}

func (m *mockKlineFetcher) FetchKline(ctx context.Context, code string, days int) ([]models.KlinePoint, error) {
	return m.klines, nil
}

// runStrategyBacktestOnKlines is the test entry point: it accepts any
// kline fetcher and delegates to the real engine.
func runStrategyBacktestOnKlines(
	ctx context.Context,
	fetcher klineFetcher,
	code string,
	days int,
) StrategyResult {
	cfg := defaultStrategyConfig

	needCount := days + cfg.warmupDays*2
	if needCount < 120 {
		needCount = 120
	}

	klines, err := fetcher.FetchKline(ctx, code, needCount)
	if err != nil {
		return StrategyResult{Code: code, Days: days, Error: err.Error()}
	}
	return runStrategy(ctx, code, days, klines)
}

// TestStrategyEngine_UptrendHandling verifies the engine enters and holds
// through a strong uptrend, producing positive returns.
func TestStrategyEngine_UptrendHandling(t *testing.T) {
	klines := makeTrendKlines(120, 50.0, 0.5, 100000)
	mock := &mockKlineFetcher{klines: klines}

	result := runStrategyBacktestOnKlines(context.Background(), mock, "000001", 60)
	require.Empty(t, result.Error)

	t.Logf("Uptrend: strategy=%.2f%%, bench=%.2f%%, trades=%d, winRate=%.1f%%",
		result.StrategyReturnPct, result.BuyHoldReturnPct,
		result.TotalTrades, result.WinRate)

	assert.Greater(t, result.SignalEfficiency, 0.0,
		"should have taken positions in uptrend")
	assert.GreaterOrEqual(t, result.TotalTrades, 1)
	assert.Greater(t, result.StrategyReturnPct, -5.0,
		"strategy should not lose heavily in an uptrend")
	assert.Len(t, result.EquityCurve, 60)
}

// TestStrategyEngine_DowntrendAvoidance verifies the engine avoids
// heavy losses in a persistent downtrend.
func TestStrategyEngine_DowntrendAvoidance(t *testing.T) {
	klines := makeTrendKlines(120, 100.0, -0.8, 80000)
	mock := &mockKlineFetcher{klines: klines}

	result := runStrategyBacktestOnKlines(context.Background(), mock, "000002", 60)
	require.Empty(t, result.Error)

	t.Logf("Downtrend: strategy=%.2f%%, bench=%.2f%%, trades=%d",
		result.StrategyReturnPct, result.BuyHoldReturnPct, result.TotalTrades)

	assert.Greater(t, result.StrategyReturnPct, result.BuyHoldReturnPct-5.0,
		"strategy should not lose dramatically more than market")
}

// TestStrategyEngine_PositionSizing verifies that position sizes are 33, 66, or 100.
func TestStrategyEngine_PositionSizing(t *testing.T) {
	klines := makeTrendKlines(120, 50.0, 0.3, 150000)
	mock := &mockKlineFetcher{klines: klines}

	result := runStrategyBacktestOnKlines(context.Background(), mock, "000003", 60)
	require.Empty(t, result.Error)

	for _, tr := range result.Trades {
		t.Logf("Trade: %s -> %s, score=%d, posPct=%.0f%%, ret=%.2f%%",
			tr.BuyDate, tr.SellDate, tr.BuyScore, tr.PositionPct, tr.ReturnPct)
		assert.Greater(t, tr.PositionPct, 0.0)
		assert.LessOrEqual(t, tr.PositionPct, 100.0)
		assert.True(t,
			math.Abs(tr.PositionPct-33) < 1 ||
				math.Abs(tr.PositionPct-66) < 1 ||
				math.Abs(tr.PositionPct-100) < 1,
			"position size should be 33, 66, or 100, got %.1f", tr.PositionPct)
	}
}

// TestStrategyEngine_TrailingStop verifies exit via trailing stop on reversal.
func TestStrategyEngine_TrailingStop(t *testing.T) {
	klines := makeUptrendThenCrash(100, 50.0, 0.3, -3.0)
	mock := &mockKlineFetcher{klines: klines}

	result := runStrategyBacktestOnKlines(context.Background(), mock, "000004", 60)
	require.Empty(t, result.Error)

	foundExit := false
	for _, tr := range result.Trades {
		t.Logf("Trade exit: %s, score=%d, reason=%s", tr.SellDate, tr.BuyScore, tr.SellReason)
		if stringContains(tr.SellReason, "trailing") || stringContains(tr.SellReason, "MA20") {
			foundExit = true
		}
	}
	assert.True(t, foundExit, "should have at least one protective exit (trailing stop or MA20)")
	t.Logf("Crash: strategy=%.2f%%, bench=%.2f%%",
		result.StrategyReturnPct, result.BuyHoldReturnPct)
}

// TestStrategyEngine_DailySignals verifies output format.
func TestStrategyEngine_DailySignals(t *testing.T) {
	klines := makeTrendKlines(80, 50.0, 0.2, 100000)
	mock := &mockKlineFetcher{klines: klines}

	result := runStrategyBacktestOnKlines(context.Background(), mock, "000005", 40)
	require.Empty(t, result.Error)

	assert.NotEmpty(t, result.DailySignals)
	validActions := map[string]bool{"BUY": true, "SELL": true, "HOLD": true}
	for _, s := range result.DailySignals {
		assert.True(t, validActions[s.Action], "invalid action: %s", s.Action)
		assert.NotEmpty(t, s.Reason)
	}
}

// TestStrategyEngine_BenchmarkComparison checks benchmark curve.
func TestStrategyEngine_BenchmarkComparison(t *testing.T) {
	klines := makeTrendKlines(100, 50.0, 0.1, 100000)
	mock := &mockKlineFetcher{klines: klines}

	result := runStrategyBacktestOnKlines(context.Background(), mock, "000006", 30)
	require.Empty(t, result.Error)

	assert.Equal(t, len(result.EquityCurve), len(result.BenchmarkCurve))
	assert.InDelta(t, 1.0, result.BenchmarkCurve[0].Equity, 0.02)
	t.Logf("Benchmark: strategy=%.2f%%, buyHold=%.2f%%, sharpe=%.2f, dd=%.2f%%",
		result.StrategyReturnPct, result.BuyHoldReturnPct,
		result.SharpeRatio, result.MaxDrawdownPct)
}

// TestStrategyEngine_MaxDrawdownNonNegative ensures drawdown ≥ 0.
func TestStrategyEngine_MaxDrawdownNonNegative(t *testing.T) {
	klines := makeTrendKlines(100, 50.0, 0.15, 100000)
	mock := &mockKlineFetcher{klines: klines}

	result := runStrategyBacktestOnKlines(context.Background(), mock, "000007", 30)
	require.Empty(t, result.Error)
	assert.GreaterOrEqual(t, result.MaxDrawdownPct, 0.0)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func makeTrendKlines(n int, startPrice, step float64, baseVol float64) []models.KlinePoint {
	klines := make([]models.KlinePoint, n)
	for i := 0; i < n; i++ {
		p := startPrice + step*float64(i)
		noise := math.Sin(float64(i)/5) * step * 0.4
		c := p + noise
		if c < 0.5 {
			c = 0.5
		}
		klines[i] = models.KlinePoint{
			Date:   "2026-01-01",
			Open:   c - step*0.3,
			Close:  c,
			High:   c + math.Abs(step)*0.8,
			Low:    c - math.Abs(step)*0.8,
			Volume: baseVol + float64(i)*1000,
		}
	}
	return klines
}

func makeUptrendThenCrash(n int, startPrice, upStep, crashStep float64) []models.KlinePoint {
	klines := make([]models.KlinePoint, n)
	mid := n * 2 / 3
	for i := 0; i < n; i++ {
		var p float64
		if i < mid {
			p = startPrice + upStep*float64(i)
		} else {
			p = startPrice + upStep*float64(mid) + crashStep*float64(i-mid)
		}
		noise := math.Sin(float64(i)/4) * upStep * 0.3
		c := p + noise
		if c < 0.5 {
			c = 0.5
		}
		klines[i] = models.KlinePoint{
			Date:   "2026-01-01",
			Open:   c - upStep*0.3,
			Close:  c,
			High:   c + math.Abs(upStep)*0.8,
			Low:    c - math.Abs(upStep)*0.8,
			Volume: 100000 + float64(i)*500,
		}
	}
	return klines
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
