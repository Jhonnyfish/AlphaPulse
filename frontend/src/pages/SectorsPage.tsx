import { useState, useEffect, useCallback } from 'react';
import { marketApi, type Sector } from '@/lib/api';
import { RefreshCw, TrendingUp, TrendingDown, Minus } from 'lucide-react';

export default function SectorsPage() {
  const [sectors, setSectors] = useState<Sector[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

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

  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

  // Sort by change_percent descending
  const sorted = [...sectors].sort((a, b) => b.change_percent - a.change_percent);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold">板块行情</h1>
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
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {sorted.map((sector) => {
            const pct = sector.change_percent;
            const Icon = pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Minus;
            return (
              <div
                key={sector.code}
                className="rounded-lg border p-4 hover:bg-[var(--color-bg-hover)] transition-colors"
                style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="font-medium text-sm">{sector.name}</span>
                  <span className="text-xs font-mono" style={{ color: 'var(--color-text-muted)' }}>
                    {sector.code}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="font-mono font-bold" style={{ color: changeColor(pct) }}>
                    {sector.price.toFixed(2)}
                  </span>
                  <span className="flex items-center gap-1 font-mono text-sm" style={{ color: changeColor(pct) }}>
                    <Icon className="w-3.5 h-3.5" />
                    {pct >= 0 ? '+' : ''}
                    {pct.toFixed(2)}%
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
