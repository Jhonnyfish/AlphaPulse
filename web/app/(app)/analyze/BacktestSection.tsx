"use client";

import { useState, useCallback, useEffect } from "react";
import type { StrategyResult, EquityPoint, StrategyTrade } from "@/lib/types";
import { strategyApi } from "@/lib/api-client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  Legend,
} from "recharts";

// ──────────────────────────────────────────────
// Chart CSS (works in both light & dark)
// ──────────────────────────────────────────────

const CHART_CSS = `
  .bt-grid line { stroke: oklch(0.7 0 0 / 0.25); }
  .bt-axis .recharts-cartesian-axis-tick text { fill: oklch(0.6 0 0); font-size: 10px; }
  .bt-axis .recharts-cartesian-axis-line { stroke: oklch(0.7 0 0 / 0.3); }
  .bt-ref line { stroke: oklch(0.55 0 0 / 0.4); stroke-dasharray: 4 4; }
  .bt-line-strategy { stroke: oklch(0.55 0.2 250); stroke-width: 2; }
  .bt-line-hold { stroke: oklch(0.6 0.12 60); stroke-width: 2; stroke-dasharray: 6 3; }
  .bt-tooltip { font-size: 11px; border-radius: 6px; background: oklch(0.99 0 0); border: 1px solid oklch(0.82 0 0); color: oklch(0.2 0 0); padding: 8px 12px; }
  .bt-legend { font-size: 11px; }
  .bt-legend .recharts-legend-item-text { color: oklch(0.4 0 0); }
  @media (prefers-color-scheme: dark) {
    .bt-grid line { stroke: oklch(0.4 0 0 / 0.3); }
    .bt-axis .recharts-cartesian-axis-tick text { fill: oklch(0.65 0 0); }
    .bt-axis .recharts-cartesian-axis-line { stroke: oklch(0.4 0 0 / 0.3); }
    .bt-ref line { stroke: oklch(0.55 0 0 / 0.3); }
    .bt-line-strategy { stroke: oklch(0.7 0.18 250); }
    .bt-line-hold { stroke: oklch(0.65 0.1 60); }
    .bt-tooltip { background: oklch(0.22 0 0); border-color: oklch(0.35 0 0); color: oklch(0.9 0 0); }
    .bt-legend .recharts-legend-item-text { color: oklch(0.7 0 0); }
  }
`;

// ──────────────────────────────────────────────
// BacktestSection
// ──────────────────────────────────────────────

const DEFAULT_BACKTEST_DAYS = 30;
const MIN_BACKTEST_DAYS = 5;
const MAX_BACKTEST_DAYS = 500;
const DEFAULT_STRATEGY_ID = "balanced";
const STRATEGY_OPTIONS = [
  { id: "conservative", label: "保守防守" },
  { id: "balanced", label: "均衡持有" },
  { id: "aggressive", label: "进攻持有" },
  { id: "rebound", label: "反弹恢复" },
  { id: "exit_weak", label: "弱势退出" },
];

function strategyLabel(id: string) {
  return STRATEGY_OPTIONS.find((opt) => opt.id === id)?.label || "均衡持有";
}

interface BacktestSectionProps {
  code: string;
  selectedStrategyID?: string;
  onStrategyChange?: (id: string) => void;
}

