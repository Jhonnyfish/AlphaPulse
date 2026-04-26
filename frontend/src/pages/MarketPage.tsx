import { useState, useCallback, useEffect, useMemo } from 'react';
import {
  marketApi,
  type Quote,
  type SearchSuggestion,
  type TopMover,
} from '@/lib/api';
import {
  TrendingUp,
  TrendingDown,
  Minus,
  BarChart3,
  RefreshCw,
} from 'lucide-react';
import EmptyState from '@/components/EmptyState';
import StockSearch from '@/components/StockSearch';
import { SortableHeader, TableToolbar, Pagination } from '@/components/table';
import { useTableSort } from '@/hooks/useTableSort';
import { SkeletonInlineTable } from '@/components/ui/Skeleton';

/* ── Change percent filter options ────────────────────────────── */
const CHANGE_FILTERS = [
  { key: 'all', label: '全部' },
  { key: 'up', label: '涨' },
  { key: 'down', label: '跌' },
  { key: 'limit_up', label: '涨停' },
  { key: 'limit_down', label: '跌停' },
] as const;

type ChangeFilterKey = (typeof CHANGE_FILTERS)[number]['key'];

/** Apply the change_percent filter to a TopMover array */
function applyChangeFilter(data: TopMover[], filter: ChangeFilterKey): TopMover[] {
  switch (filter) {
    case 'up':
      return data.filter((s) => s.change_percent > 0);
    case 'down':
      return data.filter((s) => s.change_percent < 0);
    case 'limit_up':
      return data.filter((s) => s.change_percent >= 9.9);
    case 'limit_down':
      return data.filter((s) => s.change_percent <= -9.9);
    default:
      return data;
  }
}

/** Format number with Chinese units (万/亿) */
function formatAmount(val: number): string {
  if (val >= 1e8) return `${(val / 1e8).toFixed(2)}亿`;
  if (val >= 1e4) return `${(val / 1e4).toFixed(2)}万`;
  return val.toFixed(2);
}

function formatVolume(val: number): string {
  if (val >= 1e4) return `${(val / 1e4).toFixed(0)}万`;
  return val.toLocaleString('zh-CN');
}

/** Color helper: 红涨绿跌 */
function changeColor(pct: number): string {
  if (pct > 0) return 'var(--color-danger)';
  if (pct < 0) return 'var(--color-success)';
  return 'var(--color-text-secondary)';
}

/* ═══════════════════════════════════════════════════════════════
   MarketPage
   ═══════════════════════════════════════════════════════════════ */
