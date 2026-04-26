import { useState, useEffect } from 'react';
import api from '@/lib/api';
import { Monitor, RefreshCw, Database, Clock, Activity, Zap, AlertTriangle } from 'lucide-react';

interface SystemStatus {
  ok: boolean;
  cache_count: number;
  db_active_conns: number;
  db_total_conns: number;
  uptime: string;
  uptime_seconds: number;
  server_time: string;
}

interface SlowQuery {
  timestamp: string;
  method: string;
  path: string;
  endpoint: string;
  duration_ms: number;
  status_code: number;
  client_ip: string;
}

interface SlowQueriesResponse {
  ok: boolean;
  items: SlowQuery[];
}

interface PerformanceStats {
  ok: boolean;
  endpoints: {
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
  }[];
}

export default function DiagPage() {
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [slowQueries, setSlowQueries] = useState<SlowQuery[]>([]);
  const [perfStats, setPerfStats] = useState<PerformanceStats['endpoints']>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = () => {
    setLoading(true);
    setError('');
    Promise.all([
      api.get<SystemStatus>('/system-status'),
      api.get<SlowQueriesResponse>('/slow-queries'),
      api.get<PerformanceStats>('/performance-stats'),
    ])
      .then(([statusRes, sqRes, perfRes]) => {
        setStatus(statusRes.data);
        setSlowQueries(sqRes.data.items || []);
        setPerfStats(perfRes.data.endpoints || []);
      })
      .catch(() => setError('加载诊断数据失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchData();
  }, []);

  const formatDuration = (ms: number) => {
    if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
    return `${ms.toFixed(1)}ms`;
  };

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Monitor className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">系统诊断</h1>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '80px' }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Monitor className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">系统诊断</h1>
        </div>
        <div className="text-sm px-4 py-3 rounded-lg" style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
          <button onClick={fetchData} className="ml-3 underline">重试</button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Monitor className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">系统诊断</h1>
        </div>
        <button onClick={fetchData} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      {/* Status cards */}
      {status && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <div className="flex items-center gap-2 mb-1">
              <Clock className="w-4 h-4" style={{ color: 'var(--color-success)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>运行时间</span>
            </div>
            <div className="text-lg font-bold font-mono">{status.uptime}</div>
          </div>
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <div className="flex items-center gap-2 mb-1">
              <Database className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>数据库连接</span>
            </div>
            <div className="text-lg font-bold font-mono">{status.db_active_conns} / {status.db_total_conns}</div>
          </div>
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <div className="flex items-center gap-2 mb-1">
              <Zap className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>缓存条目</span>
            </div>
            <div className="text-lg font-bold font-mono">{status.cache_count}</div>
          </div>
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <div className="flex items-center gap-2 mb-1">
              <Activity className="w-4 h-4" style={{ color: 'var(--color-cyan)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>服务器时间</span>
            </div>
            <div className="text-sm font-mono">{new Date(status.server_time).toLocaleTimeString('zh-CN')}</div>
          </div>
        </div>
      )}

      {/* Performance stats */}
      {perfStats.length > 0 && (
        <div className="glass-panel rounded-xl p-4 mb-6" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
            API 性能统计
          </h3>
          <div className="overflow-x-auto">
            <table className="data-table">
              <thead>
                <tr>
                  <th>接口</th>
                  <th>调用次数</th>
                  <th>平均耗时</th>
                  <th>P95</th>
                  <th>P99</th>
                  <th>最大</th>
                  <th>慢查询</th>
                </tr>
              </thead>
              <tbody>
                {perfStats.slice(0, 20).map((ep) => (
                  <tr key={ep.endpoint}>
                    <td className="font-mono text-xs">{ep.endpoint}</td>
                    <td className="font-mono">{ep.count}</td>
                    <td className="font-mono">{formatDuration(ep.avg_duration_ms)}</td>
                    <td className="font-mono">{formatDuration(ep.p95_duration_ms)}</td>
                    <td className="font-mono">{formatDuration(ep.p99_duration_ms)}</td>
                    <td className="font-mono">{formatDuration(ep.max_duration_ms)}</td>
                    <td>
                      {ep.slow_count > 0 && (
                        <span className="flex items-center gap-1" style={{ color: 'var(--color-warning)' }}>
                          <AlertTriangle className="w-3 h-3" />
                          {ep.slow_count}
                        </span>
                      )}
                      {ep.slow_count === 0 && <span style={{ color: 'var(--color-text-muted)' }}>0</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Slow queries */}
      {slowQueries.length > 0 && (
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
            最近慢查询
          </h3>
          <div className="space-y-2">
            {slowQueries.slice(0, 10).map((sq, i) => (
              <div
                key={i}
                className="flex items-center justify-between p-2 rounded-lg text-xs"
                style={{ background: 'var(--color-bg-hover)' }}
              >
                <div className="flex items-center gap-2">
                  <span
                    className="px-1.5 py-0.5 rounded font-mono font-medium"
                    style={{
                      background: sq.method === 'GET' ? 'rgba(34,197,94,0.15)' : 'rgba(59,130,246,0.15)',
                      color: sq.method === 'GET' ? 'var(--color-success)' : 'var(--color-accent)',
                    }}
                  >
                    {sq.method}
                  </span>
                  <span className="font-mono">{sq.path}</span>
                </div>
                <div className="flex items-center gap-3">
                  <span
                    className="font-mono font-medium"
                    style={{ color: sq.duration_ms > 3000 ? 'var(--color-danger)' : 'var(--color-warning)' }}
                  >
                    {formatDuration(sq.duration_ms)}
                  </span>
                  <span style={{ color: 'var(--color-text-muted)' }}>
                    {new Date(sq.timestamp).toLocaleTimeString('zh-CN')}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Empty state */}
      {perfStats.length === 0 && slowQueries.length === 0 && (
        <div
          className="text-center py-16 rounded-xl"
          style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}
        >
          <Monitor className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>系统运行正常，暂无诊断数据</p>
        </div>
      )}
    </div>
  );
}