export default function BacktestSection({
  code,
  selectedStrategyID = DEFAULT_STRATEGY_ID,
  onStrategyChange,
}: BacktestSectionProps) {
  const [open, setOpen] = useState(false);
  const [days, setDays] = useState(DEFAULT_BACKTEST_DAYS);
  const [daysInput, setDaysInput] = useState(String(DEFAULT_BACKTEST_DAYS));
  const [result, setResult] = useState<StrategyResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const strategyID = selectedStrategyID || DEFAULT_STRATEGY_ID;

  useEffect(() => {
    setResult(null);
    setError("");
  }, [code]);

  const fetchBacktest = useCallback(async (d: number, selectedStrategy = strategyID) => {
    if (!code) return;
    setLoading(true);
    setError("");
    try {
      const res = await strategyApi.backtest(code, d, selectedStrategy);
      if (res.error) {
        setError(res.error);
        setResult(null);
      } else {
        setResult(res);
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "回测失败");
      setResult(null);
    } finally {
      setLoading(false);
    }
  }, [code, strategyID]);

  useEffect(() => {
    if (open && result && !loading) {
      fetchBacktest(days, strategyID);
    }
  }, [strategyID]);

  function normalizedDays() {
    const parsed = Number.parseInt(daysInput, 10);
    if (!Number.isFinite(parsed)) return days;
    return Math.max(MIN_BACKTEST_DAYS, Math.min(MAX_BACKTEST_DAYS, parsed));
  }

  function runWithInputDays() {
    const nextDays = normalizedDays();
    setDays(nextDays);
    setDaysInput(String(nextDays));
    fetchBacktest(nextDays, strategyID);
  }

  function chooseStrategy(nextStrategy: string) {
    onStrategyChange?.(nextStrategy);
    if (result && !loading) {
      fetchBacktest(days, nextStrategy);
    }
  }

  // Trigger fetch when opening
  function handleToggle() {
    const next = !open;
    setOpen(next);
    if (next && !result && !loading) {
      fetchBacktest(days, strategyID);
    }
  }

  const beatsHold = (result?.strategy_return_pct ?? 0) > (result?.buy_hold_return_pct ?? 0);

  return (
    <Card>
      <CardHeader
        className="cursor-pointer select-none py-3 px-4"
        onClick={handleToggle}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {open
              ? <ChevronDown className="h-4 w-4 text-muted-foreground" />
              : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
            <CardTitle className="text-sm font-semibold">策略回测</CardTitle>
            {result && !result.error && (
              <>
                <Badge
                  variant="outline"
                  className={`text-[10px] border-0 ${
                    result.strategy_return_pct >= 0
                      ? "text-emerald-700 dark:text-emerald-300"
                      : "text-rose-700 dark:text-rose-300"
                  }`}
                >
                  {result.strategy_return_pct >= 0 ? "+" : ""}{result.strategy_return_pct.toFixed(2)}%
                </Badge>
                <span className={`text-[10px] ${beatsHold ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400"}`}>
                  {beatsHold ? "↑" : "↓"} vs 持有 {result.buy_hold_return_pct >= 0 ? "+" : ""}{result.buy_hold_return_pct.toFixed(2)}%
                </span>
                <span className="text-[10px] text-muted-foreground">
                  {result.total_trades} 笔 · 胜率 {result.win_rate.toFixed(0)}%
                </span>
              </>
            )}
          </div>
          <span className="text-[10px] text-muted-foreground">
            {result?.days ?? days} 天 · {strategyLabel(strategyID)} · 夏普 {result?.sharpe_ratio.toFixed(2) ?? "—"}
          </span>
        </div>
      </CardHeader>

      {open && (
        <CardContent className="pt-0 px-4 pb-4 space-y-4">
          {/* Day range */}
          <div className="flex flex-wrap items-center gap-2">
            <label htmlFor="strategy-backtest-days" className="text-xs text-muted-foreground">回测天数</label>
            <Input
              id="strategy-backtest-days"
              type="number"
              inputMode="numeric"
              min={MIN_BACKTEST_DAYS}
              max={MAX_BACKTEST_DAYS}
              step={1}
              value={daysInput}
              onChange={(e) => setDaysInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") runWithInputDays();
              }}
              className="h-7 w-24 px-2 text-xs"
              disabled={loading}
            />
            <Button
              variant="outline"
              size="sm"
              className="h-7 px-2.5 text-xs"
              onClick={runWithInputDays}
              disabled={loading}
            >
              运行
            </Button>
            <label htmlFor="strategy-backtest-profile" className="ml-1 text-xs text-muted-foreground">策略</label>
            <select
              id="strategy-backtest-profile"
              value={strategyID}
              onChange={(e) => chooseStrategy(e.target.value)}
              disabled={loading}
              className="h-7 rounded-md border bg-background px-2 text-xs text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              {STRATEGY_OPTIONS.map((opt) => (
                <option key={opt.id} value={opt.id}>{opt.label}</option>
              ))}
            </select>
          </div>

          {loading && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground py-8 justify-center">
              <Loader2 className="h-4 w-4 animate-spin" />
              正在运行策略回测引擎…
            </div>
          )}

          {error && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-xs text-destructive">
              {error}
            </div>
          )}

          {!loading && result && !result.error && (
            <>
              {/* Summary cards */}
              <SummaryBar result={result} />

              {/* Equity chart */}
              {result.equity_curve.length > 0 && (
                <div className="rounded-md border p-3">
                  <div className="mb-2 text-xs text-muted-foreground font-medium">
                    净值曲线对比（策略 vs 买入持有）
                  </div>
                  <EquityChart
                    strategy={result.equity_curve}
                    benchmark={result.benchmark_curve}
                  />
                </div>
              )}

              {/* Trade history */}
              {result.trades.length > 0 && (
                <TradeTable trades={result.trades} />
              )}

              {/* Daily signals */}
              {result.daily_signals && result.daily_signals.length > 0 && (
                <SignalTable signals={result.daily_signals} />
              )}
            </>
          )}
        </CardContent>
      )}
    </Card>
  );
}

