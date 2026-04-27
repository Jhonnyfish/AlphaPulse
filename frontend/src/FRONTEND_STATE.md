# AlphaPulse Frontend — Iteration State

## Last Updated: 2026-04-27 13:41

## ✅ Completed

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

## ⚠️ Known Issues

### EastMoney Kline API Broken
- `/api/market/kline` returns `[]` — EastMoney kline endpoint returns `rc:102` with `data:null`
- All other EastMoney endpoints work, but kline is consistently broken
- Likely EastMoney API change or blocking pattern

### Intermittent 500s (EastMoney Rate-Limiting)
- `/api/market/top-movers`, `/api/market/hot-concepts`, `/api/market/sectors`, `/api/sector-rotation`
- These fail under rapid-fire requests but pass individually with 1-2s delays
- Add retry logic with backoff to improve resilience

### API Response Inconsistency
Some endpoints return `{ ok, data: {...} }` (WRAPPED), others return `{ ok, items, ... }` (DIRECT).
Pages have been fixed to handle both patterns with defensive extraction.

## 📋 Next Steps

1. **Fix EastMoney kline API** — `rc:102` response, empty kline data (backend data source issue)
2. **Build release APK** — current is debug signed
3. **Add retry/caching for EastMoney** — reduce intermittent 500s under rapid requests

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
