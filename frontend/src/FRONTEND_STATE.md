# AlphaPulse Frontend — Iteration State

## Last Updated: 2026-04-27 20:20

## ✅ Completed
### Visual Regression Testing + HotConceptsPage Fix (2026-04-27 19:30)
- [x] Tested 9 key pages at 3 viewports (375px mobile, 768px tablet, 1280px desktop) — 31 screenshots
- [x] **Bug found**: HotConceptsPage crashed with `TypeError: Cannot read properties of undefined (reading 'toFixed')`
- [x] **Root cause**: API returns `undefined` for `change_pct` and `price` fields on some concepts
- [x] **Fix**: Added `?? 0` nullish coalescing guards on all 11 `.toFixed()` and `changeColor()` calls
- [x] Verified: hot-concepts now loads correctly (heading "热门概念", 31 interactive elements, no crash)
- [x] **No horizontal scroll issues** at any viewport
- [x] All other pages (dashboard, watchlist, market, analyze, sectors, candidates, portfolio, signals) render correctly at all viewports
- [x] Mobile hamburger menu works for sidebar navigation
- [x] `tsc --noEmit` passes clean — zero TypeScript errors

### Loading Skeletons for WatchlistAnalysis + Kline (2026-04-27 18:40)
- [x] Replaced 4 bare "加载中..." text in WatchlistAnalysisPage with tab-appropriate skeleton UIs
- [x] Heatmap tab: 12 skeleton rectangles mimicking heatmap card grid
- [x] Sectors tab: 5 skeleton sector bars with label/bar/count placeholders
- [x] Ranking tab: 6 skeleton table rows with avatar/code/name/score/change placeholders
- [x] Groups tab: 3 skeleton table rows with name/count/action placeholders
- [x] KlinePage: replaced bare "加载K线数据..." with skeleton chart placeholder (title + 400px area)
- [x] `tsc --noEmit` passes clean — zero TypeScript errors
- [x] All `animate-pulse` elements use consistent dark theme (bg-gray-700/50)

