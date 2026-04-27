# Frontend Fix & Alpha300 Selector — SPEC

## Task 1: Fix Data Extraction Bugs

### 1a. SignalsPage.tsx — signal history broken
The `signalApi.history()` endpoint `/api/signal-history` returns `{ ok, items, total, level_counts }`.
Line ~245 does `setHistory(res.data ?? [])` which sets the whole object as array.
Fix: `setHistory(res.data?.items ?? [])`

### 1b. Verify all pages with `.data` extraction
For each page, the API response format is inconsistent. Some endpoints return `{ ok, data: {...} }` (WRAPPED), others return `{ ok, items, ... }` (DIRECT).

**WRAPPED endpoints** (response has `{ ok, data }`):
- `/api/portfolio` → `{ ok, data: [] }`
- `/api/portfolio/analytics` → `{ ok, data: {...} }`
- `/api/portfolio/risk` → `{ ok, data: {...} }`
- `/api/trading-journal` → `{ ok, data: { items, summary, positions } }`
- `/api/trading-journal/stats` → `{ ok, data: {...} }`
- `/api/trading-journal/calendar` → `{ ok, data: {...} }`
- `/api/candidates` → `{ ok, data: { items, limit, tier_counts, fetched_at } }`
- `/api/market/breadth` → `{ ok, data: {...} }`
- `/api/watchlist-groups` → `{ ok, data: { groups, assignments } }`

**DIRECT endpoints** (no `data` wrapper):
- `/api/watchlist` → `[...]` (direct array)
- `/api/market/overview` → `{ advance_count, ... }`
- `/api/market/trends` → `{ indices, watchlist_stocks, ... }`
- `/api/market/sentiment` → `{ ok, fear_greed_index, ... }`
- `/api/screener` → `{ ok, count, results, ... }`
- `/api/dragon-tiger` → `{ ok, items, ... }`
- `/api/market/hot-concepts` → `{ concepts, ok }` (NOTE: currently 500 error)
- `/api/signal-history` → `{ ok, items, total, level_counts }`
- `/api/alerts` → `{ ok, alerts, ... }`
- `/api/investment-plans` → `{ ok, plans }`
- `/api/daily-report/list` → `{ ok, reports }`
- `/api/daily-brief` → `{ ok, market, sectors, watchlist, ... }`
- `/api/strategies` → `{ ok, strategies, ... }`
- `/api/performance-stats` → `{ ok, endpoints, ... }`
- `/api/stockinfo` → `{ code, name, quote, flow, ... }`
- `/api/analyze` → `{ code, name, quote, order_flow, ... }`

The following pages already have correct extraction (DO NOT change):
- CandidatesPage: `res.data.data` ✅
- BreadthPage: `res.data.data` ✅
- PortfolioPage: `posRes.value.data.data` ✅
- TradingJournalPage: `tradesRes.value.data.data` ✅
- PortfolioRiskPage: `res.data.data` ✅
- WatchlistAnalysisPage: defensive extraction ✅
- DragonTigerPage: defensive extraction ✅
- HotConceptsPage: defensive extraction ✅
- SectorsPage: defensive extraction ✅

## Task 2: Create Alpha300Selector Component

Create `/home/finn/alphapulse/frontend/src/components/Alpha300Selector.tsx`

A reusable dropdown/popup that:
1. Fetches candidates from `candidatesApi.list({ limit: 300 })` on first open
2. Shows a searchable list of stocks with: rank, code, name, score, tier badge
3. Has a search input to filter by code or name
4. Clicking a stock calls `onSelect(code: string)` callback
5. Caches the data for 5 minutes to avoid refetching
6. Uses the app's design system (CSS variables for colors, glass morphism style)
7. Shows tier badges: focus=red, observe=orange, default=gray
8. Compact design — fits in a popup/dropdown

```tsx
interface Alpha300SelectorProps {
  open: boolean;
  onClose: () => void;
  onSelect: (code: string) => void;
}
```

## Task 3: Add Alpha300 Quick-Select to All Stock Code Inputs

Add a button/icon next to each stock code input that opens the Alpha300Selector.

**Pages with StockSearch component** (already have autocomplete — add Alpha300 button):
1. `AnalyzePage.tsx` — two search bars
2. `WatchlistPage.tsx` — add to watchlist search
3. `MarketPage.tsx` — quote lookup
4. `KlinePage.tsx` — K-line chart

**Pages with plain text inputs** (add Alpha300 button next to input):
5. `ComparePage.tsx` line ~131 — sector compare code input
6. `ComparePage.tsx` line ~297 — backtest compare codes input (multi-select, comma separated)
7. `BacktestPage.tsx` line ~183 — backtest codes input (multi-select, comma separated)
8. `PortfolioPage.tsx` line ~739 — add position code input
9. `TradingJournalPage.tsx` line ~1002 — add trade code input
10. `InvestmentPlansPage.tsx` line ~132 — new plan code input
11. `FlowPanelPage.tsx` line ~262 — fund flow code input

For multi-select inputs (ComparePage backtest, BacktestPage), selecting a stock should APPEND the code with comma separator.

For single-select inputs, selecting should REPLACE the current value.

## Implementation Notes

- The Alpha300Selector should be a modal/popup overlay (similar to CommandPalette pattern)
- Use the same glass morphism styling as other modals
- For mobile, make it full-screen
- The selector should show a small badge/icon button (🎯 or similar) next to the input
- When no candidates data available, show a loading spinner then error state
- Import `candidatesApi` from `@/lib/api`
- All changes in `/home/finn/alphapulse/frontend/src/`
