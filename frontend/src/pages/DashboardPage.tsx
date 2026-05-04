import { useState, useEffect, useCallback, useRef } from 'react';
import {
  marketApi,
  signalApi,
  systemApi,
  dashboardApi,
  type MarketOverview,
  type Sector,
  type DashboardSignal,
  type ActivityEntry,
  type KlinePoint,
} from '@/lib/api';
import EChart from '@/components/charts/EChart';
import ErrorState from '@/components/ErrorState';
import type { EChartsOption } from 'echarts';
import {
  Activity,
  RefreshCw,
  TrendingUp,
  TrendingDown,
  Minus,
  Zap,
  Clock,
  BarChart3,
  Layers,
  Bell,
  ChevronRight,
} from 'lucide-react';

const REFRESH_INTERVAL = 60_000;

/* ───────────────── helpers ───────────────── */

const changeColor = (n: number) =>
  n > 0 ? 'text-red-500' : n < 0 ? 'text-green-500' : 'text-white/60';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const TrendIcon = (pct: number) =>
  pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Minus;

function formatTime(ts: string) {
  try {
    return new Date(ts).toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return ts;
  }
}

function formatDate(ts: string) {
  try {
    return new Date(ts).toLocaleDateString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
    });
  } catch {
    return ts;
  }
}

/* ───────────────── Glass Card wrapper ───────────────── */
function GlassCard({
  children,
  className = '',
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`bg-[#222536]/80 backdrop-blur-xl border border-white/10 rounded-2xl ${className}`}
    >
      {children}
    </div>
  );
}

/* ───────────────── Empty State ───────────────── */
function EmptyState({ text }: { text: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-white/40">
      <BarChart3 className="w-10 h-10 mb-3 opacity-40" />
      <p className="text-sm">{text}</p>
    </div>
  );
}

/* ───────────────── Skeleton Shimmer Components ───────────────── */

function ChartSkeleton({ height }: { height: number }) {
  return (
    <div
      className="w-full rounded-lg animate-pulse bg-gray-700/50"
      style={{ height }}
    />
  );
}

function IndexChartSkeleton() {
  return (
    <div className="animate-pulse">
      {/* Legend dots placeholder */}
      <div className="flex items-center gap-6 mb-3">
        {[0, 1, 2].map((i) => (
          <div key={i} className="flex items-center gap-1.5">
            <div className="w-2.5 h-2.5 rounded-full bg-gray-700/50" />
            <div className="w-16 h-3 rounded bg-gray-700/50" />
          </div>
        ))}
      </div>
      <ChartSkeleton height={300} />
    </div>
  );
}

function BreadthChartSkeleton() {
  return (
    <div className="animate-pulse">
      <ChartSkeleton height={180} />
      {/* Bar shape placeholders */}
      <div className="flex items-end justify-center gap-8 mt-4">
        {[80, 60, 40].map((h, i) => (
          <div
            key={i}
            className="w-12 rounded-t bg-gray-700/50"
            style={{ height: h }}
          />
        ))}
      </div>
    </div>
  );
}

function TreemapSkeleton() {
  return (
    <div className="w-full rounded-lg bg-gray-700/30 p-3 animate-pulse" style={{ height: 280 }}>
      <div className="grid grid-cols-4 gap-2 h-full">
        {Array.from({ length: 12 }).map((_, i) => (
          <div
            key={i}
            className="rounded bg-gray-700/50"
            style={{
              gridColumn: i === 0 ? 'span 2' : i === 3 ? 'span 2' : 'span 1',
              gridRow: i === 0 ? 'span 2' : 'span 1',
            }}
          />
        ))}
      </div>
    </div>
  );
}

