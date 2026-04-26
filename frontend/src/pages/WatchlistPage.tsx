import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
} from '@dnd-kit/core';
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
  arrayMove,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { watchlistApi, marketApi, type WatchlistItem, type Quote, type SearchSuggestion } from '@/lib/api';
import { Trash2, RefreshCw, ArrowUp, ArrowDown, ArrowUpDown, GripVertical } from 'lucide-react';
import { Link } from 'react-router-dom';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import StockSearch from '@/components/StockSearch';

const REFRESH_INTERVAL = 30_000; // 30 seconds

// ─── Formatting helpers (pure) ─────────────────────────────────────────────
const formatPrice = (n: number) => n.toFixed(2);
const formatChange = (n: number) => (n >= 0 ? `+${n.toFixed(2)}` : n.toFixed(2));
const formatPercent = (n: number) => (n >= 0 ? `+${n.toFixed(2)}%` : `${n.toFixed(2)}%`);
const changeColor = (n: number) =>
  n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

// Sparkline line color: red for up, green for down (红涨绿跌)
const sparklineColor = (pct: number): string =>
  pct > 0 ? '#ef4444' : pct < 0 ? '#22c55e' : '#94a3b8';

// ─── Minimal Sparkline component ─────────────────────────────────────────────
function Sparkline({ data, color }: { data: number[]; color: string }) {
  const option = useMemo<EChartsOption>(() => ({
    animation: false,
    grid: { top: 0, right: 0, bottom: 0, left: 0 },
    xAxis: { type: 'category', show: false, boundaryGap: false },
    yAxis: { type: 'value', show: false, min: 'dataMin', max: 'dataMax' },
    series: [
      {
        type: 'line',
        data,
        symbol: 'none',
        lineStyle: { width: 1.5, color },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: color.replace(')', ',0.25)').replace('rgb', 'rgba') },
              { offset: 1, color: 'transparent' },
            ],
          },
        },
        smooth: true,
      },
    ],
  }), [data, color]);

  return (
    <ReactECharts
      option={option}
      style={{ width: 100, height: 30 }}
      opts={{ renderer: 'canvas' }}
      notMerge
    />
  );
}

// ─── Fallback: generate plausible sparkline from quote data ──────────────────
function generateMockSparkline(quote: Quote): number[] {
  const { price, open, high, low, prev_close } = quote;
  const start = prev_close || open;
  const points: number[] = [start];
  // Walk 19 intermediate points between start and current price
  for (let i = 1; i < 19; i++) {
    const t = i / 19;
    const base = start + (price - start) * t;
    const noise = (high - low) * 0.3 * (Math.sin(i * 1.7) * 0.5 + Math.cos(i * 0.9) * 0.5);
    points.push(Math.max(low, Math.min(high, base + noise)));
  }
  points.push(price);
  return points;
}

// ─── Props shared by sortable sub-components ──────────────────────────────
interface SortableRowProps {
  item: WatchlistItem;
  quote: Quote | undefined;
  sparkData: number[];
  pct: number;
  overId: string | null;
  onRemove: (code: string) => void;
}

// ─── Sortable Desktop Table Row ───────────────────────────────────────────
function SortableDesktopRow({ item, quote, sparkData, pct, overId, onRemove }: SortableRowProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: item.id });

  const isOver = overId === item.id;

  const rowStyle: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
    ...(isOver ? { boxShadow: 'inset 0 2px 0 0 #3b82f6' } : {}),
  };

  return (
    <tr
      ref={setNodeRef}
      style={rowStyle}
      className="hover:bg-[var(--color-bg-hover)] transition-colors"
    >
      {/* Drag handle */}
      <td className="pl-2 pr-0 py-3 w-8">
        <button
          {...attributes}
          {...listeners}
          className="p-1 cursor-grab active:cursor-grabbing rounded hover:bg-white/5 transition-colors"
          style={{ color: 'var(--color-text-muted)', touchAction: 'none' }}
        >
          <GripVertical className="w-4 h-4" />
        </button>
      </td>
      <td className="px-4 py-3 font-mono">
        <Link to={`/kline?code=${item.code}`} className="hover:underline" style={{ color: 'var(--color-accent)' }}>
          {item.code}
        </Link>
      </td>
      <td className="px-4 py-3" style={{ color: 'var(--color-text-secondary)' }}>
        {quote?.name || item.name || '—'}
      </td>
      <td className="px-4 py-3 text-right font-mono" style={{ color: changeColor(pct) }}>
        {quote ? formatPrice(quote.price) : '—'}
      </td>
      <td className="px-4 py-3 text-right font-mono" style={{ color: changeColor(pct) }}>
        {quote ? formatPercent(pct) : '—'}
      </td>
      <td className="px-4 py-3 text-right font-mono" style={{ color: changeColor(pct) }}>
        {quote ? formatChange(quote.change) : '—'}
      </td>
      <td className="px-2 py-1.5">
        <div className="flex items-center justify-center">
          {sparkData.length > 1 ? (
            <Sparkline data={sparkData} color={sparklineColor(pct)} />
          ) : (
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>—</span>
          )}
        </div>
      </td>
      <td className="px-4 py-3 text-right">
        <button
          onClick={() => onRemove(item.code)}
          className="p-1.5 rounded hover:bg-[var(--color-bg-card)] transition-colors"
          title="删除"
        >
          <Trash2 className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </td>
    </tr>
  );
}

