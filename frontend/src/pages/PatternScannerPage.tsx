import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { patternScannerApi, type PatternResult, type PatternScannerResponse } from '@/lib/api';
import { Scan, TrendingUp, TrendingDown, Minus, RefreshCw, Filter, BarChart3 } from 'lucide-react';

type DirectionFilter = 'all' | 'bullish' | 'bearish' | 'neutral';
type CategoryFilter = 'all' | 'kline' | 'chart' | 'volume';

const directionConfig: Record<string, { icon: typeof TrendingUp; label: string; color: string }> = {
  bullish: { icon: TrendingUp, label: '看涨', color: 'var(--color-danger)' },
  bearish: { icon: TrendingDown, label: '看跌', color: 'var(--color-success)' },
  neutral: { icon: Minus, label: '中性', color: 'var(--color-text-muted)' },
};

const categoryConfig: Record<string, { icon: typeof BarChart3; label: string }> = {
  kline: { icon: BarChart3, label: 'K线' },
  chart: { icon: BarChart3, label: '图表' },
  volume: { icon: BarChart3, label: '成交量' },
};

const directionButtons: { key: DirectionFilter; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'bullish', label: '看涨' },
  { key: 'bearish', label: '看跌' },
  { key: 'neutral', label: '中性' },
];

const categoryButtons: { key: CategoryFilter; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'kline', label: 'K线' },
  { key: 'chart', label: '图表' },
  { key: 'volume', label: '成交量' },
];

