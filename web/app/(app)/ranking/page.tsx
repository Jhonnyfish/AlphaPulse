"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { rankingApi, watchlistApi, searchApi } from "@/lib/api-client";
import { useAuth } from "@/lib/auth";
import type { RankingItem, RankingResponse, SearchSuggestion } from "@/lib/types";
import { DIM_LABELS, signalLabel, formatPct, formatPrice } from "@/lib/constants";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Trophy,
  TrendingUp,
  TrendingDown,
  BarChart3,
  RefreshCw,
  ChevronDown,
  ChevronRight,
  Search,
  Plus,
  Trash2,
  X,
} from "lucide-react";

type SortKey = "rank" | "overall_score" | "change_pct" | "price" | "confidence";
type SortDir = "asc" | "desc";

const CACHE_KEY = "alphapulse_ranking_cache";
const CACHE_TTL = 2 * 60 * 1000;

function getCached(): RankingResponse | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (!raw) return null;
    const { data, ts } = JSON.parse(raw);
    if (Date.now() - ts > CACHE_TTL) return null;
    return data;
  } catch {
    return null;
  }
}

function setCache(data: RankingResponse) {
  localStorage.setItem(CACHE_KEY, JSON.stringify({ data, ts: Date.now() }));
}

function invalidateCache() {
  localStorage.removeItem(CACHE_KEY);
}