// ─── Sortable Mobile Card ────────────────────────────────────────────────
function SortableMobileCard({ item, quote, sparkData, pct, overId, onRemove }: SortableRowProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: item.id });

  const isOver = overId === item.id;

  const cardStyle: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
    background: 'var(--color-bg-secondary)',
    borderColor: isOver ? '#3b82f6' : 'var(--color-border)',
    borderWidth: isOver ? '2px' : '1px',
    ...(isOver ? { boxShadow: '0 0 8px rgba(59,130,246,0.3)' } : {}),
  };

  return (
    <div
      ref={setNodeRef}
      style={cardStyle}
      className="rounded-lg border p-3 flex items-center justify-between"
    >
      {/* Drag handle */}
      <button
        {...attributes}
        {...listeners}
        className="p-1 mr-2 cursor-grab active:cursor-grabbing rounded hover:bg-white/5 transition-colors flex-shrink-0"
        style={{ color: 'var(--color-text-muted)', touchAction: 'none' }}
      >
        <GripVertical className="w-4 h-4" />
      </button>
      <div className="flex-1 min-w-0">
        <Link to={`/kline?code=${item.code}`} className="font-mono text-sm" style={{ color: 'var(--color-accent)' }}>
          {item.code}
        </Link>
        <div className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>
          {quote?.name || item.name || '—'}
        </div>
      </div>
      {/* Mobile sparkline */}
      {sparkData.length > 1 && (
        <div className="mx-2 flex-shrink-0">
          <Sparkline data={sparkData} color={sparklineColor(pct)} />
        </div>
      )}
      <div className="text-right mr-3">
        <div className="font-mono text-sm font-bold" style={{ color: changeColor(pct) }}>
          {quote ? formatPrice(quote.price) : '—'}
        </div>
        <div className="font-mono text-xs" style={{ color: changeColor(pct) }}>
          {quote ? formatPercent(pct) : '—'}
        </div>
      </div>
      <button
        onClick={() => onRemove(item.code)}
        className="p-1.5 rounded hover:bg-[var(--color-bg-card)] transition-colors flex-shrink-0"
        title="删除"
      >
        <Trash2 className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
      </button>
    </div>
  );
}

