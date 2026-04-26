# AlphaPulse 前端改进计划

## 背景
后端 103 个 API 已全部迁移完成（22个模块），但 React 前端与原版 Python 单页应用相比存在严重差距：
- 原版 37 个视图 → React 仅 18 个页面（缺 21 个）
- 原版 900K+ 字符代码 → React 仅 6K 行
- 原版有完整的 glass morphism 设计系统 → React 仅有基础 Tailwind
- 原版有 Canvas 图表、Toast、拖拽、快捷键、主题切换 → React 全部缺失

## 原则
- **后端不再新增 API**，仅保证现有接口可用性
- 前端优先补设计基础设施，再补页面
- 每轮迭代完成 1-2 个可交付单元
- 保持中文 UI（红涨绿跌）
- 后端端口 8899，前端端口 5173

---

## 阶段一：设计基础设施（优先级最高）

### 1.1 安装图表库
- [ ] 安装 `echarts` + `echarts-for-react`
- [ ] 创建通用 `<EChart>` 封装组件，支持主题、loading、resize

### 1.2 Glass Morphism 设计系统
- [ ] 升级 `index.css`：glass morphism 变量、panel glow、网格纹理背景
- [ ] 创建 `.glass` / `.panel-glow` 基础样式类
- [ ] 参照原版 `dashboard.css` 的深色科技风配色
- [ ] 添加 pulse、fade、slide 等 keyframes 动画
- [ ] 响应式媒体查询

### 1.3 Toast 通知系统
- [ ] 创建 `<Toast>` 组件 + `useToast` hook
- [ ] 支持 success/error/warning/info 类型
- [ ] 自动消失 + 手动关闭
- [ ] 右上角堆叠动画

### 1.4 骨架屏加载
- [ ] 创建 `<Skeleton>` 组件（矩形、圆形、文本行）
- [ ] 创建 `<SkeletonCard>` / `<SkeletonTable>` 组合组件
- [ ] 每个页面用 skeleton 替代 "加载中..." 文字

### 1.5 主题切换
- [ ] 创建 `ThemeContext`，支持 dark/light 切换
- [ ] 在 Layout 中加入主题切换按钮
- [ ] localStorage 持久化用户偏好

### 1.6 Command Palette (⌘K)
- [ ] 全局快捷键 Ctrl/Cmd+K 打开搜索面板
- [ ] 搜索股票 + 快速导航到页面
- [ ] 参照原版 `searchModal` 功能

### 1.7 Ticker Tape 滚动条
- [ ] 顶部实时行情滚动条组件
- [ ] 自选股涨跌数据 + 点击跳转

### 1.8 全局快捷键
- [ ] `useKeyboard` hook
- [ ] R=刷新, S=搜索, 数字键=切换页面
- [ ] 帮助面板显示快捷键列表

---

## 阶段二：缺失页面（21个）

### 2.1 高优先（核心分析功能）
- [ ] **AnalyzePage** — 个股深度分析（8维度雷达图 + 评分详情）
  - API: GET /analyze?code=xxx
  - 包含：order_flow, volume_price, valuation, volatility, money_flow, technical, sector, sentiment
- [ ] **TrendsPage** — 市场趋势分析
  - API: GET /trends
- [ ] **BreadthPage** — 市场广度（涨跌家数、新高新低）
  - API: GET /breadth
- [ ] **RankingPage** — 综合排名
  - API: GET /ranking
- [ ] **FlowPanelPage** — 资金流向面板（主力/散户/北向）
  - API: GET /flow
- [ ] **SentimentPage** — 市场情绪分析
  - API: GET /sentiment

### 2.2 中优先（报告与进阶）
- [ ] **DailyBriefPage** — 每日简报
  - API: GET /daily-brief
- [ ] **DailyReportPage** — 每日报告
  - API: GET /daily-report/latest, GET /daily-report/list
- [ ] **PerfStatsPage** — 绩效统计
  - API: GET /performance-stats
- [ ] **BacktestPage** — 策略回测
  - API: GET /backtest
- [ ] **StrategyEvalPage** — 策略评估
  - API: GET /strategies（复用策略API）
- [ ] **TradeCalendarPage** — 交易日历
  - API: GET /calendar
- [ ] **MultiTrendPage** — 多周期趋势
  - API: GET /multi-trend

### 2.3 低优先（辅助功能）
- [ ] **CorrelationPage** — 相关性分析
  - API: GET /correlation
- [ ] **AnomalyPage** — 异常检测
  - API: GET /anomalies
- [ ] **PatternScannerPage** — 形态扫描器
  - API: GET /pattern-scanner（待确认后端路由）
- [ ] **InstitutionPage** — 机构动向
  - API: GET /institution-tracker
- [ ] **InvestmentPlansPage** — 投资计划
  - API: GET /investment-plans
- [ ] **PortfolioRiskPage** — 组合风险分析
  - API: GET /risk
- [ ] **QuickActionsPage** — 快捷操作面板
- [ ] **DiagPage** — 系统诊断
  - API: GET /system-status, GET /slow-queries

---

## 阶段三：现有页面提升

### 3.1 DashboardPage 提升
- [ ] 添加市场指数 ECharts 走势图
- [ ] 涨跌家数柱状图
- [ ] 热力图板块展示
- [ ] 信号摘要卡片 + 动画
- [ ] 最近活动时间线

### 3.2 WatchlistPage 提升
- [ ] 添加迷你走势图（Sparkline）
- [ ] 拖拽排序支持
- [ ] 涨跌颜色渐变
- [ ] 批量操作

### 3.3 CandidatesPage 提升
- [ ] 分层展示（核心/关注/观察）
- [ ] 雷达图展示各维度得分
- [ ] 点击跳转到 AnalyzePage

### 3.4 PortfolioPage 提升
- [ ] 持仓饼图 + 行业分布
- [ ] 收益曲线图
- [ ] 风险指标卡片

### 3.5 TradingJournalPage 提升
- [ ] 交易日历热力图
- [ ] 收益分布直方图
- [ ] 月度/年度统计图表

### 3.6 其他页面
- [ ] 所有表格添加排序、筛选、分页
- [ ] 所有列表添加 skeleton 加载态
- [ ] 所有页面添加空状态提示
- [ ] 错误状态统一处理 + 重试按钮

---

## 每轮迭代流程

1. **检查服务状态** — 后端 8899、前端 5173 是否正常
2. **选择任务** — 从计划中选取下一个未完成项
3. **实现** — 代码编写（优先 Claude Code，复杂用 Codex）
4. **验证** — TypeScript 编译通过 + 页面可访问
5. **提交** — git commit + 推送
6. **报告** — 汇报本轮完成内容

## 技术约束
- React 19 + TypeScript + Vite + Tailwind CSS 4
- 图表：ECharts (echarts-for-react)
- 图标：lucide-react
- 路由：react-router-dom v7
- HTTP：axios
- K线：lightweight-charts（已有）
- 端口：后端 8899，前端 5173
