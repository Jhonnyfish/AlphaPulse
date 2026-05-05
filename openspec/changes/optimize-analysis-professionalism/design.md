# Design: 分析引擎优化

## Architecture Overview

```
┌──────────────────────────────────────────────────────────┐
│                    API Layer (Gin)                        │
│  /api/analyze  /api/watchlist-ranking  /api/daily-report │
├──────────────────────────────────────────────────────────┤
│                  Analysis Engine                          │
│  ┌─────────────┬─────────────┬─────────────┐            │
│  │  Technical   │   Fund      │  Valuation  │            │
│  │  Indicators  │   Flow      │  Analysis   │            │
│  │  (MACD/KDJ/  │  (主力/北向/ │  (PE/PB     │            │
│  │   RSI/BB)    │   融资融券)  │   分位数)    │            │
│  └─────────────┴─────────────┴─────────────┘            │
│  ┌─────────────────────────────────────────┐            │
│  │        Scoring Engine (加权评分)          │            │
│  └─────────────────────────────────────────┘            │
├──────────────────────────────────────────────────────────┤
│                  Data Layer                               │
│  EastMoney API │ Tencent API │ K-line Cache │ DB         │
└──────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. 技术指标计算引擎
**Decision**: 新建 `internal/services/indicators.go` 纯计算模块

**Rationale**: 
- 纯函数，无副作用，易于测试
- 输入 []KlinePoint，输出结构化指标值
- 与数据源解耦

```go
type TechnicalIndicators struct {
    MACD    MACDResult
    KDJ     KDJResult
    RSI     RSIResult
    Boll    BollResult
    Volume  VolumeAnalysis
}

func ComputeIndicators(klines []models.KlinePoint) TechnicalIndicators
```

### 2. 资金流向整合
**Decision**: 复用现有 EastMoneyService.FetchMoneyFlow，增加评分逻辑

**Rationale**: 数据源已有，只需增加分析层

```go
type FundFlowScore struct {
    MainFlowScore    int  // 主力净流入评分 0-100
    NorthFlowScore   int  // 北向资金评分 0-100
    MarginScore      int  // 融资融券评分 0-100
    Trend5D          string  // 5日资金趋势
}
```

### 3. 评分引擎重构
**Decision**: 统一评分接口，各维度独立计算后加权汇总

```go
type DimensionScore struct {
    Name    string
    Score   int      // 0-100
    Weight  float64
    Signal  string   // bullish/bearish/neutral
    Details string   // 人类可读的解读
}

type AnalysisResult struct {
    Code        string
    Name        string
    Dimensions  []DimensionScore
    TotalScore  int
    Signal      string
    Suggestion  string
}
```

### 4. 报告模板引擎
**Decision**: 使用 Go template 替代字符串拼接

**Rationale**: 更灵活，支持条件渲染和循环

### 5. 缓存策略
- 技术指标: 5 分钟缓存（交易时段）
- 资金流向: 3 分钟缓存
- 估值数据: 1 小时缓存
- 报告生成: 10 分钟冷却

## Files to Change

### New Files
- `internal/services/indicators.go` — 技术指标计算引擎
- `internal/services/scoring.go` — 统一评分引擎
- `internal/services/valuation.go` — 估值分析

### Modified Files
- `internal/handlers/analyze.go` — 使用新评分引擎
- `internal/handlers/watchlist_analysis.go` — 排名使用新因子
- `internal/handlers/reports.go` — 报告模板升级
- `internal/models/trend.go` — 新增指标模型

### Frontend
- `src/pages/AnalyzePage.tsx` — 展示新维度数据
- `src/pages/RankingPage.tsx` — 展示因子得分
- `src/pages/DailyReportPage.tsx` — 新报告模板

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| EastMoney API 限流 | 数据不完整 | 已有重试+缓存+降级机制 |
| K线数据不足 | 指标计算失败 | 最少 20 根 K 线，不足时返回默认值 |
| 计算耗时 | 接口响应慢 | 并发计算 + 结果缓存 |
