package services

import (
	"math"

	"alphapulse/internal/models"
)

// ──────────────────────────────────────────────
// Valuation Analysis types
// ──────────────────────────────────────────────

// ValuationScore holds comprehensive valuation metrics.
type ValuationScore struct {
	PE           float64 `json:"pe"`
	PB           float64 `json:"pb"`
	PELevel      string  `json:"pe_level"`
	PBLevel      string  `json:"pb_level"`
	TotalMV      float64 `json:"total_mv"`
	MVLevel      string  `json:"mv_level"`
	PEPercentile float64 `json:"pe_percentile"` // 0-100, lower = cheaper
	PBPercentile float64 `json:"pb_percentile"`
	Score        int     `json:"score"`  // 0-100
	Signal       string  `json:"signal"` // undervalued / overvalued / fair
	Verdict      string  `json:"verdict"`
}

// ──────────────────────────────────────────────
// Main function
// ──────────────────────────────────────────────

// AnalyzeValuationEnhanced provides enhanced valuation analysis.
func AnalyzeValuationEnhanced(quote models.Quote) ValuationScore {
	pe := quote.PE
	pb := quote.PB
	totalMV := quote.TotalMV

	peLevel := levelPE(pe)
	pbLevel := levelPB(pb)
	mvLevel := classifyMV(totalMV)

	pePct := calculatePercentile(pe, 5, 100)
	pbPct := calculatePercentile(pb, 0.5, 10)

	score := calculateValuationScore(peLevel, pbLevel)

	signal := "fair"
	if score >= 70 {
		signal = "undervalued"
	} else if score <= 30 {
		signal = "overvalued"
	}

	verdict := buildValuationVerdict(peLevel, pbLevel, mvLevel, pe, pb)

	return ValuationScore{
		PE:           round2(pe),
		PB:           round2(pb),
		PELevel:      peLevel,
		PBLevel:      pbLevel,
		TotalMV:      round2(totalMV),
		MVLevel:      mvLevel,
		PEPercentile: round2(pePct),
		PBPercentile: round2(pbPct),
		Score:        score,
		Signal:       signal,
		Verdict:      verdict,
	}
}

// ──────────────────────────────────────────────
// Percentile calculation (log-scale)
// ──────────────────────────────────────────────

func calculatePercentile(value, min, max float64) float64 {
	if value <= 0 {
		return 50
	}
	if value <= min {
		return 5
	}
	if value >= max {
		return 95
	}
	logMin := math.Log(min)
	logMax := math.Log(max)
	logVal := math.Log(value)
	return (logVal-logMin)/(logMax-logMin)*90 + 5
}

// ──────────────────────────────────────────────
// Valuation score
// ──────────────────────────────────────────────

func calculateValuationScore(peLevel, pbLevel string) int {
	score := 50.0

	// PE contribution (50% weight)
	switch peLevel {
	case "偏低":
		score += 20
	case "合理":
		score += 8
	case "偏高":
		score -= 10
	case "很高":
		score -= 20
	case "亏损或无效":
		score -= 12
	}

	// PB contribution (50% weight)
	switch pbLevel {
	case "偏低":
		score += 15
	case "合理":
		score += 5
	case "偏高":
		score -= 8
	case "很高":
		score -= 15
	case "无效":
		score -= 8
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return int(math.Round(score))
}

// ──────────────────────────────────────────────
// Classification helpers
// ──────────────────────────────────────────────

func classifyMV(totalMV float64) string {
	if totalMV >= 1000 {
		return "大盘股"
	} else if totalMV >= 300 {
		return "中大盘股"
	} else if totalMV >= 50 {
		return "中小盘股"
	}
	return "小盘股"
}

func buildValuationVerdict(peLevel, pbLevel, mvLevel string, pe, pb float64) string {
	parts := []string{}

	if pe > 0 {
		parts = append(parts, "PE"+peLevel)
	}
	if pb > 0 {
		parts = append(parts, "PB"+pbLevel)
	}
	parts = append(parts, mvLevel)

	if (peLevel == "偏低" || peLevel == "合理") && (pbLevel == "偏低" || pbLevel == "合理") {
		return joinStringsVal(append(parts, "估值合理"))
	} else if (peLevel == "偏高" || peLevel == "很高") && (pbLevel == "偏高" || pbLevel == "很高") {
		return joinStringsVal(append(parts, "估值偏高"))
	} else if peLevel == "亏损或无效" {
		return joinStringsVal(append(parts, "公司亏损"))
	}
	return joinStringsVal(parts)
}

func joinStringsVal(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += "，" + p
	}
	return result
}
