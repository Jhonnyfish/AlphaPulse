import { useState, useEffect, useCallback } from 'react';
import { watchlistApi, marketApi, type WatchlistItem, type Quote } from '@/lib/api';
import { Plus, Trash2, Search, RefreshCw } from 'lucide-react';
import { Link } from 'react-router-dom';

export default function WatchlistPage() {
  const [items, setItems] = useState<WatchlistItem[]>([]);
  const [quotes, setQuotes] = useState<Map<string, Quote>>(new Map());
  const [loading, setLoading] = useState(true);
  const [addCode, setAddCode] = useState('');
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState('');

  const fetchWatchlist = useCallback(async () => {
    try {
      const res = await watchlistApi.list();
      setItems(res.data);
    } catch {
      setError('加载自选股失败');
    } finally {
      setLoading(false);
    }
  }, []);

  // Fetch quotes for all watchlist items
  const fetchQuotes = useCallback(async (codes: string[]) => {
    const newQuotes = new Map<string, Quote>();
    await Promise.allSettled(
      codes.map(async (code) => {
        try {
          const res = await marketApi.quote(code);
          newQuotes.set(code, res.data);
        } catch {
          // quote fetch failed — skip
        }
      })
    );
    setQuotes(newQuotes);
  }, []);

  useEffect(() => {
    fetchWatchlist();
  }, [fetchWatchlist]);

  useEffect(() => {
    if (items.length > 0) {
      fetchQuotes(items.map((i) => i.code));
    }
  }, [items, fetchQuotes]);

  const handleAdd = async () => {
    const code = addCode.trim();
    if (!code) return;
    setAdding(true);
    setError('');
    try {
      await watchlistApi.add(code);
      setAddCode('');
      await fetchWatchlist();
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        '添加失败';
      setError(msg);
    } finally {
      setAdding(false);
    }
  };

  const handleRemove = async (code: string) => {
    try {
      await watchlistApi.remove(code);
      await fetchWatchlist();
    } catch {
      setError('删除失败');
    }
  };

  const handleRefresh = () => {
    if (items.length > 0) {
      fetchQuotes(items.map((i) => i.code));
    }
  };

  const formatPrice = (n: number) => n.toFixed(2);
  const formatChange = (n: number) => (n >= 0 ? `+${n.toFixed(2)}` : n.toFixed(2));
  const formatPercent = (n: number) => (n >= 0 ? `+${n.toFixed(2)}%` : `${n.toFixed(2)}%`);
  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold">自选股</h1>
        <button
          onClick={handleRefresh}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm hover:bg-[var(--color-bg-hover)] transition-colors"
          style={{ color: 'var(--color-text-secondary)' }}
        >
          <RefreshCw className="w-3.5 h-3.5" />
          刷新行情
        </button>
      </div>

      {/* Add stock input */}
      <div className="flex gap-2 mb-4">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4"
            style={{ color: 'var(--color-text-muted)' }} />
          <input
            type="text"
            value={addCode}
            onChange={(e) => setAddCode(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
            placeholder="输入股票代码，如 000001"
            className="w-full pl-9 pr-3 py-2 rounded-lg border text-sm outline-none"
            style={{
              background: 'var(--color-bg-card)',
              borderColor: 'var(--color-border)',
              color: 'var(--color-text-primary)',
            }}
          />
        </div>
        <button
          onClick={handleAdd}
          disabled={adding || !addCode.trim()}
          className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium text-white transition-colors disabled:opacity-50"
          style={{ background: 'var(--color-accent)' }}
        >
          <Plus className="w-4 h-4" />
          {adding ? '添加中...' : '添加'}
        </button>
      </div>

      {error && (
        <div className="text-sm px-3 py-2 rounded-lg mb-4"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
        </div>
      )}

      {/* Table */}
      {loading ? (
        <div className="text-center py-12" style={{ color: 'var(--color-text-muted)' }}>
          加载中...
        </div>
      ) : items.length === 0 ? (
        <div className="text-center py-16 rounded-lg border"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}>
          <p style={{ color: 'var(--color-text-muted)' }}>暂无自选股</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            输入股票代码添加到自选列表
          </p>
        </div>
      ) : (
        <div className="rounded-lg border overflow-hidden"
          style={{ borderColor: 'var(--color-border)' }}>
          <table className="w-full text-sm">
            <thead>
              <tr style={{ background: 'var(--color-bg-secondary)' }}>
                <th className="text-left px-4 py-3 font-medium"
                  style={{ color: 'var(--color-text-secondary)' }}>代码</th>
                <th className="text-left px-4 py-3 font-medium"
                  style={{ color: 'var(--color-text-secondary)' }}>名称</th>
                <th className="text-right px-4 py-3 font-medium"
                  style={{ color: 'var(--color-text-secondary)' }}>最新价</th>
                <th className="text-right px-4 py-3 font-medium"
                  style={{ color: 'var(--color-text-secondary)' }}>涨跌幅</th>
                <th className="text-right px-4 py-3 font-medium"
                  style={{ color: 'var(--color-text-secondary)' }}>涨跌额</th>
                <th className="text-right px-4 py-3 font-medium"
                  style={{ color: 'var(--color-text-secondary)' }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => {
                const q = quotes.get(item.code);
                const pct = q?.change_percent ?? 0;
                return (
                  <tr key={item.id}
                    className="hover:bg-[var(--color-bg-hover)] transition-colors border-t"
                    style={{ borderColor: 'var(--color-border)' }}>
                    <td className="px-4 py-3 font-mono">
                      <Link to={`/kline?code=${item.code}`} className="hover:underline" style={{ color: 'var(--color-accent)' }}>
                        {item.code}
                      </Link>
                    </td>
                    <td className="px-4 py-3" style={{ color: 'var(--color-text-secondary)' }}>
                      {q?.name || item.name || '—'}
                    </td>
                    <td className="px-4 py-3 text-right font-mono" style={{ color: changeColor(pct) }}>
                      {q ? formatPrice(q.price) : '—'}
                    </td>
                    <td className="px-4 py-3 text-right font-mono" style={{ color: changeColor(pct) }}>
                      {q ? formatPercent(pct) : '—'}
                    </td>
                    <td className="px-4 py-3 text-right font-mono" style={{ color: changeColor(pct) }}>
                      {q ? formatChange(q.change) : '—'}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        onClick={() => handleRemove(item.code)}
                        className="p-1.5 rounded hover:bg-[var(--color-bg-card)] transition-colors"
                        title="删除"
                      >
                        <Trash2 className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
