import { useState, useEffect, useRef, useCallback } from 'react';
import { useView } from '@/lib/ViewContext';
import { watchlistApi, marketApi, type WatchlistItem, type Quote } from '@/lib/api';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';

interface TickerQuote {
  code: string;
  name: string;
  price: number;
  change: number;
  changePercent: number;
}

export default function TickerTape() {
  const [quotes, setQuotes] = useState<TickerQuote[]>([]);
  const [loading, setLoading] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);
  const { navigate } = useView();

  const fetchQuotes = useCallback(async () => {
    try {
      const wlRes = await watchlistApi.list();
      const items: WatchlistItem[] = wlRes.data || [];
      if (items.length === 0) {
        setQuotes([]);
        setLoading(false);
        return;
      }

      // Fetch quotes for each watchlist item (max 20 to keep it fast)
      const top = items.slice(0, 20);
      const results = await Promise.allSettled(
        top.map((item) => marketApi.quote(item.code))
      );

      const tickerQuotes: TickerQuote[] = results
        .map((r, i) => {
          if (r.status === 'fulfilled') {
            const q: Quote = r.value.data;
            return {
              code: q.code,
              name: q.name || top[i].code,
              price: q.price,
              change: q.change,
              changePercent: q.change_percent,
            };
          }
          return null;
        })
        .filter((q): q is TickerQuote => q !== null && q.price > 0);

      setQuotes(tickerQuotes);
    } catch {
      // Silently fail — ticker is non-critical
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchQuotes();
    // Refresh every 60 seconds
    const interval = setInterval(fetchQuotes, 60000);
    return () => clearInterval(interval);
  }, [fetchQuotes]);

  if (loading || quotes.length === 0) return null;

  const formatChange = (val: number) => {
    const prefix = val > 0 ? '+' : '';
    return `${prefix}${val.toFixed(2)}%`;
  };

  const getColor = (val: number) => {
    if (val > 0) return 'var(--color-danger)';   // 红涨
    if (val < 0) return 'var(--color-success)';  // 绿跌
    return 'var(--color-text-muted)';
  };

  const getIcon = (val: number) => {
    if (val > 0) return <TrendingUp className="w-3 h-3" />;
    if (val < 0) return <TrendingDown className="w-3 h-3" />;
    return <Minus className="w-3 h-3" />;
  };

  // Double the items for seamless looping
  const allQuotes = [...quotes, ...quotes];

  return (
    <div
      ref={containerRef}
      className="relative overflow-hidden border-b"
      style={{
        height: '32px',
        background: 'rgba(15, 23, 42, 0.6)',
        borderColor: 'var(--color-border)',
      }}
    >
      <div
        className="flex items-center gap-6 whitespace-nowrap ticker-scroll"
      >
        {allQuotes.map((q, i) => (
          <button
            key={`${q.code}-${i}`}
            onClick={() => navigate('kline', { code: q.code })}
            className="flex items-center gap-1.5 px-2 py-1 rounded text-xs hover:bg-[var(--color-bg-hover)] transition-colors cursor-pointer"
          >
            <span style={{ color: 'var(--color-text-secondary)' }}>{q.name}</span>
            <span className="font-mono font-medium" style={{ color: getColor(q.changePercent) }}>
              {q.price.toFixed(2)}
            </span>
            <span className="flex items-center gap-0.5 font-mono" style={{ color: getColor(q.changePercent) }}>
              {getIcon(q.changePercent)}
              {formatChange(q.changePercent)}
            </span>
          </button>
        ))}
      </div>

      {/* Gradient fades on edges */}
      <div
        className="absolute inset-y-0 left-0 w-8 pointer-events-none"
        style={{ background: 'linear-gradient(90deg, rgba(15,23,42,0.9), transparent)' }}
      />
      <div
        className="absolute inset-y-0 right-0 w-8 pointer-events-none"
        style={{ background: 'linear-gradient(270deg, rgba(15,23,42,0.9), transparent)' }}
      />
    </div>
  );
}
