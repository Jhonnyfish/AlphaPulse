# AlphaPulse 前端迭代状态

> Cron 每轮只读这一个文件即可，不要重复分析项目结构。

## 项目结构
```
/home/finn/alphapulse/
├── cmd/server/main.go          # Go 后端入口，端口 8899
├── internal/handlers/           # 27 个 handler 文件
├── internal/services/           # 业务逻辑
├── frontend/
│   ├── src/
│   │   ├── App.tsx              # 路由入口（需改为视图切换）
│   │   ├── main.tsx
│   │   ├── index.css            # 全局样式 + glass morphism 设计系统
│   │   ├── lib/
│   │   │   ├── api.ts           # 所有 API 客户端（675行，接口齐全）
│   │   │   └── auth.tsx         # 认证
│   │   ├── components/
│   │   │   ├── Layout.tsx       # 侧边栏布局（361行，38个导航项）
│   │   │   ├── StockSearch.tsx
│   │   │   └── charts/
│   │   │       └── EChart.tsx   # ECharts 封装组件
│   │   └── pages/               # 18 个页面组件
│   │       ├── DashboardPage.tsx
│   │       ├── WatchlistPage.tsx
│   │       ├── MarketPage.tsx
│   │       ├── KlinePage.tsx
│   │       ├── CandidatesPage.tsx
│   │       ├── ScreenerPage.tsx
│   │       ├── ComparePage.tsx
│   │       ├── PortfolioPage.tsx
│   │       ├── TradingJournalPage.tsx
│   │       ├── SectorsPage.tsx
│   │       ├── HotConceptsPage.tsx
│   │       ├── DragonTigerPage.tsx
│   │       ├── NewsPage.tsx
│   │       ├── SignalsPage.tsx
│   │       ├── StrategiesPage.tsx
│   │       ├── WatchlistAnalysisPage.tsx
│   │       ├── SettingsPage.tsx
│   │       └── LoginPage.tsx
│   ├── package.json
│   └── vite.config.ts
├── FRONTEND_IMPROVEMENT_PLAN.md  # 改进计划（标记 [x] 已完成 / [ ] 未完成）
└── FRONTEND_STATE.md             # 本文件
```

## 技术栈
- React 19 + TypeScript + Vite 8 + Tailwind CSS 4
- echarts + echarts-for-react（已安装）
- lightweight-charts（K线图，已安装）
- lucide-react 图标
- axios（@/lib/api）
- 不使用 react-router-dom 做页面路由（视图切换用 state 管理）

## 设计规范
- 配色：bg-primary #0f1117, bg-secondary #1a1d27, bg-card #222536, accent #3b82f6
- Glass morphism：backdrop-filter: blur(18px), 半透明面板
- 深色科技风：径向渐变背景 + 网格纹理
- 中文 UI，红涨绿跌
- 原版 CSS 参考：/home/finn/stock-quote-service/static/dashboard.css
- 原版 HTML 参考：/home/finn/stock-quote-service/templates/dashboard.html

## 核心架构要求
- **单页面应用**——所有视图在同一页面上，不做二级页面跳转
- 用 useState 控制当前 activeView，CSS 控制显示/隐藏
- 侧边栏点击切换 view state，不跳转路由
- 详情用 Modal/Drawer 弹窗展示

## 后端 API
- 端口：8899
- 健康检查：GET /health
- 所有 API 在 api.ts 中已有 TypeScript 接口定义
- **后端不再新增 API**，仅保证现有接口可用

## 当前进度
> 每轮 Codex 完成任务后，由 Hermes 更新此区域。
> 任务粒度：每个 [ ] 应控制在 Codex 1-2 分钟内可完成的范围。

### 阶段一：设计基础设施 (8/8) ✅ 全部完成
- [x] ECharts 安装 + EChart 封装组件
- [x] Glass Morphism 设计系统（index.css）
- [x] Toast 通知系统
- [x] 骨架屏加载组件
- [x] 主题切换（dark/light）
- [x] Command Palette（⌘K）
- [x] Ticker Tape 滚动条
- [x] 全局快捷键

