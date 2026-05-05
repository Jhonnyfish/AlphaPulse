import { useState, useEffect } from 'react';
import api from '@/lib/api';
import ReactECharts from '@/components/charts/ReactECharts';
import { TrendingUp, TrendingDown, Minus, BarChart3 } from 'lucide-react';

interface TrendStock {
  code: string;
  name: string;
  change_1d: number | null;
  change_5d: number | null;
  change_20d: number | null;
  change_30d: number | null;
  kline_data: number[];
}

interface TrendsData {
  indices: TrendStock[];
  watchlist_stocks: TrendStock[];
}

function TrendCard({ stock }: { stock: TrendStock }) {
  // Returns CSS variable for JSX style props
  const getChangeColorCSS = (val: number | null) => {
    if (val === null || val === undefined) return 'var(--color-text-muted)';
    if (val > 0) return 'var(--color-danger)';
    if (val < 0) return 'var(--color-success)';
    return 'var(--color-text-muted)';
  };

  // Returns hex color for ECharts canvas (CSS vars don't work in canvas)
  const getChangeColorHex = (val: number | null) => {
    if (val === null || val === undefined) return '#94a3b8';
    if (val > 0) return '#ef4444';
    if (val < 0) return '#22c55e';
    return '#94a3b8';
  };

  const formatChange = (val: number | null) => {
    if (val === null || val === undefined) return '—';
    const prefix = val > 0 ? '+' : '';
    return `${prefix}${val.toFixed(2)}%`;
  };

  // Mini sparkline from kline_data
  const sparkOption = stock.kline_data && stock.kline_data.length > 1
    ? {
        grid: { top: 2, right: 0, bottom: 2, left: 0 },
        xAxis: { type: 'category' as const, show: false, data: stock.kline_data.map((_, i) => i) },
        yAxis: { type: 'value' as const, show: false },
        series: [{
          type: 'line' as const,
          data: stock.kline_data,
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 1.5, color: getChangeColorHex(stock.change_1d) },
          areaStyle: {
            color: {
              type: 'linear' as const,
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: `${getChangeColorHex(stock.change_1d)}33` },
                { offset: 1, color: 'transparent' },
              ],
            },
          },
        }],
      }
    : null;

  return (
    <div
      className="glass-panel p-4 rounded-xl hover:border-[var(--color-accent)] transition-all"
      style={{ borderColor: 'var(--color-border)' }}
    >
      <div className="flex items-start justify-between mb-2">
        <div>
          <div className="font-medium text-sm">{stock.name || stock.code}</div>
          <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{stock.code}</div>
        </div>
        <div className="flex items-center gap-1" style={{ color: getChangeColorCSS(stock.change_1d) }}>
          {stock.change_1d !== null && stock.change_1d > 0 && <TrendingUp className="w-3.5 h-3.5" />}
          {stock.change_1d !== null && stock.change_1d < 0 && <TrendingDown className="w-3.5 h-3.5" />}
          {stock.change_1d !== null && stock.change_1d === 0 && <Minus className="w-3.5 h-3.5" />}
          <span className="text-sm font-mono font-medium">{formatChange(stock.change_1d)}</span>
        </div>
      </div>

      {sparkOption && (
        <div className="h-10 mb-2">
          <ReactECharts
            option={sparkOption}
            style={{ height: '100%', width: '100%' }}
            opts={{ renderer: 'svg' }}
          />
        </div>
      )}

      <div className="grid grid-cols-3 gap-2 text-xs">
        <div>
          <div style={{ color: 'var(--color-text-muted)' }}>5日</div>
          <div className="font-mono" style={{ color: getChangeColorCSS(stock.change_5d) }}>
            {formatChange(stock.change_5d)}
          </div>
        </div>
        <div>
          <div style={{ color: 'var(--color-text-muted)' }}>20日</div>
          <div className="font-mono" style={{ color: getChangeColorCSS(stock.change_20d) }}>
            {formatChange(stock.change_20d)}
          </div>
        </div>
        <div>
          <div style={{ color: 'var(--color-text-muted)' }}>30日</div>
          <div className="font-mono" style={{ color: getChangeColorCSS(stock.change_30d) }}>
            {formatChange(stock.change_30d)}
          </div>
        </div>
      </div>
    </div>
  );
}

export default function TrendsPage() {
  const [data, setData] = useState<TrendsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api
      .get<TrendsData>('/market/trends')
      .then((res) => setData(res.data))
      .catch(() => setError('加载趋势数据失败'))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center gap-2 mb-6">
          <BarChart3 className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">市场趋势</h1>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="glass-panel p-4 rounded-xl animate-pulse"
              style={{ height: '140px', borderColor: 'var(--color-border)' }}
            >
              <div className="h-4 rounded mb-2" style={{ background: 'var(--color-bg-hover)', width: '60%' }} />
              <div className="h-3 rounded mb-4" style={{ background: 'var(--color-bg-hover)', width: '40%' }} />
              <div className="h-10 rounded" style={{ background: 'var(--color-bg-hover)' }} />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <BarChart3 className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">市场趋势</h1>
        </div>
        <div
          className="text-sm px-4 py-3 rounded-lg"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
        >
          {error}
          <button
            onClick={() => { setLoading(true); setError(''); api.get<TrendsData>('/market/trends').then(r => setData(r.data)).catch(() => setError('加载失败')).finally(() => setLoading(false)); }}
            className="ml-3 underline"
          >
            重试
          </button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center gap-2 mb-6">
        <BarChart3 className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
        <h1 className="text-xl font-bold">市场趋势</h1>
      </div>

      {/* Indices */}
      {data?.indices && data.indices.length > 0 && (
        <div className="mb-6">
          <h2 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
            大盘指数
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data.indices.map((idx) => (
              <TrendCard key={idx.code} stock={idx} />
            ))}
          </div>
        </div>
      )}

      {/* Watchlist stocks */}
      {data?.watchlist_stocks && data.watchlist_stocks.length > 0 && (
        <div>
          <h2 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
            自选股趋势
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data.watchlist_stocks.map((stock) => (
              <TrendCard key={stock.code} stock={stock} />
            ))}
          </div>
        </div>
      )}

      {/* Empty state */}
      {(!data?.indices || data.indices.length === 0) &&
       (!data?.watchlist_stocks || data.watchlist_stocks.length === 0) && (
        <div
          className="text-center py-16 rounded-xl"
          style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}
        >
          <BarChart3 className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>暂无趋势数据</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            请在自选股中添加股票后查看
          </p>
        </div>
      )}
    </div>
  );
}
