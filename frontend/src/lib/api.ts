import axios from 'axios';

// Token readiness gate — prevents API calls before auth initialization
let tokenReadyResolve: () => void;
export const tokenReady = new Promise<void>((resolve) => {
  tokenReadyResolve = resolve;
});

/** Call this from AuthProvider once auth init completes (success or failure) */
export function resolveTokenReady() {
  tokenReadyResolve();
}

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
});

// Request interceptor — attach JWT token
api.interceptors.request.use((config) => {
  const url = config.url ?? '';

  // Auth endpoints bypass the gate to avoid deadlock
  if (url.includes('/auth/login') || url.includes('/auth/verify')) {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  }

  // All other requests wait for auth initialization to complete
  return tokenReady.then(() => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  });
});

// Retryable server-error status codes (e.g. EastMoney rate-limiting)
const RETRYABLE_STATUSES = new Set([500, 502, 503, 504]);
const MAX_RETRIES = 3; // up to 4 total attempts

// Response interceptor — retry on transient 5xx, handle 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    const config = error.config as (typeof error.config & { _retryCount?: number }) | undefined;
    const status = error.response?.status;

    // --- 401 handler (unchanged) ---
    if (status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
      return Promise.reject(error);
    }

    // --- Retry logic for transient 5xx on GET requests ---
    if (
      config &&
      status !== undefined &&
      RETRYABLE_STATUSES.has(status) &&
      (config.method ?? 'get').toLowerCase() === 'get'
    ) {
      const retryCount = config._retryCount ?? 0;
      if (retryCount < MAX_RETRIES) {
        config._retryCount = retryCount + 1;
        // Exponential backoff: 1s, 2s, 4s + jitter (0–500 ms)
        const delay = Math.pow(2, retryCount) * 1000 + Math.random() * 500;
        return new Promise((resolve) =>
          setTimeout(() => resolve(api(config)), delay)
        );
      }
    }

    return Promise.reject(error);
  }
);

export default api;

// --- Auth ---
export interface LoginResponse {
  token: string;
  refresh_token: string;
  user: User;
}

export interface User {
  id: string;
  username: string;
  role: string;
  created_at: string;
}

export const authApi = {
  login: (username: string, password: string) =>
    api.post<LoginResponse>('/auth/login', { username, password }),
  verify: () =>
    api.get<{ valid: boolean; user: User }>('/auth/verify'),
};

// --- Watchlist ---
export interface WatchlistItem {
  id: string;
  code: string;
  name?: string;
  group_name: string;
  added_at: string;
}

export const watchlistApi = {
  list: () => api.get<WatchlistItem[]>('/watchlist'),
  add: (code: string, group_name?: string) =>
    api.post('/watchlist', { code, group_name }),
  remove: (code: string) => api.delete(`/watchlist/${code}`),
  batchAdd: (codes: string[]) =>
    api.post('/watchlist/batch', { codes }),
};

// --- Market ---
export interface Quote {
  code: string;
  name: string;
  price: number;
  open: number;
  prev_close: number;
  high: number;
  low: number;
  change: number;
  change_percent: number;
  updated_at: string;
}

export interface KlinePoint {
  date: string;
  open: number;
  close: number;
  high: number;
  low: number;
  volume: number;
  amount: number;
}

export interface Sector {
  code: string;
  name: string;
  price: number;
  change: number;
  change_percent: number;
}

export interface OverviewIndex {
  code: string;
  name: string;
  price: number;
  change: number;
  change_percent: number;
  advance_count: number;
  decline_count: number;
  flat_count: number;
}

export interface MarketOverview {
  advance_count: number;
  decline_count: number;
  flat_count: number;
  indices: OverviewIndex[];
  updated_at: string;
}

export interface NewsItem {
  code?: string;
  title: string;
  summary?: string;
  source?: string;
  url: string;
  published_at: string;
}

export interface SearchSuggestion {
  code: string;
  name: string;
}

export interface TopMover {
  code: string;
  name: string;
  price: number;
  change: number;
  change_percent: number;
  volume: number;
  amount: number;
  amplitude: number;
}

