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

### 阶段三：现有页面提升 (0/18)
- [ ] 3.1 DashboardPage — ECharts 指数走势图、涨跌家数柱状图、板块热力图、信号摘要卡片、活动时间线
- [ ] 3.2 WatchlistPage — 迷你走势图(Sparkline)、拖拽排序、涨跌颜色渐变、批量操作
- [ ] 3.3 CandidatesPage — 分层展示(核心/关注/观察)、雷达图、点击弹窗分析详情
- [ ] 3.4 PortfolioPage — 持仓饼图、收益曲线、风险指标卡片
- [ ] 3.5 TradingJournalPage — 交易日历热力图、收益分布直方图、月度统计
- [ ] 3.6 所有表格 — 排序、筛选、分页
- [ ] 3.7 所有列表 — skeleton 加载态
- [ ] 3.8 所有页面 — 空状态提示
- [ ] 3.9 所有页面 — 错误状态统一处理 + 重试按钮
- [ ] 3.10 KlinePage — 技术指标叠加、画线工具
- [ ] 3.11 SectorsPage — 板块资金流向图、板块内个股排名
- [ ] 3.12 HotConceptsPage — 概念热度趋势图、概念内个股联动
- [ ] 3.13 DragonTigerPage — 龙虎榜资金分布图、游资跟踪
- [ ] 3.14 NewsPage — 新闻情感标签、关联个股标记
- [ ] 3.15 SignalsPage — 信号统计图表、信号强度趋势
- [ ] 3.16 ScreenerPage — 选股条件可视化、结果对比图
- [ ] 3.17 ComparePage — 多维对比雷达图、叠加走势对比
- [ ] 3.18 全局 — 移除 react-router-dom，改为视图切换架构
