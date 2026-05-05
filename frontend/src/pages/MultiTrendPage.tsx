import { useState, useEffect, useCallback } from 'react';
import api from '@/lib/api';
import ReactECharts from '@/components/charts/ReactECharts';
import { GitBranch, RefreshCw, TrendingUp, TrendingDown, Minus } from 'lucide-react';

interface PeriodIndicators {
  return_pct: number;
  ma5: number;
  ma10: number;
  ma20: number;
  ma_aligned: boolean;
  rsi: number;
  volume_trend: string;
  strength: number;
}

interface TrendStock {
  code: string;
  name: string;
  daily: PeriodIndicators;
  weekly: PeriodIndicators;
  monthly: PeriodIndicators;
  overall_strength: number;
}

interface MultiTrendData {
  ok: boolean;
  stocks: TrendStock[];
  cached: boolean;
}

const PERIOD_LABELS: Record<string, string> = { daily: '日线', weekly: '周线', monthly: '月线' };
const PERIOD_KEYS = ['daily', 'weekly', 'monthly'] as const;

function strengthColor(s: number): string {
  if (s >= 80) return '#ef4444';
  if (s >= 60) return '#f59e0b';
  if (s >= 40) return '#3b82f6';
  return '#22c55e';
}

function strengthLabel(s: number): string {
  if (s >= 80) return '强势';
  if (s >= 60) return '偏强';
  if (s >= 40) return '中性';
  if (s >= 20) return '偏弱';
  return '弱势';
}

function volTrendLabel(v: string): string {
  const map: Record<string, string> = { increasing: '放量', decreasing: '缩量', stable: '平稳' };
  return map[v] || v;
}

function volTrendColor(v: string): string {
  if (v === 'increasing') return '#ef4444';
  if (v === 'decreasing') return '#22c55e';
  return '#6b7280';
}