export const marketApi = {
  quote: (code: string) => api.get<Quote>('/market/quote', { params: { code } }),
  kline: (code: string, days?: number) =>
    api.get<KlinePoint[]>('/market/kline', { params: { code, days } }),
  sectors: () => api.get<Sector[]>('/market/sectors'),
  overview: () => api.get<MarketOverview>('/market/overview'),
  news: () => api.get<NewsItem[]>('/market/news'),
  search: (q: string) =>
    api.get<SearchSuggestion[]>('/market/search', { params: { q } }),
  topMovers: (sort?: 'asc' | 'desc', limit?: number) =>
    api.get<TopMover[]>('/market/top-movers', { params: { sort, limit } }),
  session: () => api.get<{ status: string; message: string }>('/market/session'),
  trends: () => api.get('/market/trends'),
  breadth: () => api.get('/market/breadth'),
  sentiment: () => api.get('/market/sentiment'),
};

// --- Compare ---
export interface SectorMember {
  name: string;
  code: string;
  change_pct: number;
  pe: number;
  pb: number;
  amount: number;
}

export interface SectorCompareResult {
  code: string;
  sector_name: string;
  board_code: string;
  top5: SectorMember[];
  current_rank: number;
  total_count: number;
}

export interface BacktestTrade {
  signal_date: string;
  sell_date: string;
  buy_price: number;
  sell_price: number;
  holding_days: number;
  score: number;
  return_pct: number;
}

export interface BacktestResult {
  code: string;
  name: string;
  signal_count: number;
  win_rate: number;
  avg_return_pct: number;
  max_drawdown_pct: number;
  equity_curve: number[];
  trades: BacktestTrade[];
  error?: string;
}

export const compareApi = {
  sectorCompare: (code: string) =>
    api.get<SectorCompareResult>('/compare/sector', { params: { code } }),
  backtestCompare: (codes: string, days?: number) =>
    api.get<BacktestResult[]>('/compare/backtest', { params: { codes, days } }),
};

// --- Portfolio ---
export interface PortfolioPosition {
  id: string;
  code: string;
  name: string;
  quantity: number;
  cost_price: number;
  current_price: number;
  market_value: number;
  profit_loss: number;
  profit_loss_pct: number;
  buy_date: string;
  notes?: string;
}

export interface PortfolioAnalytics {
  total_value: number;
  total_cost: number;
  total_profit_loss: number;
  total_profit_loss_pct: number;
  position_count: number;
  top_gainers: PortfolioPosition[];
  top_losers: PortfolioPosition[];
  sector_allocation: { sector: string; value: number; pct: number }[];
}

export interface PortfolioRisk {
  concentration_risk: number;
  max_single_position_pct: number;
  sector_concentration: { sector: string; pct: number }[];
  risk_level: string;
  suggestions: string[];
}

export const portfolioApi = {
  list: () => api.get<WrappedResponse<PortfolioPosition[]>>('/portfolio'),
  add: (data: { code: string; quantity: number; cost_price: number; buy_date: string; notes?: string }) =>
    api.post('/portfolio', data),
  update: (id: string, data: Partial<PortfolioPosition>) =>
    api.put(`/portfolio/${id}`, data),
  remove: (id: string) => api.delete(`/portfolio/${id}`),
  analytics: () => api.get<WrappedResponse<PortfolioAnalytics>>('/portfolio/analytics'),
  risk: () => api.get<WrappedResponse<PortfolioRisk>>('/portfolio/risk'),
};

// --- Trading Journal ---
export interface TradeRecord {
  id: string;
  code: string;
  name: string;
  direction: 'buy' | 'sell';
  price: number;
  quantity: number;
  amount: number;
  trade_date: string;
  strategy?: string;
  reason?: string;
  emotion?: string;
  result?: string;
  profit_loss?: number;
  profit_loss_pct?: number;
  notes?: string;
}

export interface TradeStats {
  total_trades: number;
  win_rate: number;
  avg_profit_pct: number;
  avg_loss_pct: number;
  total_profit_loss: number;
  best_trade: TradeRecord | null;
  worst_trade: TradeRecord | null;
  avg_holding_days: number;
  profit_factor: number;
}

export interface TradeCalendarDay {
  date: string;
  trade_count: number;
  profit_loss: number;
}

