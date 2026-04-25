# AlphaPulse Go 完整迁移计划

## 目标
将原版 Python server.py 的 103 个 API 路由完整迁移到 Go 后端 + React 前端

## 原版功能清单（按模块分类）

### 模块 1: 核心行情 (已完成 ✅)
- [x] GET /quote — 实时行情
- [x] GET /api/klines — K线数据
- [x] GET /sectors — 板块列表
- [x] GET /api/sectors-overview — 板块概览
- [x] GET /news — 新闻资讯
- [x] GET /api/news-feed — 新闻源

### 模块 2: 自选股 (已完成 ✅)
- [x] GET /api/watchlist — 自选股列表
- [x] POST /api/watchlist/add — 添加自选股
- [x] POST /api/watchlist/batch-add — 批量添加
- [x] POST /api/watchlist/remove — 删除自选股
- [x] POST /api/watchlist/sync — 同步自选股

### 模块 3: 仪表盘 (已完成 ✅)
- [x] GET /api/dashboard-summary — 仪表盘摘要
- [x] GET /api/market-overview — 市场概览（大盘指数）
- [x] GET /api/market-session — 市场状态（开盘/收盘/午休）
- [x] GET /api/market-trends — 市场趋势
- [x] GET /api/market-breadth — 市场宽度（涨跌家数）
- [x] GET /api/market-sentiment — 市场情绪

### 模块 4: 选股分析 (已完成 ✅)
- [x] GET /api/candidates — AI 候选股
- [x] GET /api/screener — 选股器
- [x] GET /analyze — 8维度综合分析详情
- [x] GET /api/score-history/{code} — 评分历史
- [x] GET /api/pattern-scanner — 形态扫描
- [x] GET /api/multi-trend — 多周期趋势
- [x] GET /api/correlation — 相关性分析

### 模块 5: 股票对比 (已完成 ✅)
- [x] GET /compare — 对比页面
- [x] GET /api/backtest-compare — 回测对比
- [x] GET /sector-compare — 板块对比

### 模块 6: 投资组合 (已完成 ✅)
- [x] GET /api/portfolio — 持仓列表
- [x] POST /api/portfolio — 添加持仓
- [x] PUT /api/portfolio/{id} — 更新持仓
- [x] DELETE /api/portfolio/{id} — 删除持仓
- [x] GET /api/portfolio/analytics — 组合分析
- [x] GET /api/portfolio-risk — 风险分析

### 模块 7: 交易日志 (已完成 ✅)
- [x] GET /api/trading-journal — 交易记录
- [x] POST /api/trading-journal — 添加交易
- [x] DELETE /api/trading-journal/{id} — 删除交易
- [x] GET /api/trading-journal/stats — 交易统计
- [x] GET /api/trading-journal/calendar — 交易日历
- [x] GET /api/trade-strategy-eval — 策略评估

### 模块 8: 自定义策略 (已完成 ✅)
- [x] GET /api/strategies — 策略列表
- [x] POST /api/strategies — 创建策略
- [x] PUT /api/strategies/{id} — 更新策略
- [x] DELETE /api/strategies/{id} — 删除策略
- [x] POST /api/strategies/{id}/activate — 激活策略
- [x] POST /api/strategies/{id}/deactivate — 停用策略
- [ ] GET /api/candidates?strategy_id=xxx — 策略选股

### 模块 9: 自定义预警 (已完成 ✅)
- [x] GET /api/custom-alerts — 预警列表
- [x] POST /api/custom-alerts — 创建预警
- [x] DELETE /api/custom-alerts/{id} — 删除预警
- [x] GET /api/custom-alerts/check — 检查预警

### 模块 10: 股票笔记 (已完成 ✅)
- [x] GET /api/stock-notes/{code} — 获取笔记
- [x] POST /api/stock-notes — 添加笔记
- [x] PUT /api/stock-notes/{id} — 更新笔记
- [x] DELETE /api/stock-notes/{id} — 删除笔记
- [x] GET /api/stock-notes/tags/all — 所有标签

### 模块 11: 龙虎榜/机构 (已完成 ✅)
- [x] GET /api/dragon-tiger — 龙虎榜
- [x] GET /api/dragon-tiger-history — 龙虎榜历史
- [x] GET /api/institution-tracker — 机构追踪

