import { useState, useEffect, useMemo, useCallback } from 'react';
import {
  getVitals,
  getLatestVital,
  getVitalsByName,
  getMockVitals,
  clearVitals,
  type VitalEntry,
  type VitalRating,
} from '@/lib/vitals';
import EChart from '@/components/charts/EChart';
import type { EChartsOption } from 'echarts';
import {
  Activity,
  Gauge,
  Timer,
  Zap,
  BarChart3,
  Trash2,
  RefreshCw,
} from 'lucide-react';
import { SkeletonStatCards, SkeletonTable } from '@/components/ui/Skeleton';

/* ─── Metric definitions ─── */
interface MetricDef {
  name: string;
  label: string;
  unit: string;
  description: string;
  icon: typeof Activity;
  format: (v: number) => string;
}

const METRICS: MetricDef[] = [
  {
    name: 'LCP',
    label: 'Largest Contentful Paint',
    unit: 'ms',
    description: '最大内容渲染时间',
    icon: Timer,
    format: (v) => `${(v / 1000).toFixed(2)}s`,
  },
  {
    name: 'FCP',
    label: 'First Contentful Paint',
    unit: 'ms',
    description: '首次内容渲染',
    icon: Zap,
    format: (v) => `${v.toFixed(0)}ms`,
  },
  {
    name: 'CLS',
    label: 'Cumulative Layout Shift',
    unit: '',
    description: '累积布局偏移',
    icon: BarChart3,
    format: (v) => v.toFixed(3),
  },
  {
    name: 'TTFB',
    label: 'Time to First Byte',
    unit: 'ms',
    description: '首字节时间',
    icon: Gauge,
    format: (v) => `${v.toFixed(0)}ms`,
  },
  {
    name: 'INP',
    label: 'Interaction to Next Paint',
    unit: 'ms',
    description: '交互延迟',
    icon: Activity,
    format: (v) => `${v.toFixed(0)}ms`,
  },
];

const RATING_COLORS: Record<VitalRating, string> = {
  good: '#22c55e',
  'needs-improvement': '#eab308',
  poor: '#ef4444',
};

const RATING_LABELS: Record<VitalRating, string> = {
  good: '良好',
  'needs-improvement': '需改进',
  poor: '较差',
};

function ratingBgClass(rating: VitalRating): string {
  switch (rating) {
    case 'good':
      return 'border-green-500/30 bg-green-500/5';
    case 'needs-improvement':
      return 'border-yellow-500/30 bg-yellow-500/5';
    case 'poor':
      return 'border-red-500/30 bg-red-500/5';
  }
}