export const tradingJournalApi = {
  list: (params?: { page?: number; limit?: number; code?: string }) =>
    api.get<WrappedResponse<TradeRecord[]>>('/trading-journal', { params }),
  create: (data: Omit<TradeRecord, 'id'>) =>
    api.post('/trading-journal', data),
  remove: (id: string) => api.delete(`/trading-journal/${id}`),
  stats: () => api.get<WrappedResponse<TradeStats>>('/trading-journal/stats'),
  calendar: (params?: { year?: number; month?: number }) =>
    api.get<WrappedResponse<TradeCalendarDay[]>>('/trading-journal/calendar', { params }),
  strategyEval: () => api.get('/trade-strategy-eval'),
};

// --- Strategies ---
export interface Strategy {
  id: string;
  name: string;
  description: string;
  rules: Record<string, unknown>;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export const strategiesApi = {
  list: () => api.get<Strategy[]>('/strategies'),
  create: (data: { name: string; description: string; rules: Record<string, unknown> }) =>
    api.post('/strategies', data),
  update: (id: string, data: Partial<Strategy>) =>
    api.put(`/strategies/${id}`, data),
  remove: (id: string) => api.delete(`/strategies/${id}`),
  activate: (id: string) => api.post(`/strategies/${id}/activate`),
  deactivate: (id: string) => api.post(`/strategies/${id}/deactivate`),
};

// --- Custom Alerts ---
export interface CustomAlert {
  id: string;
  code: string;
  condition: string;
  threshold: number;
  triggered: boolean;
  created_at: string;
}

export const customAlertsApi = {
  list: () => api.get<CustomAlert[]>('/custom-alerts'),
  create: (data: { code: string; condition: string; threshold: number }) =>
    api.post('/custom-alerts', data),
  remove: (id: string) => api.delete(`/custom-alerts/${id}`),
  check: () => api.get('/custom-alerts/check'),
};

// --- Stock Notes ---
export interface StockNote {
  id: string;
  code: string;
  content: string;
  tags: string[];
  created_at: string;
  updated_at: string;
}

export const stockNotesApi = {
  getNotes: (code: string) => api.get<StockNote[]>(`/stock-notes/${code}`),
  create: (data: { code: string; content: string; tags?: string[] }) =>
    api.post('/stock-notes', data),
  update: (id: string, data: Partial<StockNote>) =>
    api.put(`/stock-notes/${id}`, data),
  remove: (id: string) => api.delete(`/stock-notes/${id}`),
  allTags: () => api.get<string[]>('/stock-notes/tags/all'),
};

// --- Dragon Tiger ---
export interface DragonTigerItem {
  code: string;
  name: string;
  close: number;
  change_pct: number;
  reason: string;
  buy_total: number;
  sell_total: number;
  net_buy: number;
  trade_date: string;
  departments?: string;
}

export const dragonTigerApi = {
  list: (days?: number) => api.get<DragonTigerItem[]>('/dragon-tiger', { params: { days } }),
  history: (code: string) => api.get('/dragon-tiger-history', { params: { code } }),
  institutions: (days?: number) => api.get('/institution-tracker', { params: { days } }),
};

// --- Hot Concepts ---
export interface HotConceptLeaderStock {
  code: string;
  name: string;
  change_pct: number;
}

export interface HotConcept {
  code: string;
  name: string;
  price: number;
  change_pct: number;
  change: number;
  rise_count: number;
  fall_count: number;
  leader_stock: HotConceptLeaderStock | null;
}

export const hotConceptsApi = {
  list: () => api.get<HotConcept[]>('/market/hot-concepts'),
  stocks: (code: string) => api.get(`/market/hot-concepts/${code}/stocks`),
  watchlistOverlap: () => api.get('/watchlist-concept-overlap'),
};

// --- Fund Flow ---
export interface FundFlowItem {
  code: string;
  name: string;
  main_net_inflow: number;
  super_large_net: number;
  large_net: number;
  medium_net: number;
  small_net: number;
  change_pct: number;
}

export const fundFlowApi = {
  flow: (params?: { code?: string; days?: number }) =>
    api.get<FundFlowItem[]>('/fund-flow/flow', { params }),
};

// --- Sector Rotation ---
export interface SectorRotationItem {
  code: string;
  name: string;
  change_pct: number;
  volume_change_pct: number;
  net_inflow: number;
  momentum_score: number;
}

export const sectorRotationApi = {
  rotation: (days?: number) => api.get<SectorRotationItem[]>('/sector-rotation', { params: { days } }),
  history: (code: string, days?: number) =>
    api.get('/sector-rotation/history', { params: { code, days } }),
};

// --- Reports ---
export interface Report {
  filename: string;
  title: string;
  date: string;
  type: string;
  size: number;
}

export const reportsApi = {
  list: () => api.get<Report[]>('/reports'),
  get: (filename: string) => api.get(`/reports/${filename}`),
  latest: () => api.get('/daily-report/latest'),
  dailyList: () => api.get('/daily-report/list'),
  generate: () => api.post('/daily-report/generate'),
  dailyBrief: () => api.get('/daily-brief'),
};

// --- Investment Plans ---
export interface InvestmentPlan {
  code: string;
  name: string;
  target_price: number;
  stop_loss: number;
  buy_amount: number;
  notes: string;
  created_at: string;
}

export const investmentPlansApi = {
  list: () => api.get<InvestmentPlan[]>('/investment-plans'),
  upsert: (data: InvestmentPlan) => api.post('/investment-plans', data),
  remove: (code: string) => api.delete(`/investment-plans/${code}`),
};

// --- Signal System ---
export interface SignalEvent {
  code: string;
  name: string;
  signal_type: string;
  direction: string;
  score: number;
  date: string;
}

export interface Anomaly {
  code: string;
  name: string;
  anomaly_type: string;
  description: string;
  severity: string;
  date: string;
}

export const signalApi = {
  calendar: (params?: { code?: string; days?: number }) =>
    api.get<SignalEvent[]>('/signal-calendar', { params }),
  history: (params?: { code?: string; days?: number }) =>
    api.get<{ ok: boolean; items: DashboardSignal[] }>('/signal-history', { params }),
  anomalies: (days?: number) => api.get<Anomaly[]>('/anomalies', { params: { days } }),
};

// --- Dashboard Signals & Activity ---
export interface DashboardSignal {
  timestamp: string;
  code: string;
  name: string;
  level: string;
  message: string;
}

export interface ActivityEntry {
  action: string;
  detail: string;
  timestamp: string;
}

// --- Alerts ---
export interface Alert {
  id: string;
  type: string;
  message: string;
  severity: string;
  created_at: string;
  read: boolean;
}

export const alertsApi = {
  list: () => api.get<Alert[]>('/alerts'),
};

// --- Watchlist Analysis ---
export interface HeatmapItem {
  code: string;
  name: string;
  change_pct: number;
  volume: number;
  sector: string;
  price: number;
  amount: number;
}

export interface RankingItem {
  code: string;
  name: string;
  overall_score: number;
  overall_signal: string;
  dimension_scores: Record<string, number>;
  change_pct: number;
  price: number;
  strengths: string[];
  risks: string[];
  rank: number;
  error?: string;
}

export interface RankingBest {
  code: string;
  name: string;
  score: number;
}

export interface RankingSummary {
  avg_score: number;
  best: RankingBest | null;
  worst: RankingBest | null;
  count: number;
}

export interface RankingResponse {
  ok: boolean;
  items: RankingItem[];
  summary: RankingSummary;
  fetched_at: string;
  error?: string;
}

export interface WatchlistRanking {
  code: string;
  name: string;
  rank: number;
  score: number;
  change_pct: number;
}

export interface WatchlistGroup {
  id: string;
  name: string;
  stock_count: number;
}

export const watchlistAnalysisApi = {
  heatmap: () => api.get<{ ok: boolean; items: HeatmapItem[] }>('/watchlist-heatmap'),
  sectors: () => api.get('/watchlist-sectors'),
  ranking: () => api.get<WatchlistRanking[]>('/watchlist-ranking'),
  fullRanking: () => api.get<RankingResponse>('/watchlist-ranking'),
  groups: () => api.get<WatchlistGroup[]>('/watchlist-groups'),
  createGroup: (name: string) => api.post('/watchlist-groups', { name }),
  updateGroup: (id: string, name: string) => api.put(`/watchlist-groups/${id}`, { name }),
  deleteGroup: (id: string) => api.delete(`/watchlist-groups/${id}`),
  assignStock: (code: string, group_id: string) =>
    api.post('/watchlist-groups/assign', { code, group_id }),
};

// --- Analyze ---
export interface AnalyzeResult {
  code: string;
  name: string;
  score: number;
  dimensions: { name: string; score: number; detail: string }[];
  recommendation: string;
  summary: string;
  fetched_at?: string;
  data_stale?: boolean;
}

export const analyzeApi = {
  analyze: (code: string) => api.get<AnalyzeResult>('/analyze', { params: { code } }),
  stockInfo: (code: string) => api.get('/stockinfo', { params: { code } }),
};

// --- Candidates & Screener ---
export interface CandidateListItem {
  code: string;
  name: string;
  rank: number;
  score: number;
  recommendation_tier: string;
}

export interface CandidatesListPayload {
  limit: number;
  items: CandidateListItem[];
  tier_counts: Record<string, number>;
  fetched_at: string;
}

export interface WrappedResponse<T> {
  ok: boolean;
  data: T;
  degraded?: boolean;
}

export const candidatesApi = {
  list: (params?: { strategy_id?: string; limit?: number }) =>
    api.get<WrappedResponse<CandidatesListPayload>>('/candidates', { params }),
};

export const screenerApi = {
  screen: (params?: Record<string, string | number>) =>
    api.get('/screener', { params }),
};

// --- System ---
export interface SystemStatus {
  version: string;
  uptime: string;
  database: string;
  cache_hit_rate: number;
  memory_usage: number;
  goroutines: number;
}

// --- Dashboard Summary (composite) ---
export interface DashboardSummaryResponse {
  ok: boolean;
  indices?: Array<{
    code: string;
    name: string;
    price: number;
    change: number;
    change_pct: number;
    prev_close?: number;
    open?: number;
    high?: number;
    low?: number;
    volume: number;
    amount: number;
  }>;
  market_overview?: {
    ok: boolean;
    indices: Array<{
      code: string;
      name: string;
      price: number;
      change: number;
      change_pct: number;
      prev_close?: number;
      volume: number;
      amount: number;
    }>;
    market: {
      up_count: number;
      down_count: number;
      flat_count: number;
      limit_up: number;
      limit_down: number;
      sentiment: string;
      sentiment_ratio: number;
    };
    updated_at: string;
  };
  sectors?: Sector[];
  signals?: DashboardSignal[];
  recent_activity?: ActivityEntry[];
  watchlist?: { total: number; avg_change_pct: number | null };
  top_gainers?: Array<{ code: string; name: string; price: number; change_pct: number }>;
  top_losers?: Array<{ code: string; name: string; price: number; change_pct: number }>;
  active_alerts?: number;
  last_report_date?: string;
}

export const dashboardApi = {
  summary: () => api.get<DashboardSummaryResponse>('/dashboard-summary'),
};

export const systemApi = {
  health: () => api.get('/health'),
  info: () => api.get('/system/info'),
  status: () => api.get<SystemStatus>('/system-status'),
  activityLog: () => api.get<{ entries: ActivityEntry[] }>('/activity-log'),
  slowQueries: () => api.get('/slow-queries'),
  performanceStats: () => api.get('/performance-stats'),
  cacheClear: () => api.post('/cache/clear'),
};

// --- Pattern Scanner ---
export interface PatternResult {
  pattern: string;
  category: string;
  direction: string;
  confidence: number;
  date: string;
  description: string;
  code: string;
  name: string;
}

export interface PatternScannerResponse {
  ok: boolean;
  patterns: PatternResult[];
  summary: {
    total: number;
    bullish: number;
    bearish: number;
    neutral: number;
    by_category: Record<string, number>;
    scanned: number;
  };
  cached: boolean;
}

export const patternScannerApi = {
  scan: () => api.get<PatternScannerResponse>('/pattern-scanner'),
};
