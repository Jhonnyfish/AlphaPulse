import { useState, useEffect, useCallback, useRef } from 'react';
import { watchlistApi, marketApi, type WatchlistItem, type Quote, type SearchSuggestion } from '@/lib/api';
import { Trash2, RefreshCw, ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-react';
import { Link } from 'react-router-dom';
import StockSearch from '@/components/StockSearch';

const REFRESH_INTERVAL = 30_000; // 30 seconds

export default function WatchlistPage() {
  const [items, setItems] = useState<WatchlistItem[]>([]);
  const [quotes, setQuotes] = useState<Map<string, Quote>>(new Map());
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [sortField, setSortField] = useState<'code' | 'name' | 'price' | 'change_percent' | 'change'>('code');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const intervalRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

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
    setLastUpdated(new Date());
  }, []);


  useEffect(() => {
    fetchWatchlist();
  }, [fetchWatchlist]);

  useEffect(() => {
    if (items.length > 0) {
      fetchQuotes(items.map((i) => i.code));
    }
  }, [items, fetchQuotes]);

  // Auto-refresh polling
  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(() => {
        if (items.length > 0) {
          fetchQuotes(items.map((i) => i.code));
        }
      }, REFRESH_INTERVAL);
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [autoRefresh, items, fetchQuotes]);

  const handleAddFromSearch = async (suggestion: SearchSuggestion) => {
    setAdding(true);
    setError('');
    try {
      await watchlistApi.add(suggestion.code);
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

  const formatPrice = (n: number) => n.toFixed(2);
  const formatChange = (n: number) => (n >= 0 ? `+${n.toFixed(2)}` : n.toFixed(2));
  const formatPercent = (n: number) => (n >= 0 ? `+${n.toFixed(2)}%` : `${n.toFixed(2)}%`);
  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

  // Sort toggle handler
  const toggleSort = (field: typeof sortField) => {
    if (sortField === field) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortField(field);
      // Default: desc for numeric fields, asc for text fields
      setSortDir(field === 'code' || field === 'name' ? 'asc' : 'desc');
    }
  };

  // Compute sorted items
  const sortedItems = [...items].sort((a, b) => {
    const qa = quotes.get(a.code);
    const qb = quotes.get(b.code);
    let va: number | string, vb: number | string;

    switch (sortField) {
      case 'code':
        va = a.code; vb = b.code;
        break;
      case 'name':
        va = qa?.name || a.name || ''; vb = qb?.name || b.name || '';
        break;
      case 'price':
        va = qa?.price ?? -Infinity; vb = qb?.price ?? -Infinity;
        break;
      case 'change_percent':
        va = qa?.change_percent ?? -Infinity; vb = qb?.change_percent ?? -Infinity;
        break;
      case 'change':
        va = qa?.change ?? -Infinity; vb = qb?.change ?? -Infinity;
        break;
    }

    if (typeof va === 'string' && typeof vb === 'string') {
      return sortDir === 'asc' ? va.localeCompare(vb) : vb.localeCompare(va);
    }
    return sortDir === 'asc' ? (va as number) - (vb as number) : (vb as number) - (va as number);
  });

  // Sort indicator component
  const SortIcon = ({ field }: { field: typeof sortField }) => {
    if (sortField !== field) return <ArrowUpDown className="w-3 h-3 opacity-30" />;
    return sortDir === 'asc' ? <ArrowUp className="w-3 h-3" /> : <ArrowDown className="w-3 h-3" />;
  };

  // Table header cell with sort
  const Th = ({ field, label, align = 'left' }: { field: typeof sortField; label: string; align?: 'left' | 'right' }) => (
    <th
      className={`px-4 py-3 font-medium cursor-pointer select-none hover:text-[var(--color-text-primary)] transition-colors ${align === 'right' ? 'text-right' : 'text-left'}`}
      style={{ color: sortField === field ? 'var(--color-text-primary)' : 'var(--color-text-secondary)' }}
      onClick={() => toggleSort(field)}
    >
      <span className={`inline-flex items-center gap-1 ${align === 'right' ? 'justify-end' : ''}`}>
        {label}
        <SortIcon field={field} />
      </span>
    </th>
  );

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold">自选股</h1>
        <div className="flex items-center gap-2">
          {/* Auto-refresh toggle */}
          <button
            onClick={() => setAutoRefresh((prev) => !prev)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors"
            style={{
              background: autoRefresh ? 'rgba(59,130,246,0.15)' : 'transparent',
              color: autoRefresh ? 'var(--color-accent)' : 'var(--color-text-muted)',
              border: `1px solid ${autoRefresh ? 'var(--color-accent)' : 'var(--color-border)'}`,
            }}
          >
            <span className={`w-2 h-2 rounded-full ${autoRefresh ? 'bg-green-400 animate-pulse' : 'bg-gray-500'}`} />
            {autoRefresh ? '自动刷新' : '已暂停'}
          </button>
          <button
            onClick={() => {
              if (items.length > 0) fetchQuotes(items.map((i) => i.code));
            }}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm hover:bg-[var(--color-bg-hover)] transition-colors"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            <RefreshCw className="w-3.5 h-3.5" />
            刷新行情
          </button>
        </div>
      </div>

      {/* Search with autocomplete */}
      <div className="mb-4 max-w-sm">
        <StockSearch
          onSelect={handleAddFromSearch}
          placeholder={adding ? '添加中...' : '搜索股票代码或名称，选中即添加...'}
        />
      </div>

      {error && (
        <div className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
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
            搜索股票代码或名称添加到自选列表
          </p>
        </div>
      ) : (
        <>
          {/* Desktop table */}
          <div className="hidden sm:block rounded-lg border overflow-hidden"
            style={{ borderColor: 'var(--color-border)' }}>
            <table className="w-full text-sm">
              <thead>
                <tr style={{ background: 'var(--color-bg-secondary)' }}>
                  <Th field="code" label="代码" />
                  <Th field="name" label="名称" />
                  <Th field="price" label="最新价" align="right" />
                  <Th field="change_percent" label="涨跌幅" align="right" />
                  <Th field="change" label="涨跌额" align="right" />
                  <th className="text-right px-4 py-3 font-medium"
                    style={{ color: 'var(--color-text-secondary)' }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {sortedItems.map((item) => {
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

          {/* Mobile card list */}
          <div className="sm:hidden space-y-2">
            {sortedItems.map((item) => {
              const q = quotes.get(item.code);
              const pct = q?.change_percent ?? 0;
              return (
                <div
                  key={item.id}
                  className="rounded-lg border p-3 flex items-center justify-between"
                  style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
                >
                  <div className="flex-1 min-w-0">
                    <Link to={`/kline?code=${item.code}`} className="font-mono text-sm" style={{ color: 'var(--color-accent)' }}>
                      {item.code}
                    </Link>
                    <div className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>
                      {q?.name || item.name || '—'}
                    </div>
                  </div>
                  <div className="text-right mr-3">
                    <div className="font-mono text-sm font-bold" style={{ color: changeColor(pct) }}>
                      {q ? formatPrice(q.price) : '—'}
                    </div>
                    <div className="font-mono text-xs" style={{ color: changeColor(pct) }}>
                      {q ? formatPercent(pct) : '—'}
                    </div>
                  </div>
                  <button
                    onClick={() => handleRemove(item.code)}
                    className="p-1.5 rounded hover:bg-[var(--color-bg-card)] transition-colors flex-shrink-0"
                    title="删除"
                  >
                    <Trash2 className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
                  </button>
                </div>
              );
            })}
          </div>
        </>
      )}

      {/* Footer info */}
      <div className="mt-4 flex items-center justify-between text-xs" style={{ color: 'var(--color-text-muted)' }}>
        <span>共 {items.length} 只自选股</span>
        <span>
          {lastUpdated && `上次刷新: ${lastUpdated.toLocaleTimeString('zh-CN')}`}
        </span>
      </div>
    </div>
  );
}
