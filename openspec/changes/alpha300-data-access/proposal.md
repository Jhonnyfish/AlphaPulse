## Why

AlphaPulse 当前依赖 EastMoney API 获取日K线、估值、资金流向等数据，存在以下问题：
1. API 限流 — 频繁请求会被限速，影响分析准确性
2. 数据不完整 — 部分字段返回空值或默认值
3. 延迟高 — 每次分析都需要实时调用 API

Alpha300 数据库包含完整的日K线、估值、资金流向、行业分类、龙虎榜等数据，且为只读访问，可以作为 AlphaPulse 的主要数据源。

## What Changes

### 新增能力
- 创建 `internal/services/alpha300_db.go` — Alpha300 数据库只读访问层
- 提供日K线、估值、资金流向、行业分类、龙虎榜等数据查询接口
- 支持批量查询和缓存

### 改进能力
- 八维分析引擎使用 Alpha300 数据源
- 每日报告使用 Alpha300 排名数据
- 综合排名使用 Alpha300 因子数据

## Capabilities

- [alpha300-data-access](specs/alpha300-data-access/spec.md) — Alpha300 数据库只读访问层
