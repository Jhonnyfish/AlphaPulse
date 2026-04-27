# AlphaPulse Frontend Test Report
**Date:** 2026-04-27 15:01
**Total pages:** 38
**Results:** ✅ 31 OK | ⚠️ 5 Warnings | ❌ 2 Failed

## ❌ Failed Pages (need immediate fix)

### 持仓 (`portfolio`)
- **Status:** ❌ CRASH
- **Detail:** ErrorBoundary triggered — 页面加载出错
- **Console errors:**
  - `%o

%s

%s
 TypeError: Cannot read properties of undefined (reading 'length')
    at PortfolioPage (http://localhost:5173/src/pages/PortfolioPage.tsx:1350:31)
    at Object.react_stack_bottom_frame (h`
  - `[ErrorBoundary] TypeError: Cannot read properties of undefined (reading 'length')
    at PortfolioPage (http://localhost:5173/src/pages/PortfolioPage.tsx:1350:31)
    at Object.react_stack_bottom_fram`
- **Screenshot:** `/home/finn/alphapulse/test_screenshots/portfolio.png`

### 交易日志 (`journal`)
- **Status:** ❌ CRASH
- **Detail:** ErrorBoundary triggered — 页面加载出错
- **Console errors:**
  - `%o

%s

%s
 TypeError: Cannot read properties of undefined (reading 'slice')
    at http://localhost:5173/src/pages/TradingJournalPage.tsx:478:31
    at Array.forEach (<anonymous>)
    at http://local`
  - `[ErrorBoundary] TypeError: Cannot read properties of undefined (reading 'slice')
    at http://localhost:5173/src/pages/TradingJournalPage.tsx:478:31
    at Array.forEach (<anonymous>)
    at http://l`
- **Screenshot:** `/home/finn/alphapulse/test_screenshots/journal.png`

## ⚠️ Warning Pages

### 仪表盘 (`dashboard`)
- **Status:** ⚠️ CONSOLE_ERR
- **Detail:** Console errors: WebSocket connection to 'ws://100.100.116.63:5173/?token=jIhTjiX21P-8' failed: Connection closed before receiving a handshake response; [vite] failed to connect to websocket (Error: WebSocket closed without opened.). ; PAGE_ERROR: WebSocket closed without opened.
- **Console errors:**
  - `WebSocket connection to 'ws://100.100.116.63:5173/?token=jIhTjiX21P-8' failed: Connection closed before receiving a handshake response`
  - `[vite] failed to connect to websocket (Error: WebSocket closed without opened.). `
  - `PAGE_ERROR: WebSocket closed without opened.`

### 自选股 (`watchlist`)
- **Status:** ⚠️ CONSOLE_ERR
- **Detail:** Console errors: In HTML, whitespace text nodes cannot be a child of <%s>. Make sure you don't have any extra whitespace between tags on each line of your source code.
This will cause a hydration error.%s tr 

  ...
    <AnimatedView>
      <div className="animated-v...">
        <Suspense fallback={<LoadingSpinner>}>
          <WatchlistPage>
            <DndContext sensors={[...]} collisionDetection={function closestCenter} onDragStart={function} ...>
              <div>
                <div>
                <div>
                <div className="hidden sm:..." style={{...}}>
                  <table className="w-full tex...">
                    <thead>
>                     <tr style={{background:"var(--colo..."}}>
>                       {" "}
                        ...
                    ...
                ...
                ...
              ...
              ...
; PAGE_ERROR: Failed to execute 'addColorStop' on 'CanvasGradient': The value provided ('var(--color-text-secondary)') could not be parsed as a color.; PAGE_ERROR: Failed to execute 'addColorStop' on 'CanvasGradient': The value provided ('var(--color-text-secondary)') could not be parsed as a color.
- **Console errors:**
  - `In HTML, whitespace text nodes cannot be a child of <%s>. Make sure you don't have any extra whitespace between tags on each line of your source code.
This will cause a hydration error.%s tr 

  ...
 `
  - `PAGE_ERROR: Failed to execute 'addColorStop' on 'CanvasGradient': The value provided ('var(--color-text-secondary)') could not be parsed as a color.`
  - `PAGE_ERROR: Failed to execute 'addColorStop' on 'CanvasGradient': The value provided ('var(--color-text-secondary)') could not be parsed as a color.`
  - `PAGE_ERROR: Failed to execute 'addColorStop' on 'CanvasGradient': The value provided ('var(--color-text-secondary)') could not be parsed as a color.`
  - `PAGE_ERROR: Failed to execute 'addColorStop' on 'CanvasGradient': The value provided ('var(--color-text-secondary)') could not be parsed as a color.`

### 信号 (`signals`)
- **Status:** ⚠️ ERROR
- **Detail:** Shows error message, no data
- **Console errors:**
  - `Failed to load resource: the server responded with a status of 400 (Bad Request)`
  - `Failed to load resource: the server responded with a status of 400 (Bad Request)`

