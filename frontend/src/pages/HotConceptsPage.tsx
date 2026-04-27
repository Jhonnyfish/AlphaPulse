import { useState, useEffect, useCallback, useMemo } from 'react';
import { hotConceptsApi, type HotConcept } from '@/lib/api';
import EmptyState from '@/components/EmptyState';
import ErrorState from '@/components/ErrorState';
import { Flame, RefreshCw, TrendingUp, TrendingDown, ChevronDown, ChevronUp, Star, Link2 } from 'lucide-react';
import { TrendingUp as TrendingUpIcon } from 'lucide-react';
import { SkeletonGridCard, SkeletonList, SkeletonInlineTable } from '@/components/ui/Skeleton';
import EChart from '@/components/charts/EChart';
import type { EChartsOption } from 'echarts';

interface ConceptStock {
  code: string;
  name: string;
  change_pct: number;
  price: number;
}

interface WatchlistOverlap {
  code: string;
  name: string;
  concepts: string[];
}

export default function HotConceptsPage() {
  const [concepts, setConcepts] = useState<HotConcept[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [expandedCode, setExpandedCode] = useState<string | null>(null);
  const [expandedStocks, setExpandedStocks] = useState<ConceptStock[]>([]);
  const [expandingLoading, setExpandingLoading] = useState(false);
  const [overlap, setOverlap] = useState<WatchlistOverlap[]>([]);
  const [overlapLoading, setOverlapLoading] = useState(true);

  const fetchConcepts = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await hotConceptsApi.list();
      const d = res.data;
      setConcepts(Array.isArray(d) ? d : Array.isArray(d.concepts) ? d.concepts : []);
    } catch {
      setError('加载热门概念失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchOverlap = useCallback(async () => {
    setOverlapLoading(true);
    try {
      const res = await hotConceptsApi.watchlistOverlap();
      const data = res.data;
      setOverlap(Array.isArray(data) ? data : []);
    } catch {
      // ignore
    } finally {
      setOverlapLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConcepts();
    fetchOverlap();
  }, [fetchConcepts, fetchOverlap]);

  const toggleExpand = useCallback(async (code: string) => {
    if (expandedCode === code) {
      setExpandedCode(null);
      setExpandedStocks([]);
      return;
    }
    setExpandedCode(code);
    setExpandingLoading(true);
    try {
      const res = await hotConceptsApi.stocks(code);
      const data = res.data;
      setExpandedStocks(Array.isArray(data) ? data : []);
    } catch {
      setExpandedStocks([]);
    } finally {
      setExpandingLoading(false);
    }
  }, [expandedCode]);

  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

  const sorted = [...concepts].sort((a, b) => b.change_pct - a.change_pct);

  // ── Mock trend data for top concepts ──────────────────────
  const trendColors = [
    '#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#a855f7',
    '#ec4899', '#22d3ee', '#f97316',
  ];

  const generateMockDates = useCallback((days: number) => {
    const dates: string[] = [];
    const now = new Date();
    for (let i = days - 1; i >= 0; i--) {
      const d = new Date(now);
      d.setDate(d.getDate() - i);
      dates.push(
        `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`,
      );
    }
    return dates;
  }, []);

  const mockTrendData = useMemo(() => {
    if (concepts.length === 0) return null;
    const topConcepts = [...concepts]
      .sort((a, b) => (b.rise_count + b.fall_count) - (a.rise_count + a.fall_count))
      .slice(0, 6);
    const dates = generateMockDates(10);

    return {
      dates,
      series: topConcepts.map((concept, idx) => {
        const baseHeat = 30 + Math.random() * 40;
        const volatility = 5 + Math.random() * 10;
        const values: number[] = [];
        let current = baseHeat;
        for (let i = 0; i < dates.length; i++) {
          const drift =
            i === dates.length - 1
              ? concept.change_pct * 0.5
              : (Math.random() - 0.48) * volatility;
          current = Math.max(0, Math.min(100, current + drift));
          values.push(Math.round(current * 10) / 10);
        }
        return {
          name: concept.name,
          values,
          color: trendColors[idx % trendColors.length],
        };
      }),
    };
  }, [concepts, generateMockDates]);

  const trendChartOption = useMemo<EChartsOption | null>(() => {
    if (!mockTrendData) return null;
    return {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(15, 23, 42, 0.92)',
        borderColor: 'rgba(148, 163, 184, 0.2)',
        textStyle: { color: '#e2e8f0', fontSize: 12 },
        axisPointer: { type: 'cross', crossStyle: { color: '#64748b' } },
      },
      legend: {
        type: 'scroll',
        top: 0,
        textStyle: { color: '#94a3b8', fontSize: 11 },
        pageTextStyle: { color: '#94a3b8' },
        pageIconColor: '#3b82f6',
        pageIconInactiveColor: '#64748b',
      },
      grid: { top: 40, left: 12, right: 16, bottom: 8, containLabel: true },
      xAxis: {
        type: 'category',
        data: mockTrendData.dates,
        boundaryGap: false,
        axisLabel: { color: '#94a3b8', fontSize: 11 },
        axisLine: { lineStyle: { color: '#334155' } },
        axisTick: { show: false },
      },
      yAxis: {
        type: 'value',
        name: '热度',
        nameTextStyle: { color: '#94a3b8', fontSize: 11 },
        axisLabel: { color: '#94a3b8', fontSize: 11 },
        axisLine: { show: false },
        splitLine: { lineStyle: { color: 'rgba(51, 65, 85, 0.4)' } },
      },
      series: mockTrendData.series.map((s) => ({
        name: s.name,
        type: 'line' as const,
        data: s.values,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 2, color: s.color },
        itemStyle: { color: s.color },
        emphasis: { focus: 'series' as const, lineStyle: { width: 3 } },
        areaStyle: {
          color: {
            type: 'linear' as const,
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: s.color + '20' },
              { offset: 1, color: s.color + '02' },
            ],
          },
        },
      })),
    };
  }, [mockTrendData]);

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Flame className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <div>
            <h1 className="text-xl font-bold">热门概念</h1>
            <p className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>
              当前市场热门概念板块
            </p>
          </div>
        </div>
        <button
          onClick={() => { fetchConcepts(); fetchOverlap(); }}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors"
          style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {/* Error */}
      {error && (
        <div className="mb-4">
          <ErrorState
            title="加载失败"
            description={error}
            onRetry={() => { setError(''); fetchConcepts(); fetchOverlap(); }}
          />
        </div>
      )}

      {/* Loading */}
      {loading && concepts.length === 0 ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <SkeletonGridCard key={i} />
          ))}
        </div>
      ) : concepts.length === 0 ? (
        <EmptyState
          icon={Flame}
          title="暂无热门概念"
          description="暂无概念热点数据"
        />
      ) : (
        /* Concept card grid */
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 mb-6">
          {sorted.map((concept) => {
            const isExpanded = expandedCode === concept.code;
            const pct = concept.change_pct;
            const Icon = pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Flame;

            return (
              <div
                key={concept.code}
                className="rounded-lg border overflow-hidden transition-all"
                style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
              >
                {/* Card header */}
                <button
                  onClick={() => toggleExpand(concept.code)}
                  className="w-full text-left p-4 transition-colors hover:bg-[var(--color-bg-hover)]"
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Icon className="w-4 h-4" style={{ color: changeColor(pct) }} />
                      <span className="font-medium text-sm">{concept.name}</span>
                    </div>
                    {isExpanded ? (
                      <ChevronUp className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
                    ) : (
                      <ChevronDown className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
                    )}
                  </div>
                  <div className="flex items-center justify-between">
                    <span
                      className="font-mono text-sm font-bold"
                      style={{ color: changeColor(pct) }}
                    >
                      {pct >= 0 ? '+' : ''}{pct.toFixed(2)}%
                    </span>
                    <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {(concept.rise_count + concept.fall_count)} 只股票
                    </span>
                  </div>

                  {/* Leader info */}
                  {concept.leader_stock && (
                    <div
                      className="mt-2 flex items-center gap-1.5 text-xs rounded-md px-2 py-1"
                      style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}
                    >
                      <Star className="w-3 h-3" style={{ color: 'var(--color-accent)' }} />
                      <span>龙头: {concept.leader_stock.name}</span>
                      <span className="font-mono" style={{ color: changeColor(concept.leader_stock.change_pct) }}>
                        {concept.leader_stock.change_pct >= 0 ? '+' : ''}{concept.leader_stock.change_pct.toFixed(2)}%
                      </span>
                    </div>
                  )}
                </button>

                {/* Expanded stocks */}
                {isExpanded && (
                  <div className="border-t" style={{ borderColor: 'var(--color-border)' }}>
                    {expandingLoading ? (
                      <div className="p-4">
                        <SkeletonInlineTable rows={4} columns={4} />
                      </div>
                    ) : expandedStocks.length === 0 ? (
                      <div className="text-center py-6 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                        暂无成分股数据
                      </div>
                    ) : (
                      <div className="max-h-[240px] overflow-y-auto">
                        {expandedStocks.map((stock, idx) => (
                          <div
                            key={`${stock.code}-${idx}`}
                            className="flex items-center justify-between px-4 py-2 border-b last:border-b-0 transition-colors hover:bg-[var(--color-bg-hover)]"
                            style={{ borderColor: 'var(--color-border)' }}
                          >
                            <div className="flex items-center gap-2 min-w-0">
                              <span className="text-xs font-mono font-medium" style={{ color: 'var(--color-accent)' }}>
                                {stock.code}
                              </span>
                              <span className="text-xs truncate" style={{ color: 'var(--color-text-primary)' }}>
                                {stock.name}
                              </span>
                            </div>
                            <div className="flex items-center gap-3 flex-shrink-0">
                              {stock.price > 0 && (
                                <span className="text-xs font-mono" style={{ color: 'var(--color-text-secondary)' }}>
                                  {stock.price.toFixed(2)}
                                </span>
                              )}
                              <span
                                className="text-xs font-mono font-medium"
                                style={{ color: changeColor(stock.change_pct) }}
                              >
                                {stock.change_pct >= 0 ? '+' : ''}{stock.change_pct.toFixed(2)}%
                              </span>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Concept Trend Chart */}
      {trendChartOption && concepts.length > 0 && (
        <div
          className="rounded-xl border mb-6 overflow-hidden"
          style={{
            background: 'linear-gradient(135deg, rgba(15,23,42,0.85), rgba(30,41,59,0.65))',
            borderColor: 'var(--color-border)',
            backdropFilter: 'blur(12px)',
          }}
        >
          <div className="flex items-center gap-2 px-5 pt-4 pb-1">
            <TrendingUpIcon className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
            <h2 className="text-sm font-bold">概念热度趋势</h2>
            <span className="text-xs ml-1" style={{ color: 'var(--color-text-muted)' }}>
              Top {mockTrendData?.series.length ?? 0} · 近 10 日
            </span>
          </div>
          <div className="px-3 pb-3">
            <EChart option={trendChartOption} height={350} />
          </div>
        </div>
      )}

      {/* Watchlist-concept overlap section */}
      <div className="mt-6">
        <div className="flex items-center gap-2 mb-3">
          <Link2 className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <h2 className="text-sm font-bold">自选股 × 热门概念</h2>
        </div>

        {overlapLoading ? (
          <SkeletonList rows={3} />
        ) : overlap.length === 0 ? (
          <div
            className="text-center py-8 rounded-lg border"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
              暂无自选股与热门概念的交集数据
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {overlap.map((item, idx) => (
              <div
                key={`${item.code}-${idx}`}
                className="rounded-lg border p-3 transition-colors hover:bg-[var(--color-bg-hover)]"
                style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
              >
                <div className="flex items-center gap-2 mb-2">
                  <span className="font-mono text-xs font-medium" style={{ color: 'var(--color-accent)' }}>
                    {item.code}
                  </span>
                  <span className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
                    {item.name}
                  </span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {item.concepts.map((c) => (
                    <span
                      key={c}
                      className="text-xs px-2 py-0.5 rounded-full"
                      style={{ background: 'var(--color-bg-hover)', color: 'var(--color-accent)' }}
                    >
                      {c}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
