import { useState, useEffect, useCallback } from 'react';
import { marketApi, type Sector } from '@/lib/api';
import { RefreshCw, TrendingUp, TrendingDown, Minus, LayoutGrid, List } from 'lucide-react';

// Interpolate color based on change percent
function heatColor(pct: number): string {
  // Clamp to [-5, 5] range for color intensity
  const clamped = Math.max(-5, Math.min(5, pct));
  const intensity = Math.abs(clamped) / 5; // 0..1

  if (pct > 0) {
    // Red spectrum (Chinese market: red = up)
    const r = Math.round(180 + 75 * intensity);
    const g = Math.round(60 - 30 * intensity);
    const b = Math.round(50 - 20 * intensity);
    return `rgb(${r}, ${g}, ${b})`;
  } else if (pct < 0) {
    // Green spectrum (Chinese market: green = down)
    const r = Math.round(40 - 15 * intensity);
    const g = Math.round(150 + 80 * intensity);
    const b = Math.round(70 + 30 * intensity);
    return `rgb(${r}, ${g}, ${b})`;
  }
  return 'var(--color-bg-card)';
}

function heatTextColor(pct: number): string {
  const absPct = Math.abs(pct);
  return absPct > 1.5 ? '#fff' : 'var(--color-text-primary)';
}

