"use client";

import { useState, useCallback } from "react";
import { screenerApi } from "@/lib/api-client";
import type { ScreenerResult } from "@/lib/types";
import { formatPrice } from "@/lib/constants";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Filter, RefreshCw, ChevronDown, ChevronRight, Star } from "lucide-react";

interface Filters {
  min_score: string;
  max_score: string;
  min_momentum: string;
  min_trend: string;
  max_volatility: string;
  tier: string;
}

const DEFAULT_FILTERS: Filters = {
  min_score: "",
  max_score: "",
  min_momentum: "",
  min_trend: "",
  max_volatility: "",
  tier: "",
};

const TIER_OPTIONS = [
  { value: "", label: "全部" },
  { value: "S", label: "S (强推)" },
  { value: "A", label: "A (推荐)" },
  { value: "B", label: "B (观察)" },
  { value: "C", label: "C (关注)" },
];

export default function ScreenerPage() {
  const [filters, setFilters] = useState<Filters>(DEFAULT_FILTERS);
  const [results, setResults] = useState<ScreenerResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [totalCandidates, setTotalCandidates] = useState(0);
  const [filteredCount, setFilteredCount] = useState(0);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [hasSearched, setHasSearched] = useState(false);

  const doScreener = useCallback(async () => {
    setLoading(true);
    setError("");
    setHasSearched(true);
    try {
      const params: Record<string, any> = { limit: 100 };
      if (filters.min_score) params.min_score = Number(filters.min_score);
      if (filters.max_score) params.max_score = Number(filters.max_score);
      if (filters.min_momentum) params.min_momentum = Number(filters.min_momentum);
      if (filters.min_trend) params.min_trend = Number(filters.min_trend);
      if (filters.max_volatility) params.max_volatility = Number(filters.max_volatility);
      if (filters.tier) params.tier = filters.tier;

      const res = await screenerApi.query(params);
      setResults(res.results || []);
      setTotalCandidates(res.total_candidates);
      setFilteredCount(res.filtered);
    } catch (err: any) {
      setError(err.message || "筛选失败");
    } finally {
      setLoading(false);
    }
  }, [filters]);

  function resetFilters() {
    setFilters(DEFAULT_FILTERS);
    setResults([]);
    setHasSearched(false);
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Filter className="h-5 w-5 text-muted-foreground" />
          <h1 className="text-xl font-semibold">Alpha300 筛选器</h1>
          {totalCandidates > 0 && (
            <Badge variant="secondary">
              {filteredCount}/{totalCandidates}
            </Badge>
          )}
        </div>
        <Button variant="outline" size="sm" onClick={resetFilters}>
          重置
        </Button>
      </div>

      {/* Filter panel */}
      <Card>
        <CardHeader className="pb-3 pt-4 px-4">
          <CardTitle className="text-sm font-medium text-muted-foreground">筛选条件</CardTitle>
        </CardHeader>
        <CardContent className="px-4 pb-4">
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-4">
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">最低评分</Label>
              <Input
                type="number"
                placeholder="0"
                value={filters.min_score}
                onChange={(e) => setFilters({ ...filters, min_score: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">最高评分</Label>
              <Input
                type="number"
                placeholder="100"
                value={filters.max_score}
                onChange={(e) => setFilters({ ...filters, max_score: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">最低动量</Label>
              <Input
                type="number"
                placeholder="-100"
                value={filters.min_momentum}
                onChange={(e) => setFilters({ ...filters, min_momentum: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">最低趋势</Label>
              <Input
                type="number"
                placeholder="-100"
                value={filters.min_trend}
                onChange={(e) => setFilters({ ...filters, min_trend: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">最大波动</Label>
              <Input
                type="number"
                placeholder="100"
                value={filters.max_volatility}
                onChange={(e) => setFilters({ ...filters, max_volatility: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">层级</Label>
              <select
                value={filters.tier}
                onChange={(e) => setFilters({ ...filters, tier: e.target.value })}
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                {TIER_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </div>
            <div className="flex items-end">
              <Button onClick={doScreener} disabled={loading} className="w-full">
                {loading ? (
                  <><RefreshCw className="mr-1.5 h-3.5 w-3.5 animate-spin" />筛选中...</>
                ) : (
                  "筛选"
                )}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Error */}
      {error && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      )}

      {/* Results table */}
      {!loading && results.length > 0 && (
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
              {results.map((item, i) => (
                <ScreenerRow
                  key={`${item.code}-${item.rank}-${i}`}
                  item={item}
                  isExpanded={expanded === `${item.code}-${item.rank}`}
                  onToggle={() => setExpanded(expanded === `${item.code}-${item.rank}` ? null : `${item.code}-${item.rank}`)}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {/* Empty state */}
      {!loading && hasSearched && results.length === 0 && !error && (
        <div className="flex flex-col items-center gap-2 py-16 text-muted-foreground">
          <Filter className="h-10 w-10" />
          <p>没有符合条件的股票，请调整筛选条件</p>
        </div>
      )}

      {!loading && !hasSearched && (
        <div className="flex flex-col items-center gap-2 py-16 text-muted-foreground">
          <Filter className="h-10 w-10" />
          <p>设置筛选条件后点击"筛选"</p>
        </div>
      )}
    </div>
  );
}

function ScreenerRow({
  item, isExpanded, onToggle,
}: {
  item: ScreenerResult; isExpanded: boolean; onToggle: () => void;
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
              {item.in_watchlist && (
                <Star className="h-3 w-3 fill-current text-muted-foreground" />
              )}
            </div>
            <span className="text-xs text-muted-foreground font-mono">{item.code}</span>
          </div>
        </TableCell>
        <TableCell className="text-xs text-muted-foreground">{item.industry || "—"}</TableCell>
        <TableCell className="text-center font-mono text-sm tabular-nums">{item.score.toFixed(1)}</TableCell>
        <TableCell className="text-center font-mono text-xs tabular-nums">{item.momentum.toFixed(0)}</TableCell>
        <TableCell className="text-center font-mono text-xs tabular-nums">{item.trend.toFixed(0)}</TableCell>
        <TableCell className="text-center font-mono text-xs tabular-nums">{item.volatility.toFixed(0)}%</TableCell>
        <TableCell className="text-right font-mono text-sm tabular-nums">{formatPrice(item.close)}</TableCell>
        <TableCell className="text-center">
          <Badge variant="outline">{item.recommendation_tier || "—"}</Badge>
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

      {isExpanded && (
        <TableRow className="bg-muted/30 hover:bg-muted/30">
          <TableCell colSpan={11} className="p-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {item.focus_reason && (
                <div>
                  <h4 className="text-sm font-medium text-muted-foreground mb-1">关注理由</h4>
                  <p className="text-xs text-muted-foreground">{item.focus_reason}</p>
                </div>
              )}
              {item.harvest_risk_level && item.harvest_risk_level !== "low" && (
                <div>
                  <h4 className="text-sm font-medium text-muted-foreground mb-1">收割风险</h4>
                  <Badge variant="outline">
                    {item.harvest_risk_level === "high" ? "高" : "中"}
                  </Badge>
                </div>
              )}
              <div>
                <h4 className="text-sm font-medium text-muted-foreground mb-1">指标</h4>
                <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                  <span className="text-muted-foreground">流动性</span>
                  <span className="font-mono tabular-nums">{item.liquidity.toFixed(0)}</span>
                </div>
              </div>
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}