// ──────────────────────────────────────────────
// Summary bar
// ──────────────────────────────────────────────

function SummaryBar({ result }: { result: StrategyResult }) {
  const items = [
    { label: "策略收益", value: `${result.strategy_return_pct >= 0 ? "+" : ""}${result.strategy_return_pct.toFixed(2)}%`, positive: result.strategy_return_pct >= 0 },
    { label: "持有收益", value: `${result.buy_hold_return_pct >= 0 ? "+" : ""}${result.buy_hold_return_pct.toFixed(2)}%`, positive: result.buy_hold_return_pct >= 0 },
    { label: "夏普比率", value: result.sharpe_ratio.toFixed(2), positive: result.sharpe_ratio > 1 },
    { label: "最大回撤", value: `-${result.max_drawdown_pct.toFixed(2)}%`, positive: false },
    { label: "胜率", value: `${result.win_rate.toFixed(0)}%`, positive: result.win_rate > 50 },
    { label: "信号效率", value: `${result.signal_efficiency.toFixed(0)}%`, positive: true },
    { label: "均持仓", value: `${result.avg_holding_days.toFixed(1)}天`, positive: true },
  ];

  return (
    <div className="grid grid-cols-4 md:grid-cols-7 gap-x-3 gap-y-1 rounded-md bg-muted/40 p-2 text-center">
      {items.map((it) => (
        <div key={it.label}>
          <div className="text-[10px] text-muted-foreground">{it.label}</div>
          <div className={`text-xs font-mono font-medium ${it.positive ? "text-emerald-700 dark:text-emerald-300" : "text-rose-700 dark:text-rose-300"}`}>
            {it.value}
          </div>
        </div>
      ))}
    </div>
  );
}

// ──────────────────────────────────────────────
// Equity chart
// ──────────────────────────────────────────────

interface ChartPoint {
  date: string;
  strategy: number;
  hold: number;
}

function EquityChart({ strategy, benchmark }: { strategy: EquityPoint[]; benchmark: EquityPoint[] }) {
  const data: ChartPoint[] = strategy.map((s, i) => ({
    date: s.date,
    strategy: s.equity,
    hold: benchmark[i]?.equity ?? 1,
  }));

  return (
    <div style={{ width: "100%", height: 300 }}>
      <style dangerouslySetInnerHTML={{ __html: CHART_CSS }} />
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 10, right: 20, bottom: 5, left: 10 }}>
          <CartesianGrid className="bt-grid" strokeDasharray="3 3" />
          <XAxis
            dataKey="date"
            className="bt-axis"
            tick={{ fontSize: 10 }}
            tickFormatter={(v: string) => v.slice(5)}
            interval="preserveStartEnd"
          />
          <YAxis
            className="bt-axis"
            tick={{ fontSize: 10 }}
            domain={["auto", "auto"]}
            tickFormatter={(v: number) => v.toFixed(2)}
          />
          <Tooltip
            contentStyle={{ display: "none" }}
            content={({ active, payload, label }) => {
              if (!active || !payload?.length) return null;
              const stratNav = Number(payload[0]?.value ?? 1);
              const holdNav = Number(payload[1]?.value ?? 1);
              const stratPnl = ((stratNav - 1) * 100).toFixed(2);
              const holdPnl = ((holdNav - 1) * 100).toFixed(2);
              return (
                <div className="bt-tooltip">
                  <div style={{ opacity: 0.6, marginBottom: 4 }}>{String(label)}</div>
                  <div>策略: <strong>{stratNav.toFixed(4)}</strong> <span style={{ color: stratNav >= 1 ? "#16a34a" : "#dc2626" }}>({stratPnl >= "0" ? "+" : ""}{stratPnl}%)</span></div>
                  <div>持有: <strong>{holdNav.toFixed(4)}</strong> <span style={{ color: holdNav >= 1 ? "#16a34a" : "#dc2626" }}>({holdPnl >= "0" ? "+" : ""}{holdPnl}%)</span></div>
                </div>
              );
            }}
          />
          <ReferenceLine y={1} className="bt-ref" />
          <Legend className="bt-legend" formatter={(value: string) => value === "strategy" ? "策略" : "买入持有"} />
          <Line type="monotone" dataKey="strategy" name="strategy" className="bt-line-strategy" strokeWidth={2} dot={false} activeDot={{ r: 3 }} />
          <Line type="monotone" dataKey="hold" name="hold" className="bt-line-hold" strokeWidth={2} dot={false} activeDot={{ r: 3 }} strokeDasharray="6 3" />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

