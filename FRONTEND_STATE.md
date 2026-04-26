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

### 阶段三：现有页面提升 (0/18)
- [ ] 3.1a DashboardPage — ECharts 大盘指数走势图（上证/深证/创业板）
- [ ] 3.1b DashboardPage — 涨跌家数柱状图 + 市场广度指标
- [ ] 3.1c DashboardPage — 板块热力图（ECharts treemap）
- [ ] 3.1d DashboardPage — 信号摘要卡片（活跃信号数、最近告警）+ 动画
- [ ] 3.1e DashboardPage — 最近活动时间线
- [ ] 3.2a WatchlistPage — 每行添加迷你 Sparkline 走势图
- [ ] 3.2b WatchlistPage — 拖拽排序支持
- [ ] 3.3a CandidatesPage — 分层展示（核心/关注/观察三个区域）
- [ ] 3.3b CandidatesPage — 点击个股弹出分析详情 Modal（不跳转）
- [ ] 3.4a PortfolioPage — 持仓行业分布饼图
- [ ] 3.4b PortfolioPage — 收益曲线图 + 风险指标卡片
- [ ] 3.5a TradingJournalPage — 交易日历热力图
- [ ] 3.5b TradingJournalPage — 收益分布直方图 + 月度统计
- [ ] 3.6 所有数据表格 — 添加排序、筛选、分页功能
- [ ] 3.7 所有列表页面 — skeleton 加载态替代"加载中..."文字
- [ ] 3.8 所有页面 — 空状态提示（无数据时显示引导）
- [ ] 3.9 所有页面 — 错误状态统一处理 + 重试按钮
- [ ] 3.10 KlinePage — 技术指标叠加（MA/MACD/KDJ/RSI）
- [ ] 3.11 SectorsPage — 板块资金流向柱状图
- [ ] 3.12 HotConceptsPage — 概念热度趋势折线图
- [ ] 3.13 DragonTigerPage — 龙虎榜资金分布饼图
- [ ] 3.14 NewsPage — 新闻情感标签 + 关联个股标记
- [ ] 3.15 SignalsPage — 信号统计图表 + 信号强度趋势
- [ ] 3.16 ScreenerPage — 选股条件可视化展示
- [ ] 3.17 ComparePage — 多维对比雷达图
- [ ] 3.18 全局 — 移除 react-router-dom，改为视图切换架构

## 已安装依赖
> Hermes 在委托 Codex 前检查此列表，缺的先装好再委托。
- echarts ^6.0.0
- echarts-for-react ^3.0.6
- lightweight-charts ^5.2.0
- lucide-react ^1.11.0
- axios ^1.15.2
- react-router-dom ^7.14.2（待移除）
- class-variance-authority, clsx, tailwind-merge

## 变更日志
> 每轮完成后追加一行。
- （暂无记录）