function SignalSkeleton() {
  return (
    <div className="space-y-2 animate-pulse">
      {/* Summary cards placeholder */}
      <div className="grid grid-cols-3 gap-2 mb-3">
        {[0, 1, 2].map((i) => (
          <div key={i} className="flex flex-col items-center py-3 rounded-lg bg-gray-700/30">
            <div className="w-8 h-5 rounded bg-gray-700/50" />
            <div className="w-10 h-2.5 rounded bg-gray-700/50 mt-1.5" />
          </div>
        ))}
      </div>
      {/* Signal row placeholders */}
      {[0, 1, 2, 3].map((i) => (
        <div
          key={i}
          className="flex items-start gap-3 px-3 py-2.5 rounded-xl bg-white/[0.03]"
        >
          <div className="w-3.5 h-3.5 rounded-full bg-gray-700/50 flex-shrink-0 mt-0.5" />
          <div className="flex-1 space-y-1.5">
            <div className="flex items-center gap-2">
              <div className="w-12 h-3 rounded bg-gray-700/50" />
              <div className="w-20 h-3 rounded bg-gray-700/50" />
              <div className="w-10 h-2.5 rounded bg-gray-700/50 ml-auto" />
            </div>
            <div className="w-3/4 h-2.5 rounded bg-gray-700/50" />
          </div>
        </div>
      ))}
    </div>
  );
}

function TimelineSkeleton() {
  return (
    <div className="space-y-0 animate-pulse">
      {[0, 1, 2, 3, 4].map((i) => (
        <div key={i} className="flex items-start gap-3">
          {/* Timeline dot + line */}
          <div className="flex flex-col items-center flex-shrink-0">
            <div className="w-2 h-2 rounded-full bg-gray-700/50 mt-1.5" />
            <div className="w-px flex-1 min-h-[20px] bg-white/[0.06]" />
          </div>
          {/* Content placeholder */}
          <div className="pb-3 flex-1 space-y-1.5">
            <div className="flex items-center gap-2">
              <div className="w-24 h-3 rounded bg-gray-700/50" />
              <div className="w-10 h-2.5 rounded bg-gray-700/50" />
            </div>
            <div className="w-3/5 h-2.5 rounded bg-gray-700/50" />
          </div>
        </div>
      ))}
    </div>
  );
}

/* ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   DashboardPage
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ */

