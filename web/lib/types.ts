// ---- Auth ----
export interface User {
  id: string;
  username: string;
  role: string;
  created_at: string;
}

export interface LoginResponse {
  token: string;
  refresh_token: string;
  user: User;
}

export interface VerifyResponse {
  valid: boolean;
  user: { id: string; username: string; role: string };
}

// ---- Search ----
export interface SearchSuggestion {
  code: string;
  name: string;
  type?: string;
}

// ---- Ranking ----
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
  weighted_score: number;
  period_scores: { short: number; medium: number; long: number };
  confidence: {
    overall: number;
    by_dim: Record<string, number>;
    data_age: string;
    missing: string[];
  };
  dim_contributions: Record<string, number>;
  sector: string;
  sector_rank: number;
  sector_total: number;
  strategy: string;
  score_trend: string;
  score_change_7d?: number;
  error?: string;
}

export interface RankingSummary {
  avg_score: number;
  best: { code: string; name: string; score: number } | null;
  worst: { code: string; name: string; score: number } | null;
  count: number;
}

export interface RankingResponse {
  ok: boolean;
  items: RankingItem[];
  summary: RankingSummary;
  fetched_at: string;
  error?: string;
}

// ---- Alpha300 ----
export interface Alpha300Candidate {
  code: string;
  ts_code: string;
  name: string;
  rank: number;
  score: number;
  close: number;
  atr14: number;
  buy_low: number;
  buy_high: number;
  sell_low: number;
  sell_high: number;
  stop_loss: number;
  momentum: number;
  trend: number;
  volatility: number;
  liquidity: number;
  industry: string;
  limit_up_today: boolean;
  limit_up_prev_day: boolean;
  leader_signal: string;
  harvest_risk_level: string;
  focus_rank: number;
  focus_score: number;
  recommendation_tier: string;
  focus_reason: string;
  harvest_risk_note: string;
  in_watchlist: boolean;
}

export interface CandidatesResponse {
  ok: boolean;
  data: {
    limit: number;
    items: Alpha300Candidate[];
    tier_counts: Record<string, number>;
    fetched_at: string;
  };
}

export interface ScreenerResult {
  code: string;
  name: string;
  rank: number;
  score: number;
  close: number;
  momentum: number;
  trend: number;
  volatility: number;
  liquidity: number;
  industry: string;
  recommendation_tier: string;
  leader_signal: string;
  harvest_risk_level: string;
  focus_reason: string;
  in_watchlist: boolean;
}

export interface ScreenerResponse {
  ok: boolean;
  degraded?: boolean;
  count: number;
  total_candidates: number;
  filtered: number;
  results: ScreenerResult[];
  filters: Record<string, string | number>;
}

// ---- Stock Analysis ----
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
  volume: number;
  turnover: number;
  pe?: number;
  pb?: number;
  total_mv?: number;
  amplitude?: number;
  limit_up?: number;
  limit_down?: number;
  updated_at: string;
}

export interface AnalysisSummary {
  overall_score: number;
  overall_signal: string;
  strengths: string[];
  risks: string[];
  suggestion: string;
}