### 阶段二：缺失页面 (21/21) ✅ 全部完成
- [x] AnalyzePage, TrendsPage, BreadthPage, RankingPage
- [x] FlowPanelPage, SentimentPage, DailyBriefPage, DailyReportPage
- [x] PerfStatsPage, BacktestPage, StrategyEvalPage, TradeCalendarPage
- [x] MultiTrendPage, CorrelationPage, AnomalyPage, PatternScannerPage
- [x] InstitutionPage, InvestmentPlansPage, PortfolioRiskPage
- [x] QuickActionsPage, DiagPage

### 阶段三：现有页面提升 (18/24)
- [x] 3.1 DashboardPage — ECharts 指数走势图、涨跌家数柱状图、板块热力图、信号摘要卡片、活动时间线
- [x] 3.1b DashboardPage — 涨跌家数柱状图 + 市场广度指标
- [x] 3.1c DashboardPage — 板块热力图（ECharts treemap）
- [x] 3.1d DashboardPage — 信号摘要卡片（活跃信号数、最近告警）+ 动画
- [x] 3.1e DashboardPage — 最近活动时间线
- [x] 3.2a WatchlistPage — 每行添加迷你 Sparkline 走势图
- [x] 3.2b WatchlistPage — 拖拽排序支持
- [x] 3.3a CandidatesPage — 分层展示（核心/关注/观察三个区域）
- [x] 3.3b CandidatesPage — 点击个股弹出分析详情 Modal（不跳转）
- [x] 3.4a PortfolioPage — 持仓行业分布饼图
- [x] 3.4b PortfolioPage — 收益曲线图 + 风险指标卡片
- [x] 3.5a TradingJournalPage — 交易日历热力图
- [x] 3.5b TradingJournalPage — 收益分布直方图 + 月度统计
- [x] 3.6 所有数据表格 — 添加排序、筛选、分页功能（MarketPage 已实现，可复用组件已抽取）
- [x] 3.7 所有列表页面 — skeleton 加载态替代"加载中..."文字
- [x] 3.8 所有页面 — 空状态提示（无数据时显示引导）
- [x] 3.9 所有页面 — 错误状态统一处理 + 重试按钮
- [x] 3.10 KlinePage — 技术指标叠加（MA/MACD/KDJ/RSI）
- [x] 3.11 SectorsPage — 板块资金流向柱状图
- [x] 3.12 HotConceptsPage — 概念热度趋势折线图
- [x] 3.13 DragonTigerPage — 龙虎榜资金分布饼图
- [x] 3.14 NewsPage — 新闻情感标签 + 关联个股标记
- [x] 3.15 SignalsPage — 信号统计图表 + 信号强度趋势
- [x] 3.16 ScreenerPage — 选股条件可视化展示
- [x] 3.17 ComparePage — 多维对比雷达图
- [x] 3.18 全局 — 移除 react-router-dom，改为视图切换架构（ViewContext + useState）

## 已安装依赖
> Hermes 在委托 Codex 前检查此列表，缺的先装好再委托。
- echarts ^6.0.0
- echarts-for-react ^3.0.6
- lightweight-charts ^5.2.0
- lucide-react ^1.11.0
- axios ^1.15.2
- ~~react-router-dom~~ 已移除
- class-variance-authority, clsx, tailwind-merge
- @dnd-kit/core, @dnd-kit/sortable, @dnd-kit/utilities

