import { useState, useEffect, useCallback } from 'react';
import { useView } from '@/lib/ViewContext';
import api from '@/lib/api';
import type { RankingItem, RankingSummary, RankingResponse } from '@/lib/api';
import ReactECharts from '@/components/charts/ReactECharts';
import {
  Trophy, RefreshCw, TrendingUp, TrendingDown,
  ChevronUp, ChevronDown, Award, AlertTriangle,
  Star, Shield, ChevronRight,
} from 'lucide-react';

/* ---- dimension labels ---- */
const DIM_LABELS: Record<string, string> = {
  order_flow: '委托流',
  volume_price: '量价',
  valuation: '估值',
  volatility: '波动',
  money_flow: '资金流',
  technical: '技术面',
  sector: '板块',
  sentiment: '情绪',
};

const DIMENSIONS = Object.keys(DIM_LABELS);

type SortKey = 'rank' | 'overall_score' | 'change_pct' | 'price' | string;
type SortDir = 'asc' | 'desc';

/* ---- helpers ---- */
function signalColor(signal: string): string {
  if (!signal) return 'var(--color-text-muted)';
  const s = signal.toLowerCase();
  if (s.includes('strong_buy') || s.includes('buy') || s.includes('bullish'))
    return 'var(--color-danger)';
  if (s.includes('strong_sell') || s.includes('sell') || s.includes('bearish'))
    return 'var(--color-success)';
  return 'var(--color-warning)';
}

function signalLabel(signal: string): string {
  if (!signal) return '—';
  const map: Record<string, string> = {
    strong_buy: '强烈看多',
    buy: '看多',
    bullish: '偏多',
    hold: '中性',
    neutral: '中性',
    bearish: '偏空',
    sell: '看空',
    strong_sell: '强烈看空',
  };
  return map[signal.toLowerCase()] || signal;
}

function scoreColor(score: number): string {
  if (score >= 75) return 'var(--color-danger)';
  if (score >= 55) return 'var(--color-warning)';
  if (score >= 35) return 'var(--color-text-secondary)';
  return 'var(--color-success)';
}

function scoreBg(score: number): string {
  if (score >= 75) return 'rgba(239,68,68,0.12)';
  if (score >= 55) return 'rgba(234,179,8,0.12)';
  if (score >= 35) return 'rgba(148,163,184,0.08)';
  return 'rgba(34,197,94,0.12)';
}

/* ---- localStorage cache ---- */
const CACHE_KEY = 'ranking_cache';
const CACHE_TTL = 30 * 60 * 1000; // 30 minutes

interface RankingCache {
  items: RankingItem[];
  summary: RankingSummary | null;
  fetched_at: string;
  ts: number;
}

function loadCache(): RankingCache | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (!raw) return null;
    const c: RankingCache = JSON.parse(raw);
    return c;
  } catch {
    return null;
  }
}

function saveCache(items: RankingItem[], summary: RankingSummary | null, fetched_at: string) {
  try {
    const c: RankingCache = { items, summary, fetched_at, ts: Date.now() };
    localStorage.setItem(CACHE_KEY, JSON.stringify(c));
  } catch { /* quota exceeded, ignore */ }
}

function isCacheStale(c: RankingCache): boolean {
  return Date.now() - c.ts > CACHE_TTL;
}