export interface StockAnalysis {
  code: string;
  name: string;
  version: string;
  quote: Quote;
  volume_price: {
    today_change_pct: number;
    today_volume: number;
    avg_volume_5d: number;
    volume_ratio: number;
    turnover: number;
    turnover_level: string;
    price_volume_harmony: string;
    verdict: string;
  };
  valuation: {
    pe: number;
    pe_level: string;
    pb: number;
    pb_level: string;
    total_mv: number;
    mv_level: string;
    verdict: string;
  };
  volatility: {
    amplitude: number;
    amplitude_level: string;
    distance_to_limit_up: number;
    distance_to_limit_down: number;
    verdict: string;
  };
  money_flow: {
    today_main_net: number;
    today_main_direction: string;
    today_huge_net: number;
    today_big_net: number;
    institution_vs_hotmoney: string;
    main_consecutive_days: number;
    main_consecutive_direction: string;
    retail_behavior: string;
    verdict: string;
  };
  technical: {
    ma5: number;
    ma10: number;
    ma20: number;
    ma60: number;
    ma_arrangement: string;
    macd_dif: number;
    macd_dea: number;
    macd_hist: number;
    macd_signal: string;
    macd_hist_trend: string;
    kdj_k: number;
    kdj_d: number;
    kdj_j: number;
    kdj_signal: string;
    rsi_14: number;
    rsi_level: string;
    boll_upper: number;
    boll_mid: number;
    boll_lower: number;
    boll_position: string;
    period_align: string;
    verdict: string;
  };
  sector: {
    sectors: { name: string; code: string }[];
    primary_sector: string;
    is_sector_leader: boolean;
    sector_pct_chg_5d: number;
    stock_pct_chg_5d: number;
    rel_strength: number;
    rel_strength_tag: string;
    verdict: string;
  };
  sentiment: {
    news_count: number;
    announcement_count: number;
    key_events: string[];
    sentiment_score: number;
    sentiment_label: string;
    verdict: string;
  };
  fundamentals: {
    roe: number;
    roe_level: string;
    gross_margin: number;
    gross_margin_level: string;
    net_margin: number;
    net_margin_level: string;
    debt_ratio: number;
    debt_ratio_level: string;
    revenue_growth: number;
    revenue_growth_level: string;
    net_profit_growth: number;
    net_profit_growth_level: string;
    score: number;
    verdict: string;
  };
  northbound: {
    latest_net_flow: number;
    trend_5d: string;
    flow_direction: string;
    stock_net_amount: number;
    stock_action: string;
    signal: string;
    verdict: string;
  };
  margin_detail: {
    latest_margin_balance: number;
    margin_balance_trend: string;
    margin_buying_trend: string;
    short_selling_trend: string;
    signal: string;
    sentiment_score: number;
    verdict: string;
  };
  trend_analysis: {
    trend_stage: {
      direction: string;
      stage: string;
      strength: string;
      confidence: number;
      signals: string[];
      description: string;
    };
    support_resistance: {
      support1: number;
      support2: number;
      support3: number;
      resistance1: number;
      resistance2: number;
      resistance3: number;
      price_position: string;
      nearest_level: number;
      nearest_type: string;
      distance_pct: number;
    };
    verdict: string;
  };
  buy_zone?: {
    zones: {
      method: string;
      upper_bound: number;
      lower_bound: number;
      optimal_entry: number;
      stop_loss: number;
      safety_score: number;
    }[];
    optimal: {
      method: string;
      upper_bound: number;
      lower_bound: number;
      optimal_entry: number;
      stop_loss: number;
      safety_score: number;
    } | null;
    verdict: string;
  };
  t_suggestion?: {
    type: string;
    action: string;
    entry_price: number;
    target_price: number;
    stop_loss: number;
    reason: string;
    confidence: number;
    signal_score: number;
    t_quantity: number;
    position_ratio: number;
    trigger_pct: number;
    expected_profit_pct: number;
    max_loss_pct: number;
    risk_reward: number;
    execution_tip?: string;
    signal_details?: string[];
    risk_notes?: string[];
    condition_buy?: {
      direction: string;
      trigger_price: number;
      trigger_desc: string;
      order_price: number;
      order_type: string;
      quantity_ratio: string;
      stop_price: number;
      stop_desc: string;
      note: string;
    };
    condition_sell?: {
      direction: string;
      trigger_price: number;
      trigger_desc: string;
      order_price: number;
      order_type: string;
      quantity_ratio: string;
      stop_price: number;
      stop_desc: string;
      note: string;
    };
  };
  intraday_forecast?: {
    predicted_high: number;
    predicted_low: number;
    // ±1σ confidence bands — when present, the actual high/low should fall
    // in [predicted_high_down, predicted_high_up] / [predicted_low_down, predicted_low_up]
    // ~68% of the time. Tight band = high-confidence forecast.
    predicted_high_up?: number;
    predicted_high_down?: number;
    predicted_low_up?: number;
    predicted_low_down?: number;
    /** P50 of historical up/down excursions. ~50% of days reach this level — use for high-fill-rate orders. */
    predicted_high_median?: number;
    predicted_low_median?: number;
    sigma_high?: number;
    sigma_low?: number;
    current_zone: string;
    zone_pct: number;
    bias?: string;
    bias_reason?: string;
    /** Sum of pattern confidences driving the bias (0 = none, >1.5 = strong consensus). */
    bias_strength?: number;
    support_level?: number;
    resist_level?: number;
  };
  defensive_strategy?: {
    action: "HOLD" | "REDUCE" | "CLEAR" | "WAIT" | "BUILD" | "RESTORE";
    action_label: string;
    risk_score: number;
    alpha_score: number;
    risk_level: string;
    current_position_pct: number;
    target_position_pct: number;
    allow_intraday_orders: boolean;
    reason: string;
    execution_tip: string;
    risk_notes?: string[];
  };
  strategy_options?: {
    id: string;
    name: string;
    style: string;
    action: "HOLD" | "REDUCE" | "CLEAR" | "WAIT" | "BUILD" | "RESTORE";
    action_label: string;
    target_position_pct: number;
    min_position_pct: number;
    expected_use: string;
    reason: string;
    execution_tip: string;
    risk_notes?: string[];
    recommended: boolean;
  }[];
  pattern_analysis?: {
    signals: PatternSignal[];
    kline_signals?: PatternSignal[];
    chart_signals?: PatternSignal[];
    volume_signals?: PatternSignal[];
    primary?: PatternSignal;
    bullish_count: number;
    bearish_count: number;
    neutral_count: number;
    net_bias: string;
    score_impact: number;
    verdict: string;
  };
  short_term_score?: {
    score: number;
    grade: string;
    signal: string;
    components: {
      name: string;
      score: number;
      weight: number;
      reason: string;
    }[];
    reasons?: string[];
    risks?: string[];
    verdict: string;
  };
  summary: AnalysisSummary;
  data_sources: Record<string, string>;
  errors: Record<string, string>;
  fetched_at: string;
  holding?: {
    quantity: number;
    cost_price: number;
    market_value: number;
    pnl: number;
    pnl_pct: number;
  };
}

