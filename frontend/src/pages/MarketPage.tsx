import { useState, useCallback } from 'react';
import { marketApi, type Quote, type SearchSuggestion } from '@/lib/api';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import StockSearch from '@/components/StockSearch';

export default function MarketPage() {
  const [quote, setQuote] = useState<Quote | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSelect = useCallback(async (suggestion: SearchSuggestion) => {
    setLoading(true);
    setError('');
    setQuote(null);
    try {
      const res = await marketApi.quote(suggestion.code);
      setQuote(res.data);
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        '查询失败，请检查股票代码';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  const pct = quote?.change_percent ?? 0;
  const color =
    pct > 0 ? 'var(--color-danger)' : pct < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';
  const TrendIcon = pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Minus;

  return (
    <div>
      <h1 className="text-xl font-bold mb-6">行情查询</h1>

      {/* Search */}
      <div className="mb-6 max-w-sm">
        <StockSearch
          onSelect={handleSelect}
          placeholder="搜索股票代码或名称..."
        />
      </div>

      {loading && (
        <div className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
          style={{ background: 'rgba(59,130,246,0.1)', color: 'var(--color-accent)' }}>
          查询中...
        </div>
      )}

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
          <p style={{ color: 'var(--color-text-muted)' }}>搜索股票查看实时行情</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            例如: 平安银行、贵州茅台、000001
          </p>
        </div>
      )}
    </div>
  );
}