### 资金流向 (`flow`)
- **Status:** ⚠️ CONSOLE_ERR
- **Detail:** Console errors: Failed to load resource: the server responded with a status of 400 (Bad Request); Failed to load resource: the server responded with a status of 400 (Bad Request)
- **Console errors:**
  - `Failed to load resource: the server responded with a status of 400 (Bad Request)`
  - `Failed to load resource: the server responded with a status of 400 (Bad Request)`

### 设置 (`settings`)
- **Status:** ⚠️ CONSOLE_ERR
- **Detail:** Console errors: Encountered two children with the same key, `%s`. Keys should be unique so that components maintain their identity across updates. Non-unique keys may cause children to be duplicated and/or omitted — the behavior is unsupported and could change in a future version. /api/watchlist; Encountered two children with the same key, `%s`. Keys should be unique so that components maintain their identity across updates. Non-unique keys may cause children to be duplicated and/or omitted — the behavior is unsupported and could change in a future version. /api/watchlist
- **Console errors:**
  - `Encountered two children with the same key, `%s`. Keys should be unique so that components maintain their identity across updates. Non-unique keys may cause children to be duplicated and/or omitted — `
  - `Encountered two children with the same key, `%s`. Keys should be unique so that components maintain their identity across updates. Non-unique keys may cause children to be duplicated and/or omitted — `

## Full Results

| Page | View | Status | Detail |
|------|------|--------|--------|
| 仪表盘 | `dashboard` | ⚠️ CONSOLE_ERR | Console errors: WebSocket connection to 'ws://100.100.116.63:5173/?token=jIhTjiX |
| 自选股 | `watchlist` | ⚠️ CONSOLE_ERR | Console errors: In HTML, whitespace text nodes cannot be a child of <%s>. Make s |
| 行情 | `market` | ✅ OK | Has content (tables:True, cards:False) |
| K线 | `kline` | ✅ OK | Page loaded |
| 个股分析 | `analyze` | ✅ OK | Page loaded |
| 板块 | `sectors` | ✅ OK | Page loaded |
| 对比 | `compare` | ✅ OK | Page loaded |
| 新闻 | `news` | ✅ OK | Page loaded |
| 持仓 | `portfolio` | ❌ CRASH | ErrorBoundary triggered — 页面加载出错 |
| 交易日志 | `journal` | ❌ CRASH | ErrorBoundary triggered — 页面加载出错 |
| 候选股 | `candidates` | ✅ OK | Has content (tables:True, cards:False) |
| 选股器 | `screener` | ✅ OK | Page loaded |
| 龙虎榜 | `dragon-tiger` | ✅ OK | Has content (tables:True, cards:False) |
| 热门概念 | `hot-concepts` | ✅ OK | Page loaded |
| 策略 | `strategies` | ✅ OK | Page loaded |
| 信号 | `signals` | ⚠️ ERROR | Shows error message, no data |
| 自选分析 | `watchlist-analysis` | ✅ OK | Page loaded |
| 资金流向 | `flow` | ⚠️ CONSOLE_ERR | Console errors: Failed to load resource: the server responded with a status of 4 |
| 趋势 | `trends` | ✅ OK | Page loaded |
| 市场广度 | `breadth` | ✅ OK | Has content (tables:False, cards:True) |
| 市场情绪 | `sentiment` | ✅ OK | Has content (tables:False, cards:True) |
| 每日简报 | `daily-brief` | ✅ OK | Has content (tables:False, cards:True) |
| 系统诊断 | `diag` | ✅ OK | Has content (tables:True, cards:True) |
| 异常检测 | `anomalies` | ✅ OK | Has content (tables:False, cards:True) |
| 机构追踪 | `institutions` | ✅ OK | Has content (tables:False, cards:True) |
| 自选排名 | `ranking` | ✅ OK | Has content (tables:False, cards:True) |
| 每日报告 | `daily-report` | ✅ OK | Has content (tables:True, cards:True) |
| 绩效统计 | `perf-stats` | ✅ OK | Has content (tables:True, cards:True) |
| 多周期趋势 | `multi-trend` | ✅ OK | Has content (tables:False, cards:True) |
| 相关性 | `correlation` | ✅ OK | Page loaded |
| 投资计划 | `investment-plans` | ✅ OK | Page loaded |
| 回测 | `backtest` | ✅ OK | Page loaded |
| 策略评估 | `strategy-eval` | ✅ OK | Page loaded |
| 交易日历 | `trade-calendar` | ✅ OK | Page loaded |
| 形态扫描 | `pattern-scanner` | ✅ OK | Has content (tables:True, cards:True) |
| 持仓风险 | `portfolio-risk` | ✅ OK | Has content (tables:True, cards:True) |
| 快捷操作 | `quick-actions` | ✅ OK | Has content (tables:False, cards:True) |
| 设置 | `settings` | ⚠️ CONSOLE_ERR | Console errors: Encountered two children with the same key, `%s`. Keys should be |
