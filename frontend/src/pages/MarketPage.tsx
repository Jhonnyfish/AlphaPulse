import { useState, useCallback } from 'react';
import { marketApi, type Quote } from '@/lib/api';
import { Search, TrendingUp, TrendingDown, Minus } from 'lucide-react';

export default function MarketPage() {
  const [code, setCode] = useState('');
  const [quote, setQuote] = useState<Quote | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSearch = useCallback(async () => {
    const c = code.trim();
    if (!c) return;
    setLoading(true);
    setError('');
    setQuote(null);
    try {
      const res = await marketApi.quote(c);
      setQuote(res.data);
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        '查询失败，请检查股票代码';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [code]);

  const pct = quote?.change_percent ?? 0;
  const color =
    pct > 0 ? 'var(--color-danger)' : pct < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';
  const TrendIcon = pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Minus;

  return (
    <div>
      <h1 className="text-xl font-bold mb-6">行情查询</h1>

      {/* Search */}
      <div className="flex gap-2 mb-6">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4"
            style={{ color: 'var(--color-text-muted)' }} />
          <input
            type="text"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            placeholder="输入股票代码，如 600519"
            className="w-full pl-9 pr-3 py-2 rounded-lg border text-sm outline-none"
            style={{
              background: 'var(--color-bg-card)',
              borderColor: 'var(--color-border)',
              color: 'var(--color-text-primary)',
            }}
          />
        </div>
        <button
          onClick={handleSearch}
          disabled={loading || !code.trim()}
          className="px-4 py-2 rounded-lg text-sm font-medium text-white transition-colors disabled:opacity-50"
          style={{ background: 'var(--color-accent)' }}
        >
          {loading ? '查询中...' : '查询'}
        </button>
      </div>

      {error && (
        <div className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
        </div>
      )}

      {/* Quote card */}
      {quote && (
        <div className="rounded-xl border p-6 max-w-lg"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}>
          {/* Header */}
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-bold">{quote.name}</h2>
              <span className="text-sm font-mono" style={{ color: 'var(--color-text-muted)' }}>
                {quote.code}
              </span>
            </div>
            <div className="flex items-center gap-1.5" style={{ color }}>
              <TrendIcon className="w-5 h-5" />
              <span className="text-lg font-bold font-mono">
                {pct >= 0 ? '+' : ''}{pct.toFixed(2)}%
              </span>
            </div>
          </div>

          {/* Price */}
          <div className="mb-6">
            <span className="text-4xl font-bold font-mono" style={{ color }}>
              {quote.price.toFixed(2)}
            </span>
            <span className="ml-3 text-sm font-mono" style={{ color }}>
              {quote.change >= 0 ? '+' : ''}{quote.change.toFixed(2)}
            </span>
          </div>

          {/* Details grid */}
          <div className="grid grid-cols-2 gap-3">
            {[
              { label: '开盘', value: quote.open },
              { label: '昨收', value: quote.prev_close },
              { label: '最高', value: quote.high },
              { label: '最低', value: quote.low },
            ].map(({ label, value }) => (
              <div key={label} className="flex justify-between px-3 py-2 rounded-lg"
                style={{ background: 'var(--color-bg-card)' }}>
                <span className="text-sm" style={{ color: 'var(--color-text-muted)' }}>{label}</span>
                <span className="text-sm font-mono">{value.toFixed(2)}</span>
              </div>
            ))}
          </div>

          {/* Updated time */}
          <div className="mt-4 text-xs" style={{ color: 'var(--color-text-muted)' }}>
            更新时间: {quote.updated_at}
          </div>
        </div>
      )}

      {/* Empty state */}
      {!quote && !error && !loading && (
        <div className="text-center py-16 rounded-lg border max-w-lg"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}>
          <p style={{ color: 'var(--color-text-muted)' }}>输入股票代码查询实时行情</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            例如: 000001 (平安银行), 600519 (贵州茅台)
          </p>
        </div>
      )}
    </div>
  );
}
