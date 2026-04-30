package services

import (
	"alphapulse/internal/models"
)

// ──────────────────────────────────────────────
// Volume Analysis types
// ──────────────────────────────────────────────

// VolumeAnalysis holds volume-related indicators.
type VolumeAnalysis struct {
	VolumeRatio  float64 `json:"volume_ratio"`  // 今日成交量 / 5日均量
	Trend5D      string  `json:"trend5d"`        // increasing / decreasing / stable
	Trend10D     string  `json:"trend10d"`       // increasing / decreasing / stable
	Coordination string  `json:"coordination"`   // strong / weak / divergence / normal
	Score        int     `json:"score"`          // 0-100
	Signal       string  `json:"signal"`         // bullish / bearish / neutral
}

// ──────────────────────────────────────────────
// Main function
// ──────────────────────────────────────────────

// AnalyzeVolume computes volume analysis from kline data.
func AnalyzeVolume(klines []models.KlinePoint) VolumeAnalysis {
	if len(klines) < 5 {
		return VolumeAnalysis{Score: 50, Signal: "neutral", Coordination: "normal"}
	}

	volumes := extractVolumes(klines)
	closes := extractCloses(klines)

	ratio := computeVolumeRatio(volumes)
	trend5 := computeVolumeTrendN(volumes, 5)
	trend10 := computeVolumeTrendN(volumes, 10)
	coord := computeVolumePriceCoordination(closes, volumes)

	score, signal := computeVolumeScore(ratio, trend5, coord, closes)

	return VolumeAnalysis{
		VolumeRatio:  round2(ratio),
		Trend5D:      trend5,
		Trend10D:     trend10,
		Coordination: coord,
		Score:        score,
		Signal:       signal,
	}
}

// ──────────────────────────────────────────────
// Volume Ratio
// ──────────────────────────────────────────────

// computeVolumeRatio = today's volume / 5-day average volume.
func computeVolumeRatio(volumes []float64) float64 {
	if len(volumes) < 6 {
		return 1.0
	}

	today := volumes[len(volumes)-1]
	avg5 := 0.0
	for _, v := range volumes[len(volumes)-6 : len(volumes)-1] {
		avg5 += v
	}
	avg5 /= 5

	if avg5 == 0 {
		return 1.0
	}
	return today / avg5
}

// ──────────────────────────────────────────────
// Volume Trend
// ──────────────────────────────────────────────

// computeVolumeTrendN checks if volume is increasing/decreasing over N days.
func computeVolumeTrendN(volumes []float64, n int) string {
	if len(volumes) < n+1 {
		return "stable"
	}

	recent := volumes[len(volumes)-n:]
	half := n / 2
	if half < 1 {
		half = 1
	}

	avgRecent := avgFloat(recent[len(recent)-half:])
	avgOlder := avgFloat(recent[:len(recent)-half])

	if avgOlder == 0 {
		return "stable"
	}

	ratio := avgRecent / avgOlder
	if ratio > 1.15 {
		return "increasing"
	} else if ratio < 0.85 {
		return "decreasing"
	}
	return "stable"
}

// ──────────────────────────────────────────────
// Volume-Price Coordination
// ──────────────────────────────────────────────

// computeVolumePriceCoordination checks if price and volume move together.
func computeVolumePriceCoordination(closes, volumes []float64) string {
	if len(closes) < 6 || len(volumes) < 6 {
		return "normal"
	}

	// Compare last 5 days
	recentCloses := closes[len(closes)-5:]
	recentVolumes := volumes[len(volumes)-5:]

	priceChange := (recentCloses[4] - recentCloses[0]) / recentCloses[0]
	volumeChange := (recentVolumes[4] - recentVolumes[0]) / recentVolumes[0]

	// Both rising strongly
	if priceChange > 0.02 && volumeChange > 0.15 {
		return "strong"
	}

	// Price rising but volume declining = divergence
	if priceChange > 0.02 && volumeChange < -0.15 {
		return "divergence"
	}

	// Price falling with increasing volume = weak
	if priceChange < -0.02 && volumeChange > 0.15 {
		return "weak"
	}

	// Price falling with decreasing volume = healthy pullback
	if priceChange < -0.02 && volumeChange < -0.15 {
		return "normal"
	}

	return "normal"
}

// ──────────────────────────────────────────────
// Volume Score
// ──────────────────────────────────────────────

func computeVolumeScore(ratio float64, trend, coord string, closes []float64) (int, string) {
	score := 50
	signal := "neutral"

	priceUp := len(closes) >= 2 && closes[len(closes)-1] > closes[len(closes)-2]

	// Volume ratio contribution
	if ratio > 1.5 {
		if priceUp {
			score += 15 // heavy volume up
		} else {
			score -= 10 // heavy volume down
		}
	} else if ratio < 0.5 {
		score -= 5 // very light volume
	}

	// Trend contribution
	switch trend {
	case "increasing":
		if priceUp {
			score += 10
		}
	case "decreasing":
		if !priceUp {
			score -= 5
		}
	}

	// Coordination contribution
	switch coord {
	case "strong":
		score += 15
		signal = "bullish"
	case "divergence":
		score -= 15
		signal = "bearish"
	case "weak":
		score -= 10
		signal = "bearish"
	}

	// Clamp
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	// Override signal based on final score
	if signal == "neutral" {
		if score >= 65 {
			signal = "bullish"
		} else if score <= 35 {
			signal = "bearish"
		}
	}

	return score, signal
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func extractVolumes(klines []models.KlinePoint) []float64 {
	result := make([]float64, len(klines))
	for i, k := range klines {
		result[i] = k.Volume
	}
	return result
}

func avgFloat(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}
