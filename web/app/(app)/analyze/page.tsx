"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { analysisApi, searchApi } from "@/lib/api-client";
import type { StockAnalysis, DeepAnalysisResponse, ScoreHistoryResponse, SearchSuggestion, IntradayForecastAccuracy, OrderLevelStat } from "@/lib/types";
import BacktestSection from "./BacktestSection";
import { DIM_LABELS, formatPct, formatPrice, formatVolume, formatMoney } from "@/lib/constants";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Search, Loader2, Brain, TrendingUp, TrendingDown, BarChart3,
  ChevronDown, ChevronRight, Activity, Clock, X,
} from "lucide-react";

// ---- Search history (localStorage) ----
const HISTORY_KEY = "alphapulse_search_history";
const HISTORY_MAX = 20;

interface HistoryItem {
  code: string;
  name: string;
  ts: number;
}

function loadHistory(): HistoryItem[] {
  try {
    return JSON.parse(localStorage.getItem(HISTORY_KEY) || "[]");
  } catch { return []; }
}

function saveHistory(code: string, name: string) {
  const list = loadHistory().filter((h) => h.code !== code);
  list.unshift({ code, name, ts: Date.now() });
  localStorage.setItem(HISTORY_KEY, JSON.stringify(list.slice(0, HISTORY_MAX)));
}

function clearHistory() {
  localStorage.removeItem(HISTORY_KEY);
}

