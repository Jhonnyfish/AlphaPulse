"use client";

import { useState, useEffect, useCallback } from "react";
import { candidatesApi } from "@/lib/api-client";
import { useAuth } from "@/lib/auth";
import type { Alpha300Candidate } from "@/lib/types";
import { TIER_CONFIG, formatPrice } from "@/lib/constants";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  BarChart3,
  RefreshCw,
  Target,
  ChevronDown,
  ChevronRight,
  TrendingUp,
  AlertTriangle,
  Star,
} from "lucide-react";

type TabValue = "focus" | "observe" | "all";

function tierFor(item: Alpha300Candidate): string {
  return item.recommendation_tier || "watch";
}

function tierFilter(tab: TabValue, item: Alpha300Candidate): boolean {
  const t = tierFor(item);
  switch (tab) {
    case "focus":
      return t === "focus" || t === "strong_buy" || t === "buy";
    case "observe":
      return t === "observe";
    default:
      return true;
  }
}

function tierLabel(tier: string): string {
  return TIER_CONFIG[tier]?.label ?? tier ?? "—";
}

export default function Alpha300Page() {
  const { user, loading: authLoading } = useAuth();
  const [candidates, setCandidates] = useState<Alpha300Candidate[]>([]);
  const [tierCounts, setTierCounts] = useState<Record<string, number>>({});
  const [fetchedAt, setFetchedAt] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [tab, setTab] = useState<TabValue>("focus");
  const [expanded, setExpanded] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const res = await candidatesApi.list({ limit: 100, strategy: "builtin" });
      setCandidates(res.data.items || []);
      setTierCounts(res.data.tier_counts || {});
      setFetchedAt(res.data.fetched_at);
    } catch (err: any) {
      setError(err.message || "加载 Alpha300 数据失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!authLoading && user) {
      fetchData();
    }
  }, [authLoading, user, fetchData]);

  const filtered = candidates.filter((c) => tierFilter(tab, c));
  const leaderCount = candidates.filter(
    (c) => c.leader_signal && c.leader_signal !== "none"
  ).length;
  const avgScore =
    candidates.length > 0
      ? candidates.reduce((s, c) => s + c.score, 0) / candidates.length
      : 0;

  const focusCount =
    (tierCounts["focus"] || 0) +
    (tierCounts["strong_buy"] || 0) +
    (tierCounts["buy"] || 0);

  if (authLoading) return null;

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <BarChart3 className="h-5 w-5 text-muted-foreground" />
          <h1 className="text-xl font-semibold">Alpha300 排行榜</h1>
          <Badge variant="secondary">{candidates.length} 只</Badge>
        </div>
        <div className="flex items-center gap-3">
          {fetchedAt && (
            <span className="text-xs text-muted-foreground">
              更新于{" "}
              {new Date(fetchedAt).toLocaleTimeString("zh-CN", {
                hour: "2-digit",
                minute: "2-digit",
              })}
            </span>
          )}
          <Button variant="outline" size="sm" onClick={fetchData} disabled={loading}>
            <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
            刷新
          </Button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {/* Stats cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-1 pt-4 px-4">
            <CardTitle className="text-xs font-medium text-muted-foreground">重点关注</CardTitle>
            <Star className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="pb-4 px-4">
            <div className="text-xl font-bold tabular-nums">{focusCount}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-1 pt-4 px-4">
            <CardTitle className="text-xs font-medium text-muted-foreground">观察池</CardTitle>
            <Target className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="pb-4 px-4">
            <div className="text-xl font-bold tabular-nums">{tierCounts["observe"] || 0}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-1 pt-4 px-4">
            <CardTitle className="text-xs font-medium text-muted-foreground">龙头信号</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="pb-4 px-4">
            <div className="text-xl font-bold tabular-nums">{leaderCount}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-1 pt-4 px-4">
            <CardTitle className="text-xs font-medium text-muted-foreground">平均评分</CardTitle>
            <BarChart3 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="pb-4 px-4">
            <div className="text-xl font-bold tabular-nums">{avgScore.toFixed(1)}</div>
          </CardContent>
        </Card>
      </div>

      {/* Loading skeleton */}
      {loading && candidates.length === 0 && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      )}

      {/* Tabs + Table */}
      {(!loading || candidates.length > 0) && (
        <Tabs value={tab} onValueChange={(v) => setTab(v as TabValue)} className="space-y-4">
          <TabsList>
            <TabsTrigger value="focus">
              重点关注
              {focusCount > 0 && (
                <Badge variant="secondary" className="ml-1.5 text-[10px] px-1">{focusCount}</Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="observe">观察池</TabsTrigger>
            <TabsTrigger value="all">全部</TabsTrigger>
          </TabsList>

          <TabsContent value={tab} className="mt-0">
            <div className="rounded-lg border overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="w-8" />
                    <TableHead className="w-14">排名</TableHead>
                    <TableHead>代码/名称</TableHead>
                    <TableHead className="w-16">行业</TableHead>
                    <TableHead className="w-16 text-center">评分</TableHead>
                    <TableHead className="w-16 text-center">动量</TableHead>
                    <TableHead className="w-16 text-center">趋势</TableHead>
                    <TableHead className="w-16 text-center">波动</TableHead>
                    <TableHead className="w-20 text-right">收盘价</TableHead>
                    <TableHead className="w-20 text-center">层级</TableHead>
                    <TableHead className="w-16 text-center">信号</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filtered.map((item, i) => (
                    <CandidateRow
                      key={`${item.code}-${item.rank}-${i}`}
                      item={item}
                      isExpanded={expanded === `${item.code}-${item.rank}`}
                      onToggle={() => setExpanded(expanded === `${item.code}-${item.rank}` ? null : `${item.code}-${item.rank}`)}
                    />
                  ))}
                </TableBody>
              </Table>
            </div>

            {filtered.length === 0 && (
              <div className="flex flex-col items-center gap-2 py-16 text-muted-foreground">
                <Target className="h-10 w-10" />
                <p>当前分类暂无数据</p>
              </div>
            )}
          </TabsContent>
        </Tabs>
      )}
    </div>
  );
}

