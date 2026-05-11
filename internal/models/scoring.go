package models

// ScoringModel defines the weighted scoring configuration.
// Each dimension has a weight (0-1) that determines its contribution to the overall score.
// Weights should sum to 1.0 for proper normalization.
type ScoringModel struct {
	// Dimension weights - must sum to 1.0
	Weights DimensionWeights `json:"weights"`
	// Period-specific weights for multi-timeframe analysis
	PeriodWeights map[string]DimensionWeights `json:"period_weights"`
}

// DimensionWeights holds the weight for each analysis dimension.
type DimensionWeights struct {
	OrderFlow    float64 `json:"order_flow"`    // 委托流
	VolumePrice  float64 `json:"volume_price"`  // 量价
	Valuation    float64 `json:"valuation"`     // 估值
	Volatility   float64 `json:"volatility"`    // 波动
	MoneyFlow    float64 `json:"money_flow"`    // 资金流
	Technical    float64 `json:"technical"`     // 技术面
	Sector       float64 `json:"sector"`        // 板块
	Sentiment    float64 `json:"sentiment"`     // 情绪
	Fundamentals float64 `json:"fundamentals"`  // 基本面
	Northbound   float64 `json:"northbound"`    // 北向
	Margin       float64 `json:"margin"`        // 融资
}

// DefaultScoringModel returns the default scoring model with balanced weights.
func DefaultScoringModel() ScoringModel {
	// Default weights - technical and money flow weighted higher for short-term
	defaultWeights := DimensionWeights{
		OrderFlow:    0.05, // 5% - 内外盘信号较弱
		VolumePrice:  0.12, // 12% - 量价关系重要
		Valuation:    0.08, // 8% - 估值中长期重要
		Volatility:   0.05, // 5% - 波动率辅助
		MoneyFlow:    0.15, // 15% - 资金流向核心
		Technical:    0.15, // 15% - 技术面核心
		Sector:       0.08, // 8% - 板块轮动
		Sentiment:    0.07, // 7% - 情绪面
		Fundamentals: 0.12, // 12% - 基本面中长期
		Northbound:   0.08, // 8% - 北向资金
		Margin:       0.05, // 5% - 融资融券
	}
	// Sum = 1.0

	// Short-term (5-day) - emphasize technical, money flow, volume
	shortTerm := DimensionWeights{
		OrderFlow:    0.08,
		VolumePrice:  0.18,
		Valuation:    0.03,
		Volatility:   0.08,
		MoneyFlow:    0.20,
		Technical:    0.20,
		Sector:       0.05,
		Sentiment:    0.05,
		Fundamentals: 0.03,
		Northbound:   0.05,
		Margin:       0.05,
	}

	// Medium-term (20-day) - balanced
	mediumTerm := DimensionWeights{
		OrderFlow:    0.05,
		VolumePrice:  0.12,
		Valuation:    0.10,
		Volatility:   0.05,
		MoneyFlow:    0.15,
		Technical:    0.15,
		Sector:       0.10,
		Sentiment:    0.08,
		Fundamentals: 0.10,
		Northbound:   0.05,
		Margin:       0.05,
	}

	// Long-term (60-day) - emphasize fundamentals, valuation
	longTerm := DimensionWeights{
		OrderFlow:    0.02,
		VolumePrice:  0.08,
		Valuation:    0.18,
		Volatility:   0.03,
		MoneyFlow:    0.08,
		Technical:    0.10,
		Sector:       0.12,
		Sentiment:    0.05,
		Fundamentals: 0.22,
		Northbound:   0.07,
		Margin:       0.05,
	}

	return ScoringModel{
		Weights: defaultWeights,
		PeriodWeights: map[string]DimensionWeights{
			"short":  shortTerm,
			"medium": mediumTerm,
			"long":   longTerm,
		},
	}
}

// MultiPeriodScore holds scores for different time horizons.
type MultiPeriodScore struct {
	Short  float64 `json:"short"`  // 5-day score
	Medium float64 `json:"medium"` // 20-day score
	Long   float64 `json:"long"`   // 60-day score
}

// Confidence represents data completeness and reliability.
type Confidence struct {
	Overall  float64            `json:"overall"`  // 0-100 overall confidence
	ByDim    map[string]float64 `json:"by_dim"`   // per-dimension confidence
	DataAge  string             `json:"data_age"` // "today", "yesterday", "stale"
	Missing  []string           `json:"missing"`  // missing data dimensions
}

// EnhancedSummary extends AnalysisSummary with weighted scoring.
type EnhancedSummary struct {
	AnalysisSummary
	// Weighted overall score (0-100)
	WeightedScore float64 `json:"weighted_score"`
	// Multi-period scores
	PeriodScores MultiPeriodScore `json:"period_scores"`
	// Data confidence
	Confidence Confidence `json:"confidence"`
	// Per-dimension weighted contributions
	DimContributions map[string]float64 `json:"dim_contributions"`
	// Score trend (compared to previous analysis)
	ScoreTrend string `json:"score_trend"` // "rising", "falling", "stable"
}