## 变更日志
- 2026-04-26 22:40 | 3.18 移除 react-router-dom | 新建ViewContext.tsx(ViewName类型+useView hook)，App.tsx改为useState视图切换，Layout.tsx NavLink改为button onClick，12个页面useNavigate/useSearchParams/Link替换为useView+button，npm uninstall react-router-dom，tsc 0 errors | files: ViewContext.tsx, App.tsx, Layout.tsx, main.tsx, CommandPalette.tsx, TickerTape.tsx, AnalyzePage.tsx, KlinePage.tsx, WatchlistPage.tsx, QuickActionsPage.tsx, NewsPage.tsx, PatternScannerPage.tsx, RankingPage.tsx, PortfolioRiskPage.tsx, StrategyEvalPage.tsx, TradeCalendarPage.tsx, WatchlistAnalysisPage.tsx
> 每轮完成后追加一行。
- 2026-04-26 16:00 | 3.2a WatchlistPage | 每行添加迷你 Sparkline 走势图，使用 ECharts 折线图，红涨绿跌，100x30px，支持桌面表格和移动卡片视图 | files: WatchlistPage.tsx
- 2026-04-26 16:20 | 3.2b WatchlistPage | 使用 @dnd-kit 实现拖拽排序，GripVertical 手柄，拖拽半透明+蓝色插入线，桌面表格和移动卡片双视图支持，arrayMove 本地排序 | files: WatchlistPage.tsx, package.json
- 2026-04-26 16:40 | 3.3a CandidatesPage | 三层分级展示（核心≥80分/关注60-79分/观察<60分），score分层，glass morphism卡片，彩色边框区分层级，折叠展开，每个股票显示代码/名称/价格/动量/趋势/买入区间/止损/行业/信号 | files: CandidatesPage.tsx
- 2026-04-26 17:00 | 3.3b CandidatesPage | 新建 StockDetailModal 分析详情 Modal，点击股票卡片弹出，包含 K 线走势图(EChart)、技术指标网格(MA/MACD/RSI/KDJ)、信号列表、评分进度条、买卖区间、行业标签，glass morphism 风格，支持 ESC/遮罩关闭，body 可滚动 | files: StockDetailModal.tsx, CandidatesPage.tsx
- 2026-04-26 17:20 | 3.4a PortfolioPage | 添加持仓行业分布环形饼图(ECharts)，12色配色，glass morphism卡片，响应式布局(桌面并排/移动端堆叠)，使用现有analytics.sector_allocation数据，tooltip显示行业/市值/占比 | files: PortfolioPage.tsx
- 2026-04-26 17:40 | 3.4b PortfolioPage | 添加收益曲线面积图(ECharts)正负渐变色+零线标记，6项风险指标卡片(最大回撤/夏普比率/年化收益/波动率/胜率/盈亏比)条件着色，30天mock数据，响应式布局(lg:grid-cols-3)，glass morphism风格 | files: PortfolioPage.tsx
- 2026-04-26 18:00 | 3.5a TradingJournalPage | 添加12个月交易日历热力图(ECharts calendar heatmap)，红涨绿跌配色，mock数据(62%工作日有交易)，tooltip显示日期/交易次数/总盈亏/平均收益率，自定义图例，glass morphism卡片，响应式全宽布局 | files: TradingJournalPage.tsx
- 2026-04-26 18:20 | 3.5b TradingJournalPage | 添加收益分布直方图(ECharts bar, 12区间红绿渐变) + 月度统计卡片(12个月响应式grid, 交易次数/总盈亏/胜率/平均收益率)，同源mock数据，glass morphism风格 | files: TradingJournalPage.tsx
- 2026-04-26 18:40 | 3.6 MarketPage 数据表格 | 列排序(升序/降序/无排序+箭头图标)、搜索筛选(代码/名称模糊搜索)、涨跌幅快筛(全部/涨/跌/涨停/跌停)、分页(20条/页+智能页码)；抽取4个可复用组件：useTableSort hook、SortableHeader、TableToolbar、Pagination | files: MarketPage.tsx, hooks/useTableSort.ts, components/table/{SortableHeader,TableToolbar,Pagination,index}.ts
- 2026-04-26 19:00 | 3.7 skeleton 加载态 | 增强Skeleton组件(新增SkeletonGridCard/SkeletonList/SkeletonInlineTable/SkeletonCalendar/SkeletonStatCards)；18个页面替换"加载中..."文字为对应骨架屏 | files: Skeleton.tsx, MarketPage, WatchlistPage, NewsPage, SignalsPage, SectorsPage, DragonTigerPage, HotConceptsPage, StrategiesPage, PortfolioPage, TradingJournalPage, TradeCalendarPage, StrategyEvalPage, ScreenerPage, BacktestPage, PortfolioRiskPage, SettingsPage
- 2026-04-26 19:20 | 3.8 空状态提示 | 创建 EmptyState 可复用组件(glass morphism, 48px图标, 标题+描述+可选按钮, fadeIn动画)；12个页面添加无数据空状态引导(MarketPage/WatchlistPage/CandidatesPage/PortfolioPage/TradingJournalPage/NewsPage/SignalsPage/SectorsPage/HotConceptsPage/DragonTigerPage/StrategiesPage/ScreenerPage) | files: EmptyState.tsx, MarketPage, WatchlistPage, CandidatesPage, PortfolioPage, TradingJournalPage, NewsPage, SignalsPage, SectorsPage, HotConceptsPage, DragonTigerPage, StrategiesPage, ScreenerPage
- 2026-04-26 19:40 | 3.9 错误状态统一处理 | 新建 ErrorState 可复用组件(红色AlertTriangle图标, glass morphism, 重试按钮)；17个页面统一错误处理：替换内联错误div为ErrorState组件, 添加onRetry回调, catch中设置error state | files: ErrorState.tsx, DashboardPage, MarketPage, WatchlistPage, PortfolioPage, CandidatesPage, KlinePage, ScreenerPage, ComparePage, TradingJournalPage, SectorsPage, HotConceptsPage, DragonTigerPage, NewsPage, SignalsPage, StrategiesPage, WatchlistAnalysisPage, SettingsPage
- 2026-04-26 20:00 | 3.10 KlinePage 技术指标叠加 | 新建indicators.ts(MA/MACD/KDJ/RSI计算函数)；KlinePage添加指标切换按钮组：MA叠加在主图(4条均线)、MACD/KDJ/RSI副图(150px ECharts)，glass morphism按钮+accent蓝色高亮，副图互斥(单选)，MA独立切换 | files: lib/indicators.ts, KlinePage.tsx
- 2026-04-26 20:21 | 3.11 SectorsPage 板块资金流向柱状图 | 添加ECharts水平条形图展示Top15板块净流入/流出排名，红涨绿跌配色(红#ef4444正流入/绿#22c55e负流出)，20个mock板块数据，glass morphism卡片，400px高度自适应，tooltip显示板块+金额(亿元)，BarChart3图标标题 | files: SectorsPage.tsx
- 2026-04-26 20:40 | 3.12 HotConceptsPage 概念热度趋势折线图 | 添加ECharts折线图展示Top6概念近10日热度趋势，smooth曲线+渐变填充+圆形标记，8色配色区分概念，mock数据(随机游走+动量偏移)，glass morphism卡片(350px)，tooltip展示各概念热度值，TrendingUp图标标题 | files: HotConceptsPage.tsx
- 2026-04-26 21:00 | 3.13 DragonTigerPage 龙虎榜资金分布饼图 | 添加ECharts环形饼图展示6类资金来源分布(机构专用45.2%/知名游资23.8%/沪深股通12.5%/量化基金8.3%/营业部席位6.7%/其他3.5%)，6色配色，glass morphism卡片，PieChart图标标题，300px高度，tooltip显示金额+占比，mock数据 | files: DragonTigerPage.tsx
- 2026-04-26 21:20 | 3.14 NewsPage | 新闻情感标签+关联个股 | 添加情感badge(利好绿/利空红/中性灰+图标)、关键词情感推断(44个关键词)、关联个股pill badge(代码提取+行业匹配,最多3个,点击导航analyze)、glass morphism风格 | files: NewsPage.tsx
- 2026-04-26 21:40 | 3.15 SignalsPage | 信号统计饼图+信号强度趋势折线图 | 添加ECharts环形饼图(4类信号:买入绿/卖出红/持有蓝/观望灰,中心总数) + 30天信号强度均值趋势折线图(渐变面积填充, min/max/trend摘要)，响应式并排布局, glass morphism卡片, 保留现有信号列表功能不变 | files: SignalsPage.tsx
- 2026-04-26 22:00 | 3.16 ScreenerPage | 选股条件可视化 | 添加筛选条件彩色标签(蓝底+图标,点击移除)、行业分布环形饼图(ECharts Top10+其他)、评分分布柱状图(ECharts 7区间红→蓝渐变)、glass morphism面板、替换react-router-dom Link为button、去除SkeletonInlineTable引用改用animate-pulse骨架屏 | files: ScreenerPage.tsx
- 2026-04-26 22:20 | 3.17 ComparePage | 多维对比雷达图 | 回测对比结果下方新增ECharts六维雷达图(胜率/平均收益/信号频率/风控能力/稳定性/综合评分)，5色配色区分多只股票，polygon形状+半透明面积填充，从backtest数据派生维度评分，glass morphism卡片，≥2只股票时显示 | files: ComparePage.tsx