// ---- Candidate Row ----

function CandidateRow({
  item, isExpanded, onToggle,
}: {
  item: Alpha300Candidate; isExpanded: boolean; onToggle: () => void;
}) {
  return (
    <>
      <TableRow className="cursor-pointer" onClick={onToggle}>
        <TableCell className="w-8">
          {isExpanded ? <ChevronDown className="h-4 w-4 text-muted-foreground" />
            : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
        </TableCell>
        <TableCell className="font-mono text-sm tabular-nums">{item.rank}</TableCell>
        <TableCell>
          <div className="flex flex-col">
            <div className="flex items-center gap-1.5">
              <span className="font-medium">{item.name}</span>
              {item.limit_up_today && (
                <Badge variant="outline" className="text-[10px] px-1 py-0">涨停</Badge>
              )}
              {item.limit_up_prev_day && (
                <Badge variant="outline" className="text-[10px] px-1 py-0">昨涨停</Badge>
              )}
              {item.in_watchlist && (
                <Star className="h-3 w-3 fill-current text-muted-foreground" />
              )}
            </div>
            <span className="text-xs text-muted-foreground font-mono">{item.code}</span>
          </div>
        </TableCell>
        <TableCell className="text-xs text-muted-foreground">{item.industry || "—"}</TableCell>
        <TableCell className="text-center font-mono text-sm tabular-nums">{item.score.toFixed(1)}</TableCell>
        <TableCell className="text-center">
          <FactorBar value={item.momentum} />
        </TableCell>
        <TableCell className="text-center">
          <FactorBar value={item.trend} />
        </TableCell>
        <TableCell className="text-center text-xs text-muted-foreground tabular-nums">
          {item.volatility.toFixed(0)}%
        </TableCell>
        <TableCell className="text-right font-mono text-sm tabular-nums">{formatPrice(item.close)}</TableCell>
        <TableCell className="text-center">
          <Badge variant="outline">{tierLabel(tierFor(item))}</Badge>
        </TableCell>
        <TableCell className="text-center">
          {item.leader_signal && item.leader_signal !== "none" ? (
            <Badge variant="outline" className="text-[10px]">
              {item.leader_signal === "sector_leader" ? "龙头" : item.leader_signal}
            </Badge>
          ) : (
            <span className="text-muted-foreground">—</span>
          )}
        </TableCell>
      </TableRow>

      {/* Expanded detail */}
      {isExpanded && (
        <TableRow className="bg-muted/30 hover:bg-muted/30">
          <TableCell colSpan={11} className="p-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {/* Trade plan */}
              <div className="space-y-2">
                <h4 className="text-sm font-medium text-muted-foreground flex items-center gap-1.5">
                  <TrendingUp className="h-3.5 w-3.5" />
                  交易计划
                </h4>
                <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 text-sm">
                  <span className="text-muted-foreground">买入区间</span>
                  <span className="font-mono tabular-nums">
                    {formatPrice(item.buy_low)} ~ {formatPrice(item.buy_high)}
                  </span>
                  <span className="text-muted-foreground">止盈区间</span>
                  <span className="font-mono tabular-nums">
                    {formatPrice(item.sell_low)} ~ {formatPrice(item.sell_high)}
                  </span>
                  <span className="text-muted-foreground">止损价</span>
                  <span className="font-mono tabular-nums">{formatPrice(item.stop_loss)}</span>
                  <span className="text-muted-foreground">ATR14</span>
                  <span className="font-mono tabular-nums">{item.atr14.toFixed(2)}</span>
                </div>
              </div>

              {/* Risk & signal */}
              <div className="space-y-2">
                {item.harvest_risk_level && item.harvest_risk_level !== "low" && (
                  <div>
                    <h4 className="text-sm font-medium text-muted-foreground flex items-center gap-1.5">
                      <AlertTriangle className="h-3.5 w-3.5" />
                      收割风险
                    </h4>
                    <div className="mt-1">
                      <Badge variant="outline">
                        {item.harvest_risk_level === "high" ? "高" : "中"}
                      </Badge>
                    </div>
                    {item.harvest_risk_note && (
                      <p className="mt-1 text-xs text-muted-foreground">{item.harvest_risk_note}</p>
                    )}
                  </div>
                )}
                {item.focus_reason && (
                  <div>
                    <h4 className="text-sm font-medium text-muted-foreground">关注理由</h4>
                    <p className="text-xs text-muted-foreground mt-0.5">{item.focus_reason}</p>
                  </div>
                )}
              </div>

              {/* Focus metrics */}
              <div className="space-y-2">
                <h4 className="text-sm font-medium text-muted-foreground">关注指标</h4>
                <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                  {item.focus_rank > 0 && (
                    <>
                      <span className="text-muted-foreground">关注排名</span>
                      <span className="font-mono tabular-nums">#{item.focus_rank}</span>
                    </>
                  )}
                  {item.focus_score > 0 && (
                    <>
                      <span className="text-muted-foreground">关注评分</span>
                      <span className="font-mono tabular-nums">{item.focus_score.toFixed(1)}</span>
                    </>
                  )}
                  <span className="text-muted-foreground">流动性</span>
                  <span><FactorBar value={item.liquidity} /></span>
                </div>
              </div>
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

// ---- Factor bar ----

function FactorBar({ value }: { value: number }) {
  const pct = Math.max(0, Math.min(100, ((value + 100) / 200) * 100));

  return (
    <div className="flex items-center gap-1.5">
      <div className="h-1.5 flex-1 rounded-full bg-muted overflow-hidden">
        <div
          className="h-full rounded-full bg-foreground/50 transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-[10px] font-mono text-muted-foreground w-8 text-right tabular-nums">
        {value.toFixed(0)}
      </span>
    </div>
  );
}
