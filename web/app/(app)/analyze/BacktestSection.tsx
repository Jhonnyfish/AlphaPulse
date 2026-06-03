"use client";

import { useState, useMemo } from "react";
import type { IntradayForecastAccuracy, IntradayForecastDay } from "@/lib/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
// Strategy definitions
// ──────────────────────────────────────────────
// Each strategy is a buy→sell round trip with optional bias filter.
// biasFilter controls WHEN to attempt buying:
//   "none"  — trade every day regardless of bias
//   "bull"  — only buy when bias contains "多" (bullish)
//   "strong"— only buy when bias contains "多" AND bias_strength > 1.0
//
// A "买入持有" (buy-and-hold) baseline is always shown for comparison.

/* eslint-disable @typescript-eslint/no-explicit-any */
type DayRec = any;

interface Strategy {
  id: string;
  label: string;
  desc: string;
  buyLevelFn: (d: DayRec) => number;
  sellLevelFn: (d: DayRec) => number;
  biasFilter: "none" | "bull" | "strong";
  /** Force-sell at actual_close when bias turns bearish while holding */
  bearExit: boolean;
}

const STRATEGIES: Strategy[] = [
  {
    id: "classic",
    label: "经典",
    desc: "买P17→卖P83，无视偏向",
    buyLevelFn: (d) => d.predicted_low,
    sellLevelFn: (d) => d.predicted_high,
    biasFilter: "none",
    bearExit: false,
  },
  {
    id: "bull-only",
    label: "看多才买",
    desc: "偏多时买P17→卖P83",
    buyLevelFn: (d) => d.predicted_low,
    sellLevelFn: (d) => d.predicted_high,
    biasFilter: "bull",
    bearExit: false,
  },
  {
    id: "bull-bear-exit",
    label: "看多买+看空卖",
    desc: "偏多买P17，偏空强制卖出",
    buyLevelFn: (d) => d.predicted_low,
    sellLevelFn: (d) => d.predicted_high,
    biasFilter: "bull",
    bearExit: true,
  },
  {
    id: "strong-signal",
    label: "强信号",
    desc: "偏向强>1.0才买P17→卖P83",
    buyLevelFn: (d) => d.predicted_low,
    sellLevelFn: (d) => d.predicted_high,
    biasFilter: "strong",
    bearExit: false,
  },
  {
    id: "conservative",
    label: "保守",
    desc: "偏多买P5→卖P50，偏空卖出",
    buyLevelFn: (d) => d.predicted_low_down,
    sellLevelFn: (d) => d.predicted_high_median || 0,
    biasFilter: "bull",
    bearExit: true,
  },
  {
    id: "scalp",
    label: "快进快出",
    desc: "偏多买P50→卖P50(高位)",
    buyLevelFn: (d) => d.predicted_low_median || 0,
    sellLevelFn: (d) => d.predicted_high_median || 0,
    biasFilter: "bull",
    bearExit: false,
  },
];

const DAY_PRESETS = [30, 60, 120, 250];

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

function isBullish(d: DayRec): boolean {
  return (d.bias || "").includes("多");
}
function isBearish(d: DayRec): boolean {
  return (d.bias || "").includes("空");
}
function isStrongBias(d: DayRec): boolean {
  return isBullish(d) && (d.bias_strength || 0) > 1.0;
}

function shouldBuy(d: DayRec, filter: Strategy["biasFilter"]): boolean {
  if (filter === "none") return true;
  if (filter === "bull") return isBullish(d);
  if (filter === "strong") return isStrongBias(d);
  return false;
}

// ──────────────────────────────────────────────
// Simulation
// ──────────────────────────────────────────────

interface SimRow {
  date: string;
  prevClose: number;
  buyLevel: number;
  sellLevel: number;
  actualHigh: number;
  actualLow: number;
  actualClose: number;
  bias: string;
  biasStrength: number;
  action: "buy" | "sell" | "bear-exit" | "hold" | "skip" | "cash";
  tradePrice: number;
  holding: boolean;
  buyPrice: number;
  nav: number;
  dailyPnl: number;
}

