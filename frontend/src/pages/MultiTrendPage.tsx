import { useState, useEffect } from 'react';
import api from '@/lib/api';
import ReactECharts from '@/components/charts/ReactECharts';
import { GitBranch, RefreshCw, TrendingUp, TrendingDown } from 'lucide-react';

interface TrendStock {
  code: string;
  name: string;
  price: number;
  change_pct: number;
  ma5: number;
  ma10: number;
  ma20: number;
  ma60: number;
  trend_score: number;
  short_trend: string;
  mid_trend: string;
  long_trend: string;
}

interface MultiTrendData {
  ok: boolean;
  stocks: TrendStock[];
  cached: boolean;
}

export default function MultiTrendPage() {
  const [data, setData] = useState<MultiTrendData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = () => {
    setLoading(true);
    setError('');
    api.get<MultiTrendData>('/multi-trend')
      .then((res) => setData(res.data))
      .catch(() => setError('加载多周期趋势数据失败'))
      .finally(() => setLoading(false));
  };

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { fetchData(); }, []);

  const getTrendColor = (trend: string) => {
    if (trend === 'up' || trend === 'bullish') return 'var(--color-danger)';
    if (trend === 'down' || trend === 'bearish') return 'var(--color-success)';
    return 'var(--color-text-muted)';
  };

  const getTrendLabel = (trend: string) => {
    const map: Record<string, string> = { up: '看涨', down: '看跌', bullish: '看涨', bearish: '看跌', neutral: '震荡', sideways: '震荡' };
    return map[trend] || trend;
  };

  const getTrendIcon = (trend: string) => {
    if (trend === 'up' || trend === 'bullish') return <TrendingUp className="w-3.5 h-3.5" />;
    if (trend === 'down' || trend === 'bearish') return <TrendingDown className="w-3.5 h-3.5" />;
    return null;
  };

  // Scatter chart: trend score vs change_pct
  const scatterChart = data?.stocks && data.stocks.length > 0 ? {
    tooltip: {
      trigger: 'item' as const,
      formatter: (p: { data: [number, number, string, number] }) => {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const [x, _y, name, score] = p.data;
        return `${name}<br/>涨跌幅: ${(x ?? 0).toFixed(2)}%<br/>趋势评分: ${score}`;
      },
    },
    grid: { left: '5%', right: '5%', bottom: '5%', top: '5%', containLabel: true },
    xAxis: {
      type: 'value' as const,
      name: '涨跌幅%',
      axisLabel: { color: '#94a3b8' },
      splitLine: { lineStyle: { color: 'rgba(148,163,184,0.1)' } },
    },
    yAxis: {
      type: 'value' as const,
      name: '趋势评分',
      axisLabel: { color: '#94a3b8' },
      splitLine: { lineStyle: { color: 'rgba(148,163,184,0.1)' } },
    },
    series: [{
      type: 'scatter' as const,
      data: data.stocks.map(s => [s.change_pct, s.trend_score, s.name, s.trend_score]),
      symbolSize: 12,
      itemStyle: {
        color: (params: { data: [number, number, string, number] }) => {
          const changePct = params.data[0];
          return changePct > 0 ? '#ef4444' : changePct < 0 ? '#22c55e' : '#6b7280';
        },
      },
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
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '100px' }} />
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

  const stocks = data?.stocks || [];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <GitBranch className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">多周期趋势</h1>
          {data?.cached && (
            <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>缓存</span>
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
          {/* Scatter chart */}
          {scatterChart && (
            <div className="glass-panel rounded-xl p-4 mb-6" style={{ borderColor: 'var(--color-border)' }}>
              <h3 className="text-sm font-medium mb-2" style={{ color: 'var(--color-text-secondary)' }}>趋势评分 vs 涨跌幅</h3>
              <ReactECharts option={scatterChart} style={{ height: '300px' }} />
            </div>
          )}

          {/* Stock cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {stocks.map((s) => (
              <div key={s.code} className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <span className="font-medium">{s.name}</span>
                    <span className="text-xs ml-2 font-mono" style={{ color: 'var(--color-text-muted)' }}>{s.code}</span>
                  </div>
                  <span className="text-lg font-bold font-mono" style={{ color: s.change_pct > 0 ? 'var(--color-danger)' : s.change_pct < 0 ? 'var(--color-success)' : 'var(--color-text-muted)' }}>
                    {s.change_pct > 0 ? '+' : ''}{(s.change_pct ?? 0).toFixed(2)}%
                  </span>
                </div>
                <div className="text-sm font-mono mb-3" style={{ color: 'var(--color-text-secondary)' }}>
                  ¥{(s.price ?? 0).toFixed(2)}
                </div>
                <div className="grid grid-cols-3 gap-2">
                  {[
                    { label: '短期', trend: s.short_trend },
                    { label: '中期', trend: s.mid_trend },
                    { label: '长期', trend: s.long_trend },
                  ].map(({ label, trend }) => (
                    <div key={label} className="text-center p-2 rounded-lg" style={{ background: 'var(--color-bg-hover)' }}>
                      <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>{label}</div>
                      <div className="flex items-center justify-center gap-1 text-sm font-medium" style={{ color: getTrendColor(trend) }}>
                        {getTrendIcon(trend)}
                        {getTrendLabel(trend)}
                      </div>
                    </div>
                  ))}
                </div>
                <div className="flex items-center justify-between mt-3 pt-2" style={{ borderTop: '1px solid var(--color-border)' }}>
                  <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>趋势评分</span>
                  <span className="text-sm font-bold font-mono">{s.trend_score}</span>
                </div>
                <div className="flex gap-2 mt-2 text-xs font-mono" style={{ color: 'var(--color-text-muted)' }}>
                  <span>MA5:{(s.ma5 ?? 0).toFixed(2)}</span>
                  <span>MA10:{(s.ma10 ?? 0).toFixed(2)}</span>
                  <span>MA20:{(s.ma20 ?? 0).toFixed(2)}</span>
                  <span>MA60:{(s.ma60 ?? 0).toFixed(2)}</span>
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