export default function SectorsPage() {
  const [sectors, setSectors] = useState<Sector[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [viewMode, setViewMode] = useState<'heatmap' | 'list'>('heatmap');

  const fetchSectors = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await marketApi.sectors();
      setSectors(res.data);
    } catch {
      setError('加载板块数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSectors();
  }, [fetchSectors]);

  // Sort by change_percent descending
  const sorted = [...sectors].sort((a, b) => b.change_percent - a.change_percent);

  // Summary stats
  const upCount = sectors.filter((s) => s.change_percent > 0).length;
  const downCount = sectors.filter((s) => s.change_percent < 0).length;
  const flatCount = sectors.filter((s) => s.change_percent === 0).length;
  const maxGainer = sorted[0];
  const maxLoser = sorted[sorted.length - 1];

  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-bold">板块行情</h1>
          <p className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>
            颜色深浅反映涨跌幅度，红色上涨，绿色下跌
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* View toggle */}
          <div className="flex rounded-lg overflow-hidden border" style={{ borderColor: 'var(--color-border)' }}>
            <button
              onClick={() => setViewMode('heatmap')}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm transition-colors"
              style={{
                background: viewMode === 'heatmap' ? 'var(--color-accent)' : 'var(--color-bg-card)',
                color: viewMode === 'heatmap' ? '#fff' : 'var(--color-text-muted)',
              }}
            >
              <LayoutGrid className="w-3.5 h-3.5" />
              热力图
            </button>
            <button
              onClick={() => setViewMode('list')}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm transition-colors"
              style={{
                background: viewMode === 'list' ? 'var(--color-accent)' : 'var(--color-bg-card)',
                color: viewMode === 'list' ? '#fff' : 'var(--color-text-muted)',
              }}
            >
              <List className="w-3.5 h-3.5" />
              列表
            </button>
          </div>
          <button
            onClick={fetchSectors}
            disabled={loading}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm hover:bg-[var(--color-bg-hover)] transition-colors"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </button>
        </div>
      </div>

      {/* Summary bar */}
      {sectors.length > 0 && (
        <div
          className="flex items-center gap-4 mb-4 px-4 py-2.5 rounded-lg text-xs"
          style={{ background: 'var(--color-bg-secondary)' }}
        >
          <span style={{ color: 'var(--color-text-muted)' }}>共 {sectors.length} 个板块</span>
          <span style={{ color: 'var(--color-danger)' }}>
            <TrendingUp className="w-3 h-3 inline mr-1" />
            上涨 {upCount}
          </span>
          <span style={{ color: 'var(--color-success)' }}>
            <TrendingDown className="w-3 h-3 inline mr-1" />
            下跌 {downCount}
          </span>
          {flatCount > 0 && (
            <span style={{ color: 'var(--color-text-muted)' }}>
              <Minus className="w-3 h-3 inline mr-1" />
              平盘 {flatCount}
            </span>
          )}
          {maxGainer && (
            <span style={{ color: 'var(--color-danger)' }}>
              领涨: {maxGainer.name} +{maxGainer.change_percent.toFixed(2)}%
            </span>
          )}
          {maxLoser && maxLoser.change_percent < 0 && (
            <span style={{ color: 'var(--color-success)' }}>
              领跌: {maxLoser.name} {maxLoser.change_percent.toFixed(2)}%
            </span>
          )}
        </div>
      )}

      {error && (
        <div
          className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
        >
          {error}
        </div>
      )}

      {loading && sectors.length === 0 ? (
        <div className="text-center py-12" style={{ color: 'var(--color-text-muted)' }}>
          加载中...
        </div>
      ) : sorted.length === 0 ? (
        <div
          className="text-center py-16 rounded-lg border"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <p style={{ color: 'var(--color-text-muted)' }}>暂无板块数据</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            非交易时间或网络异常时无法获取数据
          </p>
        </div>
      ) : viewMode === 'heatmap' ? (
        /* ── Heatmap / Treemap view ── */
        <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 gap-1">
          {sorted.map((sector) => {
            const pct = sector.change_percent;
            const bg = heatColor(pct);
            const fg = heatTextColor(pct);
            const absPct = Math.abs(pct);

            return (
              <div
                key={sector.code}
                className="rounded-md p-2.5 flex flex-col justify-between cursor-default transition-transform hover:scale-[1.03] hover:z-10 min-h-[72px]"
                style={{ background: bg }}
                title={`${sector.name} (${sector.code})\n价格: ${sector.price.toFixed(2)}\n涨跌幅: ${pct >= 0 ? '+' : ''}${pct.toFixed(2)}%`}
              >
                <div className="text-xs font-medium truncate" style={{ color: fg }}>
                  {sector.name}
                </div>
                <div className="mt-auto">
                  <div className="text-sm font-bold font-mono" style={{ color: fg }}>
                    {pct >= 0 ? '+' : ''}
                    {pct.toFixed(2)}%
                  </div>
                  {absPct >= 2 && (
                    <div className="text-[10px] font-mono mt-0.5" style={{ color: fg, opacity: 0.7 }}>
                      {sector.price.toFixed(2)}
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        /* ── List / Table view ── */
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {sorted.map((sector) => {
            const pct = sector.change_percent;
            const Icon = pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Minus;
            const bg = heatColor(pct);

            return (
              <div
                key={sector.code}
                className="rounded-lg border p-3 flex items-center gap-3 hover:bg-[var(--color-bg-hover)] transition-colors"
                style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
              >
                {/* Color indicator bar */}
                <div
                  className="w-1 self-stretch rounded-full flex-shrink-0"
                  style={{ background: bg }}
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <span className="font-medium text-sm truncate">{sector.name}</span>
                    <span className="text-xs font-mono ml-2 flex-shrink-0" style={{ color: 'var(--color-text-muted)' }}>
                      {sector.code}
                    </span>
                  </div>
                  <div className="flex items-center justify-between mt-1">
                    <span className="font-mono text-sm" style={{ color: 'var(--color-text-secondary)' }}>
                      {sector.price.toFixed(2)}
                    </span>
                    <span className="flex items-center gap-1 font-mono text-sm font-medium" style={{ color: changeColor(pct) }}>
                      <Icon className="w-3.5 h-3.5" />
                      {pct >= 0 ? '+' : ''}
                      {pct.toFixed(2)}%
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Color legend */}
      {viewMode === 'heatmap' && sectors.length > 0 && (
        <div className="flex items-center justify-center gap-2 mt-4">
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>跌</span>
          <div className="flex h-2 rounded-full overflow-hidden" style={{ width: 200 }}>
            {[-5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5].map((v) => (
              <div
                key={v}
                className="flex-1"
                style={{ background: heatColor(v) }}
              />
            ))}
          </div>
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>涨</span>
        </div>
      )}
    </div>
  );
}
