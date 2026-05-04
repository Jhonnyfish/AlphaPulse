import { useState, useEffect } from 'react';
import api from '@/lib/api';
import ReactECharts from '@/components/charts/ReactECharts';
import { Gauge, RefreshCw, AlertTriangle, Zap, Timer, BarChart3, TrendingDown } from 'lucide-react';

interface EndpointStat {
  endpoint: string;
  method: string;
  path: string;
  count: number;
  avg_duration_ms: number;
  p50_duration_ms: number;
  p95_duration_ms: number;
  p99_duration_ms: number;
  max_duration_ms: number;
  slow_count: number;
}

interface PerfSummary {
  total_endpoints: number;
  total_requests: number;
  avg_duration_ms: number;
  p50_duration_ms: number;
  p95_duration_ms: number;
  p99_duration_ms: number;
  max_duration_ms: number;
  slow_count: number;
  slowest_endpoint: string;
}

interface PerfData {
  ok: boolean;
  endpoints: EndpointStat[];
  total_requests: number;
  summary: PerfSummary;
}

type SortKey = 'count' | 'avg_duration_ms' | 'p95_duration_ms' | 'max_duration_ms' | 'slow_count';

export default function PerfStatsPage() {
  const [data, setData] = useState<PerfData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [sortKey, setSortKey] = useState<SortKey>('count');
  const [sortAsc, setSortAsc] = useState(false);

  const fetchData = () => {
    setLoading(true);
    setError('');
    api.get<PerfData>('/performance-stats')
      .then((res) => setData(res.data))
      .catch(() => setError('加载性能数据失败'))
      .finally(() => setLoading(false));
  };

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { fetchData(); }, []);

  const formatMs = (ms: number) => {
    if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
    return `${ms.toFixed(1)}ms`;
  };

  const handleSort = (key: SortKey) => {
    if (sortKey === key) setSortAsc(!sortAsc);
    else { setSortKey(key); setSortAsc(false); }
  };

  const sortedEndpoints = data?.endpoints
    ? [...data.endpoints].sort((a, b) => {
        const mul = sortAsc ? 1 : -1;
        return (a[sortKey] - b[sortKey]) * mul;
      })
    : [];

  // Bar chart: top 15 endpoints by request count
  const topEndpointsChart = data?.endpoints ? {
    tooltip: { trigger: 'axis' as const, axisPointer: { type: 'shadow' as const } },
    grid: { left: '3%', right: '4%', bottom: '3%', top: '15%', containLabel: true },
    xAxis: { type: 'category' as const, data: sortedEndpoints.slice(0, 15).map(e => e.endpoint.replace('/api/', '')), axisLabel: { rotate: 45, fontSize: 10, color: '#94a3b8' } },
    yAxis: { type: 'value' as const, axisLabel: { color: '#94a3b8' }, splitLine: { lineStyle: { color: 'rgba(148,163,184,0.1)' } } },
    series: [
      { name: '调用次数', type: 'bar', data: sortedEndpoints.slice(0, 15).map(e => e.count), itemStyle: { color: '#3b82f6', borderRadius: [4, 4, 0, 0] } },
    ],
  } : null;

  // Latency comparison chart
  const latencyChart = data?.endpoints ? {
    tooltip: { trigger: 'axis' as const },
    legend: { data: ['平均', 'P95', 'P99', '最大'], textStyle: { color: '#94a3b8', fontSize: 11 }, top: 0 },
    grid: { left: '3%', right: '4%', bottom: '3%', top: '18%', containLabel: true },
    xAxis: { type: 'category' as const, data: sortedEndpoints.slice(0, 10).map(e => e.endpoint.replace('/api/', '')), axisLabel: { rotate: 45, fontSize: 10, color: '#94a3b8' } },
    yAxis: { type: 'value' as const, name: 'ms', axisLabel: { color: '#94a3b8' }, splitLine: { lineStyle: { color: 'rgba(148,163,184,0.1)' } } },
    series: [
      { name: '平均', type: 'bar', data: sortedEndpoints.slice(0, 10).map(e => +e.avg_duration_ms.toFixed(1)), itemStyle: { color: '#3b82f6' } },
      { name: 'P95', type: 'bar', data: sortedEndpoints.slice(0, 10).map(e => +e.p95_duration_ms.toFixed(1)), itemStyle: { color: '#f59e0b' } },
      { name: 'P99', type: 'bar', data: sortedEndpoints.slice(0, 10).map(e => +e.p99_duration_ms.toFixed(1)), itemStyle: { color: '#ef4444' } },
      { name: '最大', type: 'bar', data: sortedEndpoints.slice(0, 10).map(e => +e.max_duration_ms.toFixed(1)), itemStyle: { color: '#6b7280' } },
    ],
  } : null;

  const SortHeader = ({ label, field }: { label: string; field: SortKey }) => (
    <th
      className="cursor-pointer hover:opacity-80 select-none"
      onClick={() => handleSort(field)}
      style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', textAlign: 'left', fontSize: '12px', color: 'var(--color-text-secondary)' }}
    >
      {label}
      {sortKey === field && (
        <span className="ml-1">{sortAsc ? '↑' : '↓'}</span>
      )}
    </th>
  );

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Gauge className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">绩效统计</h1>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '80px' }} />
          ))}
        </div>
        <div className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '300px' }} />
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Gauge className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">绩效统计</h1>
        </div>
        <div className="text-sm px-4 py-3 rounded-lg" style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
          <button onClick={fetchData} className="ml-3 underline">重试</button>
        </div>
      </div>
    );
  }

  const summary = data?.summary;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Gauge className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">绩效统计</h1>
        </div>
        <button onClick={fetchData} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      {/* Summary cards */}
      {summary && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <div className="flex items-center gap-2 mb-1">
              <BarChart3 className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>总请求数</span>
            </div>
            <div className="text-lg font-bold font-mono">{summary.total_requests.toLocaleString()}</div>
          </div>
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <div className="flex items-center gap-2 mb-1">
              <Zap className="w-4 h-4" style={{ color: 'var(--color-success)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>端点数</span>
            </div>
            <div className="text-lg font-bold font-mono">{summary.total_endpoints}</div>
          </div>
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <div className="flex items-center gap-2 mb-1">
              <Timer className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>平均耗时</span>
            </div>
            <div className="text-lg font-bold font-mono">{formatMs(summary.avg_duration_ms)}</div>
          </div>
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <div className="flex items-center gap-2 mb-1">
              <TrendingDown className="w-4 h-4" style={{ color: 'var(--color-danger)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>慢查询</span>
            </div>
            <div className="text-lg font-bold font-mono" style={{ color: summary.slow_count > 0 ? 'var(--color-danger)' : 'var(--color-success)' }}>
              {summary.slow_count}
            </div>
          </div>
        </div>
      )}

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {topEndpointsChart && (
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <h3 className="text-sm font-medium mb-2" style={{ color: 'var(--color-text-secondary)' }}>请求量 TOP 15</h3>
            <ReactECharts option={topEndpointsChart} style={{ height: '300px' }} />
          </div>
        )}
        {latencyChart && (
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <h3 className="text-sm font-medium mb-2" style={{ color: 'var(--color-text-secondary)' }}>延迟分布 TOP 10</h3>
            <ReactECharts option={latencyChart} style={{ height: '300px' }} />
          </div>
        )}
      </div>

      {/* Endpoints table */}
      {sortedEndpoints.length > 0 && (
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>全部端点详情</h3>
          <div className="overflow-x-auto">
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', textAlign: 'left', fontSize: '12px', color: 'var(--color-text-secondary)' }}>接口</th>
                  <SortHeader label="调用次数" field="count" />
                  <SortHeader label="平均耗时" field="avg_duration_ms" />
                  <SortHeader label="P95" field="p95_duration_ms" />
                  <th style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', textAlign: 'left', fontSize: '12px', color: 'var(--color-text-secondary)' }}>P99</th>
                  <SortHeader label="最大耗时" field="max_duration_ms" />
                  <SortHeader label="慢查询" field="slow_count" />
                </tr>
              </thead>
              <tbody>
                {sortedEndpoints.map((ep) => (
                  <tr key={ep.endpoint} className="hover:bg-[var(--color-bg-hover)] transition-colors">
                    <td style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', fontSize: '12px', fontFamily: 'monospace' }}>{ep.endpoint}</td>
                    <td style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', fontFamily: 'monospace' }}>{ep.count}</td>
                    <td style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', fontFamily: 'monospace', color: ep.avg_duration_ms > 500 ? 'var(--color-warning)' : 'inherit' }}>{formatMs(ep.avg_duration_ms)}</td>
                    <td style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', fontFamily: 'monospace' }}>{formatMs(ep.p95_duration_ms)}</td>
                    <td style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', fontFamily: 'monospace' }}>{formatMs(ep.p99_duration_ms)}</td>
                    <td style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)', fontFamily: 'monospace', color: ep.max_duration_ms > 3000 ? 'var(--color-danger)' : 'inherit' }}>{formatMs(ep.max_duration_ms)}</td>
                    <td style={{ padding: '8px 12px', borderBottom: '1px solid var(--color-border)' }}>
                      {ep.slow_count > 0 ? (
                        <span className="flex items-center gap-1" style={{ color: 'var(--color-warning)' }}>
                          <AlertTriangle className="w-3 h-3" />{ep.slow_count}
                        </span>
                      ) : (
                        <span style={{ color: 'var(--color-text-muted)' }}>0</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Empty */}
      {sortedEndpoints.length === 0 && (
        <div className="text-center py-16 rounded-xl" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
          <Gauge className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>暂无性能数据</p>
        </div>
      )}
    </div>
  );
}