export default function PatternScannerPage() {
  const [data, setData] = useState<PatternScannerResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [directionFilter, setDirectionFilter] = useState<DirectionFilter>('all');
  const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('all');
  const navigate = useNavigate();

  const fetchData = () => {
    setLoading(true);
    setError('');
    patternScannerApi
      .scan()
      .then((res) => setData(res.data))
      .catch(() => setError('加载形态数据失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchData();
  }, []);

  // Filter and sort patterns
  const filteredPatterns = data
    ? data.patterns
        .filter((p) => directionFilter === 'all' || p.direction === directionFilter)
        .filter((p) => categoryFilter === 'all' || p.category === categoryFilter)
        .sort((a, b) => b.confidence - a.confidence)
    : [];

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Scan className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">形态扫描器</h1>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '80px' }} />
          ))}
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
          <Scan className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">形态扫描器</h1>
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
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Scan className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">形态扫描器</h1>
          {data && (
            <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>
              扫描 {data.summary.scanned} 只
            </span>
          )}
          {data?.cached && (
            <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'rgba(245,158,11,0.15)', color: 'var(--color-warning)' }}>
              缓存
            </span>
          )}
        </div>
        <button onClick={fetchData} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <Scan className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>总形态数</span>
          </div>
          <div className="text-xl font-bold font-mono" style={{ color: 'var(--color-accent)' }}>
            {data?.summary.total || 0}
          </div>
        </div>
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <TrendingUp className="w-4 h-4" style={{ color: 'var(--color-danger)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>看涨形态</span>
          </div>
          <div className="text-xl font-bold font-mono" style={{ color: (data?.summary.bullish || 0) > 0 ? 'var(--color-danger)' : 'var(--color-text-muted)' }}>
            {data?.summary.bullish || 0}
          </div>
        </div>
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <TrendingDown className="w-4 h-4" style={{ color: 'var(--color-success)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>看跌形态</span>
          </div>
          <div className="text-xl font-bold font-mono" style={{ color: (data?.summary.bearish || 0) > 0 ? 'var(--color-success)' : 'var(--color-text-muted)' }}>
            {data?.summary.bearish || 0}
          </div>
        </div>
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <Minus className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>中性形态</span>
          </div>
          <div className="text-xl font-bold font-mono" style={{ color: (data?.summary.neutral || 0) > 0 ? 'var(--color-text-secondary)' : 'var(--color-text-muted)' }}>
            {data?.summary.neutral || 0}
          </div>
        </div>
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap items-center gap-3 mb-6">
        <div className="flex items-center gap-1.5">
          <Filter className="w-3.5 h-3.5" style={{ color: 'var(--color-text-muted)' }} />
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>方向:</span>
        </div>
        <div className="flex gap-1.5">
          {directionButtons.map((btn) => (
            <button
              key={btn.key}
              onClick={() => setDirectionFilter(btn.key)}
              className="text-xs px-2.5 py-1 rounded-lg transition-colors"
              style={{
                background: directionFilter === btn.key ? 'var(--color-accent)' : 'var(--color-bg-hover)',
                color: directionFilter === btn.key ? '#fff' : 'var(--color-text-secondary)',
                border: '1px solid var(--color-border)',
              }}
            >
              {btn.label}
            </button>
          ))}
        </div>
        <div className="w-px h-4" style={{ background: 'var(--color-border)' }} />
        <div className="flex items-center gap-1.5">
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>类别:</span>
        </div>
        <div className="flex gap-1.5">
          {categoryButtons.map((btn) => (
            <button
              key={btn.key}
              onClick={() => setCategoryFilter(btn.key)}
              className="text-xs px-2.5 py-1 rounded-lg transition-colors"
              style={{
                background: categoryFilter === btn.key ? 'var(--color-accent)' : 'var(--color-bg-hover)',
                color: categoryFilter === btn.key ? '#fff' : 'var(--color-text-secondary)',
                border: '1px solid var(--color-border)',
              }}
            >
              {btn.label}
            </button>
          ))}
        </div>
      </div>

      {/* Pattern table */}
      {filteredPatterns.length > 0 ? (
        <div className="glass-panel rounded-xl overflow-hidden" style={{ borderColor: 'var(--color-border)' }}>
          <table className="data-table">
            <thead>
              <tr>
                <th>形态名称</th>
                <th>股票</th>
                <th>方向</th>
                <th>类别</th>
                <th>置信度</th>
                <th>日期</th>
                <th>描述</th>
              </tr>
            </thead>
            <tbody>
              {filteredPatterns.map((p, i) => {
                const dir = directionConfig[p.direction] || directionConfig.neutral;
                const DirIcon = dir.icon;
                const cat = categoryConfig[p.category] || { label: p.category };
                return (
                  <tr
                    key={`${p.code}-${p.pattern}-${i}`}
                    className="cursor-pointer hover:bg-[var(--color-bg-hover)] transition-colors"
                    onClick={() => navigate(`/analyze?code=${encodeURIComponent(p.code)}`)}
                  >
                    <td className="font-medium">{p.pattern || '—'}</td>
                    <td>
                      <div className="font-medium">{p.name || '—'}</div>
                      <div className="font-mono text-xs" style={{ color: 'var(--color-text-muted)' }}>{p.code}</div>
                    </td>
                    <td>
                      <span
                        className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium"
                        style={{ background: `${dir.color}15`, color: dir.color }}
                      >
                        <DirIcon className="w-3 h-3" />
                        {dir.label}
                      </span>
                    </td>
                    <td>
                      <span
                        className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium"
                        style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}
                      >
                        {cat.label}
                      </span>
                    </td>
                    <td style={{ minWidth: '120px' }}>
                      <div className="flex items-center gap-2">
                        <div
                          className="flex-1 rounded-full h-1.5"
                          style={{ background: 'var(--color-bg-hover)' }}
                        >
                          <div
                            className="h-full rounded-full transition-all"
                            style={{
                              width: `${Math.round(p.confidence * 100)}%`,
                              background:
                                p.confidence >= 0.8
                                  ? 'var(--color-danger)'
                                  : p.confidence >= 0.5
                                    ? 'var(--color-warning)'
                                    : 'var(--color-text-muted)',
                            }}
                          />
                        </div>
                        <span className="text-xs font-mono" style={{ color: 'var(--color-text-muted)' }}>
                          {Math.round(p.confidence * 100)}%
                        </span>
                      </div>
                    </td>
                    <td className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{p.date || '—'}</td>
                    <td className="text-xs" style={{ color: 'var(--color-text-secondary)', maxWidth: '200px' }}>
                      <div className="truncate">{p.description || '—'}</div>
                    </td>
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
          <Scan className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>暂无形态数据</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            请先添加自选股
          </p>
        </div>
      )}

      {/* Timestamp footer */}
      <div className="text-xs mt-4 text-right" style={{ color: 'var(--color-text-muted)' }}>
        {data?.cached ? '数据来源: 缓存' : '实时扫描结果'}
      </div>
    </div>
  );
}