// ─── Main Page ───────────────────────────────────────────────────────────
export default function WatchlistPage() {
  const [items, setItems] = useState<WatchlistItem[]>([]);
  const [quotes, setQuotes] = useState<Map<string, Quote>>(new Map());
  const [klineData, setKlineData] = useState<Map<string, number[]>>(new Map());
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [sortField, setSortField] = useState<'code' | 'name' | 'price' | 'change_percent' | 'change'>('code');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const intervalRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  // ─── DnD state ─────────────────────────────────────────────────────────
  const [activeId, setActiveId] = useState<string | null>(null);
  const [overId, setOverId] = useState<string | null>(null);
  const [useManualOrder, setUseManualOrder] = useState(false);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 5 },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

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

  // Fetch kline sparkline data for all watchlist items
  const fetchKlineData = useCallback(async (codes: string[]) => {
    const newKline = new Map<string, number[]>();
    await Promise.allSettled(
      codes.map(async (code) => {
        try {
          const res = await marketApi.kline(code, 20);
          if (res.data && res.data.length > 0) {
            newKline.set(code, res.data.map((k) => k.close));
          }
        } catch {
          // kline fetch failed — will fall back to mock
        }
      })
    );
    setKlineData(newKline);
  }, []);

  useEffect(() => {
    fetchWatchlist();
  }, [fetchWatchlist]);

  useEffect(() => {
    if (items.length > 0) {
      fetchQuotes(items.map((i) => i.code));
      fetchKlineData(items.map((i) => i.code));
    }
  }, [items, fetchQuotes, fetchKlineData]);

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

  // Sort toggle handler — also clears manual drag order
  const toggleSort = (field: typeof sortField) => {
    setUseManualOrder(false);
    if (sortField === field) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortField(field);
      // Default: desc for numeric fields, asc for text fields
      setSortDir(field === 'code' || field === 'name' ? 'asc' : 'desc');
    }
  };

  // Compute sorted items
  const sortedItems = useMemo(() => {
    return [...items].sort((a, b) => {
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
  }, [items, quotes, sortField, sortDir]);

  // Display items: manual order (after drag) or column-sorted
  const displayItems = useManualOrder ? items : sortedItems;

  // ─── DnD handlers ──────────────────────────────────────────────────────
  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveId(String(event.active.id));
  }, []);

  const handleDragOver = useCallback((event: DragOverEvent) => {
    setOverId(event.over ? String(event.over.id) : null);
  }, []);

  const handleDragEnd = useCallback((event: DragEndEvent) => {
    const { active, over } = event;
    setActiveId(null);
    setOverId(null);

    if (over && active.id !== over.id) {
      setItems((prev) => {
        const oldIndex = prev.findIndex((i) => i.id === active.id);
        const newIndex = prev.findIndex((i) => i.id === over.id);
        if (oldIndex === -1 || newIndex === -1) return prev;
        return arrayMove(prev, oldIndex, newIndex);
      });
      // Switch to manual order so the drag result is preserved
      setUseManualOrder(true);
    }
  }, []);

  const handleDragCancel = useCallback(() => {
    setActiveId(null);
    setOverId(null);
  }, []);

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

  // Helper: get sparkline data for a stock
  const getSparklineData = (code: string, quote: Quote | undefined): number[] => {
    const kline = klineData.get(code);
    if (kline && kline.length > 1) return kline;
    if (quote) return generateMockSparkline(quote);
    return [];
  };

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
      onDragCancel={handleDragCancel}
    >
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

        {/* Table / Cards */}
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
              <table className="w-full text-sm border-separate border-spacing-0">
                <thead>
                  <tr style={{ background: 'var(--color-bg-secondary)' }}>
                    <th className="w-8 px-2 py-3" /> {/* drag handle column */}
                    <Th field="code" label="代码" />
                    <Th field="name" label="名称" />
                    <Th field="price" label="最新价" align="right" />
                    <Th field="change_percent" label="涨跌幅" align="right" />
                    <Th field="change" label="涨跌额" align="right" />
                    <th className="px-4 py-3 font-medium text-center"
                      style={{ color: 'var(--color-text-secondary)' }}>走势</th>
                    <th className="text-right px-4 py-3 font-medium"
                      style={{ color: 'var(--color-text-secondary)' }}>操作</th>
                  </tr>
                </thead>
                <SortableContext items={displayItems.map((i) => i.id)} strategy={verticalListSortingStrategy}>
                  <tbody>
                    {displayItems.map((item) => {
                      const q = quotes.get(item.code);
                      const pct = q?.change_percent ?? 0;
                      const sparkData = getSparklineData(item.code, q);
                      return (
                        <SortableDesktopRow
                          key={item.id}
                          item={item}
                          quote={q}
                          sparkData={sparkData}
                          pct={pct}
                          overId={overId}
                          onRemove={handleRemove}
                        />
                      );
                    })}
                  </tbody>
                </SortableContext>
              </table>
            </div>

            {/* Mobile card list */}
            <div className="sm:hidden space-y-2">
              <SortableContext items={displayItems.map((i) => i.id)} strategy={verticalListSortingStrategy}>
                {displayItems.map((item) => {
                  const q = quotes.get(item.code);
                  const pct = q?.change_percent ?? 0;
                  const sparkData = getSparklineData(item.code, q);
                  return (
                    <SortableMobileCard
                      key={item.id}
                      item={item}
                      quote={q}
                      sparkData={sparkData}
                      pct={pct}
                      overId={overId}
                      onRemove={handleRemove}
                    />
                  );
                })}
              </SortableContext>
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
    </DndContext>
  );
}