// ──────────────────────────────────────────────
// Trade history table
// ──────────────────────────────────────────────

function TradeTable({ trades }: { trades: StrategyTrade[] }) {
  return (
    <div className="border rounded-md">
      <div className="px-3 py-2 bg-muted/30 border-b text-xs font-medium">
        交易记录（{trades.length} 笔）
      </div>
      <div className="max-h-[300px] overflow-auto">
        <table className="w-full text-[10px] font-mono tabular-nums">
          <thead className="bg-muted/60 sticky top-0 z-10">
            <tr className="text-left">
              <th className="px-2 py-1">ID</th>
              <th className="px-2 py-1">买入日</th>
              <th className="px-2 py-1 text-right">买价</th>
              <th className="px-2 py-1 text-center">评分</th>
              <th className="px-2 py-1 text-center">仓位</th>
              <th className="px-2 py-1">卖出日</th>
              <th className="px-2 py-1 text-right">卖价</th>
              <th className="px-2 py-1">卖出原因</th>
              <th className="px-2 py-1 text-right">天数</th>
              <th className="px-2 py-1 text-right">收益</th>
            </tr>
          </thead>
          <tbody>
            {trades.map((t) => (
              <tr key={t.trade_id} className="border-t hover:bg-muted/30">
                <td className="px-2 py-1">{t.trade_id}</td>
                <td className="px-2 py-1">{t.buy_date}</td>
                <td className="px-2 py-1 text-right">{t.buy_price.toFixed(2)}</td>
                <td className="px-2 py-1 text-center">{t.buy_score}</td>
                <td className="px-2 py-1 text-center">{t.position_pct}%</td>
                <td className="px-2 py-1">{t.sell_date}</td>
                <td className="px-2 py-1 text-right">{t.sell_price.toFixed(2)}</td>
                <td className="px-2 py-1 text-muted-foreground">{t.sell_reason}</td>
                <td className="px-2 py-1 text-right">{t.holding_days}</td>
                <td className={`px-2 py-1 text-right font-medium ${t.return_pct >= 0 ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400"}`}>
                  {t.return_pct >= 0 ? "+" : ""}{t.return_pct.toFixed(2)}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ──────────────────────────────────────────────
// Daily signal table
// ──────────────────────────────────────────────

function SignalTable({ signals }: { signals: StrategyResult["daily_signals"] }) {
  const [showAll, setShowAll] = useState(false);
  const list = signals ?? [];
  const display = showAll ? list : list.slice(-30);

  const actionColor = (a: string) => {
    if (a === "BUY") return "text-blue-600 dark:text-blue-400 font-medium";
    if (a === "SELL") return "text-amber-600 dark:text-amber-400 font-medium";
    return "text-muted-foreground";
  };

  return (
    <div className="border rounded-md">
      <div className="px-3 py-2 bg-muted/30 border-b text-xs font-medium">
        每日信号（近 {list.length} 天）
      </div>
      <div className="max-h-[300px] overflow-auto">
        <table className="w-full text-[10px] font-mono tabular-nums">
          <thead className="bg-muted/60 sticky top-0 z-10">
            <tr className="text-left">
              <th className="px-2 py-1">日期</th>
              <th className="px-2 py-1 text-right">价格</th>
              <th className="px-2 py-1 text-center">评分</th>
              <th className="px-2 py-1 text-center">操作</th>
              <th className="px-2 py-1 text-center">仓位</th>
              <th className="px-2 py-1">原因</th>
            </tr>
          </thead>
          <tbody>
            {display.map((s) => (
              <tr key={s.date} className={`border-t hover:bg-muted/30 ${s.action === "HOLD" ? "opacity-50" : ""}`}>
                <td className="px-2 py-1">{s.date}</td>
                <td className="px-2 py-1 text-right">{s.price.toFixed(2)}</td>
                <td className="px-2 py-1 text-center">{s.score}</td>
                <td className={`px-2 py-1 text-center ${actionColor(s.action)}`}>{s.action}</td>
                <td className="px-2 py-1 text-center">{s.position_pct}%</td>
                <td className="px-2 py-1 text-muted-foreground">{s.reason}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {list.length > 30 && (
        <button
          className="w-full py-1.5 text-[10px] text-muted-foreground hover:text-foreground border-t"
          onClick={() => setShowAll(!showAll)}
        >
          {showAll ? "收起" : `显示全部 ${list.length} 天`}
        </button>
      )}
    </div>
  );
}