export default function AnalyzePage() {
  const searchParams = useSearchParams();
  const codeParam = searchParams.get("code");

  const [query, setQuery] = useState(codeParam || "");
  const [code, setCode] = useState(codeParam || "");
  const [data, setData] = useState<StockAnalysis | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  // Deep analysis state
  const [deepStatus, setDeepStatus] = useState<DeepAnalysisResponse | null>(null);
  const [deepLoading, setDeepLoading] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Score history
  const [history, setHistory] = useState<ScoreHistoryResponse | null>(null);

  // Intraday-forecast accuracy (lazy-fetched on demand)
  const [forecastAccuracy, setForecastAccuracy] = useState<IntradayForecastAccuracy | null>(null);
  const [forecastAccuracyLoading, setForecastAccuracyLoading] = useState(false);
  const [forecastAccuracyError, setForecastAccuracyError] = useState("");
  const [forecastAccuracyRequested, setForecastAccuracyRequested] = useState(false);
  const [accuracyDays, setAccuracyDays] = useState(120);

  // Search suggestions & search history
  const [suggestions, setSuggestions] = useState<SearchSuggestion[]>([]);
  const [showSugg, setShowSugg] = useState(false);
  const [searchHistory, setSearchHistory] = useState<HistoryItem[]>([]);
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Load search history on mount
  useEffect(() => {
    setSearchHistory(loadHistory());
  }, []);

  // refetchAccuracy re-runs the backtest with the current date-range state.
  // Used both for the initial fetch and when the user changes the window.
  const refetchAccuracy = useCallback(async (
    c: string,
    days: number,
    from: string,
    to: string,
  ) => {
    setForecastAccuracyRequested(true);
    setForecastAccuracyLoading(true);
    setForecastAccuracyError("");
    try {
      const acc = await analysisApi.intradayForecastAccuracy(c, { days, from: from || undefined, to: to || undefined });
      setForecastAccuracy(acc);
    } catch (e: any) {
      setForecastAccuracyError(e?.message || "回测失败");
    } finally {
      setForecastAccuracyLoading(false);
    }
  }, []);

  const doAnalyze = useCallback(async (c: string) => {
    if (!c) return;
    setLoading(true);
    setError("");
    setData(null);
    setHistory(null);
    setDeepStatus(null);
    // Reset accuracy state for the new ticker.
    setForecastAccuracy(null);
    setForecastAccuracyError("");
    setForecastAccuracyLoading(false);
    setForecastAccuracyRequested(false);
    try {
      const res = await analysisApi.analyze(c);
      setData(res);
      saveHistory(c, res.name);
      setSearchHistory(loadHistory());
      analysisApi.scoreHistory(c).then(setHistory).catch(() => {});
      // Only fetch accuracy if the ticker actually has an intraday forecast.
      if (res.intraday_forecast) {
        refetchAccuracy(c, accuracyDays, "", "");
      }
    } catch (err: any) {
      setError(err.message || "分析失败");
    } finally {
      setLoading(false);
    }
  }, [accuracyDays, refetchAccuracy]);

  // Auto-analyze if code param present
  useEffect(() => {
    if (codeParam) doAnalyze(codeParam);
  }, [codeParam, doAnalyze]);

  // Search debounce
  function handleInput(v: string) {
    setQuery(v);
    if (searchTimer.current) clearTimeout(searchTimer.current);
    if (!v.trim()) { setSuggestions([]); setShowSugg(false); return; }
    searchTimer.current = setTimeout(async () => {
      try {
        const res = await searchApi.stocks(v.trim());
        setSuggestions(res || []);
        setShowSugg(true);
      } catch { /* ignore */ }
    }, 300);
  }

  function selectStock(s: SearchSuggestion) {
    setQuery(`${s.name} (${s.code})`);
    setCode(s.code);
    setShowSugg(false);
    doAnalyze(s.code);
  }

  function selectFromHistory(h: HistoryItem) {
    setQuery(`${h.name} (${h.code})`);
    setCode(h.code);
    doAnalyze(h.code);
  }

  function handleSubmit() {
    // If user typed a raw code like 600176
    const match = query.match(/(\d{6})/);
    if (match) {
      const c = match[1];
      setCode(c);
      doAnalyze(c);
    }
  }

  function handleClearHistory() {
    clearHistory();
    setSearchHistory([]);
  }

  // ---- Deep analysis ----
  async function startDeep() {
    if (!code) return;
    setDeepLoading(true);
    setDeepStatus(null);
    try {
      const res = await analysisApi.deepAnalysis(code);
      setDeepStatus(res);
      if (res.status === "running") startPolling();
    } catch (err: any) {
      setDeepStatus({ ok: false, code, status: "failed", error: err.message });
    } finally {
      setDeepLoading(false);
    }
  }

  function startPolling() {
    if (pollRef.current) clearInterval(pollRef.current);
    pollRef.current = setInterval(async () => {
      try {
        const res = await analysisApi.deepAnalysisStatus(code);
        setDeepStatus(res);
        if (res.status === "completed" || res.status === "failed") {
          if (pollRef.current) clearInterval(pollRef.current);
        }
      } catch {
        if (pollRef.current) clearInterval(pollRef.current);
      }
    }, 3000);
  }

  useEffect(() => {
    return () => { if (pollRef.current) clearInterval(pollRef.current); };
  }, []);

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 space-y-6">
      {/* Header + Search */}
      <div className="flex items-center gap-3">
        <BarChart3 className="h-5 w-5 text-muted-foreground shrink-0" />
        <h1 className="text-xl font-semibold shrink-0">个股分析</h1>
        <div className="relative flex-1 max-w-md ml-4">
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="输入股票代码或名称，如 600176 或 茅台"
                value={query}
                onChange={(e) => handleInput(e.target.value)}
                onFocus={() => suggestions.length > 0 && setShowSugg(true)}
                onBlur={() => setTimeout(() => setShowSugg(false), 200)}
                onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
                className="pl-9"
              />
              {showSugg && suggestions.length > 0 && (
                <div className="absolute top-full left-0 right-0 z-50 mt-1 border rounded-md bg-popover shadow-md max-h-60 overflow-auto">
                  {suggestions.map((s, i) => (
                    <button
                      key={`${s.code}-${i}`}
                      className="flex items-center justify-between w-full px-3 py-2 text-sm hover:bg-accent transition-colors"
                      onMouseDown={() => selectStock(s)}
                    >
                      <span>{s.name}</span>
                      <span className="text-muted-foreground font-mono text-xs">{s.code}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
            <Button onClick={handleSubmit} disabled={!query.trim()}>
              分析
            </Button>
          </div>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="space-y-4">
          <Skeleton className="h-24 w-full" />
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-32" />
            ))}
          </div>
        </div>
      )}

      {/* Analysis result */}
      {data && !loading && (
        <>
          {/* Quote bar */}
          <Card>
            <CardContent className="py-4">
              <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
                <div className="flex items-baseline gap-2">
                  <span className="text-lg font-bold">{data.name}</span>
                  <span className="text-sm text-muted-foreground font-mono">{data.code}</span>
                </div>
                <span className="text-2xl font-bold tabular-nums">{formatPrice(data.quote.price)}</span>
                <span className="tabular-nums">{formatPct(data.quote.change_percent)}</span>
                <span className="text-xs text-muted-foreground">
                  {data.quote.change > 0 ? "+" : ""}{formatPrice(data.quote.change)}
                </span>
                <Separator orientation="vertical" className="h-4" />
                <span className="text-xs text-muted-foreground">
                  开 {formatPrice(data.quote.open)} 高 {formatPrice(data.quote.high)} 低 {formatPrice(data.quote.low)}
                </span>
                <span className="text-xs text-muted-foreground">
                  量 {formatVolume(data.quote.volume)}
                </span>
                {data.quote.pe != null && (
                  <span className="text-xs text-muted-foreground">PE {data.quote.pe.toFixed(1)}</span>
                )}
                {data.quote.total_mv != null && (
                  <span className="text-xs text-muted-foreground">
                    市值 {formatMoney(data.quote.total_mv * 10000)}
                  </span>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Summary card */}
          {data.summary && (
            <Card>
              <CardHeader className="pb-2 pt-4 px-4">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm font-medium text-muted-foreground">
                    综合评估
                  </CardTitle>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary" className="text-lg tabular-nums px-3">
                      {data.summary.overall_score}
                    </Badge>
                    <Badge variant="outline">{data.summary.overall_signal}</Badge>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="px-4 pb-4">
                {data.summary.suggestion && (
                  <p className="text-sm text-muted-foreground mb-3">{data.summary.suggestion}</p>
                )}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {data.summary.strengths?.length > 0 && (
                    <div>
                      <h4 className="text-xs font-medium text-muted-foreground mb-1">优势</h4>
                      <ul className="space-y-0.5">
                        {data.summary.strengths.map((s, i) => (
                          <li key={i} className="text-sm text-muted-foreground">• {s}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                  {data.summary.risks?.length > 0 && (
                    <div>
                      <h4 className="text-xs font-medium text-muted-foreground mb-1">风险</h4>
                      <ul className="space-y-0.5">
                        {data.summary.risks.map((r, i) => (
                          <li key={i} className="text-sm text-muted-foreground">• {r}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Holding info */}
          {data.holding && (
            <Card className={data.holding.pnl >= 0 ? "border-green-500/30" : "border-red-500/30"}>
              <CardContent className="px-4 py-3">
                <div className="flex items-center gap-4 flex-wrap">
                  <div className="flex items-center gap-1.5">
                    <span className="text-xs text-muted-foreground">持仓</span>
                    <span className="text-sm font-semibold font-mono">{data.holding.quantity.toLocaleString()}股</span>
                  </div>
                  <Separator orientation="vertical" className="h-4" />
                  <div className="flex items-center gap-1.5">
                    <span className="text-xs text-muted-foreground">成本</span>
                    <span className="text-sm font-mono">{data.holding.cost_price.toFixed(2)}</span>
                  </div>
                  <Separator orientation="vertical" className="h-4" />
                  <div className="flex items-center gap-1.5">
                    <span className="text-xs text-muted-foreground">市值</span>
                    <span className="text-sm font-mono">{formatMoney(data.holding.market_value)}</span>
                  </div>
                  <Separator orientation="vertical" className="h-4" />
                  <div className="flex items-center gap-1.5">
                    <span className="text-xs text-muted-foreground">盈亏</span>
                    <span className={`text-sm font-semibold font-mono ${data.holding.pnl >= 0 ? "text-green-500" : "text-red-500"}`}>
                      {data.holding.pnl >= 0 ? "+" : ""}{formatMoney(data.holding.pnl)}
                      <span className="text-xs ml-1">({data.holding.pnl >= 0 ? "+" : ""}{data.holding.pnl_pct.toFixed(2)}%)</span>
                    </span>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Trading plan */}
          <Card className="border-primary/20">
            <CardHeader className="pb-2 pt-4 px-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                  <Activity className="h-4 w-4 text-primary" />
                  <CardTitle className="text-sm font-medium">交易计划</CardTitle>
                </div>
                <div className="flex items-center gap-1.5">
                  {data.t_suggestion ? (
                    <>
                      <Badge variant="outline">{data.t_suggestion.type}</Badge>
                      <Badge variant="secondary" className="text-[10px]">
                        置信度 {data.t_suggestion.confidence.toFixed(0)}%
                      </Badge>
                    </>
                  ) : (
                    <Badge variant="outline">未生成做T</Badge>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent className="px-4 pb-4">
              <Tabs defaultValue="t" className="gap-3">
                <TabsList className="w-full justify-start overflow-x-auto">
                  <TabsTrigger value="t" className="min-w-20">
                    <TrendingUp className="h-3.5 w-3.5" />
                    做T
                  </TabsTrigger>
                  <TabsTrigger value="buy" className="min-w-24">
                    <TrendingDown className="h-3.5 w-3.5" />
                    买入区间
                  </TabsTrigger>
                  <TabsTrigger value="day" className="min-w-24">
                    <Clock className="h-3.5 w-3.5" />
                    日内预测
                  </TabsTrigger>
                  <TabsTrigger value="pattern" className="min-w-24">
                    <BarChart3 className="h-3.5 w-3.5" />
                    形态评分
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="t" className="mt-0">
                  {data.t_suggestion ? (
                    <div className="space-y-3">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-sm font-medium">{data.t_suggestion.action}</span>
                        {data.t_suggestion.reason && (
                          <span className="text-xs text-muted-foreground">{data.t_suggestion.reason}</span>
                        )}
                      </div>
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-x-4 gap-y-1 rounded-md bg-muted/40 p-2">
                        <TMetric l="T仓" v={data.t_suggestion.t_quantity > 0 ? `${data.t_suggestion.t_quantity}股` : "—"} />
                        <TMetric l="仓位" v={data.t_suggestion.position_ratio > 0 ? `${data.t_suggestion.position_ratio.toFixed(0)}%` : "—"} />
                        <TMetric l="收益/风险" v={data.t_suggestion.risk_reward > 0 ? data.t_suggestion.risk_reward.toFixed(2) : "—"} />
                        <TMetric l="触发阈值" v={data.t_suggestion.trigger_pct > 0 ? `±${data.t_suggestion.trigger_pct.toFixed(2)}%` : "—"} />
                        <TMetric l="目标收益" v={data.t_suggestion.expected_profit_pct > 0 ? formatPct(data.t_suggestion.expected_profit_pct) : "—"} />
                        <TMetric l="最大回撤" v={data.t_suggestion.max_loss_pct > 0 ? `-${data.t_suggestion.max_loss_pct.toFixed(2)}%` : "—"} />
                        <TMetric l="信号分" v={data.t_suggestion.signal_score > 0 ? data.t_suggestion.signal_score.toFixed(0) : "—"} />
                        <TMetric l="止损" v={data.t_suggestion.stop_loss > 0 ? data.t_suggestion.stop_loss.toFixed(2) : "—"} />
                      </div>
                      {data.t_suggestion.signal_details && data.t_suggestion.signal_details.length > 0 && (
                        <div className="flex flex-wrap gap-1.5">
                          {data.t_suggestion.signal_details.slice(0, 4).map((detail) => (
                            <Badge key={detail} variant="secondary" className="text-[10px] font-normal">
                              {detail}
                            </Badge>
                          ))}
                        </div>
                      )}
                      {data.t_suggestion.risk_notes && data.t_suggestion.risk_notes.length > 0 && (
                        <div className="space-y-1">
                          {data.t_suggestion.risk_notes.slice(0, 3).map((note) => (
                            <p key={note} className="text-[11px] text-amber-700 dark:text-amber-300">{note}</p>
                          ))}
                        </div>
                      )}
                      {data.t_suggestion.execution_tip && (
                        <p className="text-[11px] text-muted-foreground">{data.t_suggestion.execution_tip}</p>
                      )}
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                        {data.t_suggestion.condition_buy && (
                          <ConditionOrderCard order={data.t_suggestion.condition_buy} step={1} />
                        )}
                        {data.t_suggestion.condition_sell && (
                          <ConditionOrderCard order={data.t_suggestion.condition_sell} step={2} />
                        )}
                      </div>
                    </div>
                  ) : (
                    <EmptyTradePanel
                      title="未检测到可执行做T计划"
                      description={data.holding ? "当前信号未达到做T触发条件，先按买入区间和日内位置观察。" : "当前股票未匹配到持仓，暂不生成T仓数量和条件单。"}
                    />
                  )}
                </TabsContent>

                <TabsContent value="buy" className="mt-0">
                  {data.buy_zone ? (
                    <div className="space-y-3">
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                        {data.buy_zone.zones.map((z, i) => (
                          <div key={`${z.method}-${i}`} className="rounded-md border bg-background p-3 text-xs">
                            <div className="mb-2 flex items-center justify-between">
                              <span className="font-medium">{z.method}</span>
                              <Badge variant="outline" className="text-[10px]">
                                安全 {z.safety_score.toFixed(0)}
                              </Badge>
                            </div>
                            <div className="flex justify-between text-muted-foreground font-mono tabular-nums">
                              <span>{z.lower_bound.toFixed(2)}</span>
                              <span>~</span>
                              <span>{z.upper_bound.toFixed(2)}</span>
                            </div>
                            <div className="mt-1 flex justify-between text-muted-foreground">
                              <span>最优 {z.optimal_entry.toFixed(2)}</span>
                              <span>止损 {z.stop_loss.toFixed(2)}</span>
                            </div>
                          </div>
                        ))}
                      </div>
                      {data.buy_zone.verdict && (
                        <p className="text-xs text-muted-foreground">{data.buy_zone.verdict}</p>
                      )}
                    </div>
                  ) : (
                    <EmptyTradePanel title="买入区间暂不可用" description="ATR、支撑位或布林带数据不足，暂不展示价格区间。" />
                  )}
                </TabsContent>

                <TabsContent value="day" className="mt-0">
                  {data.intraday_forecast ? (
                    <div className="space-y-3">
                      <div className="grid grid-cols-1 md:grid-cols-[260px_1fr] gap-4">
                        <div className="grid grid-cols-2 gap-2">
                          <ForecastBox
                            label="预测低"
                            value={data.intraday_forecast.predicted_low}
                            upBand={data.intraday_forecast.predicted_low_up}
                            downBand={data.intraday_forecast.predicted_low_down}
                            sigma={data.intraday_forecast.sigma_low}
                          />
                          <ForecastBox
                            label="预测高"
                            value={data.intraday_forecast.predicted_high}
                            upBand={data.intraday_forecast.predicted_high_up}
                            downBand={data.intraday_forecast.predicted_high_down}
                            sigma={data.intraday_forecast.sigma_high}
                          />
                        </div>
                        <div className="space-y-2 self-center">
                          <div className="flex justify-between text-xs text-muted-foreground">
                            <span>当前位置</span>
                            <span>{data.intraday_forecast.zone_pct.toFixed(0)}%</span>
                          </div>
                          <div className="h-3 rounded-full bg-muted overflow-hidden">
                            <div
                              className="h-full rounded-full bg-primary transition-all"
                              style={{ width: `${Math.max(2, Math.min(100, data.intraday_forecast.zone_pct))}%` }}
                            />
                          </div>
                          <div className="flex justify-between text-[10px] text-muted-foreground">
                            <span>低位</span>
                            <span>{data.intraday_forecast.current_zone === "lower" ? "偏低区域" : data.intraday_forecast.current_zone === "upper" ? "偏高区域" : "中间区域"}</span>
                            <span>高位</span>
                          </div>
                          <div className="grid grid-cols-2 md:grid-cols-4 gap-x-4 gap-y-1 rounded-md bg-muted/40 p-2">
                            <TMetric l="偏向" v={formatBias(data.intraday_forecast.bias)} />
                            <TMetric l="偏向强度" v={formatBiasStrength(data.intraday_forecast.bias_strength, data.intraday_forecast.bias)} />
                            <TMetric l="支撑" v={data.intraday_forecast.support_level ? data.intraday_forecast.support_level.toFixed(2) : "—"} />
                            <TMetric l="压力" v={data.intraday_forecast.resist_level ? data.intraday_forecast.resist_level.toFixed(2) : "—"} />
                            <div className="col-span-2 md:col-span-4">
                              <TMetric l="理由" v={data.intraday_forecast.bias_reason || "—"} />
                            </div>
                          </div>
                        </div>
                      </div>
                      {forecastAccuracyRequested && (
                        <ForecastAccuracyPanel
                          accuracy={forecastAccuracy}
                          loading={forecastAccuracyLoading}
                          error={forecastAccuracyError}
                        />
                      )}
                      <OrderSuggestionPanel
                        fc={data.intraday_forecast}
                        accuracy={forecastAccuracy}
                        isHolding={!!data.holding}
                      />
                    </div>
                  ) : (
                    <EmptyTradePanel title="日内预测暂不可用" description="K线或昨收数据不足，暂不展示日内高低点预测。" />
                  )}
                </TabsContent>

                <TabsContent value="pattern" className="mt-0">
                  <div className="grid grid-cols-1 lg:grid-cols-[260px_1fr] gap-4">
                    {data.short_term_score ? (
                      <div className="rounded-md border bg-background p-3">
                        <div className="flex items-center justify-between">
                          <span className="text-xs text-muted-foreground">短线综合评分</span>
                          <Badge variant="secondary">{data.short_term_score.grade}</Badge>
                        </div>
                        <div className="mt-2 flex items-end gap-2">
                          <span className="font-mono text-3xl font-semibold tabular-nums">{data.short_term_score.score.toFixed(0)}</span>
                          <span className="pb-1 text-sm text-muted-foreground">{data.short_term_score.signal}</span>
                        </div>
                        <p className="mt-2 text-xs text-muted-foreground">{data.short_term_score.verdict}</p>
                        <div className="mt-3 space-y-1">
                          {data.short_term_score.components.map((c) => (
                            <div key={c.name} className="grid grid-cols-[72px_1fr_36px] items-center gap-2 text-xs">
                              <span className="text-muted-foreground">{c.name}</span>
                              <div className="h-1.5 overflow-hidden rounded-full bg-muted">
                                <div className="h-full rounded-full bg-primary" style={{ width: `${Math.max(2, Math.min(100, c.score))}%` }} />
                              </div>
                              <span className="font-mono text-right tabular-nums">{c.score.toFixed(0)}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    ) : (
                      <EmptyTradePanel title="短线评分暂不可用" description="技术、量价或资金数据不足，暂不输出综合评分。" />
                    )}

                    {data.pattern_analysis ? (
                      <div className="space-y-3">
                        <div className="flex flex-wrap items-center gap-2">
                          <Badge variant="outline">{formatBias(data.pattern_analysis.net_bias)}</Badge>
                          <span className="text-xs text-muted-foreground">
                            多 {data.pattern_analysis.bullish_count} / 空 {data.pattern_analysis.bearish_count} / 中性 {data.pattern_analysis.neutral_count}
                          </span>
                          <span className="text-xs text-muted-foreground">评分影响 {data.pattern_analysis.score_impact.toFixed(0)}</span>
                        </div>
                        <p className="text-xs text-muted-foreground">{data.pattern_analysis.verdict}</p>
                        {data.pattern_analysis.signals.length > 0 ? (
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                            {data.pattern_analysis.signals.slice(0, 6).map((p) => (
                              <div key={`${p.category}-${p.pattern}-${p.date}`} className="rounded-md border bg-background p-3">
                                <div className="flex items-center gap-2">
                                  <span className="text-sm font-medium">{p.pattern}</span>
                                  <PatternDirectionBadge direction={p.direction} />
                                  <Badge variant="outline" className="text-[10px] ml-auto">{formatPatternCategory(p.category)}</Badge>
                                </div>
                                <p className="mt-1 text-[11px] text-muted-foreground">{p.description}</p>
                                <div className="mt-2 flex justify-between text-[10px] text-muted-foreground">
                                  <span>{p.date || "—"}</span>
                                  <span>置信度 {(p.confidence * 100).toFixed(0)}%</span>
                                </div>
                              </div>
                            ))}
                          </div>
                        ) : (
                          <EmptyTradePanel title="暂无明确形态" description="最近K线未识别出高置信度K线组合、双底双顶、头肩、三角或旗形。" />
                        )}
                      </div>
                    ) : (
                      <EmptyTradePanel title="形态识别暂不可用" description="K线数量不足，暂不输出形态识别结果。" />
                    )}
                  </div>
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>

          {/* Backtest section */}
          {forecastAccuracyRequested && (
            <BacktestSection
              accuracy={forecastAccuracy}
              loading={forecastAccuracyLoading}
              error={forecastAccuracyError}
              days={accuracyDays}
              onDaysChange={(d) => {
                setAccuracyDays(d);
                if (code) refetchAccuracy(code, d, "", "");
              }}
              onApplyRange={() => {
                if (code) refetchAccuracy(code, accuracyDays, "", "");
              }}
            />
          )}

          {/* Dimension cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <DimensionCard title="量价分析" verdict={data.volume_price?.verdict}>
              <KV l="今日涨跌" v={formatPct(data.volume_price?.today_change_pct ?? 0)} />
              <KV l="成交量" v={formatVolume(data.volume_price?.today_volume ?? 0)} />
              <KV l="5日均量" v={formatVolume(data.volume_price?.avg_volume_5d ?? 0)} />
              <KV l="量比" v={(data.volume_price?.volume_ratio ?? 0).toFixed(2)} />
              <KV l="换手率" v={`${(data.volume_price?.turnover ?? 0).toFixed(2)}%`} />
              <KV l="量价配合" v={data.volume_price?.price_volume_harmony || "—"} />
            </DimensionCard>

            <DimensionCard title="估值分析" verdict={data.valuation?.verdict}>
              <KV l="PE" v={`${(data.valuation?.pe ?? 0).toFixed(1)} (${data.valuation?.pe_level || "—"})`} />
              <KV l="PB" v={`${(data.valuation?.pb ?? 0).toFixed(2)} (${data.valuation?.pb_level || "—"})`} />
              <KV l="总市值" v={formatMoney((data.valuation?.total_mv ?? 0) * 10000)} />
              <KV l="市值级别" v={data.valuation?.mv_level || "—"} />
            </DimensionCard>

            <DimensionCard title="资金流向" verdict={data.money_flow?.verdict}>
              <KV l="主力净流入" v={formatMoney(data.money_flow?.today_main_net ?? 0)} />
              <KV l="主力方向" v={data.money_flow?.today_main_direction || "—"} />
              <KV l="超大单" v={formatMoney(data.money_flow?.today_huge_net ?? 0)} />
              <KV l="大单" v={formatMoney(data.money_flow?.today_big_net ?? 0)} />
              <KV l="连续天数" v={`${data.money_flow?.main_consecutive_days ?? 0}天 ${data.money_flow?.main_consecutive_direction || ""}`} />
              <KV l="散户行为" v={data.money_flow?.retail_behavior || "—"} />
            </DimensionCard>

            <DimensionCard title="技术指标" verdict={data.technical?.verdict}>
              <KV l="均线排列" v={data.technical?.ma_arrangement || "—"} />
              <KV l="MACD" v={data.technical?.macd_signal || "—"} />
              <KV l="MACD趋势" v={data.technical?.macd_hist_trend || "—"} />
              <KV l="KDJ" v={data.technical?.kdj_signal || "—"} />
              <KV l="RSI(14)" v={`${data.technical?.rsi_14?.toFixed(1) ?? "—"} (${data.technical?.rsi_level || "—"})`} />
              <KV l="布林位置" v={data.technical?.boll_position || "—"} />
              <KV l="周期共振" v={data.technical?.period_align || "—"} />
            </DimensionCard>

            <DimensionCard title="板块分析" verdict={data.sector?.verdict}>
              <KV l="所属板块" v={data.sector?.primary_sector || "—"} />
              <KV l="板块龙头" v={data.sector?.is_sector_leader ? "是" : "否"} />
              <KV l="相对强度" v={`${(data.sector?.rel_strength ?? 0).toFixed(2)} (${data.sector?.rel_strength_tag || "—"})`} />
              <KV l="板块5日" v={formatPct(data.sector?.sector_pct_chg_5d ?? 0)} />
            </DimensionCard>

            <DimensionCard title="市场情绪" verdict={data.sentiment?.verdict}>
              <KV l="新闻数" v={`${data.sentiment?.news_count ?? 0}`} />
              <KV l="公告数" v={`${data.sentiment?.announcement_count ?? 0}`} />
              <KV l="情绪评分" v={`${data.sentiment?.sentiment_score ?? 0} (${data.sentiment?.sentiment_label || "—"})`} />
              {data.sentiment?.key_events?.length > 0 && (
                <div className="col-span-2 mt-1">
                  <span className="text-xs text-muted-foreground">关键事件：</span>
                  <span className="text-xs text-muted-foreground">{data.sentiment.key_events.join("；")}</span>
                </div>
              )}
            </DimensionCard>

            <DimensionCard title="基本面" verdict={data.fundamentals?.verdict}>
              <KV l="ROE" v={`${(data.fundamentals?.roe ?? 0).toFixed(1)}% (${data.fundamentals?.roe_level || "—"})`} />
              <KV l="毛利率" v={`${(data.fundamentals?.gross_margin ?? 0).toFixed(1)}%`} />
              <KV l="净利率" v={`${(data.fundamentals?.net_margin ?? 0).toFixed(1)}%`} />
              <KV l="营收增长" v={formatPct(data.fundamentals?.revenue_growth ?? 0)} />
              <KV l="净利增长" v={formatPct(data.fundamentals?.net_profit_growth ?? 0)} />
              <KV l="评分" v={`${data.fundamentals?.score ?? 0}`} />
            </DimensionCard>

            {data.northbound && (
              <DimensionCard title="北向资金" verdict={data.northbound.verdict}>
                <KV l="最新净流" v={formatMoney(data.northbound.latest_net_flow)} />
                <KV l="方向" v={data.northbound.flow_direction || "—"} />
                <KV l="5日趋势" v={data.northbound.trend_5d || "—"} />
                <KV l="信号" v={data.northbound.signal || "—"} />
              </DimensionCard>
            )}

            {data.margin_detail && (
              <DimensionCard title="融资融券" verdict={data.margin_detail.verdict}>
                <KV l="余额趋势" v={data.margin_detail.margin_balance_trend || "—"} />
                <KV l="买入趋势" v={data.margin_detail.margin_buying_trend || "—"} />
                <KV l="信号" v={data.margin_detail.signal || "—"} />
                <KV l="情绪评分" v={`${data.margin_detail.sentiment_score ?? 0}`} />
              </DimensionCard>
            )}

            {data.trend_analysis && (
              <DimensionCard title="趋势分析" verdict={data.trend_analysis.verdict}>
                <KV l="方向" v={data.trend_analysis.trend_stage?.direction || "—"} />
                <KV l="阶段" v={data.trend_analysis.trend_stage?.stage || "—"} />
                <KV l="强度" v={data.trend_analysis.trend_stage?.strength || "—"} />
                <KV l="价格位置" v={data.trend_analysis.support_resistance?.price_position || "—"} />
                <KV l="最近支撑" v={formatPrice(data.trend_analysis.support_resistance?.support1 ?? 0)} />
                <KV l="最近阻力" v={formatPrice(data.trend_analysis.support_resistance?.resistance1 ?? 0)} />
              </DimensionCard>
            )}
          </div>

          {/* Score history */}
          {history && history.history?.length > 0 && (
            <Card>
              <CardHeader className="pb-2 pt-4 px-4">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  评分历史
                </CardTitle>
              </CardHeader>
              <CardContent className="px-4 pb-4">
                {history.trend && (
                  <div className="flex flex-wrap gap-4 mb-3 text-sm">
                    <span>
                      当前 <span className="font-mono tabular-nums">{history.trend.current?.toFixed(0)}</span>
                    </span>
                    <span>
                      7日变化 <span className="font-mono tabular-nums">{formatPct(history.trend.change_7d ?? 0)}</span>
                    </span>
                    <span>
                      30日变化 <span className="font-mono tabular-nums">{formatPct(history.trend.change_30d ?? 0)}</span>
                    </span>
                    <Badge variant="outline">
                      {history.trend.trend_7d === "rising" ? "7日上升"
                        : history.trend.trend_7d === "falling" ? "7日下降"
                        : "7日平稳"}
                    </Badge>
                  </div>
                )}
                <div className="space-y-1">
                  {history.history.slice(0, 10).map((entry, i) => (
                    <div key={i} className="flex items-center gap-4 text-xs text-muted-foreground">
                      <span className="w-24 shrink-0">
                        {new Date(entry.recorded_at).toLocaleDateString("zh-CN")}
                      </span>
                      <Badge variant="secondary" className="tabular-nums">{entry.score.toFixed(0)}</Badge>
                      <div className="flex gap-3 flex-wrap">
                        {Object.entries(entry.dimensions || {}).map(([dim, val], di) => (
                          <span key={`${dim}-${di}`} className="font-mono tabular-nums">
                            {DIM_LABELS[dim] || dim} {val.toFixed(0)}
                          </span>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Deep analysis */}
          <Card>
            <CardHeader className="pb-2 pt-4 px-4">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-1.5">
                  <Brain className="h-3.5 w-3.5" />
                  AI 深度分析
                </CardTitle>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={startDeep}
                  disabled={deepLoading || !code}
                >
                  {deepLoading ? (
                    <><Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />分析中...</>
                  ) : (
                    "开始分析"
                  )}
                </Button>
              </div>
            </CardHeader>
            <CardContent className="px-4 pb-4">
              {deepStatus?.status === "running" && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  {deepStatus.pct_done || "AI 正在分析..."}
                </div>
              )}
              {deepStatus?.status === "failed" && (
                <div className="text-sm text-destructive">{deepStatus.error || "分析失败"}</div>
              )}
              {deepStatus?.status === "not_found" && (
                <div className="text-sm text-muted-foreground">暂无深度分析报告</div>
              )}
              {deepStatus?.report && (
                <div className="prose prose-sm prose-invert max-w-none text-sm text-muted-foreground whitespace-pre-wrap">
                  {deepStatus.report}
                </div>
              )}
              {!deepStatus && !deepLoading && (
                <p className="text-sm text-muted-foreground">点击"开始分析"生成 AI 深度研究报告</p>
              )}
            </CardContent>
          </Card>

          {/* Data sources */}
          {data.data_sources && Object.keys(data.data_sources).length > 0 && (
            <div className="flex flex-wrap gap-2 text-[10px] text-muted-foreground">
              {Object.entries(data.data_sources).map(([k, v]) => (
                <span key={k}>{k}: {v}</span>
              ))}
            </div>
          )}
        </>
      )}

      {/* Empty state — show search history */}
      {!loading && !data && !error && (
        <div className="space-y-4">
          {searchHistory.length > 0 ? (
            <Card>
              <CardHeader className="flex flex-row items-center justify-between py-3 px-4">
                <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-1.5">
                  <Clock className="h-3.5 w-3.5" />
                  搜索历史
                </CardTitle>
                <Button variant="ghost" size="sm" onClick={handleClearHistory} className="text-xs text-muted-foreground">
                  <X className="h-3 w-3 mr-1" />
                  清空
                </Button>
              </CardHeader>
              <CardContent className="px-4 pb-4 pt-0">
                <div className="flex flex-wrap gap-2">
                  {searchHistory.map((h, i) => (
                    <button
                      key={`${h.code}-${i}`}
                      onClick={() => selectFromHistory(h)}
                      className="inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm transition-colors hover:bg-accent"
                    >
                      <span className="font-medium">{h.name}</span>
                      <span className="font-mono text-xs text-muted-foreground">{h.code}</span>
                    </button>
                  ))}
                </div>
              </CardContent>
            </Card>
          ) : (
            <div className="flex flex-col items-center gap-2 py-20 text-muted-foreground">
              <Activity className="h-10 w-10" />
              <p>输入股票代码或名称开始分析</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ---- Sub-components ----

function DimensionCard({
  title, verdict, children,
}: {
  title: string; verdict?: string; children: React.ReactNode;
}) {
  const [open, setOpen] = useState(false);
  return (
    <Card>
      <CardHeader
        className="flex flex-row items-center justify-between py-3 px-4 cursor-pointer"
        onClick={() => setOpen(!open)}
      >
        <div className="flex items-center gap-2">
          {open ? <ChevronDown className="h-4 w-4 text-muted-foreground" />
            : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
          <CardTitle className="text-sm font-medium">{title}</CardTitle>
        </div>
        {verdict && <Badge variant="outline" className="text-xs">{verdict}</Badge>}
      </CardHeader>
      {open && (
        <CardContent className="px-4 pb-4 pt-0">
          <div className="grid grid-cols-2 gap-x-4 gap-y-1">
            {children}
          </div>
        </CardContent>
      )}
    </Card>
  );
}

function KV({ l, v }: { l: string; v: string }) {
  return (
    <>
      <span className="text-xs text-muted-foreground">{l}</span>
      <span className="text-xs font-mono tabular-nums text-right">{v}</span>
    </>
  );
}

function TMetric({ l, v }: { l: string; v: string }) {
  return (
    <div className="flex items-center justify-between gap-2 text-xs min-w-0">
      <span className="text-muted-foreground shrink-0">{l}</span>
      <span className="font-mono tabular-nums text-right truncate">{v}</span>
    </div>
  );
}

// ForecastBox renders a predicted high or low with its ±1σ confidence band.
// Visual conventions:
//   - central value large
//   - ±σ shown as small subtext (e.g., "±0.45")
//   - full range shown as a tiny muted line (e.g., "49.05 — 49.95")
//   - sigma hidden when zero / undefined
function ForecastBox({
  label,
  value,
  upBand,
  downBand,
  sigma,
}: {
  label: string;
  value: number;
  upBand?: number;
  downBand?: number;
  sigma?: number;
}) {
  const hasSigma = sigma !== undefined && sigma > 0;
  const sigmaStr = hasSigma ? `±${sigma.toFixed(2)}` : "";
  const rangeStr =
    upBand !== undefined && downBand !== undefined
      ? `${downBand.toFixed(2)} — ${upBand.toFixed(2)}`
      : "";

  return (
    <div className="rounded-md border bg-background p-3">
      <div className="flex items-baseline justify-between">
        <span className="text-xs text-muted-foreground">{label}</span>
        {hasSigma && (
          <span className="text-[10px] text-muted-foreground" title="预测不确定性 (1σ)">
            σ {sigmaStr}
          </span>
        )}
      </div>
      <div className="mt-1 font-mono text-lg tabular-nums">{value.toFixed(2)}</div>
      {rangeStr && (
        <div className="mt-0.5 text-[10px] text-muted-foreground tabular-nums" title="±1σ 置信区间">
          {rangeStr}
        </div>
      )}
    </div>
  );
}

// formatBiasStrength turns the bias_strength number into a label that reflects
// the confidence-weighted sum of patterns driving the bias.
//   < 0.5 : "弱" (single low-confidence pattern)
//   < 1.0 : "中"
//   < 1.5 : "强"
//   ≥ 1.5 : "极强 (共振)"
function formatBiasStrength(strength: number | undefined, bias: string | undefined): string {
  if (strength === undefined || strength <= 0 || bias === "neutral" || bias === undefined) {
    return "—";
  }
  let tier: string;
  switch (true) {
    case strength >= 1.5:
      tier = "极强";
      break;
    case strength >= 1.0:
      tier = "强";
      break;
    case strength >= 0.5:
      tier = "中";
      break;
    default:
      tier = "弱";
  }
  return `${tier} (${strength.toFixed(2)})`;
}

function EmptyTradePanel({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-md border border-dashed bg-muted/20 p-4">
      <div className="text-sm font-medium">{title}</div>
      <p className="mt-1 text-xs text-muted-foreground">{description}</p>
    </div>
  );
}

// ──────────────────────────────────────────────
// Intraday-forecast historical accuracy panel
// ──────────────────────────────────────────────
// Shows empirical hit rates for the day-range forecast so the user can decide
// whether to trust today's predicted_high / predicted_low for sell timing.
// Backtest comes from /api/analyze/intraday-forecast-accuracy (Go side:
// services.BacktestIntradayForecast, no look-ahead).

const ACCURACY_DAYS_WINDOW = 30; // how many recent days to render in the heatmap

function reliabilityMeta(r: string): { label: string; className: string } {
  switch (r) {
    case "high_confidence":
      return { label: "高可信", className: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300" };
    case "moderate":
      return { label: "中等", className: "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300" };
    default:
      return { label: "低可信", className: "bg-rose-100 text-rose-700 dark:bg-rose-900/40 dark:text-rose-300" };
  }
}

function hitStatusColor(highHit: boolean, lowHit: boolean, bothWide: boolean): string {
  // Priority: both central hit → green; wide-only hit → amber; miss → rose.
  if (highHit && lowHit) return "bg-emerald-500";
  if (bothWide) return "bg-amber-400";
  return "bg-rose-500";
}

function AccuracyHeatmap({ details }: { details: NonNullable<IntradayForecastAccuracy["details"]> }) {
  // Show last ACCURACY_DAYS_WINDOW days, oldest → newest, left to right.
  const recent = details.slice(-ACCURACY_DAYS_WINDOW);
  if (recent.length === 0) return null;

  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-1">
        <span className="w-8 text-[10px] text-muted-foreground">高</span>
        <div className="flex flex-1 gap-[2px]">
          {recent.map((d, i) => (
            <div
              key={`h-${i}-${d.date}`}
              className={`h-3 flex-1 rounded-[2px] ${hitStatusColor(d.high_in_range, true, d.high_in_wide_range)}`}
              title={`${d.date} · 预测高 ${d.predicted_high.toFixed(2)} · 实际高 ${d.actual_high.toFixed(2)} · ${d.high_in_range ? "命中" : d.high_in_wide_range ? "宽区间命中" : "偏离"}`}
            />
          ))}
        </div>
      </div>
      <div className="flex items-center gap-1">
        <span className="w-8 text-[10px] text-muted-foreground">低</span>
        <div className="flex flex-1 gap-[2px]">
          {recent.map((d, i) => (
            <div
              key={`l-${i}-${d.date}`}
              className={`h-3 flex-1 rounded-[2px] ${hitStatusColor(true, d.low_in_range, d.low_in_wide_range)}`}
              title={`${d.date} · 预测低 ${d.predicted_low.toFixed(2)} · 实际低 ${d.actual_low.toFixed(2)} · ${d.low_in_range ? "命中" : d.low_in_wide_range ? "宽区间命中" : "偏离"}`}
            />
          ))}
        </div>
      </div>
      <div className="flex items-center justify-between text-[10px] text-muted-foreground pt-0.5">
        <span>{recent[0].date}</span>
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-1"><span className="h-2 w-2 rounded-sm bg-emerald-500" />命中</span>
          <span className="flex items-center gap-1"><span className="h-2 w-2 rounded-sm bg-amber-400" />宽区间</span>
          <span className="flex items-center gap-1"><span className="h-2 w-2 rounded-sm bg-rose-500" />偏离</span>
        </div>
        <span>{recent[recent.length - 1].date}</span>
      </div>
    </div>
  );
}

function ForecastAccuracyPanel({
  accuracy,
  loading,
  error,
}: {
  accuracy: IntradayForecastAccuracy | null;
  loading: boolean;
  error: string;
}) {
  const [open, setOpen] = useState(false);

  if (loading) {
    return (
      <div className="mt-3 rounded-md border bg-muted/20 p-3">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="h-3 w-3 animate-spin" />
          正在回测日内预测历史命中率…
        </div>
      </div>
    );
  }
  if (error) {
    return (
      <div className="mt-3 rounded-md border border-dashed bg-muted/20 p-3 text-xs text-muted-foreground">
        历史命中率暂不可用：{error}
      </div>
    );
  }
  if (!accuracy) return null;

  const meta = reliabilityMeta(accuracy.reliability);
  const details = accuracy.details ?? [];
  const lowReliability = accuracy.reliability === "low_confidence";

  return (
    <div className="mt-3 rounded-md border bg-background">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left"
      >
        <div className="flex items-center gap-2">
          {open ? <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" /> : <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />}
          <span className="text-xs font-medium">历史命中率</span>
          <Badge variant="outline" className={`text-[10px] border-0 ${meta.className}`}>{meta.label}</Badge>
          <span className="text-[10px] text-muted-foreground">· {accuracy.days_evaluated} 个交易日</span>
        </div>
        <span className="text-[10px] text-muted-foreground font-mono tabular-nums">
          中央 {(accuracy.both_in_range_pct * 100).toFixed(0)}% · 宽区间 {(accuracy.both_in_wide_range_pct * 100).toFixed(0)}%
        </span>
      </button>

      {open && (
        <div className="space-y-3 border-t px-3 py-3">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-x-4 gap-y-1 rounded-md bg-muted/40 p-2">
            <TMetric l="中央命中率" v={`${(accuracy.both_in_range_pct * 100).toFixed(1)}%`} />
            <TMetric l="宽区间命中率" v={`${(accuracy.both_in_wide_range_pct * 100).toFixed(1)}%`} />
            <TMetric l="平均 σ" v={accuracy.avg_sigma > 0 ? accuracy.avg_sigma.toFixed(2) : "—"} />
            <TMetric l="平均宽度" v={accuracy.avg_range_width_pct > 0 ? `${(accuracy.avg_range_width_pct * 100).toFixed(2)}%` : "—"} />
            <TMetric l="高价命中" v={`${(accuracy.high_in_range_pct * 100).toFixed(1)}%`} />
            <TMetric l="低价命中" v={`${(accuracy.low_in_range_pct * 100).toFixed(1)}%`} />
            <TMetric
              l="高价偏离"
              v={accuracy.avg_high_miss_pct !== 0 ? `${(accuracy.avg_high_miss_pct * 100).toFixed(2)}%` : "—"}
            />
            <TMetric
              l="低价偏离"
              v={accuracy.avg_low_miss_pct !== 0 ? `${(accuracy.avg_low_miss_pct * 100).toFixed(2)}%` : "—"}
            />
          </div>

          {details.length > 0 && (
            <div>
              <div className="mb-1.5 text-[10px] text-muted-foreground">
                近 {Math.min(ACCURACY_DAYS_WINDOW, details.length)} 天命中分布
              </div>
              <AccuracyHeatmap details={details} />
            </div>
          )}

          {lowReliability && (
            <div className="rounded-md border border-rose-300/60 bg-rose-50 dark:border-rose-800/60 dark:bg-rose-950/30 p-2">
              <p className="text-[11px] text-rose-700 dark:text-rose-300">
                ⚠ 此预测历史中央命中率低于 55%，请勿单独用于决策。建议结合其他信号（形态、买卖区间、趋势）综合判断。
              </p>
            </div>
          )}
          {!lowReliability && accuracy.reliability === "moderate" && (
            <p className="text-[11px] text-muted-foreground">
              提示：中等可信度下日内预测可作为参考，但不建议作为唯一卖出触发条件。
            </p>
          )}
          {accuracy.reliability === "high_confidence" && (
            <p className="text-[11px] text-emerald-700 dark:text-emerald-300">
              ✓ 历史命中率 ≥ 70%，日内预测可信度较高，可作为日内卖出时点的辅助依据。
            </p>
          )}
        </div>
      )}
    </div>
  );
}

// ──────────────────────────────────────────────
// Today's order-suggestion panel
// ──────────────────────────────────────────────
// Maps the empirical-quantile forecast into concrete price levels with
// fill-probability annotations and a bias-aware recommendation, so the user
// can decide where to place today's condition orders without doing the
// percentile math in their head.
//
// Fill-probability semantics:
//   "成交概率 X%" = historically, X% of trading days saw the price reach this
//   level intraday. So a sell limit at P83 (17% fill rate) only fills ~17%
//   of the time, but at the top of the predicted range. A sell at P50 (50%)
//   fills half the time at a mid-range price.
//
// Levels come from the backend's empirical excursion distribution:
//   P50  → predicted_high_median / predicted_low_median  (~50% fill)
//   P83  → predicted_high / predicted_low                (~17% fill, ceiling/floor)
//   P95  → predicted_high_up / predicted_low_down        (~5% fill, spike)

type SellLevel = { price: number; fillPct: number; tag: string; useCase: string; empFillPct?: number; empPnlPct?: number; empFills?: number; empWinRate?: number; empCumPnl?: number };
type BuyLevel = { price: number; fillPct: number; tag: string; useCase: string; empFillPct?: number; empPnlPct?: number; empFills?: number; empWinRate?: number; empCumPnl?: number };

function enrichWithStats<T extends { tag: string }>(
  levels: T[],
  orderStats: OrderLevelStat[] | undefined,
  side: "sell" | "buy",
): T[] {
  if (!orderStats) return levels;
  return levels.map((lv) => {
    const stat = orderStats.find((s) => s.side === side && s.tag === lv.tag);
    if (!stat) return lv;
    return {
      ...lv,
      empFillPct: stat.empirical_fill_pct,
      empPnlPct: stat.avg_pnl_pct,
      empFills: stat.fills,
      empWinRate: stat.win_rate,
      empCumPnl: stat.cumulative_pnl_pct,
    };
  });
}

function buildSellLevels(fc: NonNullable<StockAnalysis["intraday_forecast"]>): SellLevel[] {
  const levels: SellLevel[] = [];
  if (fc.predicted_high_median && fc.predicted_high_median > 0) {
    levels.push({
      price: fc.predicted_high_median,
      fillPct: 50,
      tag: "中位",
      useCase: "稳健成交 · 半数日子触达，价格一般",
    });
  }
  if (fc.predicted_high > 0) {
    levels.push({
      price: fc.predicted_high,
      fillPct: 17,
      tag: "预测高",
      useCase: "优质成交 · 日内冲高时成交",
    });
  }
  if (fc.predicted_high_up && fc.predicted_high_up > 0) {
    levels.push({
      price: fc.predicted_high_up,
      fillPct: 5,
      tag: "上限",
      useCase: "极端冲高 · 涨停边缘/异常波动",
    });
  }
  // Sort descending by price (highest first).
  return levels.sort((a, b) => b.price - a.price);
}

function buildBuyLevels(fc: NonNullable<StockAnalysis["intraday_forecast"]>): BuyLevel[] {
  const levels: BuyLevel[] = [];
  if (fc.predicted_low_median && fc.predicted_low_median > 0) {
    levels.push({
      price: fc.predicted_low_median,
      fillPct: 50,
      tag: "中位",
      useCase: "稳健成交 · 半数日子触达，价格一般",
    });
  }
  if (fc.predicted_low > 0) {
    levels.push({
      price: fc.predicted_low,
      fillPct: 17,
      tag: "预测低",
      useCase: "优质抄底 · 日内探底时成交",
    });
  }
  if (fc.predicted_low_down && fc.predicted_low_down > 0) {
    levels.push({
      price: fc.predicted_low_down,
      fillPct: 5,
      tag: "下限",
      useCase: "暴跌接刀 · 异常下探/利空",
    });
  }
  // Sort ascending by price (lowest first).
  return levels.sort((a, b) => a.price - b.price);
}

function biasRecommendation(
  bias: string | undefined,
  biasStrength: number | undefined,
  zonePct: number,
  reliability: "high_confidence" | "moderate" | "low_confidence" | undefined,
  isHolding: boolean,
): { primary: string; secondary?: string; tone: "info" | "warn" | "danger" } {
  const strong = (biasStrength ?? 0) >= 1.0;
  const inUpper = zonePct >= 70;
  const inLower = zonePct <= 30;
  const lowRel = reliability === "low_confidence";

  if (lowRel) {
    return {
      primary: "历史命中率偏低，不建议基于此预测单独挂条件单。",
      secondary: "若需交易请结合其他信号（形态评分、买卖区间、趋势），并严格设止损。",
      tone: "danger",
    };
  }

  if (isHolding) {
    if (bias === "bullish" || inUpper) {
      const reason = inUpper ? "价格已进入预测高位区" : "形态偏多";
      return {
        primary: `${reason}，优先挂卖。建议 50% 仓位挂「预测高」，50% 挂「上限」等冲高。`,
        secondary: strong ? "偏向强烈，可适当加大挂单比例。" : undefined,
        tone: "info",
      };
    }
    if (bias === "bearish") {
      return {
        primary: "形态偏空，建议挂卖但价格不宜过高。建议挂「中位」或「预测高」先锁住成交。",
        secondary: "若不想立即减仓，至少设止损在「下限」下方。",
        tone: "warn",
      };
    }
    return {
      primary: "无明显偏向，建议分批挂卖：中位 + 预测高 各 50%。",
      tone: "info",
    };
  }

  // Not holding
  if (bias === "bullish") {
    if (inLower) {
      return {
        primary: "形态偏多且价格在预测低位区，是入场点。可挂「预测低」买入，止损「下限」下方。",
        tone: "info",
      };
    }
    return {
      primary: "形态偏多但价格已离开低位区。可挂「预测低」或「中位」等回踩，止损「下限」下方。",
      tone: "info",
    };
  }
  if (bias === "bearish") {
    return {
      primary: "形态偏空，不建议新仓买入。若必须建仓，挂「下限」接刀且严格止损。",
      tone: "warn",
    };
  }
  return {
    primary: "无明显偏向，可挂「预测低」试单，止损「下限」下方 1%。",
    tone: "info",
  };
}

function OrderSuggestionPanel({
  fc,
  accuracy,
  isHolding,
}: {
  fc: NonNullable<StockAnalysis["intraday_forecast"]>;
  accuracy: IntradayForecastAccuracy | null;
  isHolding: boolean;
}) {
  const [open, setOpen] = useState(true);
  const rawSells = buildSellLevels(fc);
  const rawBuys = buildBuyLevels(fc);
  // Enrich with empirical stats from the backtest.
  const sells = enrichWithStats(rawSells, accuracy?.order_stats, "sell");
  const buys = enrichWithStats(rawBuys, accuracy?.order_stats, "buy");
  const reliability = accuracy?.reliability;
  const rec = biasRecommendation(fc.bias, fc.bias_strength, fc.zone_pct, reliability, isHolding);

  const toneClasses: Record<typeof rec.tone, string> = {
    info: "border-sky-300/60 bg-sky-50 dark:border-sky-800/60 dark:bg-sky-950/30 text-sky-800 dark:text-sky-200",
    warn: "border-amber-300/60 bg-amber-50 dark:border-amber-800/60 dark:bg-amber-950/30 text-amber-800 dark:text-amber-200",
    danger: "border-rose-300/60 bg-rose-50 dark:border-rose-800/60 dark:bg-rose-950/30 text-rose-800 dark:text-rose-200",
  };

  return (
    <div className="mt-3 rounded-md border bg-background">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left"
      >
        <div className="flex items-center gap-2">
          {open ? <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" /> : <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />}
          <span className="text-xs font-medium">今日挂单参考</span>
          <span className="text-[10px] text-muted-foreground">· 6 档价位 + 历史回测</span>
        </div>
        <span className="text-[10px] text-muted-foreground">点击{open ? "收起" : "展开"}</span>
      </button>

      {open && (
        <div className="space-y-3 border-t px-3 py-3">
          {/* Bias-aware recommendation banner */}
          <div className={`rounded-md border p-2 text-[11px] leading-relaxed ${toneClasses[rec.tone]}`}>
            <div className="font-medium">{rec.primary}</div>
            {rec.secondary && <div className="mt-0.5 opacity-90">{rec.secondary}</div>}
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {/* Sell levels */}
            <div className="rounded-md border bg-background">
              <div className="flex items-center justify-between border-b px-2 py-1.5">
                <span className="text-[11px] font-medium">卖出挂单</span>
                <span className="text-[10px] text-muted-foreground">高 → 低</span>
              </div>
              <div className="divide-y">
                {sells.map((lv, i) => {
                  const fillBar = Math.min(100, lv.fillPct * 2);
                  const empPct = lv.empFillPct !== undefined ? (lv.empFillPct * 100).toFixed(0) : null;
                  const pnlPct = lv.empPnlPct !== undefined ? (lv.empPnlPct * 100).toFixed(2) : null;
                  const pnlPos = lv.empPnlPct !== undefined && lv.empPnlPct > 0;
                  return (
                    <div key={`s-${i}`} className="px-2 py-2">
                      <div className="flex items-baseline justify-between">
                        <span className="font-mono text-base tabular-nums">{lv.price.toFixed(2)}</span>
                        <span className="text-[10px] text-muted-foreground">{lv.tag}</span>
                      </div>
                      <div className="mt-1 flex items-center gap-2">
                        <div className="h-1 flex-1 overflow-hidden rounded-full bg-muted">
                          <div className="h-full rounded-full bg-primary" style={{ width: `${fillBar}%` }} />
                        </div>
                        <span className="text-[10px] font-mono tabular-nums text-right" style={{ minWidth: "5.5rem" }}>
                          理论 {lv.fillPct}%
                          {empPct !== null && <span className="text-muted-foreground ml-1">实际 {empPct}%</span>}
                        </span>
                      </div>
                      <div className="mt-0.5 text-[10px] text-muted-foreground">{lv.useCase}</div>
                      {(pnlPct !== null || lv.empCumPnl !== undefined) && (
                        <div className="mt-1 grid grid-cols-3 gap-1 text-[10px] rounded bg-muted/40 px-1.5 py-1">
                          <div>
                            <div className="text-muted-foreground">成交日</div>
                            <div className={pnlPos ? "text-emerald-700 dark:text-emerald-300 font-mono tabular-nums" : "text-rose-700 dark:text-rose-300 font-mono tabular-nums"}>
                              {pnlPct !== null ? `${pnlPos ? "+" : ""}${pnlPct}%` : "—"}
                            </div>
                          </div>
                          <div>
                            <div className="text-muted-foreground">胜率</div>
                            <div className="font-mono tabular-nums">
                              {lv.empWinRate !== undefined ? `${(lv.empWinRate * 100).toFixed(0)}%` : "—"}
                            </div>
                          </div>
                          <div>
                            <div className="text-muted-foreground">累计</div>
                            <div className={(lv.empCumPnl ?? 0) >= 0 ? "text-emerald-700 dark:text-emerald-300 font-mono tabular-nums" : "text-rose-700 dark:text-rose-300 font-mono tabular-nums"}>
                              {lv.empCumPnl !== undefined
                                ? `${lv.empCumPnl >= 0 ? "+" : ""}${(lv.empCumPnl * 100).toFixed(2)}%`
                                : "—"}
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Buy levels */}
            <div className="rounded-md border bg-background">
              <div className="flex items-center justify-between border-b px-2 py-1.5">
                <span className="text-[11px] font-medium">买入挂单</span>
                <span className="text-[10px] text-muted-foreground">低 → 高</span>
              </div>
              <div className="divide-y">
                {buys.map((lv, i) => {
                  const fillBar = Math.min(100, lv.fillPct * 2);
                  const empPct = lv.empFillPct !== undefined ? (lv.empFillPct * 100).toFixed(0) : null;
                  const pnlPct = lv.empPnlPct !== undefined ? (lv.empPnlPct * 100).toFixed(2) : null;
                  const pnlPos = lv.empPnlPct !== undefined && lv.empPnlPct > 0;
                  const stopLoss = lv.tag === "预测低" && fc.predicted_low_down && fc.predicted_low_down > 0
                    ? (fc.predicted_low_down * 0.99).toFixed(2)
                    : null;
                  return (
                    <div key={`b-${i}`} className="px-2 py-2">
                      <div className="flex items-baseline justify-between">
                        <span className="font-mono text-base tabular-nums">{lv.price.toFixed(2)}</span>
                        <span className="text-[10px] text-muted-foreground">{lv.tag}</span>
                      </div>
                      <div className="mt-1 flex items-center gap-2">
                        <div className="h-1 flex-1 overflow-hidden rounded-full bg-muted">
                          <div className="h-full rounded-full bg-primary" style={{ width: `${fillBar}%` }} />
                        </div>
                        <span className="text-[10px] font-mono tabular-nums text-right" style={{ minWidth: "5.5rem" }}>
                          理论 {lv.fillPct}%
                          {empPct !== null && <span className="text-muted-foreground ml-1">实际 {empPct}%</span>}
                        </span>
                      </div>
                      <div className="mt-0.5 text-[10px] text-muted-foreground">
                        {lv.useCase}
                        {stopLoss && (
                          <span className="ml-1 text-rose-700 dark:text-rose-300">· 止损 {stopLoss}</span>
                        )}
                      </div>
                      {(pnlPct !== null || lv.empCumPnl !== undefined) && (
                        <div className="mt-1 grid grid-cols-3 gap-1 text-[10px] rounded bg-muted/40 px-1.5 py-1">
                          <div>
                            <div className="text-muted-foreground">成交日</div>
                            <div className={pnlPos ? "text-emerald-700 dark:text-emerald-300 font-mono tabular-nums" : "text-rose-700 dark:text-rose-300 font-mono tabular-nums"}>
                              {pnlPct !== null ? `${pnlPos ? "+" : ""}${pnlPct}%` : "—"}
                            </div>
                          </div>
                          <div>
                            <div className="text-muted-foreground">胜率</div>
                            <div className="font-mono tabular-nums">
                              {lv.empWinRate !== undefined ? `${(lv.empWinRate * 100).toFixed(0)}%` : "—"}
                            </div>
                          </div>
                          <div>
                            <div className="text-muted-foreground">累计</div>
                            <div className={(lv.empCumPnl ?? 0) >= 0 ? "text-emerald-700 dark:text-emerald-300 font-mono tabular-nums" : "text-rose-700 dark:text-rose-300 font-mono tabular-nums"}>
                              {lv.empCumPnl !== undefined
                                ? `${lv.empCumPnl >= 0 ? "+" : ""}${(lv.empCumPnl * 100).toFixed(2)}%`
                                : "—"}
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          </div>

          <div className="text-[10px] text-muted-foreground leading-relaxed">
            理论 = 分位数构建概率；实际 = 近期回测中真实触达比例。
            「成交日」= 触达日的平均相对盈亏（卖出对比昨收，买入对比今收），正值代表挂单价比不挂单更好。
            预测每日盘前更新，盘中不要重算；任何挂单都建议设止损。
          </div>
        </div>
      )}
    </div>
  );
}

function formatBias(bias?: string) {
  switch (bias) {
    case "bullish":
      return "偏多";
    case "bearish":
      return "偏空";
    case "mixed":
      return "多空混合";
    case "neutral":
      return "中性";
    default:
      return "—";
  }
}

function formatPatternCategory(category: string) {
  switch (category) {
    case "kline":
      return "K线";
    case "chart":
      return "图形";
    case "volume":
      return "量价";
    default:
      return category || "—";
  }
}

function PatternDirectionBadge({ direction }: { direction: string }) {
  if (direction === "bullish") {
    return <Badge variant="secondary" className="text-[10px]">看多</Badge>;
  }
  if (direction === "bearish") {
    return <Badge variant="destructive" className="text-[10px]">看空</Badge>;
  }
  return <Badge variant="outline" className="text-[10px]">中性</Badge>;
}

function ConditionOrderCard({ order, step }: { order: { direction: string; trigger_price: number; trigger_desc: string; order_price: number; order_type: string; quantity_ratio: string; stop_price: number; stop_desc: string; note: string }; step: number }) {
  const isPending = order.note === "待触发";
  return (
    <div className={`rounded-md border p-3 space-y-1.5 ${isPending ? "opacity-70" : ""}`}>
      <div className="flex items-center gap-2">
        <Badge variant="secondary" className="text-[10px]">步骤{step}</Badge>
        <Badge variant="outline">{order.direction}</Badge>
        <span className="text-[10px] text-muted-foreground ml-auto text-right leading-tight">
          {order.order_type} · {order.quantity_ratio}
          {isPending && <span className="ml-1 opacity-60">（待触发）</span>}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-x-4 gap-y-0.5">
        <KV l="委托价" v={order.order_price.toFixed(2)} />
        <KV l="触发价" v={order.trigger_price.toFixed(2)} />
      </div>
      <p className="text-[11px] text-muted-foreground">{order.trigger_desc}</p>
      <div className="flex flex-wrap items-center gap-1 text-[11px]">
        <span className="text-muted-foreground">止损:</span>
        <span className="font-mono tabular-nums">{order.stop_price.toFixed(2)}</span>
        <span className="text-muted-foreground ml-2">{order.stop_desc}</span>
      </div>
      {order.note && order.note !== "待触发" && (
        <p className="text-[10px] text-muted-foreground italic">{order.note}</p>
      )}
    </div>
  );
}