function formatTimestamp(ts: number): string {
  return new Date(ts).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

/* ─── Summary Card ─── */
function VitalCard({
  metric,
  entry,
}: {
  metric: MetricDef;
  entry: VitalEntry | undefined;
}) {
  const Icon = metric.icon;
  const rating = entry?.rating ?? 'good';
  const value = entry?.value;

  return (
    <div
      className={`glass-panel rounded-xl p-4 border transition-all ${ratingBgClass(rating)}`}
    >
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Icon
            className="w-4 h-4"
            style={{ color: RATING_COLORS[rating] }}
          />
          <span className="text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
            {metric.name}
          </span>
        </div>
        <span
          className="text-[10px] px-1.5 py-0.5 rounded-full font-medium"
          style={{
            background: `${RATING_COLORS[rating]}20`,
            color: RATING_COLORS[rating],
          }}
        >
          {RATING_LABELS[rating]}
        </span>
      </div>
      <div className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
        {value != null ? metric.format(value) : '—'}
      </div>
      <div className="text-xs mt-1" style={{ color: 'var(--color-text-muted)' }}>
        {metric.description}
      </div>
    </div>
  );
}

/* ─── Main Page ─── */
export default function VitalsPage() {
  const [entries, setEntries] = useState<VitalEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [useMock, setUseMock] = useState(false);

  const loadData = useCallback(() => {
    setLoading(true);
    // Small delay to allow web-vitals to fire
    setTimeout(() => {
      const data = getVitals();
      if (data.length > 0) {
        setEntries(data);
        setUseMock(false);
      } else {
        setEntries(getMockVitals());
        setUseMock(true);
      }
      setLoading(false);
    }, 1500);
  }, []);

  useEffect(() => {
    loadData();
    // Refresh every 30s
    const interval = setInterval(loadData, 30_000);
    return () => clearInterval(interval);
  }, [loadData]);

  const latestMap = useMemo(() => {
    const map: Record<string, VitalEntry | undefined> = {};
    // If using mock, compute latest from mock data; otherwise use real data
    const src = useMock ? entries : getVitals();
    for (const def of METRICS) {
      // Find the latest entry for each metric
      for (let i = src.length - 1; i >= 0; i--) {
        if (src[i].name === def.name) {
          map[def.name] = src[i];
          break;
        }
      }
    }
    return map;
  }, [entries, useMock]);

  /* ─── LCP Trend Chart ─── */
  const lcpTrendOption: EChartsOption = useMemo(() => {
    const lcpData = entries
      .filter((e) => e.name === 'LCP')
      .sort((a, b) => a.timestamp - b.timestamp);
    if (lcpData.length === 0) return {};

    return {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(15,23,42,0.92)',
        borderColor: 'rgba(148,163,184,0.2)',
        textStyle: { color: '#e2e8f0', fontSize: 12 },
        formatter: (params: unknown) => {
          const p = Array.isArray(params) ? params[0] : null;
          if (!p || typeof p !== 'object' || !('value' in p)) return '';
          const val = (p as { value: number }).value;
          return `LCP: ${(val / 1000).toFixed(2)}s`;
        },
      },
      grid: { left: 60, right: 20, top: 30, bottom: 30 },
      xAxis: {
        type: 'category',
        data: lcpData.map((e) => formatTimestamp(e.timestamp)),
        axisLabel: { color: '#94a3b8', fontSize: 10, rotate: 30 },
      },
      yAxis: {
        type: 'value',
        name: 'ms',
        axisLabel: { color: '#94a3b8' },
        splitLine: { lineStyle: { color: 'rgba(51,65,85,0.5)' } },
      },
      series: [
        {
          type: 'line',
          data: lcpData.map((e) => e.value),
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          itemStyle: { color: '#3b82f6' },
          lineStyle: { width: 2 },
          areaStyle: {
            color: {
              type: 'linear',
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: 'rgba(59,130,246,0.3)' },
                { offset: 1, color: 'rgba(59,130,246,0.02)' },
              ],
            },
          },
          markLine: {
            silent: true,
            data: [
              {
                yAxis: 2500,
                lineStyle: { color: '#22c55e', type: 'dashed' },
                label: { formatter: 'Good: 2.5s', color: '#22c55e', fontSize: 10 },
              },
              {
                yAxis: 4000,
                lineStyle: { color: '#ef4444', type: 'dashed' },
                label: { formatter: 'Poor: 4s', color: '#ef4444', fontSize: 10 },
              },
            ],
          },
        },
      ],
    };
  }, [entries]);

  /* ─── CLS Trend Chart ─── */
  const clsTrendOption: EChartsOption = useMemo(() => {
    const clsData = entries
      .filter((e) => e.name === 'CLS')
      .sort((a, b) => a.timestamp - b.timestamp);
    if (clsData.length === 0) return {};

    return {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(15,23,42,0.92)',
        borderColor: 'rgba(148,163,184,0.2)',
        textStyle: { color: '#e2e8f0', fontSize: 12 },
        formatter: (params: unknown) => {
          const p = Array.isArray(params) ? params[0] : null;
          if (!p || typeof p !== 'object' || !('value' in p)) return '';
          const val = (p as { value: number }).value;
          return `CLS: ${val.toFixed(3)}`;
        },
      },
      grid: { left: 60, right: 20, top: 30, bottom: 30 },
      xAxis: {
        type: 'category',
        data: clsData.map((e) => formatTimestamp(e.timestamp)),
        axisLabel: { color: '#94a3b8', fontSize: 10, rotate: 30 },
      },
      yAxis: {
        type: 'value',
        axisLabel: { color: '#94a3b8' },
        splitLine: { lineStyle: { color: 'rgba(51,65,85,0.5)' } },
      },
      series: [
        {
          type: 'line',
          data: clsData.map((e) => e.value),
          smooth: true,
          symbol: 'circle',
          symbolSize: 6,
          itemStyle: { color: '#8b5cf6' },
          lineStyle: { width: 2 },
          areaStyle: {
            color: {
              type: 'linear',
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: 'rgba(139,92,246,0.3)' },
                { offset: 1, color: 'rgba(139,92,246,0.02)' },
              ],
            },
          },
          markLine: {
            silent: true,
            data: [
              {
                yAxis: 0.1,
                lineStyle: { color: '#22c55e', type: 'dashed' },
                label: { formatter: 'Good: 0.1', color: '#22c55e', fontSize: 10 },
              },
              {
                yAxis: 0.25,
                lineStyle: { color: '#ef4444', type: 'dashed' },
                label: { formatter: 'Poor: 0.25', color: '#ef4444', fontSize: 10 },
              },
            ],
          },
        },
      ],
    };
  }, [entries]);

  /* ─── Recent measurements table ─── */
  const recentEntries = useMemo(() => {
    return [...entries]
      .sort((a, b) => b.timestamp - a.timestamp)
      .slice(0, 50);
  }, [entries]);

  const handleClear = () => {
    clearVitals();
    setEntries([]);
    loadData();
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-4">
        <div>
          <h1
            className="text-2xl font-bold flex items-center gap-2"
            style={{ color: 'var(--color-text-primary)' }}
          >
            <Activity className="w-6 h-6" style={{ color: 'var(--color-accent)' }} />
            性能监控
          </h1>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            Core Web Vitals 实时监控面板
            {useMock && (
              <span className="ml-2 px-2 py-0.5 text-xs rounded bg-yellow-500/20 text-yellow-400">
                示例数据
              </span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={loadData}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg transition-colors hover:bg-[var(--color-bg-hover)]"
            style={{
              color: 'var(--color-text-secondary)',
              border: '1px solid var(--color-border)',
            }}
          >
            <RefreshCw className="w-3.5 h-3.5" />
            刷新
          </button>
          <button
            onClick={handleClear}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg transition-colors hover:bg-red-500/10"
            style={{
              color: 'var(--color-text-secondary)',
              border: '1px solid var(--color-border)',
            }}
          >
            <Trash2 className="w-3.5 h-3.5" />
            清除
          </button>
        </div>
      </div>

      {/* Summary Cards */}
      {loading ? (
        <SkeletonStatCards count={5} />
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
          {METRICS.map((m) => (
            <VitalCard key={m.name} metric={m} entry={latestMap[m.name]} />
          ))}
        </div>
      )}

      {/* Charts */}
      {loading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div className="glass-panel rounded-xl p-5">
            <div className="h-5 w-32 rounded bg-gray-700/50 animate-pulse mb-4" />
            <div className="h-64 rounded-lg bg-gray-700/30 animate-pulse" />
          </div>
          <div className="glass-panel rounded-xl p-5">
            <div className="h-5 w-32 rounded bg-gray-700/50 animate-pulse mb-4" />
            <div className="h-64 rounded-lg bg-gray-700/30 animate-pulse" />
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* LCP Trend */}
          <div className="glass-panel rounded-xl p-5">
            <h3
              className="text-sm font-medium mb-3 flex items-center gap-2"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              <Timer className="w-4 h-4" style={{ color: '#3b82f6' }} />
              LCP 趋势
            </h3>
            {entries.filter((e) => e.name === 'LCP').length > 0 ? (
              <EChart option={lcpTrendOption} style={{ height: 280 }} />
            ) : (
              <div className="flex items-center justify-center h-64 text-sm" style={{ color: 'var(--color-text-muted)' }}>
                暂无 LCP 数据
              </div>
            )}
          </div>

          {/* CLS Trend */}
          <div className="glass-panel rounded-xl p-5">
            <h3
              className="text-sm font-medium mb-3 flex items-center gap-2"
              style={{ color: 'var(--color-text-secondary)' }}
            >
              <BarChart3 className="w-4 h-4" style={{ color: '#8b5cf6' }} />
              CLS 趋势
            </h3>
            {entries.filter((e) => e.name === 'CLS').length > 0 ? (
              <EChart option={clsTrendOption} style={{ height: 280 }} />
            ) : (
              <div className="flex items-center justify-center h-64 text-sm" style={{ color: 'var(--color-text-muted)' }}>
                暂无 CLS 数据
              </div>
            )}
          </div>
        </div>
      )}

      {/* Recent Measurements Table */}
      <div className="glass-panel rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b" style={{ borderColor: 'var(--color-border)' }}>
          <h3
            className="text-sm font-medium flex items-center gap-2"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            <Gauge className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
            最近测量记录
          </h3>
        </div>
        {loading ? (
          <div className="p-5">
            <SkeletonTable rows={8} columns={5} />
          </div>
        ) : recentEntries.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12" style={{ color: 'var(--color-text-muted)' }}>
            <BarChart3 className="w-10 h-10 mb-3 opacity-40" />
            <p className="text-sm">暂无测量数据</p>
            <p className="text-xs mt-1">页面加载后将自动收集 Web Vitals 指标</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr style={{ borderBottom: '1px solid var(--color-border)' }}>
                  <th className="px-5 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                    时间
                  </th>
                  <th className="px-5 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                    指标
                  </th>
                  <th className="px-5 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                    值
                  </th>
                  <th className="px-5 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                    评级
                  </th>
                </tr>
              </thead>
              <tbody>
                {recentEntries.map((entry) => {
                  const def = METRICS.find((m) => m.name === entry.name);
                  return (
                    <tr
                      key={entry.id}
                      className="transition-colors hover:bg-[var(--color-bg-hover)]"
                      style={{ borderBottom: '1px solid var(--color-border)' }}
                    >
                      <td className="px-5 py-2.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                        {formatTimestamp(entry.timestamp)}
                      </td>
                      <td className="px-5 py-2.5">
                        <span className="font-medium" style={{ color: 'var(--color-text-primary)' }}>
                          {entry.name}
                        </span>
                        {def && (
                          <span className="ml-2 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                            {def.description}
                          </span>
                        )}
                      </td>
                      <td className="px-5 py-2.5 font-mono" style={{ color: 'var(--color-text-primary)' }}>
                        {def ? def.format(entry.value) : entry.value}
                      </td>
                      <td className="px-5 py-2.5">
                        <span
                          className="text-xs px-2 py-0.5 rounded-full font-medium"
                          style={{
                            background: `${RATING_COLORS[entry.rating]}20`,
                            color: RATING_COLORS[entry.rating],
                          }}
                        >
                          {RATING_LABELS[entry.rating]}
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
