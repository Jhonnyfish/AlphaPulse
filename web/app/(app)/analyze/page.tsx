"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { analysisApi, searchApi } from "@/lib/api-client";
import type { StockAnalysis, DeepAnalysisResponse, ScoreHistoryResponse, SearchSuggestion } from "@/lib/types";
import { DIM_LABELS, formatPct, formatPrice, formatVolume, formatMoney } from "@/lib/constants";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
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

  // Search suggestions & search history
  const [suggestions, setSuggestions] = useState<SearchSuggestion[]>([]);
  const [showSugg, setShowSugg] = useState(false);
  const [searchHistory, setSearchHistory] = useState<HistoryItem[]>([]);
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Load search history on mount
  useEffect(() => {
    setSearchHistory(loadHistory());
  }, []);

  const doAnalyze = useCallback(async (c: string) => {
    if (!c) return;
    setLoading(true);
    setError("");
    setData(null);
    setHistory(null);
    setDeepStatus(null);
    try {
      const res = await analysisApi.analyze(c);
      setData(res);
      saveHistory(c, res.name);
      setSearchHistory(loadHistory());
      analysisApi.scoreHistory(c).then(setHistory).catch(() => {});
    } catch (err: any) {
      setError(err.message || "分析失败");
    } finally {
      setLoading(false);
    }
  }, []);

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

          {/* Trading signals: buy zone + T-suggestion + intraday forecast */}
          {(data.buy_zone || data.t_suggestion || data.intraday_forecast) && (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {/* Buy Zone */}
              {data.buy_zone && (
                <Card>
                  <CardHeader className="pb-2 pt-4 px-4">
                    <CardTitle className="text-sm font-medium text-muted-foreground">买入区间</CardTitle>
                  </CardHeader>
                  <CardContent className="px-4 pb-4">
                    <div className="space-y-2">
                      {data.buy_zone.zones.map((z, i) => (
                        <div key={`${z.method}-${i}`} className="flex flex-col gap-0.5 text-xs">
                          <div className="flex items-center justify-between">
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
                          <div className="flex justify-between text-muted-foreground">
                            <span>最优 {z.optimal_entry.toFixed(2)}</span>
                            <span>止损 {z.stop_loss.toFixed(2)}</span>
                          </div>
                        </div>
                      ))}
                      {data.buy_zone.verdict && (
                        <p className="text-xs text-muted-foreground pt-1">{data.buy_zone.verdict}</p>
                      )}
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* T-suggestion */}
              {data.t_suggestion && (
                <Card className="md:col-span-2">
                  <CardHeader className="pb-2 pt-4 px-4">
                    <div className="flex items-center gap-2">
                      <CardTitle className="text-sm font-medium text-muted-foreground">做T建议</CardTitle>
                      <Badge variant="outline">{data.t_suggestion.type}</Badge>
                      <span className="text-xs text-muted-foreground">{data.t_suggestion.action}</span>
                      {data.t_suggestion.confidence > 0 && (
                        <Badge variant="secondary" className="text-[10px] ml-auto">
                          置信度 {data.t_suggestion.confidence.toFixed(0)}%
                        </Badge>
                      )}
                    </div>
                  </CardHeader>
                  <CardContent className="px-4 pb-4">
                    <div className="space-y-3">
                      {data.t_suggestion.reason && (
                        <p className="text-xs text-muted-foreground">{data.t_suggestion.reason}</p>
                      )}
                      {/* Condition orders */}
                      {data.t_suggestion.condition_buy && (
                        <ConditionOrderCard order={data.t_suggestion.condition_buy} step={1} />
                      )}
                      {data.t_suggestion.condition_sell && (
                        <ConditionOrderCard order={data.t_suggestion.condition_sell} step={2} />
                      )}
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* Intraday forecast */}
              {data.intraday_forecast && (
                <Card>
                  <CardHeader className="pb-2 pt-4 px-4">
                    <CardTitle className="text-sm font-medium text-muted-foreground">日内预测</CardTitle>
                  </CardHeader>
                  <CardContent className="px-4 pb-4">
                    <div className="space-y-3">
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">预测低</span>
                        <span className="font-mono tabular-nums">{data.intraday_forecast.predicted_low.toFixed(2)}</span>
                      </div>
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">预测高</span>
                        <span className="font-mono tabular-nums">{data.intraday_forecast.predicted_high.toFixed(2)}</span>
                      </div>
                      <div className="space-y-1">
                        <div className="flex justify-between text-xs text-muted-foreground">
                          <span>当前位置</span>
                          <span>{data.intraday_forecast.zone_pct.toFixed(0)}%</span>
                        </div>
                        <div className="h-2 rounded-full bg-muted overflow-hidden">
                          <div
                            className="h-full rounded-full bg-foreground/50 transition-all"
                            style={{ width: `${Math.max(2, Math.min(100, data.intraday_forecast.zone_pct))}%` }}
                          />
                        </div>
                        <div className="flex justify-between text-[10px] text-muted-foreground">
                          <span>低</span>
                          <span>高</span>
                        </div>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        区间: {data.intraday_forecast.current_zone === "lower" ? "偏低区域" : data.intraday_forecast.current_zone === "upper" ? "偏高区域" : "中间区域"}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
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

function ConditionOrderCard({ order, step }: { order: { direction: string; trigger_price: number; trigger_desc: string; order_price: number; order_type: string; quantity_ratio: string; stop_price: number; stop_desc: string; note: string }; step: number }) {
  const isPending = order.note === "待触发";
  return (
    <div className={`rounded-md border p-3 space-y-1.5 ${isPending ? "opacity-70" : ""}`}>
      <div className="flex items-center gap-2">
        <Badge variant="secondary" className="text-[10px]">步骤{step}</Badge>
        <Badge variant="outline">{order.direction}</Badge>
        <span className="text-[10px] text-muted-foreground ml-auto">
          {order.order_type} · {order.quantity_ratio}
          {isPending && <span className="ml-1 opacity-60">（待触发）</span>}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-x-4 gap-y-0.5">
        <KV l="委托价" v={order.order_price.toFixed(2)} />
        <KV l="触发价" v={order.trigger_price.toFixed(2)} />
      </div>
      <p className="text-[11px] text-muted-foreground">{order.trigger_desc}</p>
      <div className="flex items-center gap-1 text-[11px]">
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
