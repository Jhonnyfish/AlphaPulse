package services

import (
	"math"

	"alphapulse/internal/models"
)

// ScoringEngine computes weighted, multi-period scores with confidence.
type ScoringEngine struct {
	model models.ScoringModel
}

// NewScoringEngine creates a scoring engine with the default model.
func NewScoringEngine() *ScoringEngine {
	return &ScoringEngine{
		model: models.DefaultScoringModel(),
	}
}

// DimensionScores holds the raw 0-100 score for each dimension.
type DimensionScores struct {
	OrderFlow    float64
	VolumePrice  float64
	Valuation    float64
	Volatility   float64
	MoneyFlow    float64
	Technical    float64
	Sector       float64
	Sentiment    float64
	Fundamentals float64
	Northbound   float64
	Margin       float64
}

// ComputeEnhancedSummary produces a full enhanced summary with weighted scoring.
func (e *ScoringEngine) ComputeEnhancedSummary(
	a *models.StockAnalysis,
	dimScores DimensionScores,
) models.EnhancedSummary {
	return e.ComputeEnhancedSummaryWithStrategy(a, dimScores, models.StrategyDefault)
}

// ComputeEnhancedSummaryWithStrategy produces a full enhanced summary with a specific strategy.
func (e *ScoringEngine) ComputeEnhancedSummaryWithStrategy(
	a *models.StockAnalysis,
	dimScores DimensionScores,
	strategy models.ScoringStrategy,
) models.EnhancedSummary {
	// 1. Build base summary (existing logic)
	base := BuildSummary(a)

	// 2. Compute weighted score with strategy or default weights
	weights := e.model.Weights
	if strategy != models.StrategyDefault {
		if sw, ok := e.model.StrategyWeights[string(strategy)]; ok {
			weights = sw
		}
	}
	weightedScore := e.computeWeightedScore(dimScores, weights)

	// 3. Compute multi-period scores
	periodScores := models.MultiPeriodScore{
		Short:  e.computeWeightedScore(dimScores, e.model.PeriodWeights["short"]),
		Medium: e.computeWeightedScore(dimScores, e.model.PeriodWeights["medium"]),
		Long:   e.computeWeightedScore(dimScores, e.model.PeriodWeights["long"]),
	}

	// 4. Compute confidence
	confidence := e.computeConfidence(a)

	// 5. Compute per-dimension contributions
	dimContribs := e.computeContributions(dimScores, weights)

	// 6. Override overall score with weighted score
	base.OverallScore = int(math.Round(weightedScore))

	return models.EnhancedSummary{
		AnalysisSummary:  base,
		WeightedScore:    weightedScore,
		PeriodScores:     periodScores,
		Confidence:       confidence,
		DimContributions: dimContribs,
		ScoreTrend:       "stable", // TODO: compare with historical
	}
}

// computeWeightedScore applies weighted average to dimension scores.
func (e *ScoringEngine) computeWeightedScore(
	scores DimensionScores,
	weights models.DimensionWeights,
) float64 {
	total := scores.OrderFlow*weights.OrderFlow +
		scores.VolumePrice*weights.VolumePrice +
		scores.Valuation*weights.Valuation +
		scores.Volatility*weights.Volatility +
		scores.MoneyFlow*weights.MoneyFlow +
		scores.Technical*weights.Technical +
		scores.Sector*weights.Sector +
		scores.Sentiment*weights.Sentiment +
		scores.Fundamentals*weights.Fundamentals +
		scores.Northbound*weights.Northbound +
		scores.Margin*weights.Margin

	return math.Round(total*10) / 10 // 1 decimal place
}