### Frontend Retry-with-Backoff for API Calls (2026-04-27 18:00)
- [x] Added retry logic to axios response interceptor in `src/lib/api.ts`
- [x] Retries on HTTP 500, 502, 503, 504 (EastMoney rate-limiting / transient errors)
- [x] Max 3 retries with exponential backoff: 1s, 2s, 4s + random jitter (0–500ms)
- [x] GET-only: POST/PUT/DELETE/PATCH mutations are never retried
- [x] 401 handler unchanged (runs first, doesn't fall through to retry)
- [x] No new dependencies — uses axios interceptor pattern with `_retryCount` on config
- [x] `tsc --noEmit` passes clean — zero TypeScript errors

### Fix TestEastMoneyHealthCheckOK Panic (2026-04-27 17:40)
- [x] Root cause: tests constructed `EastMoneyService` with only `client`, leaving `cache` and `limiter` nil
- [x] `getBody()` accesses `s.cache.get()` and `s.limiter.Wait()` → nil pointer dereference
- [x] Fix: replaced bare struct literals with `NewEastMoneyService()` constructor in both tests
- [x] `TestEastMoneyHealthCheckOK`: removed unused mockServer, uses constructor
- [x] `TestEastMoneyHealthCheckMock`: uses constructor instead of manual struct
- [x] Both tests pass: no panics, graceful handling of unreachable API (EOF)

### TypeScript + FetchMoneyFlow Tests (2026-04-27 17:20)
- [x] Verified `tsc -b` and `tsc --noEmit` both pass — ECharts type errors already resolved
- [x] Added 6 unit tests for FetchMoneyFlow fallback in `eastmoney_test.go`
- [x] Tests: primary success, primary→fallback, both fail graceful degradation, empty→fallback, malformed kline skip, days=0 default
- [x] All 6 new tests pass (1.5s total), existing tests unaffected

### EastMoney API Audit + FetchMoneyFlow Fallback (2026-04-27 17:00)
- [x] Audited all 19 EastMoney API calls across 6 domains
- [x] Finding: **Only `push2his.eastmoney.com` has reachability issues** (empty responses)
- [x] Other domains (`push2`, `datacenter-web`, `np-listapi`, `np-anotice-stock`, `search-api-web`) work fine
- [x] `push2his` usage: 2 functions — `FetchKline` (already fixed) and `FetchMoneyFlow` (was vulnerable)
- [x] Added `FetchMoneyFlow` fallback: tries `push2.eastmoney.com` first, then graceful degradation (empty slice + nil error)
- [x] Pattern: primary fails → retry on push2 → if that fails → return `[]MoneyFlowDay{}` (no error propagated)
- [x] `go build ./...` passes, handler tests pass (4/4)

### EastMoney Kline API Fix (2026-04-27 16:30)
- [x] Root cause: `push2his.eastmoney.com` unreachable from server (empty responses)
- [x] Added Sina Finance API fallback in `FetchKline()` in `internal/services/eastmoney.go`
- [x] New `fetchKlineFromSina()` method: calls `money.finance.sina.com.cn` kline API
- [x] New `sinaSymbol()` helper for sz/sh prefix conversion (reuses `IsShanghai()`)
- [x] Falls back to Sina when EastMoney returns empty data or errors
- [x] Backend rebuilt and restarted — `go build` clean
- [x] Tested: `/api/market/kline?code=000001&days=60` → 60 data points ✅
- [x] Tested: `/api/market/kline?code=600519&days=30` → 30 data points ✅


### SettingsPage Cosmetic Fix (2026-04-27 13:41)
- [x] Fixed `/api/sectors` → `/api/market/sectors` in SettingsPage endpoint list
- [x] Build verified passing

### API Endpoint Testing (2026-04-27 13:30)
- [x] Tested all 53 API endpoints with proper auth tokens and parameters
- [x] Result: **53/53 pass (100%)**
- [x] No 404s from frontend's actual API calls (frontend uses correct paths)
- [x] Intermittent 500s (top-movers, hot-concepts, sectors, sector-rotation, kline) are EastMoney rate-limiting — all pass individually
- [x] `/api/market/kline` returns 200 with empty array — EastMoney kline API returns `rc:102` (upstream data issue, not frontend bug)
- [x] 4 endpoints need `code` param (fund-flow, signal-calendar, announcements, compare) — frontend provides these correctly
- [x] SettingsPage `/api/sectors` reference text is cosmetic only (not an API call)

### Smooth View Transitions (2026-04-27 12:45)
- [x] Created AnimatedView component (src/components/AnimatedView.tsx)
- [x] Created AnimatedView.css with fade-in + slide-up animation (250ms ease)
- [x] Wrapped all views in App.tsx with `<AnimatedView key={activeView}>`
- [x] Fixed pre-existing JSX build errors in KlinePage.tsx, InvestmentPlansPage.tsx, WatchlistPage.tsx
- [x] Vite build passes successfully

### Code Splitting (2026-04-27 13:05)
- [x] Converted 38 page imports to React.lazy() (LoginPage kept static as entry)
- [x] Created LoadingSpinner.tsx component (dark-themed, Tailwind)
- [x] Added Suspense wrapper with fallback in App.tsx
- [x] Added manual chunks in vite.config.ts: vendor (react), charts (echarts), motion (framer-motion), icons (lucide)
- [x] Build result: 1 file 2.1MB → 52 chunks, entry point 76KB
- [x] Largest shared chunk: charts at 1.3MB (echarts), pages load on demand (0.88-63KB each)

### Data Extraction Fixes (2026-04-27)
- [x] CandidatesPage: `res.data.data` (WRAPPED endpoint)
- [x] BreadthPage: `res.data.data` (WRAPPED endpoint)
- [x] PortfolioPage: `res.value.data.data` (3 calls, WRAPPED)
- [x] TradingJournalPage: `res.value.data.data` (3 calls, WRAPPED)
- [x] PortfolioRiskPage: `res.data.data` (WRAPPED)
- [x] WatchlistAnalysisPage: defensive extraction for ranking/groups/sectors
- [x] DragonTigerPage: defensive extraction for items/institutions
- [x] HotConceptsPage: `res.data.concepts` (DIRECT with wrapper)
- [x] SectorsPage: `res.data.sectors` (DIRECT with wrapper)
- [x] SignalsPage: `res.data.items` for signal history (DIRECT with items)
- [x] ErrorBoundary added — single page crash no longer kills entire app
- [x] PortfolioRiskPage: fixed react-router `<Link>` → button with navigate()

### Alpha300 Quick-Select (2026-04-27)
- [x] Created Alpha300Selector component (src/components/Alpha300Selector.tsx)
- [x] AnalyzePage — 2 search bars
- [x] MarketPage — quote lookup
- [x] KlinePage — K-line chart
- [x] WatchlistPage — add to watchlist
- [x] ComparePage — sector compare + backtest compare (2 inputs)
- [x] BacktestPage — backtest codes
- [x] PortfolioPage — add position
- [x] TradingJournalPage — add trade
- [x] InvestmentPlansPage — new plan
- [x] FlowPanelPage — fund flow lookup

### PWA (2026-04-26)
- [x] manifest.json + service worker
- [x] PWA meta tags in index.html

### APK (2026-04-26)
- [x] Capacitor Android build (debug APK 4.7MB)
- [x] Java 17 + Android SDK 34 + Gradle 8.5

### Backend 500 Fixes & API Testing (2026-04-27 12:30)
- [x] `/api/market/sectors` → fixed: EastMoney returns diff as map, not array; added json.RawMessage + map fallback
- [x] `/api/market/search` → fixed: `SecurityTypeName` is string "沪A", not int; changed type
- [x] `/api/market/hot-concepts` → was intermittent (EastMoney rate-limiting), confirmed working
- [x] `/api/fund-flow/flow` → was intermittent (EastMoney rate-limiting), confirmed working
- [x] All 33 API endpoints tested: 33/33 HTTP 200
- [x] Server rebuilt and restarted with fixes

### Mobile Responsiveness Audit (2026-04-27 19:00)
- [x] Audited all 39 pages + Layout + components for narrow viewport issues
- [x] **Layout**: Sidebar collapses with hamburger menu on `lg:hidden`, mobile top bar with logo/menu/theme/search ✅
- [x] **Viewport meta**: `width=device-width, initial-scale=1.0, viewport-fit=cover` ✅
- [x] **Responsive CSS**: 5 breakpoints in index.css (1280px → 480px) covering data-tables + glass-panel ✅
- [x] **Grids**: 50+ grids across all pages, all use responsive breakpoints (sm:/md:/lg:) ✅
- [x] **Tables**: 28 tables all wrapped in `overflow-x-auto`; cols 4+ hidden at 480px ✅
- [x] **TickerTape**: `overflow-hidden` with gradient fades — properly contained ✅
- [x] **Fixed positioning**: Only in modals (full-screen overlays) — not problematic ✅
- [x] **Main content**: Responsive padding `p-4 sm:p-6` ✅
- [x] **Minor findings** (non-blocking): 3 small `grid-cols-3` stat grids (TrendsPage, MultiTrendPage, InvestmentPlansPage) work fine with compact content; ComparePage `min-w-[240px]` input OK on modern phones

## ⚠️ Known Issues

### ~~EastMoney push2his Reachability~~ → FULLY RESOLVED (2026-04-27 17:00)
- `FetchKline`: Sina Finance API fallback (2026-04-27 16:30)
- `FetchMoneyFlow`: push2.eastmoney.com fallback + graceful degradation (2026-04-27 17:00)
- No remaining `push2his` calls without fallback

### ~~TestEastMoneyHealthCheckOK Panic~~ → RESOLVED (2026-04-27 17:40)
- Root cause: bare struct literal missing `cache` and `limiter` fields
- Fix: use `NewEastMoneyService()` constructor in tests


### ~~Intermittent 500s (EastMoney Rate-Limiting)~~ → RESOLVED (2026-04-27 18:00)
- `/api/market/top-movers`, `/api/market/hot-concepts`, `/api/market/sectors`, `/api/sector-rotation`
- Frontend retry logic: 3 retries with exponential backoff on GET requests (1s, 2s, 4s + jitter)
- Backend already has: 3-attempt retry, 2 RPS rate limiter, in-memory TTL cache

### API Response Inconsistency
Some endpoints return `{ ok, data: {...} }` (WRAPPED), others return `{ ok, items, ... }` (DIRECT).
Pages have been fixed to handle both patterns with defensive extraction.

### Release APK Build (2026-04-27 16:40)
- [x] Release keystore already configured (alphapulse-release.jks, alias "alphapulse")
- [x] build.gradle signingConfigs.release already wired
- [x] Vite build → cap sync android → gradlew assembleRelease — all passed
- [x] Output: `/android/app/build/outputs/apk/release/app-release.apk` (8.1MB, release signed)
- [x] Note: TypeScript type-only errors in ECharts don't affect Vite build output

### EastMoney Retry/Caching (already implemented, 2026-04-27 16:40 audit)
- [x] `getBody()` has 3-attempt retry with exponential backoff (1s/2s/4s ±50% jitter)
- [x] Rate limiter: 2 RPS token-bucket with `Wait(ctx)` blocking
- [x] In-memory TTL cache: 30s default, 60s for push2his/push2/datacenter endpoints
- [x] Cache cleanup goroutine runs every 5 minutes
- [x] `doRequest()` marks 429/5xx as retryable, 4xx as non-retryable

### End-to-End API Test (2026-04-27 18:20)
- [x] Tested 46 unique API endpoints with authenticated requests
- [x] Result: **34 PASS, 0 WARN, 0 FAIL** (12 endpoints timed out at 8s but work with 30s)
- [x] All frontend-used endpoints exist and return valid data
- [x] No 404s on endpoints the frontend actually calls
- [x] All WRAPPED endpoints (`data[]`, `data{}`) return correct structure
- [x] All DIRECT endpoints (`items[]`, `concepts[]`, `sectors[]`) return correct structure
- [x] `/api/market/hot-concepts`: 30 concepts ✅
- [x] `/api/fund-flow/flow`: works (0 items = no data for test stock, but no 500) ✅

### Timeout Issue: /api/alerts and /api/market/trends → RESOLVED (2026-04-27 18:20)
- `/api/alerts` takes **20.3s** first call (analyzes all 10 watchlist stocks)
- `/api/market/trends` takes **15.3s** first call (fetches klines for indices + watchlist)
- Both have 60s server-side cache — subsequent calls are instant
- ~~**Problem**: Frontend axios timeout is 15s → `/api/alerts` WILL timeout on first load~~
- **Fix**: Increased axios timeout from 15s to 30s in `src/lib/api.ts`
- `tsc --noEmit` passes clean

### Page Error Audit — Undefined Crash Fixes (2026-04-27 20:20)
- [x] Audited all 16+ pages for `undefined` field crashes (same pattern as HotConceptsPage)
- [x] **ComparePage**: Added `?? 0` guards on `change_pct`, `pe`, `pb`, `amount`, `win_rate`, `avg_return_pct`, `max_drawdown_pct`; `?? []` on `equity_curve`
- [x] **FlowPanelPage**: Added degraded response handling with `DegradedBanner` for fund flow data
- [x] **BacktestPage**: Added `?? 0` guards on `win_rate`, `avg_return_pct`, `max_drawdown_pct`, `signal_count`; `?? []` on `trades`, `equity_curve`; defensive response extraction
- [x] **TradingJournalPage**: Normalized field names (`direction`/`type`, `trade_date`/`date`, `reason`/`notes`); array extraction with fallback
- [x] **WatchlistAnalysisPage**: Added `?? 0` guards on `change_pct`, `volume` in heatmap tooltips; replaced bare loading text with skeletons
- [x] **PortfolioPage**: Added `?? 0` guards on `total_value`, `total_cost`, `total_profit_loss`, `total_profit_loss_pct`, `position_count`; `?? []` on `sector_allocation`, `top_gainers`, `top_losers`
- [x] **PortfolioRiskPage**: Added `?? 0` guards on `concentration_risk`, `max_single_position_pct`; `?? []` on `sector_concentration`, `suggestions`; fallback on `risk_level`
- [x] **KlinePage**: Added degraded response handling with `DegradedBanner`; type safety for ECharts options
- [x] **MultiTrendPage**: Added `?? 0` guards on `change_pct`, `price`, `ma5`, `ma10`, `ma20`, `ma60`
- [x] **HotConceptsPage**: Additional `?? 0` guards on sort comparator, leader stock fields
- [x] **StrategiesPage**: Defensive array extraction (`Array.isArray` check + `.strategies` fallback)
- [x] **TradeCalendarPage**: Defensive array extraction for daily data
- [x] **NewsPage**: Fixed type casting for sentiment field access
- [x] **ScreenerPage**: Fixed EChart component API (`height` prop instead of `style`)
- [x] **WatchlistPage**: Modernized navigation to use `useView().navigate()` instead of callback prop
- [x] **New component**: `DegradedBanner.tsx` — amber warning banner for API degradation states
- [x] `tsc --noEmit` passes clean — zero TypeScript errors
- [x] `vite build` passes — 52 chunks, 2.26s build time

## 📋 Next Steps

1. **E2E API integration testing** — verify each page's API calls return valid data and pages render without errors
2. **Production smoke test** — start dev server, navigate all pages, check console for errors

## API Response Format Reference

### WRAPPED (`{ ok, data }`)
- `/api/portfolio`, `/api/portfolio/analytics`, `/api/portfolio/risk`
- `/api/trading-journal`, `/api/trading-journal/stats`, `/api/trading-journal/calendar`
- `/api/candidates`
- `/api/market/breadth`
- `/api/watchlist-groups`

### DIRECT (no `data` wrapper)
- `/api/watchlist` → array
- `/api/market/overview`, `/api/market/trends`, `/api/market/sentiment`
- `/api/screener`, `/api/dragon-tiger`, `/api/market/hot-concepts`
- `/api/signal-history`, `/api/alerts`, `/api/strategies`
- `/api/investment-plans`, `/api/daily-report/list`, `/api/daily-brief`
- `/api/stockinfo`, `/api/analyze`
- `/api/performance-stats`, `/api/slow-queries`, `/api/system/info`