export default function MarketPage() {
  /* ── Single stock quote section ──────────────────────────── */
  const [quote, setQuote] = useState<Quote | null>(null);
  const [quoteLoading, setQuoteLoading] = useState(false);
  const [quoteError, setQuoteError] = useState('');

  const handleSelect = useCallback(async (suggestion: SearchSuggestion) => {
    setQuoteLoading(true);
    setQuoteError('');
    setQuote(null);
    try {
      const res = await marketApi.quote(suggestion.code);
      setQuote(res.data);
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message || '查询失败，请检查股票代码';
      setQuoteError(msg);
    } finally {
      setQuoteLoading(false);
    }
  }, []);

  const pct = quote?.change_percent ?? 0;
  const quoteColor =
    pct > 0
      ? 'var(--color-danger)'
      : pct < 0
        ? 'var(--color-success)'
        : 'var(--color-text-secondary)';
  const TrendIcon = pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Minus;

  /* ── Top movers table section ────────────────────────────── */
  const [movers, setMovers] = useState<TopMover[]>([]);
  const [moversLoading, setMoversLoading] = useState(false);
  const [moversError, setMoversError] = useState('');
  const [changeFilter, setChangeFilter] = useState<ChangeFilterKey>('all');

  const fetchMovers = useCallback(async () => {
    setMoversLoading(true);
    setMoversError('');
    try {
      const res = await marketApi.topMovers('desc', 200);
      setMovers(res.data);
    } catch {
      setMoversError('加载排行榜数据失败');
    } finally {
      setMoversLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchMovers();
  }, [fetchMovers]);

  /** Pre-filtered data (before table hook) */
  const filteredMovers = useMemo(
    () => applyChangeFilter(movers, changeFilter),
    [movers, changeFilter]
  );

  /** Table sort / search / pagination hook */
  const table = useTableSort<TopMover>(filteredMovers, {
    defaultSortKey: 'change_percent',
    defaultSortDir: 'desc',
    pageSize: 20,
    searchFields: ['code', 'name'],
  });

  /** Wrap filter change to sync with local state */
  const handleFilterChange = useCallback(
    (f: string) => {
      setChangeFilter(f as ChangeFilterKey);
      table.updateFilter(f);
    },
    [table]
  );

  return (
    <div className="space-y-6">
      {/* ─── Page header ──────────────────────────────────── */}
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold flex items-center gap-2">
          <BarChart3 className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          行情中心
        </h1>
      </div>

      {/* ─── Search + Quote card ──────────────────────────── */}
      <div
        className="rounded-xl p-5"
        style={{
          background: 'rgba(15, 23, 42, 0.6)',
          backdropFilter: 'blur(18px)',
          border: '1px solid rgba(148, 163, 184, 0.1)',
        }}
      >
        <div className="mb-4 max-w-sm">
          <StockSearch
            onSelect={handleSelect}
            placeholder="搜索股票代码或名称..."
          />
        </div>

        {quoteLoading && (
          <div
            className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md flex items-center gap-2"
            style={{ background: 'rgba(59,130,246,0.1)', color: 'var(--color-accent)' }}
          >
            <Loader2 className="w-4 h-4 animate-spin" />
            查询中...
          </div>
        )}

        {quoteError && (
          <div
            className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
            style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
          >
            {quoteError}
          </div>
        )}

        {quote && (
          <div
            className="rounded-xl border p-5 max-w-lg"
            style={{
              background: 'var(--color-bg-secondary)',
              borderColor: 'var(--color-border)',
            }}
          >
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-lg font-bold">{quote.name}</h2>
                <span
                  className="text-sm font-mono"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  {quote.code}
                </span>
              </div>
              <div className="flex items-center gap-1.5" style={{ color: quoteColor }}>
                <TrendIcon className="w-5 h-5" />
                <span className="text-lg font-bold font-mono">
                  {pct >= 0 ? '+' : ''}
                  {pct.toFixed(2)}%
                </span>
              </div>
            </div>

            {/* Price */}
            <div className="mb-6">
              <span
                className="text-4xl font-bold font-mono"
                style={{ color: quoteColor }}
              >
                {quote.price.toFixed(2)}
              </span>
              <span className="ml-3 text-sm font-mono" style={{ color: quoteColor }}>
                {quote.change >= 0 ? '+' : ''}
                {quote.change.toFixed(2)}
              </span>
            </div>

            {/* Details grid */}
            <div className="grid grid-cols-2 gap-3">
              {[
                { label: '开盘', value: quote.open.toFixed(2) },
                { label: '昨收', value: quote.prev_close.toFixed(2) },
                { label: '最高', value: quote.high.toFixed(2) },
                { label: '最低', value: quote.low.toFixed(2) },
                {
                  label: '振幅',
                  value:
                    quote.prev_close > 0
                      ? `${(((quote.high - quote.low) / quote.prev_close) * 100).toFixed(2)}%`
                      : '—',
                  highlight: true,
                },
                {
                  label: '涨跌额',
                  value: `${quote.change >= 0 ? '+' : ''}${quote.change.toFixed(2)}`,
                  highlight: true,
                },
              ].map(({ label, value, highlight }) => (
                <div
                  key={label}
                  className="flex justify-between px-3 py-2 rounded-lg"
                  style={{
                    background: highlight
                      ? 'rgba(59,130,246,0.08)'
                      : 'var(--color-bg-card)',
                  }}
                >
                  <span className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
                    {label}
                  </span>
                  <span
                    className={`text-sm font-mono ${highlight ? 'font-medium' : ''}`}
                  >
                    {value}
                  </span>
                </div>
              ))}
            </div>

            <div className="mt-4 text-xs" style={{ color: 'var(--color-text-muted)' }}>
              更新时间: {quote.updated_at}
            </div>
          </div>
        )}

        {!quote && !quoteError && !quoteLoading && (
          <div
            className="text-center py-10 rounded-lg border max-w-lg"
            style={{
              background: 'var(--color-bg-secondary)',
              borderColor: 'var(--color-border)',
            }}
          >
            <p style={{ color: 'var(--color-text-muted)' }}>搜索股票查看实时行情</p>
            <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
              例如: 平安银行、贵州茅台、000001
            </p>
          </div>
        )}
      </div>

      {/* ─── Top Movers Data Table ────────────────────────── */}
      <div
        className="rounded-xl p-5"
        style={{
          background: 'rgba(15, 23, 42, 0.6)',
          backdropFilter: 'blur(18px)',
          border: '1px solid rgba(148, 163, 184, 0.1)',
        }}
      >
        {/* Section header */}
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-bold flex items-center gap-2">
            <TrendingUp className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
            涨幅排行榜
          </h2>
          <button
            onClick={fetchMovers}
            disabled={moversLoading}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
            style={{
              background: 'rgba(148, 163, 184, 0.06)',
              color: 'var(--color-text-secondary)',
              border: '1px solid rgba(148, 163, 184, 0.1)',
              opacity: moversLoading ? 0.5 : 1,
            }}
          >
            <RefreshCw
              className={`w-3.5 h-3.5 ${moversLoading ? 'animate-spin' : ''}`}
            />
            刷新
          </button>
        </div>

        {moversError && (
          <div
            className="text-sm px-3 py-2 rounded-lg mb-4"
            style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
          >
            {moversError}
          </div>
        )}

        {/* Toolbar: search + filter */}
        <TableToolbar
          searchQuery={table.searchQuery}
          onSearchChange={table.updateSearch}
          searchPlaceholder="搜索代码或名称..."
          filterValue={table.filter}
          onFilterChange={handleFilterChange}
          filterOptions={CHANGE_FILTERS as unknown as { key: string; label: string }[]}
          totalCount={table.totalItems}
        />

        {moversLoading && movers.length === 0 ? (
          <SkeletonInlineTable rows={8} columns={9} />
        ) : !moversLoading && movers.length === 0 ? (
          <EmptyState
            icon={TrendingUp}
            title="暂无市场数据"
            description="市场尚未开盘或数据正在同步中"
          />
        ) : table.paginated.length === 0 ? (
          <div className="text-center py-12 text-sm" style={{ color: 'var(--color-text-muted)' }}>
            无匹配结果
          </div>
        ) : (
        <div className="overflow-x-auto rounded-lg" style={{ border: '1px solid var(--color-border)' }}>
          <table
            className="w-full data-table"
            style={{ borderCollapse: 'separate', borderSpacing: 0 }}
          >
            <thead>
              <tr>
                <SortableHeader
                  label="序号"
                  sortKey=""
                  currentKey=""
                  direction={null}
                  onSort={() => {}}
                  className="w-14"
                />
                <SortableHeader
                  label="代码"
                  sortKey="code"
                  currentKey={table.sort.key}
                  direction={table.sort.direction}
                  onSort={table.toggleSort}
                />
                <SortableHeader
                  label="名称"
                  sortKey="name"
                  currentKey={table.sort.key}
                  direction={table.sort.direction}
                  onSort={table.toggleSort}
                />
                <SortableHeader
                  label="最新价"
                  sortKey="price"
                  currentKey={table.sort.key}
                  direction={table.sort.direction}
                  onSort={table.toggleSort}
                  align="right"
                />
                <SortableHeader
                  label="涨跌额"
                  sortKey="change"
                  currentKey={table.sort.key}
                  direction={table.sort.direction}
                  onSort={table.toggleSort}
                  align="right"
                />
                <SortableHeader
                  label="涨跌幅"
                  sortKey="change_percent"
                  currentKey={table.sort.key}
                  direction={table.sort.direction}
                  onSort={table.toggleSort}
                  align="right"
                />
                <SortableHeader
                  label="成交量"
                  sortKey="volume"
                  currentKey={table.sort.key}
                  direction={table.sort.direction}
                  onSort={table.toggleSort}
                  align="right"
                />
                <SortableHeader
                  label="成交额"
                  sortKey="amount"
                  currentKey={table.sort.key}
                  direction={table.sort.direction}
                  onSort={table.toggleSort}
                  align="right"
                />
                <SortableHeader
                  label="振幅"
                  sortKey="amplitude"
                  currentKey={table.sort.key}
                  direction={table.sort.direction}
                  onSort={table.toggleSort}
                  align="right"
                />
              </tr>
            </thead>
            <tbody>
              {(
                table.paginated.map((stock, idx) => {
                  const changePct = stock.change_percent;
                  const color = changeColor(changePct);
                  const rowNumber = (table.page - 1) * 20 + idx + 1;

                  return (
                    <tr
                      key={stock.code}
                      className="transition-colors"
                      style={{ cursor: 'pointer' }}
                      onClick={() => handleSelect({ code: stock.code, name: stock.name })}
                    >
                      {/* 序号 */}
                      <td
                        className="font-mono text-xs"
                        style={{
                          padding: '10px 14px',
                          borderBottom: '1px solid rgba(148, 163, 184, 0.06)',
                          color: 'var(--color-text-muted)',
                        }}
                      >
                        {rowNumber}
                      </td>
                      {/* 代码 */}
                      <td
                        className="font-mono text-xs font-medium"
                        style={{
                          padding: '10px 14px',
                          borderBottom: '1px solid rgba(148, 163, 184, 0.06)',
                          color: 'var(--color-accent)',
                        }}
                      >
                        {stock.code}
                      </td>
                      {/* 名称 */}
                      <td
                        className="text-sm font-medium"
                        style={{
                          padding: '10px 14px',
                          borderBottom: '1px solid rgba(148, 163, 184, 0.06)',
                          color: 'var(--color-text-primary)',
                        }}
                      >
                        {stock.name}
                      </td>
                      {/* 最新价 */}
                      <td
                        className="font-mono text-sm text-right"
                        style={{
                          padding: '10px 14px',
                          borderBottom: '1px solid rgba(148, 163, 184, 0.06)',
                          color,
                        }}
                      >
                        {stock.price.toFixed(2)}
                      </td>
                      {/* 涨跌额 */}
                      <td
                        className="font-mono text-sm text-right"
                        style={{
                          padding: '10px 14px',
                          borderBottom: '1px solid rgba(148, 163, 184, 0.06)',
                          color,
                        }}
                      >
                        {stock.change >= 0 ? '+' : ''}
                        {stock.change.toFixed(2)}
                      </td>
                      {/* 涨跌幅 */}
                      <td
                        className="font-mono text-sm text-right font-medium"
                        style={{
                          padding: '10px 14px',
                          borderBottom: '1px solid rgba(148, 163, 184, 0.06)',
                          color,
                        }}
                      >
                        <span
                          className="inline-block px-2 py-0.5 rounded-md text-xs font-semibold"
                          style={{
                            background:
                              changePct > 0
                                ? 'rgba(239, 68, 68, 0.12)'
                                : changePct < 0
                                  ? 'rgba(34, 197, 94, 0.12)'
                                  : 'rgba(148, 163, 184, 0.08)',
                            color,
                          }}
                        >
                          {changePct >= 0 ? '+' : ''}
                          {changePct.toFixed(2)}%
                        </span>
                      </td>
                      {/* 成交量 */}
                      <td
                        className="font-mono text-xs text-right"
                        style={{
                          padding: '10px 14px',
                          borderBottom: '1px solid rgba(148, 163, 184, 0.06)',
                          color: 'var(--color-text-secondary)',
                        }}
                      >
                        {stock.volume != null ? formatVolume(stock.volume) : '—'}
                      </td>
                      {/* 成交额 */}
                      <td
                        className="font-mono text-xs text-right"
                        style={{
                          padding: '10px 14px',
                          borderBottom: '1px solid rgba(148, 163, 184, 0.06)',
                          color: 'var(--color-text-secondary)',
                        }}
                      >
                        {stock.amount != null ? formatAmount(stock.amount) : '—'}
                      </td>
                      {/* 振幅 */}
                      <td
                        className="font-mono text-xs text-right"
                        style={{
                          padding: '10px 14px',
                          borderBottom: '1px solid rgba(148, 163, 184, 0.06)',
                          color: 'var(--color-text-muted)',
                        }}
                      >
                        {stock.amplitude != null
                          ? `${stock.amplitude.toFixed(2)}%`
                          : '—'}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
        )}

        {/* Pagination */}
        <Pagination
          page={table.page}
          totalPages={table.totalPages}
          onPageChange={table.goToPage}
          totalItems={table.totalItems}
          pageSize={20}
        />
      </div>
    </div>
  );
}
