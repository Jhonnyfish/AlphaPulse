# AlphaPulse 前端质量提升任务

## 背景
Playwright 测试发现 38/38 页面无崩溃，但以下 14 个页面内容过于稀疏（body text < 600 字符），
视觉丰富度远不及原版 HTML 前端（4235 行 HTML + 2742 行 CSS）。

## 需要修复的页面（按优先级）

### P0 — 高频使用但内容最少
1. **MarketPage** (577字) — 行情页面，应该有：大盘指数卡片、涨跌排行、成交额排行、板块热力图
2. **KlinePage** (577字) — K线页面，应该有：K线图 + 技术指标(MA/MACD/KDJ/RSI)、买卖信号标注
3. **SectorsPage** (577字) — 板块页面，应该有：板块列表表格、板块资金流向柱状图、板块热力图
4. **AnalyzePage** (577字) — 个股分析，应该有：8维度评分雷达图、技术指标、资金流向、信号列表

### P1 — 选股相关
5. **TrendsPage** (575字) — 趋势页面，应该有：趋势指标表格、趋势强度柱状图
6. **BreadthPage** (575字) — 市场广度，应该有：涨跌家数柱状图、新高新低统计
7. **SentimentPage** (575字) — 市场情绪，应该有：情绪指数仪表盘、恐惧贪婪指数
8. **RankingPage** (575字) — 综合排名，应该有：排名表格、评分分布图
9. **PatternScannerPage** (576字) — 形态扫描，应该有：形态列表、K线形态图示

### P2 — 工具类
10. **InstitutionsPage** (575字) — 机构动向，应该有：机构持仓变动表格、资金流向图
11. **AnomalyPage** (575字) — 异常检测，应该有：异常事件列表、异常强度图
12. **MultiTrendPage** (576字) — 多周期趋势，应该有：多周期对比表格、趋势一致性图
13. **CorrelationPage** (576字) — 相关性分析，应该有：相关性热力图、强相关对列表
14. **NewsPage** (598字) — 资讯，应该有：新闻列表、情感标签、关联个股

## 通用修复要求

### 1. 每个页面必须有的元素
- **页面标题** (h1/h2) + 描述文字
- **数据卡片** (至少 2-4 个 KPI 指标卡片)
- **主表格或图表** (至少 1 个 ECharts 图表 + 1 个数据表格)
- **加载态** (skeleton 骨架屏，不是 "加载中..." 文字)
- **空状态** (无数据时显示引导提示)
- **错误状态** (API 失败时显示重试按钮)

### 2. ECharts 图表规范
- 使用 `@/components/charts/EChart` 封装组件
- 配色：深色主题，bg: transparent, text: #94a3b8
- 红涨绿跌：涨 = #ef4444, 跌 = #22c55e
- 图表类型：K线用 lightweight-charts，其他用 ECharts

### 3. 表格规范
- 使用已有的 SortableHeader、Pagination、TableToolbar 组件
- 支持排序、筛选、分页
- 涨跌幅用红绿色标注

### 4. API 错误处理
- 所有 API 调用用 try/catch 包裹
- 失败时显示 ErrorState 组件（有重试按钮）
- 不要让页面白屏或显示空白

### 5. Glass Morphism 设计风格
- 卡片：`glass rounded-2xl p-5` (index.css 已有 .glass 类)
- 背景：`bg-[#0f1117]` 或 `var(--color-bg-primary)`
- 文字：`text-slate-200` 主文字, `text-slate-400` 次要
- 边框：`border border-slate-700/50`

## 参考实现
- **DashboardPage.tsx** — 最丰富的页面，有 8 个图表，参考其结构
- **DailyReportPage.tsx** — 有 13 个表格，参考其表格实现
- **PerfStatsPage.tsx** — 有详细统计图表，参考其图表配置

## 原版参考
- 原版 HTML: `/home/finn/stock-quote-service/templates/dashboard.html` (4235行)
- 原版 CSS: `/home/finn/stock-quote-service/static/dashboard.css` (2742行)
- 原版有 52 个图表、80 个表格、20 个弹窗

## 测试验证
修复后运行 Playwright 测试验证：
```bash
cd /tmp && python3 playwright_test_v3.py
```
目标：所有页面 body_len > 1000，0 个 API 错误（或优雅降级）

## 文件位置
- 页面组件: `/home/finn/alphapulse/frontend/src/pages/`
- 图表组件: `/home/finn/alphapulse/frontend/src/components/charts/EChart.tsx`
- 表格组件: `/home/finn/alphapulse/frontend/src/components/table/`
- API 客户端: `/home/finn/alphapulse/frontend/src/lib/api.ts`
- 设计系统: `/home/finn/alphapulse/frontend/src/index.css`