export default function MultiTrendPage() {
  const [data, setData] = useState<MultiTrendData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = useCallback(() => {
    setLoading(true);
    setError('');
    api.get<MultiTrendData>('/multi-trend')
      .then((res) => setData(res.data))
      .catch(() => setError('加载多周期趋势数据失败'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const stocks = data?.stocks || [];

  // Radar chart for top 5 stocks
  const radarOption = stocks.length > 0 ? {
    tooltip: {},
    legend: { bottom: 0, textStyle: { color: '#94a3b8', fontSize: 11 }, itemWidth: 12, itemHeight: 12 },
    radar: {
      indicator: [
        { name: '日线强度', max: 100 },
        { name: '周线强度', max: 100 },
        { name: '月线强度', max: 100 },
        { name: '综合强度', max: 100 },
      ],
      shape: 'polygon' as const,
      splitNumber: 4,
      axisName: { color: '#94a3b8', fontSize: 11 },
      splitLine: { lineStyle: { color: 'rgba(148,163,184,0.15)' } },
      splitArea: { show: false },
      axisLine: { lineStyle: { color: 'rgba(148,163,184,0.15)' } },
    },
    series: [{
      type: 'radar' as const,
      data: stocks.slice(0, 5).map((s, i) => ({
        value: [s.daily.strength, s.weekly.strength, s.monthly.strength, s.overall_strength],
        name: s.name || s.code,
        lineStyle: { color: ['#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#a855f7'][i], width: 2 },
        areaStyle: { color: ['#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#a855f7'][i], opacity: 0.08 },
        itemStyle: { color: ['#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#a855f7'][i] },
        symbol: 'circle',
        symbolSize: 4,
      })),
    }],
  } : null;

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <GitBranch className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">多周期趋势</h1>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '180px' }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <GitBranch className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">多周期趋势</h1>
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
          <GitBranch className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">多周期趋势</h1>
          <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>
            {stocks.length} 只
          </span>
          {data?.cached && (
            <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'rgba(245,158,11,0.12)', color: '#f59e0b' }}>缓存</span>
          )}
        </div>
        <button onClick={fetchData} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      {stocks.length === 0 ? (
        <div className="text-center py-16 rounded-xl" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
          <GitBranch className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>暂无多周期趋势数据，请先添加自选股</p>
        </div>
      ) : (
        <>
          {/* Radar chart */}
          {radarOption && (
            <div className="glass-panel rounded-xl p-4 mb-6" style={{ borderColor: 'var(--color-border)' }}>
              <h3 className="text-sm font-medium mb-2" style={{ color: 'var(--color-text-secondary)' }}>Top 5 多周期强度雷达</h3>
              <ReactECharts option={radarOption} style={{ height: '300px' }} />
            </div>
          )}

          {/* Stock cards */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {stocks.map((s) => (
              <div key={s.code} className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
                {/* Header */}
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{s.name}</span>
                    <span className="text-xs font-mono" style={{ color: 'var(--color-text-muted)' }}>{s.code}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs px-2 py-0.5 rounded-full font-medium" style={{ background: `${strengthColor(s.overall_strength)}18`, color: strengthColor(s.overall_strength) }}>
                      {strengthLabel(s.overall_strength)} {s.overall_strength}
                    </span>
                  </div>
                </div>

                {/* Period comparison table */}
                <div className="overflow-x-auto">
                  <table className="w-full text-xs">
                    <thead>
                      <tr style={{ borderBottom: '1px solid rgba(148,163,184,0.1)' }}>
                        <th className="py-1.5 text-left font-medium" style={{ color: 'var(--color-text-muted)' }}>周期</th>
                        <th className="py-1.5 text-center font-medium" style={{ color: 'var(--color-text-muted)' }}>涨跌幅</th>
                        <th className="py-1.5 text-center font-medium" style={{ color: 'var(--color-text-muted)' }}>强度</th>
                        <th className="py-1.5 text-center font-medium" style={{ color: 'var(--color-text-muted)' }}>RSI</th>
                        <th className="py-1.5 text-center font-medium" style={{ color: 'var(--color-text-muted)' }}>MA排列</th>
                        <th className="py-1.5 text-center font-medium" style={{ color: 'var(--color-text-muted)' }}>量能</th>
                      </tr>
                    </thead>
                    <tbody>
                      {PERIOD_KEYS.map((key) => {
                        const p = s[key];
                        return (
                          <tr key={key} style={{ borderBottom: '1px solid rgba(148,163,184,0.05)' }}>
                            <td className="py-1.5 font-medium" style={{ color: 'var(--color-text-secondary)' }}>{PERIOD_LABELS[key]}</td>
                            <td className="py-1.5 text-center font-mono" style={{ color: p.return_pct > 0 ? '#ef4444' : p.return_pct < 0 ? '#22c55e' : 'var(--color-text-muted)' }}>
                              {p.return_pct > 0 ? '+' : ''}{p.return_pct.toFixed(2)}%
                            </td>
                            <td className="py-1.5 text-center">
                              <span className="font-mono font-bold" style={{ color: strengthColor(p.strength) }}>{p.strength}</span>
                            </td>
                            <td className="py-1.5 text-center font-mono" style={{ color: p.rsi > 70 ? '#ef4444' : p.rsi < 30 ? '#22c55e' : 'var(--color-text-secondary)' }}>
                              {p.rsi > 0 ? p.rsi.toFixed(1) : '—'}
                            </td>
                            <td className="py-1.5 text-center">
                              {p.ma_aligned ? (
                                <span style={{ color: '#ef4444' }}>✓ 多头</span>
                              ) : (
                                <span style={{ color: 'var(--color-text-muted)' }}>—</span>
                              )}
                            </td>
                            <td className="py-1.5 text-center" style={{ color: volTrendColor(p.volume_trend) }}>
                              {volTrendLabel(p.volume_trend)}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>

                {/* MA values */}
                <div className="flex gap-3 mt-2 pt-2 text-xs font-mono" style={{ borderTop: '1px solid rgba(148,163,184,0.06)', color: 'var(--color-text-muted)' }}>
                  <span>日MA5:{s.daily.ma5 > 0 ? s.daily.ma5.toFixed(2) : '—'}</span>
                  <span>MA10:{s.daily.ma10 > 0 ? s.daily.ma10.toFixed(2) : '—'}</span>
                  <span>MA20:{s.daily.ma20 > 0 ? s.daily.ma20.toFixed(2) : '—'}</span>
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