function simulate(
  details: IntradayForecastDay[],
  strategy: Strategy,
): SimRow[] {
  const sorted = [...details].sort((a, b) => a.date.localeCompare(b.date));
  let holding = false;
  let buyPrice = 0;
  let nav = 1.0;

  return sorted.map((d) => {
    const buyLevel = strategy.buyLevelFn(d);
    const sellLevel = strategy.sellLevelFn(d);
    let action: SimRow["action"] = "cash";
    let tradePrice = 0;
    let realizedPnl = 0;

    if (holding) {
      // ── Bear exit: force sell at actual_close when bias turns bearish ──
      if (strategy.bearExit && isBearish(d) && d.actual_close > 0) {
        action = "bear-exit";
        tradePrice = d.actual_close;
        realizedPnl = (d.actual_close - buyPrice) / buyPrice;
        nav *= (1 + realizedPnl);
        holding = false;
        buyPrice = 0;
      }
      // ── Normal sell: check if price reached sell level ──
      else if (sellLevel > 0 && d.actual_high >= sellLevel) {
        action = "sell";
        tradePrice = sellLevel;
        realizedPnl = (sellLevel - buyPrice) / buyPrice;
        nav *= (1 + realizedPnl);
        holding = false;
        buyPrice = 0;
      } else {
        action = "hold";
      }
    } else {
      // ── Buy: only if bias filter passes ──
      if (!shouldBuy(d, strategy.biasFilter)) {
        action = "skip"; // bias filter says no
      } else if (buyLevel > 0 && d.actual_low <= buyLevel) {
        action = "buy";
        tradePrice = buyLevel;
        holding = true;
        buyPrice = buyLevel;
      } else {
        action = "cash";
      }
    }

    const prevNav = nav;
    let currentNav = nav;
    if (holding && buyPrice > 0) {
      currentNav = nav * (d.actual_close / buyPrice);
    }

    return {
      date: d.date,
      prevClose: d.prev_close,
      buyLevel,
      sellLevel,
      actualHigh: d.actual_high,
      actualLow: d.actual_low,
      actualClose: d.actual_close,
      bias: d.bias || "",
      biasStrength: d.bias_strength || 0,
      action,
      tradePrice,
      holding,
      buyPrice,
      nav: round4(currentNav),
      dailyPnl: round4((currentNav - prevNav) / prevNav * 100),
    };
  });
}

/** Buy-and-hold baseline: buy at first day's prev_close, hold to end. */
function simulateBuyHold(details: IntradayForecastDay[]): SimRow[] {
  const sorted = [...details].sort((a, b) => a.date.localeCompare(b.date));
  if (sorted.length === 0) return [];
  const buyPrice = sorted[0].prev_close;
  let nav = 1.0;

  return sorted.map((d) => {
    const prevNav = nav;
    nav = d.actual_close / buyPrice;
    return {
      date: d.date,
      prevClose: d.prev_close,
      buyLevel: buyPrice,
      sellLevel: 0,
      actualHigh: d.actual_high,
      actualLow: d.actual_low,
      actualClose: d.actual_close,
      bias: d.bias || "",
      biasStrength: d.bias_strength || 0,
      action: "hold" as const,
      tradePrice: 0,
      holding: true,
      buyPrice,
      nav: round4(nav),
      dailyPnl: round4((nav - prevNav) / prevNav * 100),
    };
  });
}

// ──────────────────────────────────────────────
// BacktestSection
// ──────────────────────────────────────────────

interface BacktestSectionProps {
  accuracy: IntradayForecastAccuracy | null;
  loading: boolean;
  error: string;
  days: number;
  onDaysChange: (d: number) => void;
  onApplyRange: () => void;
}

