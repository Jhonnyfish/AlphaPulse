package services

import (
	"fmt"
	"testing"

	"alphapulse/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Kline builders for synthetic test scenarios
// ──────────────────────────────────────────────

// klinesFromCloses builds n KlinePoints where OHLC are all derived from the
// given close array. Volume defaults to 1000 for every bar.
func klinesFromCloses(closes []float64) []models.KlinePoint {
	out := make([]models.KlinePoint, len(closes))
	for i, c := range closes {
		out[i] = models.KlinePoint{
			Date:   fmt.Sprintf("2026-01-%02d", (i%28)+1),
			Open:   c,
			Close:  c,
			High:   c + 0.1,
			Low:    c - 0.1,
			Volume: 1000,
		}
	}
	return out
}

// withVol returns a copy of klines with volumes overridden at the given indices.
func withVol(klines []models.KlinePoint, vols map[int]float64) []models.KlinePoint {
	out := append([]models.KlinePoint(nil), klines...)
	for i, v := range vols {
		if i >= 0 && i < len(out) {
			out[i].Volume = v
		}
	}
	return out
}

// klineBuilder incrementally constructs a close-price slice for tests.
// Chaining moveTo(target, bars) appends `bars` linearly interpolated bars
// ending exactly at `target`. holdAt(bars) repeats the last close.
type klineBuilder struct {
	closes []float64
}

func newKlineBuilder(start float64) *klineBuilder {
	return &klineBuilder{closes: []float64{start}}
}

func (b *klineBuilder) moveTo(target float64, bars int) *klineBuilder {
	start := b.closes[len(b.closes)-1]
	for k := 1; k <= bars; k++ {
		frac := float64(k) / float64(bars)
		b.closes = append(b.closes, start+(target-start)*frac)
	}
	return b
}

func (b *klineBuilder) holdAt(bars int) *klineBuilder {
	last := b.closes[len(b.closes)-1]
	for k := 0; k < bars; k++ {
		b.closes = append(b.closes, last)
	}
	return b
}

func (b *klineBuilder) build() []float64 { return b.closes }

// ──────────────────────────────────────────────
// Test scenarios for double top (M-top)
// ──────────────────────────────────────────────

// validDoubleTop builds a 60-bar scenario with a textbook confirmed M-top:
//
//	bars  0..19 : uptrend 30 → 49      (prior uptrend)
//	bar   20    : peak1  = 50
//	bars 21..29 : decline to 41        (neckline area)
//	bar   30    : trough = 41
//	bars 31..39 : rally back to 49
//	bar   40    : peak2  = 50          (within last 25 bars)
//	bars 41..49 : decline breaking neckline to 39
//	bars 50..59 : drift at 39           (below neckline 41*0.97=39.77 ✓)
//
// Measured target = 41 - (50-41) = 32. Current 39 > target*0.95=30.4 ✓.
func validDoubleTop() []float64 {
	return newKlineBuilder(30).
		moveTo(50, 20). // bars 1..20   uptrend ending at peak1
		moveTo(41, 10). // bars 21..30  decline to trough (neckline)
		moveTo(50, 10). // bars 31..40  rally to peak2
		moveTo(39, 10). // bars 41..50  decline breaking neckline
		holdAt(9).      // bars 51..59  drift (last bar)
		build()
}

// ──────────────────────────────────────────────
// Double Top — strict detector tests
// ──────────────────────────────────────────────

func TestDetectDoubleTopStrict_Valid(t *testing.T) {
	closes := validDoubleTop()
	// Boost volume at peak1 (2000) and reduce at peak2 (800) for confirmation.
	klines := withVol(klinesFromCloses(closes), map[int]float64{20: 2000, 40: 800})

	patterns := DetectPatterns(klines)

	require.NotEmpty(t, patterns)
	var top *models.PatternSignal
	for i := range patterns {
		if patterns[i].Pattern == "双顶" {
			top = &patterns[i]
			break
		}
	}
	require.NotNil(t, top, "should detect double top")
	assert.Equal(t, "bearish", top.Direction)
	assert.Equal(t, "chart", top.Category)
	assert.True(t, top.Confidence >= 0.85, "with volume confirmation confidence should be ≥ 0.85, got %f", top.Confidence)
	assert.Contains(t, top.Description, "颈线")
}

func TestDetectDoubleTopStrict_NoNecklineBreak(t *testing.T) {
	// Same pattern but current price stays above neckline → not confirmed.
	closes := validDoubleTop()
	// Override the last 10 bars to drift at 43 (above neckline 41).
	for i := 50; i < 60; i++ {
		closes[i] = 43
	}
	klines := klinesFromCloses(closes)

	for _, p := range DetectPatterns(klines) {
		if p.Pattern == "双顶" {
			t.Fatalf("should NOT detect double top without neckline break: %+v", p)
		}
	}
}

func TestDetectDoubleTopStrict_Stale(t *testing.T) {
	// Peak2 too old: shift the pattern so peak2 sits at bar 40 but the
	// window has 60 bars ending well past it (current = bar 60, peak2 = bar 40
	// → 19 bars apart, still recent). To force a stale pattern, move peak2
	// earlier by adding a long flat tail after bar 40.
	closes := validDoubleTop()
	// Replace bars 40..59 with: peak2 at bar 30, then long flat decline.
	closes = make([]float64, 60)
	// Build: uptrend + peak1 at bar 15, trough at 22, peak2 at 30, then 30 flat bars.
	for i := 0; i < 60; i++ {
		switch {
		case i <= 14:
			closes[i] = 30 + float64(i)*1.3 // 30 → 48.2
		case i == 15:
			closes[i] = 50 // peak1
		case i <= 21:
			closes[i] = 50 - float64(i-15)*1.5 // decline
		case i == 22:
			closes[i] = 41 // trough (neckline)
		case i <= 29:
			closes[i] = 41 + float64(i-22)*1.1 // rally
		case i == 30:
			closes[i] = 50 // peak2 (30 bars ago)
		case i <= 39:
			closes[i] = 50 - float64(i-30)*1.0 // decline
		default:
			closes[i] = 39 // stable, below neckline
		}
	}
	klines := klinesFromCloses(closes)

	for _, p := range DetectPatterns(klines) {
		if p.Pattern == "双顶" {
			t.Fatalf("should NOT detect stale double top (peak2 too old): %+v", p)
		}
	}
}

func TestDetectDoubleTopStrict_NoPriorUptrend(t *testing.T) {
	// Flat history before peak1 — fails prior-uptrend check.
	closes := make([]float64, 60)
	for i := 0; i < 60; i++ {
		switch {
		case i <= 19:
			closes[i] = 50 // flat, no uptrend
		case i == 20:
			closes[i] = 52 // peak1
		case i <= 29:
			closes[i] = 52 - float64(i-20)*1.1
		case i == 30:
			closes[i] = 41 // trough
		case i <= 39:
			closes[i] = 41 + float64(i-30)*1.1
		case i == 40:
			closes[i] = 52 // peak2
		case i <= 49:
			closes[i] = 52 - float64(i-40)*1.3
		default:
			closes[i] = 39 // confirmed below neckline 41*0.97=39.77
		}
	}
	klines := klinesFromCloses(closes)

	for _, p := range DetectPatterns(klines) {
		if p.Pattern == "双顶" {
			t.Fatalf("should NOT detect double top without prior uptrend: %+v", p)
		}
	}
}

func TestDetectDoubleTopStrict_AlreadyPlayedOut(t *testing.T) {
	// Current price already well past the measured target (32) — pattern has
	// completed, no longer actionable.
	closes := validDoubleTop()
	// Override last 10 bars to 25 (far below target*0.95=30.4).
	for i := 50; i < 60; i++ {
		closes[i] = 25
	}
	klines := klinesFromCloses(closes)

	for _, p := range DetectPatterns(klines) {
		if p.Pattern == "双顶" {
			t.Fatalf("should NOT detect played-out double top: %+v", p)
		}
	}
}

// ──────────────────────────────────────────────
// Double Bottom — mirror tests
// ──────────────────────────────────────────────

// validDoubleBottom builds a 60-bar confirmed W-bottom:
//
//	bars  0..19 : downtrend 50 → 31    (prior downtrend)
//	bar   20    : trough1 = 30
//	bars 21..29 : rally to 40
//	bar   30    : peak   = 40 (neckline)
//	bars 31..39 : decline back to 31
//	bar   40    : trough2 = 30
//	bars 41..49 : rally breaking neckline to 42
//	bars 50..59 : drift at 42            (above neckline 40*1.03=41.2 ✓)
//
// Measured target = 40 + (40-30) = 50. Current 42 < target*1.05=52.5 ✓.
func validDoubleBottom() []float64 {
	return newKlineBuilder(50).
		moveTo(30, 20). // bars 1..20   downtrend ending at trough1
		moveTo(40, 10). // bars 21..30  rally to neckline
		moveTo(30, 10). // bars 31..40  decline to trough2
		moveTo(42, 10). // bars 41..50  rally breaking neckline
		holdAt(9).      // bars 51..59  drift
		build()
}

func TestDetectDoubleBottomStrict_Valid(t *testing.T) {
	closes := validDoubleBottom()
	klines := withVol(klinesFromCloses(closes), map[int]float64{20: 800, 40: 2000})

	patterns := DetectPatterns(klines)

	require.NotEmpty(t, patterns)
	var bot *models.PatternSignal
	for i := range patterns {
		if patterns[i].Pattern == "双底" {
			bot = &patterns[i]
			break
		}
	}
	require.NotNil(t, bot, "should detect double bottom")
	assert.Equal(t, "bullish", bot.Direction)
	assert.Equal(t, "chart", bot.Category)
	assert.True(t, bot.Confidence >= 0.75, "confidence should be ≥ 0.75, got %f", bot.Confidence)
}

func TestDetectDoubleBottomStrict_NoNecklineBreak(t *testing.T) {
	closes := validDoubleBottom()
	for i := 50; i < 60; i++ {
		closes[i] = 38 // still below neckline 40
	}
	klines := klinesFromCloses(closes)

	for _, p := range DetectPatterns(klines) {
		if p.Pattern == "双底" {
			t.Fatalf("should NOT detect double bottom without neckline break: %+v", p)
		}
	}
}

// ──────────────────────────────────────────────
// Head & Shoulders Top — strict detector test
// ──────────────────────────────────────────────

// validHeadShouldersTop builds a 60-bar confirmed H&S top:
//
//	bars  0..19 : uptrend 30 → 49
//	bar   20    : left shoulder  = 50
//	bars 21..24 : decline to ~46
//	bar   25    : neckline trough1 = 46
//	bars 26..29 : rally to ~55
//	bar   30    : head = 56
//	bars 31..34 : decline to ~46
//	bar   35    : neckline trough2 = 46
//	bars 36..39 : rally to ~50
//	bar   40    : right shoulder = 50
//	bars 41..49 : decline breaking neckline (46) to 43
//	bars 50..59 : drift at 43
//
// Neckline = 46. Current 43 < 46*0.97 = 44.62 ✓.
// Target = 46 - (56-46) = 36. Current 43 > 36*0.95 = 34.2 ✓.
func validHeadShouldersTop() []float64 {
	return newKlineBuilder(30).
		moveTo(50, 20). // 1..20   uptrend ending at LS
		moveTo(46, 5).  // 21..25  decline to neckline trough1
		moveTo(56, 5).  // 26..30  rally to head
		moveTo(46, 5).  // 31..35  decline to neckline trough2
		moveTo(50, 5).  // 36..40  rally to RS
		moveTo(43, 10). // 41..50  decline breaking neckline
		holdAt(9).      // 51..59  drift
		build()
}

func TestDetectHeadShouldersTopStrict_Valid(t *testing.T) {
	closes := validHeadShouldersTop()
	// LS volume high, RS volume low (classic H&S volume pattern).
	klines := withVol(klinesFromCloses(closes), map[int]float64{20: 2000, 40: 800})

	patterns := DetectPatterns(klines)

	var hst *models.PatternSignal
	for i := range patterns {
		if patterns[i].Pattern == "头肩顶" {
			hst = &patterns[i]
			break
		}
	}
	require.NotNil(t, hst, "should detect head and shoulders top")
	assert.Equal(t, "bearish", hst.Direction)
	assert.Contains(t, hst.Description, "颈线")
}

func TestDetectHeadShouldersTopStrict_NoNecklineBreak(t *testing.T) {
	closes := validHeadShouldersTop()
	for i := 50; i < 60; i++ {
		closes[i] = 47 // above neckline 46
	}
	klines := klinesFromCloses(closes)

	for _, p := range DetectPatterns(klines) {
		if p.Pattern == "头肩顶" {
			t.Fatalf("should NOT detect H&S top without neckline break: %+v", p)
		}
	}
}

// ──────────────────────────────────────────────
// Regression — Tianci Lithium (002709) bug scenario
// ──────────────────────────────────────────────

// TestRegression_StalePatternNotReported reproduces the user-reported bug:
// a stale M-top with peaks ~48.79 and neckline ~41.36 was being reported even
// though the pattern was no longer actionable. The fix requires the second
// peak to be within the last 25 bars.
func TestRegression_StalePatternNotReported(t *testing.T) {
	// Build a 60-bar chart with the M-top early in the window and the last
	// 35+ bars drifting well below the neckline (pattern already played out
	// AND stale — either condition alone should suppress).
	closes := make([]float64, 60)
	for i := 0; i < 60; i++ {
		switch {
		case i <= 9:
			closes[i] = 40 + float64(i)*0.8 // 40 → 47.2  (prior uptrend)
		case i == 10:
			closes[i] = 48.79 // peak1
		case i <= 17:
			closes[i] = 48.79 - float64(i-10)*1.0
		case i == 18:
			closes[i] = 41.36 // trough (neckline)
		case i <= 25:
			closes[i] = 41.36 + float64(i-18)*0.97
		case i == 26:
			closes[i] = 48.79 // peak2 — STALE: 33 bars before end
		case i <= 35:
			closes[i] = 48.79 - float64(i-26)*1.0
		default:
			closes[i] = 38 // long stable tail, below neckline
		}
	}
	klines := klinesFromCloses(closes)

	for _, p := range DetectPatterns(klines) {
		if p.Pattern == "双顶" {
			t.Fatalf("regression: stale M-top should not be reported: %+v", p)
		}
	}
}