/* ---- component ---- */
export default function RankingPage() {
  const { navigate } = useView();
  const [data, setData] = useState<RankingItem[]>([]);
  const [summary, setSummary] = useState<RankingSummary | null>(null);
  const [fetchedAt, setFetchedAt] = useState('');
  const [loading, setLoading] = useState(true);
  const [fromCache, setFromCache] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');
  const [sortKey, setSortKey] = useState<SortKey>('rank');
  const [sortDir, setSortDir] = useState<SortDir>('asc');
  const [expandedRow, setExpandedRow] = useState<string | null>(null);

  const fetchData = useCallback((silent = false) => {
    if (!silent && data.length === 0) setLoading(true);
    if (silent) setRefreshing(true);
    setError('');
    api
      .get<RankingResponse>('/watchlist-ranking')
      .then((res) => {
        if (res.data.ok) {
          setData(res.data.items);
          setSummary(res.data.summary);
          setFetchedAt(res.data.fetched_at);
          setFromCache(false);
          saveCache(res.data.items, res.data.summary, res.data.fetched_at);
        } else {
          if (!silent) setError(res.data.error || '加载排名数据失败');
        }
      })
      .catch(() => { if (!silent) setError('加载排名数据失败，请稍后重试'); })
      .finally(() => { setLoading(false); setRefreshing(false); });
  }, [data.length]);

  useEffect(() => {
    // Load cached data first for instant display
    const cached = loadCache();
    if (cached && cached.items.length > 0) {
      setData(cached.items);
      setSummary(cached.summary);
      setFetchedAt(cached.fetched_at);
      setFromCache(true);
      setLoading(false);
      // Refresh in background if stale
      if (isCacheStale(cached)) {
        fetchData(true);
      }
    } else {
      fetchData();
    }
  }, [fetchData]);

  /* ---- sorting ---- */
  const sorted = [...data].sort((a, b) => {
    let va: number, vb: number;
    if (sortKey === 'rank') {
      va = a.rank;
      vb = b.rank;
    } else if (sortKey === 'overall_score') {
      va = a.overall_score;
      vb = b.overall_score;
    } else if (sortKey === 'change_pct') {
      va = a.change_pct;
      vb = b.change_pct;
    } else if (sortKey === 'price') {
      va = a.price;
      vb = b.price;
    } else if (DIMENSIONS.includes(sortKey)) {
      va = a.dimension_scores?.[sortKey] ?? 0;
      vb = b.dimension_scores?.[sortKey] ?? 0;
    } else {
      va = 0;
      vb = 0;
    }
    return sortDir === 'asc' ? va - vb : vb - va;
  });

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortKey(key);
      setSortDir(key === 'rank' ? 'asc' : 'desc');
    }
  };

  const SortIcon = ({ col }: { col: SortKey }) => {
    if (sortKey !== col) return null;
    return sortDir === 'asc' ? (
      <ChevronUp className="w-3 h-3 inline ml-0.5" />
    ) : (
      <ChevronDown className="w-3 h-3 inline ml-0.5" />
    );
  };

  /* ---- radar chart for top 5 ---- */
  const getRadarOption = () => {
    const top5 = data.slice(0, 5);
    if (top5.length === 0) return {};
    const colors = ['#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#a855f7'];
    return {
      tooltip: {},
      legend: {
        bottom: 0,
        textStyle: { color: '#94a3b8', fontSize: 11 },
        itemWidth: 12,
        itemHeight: 12,
      },
      radar: {
        indicator: DIMENSIONS.map((d) => ({
          name: DIM_LABELS[d],
          max: 100,
        })),
        shape: 'polygon',
        splitNumber: 4,
        axisName: { color: '#94a3b8', fontSize: 11 },
        splitLine: { lineStyle: { color: 'rgba(148,163,184,0.15)' } },
        splitArea: { show: false },
        axisLine: { lineStyle: { color: 'rgba(148,163,184,0.15)' } },
      },
      series: [
        {
          type: 'radar',
          data: top5.map((item, i) => ({
            value: DIMENSIONS.map((d) => item.dimension_scores?.[d] ?? 0),
            name: item.name || item.code,
            lineStyle: { color: colors[i], width: 2 },
            areaStyle: { color: colors[i], opacity: 0.08 },
            itemStyle: { color: colors[i] },
            symbol: 'circle',
            symbolSize: 4,
          })),
        },
      ],
    };
  };

  /* ---- score distribution bar ---- */
  const getDistOption = () => {
    const buckets = [
      { label: '0-20', min: 0, max: 20, count: 0 },
      { label: '20-40', min: 20, max: 40, count: 0 },
      { label: '40-60', min: 40, max: 60, count: 0 },
      { label: '60-80', min: 60, max: 80, count: 0 },
      { label: '80-100', min: 80, max: 100, count: 0 },
    ];
    data.forEach((item) => {
      const s = item.overall_score;
      const b = buckets.find((bk) => s >= bk.min && s < bk.max) || buckets[buckets.length - 1];
      b.count++;
    });
    return {
      tooltip: { trigger: 'axis' as const },
      grid: { top: 10, right: 16, bottom: 28, left: 40 },
      xAxis: {
        type: 'category' as const,
        data: buckets.map((b) => b.label),
        axisLabel: { color: '#94a3b8', fontSize: 11 },
        axisLine: { lineStyle: { color: 'rgba(148,163,184,0.15)' } },
      },
      yAxis: {
        type: 'value' as const,
        axisLabel: { color: '#94a3b8', fontSize: 11 },
        splitLine: { lineStyle: { color: 'rgba(148,163,184,0.1)' } },
      },
      series: [
        {
          type: 'bar',
          data: buckets.map((b) => ({
            value: b.count,
            itemStyle: {
              color:
                b.min >= 80
                  ? '#ef4444'
                  : b.min >= 60
                    ? '#f59e0b'
                    : b.min >= 40
                      ? '#3b82f6'
                      : b.min >= 20
                        ? '#22c55e'
                        : '#64748b',
              borderRadius: [4, 4, 0, 0],
            },
          })),
          barWidth: '50%',
        },
      ],
    };
  };

  /* ---- loading skeleton ---- */
  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Trophy className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">综合排名</h1>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          {Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className="glass-panel rounded-xl p-4 animate-pulse"
              style={{ height: '100px' }}
            />
          ))}
        </div>
        <div className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '300px' }} />
      </div>
    );
  }

  /* ---- error state ---- */
  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Trophy className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">综合排名</h1>
        </div>
        <div
          className="text-sm px-4 py-3 rounded-lg flex items-center gap-3"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
        >
          <AlertTriangle className="w-4 h-4 flex-shrink-0" />
          <span className="flex-1">{error}</span>
          <button onClick={fetchData} className="underline whitespace-nowrap">
            重试
          </button>
        </div>
      </div>
    );
  }

  /* ---- empty state ---- */
  if (data.length === 0) {
    return (
      <div>
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-2">
            <Trophy className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
            <h1 className="text-xl font-bold">综合排名</h1>
          </div>
          <button
            onClick={fetchData}
            className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
          >
            <RefreshCw className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
          </button>
        </div>
        <div
          className="glass-panel rounded-xl p-12 text-center"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <Trophy
            className="w-12 h-12 mx-auto mb-3"
            style={{ color: 'var(--color-text-muted)', opacity: 0.4 }}
          />
          <div className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
            自选股列表为空，请先添加股票到自选股
          </div>
        </div>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Trophy className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">综合排名</h1>
          <span
            className="text-xs px-2 py-0.5 rounded-full"
            style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}
          >
            {data.length} 只
          </span>
          {fromCache && (
            <span
              className="text-xs px-2 py-0.5 rounded-full"
              style={{ background: 'rgba(245,158,11,0.12)', color: '#f59e0b' }}
            >
              缓存
            </span>
          )}
        </div>
        <button
          onClick={() => { localStorage.removeItem(CACHE_KEY); fetchData(true); }}
          disabled={refreshing}
          className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
        >
          <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} style={{ color: refreshing ? '#3b82f6' : 'var(--color-text-muted)' }} />
        </button>
      </div>

      {/* Summary cards */}
      {summary && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          {/* Avg score */}
          <div
            className="glass-panel rounded-xl p-4"
            style={{ borderColor: 'var(--color-border)' }}
          >
            <div className="flex items-center gap-2 mb-1">
              <Star className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
                平均评分
              </span>
            </div>
            <div
              className="text-2xl font-bold font-mono"
              style={{ color: scoreColor(summary.avg_score) }}
            >
              {summary.avg_score.toFixed(1)}
            </div>
          </div>

          {/* Best */}
          <div
            className="glass-panel rounded-xl p-4"
            style={{ borderColor: 'var(--color-border)' }}
          >
            <div className="flex items-center gap-2 mb-1">
              <Award className="w-4 h-4" style={{ color: 'var(--color-danger)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
                最佳
              </span>
            </div>
            {summary.best ? (
              <>
                <div className="text-lg font-bold" style={{ color: 'var(--color-danger)' }}>
                  {summary.best.score}
                </div>
                <div
                  className="text-xs truncate"
                  style={{ color: 'var(--color-text-secondary)' }}
                >
                  {summary.best.name}（{summary.best.code}）
                </div>
              </>
            ) : (
              <div className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
                —
              </div>
            )}
          </div>

          {/* Worst */}
          <div
            className="glass-panel rounded-xl p-4"
            style={{ borderColor: 'var(--color-border)' }}
          >
            <div className="flex items-center gap-2 mb-1">
              <Shield className="w-4 h-4" style={{ color: 'var(--color-success)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
                最弱
              </span>
            </div>
            {summary.worst ? (
              <>
                <div className="text-lg font-bold" style={{ color: 'var(--color-success)' }}>
                  {summary.worst.score}
                </div>
                <div
                  className="text-xs truncate"
                  style={{ color: 'var(--color-text-secondary)' }}
                >
                  {summary.worst.name}（{summary.worst.code}）
                </div>
              </>
            ) : (
              <div className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
                —
              </div>
            )}
          </div>

          {/* Count */}
          <div
            className="glass-panel rounded-xl p-4"
            style={{ borderColor: 'var(--color-border)' }}
          >
            <div className="flex items-center gap-2 mb-1">
              <Trophy className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
                分析数量
              </span>
            </div>
            <div
              className="text-2xl font-bold font-mono"
              style={{ color: 'var(--color-text-primary)' }}
            >
              {summary.count}
            </div>
          </div>
        </div>
      )}

      {/* Charts row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
        {/* Radar: top 5 */}
        <div
          className="glass-panel rounded-xl p-4"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <h3
            className="text-sm font-medium mb-3"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            Top 5 多维雷达
          </h3>
          <div className="h-64">
            <ReactECharts option={getRadarOption()} style={{ height: '100%' }} />
          </div>
        </div>

        {/* Score distribution */}
        <div
          className="glass-panel rounded-xl p-4"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <h3
            className="text-sm font-medium mb-3"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            评分分布
          </h3>
          <div className="h-64">
            <ReactECharts option={getDistOption()} style={{ height: '100%' }} />
          </div>
        </div>
      </div>

      {/* Ranking table */}
      <div
        className="glass-panel rounded-xl overflow-hidden"
        style={{ borderColor: 'var(--color-border)' }}
      >
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr
                style={{
                  borderBottom: '1px solid var(--color-border)',
                  background: 'rgba(148,163,184,0.04)',
                }}
              >
                <th
                  className="px-3 py-3 text-left font-medium cursor-pointer select-none whitespace-nowrap"
                  style={{ color: 'var(--color-text-muted)' }}
                  onClick={() => toggleSort('rank')}
                >
                  排名 <SortIcon col="rank" />
                </th>
                <th
                  className="px-3 py-3 text-left font-medium cursor-pointer select-none whitespace-nowrap"
                  style={{ color: 'var(--color-text-muted)' }}
                  onClick={() => toggleSort('overall_score')}
                >
                  评分 <SortIcon col="overall_score" />
                </th>
                <th
                  className="px-3 py-3 text-left font-medium whitespace-nowrap"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  股票
                </th>
                <th
                  className="px-3 py-3 text-left font-medium cursor-pointer select-none whitespace-nowrap"
                  style={{ color: 'var(--color-text-muted)' }}
                  onClick={() => toggleSort('price')}
                >
                  价格 <SortIcon col="price" />
                </th>
                <th
                  className="px-3 py-3 text-left font-medium cursor-pointer select-none whitespace-nowrap"
                  style={{ color: 'var(--color-text-muted)' }}
                  onClick={() => toggleSort('change_pct')}
                >
                  涨跌幅 <SortIcon col="change_pct" />
                </th>
                <th
                  className="px-3 py-3 text-left font-medium whitespace-nowrap"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  信号
                </th>
                {DIMENSIONS.map((d) => (
                  <th
                    key={d}
                    className="px-2 py-3 text-center font-medium cursor-pointer select-none whitespace-nowrap text-xs"
                    style={{ color: 'var(--color-text-muted)' }}
                    onClick={() => toggleSort(d)}
                  >
                    {DIM_LABELS[d]} <SortIcon col={d} />
                  </th>
                ))}
                <th
                  className="px-3 py-3 text-center font-medium whitespace-nowrap"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  详情
                </th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((item) => (
                <>
                  <tr
                    key={item.code}
                    className="transition-colors cursor-pointer"
                    style={{ borderBottom: '1px solid rgba(148,163,184,0.06)' }}
                    onMouseEnter={(e) =>
                      (e.currentTarget.style.background = 'rgba(148,163,184,0.04)')
                    }
                    onMouseLeave={(e) => (e.currentTarget.style.background = '')}
                    onClick={() => navigate('analyze', { code: item.code })}
                  >
                    {/* Rank */}
                    <td className="px-3 py-3">
                      <span
                        className="inline-flex items-center justify-center w-7 h-7 rounded-full text-xs font-bold"
                        style={{
                          background:
                            item.rank <= 3
                              ? 'linear-gradient(135deg, #f59e0b, #ef4444)'
                              : 'var(--color-bg-hover)',
                          color: item.rank <= 3 ? '#fff' : 'var(--color-text-secondary)',
                        }}
                      >
                        {item.rank}
                      </span>
                    </td>

                    {/* Score */}
                    <td className="px-3 py-3">
                      <div
                        className="inline-flex items-center justify-center px-2.5 py-1 rounded-lg font-mono font-bold text-sm"
                        style={{
                          background: scoreBg(item.overall_score),
                          color: scoreColor(item.overall_score),
                        }}
                      >
                        {item.overall_score}
                      </div>
                    </td>

                    {/* Stock info */}
                    <td className="px-3 py-3">
                      <div className="font-medium" style={{ color: 'var(--color-text-primary)' }}>
                        {item.name || '—'}
                      </div>
                      <div
                        className="text-xs font-mono"
                        style={{ color: 'var(--color-text-muted)' }}
                      >
                        {item.code}
                      </div>
                    </td>

                    {/* Price */}
                    <td className="px-3 py-3 font-mono" style={{ color: 'var(--color-text-primary)' }}>
                      {item.price > 0 ? item.price.toFixed(2) : '—'}
                    </td>

                    {/* Change % */}
                    <td className="px-3 py-3">
                      <span
                        className="inline-flex items-center gap-1 font-mono font-medium"
                        style={{
                          color:
                            item.change_pct > 0
                              ? 'var(--color-danger)'
                              : item.change_pct < 0
                                ? 'var(--color-success)'
                                : 'var(--color-text-muted)',
                        }}
                      >
                        {item.change_pct > 0 ? (
                          <TrendingUp className="w-3 h-3" />
                        ) : item.change_pct < 0 ? (
                          <TrendingDown className="w-3 h-3" />
                        ) : null}
                        {item.change_pct > 0 ? '+' : ''}
                        {item.change_pct.toFixed(2)}%
                      </span>
                    </td>

                    {/* Signal */}
                    <td className="px-3 py-3">
                      <span
                        className="text-xs px-2 py-0.5 rounded-full"
                        style={{
                          background: `${signalColor(item.overall_signal)}18`,
                          color: signalColor(item.overall_signal),
                        }}
                      >
                        {signalLabel(item.overall_signal)}
                      </span>
                    </td>

                    {/* Dimension mini bars */}
                    {DIMENSIONS.map((d) => {
                      const val = item.dimension_scores?.[d] ?? 0;
                      return (
                        <td key={d} className="px-2 py-3 text-center">
                          <div className="flex flex-col items-center gap-0.5">
                            <span
                              className="text-xs font-mono font-medium"
                              style={{ color: scoreColor(val) }}
                            >
                              {val.toFixed(0)}
                            </span>
                            <div
                              className="w-10 h-1 rounded-full overflow-hidden"
                              style={{ background: 'rgba(148,163,184,0.1)' }}
                            >
                              <div
                                className="h-full rounded-full transition-all duration-300"
                                style={{
                                  width: `${val}%`,
                                  background: scoreColor(val),
                                }}
                              />
                            </div>
                          </div>
                        </td>
                      );
                    })}

                    {/* Expand button */}
                    <td className="px-3 py-3 text-center">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setExpandedRow(expandedRow === item.code ? null : item.code);
                        }}
                        className="p-1 rounded hover:bg-[var(--color-bg-hover)] transition-colors"
                      >
                        <ChevronRight
                          className={`w-4 h-4 transition-transform duration-200 ${
                            expandedRow === item.code ? 'rotate-90' : ''
                          }`}
                          style={{ color: 'var(--color-text-muted)' }}
                        />
                      </button>
                    </td>
                  </tr>

                  {/* Expanded detail row */}
                  {expandedRow === item.code && (
                    <tr key={`${item.code}-detail`}>
                      <td
                        colSpan={4 + DIMENSIONS.length + 1}
                        className="px-6 py-4"
                        style={{ background: 'rgba(148,163,184,0.03)' }}
                      >
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          {/* Strengths */}
                          <div>
                            <div
                              className="text-xs font-medium mb-2 flex items-center gap-1.5"
                              style={{ color: 'var(--color-danger)' }}
                            >
                              <Star className="w-3.5 h-3.5" />
                              优势
                            </div>
                            {item.strengths && item.strengths.length > 0 ? (
                              <ul className="space-y-1">
                                {item.strengths.map((s, i) => (
                                  <li
                                    key={i}
                                    className="text-xs flex items-start gap-2"
                                    style={{ color: 'var(--color-text-secondary)' }}
                                  >
                                    <span
                                      className="mt-1 w-1.5 h-1.5 rounded-full flex-shrink-0"
                                      style={{ background: 'var(--color-danger)' }}
                                    />
                                    {s}
                                  </li>
                                ))}
                              </ul>
                            ) : (
                              <div
                                className="text-xs"
                                style={{ color: 'var(--color-text-muted)' }}
                              >
                                暂无
                              </div>
                            )}
                          </div>

                          {/* Risks */}
                          <div>
                            <div
                              className="text-xs font-medium mb-2 flex items-center gap-1.5"
                              style={{ color: 'var(--color-warning)' }}
                            >
                              <AlertTriangle className="w-3.5 h-3.5" />
                              风险
                            </div>
                            {item.risks && item.risks.length > 0 ? (
                              <ul className="space-y-1">
                                {item.risks.map((r, i) => (
                                  <li
                                    key={i}
                                    className="text-xs flex items-start gap-2"
                                    style={{ color: 'var(--color-text-secondary)' }}
                                  >
                                    <span
                                      className="mt-1 w-1.5 h-1.5 rounded-full flex-shrink-0"
                                      style={{ background: 'var(--color-warning)' }}
                                    />
                                    {r}
                                  </li>
                                ))}
                              </ul>
                            ) : (
                              <div
                                className="text-xs"
                                style={{ color: 'var(--color-text-muted)' }}
                              >
                                暂无
                              </div>
                            )}
                          </div>
                        </div>
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Footer */}
      <div
        className="text-xs mt-4 text-right"
        style={{ color: 'var(--color-text-muted)' }}
      >
        更新时间: {fetchedAt ? new Date(fetchedAt).toLocaleString('zh-CN') : '—'}
      </div>
    </div>
  );
}