export default function BacktestSection({
  accuracy,
  loading,
  error,
  days,
  onDaysChange,
}: BacktestSectionProps) {
  const [open, setOpen] = useState(false);
  const [strategyIdx, setStrategyIdx] = useState(1); // default: 看多才买

  const strategy = STRATEGIES[strategyIdx];
  const details = accuracy?.details ?? [];

  // Run simulation
  const simRows = useMemo(
    () => (details.length ? simulate(details, strategy) : []),
    [details, strategy],
  );
  const holdRows = useMemo(
    () => (details.length ? simulateBuyHold(details) : []),
    [details],
  );

  // Chart data: strategy NAV + buy-and-hold NAV
  const chartData = useMemo(() => {
    if (!simRows.length) return [];
    return simRows.map((r, i) => ({
      date: r.date,
      nav: r.nav,
      hold: holdRows[i]?.nav ?? 1,
    }));
  }, [simRows, holdRows]);

  // Summary
  const finalNav = chartData.length > 0 ? chartData[chartData.length - 1].nav : 1;
  const holdNav = chartData.length > 0 ? chartData[chartData.length - 1].hold : 1;
  const totalReturn = finalNav - 1;
  const holdReturn = holdNav - 1;
  const sells = simRows.filter((r) => r.action === "sell" || r.action === "bear-exit");
  const wins = sells.filter((r) => r.dailyPnl > 0);
  const winRate = sells.length > 0 ? wins.length / sells.length : 0;
  const beatsHold = totalReturn > holdReturn;

  return (
    <Card>
      <CardHeader
        className="cursor-pointer select-none py-3 px-4"
        onClick={() => setOpen((o) => !o)}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {open
              ? <ChevronDown className="h-4 w-4 text-muted-foreground" />
              : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
            <CardTitle className="text-sm font-semibold">策略回测</CardTitle>
            {accuracy && (
              <>
                <Badge
                  variant="outline"
                  className={`text-[10px] border-0 ${
                    totalReturn >= 0
                      ? "text-emerald-700 dark:text-emerald-300"
                      : "text-rose-700 dark:text-rose-300"
                  }`}
                >
                  {totalReturn >= 0 ? "+" : ""}{(totalReturn * 100).toFixed(2)}%
                </Badge>
                <span className="text-[10px] text-muted-foreground">
                  {sells.length} 笔 · 胜率 {(winRate * 100).toFixed(0)}%
                </span>
                <span className={`text-[10px] ${beatsHold ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400"}`}>
                  {beatsHold ? "↑" : "↓"} vs 持有 {(holdReturn >= 0 ? "+" : "") + (holdReturn * 100).toFixed(2)}%
                </span>
              </>
            )}
          </div>
          <span className="text-[10px] text-muted-foreground">
            {accuracy?.days_evaluated ?? 0} 交易日
          </span>
        </div>
      </CardHeader>

      {open && (
        <CardContent className="pt-0 px-4 pb-4 space-y-4">
          {loading && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground py-4 justify-center">
              <Loader2 className="h-4 w-4 animate-spin" />
              正在回测…
            </div>
          )}
          {error && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-xs text-destructive">
              {error}
            </div>
          )}

          {!loading && accuracy && (
            <>
              {/* Day range */}
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-xs text-muted-foreground">回测天数</span>
                {DAY_PRESETS.map((d) => (
                  <Button
                    key={d}
                    variant={days === d ? "default" : "outline"}
                    size="sm"
                    className="h-7 text-xs px-2.5"
                    onClick={() => onDaysChange(d)}
                  >
                    {d} 天
                  </Button>
                ))}
                {accuracy.days_evaluated < days && (
                  <span className="text-[10px] text-amber-600 dark:text-amber-400">
                    实际可用 {accuracy.days_evaluated} 天
                  </span>
                )}
              </div>

              {/* Strategy selector */}
              <div className="flex flex-wrap items-center gap-3">
                <span className="text-xs text-muted-foreground">策略</span>
                <select
                  value={strategyIdx}
                  onChange={(e) => setStrategyIdx(Number(e.target.value))}
                  className="text-xs px-3 py-1.5 rounded-md border border-input bg-background"
                >
                  {STRATEGIES.map((s, i) => (
                    <option key={s.id} value={i}>
                      {s.label} — {s.desc}
                    </option>
                  ))}
                </select>
                <span className="text-[10px] text-muted-foreground">
                  {strategy.biasFilter === "none" ? "无偏向过滤"
                    : strategy.biasFilter === "bull" ? "仅偏多日买入"
                    : "强信号(>1.0)才买"}
                  {strategy.bearExit ? " · 偏空强制卖出" : ""}
                </span>
              </div>

              {/* Net value chart with buy-and-hold baseline */}
              {chartData.length > 0 && (
                <div className="rounded-md border p-3">
                  <div className="mb-2 text-xs text-muted-foreground font-medium">
                    净值曲线对比
                  </div>
                  <NetValueChart data={chartData} />
                </div>
              )}

              {/* Daily details table */}
              {simRows.length > 0 && (
                <DailyDetailsTable rows={simRows} strategy={strategy} />
              )}
            </>
          )}
        </CardContent>
      )}
    </Card>
  );
}