// computeConfidence evaluates data completeness and reliability.
func (e *ScoringEngine) computeConfidence(a *models.StockAnalysis) models.Confidence {
	byDim := make(map[string]float64)
	var missing []string

	// Check each dimension for data availability
	checks := []struct {
		name    string
		checkFn func() float64
	}{
		{"order_flow", func() float64 {
			if a.OrderFlow.Verdict == "数据不足" || a.OrderFlow.Verdict == "" {
				return 0
			}
			return 100
		}},
		{"volume_price", func() float64 {
			if a.VolumePrice.VolumeRatio == 0 {
				return 30
			}
			return 100
		}},
		{"valuation", func() float64 {
			if a.Valuation.PE == 0 && a.Valuation.PB == 0 {
				return 0
			}
			return 100
		}},
		{"volatility", func() float64 {
			if a.Volatility.Amplitude == 0 {
				return 30
			}
			return 100
		}},
		{"money_flow", func() float64 {
			if a.MoneyFlow.TodayMainDirection == "" || a.MoneyFlow.TodayMainDirection == "持平" {
				return 40
			}
			return 100
		}},
		{"technical", func() float64 {
			if a.Technical.RSI_14 == 0 {
				return 20
			}
			return 100
		}},
		{"sector", func() float64 {
			if a.Sector.PrimarySector == "" {
				return 0
			}
			return 100
		}},
		{"sentiment", func() float64 {
			if a.Sentiment.NewsCount == 0 && a.Sentiment.AnnouncementCount == 0 {
				return 20
			}
			return 100
		}},
		{"fundamentals", func() float64 {
			if a.Fundamentals.Score == 0 && a.Fundamentals.Verdict == "暂无财务数据" {
				return 0
			}
			return 100
		}},
		{"northbound", func() float64 {
			if a.Northbound.Signal == "无明显信号" || a.Northbound.Signal == "" {
				return 40
			}
			return 100
		}},
		{"margin", func() float64 {
			if a.MarginDetail.Signal == "中性" || a.MarginDetail.Signal == "" {
				return 40
			}
			return 100
		}},
	}

	totalConf := 0.0
	for _, c := range checks {
		conf := c.checkFn()
		byDim[c.name] = conf
		totalConf += conf
		if conf == 0 {
			missing = append(missing, c.name)
		}
	}

	overallConf := totalConf / float64(len(checks))

	// Determine data age
	dataAge := "today"
	if !a.FetchedAt.IsZero() {
		// Could check against current time, but keep simple for now
		dataAge = "today"
	}

	return models.Confidence{
		Overall: math.Round(overallConf*10) / 10,
		ByDim:   byDim,
		DataAge: dataAge,
		Missing: missing,
	}
}

// computeContributions shows how much each dimension contributes to the final score.
func (e *ScoringEngine) computeContributions(
	scores DimensionScores,
	weights models.DimensionWeights,
) map[string]float64 {
	contribs := make(map[string]float64)

	// Each contribution = (score - 50) * weight
	// This shows how much each dimension moves the score from neutral (50)
	contribs["order_flow"] = math.Round((scores.OrderFlow-50)*weights.OrderFlow*10) / 10
	contribs["volume_price"] = math.Round((scores.VolumePrice-50)*weights.VolumePrice*10) / 10
	contribs["valuation"] = math.Round((scores.Valuation-50)*weights.Valuation*10) / 10
	contribs["volatility"] = math.Round((scores.Volatility-50)*weights.Volatility*10) / 10
	contribs["money_flow"] = math.Round((scores.MoneyFlow-50)*weights.MoneyFlow*10) / 10
	contribs["technical"] = math.Round((scores.Technical-50)*weights.Technical*10) / 10
	contribs["sector"] = math.Round((scores.Sector-50)*weights.Sector*10) / 10
	contribs["sentiment"] = math.Round((scores.Sentiment-50)*weights.Sentiment*10) / 10
	contribs["fundamentals"] = math.Round((scores.Fundamentals-50)*weights.Fundamentals*10) / 10
	contribs["northbound"] = math.Round((scores.Northbound-50)*weights.Northbound*10) / 10
	contribs["margin"] = math.Round((scores.Margin-50)*weights.Margin*10) / 10

	return contribs
}