export interface PatternSignal {
  pattern: string;
  category: string;
  direction: string;
  confidence: number;
  date: string;
  description: string;
}

// ---- Intraday Forecast Accuracy ----
export interface IntradayForecastDay {
  date: string;
  prev_close: number;
  predicted_high: number;
  predicted_low: number;
  predicted_high_up: number;        // P95 of up excursion
  predicted_low_down: number;       // P5 of down excursion
  predicted_high_median?: number;   // P50 of up excursion
  predicted_low_median?: number;    // P50 of down excursion
  predicted_mid: number;
  actual_high: number;
  actual_low: number;
  actual_close: number;
  high_in_range: boolean;           // actual_high ≤ predicted_high
  low_in_range: boolean;            // actual_low ≥ predicted_low
  both_in_range: boolean;
  high_in_wide_range: boolean;      // actual_high ≤ predicted_high_up
  low_in_wide_range: boolean;       // actual_low ≥ predicted_low_down
  both_in_wide_range: boolean;
  bias: string;
  bias_strength: number;
  zone: string;
}

export interface IntradayForecastAccuracy {
  days_evaluated: number;
  both_in_range_pct: number;     // central band hit rate
  high_in_range_pct: number;
  low_in_range_pct: number;
  both_in_wide_range_pct: number; // ±1σ hit rate
  high_in_wide_range_pct: number;
  low_in_wide_range_pct: number;
  avg_high_miss_pct: number;     // E[(actual-predicted)/predicted] over high-miss days
  avg_low_miss_pct: number;
  avg_range_width: number;
  avg_range_width_pct: number;
  avg_wide_width: number;
  avg_wide_width_pct: number;
  avg_sigma: number;
  /** "high_confidence" ≥70%, "moderate" ≥55%, "low_confidence" <55% */
  reliability: "high_confidence" | "moderate" | "low_confidence";
  order_stats?: OrderLevelStat[];
  details?: IntradayForecastDay[];
}

export interface OrderLevelStat {
  side: "sell" | "buy";
  tag: string;                // "上限" / "预测高" / "中位" / "预测低" / "下限"
  theoretical_fill_pct: number; // expected fill rate (5%/17%/50%)
  empirical_fill_pct: number;   // actual fill rate over backtest window
  fills: number;                // days the order filled
  sample_size: number;          // days considered
  avg_fill_price: number;       // avg price when filled ≈ limit price
  avg_pnl_pct: number;          // avg execution improvement % vs close when filled
  win_rate: number;             // % of filled days with positive P&L
  cumulative_pnl_pct: number;   // geometric compounded improvement over filled days
}

// ---- Deep Analysis ----
export interface DeepAnalysisResponse {
  ok: boolean;
  code: string;
  status?: "running" | "completed" | "failed" | "not_found";
  report?: string;
  error?: string;
  pct_done?: string;
}

// ---- Score History ----
export interface ScoreHistoryEntry {
  score: number;
  dimensions: Record<string, number>;
  recorded_at: string;
}

