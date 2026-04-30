# Design: Alpha300 数据访问层

## Architecture Overview

```
┌─────────────────────────────────────────────┐
│           AlphaPulse Services               │
├─────────────────────────────────────────────┤
│  AnalysisHandler  │  ReportsHandler  │  ... │
├─────────────────────────────────────────────┤
│           Alpha300DBService                 │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐      │
│  │ KlineDB │ │ FlowDB  │ │BasicDB  │      │
│  └────┬────┘ └────┬────┘ └────┬────┘      │
│       │           │           │            │
│  ┌────▼───────────▼───────────▼────┐      │
│  │      In-Memory Cache (TTL)      │      │
│  └────────────────┬────────────────┘      │
├───────────────────┼────────────────────────┤
│                   │                         │
│  ┌────────────────▼────────────────┐      │
│  │   PostgreSQL Connection Pool    │      │
│  │   (Read-Only, alpha300 DB)      │      │
│  └─────────────────────────────────┘      │
└─────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. 只读连接
**Decision**: 使用只读连接，不写入 Alpha300 数据库

**Rationale**: 用户明确要求只读访问，避免影响 Alpha300 系统

### 2. 独立连接池
**Decision**: 为 Alpha300 创建独立的 pgxpool 连接池

**Rationale**: 
- 与 AlphaPulse 自有数据库隔离
- 可以配置不同的连接参数
- 便于监控和故障排查

### 3. 内存缓存
**Decision**: 使用现有 cache 包，为不同类型数据设置不同 TTL

| 数据类型 | TTL | 原因 |
|---------|-----|------|
| 日K线 | 5 分钟 | 交易日数据相对稳定 |
| 估值数据 | 10 分钟 | 变化频率低 |
| 资金流向 | 3 分钟 | 需要较新数据 |
| 行业分类 | 1 小时 | 几乎不变 |
| 龙虎榜 | 30 分钟 | 每日更新 |
| 融资融券 | 10 分钟 | 每日更新 |
| 排名因子 | 5 分钟 | 每日计算 |

### 4. 降级策略
**Decision**: Alpha300 不可用时降级到 EastMoney API

**Rationale**: 保证系统可用性，不阻断分析流程

## 数据映射

### KlinePoint 映射
```go
// Alpha300 daily → models.KlinePoint
type KlinePoint struct {
    Date   string  // trade_date (YYYYMMDD → YYYY-MM-DD)
    Open   float64 // open
    Close  float64 // close
    High   float64 // high
    Low    float64 // low
    Volume float64 // vol
    Amount float64 // amount
}
```

### Quote 映射
```go
// Alpha300 daily + daily_basic → models.Quote
type Quote struct {
    Code          string  // ts_code (去掉 .SH/.SZ)
    Name          string  // stock_basic.name
    Price         float64 // daily.close
    ChangePercent float64 // daily.pct_chg
    PE            float64 // daily_basic.pe_ttm
    PB            float64 // daily_basic.pb
    TotalMV       float64 // daily_basic.total_mv
    // ... 其他字段从 daily 表获取
}
```

### MoneyFlowDay 映射
```go
// Alpha300 stock_moneyflow → models.MoneyFlowDay
type MoneyFlowDay struct {
    Date         string  // trade_date
    MainNet      float64 // main_net_amount / 10000 (转为万)
    HugeNet      float64 // super_net_amount / 10000
    BigNet       float64 // large_net_amount / 10000
    MainNetPct   float64 // main_net_pct
}
```

## Files to Change

### New Files
- `internal/services/alpha300_db.go` — Alpha300 数据库访问服务
- `internal/services/alpha300_db_test.go` — 单元测试

### Modified Files
- `internal/config/config.go` — 添加 Alpha300 数据库配置
- `cmd/server/main.go` — 初始化 Alpha300 数据库连接
- `internal/services/analysis.go` — 使用 Alpha300 数据源
- `internal/handlers/reports.go` — 使用 Alpha300 排名数据

## Configuration

```yaml
# .env 新增配置
ALPHA300_DB_HOST=198.23.251.110
ALPHA300_DB_PORT=25432
ALPHA300_DB_NAME=alpha300
ALPHA300_DB_USER=alpha300_app
ALPHA300_DB_PASSWORD=Alpha300Pg@2026!
ALPHA300_DB_SSL_MODE=disable
ALPHA300_DB_MAX_CONNS=5
ALPHA300_DB_ENABLED=true
```

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Alpha300 数据库不可用 | 数据获取失败 | 降级到 EastMoney API |
| 数据格式不兼容 | 解析错误 | 严格的数据映射和验证 |
| 连接池耗尽 | 查询超时 | 限制最大连接数，添加超时 |
| 数据延迟 | 分析不准确 | 设置合理的缓存 TTL |