// ──────────────────────────────────────────────
// Net value chart
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

interface ChartPoint {
  date: string;
  nav: number;
  hold: number;
}

function NetValueChart({ data }: { data: ChartPoint[] }) {
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
              const nav = Number(payload[0]?.value ?? 1).toFixed(4);
              const hold = Number(payload[1]?.value ?? 1).toFixed(4);
              const pnl = ((Number(payload[0]?.value ?? 1) - 1) * 100).toFixed(2);
              const holdPnl = ((Number(payload[1]?.value ?? 1) - 1) * 100).toFixed(2);
              return (
                <div className="bt-tooltip">
                  <div style={{ opacity: 0.6, marginBottom: 4 }}>{String(label)}</div>
                  <div>策略净值: <strong>{nav}</strong> <span style={{ color: Number(pnl) >= 0 ? "#16a34a" : "#dc2626" }}>({Number(pnl) >= 0 ? "+" : ""}{pnl}%)</span></div>
                  <div>持有净值: <strong>{hold}</strong> <span style={{ color: Number(holdPnl) >= 0 ? "#16a34a" : "#dc2626" }}>({Number(holdPnl) >= 0 ? "+" : ""}{holdPnl}%)</span></div>
                </div>
              );
            }}
          />
          <ReferenceLine y={1} className="bt-ref" />
          <Legend className="bt-legend" formatter={(value: string) => value === "nav" ? "策略" : "买入持有"} />
          <Line type="monotone" dataKey="nav" name="nav" className="bt-line-strategy" strokeWidth={2} dot={false} activeDot={{ r: 3 }} />
          <Line type="monotone" dataKey="hold" name="hold" className="bt-line-hold" strokeWidth={2} dot={false} activeDot={{ r: 3 }} strokeDasharray="6 3" />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

// ──────────────────────────────────────────────
// Daily details table
// ──────────────────────────────────────────────