export interface ScoreTrend {
  current: number;
  prev_7d: number;
  prev_30d: number;
  change_7d: number;
  change_30d: number;
  trend_7d: string;
  trend_30d: string;
}

export interface ScoreHistoryResponse {
  code: string;
  count: number;
  history: ScoreHistoryEntry[];
  trend?: ScoreTrend;
  dim_trends?: Record<string, { current: number; change_7d: number; trend: string }>;
  comparison?: {
    stock_score: number;
    stock_change_5d: number;
    sector_avg_score: number;
    sector_name: string;
    vs_sector: string;
    vs_market: string;
  };
}

// ---- Data Sync ----
export interface SyncStatus {
  ok: boolean;
  status: "idle" | "running" | "completed" | "failed";
  started_at?: string;
  finished_at?: string;
  error?: string;
}

export interface SyncConfig {
  ok: boolean;
  sync_enabled: boolean;
  sync_time: string;
  retry_enabled: boolean;
  retry_time: string;
}

// ---- Portfolio ----
export interface PortfolioPosition {
  id: string;
  code: string;
  name: string;
  cost_price: number;
  quantity: number;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface PortfolioPositionEnriched extends PortfolioPosition {
  current_price: number;
  market_value: number;
  total_cost: number;
  pnl: number;
  pnl_pct: number;
}

// ---- Strategy Backtest Engine ----

export interface DailySignal {
  date: string;
  action: "BUY" | "SELL" | "HOLD";
  price: number;
  position_pct: number;
  score: number;
  reason: string;
  trade_id?: string;
}

export interface StrategyTrade {
  trade_id: string;
  buy_date: string;
  buy_price: number;
  buy_score: number;
  sell_date: string;
  sell_price: number;
  sell_reason: string;
  holding_days: number;
  return_pct: number;
  position_pct: number;
}

export interface EquityPoint {
  date: string;
  equity: number;
  in_position: boolean;
}

export interface StrategyResult {
  code: string;
  name?: string;
  days: number;
  daily_signals?: DailySignal[];
  trades: StrategyTrade[];
  equity_curve: EquityPoint[];
  benchmark_curve: EquityPoint[];
  strategy_return_pct: number;
  buy_hold_return_pct: number;
  sharpe_ratio: number;
  max_drawdown_pct: number;
  win_rate: number;
  avg_holding_days: number;
  total_trades: number;
  signal_efficiency: number;
  error?: string;
}

// ---- T+0 Backtest ----

export interface TBacktestDay {
  date: string;
  signal_type: string; // "正T" / "反T" / "观望"
  entry_price: number;
  target_price: number;
  stop_loss: number;
  signal_score: number;
  t_quantity: number;
  actual_high: number;
  actual_low: number;
  actual_close: number;
  executed: boolean;
  exit_price: number;
  exit_reason: string; // "target" / "stop" / "close" / ""
  profit_pct: number;
  profit_abs: number; // yuan
}

export interface TBacktestResult {
  code: string;
  days_evaluated: number;
  holding_qty: number;
  holding_cost: number;
  details: TBacktestDay[];
  total_trades: number;
  win_count: number;
  win_rate: number;
  avg_profit_pct: number;
  avg_loss_pct: number;
  cumulative_pct: number;
  max_drawdown_pct: number;
  positive_t_trades: number;
  positive_t_win_rate: number;
  reverse_t_trades: number;
  reverse_t_win_rate: number;
  watch_days: number;
  equity_curve: EquityPoint[];
  benchmark_curve: EquityPoint[];
  t_overlay_pct: number;
  buy_hold_return_pct: number;
  enhanced_return_pct: number;
  personality?: StockPersonality;
  error?: string;
}

export interface StockPersonality {
  // Volatility
  atr_pct: number;
  avg_daily_range: number;
  range_cv: number;
  avg_gap_pct: number;
  // T Suitability
  t_suitability: number;
  t_space_pct: number;
  commission_ratio: number;
  recommended_t: string; // "正T" / "反T" / "均可" / "不适合"
  // Trend
  trend_bias: string; // "上涨" / "下跌" / "震荡"
  trend_strength: number;
  mean_reversion_pct: number;
  up_day_pct: number;
  // T History
  positive_t_advantage: number;
  best_signal_window: string;
  avg_signal_interval: number;
  // Volume
  avg_volume: number;
  volume_cv: number;
  // Summary
  tags: string[];
  summary: string;
}