export default function RankingPage() {
  const { user, loading: authLoading } = useAuth();
  const [data, setData] = useState<RankingResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("rank");
  const [sortDir, setSortDir] = useState<SortDir>("asc");
  const [expanded, setExpanded] = useState<string | null>(null);

  // Search / filter
  const [filterText, setFilterText] = useState("");

  // Add stock dialog
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [addSearchQuery, setAddSearchQuery] = useState("");
  const [addSearchResults, setAddSearchResults] = useState<SearchSuggestion[]>([]);
  const [addSearching, setAddSearching] = useState(false);
  const [addingCode, setAddingCode] = useState<string | null>(null);
  const addSearchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Delete
  const [deletingCode, setDeletingCode] = useState<string | null>(null);

  const fetchData = useCallback(async (useCache = true) => {
    setLoading(true);
    setError("");
    try {
      if (useCache) {
        const cached = getCached();
        if (cached) {
          setData(cached);
          setLoading(false);
          return;
        }
      }
      const res = await rankingApi.full();
      setData(res);
      setCache(res);
    } catch (err: any) {
      setError(err.message || "加载排名数据失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!authLoading && user) {
      fetchData();
    }
  }, [authLoading, user, fetchData]);

  function handleSort(key: SortKey) {
    if (sortKey === key) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortKey(key);
      setSortDir(key === "rank" ? "asc" : "desc");
    }
  }

  // ── Add stock: search ──
  function handleAddSearchChange(value: string) {
    setAddSearchQuery(value);
    if (addSearchTimer.current) clearTimeout(addSearchTimer.current);
    if (!value.trim()) {
      setAddSearchResults([]);
      return;
    }
    addSearchTimer.current = setTimeout(async () => {
      setAddSearching(true);
      try {
        const results = await searchApi.stocks(value.trim());
        setAddSearchResults(results || []);
      } catch {
        setAddSearchResults([]);
      } finally {
        setAddSearching(false);
      }
    }, 300);
  }

  // ── Add stock: confirm ──
  async function handleAddStock(code: string, name: string) {
    setAddingCode(code);
    try {
      await watchlistApi.add(code, name);
      invalidateCache();
      await fetchData(false);
      // Remove from search results
      setAddSearchResults((prev) => prev.filter((s) => s.code !== code));
    } catch (err: any) {
      setError(err.message || "添加失败");
    } finally {
      setAddingCode(null);
    }
  }

  // ── Delete stock ──
  async function handleDeleteStock(code: string, name?: string) {
    setDeletingCode(code);
    try {
      await watchlistApi.delete(code);
    } catch (err: any) {
      // "not found" means it was already removed — treat as success
      if (!err.message?.includes("not found")) {
        setError(`删除 ${name || code} 失败：${err.message}`);
        setDeletingCode(null);
        return;
      }
    }
    invalidateCache();
    await fetchData(false);
    setDeletingCode(null);
  }

  // ── Filter ──
  const allSorted = data?.items
    ? [...data.items]
        .filter((i) => !i.error)
        .sort((a, b) => {
          let va: number, vb: number;
          switch (sortKey) {
            case "rank":
              va = a.rank; vb = b.rank; break;
            case "overall_score":
              va = a.overall_score; vb = b.overall_score; break;
            case "change_pct":
              va = a.change_pct; vb = b.change_pct; break;
            case "price":
              va = a.price; vb = b.price; break;
            case "confidence":
              va = a.confidence?.overall ?? 0; vb = b.confidence?.overall ?? 0; break;
            default:
              return 0;
          }
          return sortDir === "asc" ? va - vb : vb - va;
        })
    : [];

  const sorted = filterText.trim()
    ? allSorted.filter((item) => {
        const q = filterText.trim().toLowerCase();
        return (
          item.code.includes(q) ||
          item.name.toLowerCase().includes(q) ||
          item.code.startsWith(q)
        );
      })
    : allSorted;

  // Codes already in ranking (for add dialog)
  const existingCodes = new Set(data?.items?.map((i) => i.code) ?? []);

  if (authLoading) return null;

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Trophy className="h-5 w-5 text-muted-foreground" />
          <h1 className="text-xl font-semibold">综合排名</h1>
          {data?.items && (
            <Badge variant="secondary">{data.items.length} 只</Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => fetchData(false)} disabled={loading}>
            <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
            刷新
          </Button>
        </div>
      </div>

      {/* Search + Add bar */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索编号或名称..."
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
            className="pl-8 h-9"
          />
          {filterText && (
            <button
              className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              onClick={() => setFilterText("")}
            >
              <X className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
        <Button size="sm" onClick={() => setAddDialogOpen(true)}>
          <Plus className="mr-1.5 h-3.5 w-3.5" />
          添加股票
        </Button>
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
          <button
            className="ml-2 underline"
            onClick={() => setError("")}
          >
            关闭
          </button>
        </div>
      )}

      {/* Summary cards */}
      {data?.summary && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-1 pt-4 px-4">
              <CardTitle className="text-xs font-medium text-muted-foreground">平均评分</CardTitle>
              <BarChart3 className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent className="pb-4 px-4">
              <div className="text-xl font-bold tabular-nums">{data.summary.avg_score.toFixed(1)}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-1 pt-4 px-4">
              <CardTitle className="text-xs font-medium text-muted-foreground">最佳</CardTitle>
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent className="pb-4 px-4">
              <div className="text-xl font-bold">
                {data.summary.best ? `${data.summary.best.name} ${data.summary.best.score}` : "—"}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-1 pt-4 px-4">
              <CardTitle className="text-xs font-medium text-muted-foreground">最弱</CardTitle>
              <TrendingDown className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent className="pb-4 px-4">
              <div className="text-xl font-bold">
                {data.summary.worst ? `${data.summary.worst.name} ${data.summary.worst.score}` : "—"}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-1 pt-4 px-4">
              <CardTitle className="text-xs font-medium text-muted-foreground">分析数量</CardTitle>
              <Trophy className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent className="pb-4 px-4">
              <div className="text-xl font-bold">{data.summary.count} 只</div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Loading skeleton */}
      {loading && !data && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      )}

      {/* Table */}
      {sorted.length > 0 && (
        <div className="rounded-lg border overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-8" />
                <SortHeader label="排名" sortKey="rank" current={sortKey} dir={sortDir} onSort={handleSort} className="w-16" />
                <SortHeader label="评分" sortKey="overall_score" current={sortKey} dir={sortDir} onSort={handleSort} className="w-20" />
                <TableHead>股票</TableHead>
                <SortHeader label="价格" sortKey="price" current={sortKey} dir={sortDir} onSort={handleSort} className="w-20" />
                <SortHeader label="涨跌幅" sortKey="change_pct" current={sortKey} dir={sortDir} onSort={handleSort} className="w-24" />
                <TableHead className="w-16">信号</TableHead>
                <TableHead className="w-16 text-center">短线</TableHead>
                <TableHead className="w-16 text-center">中线</TableHead>
                <TableHead className="w-16 text-center">长线</TableHead>
                <SortHeader label="置信度" sortKey="confidence" current={sortKey} dir={sortDir} onSort={handleSort} className="w-20 text-center" />
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {sorted.map((item, i) => (
                <RankingRow
                  key={`${item.code}-${item.rank}-${i}`}
                  item={item}
                  isExpanded={expanded === `${item.code}-${item.rank}`}
                  onToggle={() => setExpanded(expanded === `${item.code}-${item.rank}` ? null : `${item.code}-${item.rank}`)}
                  onDelete={() => handleDeleteStock(item.code, item.name)}
                  deleting={deletingCode === item.code}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {/* Empty state */}
      {!loading && !error && sorted.length === 0 && (
        <div className="flex flex-col items-center gap-2 py-20 text-muted-foreground">
          <Trophy className="h-10 w-10" />
          <p>暂无排名数据，请先添加自选股</p>
        </div>
      )}

      {/* Filter info */}
      {filterText.trim() && sorted.length > 0 && (
        <div className="text-xs text-muted-foreground text-center">
          筛选结果：{sorted.length} / {allSorted.length} 只
        </div>
      )}

      {/* ── Add Stock Dialog ── */}
      {addDialogOpen && (
        <AddStockDialog
          existingCodes={existingCodes}
          searchQuery={addSearchQuery}
          searchResults={addSearchResults}
          searching={addSearching}
          addingCode={addingCode}
          onSearchChange={handleAddSearchChange}
          onAdd={handleAddStock}
          onClose={() => {
            setAddDialogOpen(false);
            setAddSearchQuery("");
            setAddSearchResults([]);
          }}
        />
      )}
    </div>
  );
}

// ---- Add Stock Dialog (overlay) ----

function AddStockDialog({
  existingCodes,
  searchQuery,
  searchResults,
  searching,
  addingCode,
  onSearchChange,
  onAdd,
  onClose,
}: {
  existingCodes: Set<string>;
  searchQuery: string;
  searchResults: SearchSuggestion[];
  searching: boolean;
  addingCode: string | null;
  onSearchChange: (value: string) => void;
  onAdd: (code: string, name: string) => void;
  onClose: () => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    // Focus the input when dialog opens
    setTimeout(() => inputRef.current?.focus(), 50);
  }, []);

  // Close on Escape
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      {/* Dialog */}
      <div className="relative bg-background border rounded-xl shadow-2xl w-full max-w-md mx-4 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b">
          <h3 className="font-semibold text-sm">添加自选股</h3>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Search input */}
        <div className="p-4 pb-2">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref={inputRef}
              placeholder="输入股票编号或名称搜索..."
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
              className="pl-8"
            />
          </div>
        </div>

        {/* Results */}
        <div className="px-4 pb-4 max-h-80 overflow-y-auto">
          {searching && (
            <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
              <RefreshCw className="h-4 w-4 animate-spin mr-2" />
              搜索中...
            </div>
          )}

          {!searching && searchQuery.trim() && searchResults.length === 0 && (
            <div className="text-center py-8 text-sm text-muted-foreground">
              未找到匹配的股票
            </div>
          )}

          {!searching && !searchQuery.trim() && (
            <div className="text-center py-8 text-sm text-muted-foreground">
              输入编号或名称搜索股票
            </div>
          )}

          {searchResults.map((s) => {
            const alreadyAdded = existingCodes.has(s.code);
            const isAdding = addingCode === s.code;
            return (
              <div
                key={s.code}
                className="flex items-center justify-between py-2.5 px-3 rounded-lg hover:bg-muted/50 transition-colors"
              >
                <div className="flex flex-col">
                  <span className="font-medium text-sm">{s.name}</span>
                  <span className="text-xs text-muted-foreground font-mono">{s.code}</span>
                </div>
                <Button
                  size="sm"
                  variant={alreadyAdded ? "secondary" : "default"}
                  disabled={alreadyAdded || isAdding}
                  onClick={() => onAdd(s.code, s.name)}
                  className="h-7 text-xs"
                >
                  {isAdding ? (
                    <RefreshCw className="h-3 w-3 animate-spin" />
                  ) : alreadyAdded ? (
                    "已添加"
                  ) : (
                    <Plus className="h-3 w-3 mr-1" />
                  )}
                  {isAdding ? "添加中..." : alreadyAdded ? "已添加" : "添加"}
                </Button>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ---- Sub-components ----

function SortHeader({
  label, sortKey, current, dir, onSort, className,
}: {
  label: string; sortKey: SortKey; current: SortKey; dir: SortDir;
  onSort: (k: SortKey) => void; className?: string;
}) {
  const active = current === sortKey;
  return (
    <TableHead className={className}>
      <button
        onClick={() => onSort(sortKey)}
        className={`flex items-center gap-0.5 transition-colors ${
          active ? "text-foreground" : "text-muted-foreground hover:text-foreground"
        }`}
      >
        {label}
        {active && <span className="text-[10px]">{dir === "asc" ? "↑" : "↓"}</span>}
      </button>
    </TableHead>
  );
}

function RankingRow({
  item, isExpanded, onToggle, onDelete, deleting,
}: {
  item: RankingItem; isExpanded: boolean; onToggle: () => void;
  onDelete: () => void; deleting: boolean;
}) {
  return (
    <>
      <TableRow className="cursor-pointer group" onClick={onToggle}>
        <TableCell className="w-8">
          {isExpanded ? <ChevronDown className="h-4 w-4 text-muted-foreground" />
            : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
        </TableCell>
        <TableCell className="font-mono text-sm tabular-nums">{item.rank}</TableCell>
        <TableCell>
          <Badge variant="secondary" className="font-mono tabular-nums">
            {item.overall_score}
          </Badge>
        </TableCell>
        <TableCell>
          <div className="flex flex-col">
            <span className="font-medium">{item.name}</span>
            <span className="text-xs text-muted-foreground font-mono">{item.code}</span>
          </div>
        </TableCell>
        <TableCell className="font-mono text-sm tabular-nums">{formatPrice(item.price)}</TableCell>
        <TableCell className="font-mono text-sm tabular-nums">{formatPct(item.change_pct)}</TableCell>
        <TableCell>
          <span className="text-xs">{signalLabel(item.overall_signal)}</span>
        </TableCell>
        <TableCell className="text-center text-xs font-mono tabular-nums">
          {item.period_scores?.short?.toFixed(0) ?? "—"}
        </TableCell>
        <TableCell className="text-center text-xs font-mono tabular-nums">
          {item.period_scores?.medium?.toFixed(0) ?? "—"}
        </TableCell>
        <TableCell className="text-center text-xs font-mono tabular-nums">
          {item.period_scores?.long?.toFixed(0) ?? "—"}
        </TableCell>
        <TableCell className="text-center text-xs tabular-nums">
          {(item.confidence?.overall ?? 0).toFixed(0)}%
        </TableCell>
        <TableCell className="w-10" onClick={(e) => e.stopPropagation()}>
          <button
            className="p-1.5 rounded transition-colors text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10"
            onClick={onDelete}
            disabled={deleting}
            title={`删除 ${item.name}`}
          >
            {deleting ? (
              <RefreshCw className="h-3.5 w-3.5 animate-spin text-muted-foreground" />
            ) : (
              <Trash2 className="h-3.5 w-3.5" />
            )}
          </button>
        </TableCell>
      </TableRow>

      {/* Expanded detail */}
      {isExpanded && (
        <TableRow className="bg-muted/30 hover:bg-muted/30">
          <TableCell colSpan={12} className="p-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {/* Dimension scores */}
              <div className="space-y-2">
                <h4 className="text-sm font-medium text-muted-foreground">维度评分</h4>
                <div className="grid grid-cols-2 gap-x-4 gap-y-1">
                  {Object.entries(item.dimension_scores || {}).map(([dim, score]) => (
                    <div key={dim} className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">{DIM_LABELS[dim] || dim}</span>
                      <span className="font-mono tabular-nums">{score.toFixed(1)}</span>
                    </div>
                  ))}
                </div>
              </div>

              {/* Strengths & Risks */}
              <div className="space-y-3">
                {item.strengths?.length > 0 && (
                  <div>
                    <h4 className="text-sm font-medium text-muted-foreground mb-1">优势</h4>
                    <ul className="space-y-0.5">
                      {item.strengths.slice(0, 4).map((s, i) => (
                        <li key={i} className="text-xs text-muted-foreground">• {s}</li>
                      ))}
                    </ul>
                  </div>
                )}
                {item.risks?.length > 0 && (
                  <div>
                    <h4 className="text-sm font-medium text-muted-foreground mb-1">风险</h4>
                    <ul className="space-y-0.5">
                      {item.risks.slice(0, 4).map((r, i) => (
                        <li key={i} className="text-xs text-muted-foreground">• {r}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>

              {/* Meta info */}
              <div className="space-y-2">
                {item.sector && (
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">板块</span>
                    <span>{item.sector}</span>
                  </div>
                )}
                {item.sector_rank > 0 && (
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">板块排名</span>
                    <span className="font-mono">{item.sector_rank}/{item.sector_total}</span>
                  </div>
                )}
                {item.weighted_score > 0 && (
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">加权评分</span>
                    <span className="font-mono tabular-nums">{item.weighted_score.toFixed(1)}</span>
                  </div>
                )}
                {item.score_trend && (
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">趋势</span>
                    <Badge variant="outline">
                      {item.score_trend === "rising" ? "上升"
                        : item.score_trend === "falling" ? "下降"
                        : "平稳"}
                    </Badge>
                  </div>
                )}
                {item.confidence?.data_age && (
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">数据时效</span>
                    <span className="text-xs">
                      {item.confidence.data_age === "today" ? "今日"
                        : item.confidence.data_age === "yesterday" ? "昨日"
                        : "过期"}
                    </span>
                  </div>
                )}
              </div>
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}
