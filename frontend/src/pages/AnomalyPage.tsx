import { useState, useEffect } from 'react';
import api from '@/lib/api';
import { AlertTriangle, RefreshCw, TrendingUp, TrendingDown, Volume2, ArrowDown } from 'lucide-react';

interface AnomalyItem {
  code: string;
  name: string;
  anomaly_type: string;
  description: string;
  severity: string;
  date: string;
}

interface AnomaliesResponse {
  ok: boolean;
  anomalies: {
    big_move: AnomalyItem[] | null;
    limit_down: AnomalyItem[] | null;
    limit_up: AnomalyItem[] | null;
    volume_surge: AnomalyItem[] | null;
  };
  scanned: number;
  fetched_at: string;
}

const typeConfig: Record<string, { icon: typeof TrendingUp; label: string; color: string }> = {
  big_move: { icon: TrendingUp, label: '大幅波动', color: 'var(--color-warning)' },
  limit_up: { icon: TrendingUp, label: '涨停', color: 'var(--color-danger)' },
  limit_down: { icon: ArrowDown, label: '跌停', color: 'var(--color-success)' },
  volume_surge: { icon: Volume2, label: '放量', color: 'var(--color-accent)' },
};

export default function AnomalyPage() {
  const [data, setData] = useState<AnomaliesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = () => {
    setLoading(true);
    setError('');
    api
      .get<AnomaliesResponse>('/anomalies')
      .then((res) => setData(res.data))
      .catch(() => setError('加载异常数据失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchData();
  }, []);

  // Flatten all anomalies into a single list
  const allAnomalies = data
    ? Object.entries(data.anomalies).flatMap(([type, items]) =>
        (items || []).map((item) => ({ ...item, type }))
      )
    : [];

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <AlertTriangle className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">异常检测</h1>
        </div>
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '60px' }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <AlertTriangle className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">异常检测</h1>
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
          <AlertTriangle className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">异常检测</h1>
          {data && (
            <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>
              扫描 {data.scanned} 只
            </span>
          )}
        </div>
        <button onClick={fetchData} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      {/* Category summary */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        {Object.entries(data?.anomalies || {}).map(([type, items]) => {
          const config = typeConfig[type] || { icon: AlertTriangle, label: type, color: 'var(--color-text-muted)' };
          const Icon = config.icon;
          const count = (items || []).length;
          return (
            <div key={type} className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
              <div className="flex items-center gap-2 mb-1">
                <Icon className="w-4 h-4" style={{ color: config.color }} />
                <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{config.label}</span>
              </div>
              <div className="text-xl font-bold font-mono" style={{ color: count > 0 ? config.color : 'var(--color-text-muted)' }}>
                {count}
              </div>
            </div>
          );
        })}
      </div>

      {/* Anomaly list */}
      {allAnomalies.length > 0 ? (
        <div className="glass-panel rounded-xl overflow-hidden" style={{ borderColor: 'var(--color-border)' }}>
          <table className="data-table">
            <thead>
              <tr>
                <th>类型</th>
                <th>股票</th>
                <th>代码</th>
                <th>描述</th>
                <th>严重度</th>
                <th>日期</th>
              </tr>
            </thead>
            <tbody>
              {allAnomalies.map((item, i) => {
                const config = typeConfig[item.anomaly_type] || { icon: AlertTriangle, label: item.anomaly_type, color: 'var(--color-text-muted)' };
                return (
                  <tr key={`${item.code}-${i}`}>
                    <td>
                      <span
                        className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium"
                        style={{ background: `${config.color}15`, color: config.color }}
                      >
                        {config.label}
                      </span>
                    </td>
                    <td className="font-medium">{item.name || '—'}</td>
                    <td className="font-mono text-xs">{item.code}</td>
                    <td className="text-xs" style={{ color: 'var(--color-text-secondary)' }}>{item.description || '—'}</td>
                    <td>
                      <span
                        className="text-xs px-1.5 py-0.5 rounded"
                        style={{
                          background: item.severity === 'high' ? 'rgba(239,68,68,0.15)' : item.severity === 'medium' ? 'rgba(245,158,11,0.15)' : 'rgba(100,116,139,0.15)',
                          color: item.severity === 'high' ? 'var(--color-danger)' : item.severity === 'medium' ? 'var(--color-warning)' : 'var(--color-text-muted)',
                        }}
                      >
                        {item.severity || '—'}
                      </span>
                    </td>
                    <td className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{item.date || '—'}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : (
        <div
          className="text-center py-16 rounded-xl"
          style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}
        >
          <AlertTriangle className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>暂未检测到异常</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            已扫描 {data?.scanned || 0} 只股票
          </p>
        </div>
      )}

      {/* Timestamp */}
      {data?.fetched_at && (
        <div className="text-xs mt-4 text-right" style={{ color: 'var(--color-text-muted)' }}>
          更新时间: {new Date(data.fetched_at).toLocaleString('zh-CN')}
        </div>
      )}
    </div>
  );
}