### 模块 12: 热门概念 (已完成 ✅)
- [x] GET /api/hot-concepts — 热门概念
- [x] GET /api/hot-concepts/{code}/stocks — 概念成分股
- [x] GET /api/watchlist-concept-overlap — 自选股概念重叠

### 模块 13: 资金流向 (已完成 ✅)
- [x] GET /flow — 资金流向
- [x] GET /stockinfo — 个股详情（含资金流向）

### 模块 14: 公告数据 (已完成 ✅)
- [x] GET /announcements — 公告列表

### 模块 15: 板块轮动 (已完成 ✅)
- [x] GET /api/sector-rotation — 板块轮动
- [x] GET /api/sector-rotation-history — 轮动历史

### 模块 16: 报告系统 (已完成 ✅)
- [x] GET /reports — 报告列表页面(重定向到API)
- [x] GET /api/reports — 报告列表 API
- [x] GET /api/reports/:filename — 查看报告
- [x] POST /api/daily-report/generate — 生成日报
- [x] GET /api/daily-report/latest — 最新日报
- [x] GET /api/daily-report/list — 日报列表
- [x] GET /api/daily-brief — 每日简报

### 模块 17: 投资计划 (已完成 ✅)
- [x] GET /api/investment-plans — 计划列表
- [x] POST /api/investment-plans — 创建计划
- [x] DELETE /api/investment-plans/{code} — 删除计划

### 模块 18: 信号系统 (已完成 ✅)
- [x] GET /api/signal-calendar — 信号日历
- [x] GET /api/signal-history — 信号历史
- [x] GET /api/anomalies — 异常检测
- [x] GET /api/alerts — 系统告警
- [x] GET /api/activity-log — 活动日志

### 模块 19: 自选股分析 (已完成 ✅)
- [x] GET /api/watchlist-heatmap — 热力图数据
- [x] GET /api/watchlist-sectors — 板块分布
- [x] GET /api/watchlist-ranking — 排名
- [x] GET /api/watchlist-groups — 分组
- [x] POST /api/watchlist-groups — 创建分组
- [x] PUT /api/watchlist-groups/{id} — 更新分组
- [x] DELETE /api/watchlist-groups/{id} — 删除分组
- [x] POST /api/watchlist-groups/assign — 分配分组

### 模块 20: 回测系统 (已完成 ✅)
- [x] GET /backtest — 回测页面
- [x] GET /api/backtest-compare — 回测对比

### 模块 21: 系统管理 (部分完成 ✅)
- [x] GET /health — 健康检查
- [x] GET /api/info — 系统信息
- [x] GET /api/system-status — 系统状态
- [x] GET /api/status — 服务状态
- [ ] GET /api/docs — API 文档
- [x] GET /api/slow-queries — 慢查询
- [x] GET /api/performance-stats — 性能统计
- [x] POST /api/cache/clear — 清除缓存
- [x] GET /api/activity-log — 活动日志

### 模块 22: 设置页面 (待迁移)
- [ ] GET /settings — 设置页面

---

## 迁移优先级

### P0 — 立即迁移（核心功能）
1. 模块 3: 仪表盘（market-overview, market-session, market-trends）
2. 模块 4: 选股分析（candidates, screener, analyze）
3. 模块 13: 资金流向（flow）
4. 模块 14: 公告数据（announcements）

### P1 — 尽快迁移（高频功能）
5. 模块 6: 投资组合
6. 模块 7: 交易日志
7. 模块 11: 龙虎榜/机构
8. 模块 12: 热门概念
9. 模块 15: 板块轮动
10. ~~模块 19: 自选股分析~~ ✅

### P2 — 后续迁移（进阶功能）
11. 模块 5: 股票对比
12. 模块 8: 自定义策略
13. 模块 9: 自定义预警
14. 模块 10: 股票笔记
15. 模块 16: 报告系统
16. 模块 17: 投资计划
17. 模块 18: 信号系统
18. 模块 20: 回测系统

---

## 每轮迭代规则

1. **每轮迁移 1-2 个 API 路由**
2. **后端迁移后必须**:
   - `go build ./...` 编译通过
   - `go test ./... -count=1 -short` 测试通过
   - 重启后端服务
3. **前端页面同步更新**（如果需要）
4. **重启前端服务**
5. **验证 API 可用**

## 服务端口
- Go 后端: 8899
- React 前端: 5173
- PostgreSQL: 5432
