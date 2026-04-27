# P0 页面修复 — MarketPage, KlinePage, SectorsPage, AnalyzePage

这 4 个页面是最高频使用的，但目前内容最少（body=577）。
每个页面需要大幅增强。以下是每个页面的详细要求：

## 1. MarketPage.tsx — 行情页面

当前状态：只有基础表格，缺少概览指标。

需要添加：
- **顶部 KPI 卡片区**（4 个卡片）：
  - 上证指数 当前值 + 涨跌幅
  - 深证成指 当前值 + 涨跌幅
  - 创业板指 当前值 + 涨跌幅
  - 今日涨跌比 上涨家数/下跌家数
- **涨跌排行**：涨幅前 10 + 跌幅前 10（两个小表格或 tab 切换）
- **成交额排行**：成交额前 10
- 使用 `marketApi.getOverview()` 获取大盘数据
- 使用 `marketApi.getTopMovers()` 获取涨跌排行

## 2. KlinePage.tsx — K线页面

当前状态：有 K 线图但 API 500 错误时无内容。

需要添加：
- **股票搜索/选择器**（已有 StockSearch 组件）
- **K 线图**（已用 lightweight-charts）
- **技术指标面板**：MA5/MA10/MA20/MA60 均线
- **信息卡片**：当前价、涨跌幅、成交量、成交额
- **买卖区间卡片**：买入区间、止损位、目标价
- API 错误时显示 ErrorState + 重试按钮

## 3. SectorsPage.tsx — 板块页面

当前状态：只有基础列表，缺少可视化。

需要添加：
- **板块热力图**（ECharts treemap，参考 DashboardPage 的实现）
- **板块资金流向柱状图**（ECharts bar chart，按净流入排序）
- **板块涨跌排行表格**（板块名、涨跌幅、成交额、领涨股）
- 使用 `sectorsApi.getOverview()` 或 `sectorsApi.getList()`

## 4. AnalyzePage.tsx — 个股分析页面

当前状态：有基础框架但内容少。

需要添加：
- **股票搜索/选择器**
- **8 维度评分雷达图**（动量、趋势、资金、技术、估值、情绪、量价、形态）
- **技术指标卡片组**（MA/MACD/RSI/KDJ 状态）
- **资金流向**（主力净流入、散户净流入）
- **信号列表**（最近的买卖信号）
- **评分历史折线图**
- API 错误时显示 ErrorState + 重试按钮

## 通用要求
- 使用 `@/components/charts/EChart` 封装组件
- 使用 `@/components/ui/Skeleton` 骨架屏
- 使用 `@/components/ErrorState` 错误状态
- 使用 `@/components/EmptyState` 空状态
- Glass morphism 风格：`glass rounded-2xl p-5`
- 红涨绿跌配色
- 每个页面至少 2 个 ECharts 图表 + 1 个数据表格

## 参考文件
- `/home/finn/alphapulse/frontend/src/pages/DashboardPage.tsx` — 最丰富的页面
- `/home/finn/alphapulse/frontend/src/components/charts/EChart.tsx` — 图表组件
- `/home/finn/alphapulse/frontend/src/lib/api.ts` — API 客户端