export default function DashboardPage() {
  /* ─── state ─── */
  const [overview, setOverview] = useState<MarketOverview | null>(null);
  const [klines, setKlines] = useState<Record<string, KlinePoint[]>>({});
  const [sectors, setSectors] = useState<Sector[]>([]);
  const [signals, setSignals] = useState<DashboardSignal[]>([]);
  const [activity, setActivity] = useState<ActivityEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const intervalRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  /* ─── data fetching ─── */
  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try composite endpoint first
      let compositeUsed = false;
      let ov: MarketOverview | null = null;
      try {
        const compositeRes = await dashboardApi.summary();
        const data = compositeRes.data;

        if (data.ok) {
          compositeUsed = true;

          // Extract market_overview → MarketOverview
          if (data.market_overview) {
            const mo = data.market_overview;
            // Map IndexQuote fields (change_pct) to OverviewIndex fields (change_percent)
            const mappedIndices: MarketOverview['indices'] = (mo.indices ?? []).map(
              (idx) => ({
                code: idx.code.replace(/^(sh|sz)/, ''),
                name: idx.name,
                price: idx.price,
                change: idx.change,
                change_percent: idx.change_pct ?? 0,
                advance_count: 0,
                decline_count: 0,
                flat_count: 0,
              }),
            );
            ov = {
              advance_count: mo.market?.up_count ?? 0,
              decline_count: mo.market?.down_count ?? 0,
              flat_count: mo.market?.flat_count ?? 0,
              indices: mappedIndices,
              updated_at: mo.updated_at ?? '',
            };
            setOverview(ov);
          }

          // Extract sectors
          if (data.sectors) {
            setSectors(data.sectors);
          }

          // Extract signals
          if (data.signals) {
            setSignals(data.signals);
          }

          // Extract recent_activity
          if (data.recent_activity) {
            setActivity(data.recent_activity);
          }
        }
      } catch {
        // Composite endpoint failed, fall through to individual calls
        compositeUsed = false;
      }

      // Fallback: use individual API calls if composite failed
      if (!compositeUsed) {
        const [overviewRes, sectorsRes, signalsRes, activityRes] =
          await Promise.allSettled([
            marketApi.overview(),
            marketApi.sectors(),
            signalApi.history({ days: 7 }),
            systemApi.activityLog(),
          ]);

        if (overviewRes.status === 'fulfilled') {
          ov = overviewRes.value.data;
          setOverview(ov);
        }

        if (sectorsRes.status === 'fulfilled') {
          setSectors(sectorsRes.value.data ?? []);
        }

        if (signalsRes.status === 'fulfilled') {
          const body = signalsRes.value.data;
          setSignals(body?.items ?? []);
        }

        if (activityRes.status === 'fulfilled') {
          setActivity(activityRes.value.data?.entries ?? []);
        }
      }

      // Fetch klines for each index (up to 3) — always separate calls
      if (ov && ov.indices.length > 0) {
        const klineResults = await Promise.allSettled(
          ov.indices.slice(0, 3).map((idx) =>
            marketApi.kline(idx.code, 30).then((r) => ({
              code: idx.code,
              data: r.data,
            })),
          ),
        );
        const map: Record<string, KlinePoint[]> = {};
        klineResults.forEach((r) => {
          if (r.status === 'fulfilled') {
            map[r.value.code] = r.value.data;
          }
        });
        setKlines(map);
      }

      setLastUpdated(new Date());
    } catch (err) {
      if (!overview) {
        setError(err instanceof Error ? err.message : '加载数据失败');
      }
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchData, REFRESH_INTERVAL);
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [autoRefresh, fetchData]);

  /* ─── derived data ─── */
  const advCount = overview?.advance_count ?? 0;
  const decCount = overview?.decline_count ?? 0;
  const flatCount = overview?.flat_count ?? 0;

  /* ═══════════════════════════════════════════
     1. Index Trend – ECharts line chart
     ═══════════════════════════════════════════ */
  const indexLineOption: EChartsOption = (() => {
    const indices = overview?.indices ?? [];
    if (indices.length === 0) return {};

    // Build series from kline data; fall back to single-point if no klines
    const allDates = new Set<string>();
    indices.forEach((idx) => {
      (klines[idx.code] ?? []).forEach((k) => allDates.add(k.date));
    });
    const sortedDates = Array.from(allDates).sort();

    const series = indices.map((idx, i) => {
      const kl = klines[idx.code];
      const colors = ['#3b82f6', '#f59e0b', '#8b5cf6'];
      if (kl && kl.length > 1) {
        const dateToClose: Record<string, number> = {};
        kl.forEach((k) => {
          dateToClose[k.date] = k.close;
        });
        return {
          name: idx.name,
          type: 'line' as const,
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2 },
          itemStyle: { color: colors[i % colors.length] },
          areaStyle: {
            color: {
              type: 'linear' as const,
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: `${colors[i % colors.length]}33` },
                { offset: 1, color: `${colors[i % colors.length]}05` },
              ],
            },
          },
          data: sortedDates.map((d) => dateToClose[d] ?? null),
        };
      }
      // fallback: single data point
      return {
        name: idx.name,
        type: 'line' as const,
        smooth: true,
        symbol: 'circle',
        symbolSize: 8,
        itemStyle: { color: colors[i % colors.length] },
        data: [idx.price],
      };
    });

    return {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(15,23,42,0.92)',
        borderColor: 'rgba(148,163,184,0.2)',
        textStyle: { color: '#e2e8f0', fontSize: 12 },
      },
      legend: {
        top: 8,
        textStyle: { color: '#94a3b8', fontSize: 12 },
      },
      grid: { left: 60, right: 20, top: 45, bottom: 30 },
      xAxis: {
        type: 'category',
        data: sortedDates.length > 1 ? sortedDates : indices.map((i) => i.name),
        boundaryGap: false,
      },
      yAxis: {
        type: 'value',
        scale: true,
      },
      series,
    };
  })();

  /* ═══════════════════════════════════════════
     2. Breadth Bar Chart – advance / decline / flat
     ═══════════════════════════════════════════ */
  const breadthBarOption: EChartsOption = {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(15,23,42,0.92)',
      borderColor: 'rgba(148,163,184,0.2)',
      textStyle: { color: '#e2e8f0', fontSize: 12 },
    },
    grid: { left: 60, right: 20, top: 20, bottom: 30 },
    xAxis: {
      type: 'category',
      data: ['上涨', '下跌', '平盘'],
    },
    yAxis: { type: 'value' },
    series: [
      {
        type: 'bar',
        barWidth: '50%',
        data: [
          { value: advCount, itemStyle: { color: '#ef4444', borderRadius: [6, 6, 0, 0] } },
          { value: decCount, itemStyle: { color: '#22c55e', borderRadius: [6, 6, 0, 0] } },
          { value: flatCount, itemStyle: { color: '#6b7280', borderRadius: [6, 6, 0, 0] } },
        ],
        label: {
          show: true,
          position: 'top',
          color: '#94a3b8',
          fontSize: 12,
        },
      },
    ],
  };

  /* ═══════════════════════════════════════════
     3. Sector Treemap
     ═══════════════════════════════════════════ */
  const sectorTreemapOption: EChartsOption = (() => {
    if (sectors.length === 0) return {};
    const maxAbs = Math.max(
      ...sectors.map((s) => Math.abs(s.change_percent)),
      1,
    );
    const treeData = sectors.map((s) => {
      const pct = s.change_percent;
      const intensity = Math.min(Math.abs(pct) / maxAbs, 1);
      // red for up, green for down
      const r = pct >= 0 ? 239 : 34;
      const g = pct >= 0 ? 68 : 197;
      const b = pct >= 0 ? 68 : 94;
      const alpha = 0.25 + intensity * 0.55;
      return {
        name: s.name,
        value: Math.abs(s.change_percent) * 100 + 10, // ensure visible
        change_pct: s.change_percent,
        itemStyle: {
          color: `rgba(${r},${g},${b},${alpha})`,
          borderColor: 'rgba(255,255,255,0.08)',
          borderWidth: 1,
        },
      };
    });
    return {
      tooltip: {
        backgroundColor: 'rgba(15,23,42,0.92)',
        borderColor: 'rgba(148,163,184,0.2)',
        textStyle: { color: '#e2e8f0', fontSize: 12 },
        formatter: (params: unknown) => {
          const p = params as { name: string; data?: { change_pct?: number } };
          const cp = p.data?.change_pct ?? 0;
          const sign = cp >= 0 ? '+' : '';
          return `<b>${p.name}</b><br/>涨跌幅: <span style="color:${cp >= 0 ? '#ef4444' : '#22c55e'}">${sign}${cp.toFixed(2)}%</span>`;
        },
      },
      series: [
        {
          type: 'treemap',
          roam: false,
          nodeClick: false,
          breadcrumb: { show: false },
          label: {
            show: true,
            formatter: (params: unknown) => {
              const p = params as { name: string; data?: { change_pct?: number } };
              const cp = p.data?.change_pct ?? 0;
              const sign = cp >= 0 ? '+' : '';
              return `{name|${p.name}}\n{pct|${sign}${cp.toFixed(2)}%}`;
            },
            rich: {
              name: { color: '#fff', fontSize: 12, fontWeight: 600, lineHeight: 18 },
              pct: { color: '#e2e8f0', fontSize: 11, lineHeight: 16 },
            },
          },
          data: treeData,
        },
      ],
    };
  })();

  /* ═══════════════════════════════════════════
     Render
     ═══════════════════════════════════════════ */
  return (
    <div className="min-h-screen">
      {/* ─── Header ─── */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2.5">
          <Activity className="w-5 h-5 text-blue-500" />
          <h1 className="text-xl font-bold text-white">市场总览</h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setAutoRefresh((p) => !p)}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm border transition-colors ${
              autoRefresh
                ? 'bg-blue-500/15 text-blue-400 border-blue-500/40'
                : 'bg-transparent text-white/50 border-white/10'
            }`}
          >
            <span
              className={`w-2 h-2 rounded-full ${
                autoRefresh ? 'bg-green-400 animate-pulse' : 'bg-gray-500'
              }`}
            />
            {autoRefresh ? '自动刷新' : '已暂停'}
          </button>
          <button
            onClick={fetchData}
            disabled={loading}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm text-white/60 hover:bg-white/5 border border-white/10 transition-colors"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </button>
        </div>
      </div>

      {/* ─── Grid layout: 2 cols on lg, 1 col on small ─── */}
      {error && (
        <ErrorState
          title="加载失败"
          description={error}
          onRetry={fetchData}
        />
      )}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
        {/* ═══════ 1. 指数走势图 ═══════ */}
        <GlassCard className="p-5 lg:col-span-2">
          <div className="flex items-center gap-2 mb-4">
            <TrendingUp className="w-4 h-4 text-blue-400" />
            <h2 className="text-white font-semibold text-sm">指数走势</h2>
            {overview?.indices && overview.indices.length > 0 && (
              <div className="flex items-center gap-4 ml-auto">
                {overview.indices.map((idx) => (
                  <div key={idx.code} className="flex items-center gap-1.5 text-xs">
                    <span className="text-white/70">{idx.name}</span>
                    <span className={`font-mono font-medium ${changeColor(idx.change_percent)}`}>
                      {idx.price.toFixed(2)}
                    </span>
                    <span className={`font-mono ${changeColor(idx.change_percent)}`}>
                      {idx.change_percent >= 0 ? '+' : ''}
                      {idx.change_percent.toFixed(2)}%
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
          {(overview?.indices ?? []).length > 0 ? (
            <EChart option={indexLineOption} height={320} loading={loading} />
          ) : loading ? (
            <IndexChartSkeleton />
          ) : (
            <EmptyState text="暂无指数数据" />
          )}
        </GlassCard>

        {/* ═══════ 2. 涨跌家数柱状图 ═══════ */}
        <GlassCard className="p-5">
          <div className="flex items-center gap-2 mb-4">
            <BarChart3 className="w-4 h-4 text-purple-400" />
            <h2 className="text-white font-semibold text-sm">涨跌家数</h2>
          </div>
          {(advCount + decCount + flatCount) > 0 ? (
            <>
              <EChart option={breadthBarOption} height={220} loading={loading} />
              <div className="flex items-center justify-center gap-6 mt-3 text-xs">
                <div className="flex items-center gap-1.5">
                  <span className="w-2.5 h-2.5 rounded-sm bg-red-500" />
                  <span className="text-red-400 font-mono">{advCount}</span>
                  <span className="text-white/40">上涨</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="w-2.5 h-2.5 rounded-sm bg-green-500" />
                  <span className="text-green-400 font-mono">{decCount}</span>
                  <span className="text-white/40">下跌</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <span className="w-2.5 h-2.5 rounded-sm bg-gray-500" />
                  <span className="text-white/60 font-mono">{flatCount}</span>
                  <span className="text-white/40">平盘</span>
                </div>
              </div>
            </>
          ) : loading ? (
            <BreadthChartSkeleton />
          ) : (
            <EmptyState text="暂无涨跌数据" />
          )}
        </GlassCard>

        {/* ═══════ 3. 板块热力图 ═══════ */}
        <GlassCard className="p-5">
          <div className="flex items-center gap-2 mb-4">
            <Layers className="w-4 h-4 text-amber-400" />
            <h2 className="text-white font-semibold text-sm">板块热力图</h2>
          </div>
          {sectors.length > 0 ? (
            <EChart option={sectorTreemapOption} height={280} loading={loading} />
          ) : loading ? (
            <TreemapSkeleton />
          ) : (
            <EmptyState text="暂无板块数据" />
          )}
        </GlassCard>

        {/* ═══════ 4. 信号摘要卡片 ═══════ */}
        <GlassCard className="p-5">
          <div className="flex items-center gap-2 mb-4">
            <Zap className="w-4 h-4 text-yellow-400" />
            <h2 className="text-white font-semibold text-sm">信号摘要</h2>
            {signals.length > 0 && (
              <span className="ml-auto bg-yellow-500/15 text-yellow-400 text-xs font-mono px-2 py-0.5 rounded-full">
                {signals.length} 条信号
              </span>
            )}
          </div>
          {signals.length > 0 ? (
            <div className="space-y-2 max-h-[280px] overflow-y-auto pr-1 custom-scrollbar">
              {/* Signal type summary */}
              <div className="grid grid-cols-3 gap-2 mb-3">
                {(() => {
                  const typeMap: Record<string, number> = {};
                  signals.forEach((s) => {
                    typeMap[s.level] = (typeMap[s.level] || 0) + 1;
                  });
                  const colorMap: Record<string, string> = {
                    high: 'bg-red-500/15 text-red-400 border-red-500/20',
                    medium: 'bg-amber-500/15 text-amber-400 border-amber-500/20',
                    low: 'bg-blue-500/15 text-blue-400 border-blue-500/20',
                  };
                  return Object.entries(typeMap).map(([level, count]) => (
                    <div
                      key={level}
                      className={`flex flex-col items-center py-2 rounded-lg border ${
                        colorMap[level] ?? 'bg-white/5 text-white/60 border-white/10'
                      }`}
                    >
                      <span className="text-lg font-bold font-mono">{count}</span>
                      <span className="text-[10px] mt-0.5 opacity-70">{level}</span>
                    </div>
                  ));
                })()}
              </div>
              {/* Signal list */}
              {signals.slice(0, 8).map((s, i) => (
                <div
                  key={`${s.code}-${i}`}
                  className="flex items-start gap-3 px-3 py-2.5 rounded-xl bg-white/[0.03] border border-white/[0.06] hover:bg-white/[0.06] transition-colors"
                >
                  <div className="flex-shrink-0 mt-0.5">
                    <Bell
                      className={`w-3.5 h-3.5 ${
                        s.level === 'high'
                          ? 'text-red-400'
                          : s.level === 'medium'
                          ? 'text-amber-400'
                          : 'text-blue-400'
                      }`}
                    />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-mono text-blue-400">{s.code}</span>
                      <span className="text-xs text-white/80 truncate">{s.name}</span>
                      <span className="text-[10px] text-white/40 ml-auto flex-shrink-0">
                        {formatTime(s.timestamp)}
                      </span>
                    </div>
                    <p className="text-xs text-white/50 mt-0.5 line-clamp-1">{s.message}</p>
                  </div>
                </div>
              ))}
            </div>
          ) : loading ? (
            <SignalSkeleton />
          ) : (
            <EmptyState text="近7日暂无信号" />
          )}
        </GlassCard>

        {/* ═══════ 5. 活动时间线 ═══════ */}
        <GlassCard className="p-5">
          <div className="flex items-center gap-2 mb-4">
            <Clock className="w-4 h-4 text-cyan-400" />
            <h2 className="text-white font-semibold text-sm">活动时间线</h2>
          </div>
          {activity.length > 0 ? (
            <div className="space-y-0 max-h-[320px] overflow-y-auto pr-1 custom-scrollbar">
              {activity.slice(0, 12).map((entry, i) => {
                const isToday =
                  new Date(entry.timestamp).toDateString() === new Date().toDateString();
                const prevDate =
                  i > 0
                    ? new Date(activity[i - 1].timestamp).toDateString()
                    : '';
                const thisDate = new Date(entry.timestamp).toDateString();
                const showDateHeader = thisDate !== prevDate;

                return (
                  <div key={`${entry.timestamp}-${i}`}>
                    {showDateHeader && (
                      <div className="flex items-center gap-2 py-2">
                        <span className="text-[10px] font-mono text-white/40 bg-white/5 px-2 py-0.5 rounded-full">
                          {isToday ? '今天' : formatDate(entry.timestamp)}
                        </span>
                        <div className="flex-1 h-px bg-white/[0.06]" />
                      </div>
                    )}
                    <div className="flex items-start gap-3 group">
                      {/* Timeline dot + line */}
                      <div className="flex flex-col items-center flex-shrink-0">
                        <div className="w-2 h-2 rounded-full bg-cyan-400/60 group-hover:bg-cyan-400 transition-colors mt-1.5" />
                        <div className="w-px flex-1 min-h-[20px] bg-white/[0.06]" />
                      </div>
                      {/* Content */}
                      <div className="pb-3 flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-white/80 font-medium">
                            {entry.action}
                          </span>
                          <ChevronRight className="w-3 h-3 text-white/20" />
                          <span className="text-xs text-white/40 font-mono flex-shrink-0">
                            {formatTime(entry.timestamp)}
                          </span>
                        </div>
                        <p className="text-xs text-white/50 mt-0.5 line-clamp-2">
                          {entry.detail}
                        </p>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : loading ? (
            <TimelineSkeleton />
          ) : (
            <EmptyState text="暂无活动记录" />
          )}
        </GlassCard>
      </div>

      {/* ─── Footer ─── */}
      <div className="mt-5 flex items-center justify-between text-xs text-white/30">
        <span>
          {lastUpdated && `上次刷新: ${lastUpdated.toLocaleTimeString('zh-CN')}`}
        </span>
        <span>{autoRefresh && `每 ${REFRESH_INTERVAL / 1000} 秒自动刷新`}</span>
      </div>
    </div>
  );
}