function DailyDetailsTable({
  rows,
  strategy,
}: {
  rows: SimRow[];
  strategy: Strategy;
}) {
  const [sortDir, setSortDir] = useState<"desc" | "asc">("desc");
  const sorted = sortDir === "desc" ? [...rows].reverse() : rows;
  const sells = rows.filter((r) => r.action === "sell" || r.action === "bear-exit");
  const wins = sells.filter((r) => r.dailyPnl > 0);

  const actionLabel = (a: SimRow["action"]) => {
    switch (a) {
      case "buy": return "买入";
      case "sell": return "卖出";
      case "bear-exit": return "空平";
      case "hold": return "持仓";
      case "skip": return "跳过";
      default: return "空仓";
    }
  };

  const actionColor = (a: SimRow["action"]) => {
    switch (a) {
      case "buy": return "text-blue-600 dark:text-blue-400";
      case "sell": return "text-amber-600 dark:text-amber-400";
      case "bear-exit": return "text-rose-600 dark:text-rose-400 font-medium";
      case "hold": return "text-emerald-600 dark:text-emerald-400";
      case "skip": return "text-muted-foreground/50";
      default: return "text-muted-foreground";
    }
  };

  return (
    <div className="border rounded-md">
      <div className="flex flex-wrap items-center gap-3 px-3 py-2 bg-muted/30 border-b text-[10px]">
        <span className="text-muted-foreground">
          交易 <span className="font-mono">{sells.length}</span> 笔
        </span>
        <span className="text-muted-foreground">
          胜率 <span className="font-mono">{sells.length > 0 ? ((wins.length / sells.length) * 100).toFixed(0) : 0}%</span>
        </span>
        {sells.length > 0 && (
          <span className={wins.length >= sells.length - wins.length ? "text-emerald-700 dark:text-emerald-300" : "text-rose-700 dark:text-rose-300"}>
            盈 <span className="font-mono">{wins.length}</span> / 亏 <span className="font-mono">{sells.length - wins.length}</span>
          </span>
        )}
      </div>
      <div className="max-h-[400px] overflow-auto">
        <table className="w-full text-[10px] font-mono tabular-nums">
          <thead className="bg-muted/60 sticky top-0 z-10">
            <tr className="text-left">
              <th className="px-2 py-1 cursor-pointer select-none" onClick={() => setSortDir((d) => (d === "desc" ? "asc" : "desc"))}>
                日期 {sortDir === "desc" ? "↓" : "↑"}
              </th>
              <th className="px-2 py-1 text-right">昨收</th>
              <th className="px-2 py-1 text-right">买价</th>
              <th className="px-2 py-1 text-right">卖价</th>
              <th className="px-2 py-1 text-center">偏向</th>
              <th className="px-2 py-1 text-center">操作</th>
              <th className="px-2 py-1 text-right">成本</th>
              <th className="px-2 py-1 text-right">净值</th>
              <th className="px-2 py-1 text-right">收益</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((r) => (
              <tr key={r.date} className={`border-t hover:bg-muted/30 ${r.action === "cash" || r.action === "skip" ? "opacity-40" : ""}`}>
                <td className="px-2 py-1">{r.date}</td>
                <td className="px-2 py-1 text-right">{r.prevClose.toFixed(2)}</td>
                <td className="px-2 py-1 text-right">{r.buyLevel > 0 ? r.buyLevel.toFixed(2) : "—"}</td>
                <td className="px-2 py-1 text-right">{r.sellLevel > 0 ? r.sellLevel.toFixed(2) : "—"}</td>
                <td className="px-2 py-1 text-center">
                  <span className={
                    r.bias.includes("多") ? "text-emerald-600 dark:text-emerald-400" :
                    r.bias.includes("空") ? "text-rose-600 dark:text-rose-400" :
                    "text-muted-foreground"
                  }>
                    {r.bias || "—"}
                  </span>
                </td>
                <td className={`px-2 py-1 text-center ${actionColor(r.action)}`}>{actionLabel(r.action)}</td>
                <td className="px-2 py-1 text-right">{r.holding && r.buyPrice > 0 ? r.buyPrice.toFixed(2) : "—"}</td>
                <td className={`px-2 py-1 text-right font-medium ${r.nav >= 1 ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400"}`}>
                  {r.nav.toFixed(4)}
                </td>
                <td className={`px-2 py-1 text-right ${
                  r.action === "sell" || r.action === "bear-exit"
                    ? r.dailyPnl > 0 ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400"
                    : "text-muted-foreground"
                }`}>
                  {(r.action === "sell" || r.action === "bear-exit")
                    ? `${r.dailyPnl >= 0 ? "+" : ""}${r.dailyPnl.toFixed(2)}%`
                    : "—"}
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
// Helpers
// ──────────────────────────────────────────────

function round4(n: number): number {
  return Math.round(n * 10000) / 10000;
}
