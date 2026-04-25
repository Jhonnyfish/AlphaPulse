import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import {
  marketApi,
  watchlistApi,
  type MarketOverview,
  type Quote,
} from '@/lib/api';
import {
  TrendingUp,
  TrendingDown,
  Minus,
  Star,
  BarChart3,
  Activity,
  RefreshCw,
} from 'lucide-react';

export default function DashboardPage() {
  const [overview, setOverview] = useState<MarketOverview | null>(null);
  const [watchlistQuotes, setWatchlistQuotes] = useState<
    { code: string; quote: Quote | null }[]
  >([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [overviewRes, watchlistRes] = await Promise.allSettled([
        marketApi.overview(),
        watchlistApi.list(),
      ]);

      if (overviewRes.status === 'fulfilled') {
        setOverview(overviewRes.value.data);
      }

      if (watchlistRes.status === 'fulfilled') {
        const codes = watchlistRes.value.data.slice(0, 6).map((w) => w.code);
        if (codes.length > 0) {
          const quoteResults = await Promise.allSettled(
            codes.map(async (code) => {
              try {
                const res = await marketApi.quote(code);
                return { code, quote: res.data };
              } catch {
                return { code, quote: null };
              }
            })
          );
          setWatchlistQuotes(
            quoteResults
              .map((r) => (r.status === 'fulfilled' ? r.value : null))
              .filter(Boolean) as { code: string; quote: Quote | null }[]
          );
        }
      }
    } catch {
      setError('加载数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const changeColor = (n: number) =>
    n > 0
      ? 'var(--color-danger)'
      : n < 0
        ? 'var(--color-success)'
        : 'var(--color-text-secondary)';

  const TrendIcon = (pct: number) =>
    pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Minus;

  const total = (overview?.advance_count ?? 0) + (overview?.decline_count ?? 0) + (overview?.flat_count ?? 0);
  const advPct = total > 0 ? ((overview?.advance_count ?? 0) / total) * 100 : 0;
  const decPct = total > 0 ? ((overview?.decline_count ?? 0) / total) * 100 : 0;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Activity className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">市场总览</h1>
        </div>
        <button
          onClick={fetchData}
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

      {/* Indices cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 mb-6">
        {(overview?.indices ?? []).length > 0 ? (
          overview!.indices.map((idx) => {
            const pct = idx.change_percent;
            const Icon = TrendIcon(pct);
            return (
              <div
                key={idx.code}
                className="rounded-xl border p-4 transition-colors hover:bg-[var(--color-bg-hover)]"
                style={{
                  background: 'var(--color-bg-secondary)',
                  borderColor: 'var(--color-border)',
                }}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="font-medium text-sm">{idx.name}</span>
                  <span
                    className="text-xs font-mono"
                    style={{ color: 'var(--color-text-muted)' }}
                  >
                    {idx.code}
                  </span>
                </div>
                <div className="flex items-end justify-between">
                  <span
                    className="text-2xl font-bold font-mono"
                    style={{ color: changeColor(pct) }}
                  >
                    {idx.price.toFixed(2)}
                  </span>
                  <span
                    className="flex items-center gap-1 font-mono text-sm"
                    style={{ color: changeColor(pct) }}
                  >
                    <Icon className="w-4 h-4" />
                    {pct >= 0 ? '+' : ''}
                    {pct.toFixed(2)}%
                  </span>
                </div>
                <div className="flex items-center gap-3 mt-2 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                  <span className="text-[var(--color-danger)]">涨 {idx.advance_count}</span>
                  <span className="text-[var(--color-success)]">跌 {idx.decline_count}</span>
                  <span>平 {idx.flat_count}</span>
                </div>
              </div>
            );
          })
        ) : (
          <div
            className="col-span-full rounded-xl border p-6 text-center"
            style={{
              background: 'var(--color-bg-secondary)',
              borderColor: 'var(--color-border)',
            }}
          >
            <p style={{ color: 'var(--color-text-muted)' }}>
              {loading ? '加载指数数据...' : '暂无指数数据（非交易时间）'}
            </p>
          </div>
        )}
      </div>

      {/* Market breadth bar */}
      {total > 0 && (
        <div
          className="rounded-xl border p-4 mb-6"
          style={{
            background: 'var(--color-bg-secondary)',
            borderColor: 'var(--color-border)',
          }}
        >
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium">市场宽度</span>
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
              全市场涨跌统计
            </span>
          </div>
          <div className="flex h-3 rounded-full overflow-hidden" style={{ background: 'var(--color-bg-card)' }}>
            <div
              className="transition-all"
              style={{
                width: `${advPct}%`,
                background: 'var(--color-danger)',
              }}
            />
            <div
              className="transition-all"
              style={{
                width: `${decPct}%`,
                background: 'var(--color-success)',
              }}
            />
            <div
              className="flex-1"
              style={{ background: 'var(--color-text-muted)', opacity: 0.3 }}
            />
          </div>
          <div className="flex items-center justify-between mt-2 text-xs">
            <span style={{ color: 'var(--color-danger)' }}>
              上涨 {overview?.advance_count ?? 0} ({advPct.toFixed(1)}%)
            </span>
            <span style={{ color: 'var(--color-success)' }}>
              下跌 {overview?.decline_count ?? 0} ({decPct.toFixed(1)}%)
            </span>
            <span style={{ color: 'var(--color-text-muted)' }}>
              平盘 {overview?.flat_count ?? 0}
            </span>
          </div>
        </div>
      )}

      {/* Watchlist quick view */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Star className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
            <span className="text-sm font-medium">自选股速览</span>
          </div>
          <Link
            to="/watchlist"
            className="text-xs px-2 py-1 rounded hover:bg-[var(--color-bg-hover)] transition-colors"
            style={{ color: 'var(--color-accent)' }}
          >
            查看全部 →
          </Link>
        </div>

        {watchlistQuotes.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {watchlistQuotes.map(({ code, quote }) => {
              const pct = quote?.change_percent ?? 0;
              const Icon = TrendIcon(pct);
              return (
                <Link
                  key={code}
                  to={`/kline?code=${code}`}
                  className="rounded-xl border p-4 transition-colors hover:bg-[var(--color-bg-hover)]"
                  style={{
                    background: 'var(--color-bg-secondary)',
                    borderColor: 'var(--color-border)',
                  }}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className="font-medium text-sm">
                      {quote?.name || code}
                    </span>
                    <span
                      className="text-xs font-mono"
                      style={{ color: 'var(--color-text-muted)' }}
                    >
                      {code}
                    </span>
                  </div>
                  <div className="flex items-end justify-between">
                    <span
                      className="text-lg font-bold font-mono"
                      style={{ color: changeColor(pct) }}
                    >
                      {quote ? quote.price.toFixed(2) : '—'}
                    </span>
                    <span
                      className="flex items-center gap-1 font-mono text-sm"
                      style={{ color: changeColor(pct) }}
                    >
                      <Icon className="w-3.5 h-3.5" />
                      {pct >= 0 ? '+' : ''}
                      {pct.toFixed(2)}%
                    </span>
                  </div>
                </Link>
              );
            })}
          </div>
        ) : (
          <div
            className="rounded-xl border p-6 text-center"
            style={{
              background: 'var(--color-bg-secondary)',
              borderColor: 'var(--color-border)',
            }}
          >
            <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
              {loading
                ? '加载自选股...'
                : '暂无自选股，去自选股页面添加'}
            </p>
          </div>
        )}
      </div>

      {/* Quick links */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {[
          { to: '/market', label: '行情查询', icon: TrendingUp, color: 'var(--color-accent)' },
          { to: '/kline', label: 'K线图', icon: BarChart3, color: 'var(--color-warning)' },
          { to: '/sectors', label: '板块行情', icon: BarChart3, color: 'var(--color-success)' },
          { to: '/news', label: '市场资讯', icon: Activity, color: 'var(--color-danger)' },
        ].map(({ to, label, icon: Icon, color }) => (
          <Link
            key={to}
            to={to}
            className="rounded-xl border p-4 flex items-center gap-3 transition-colors hover:bg-[var(--color-bg-hover)]"
            style={{
              background: 'var(--color-bg-secondary)',
              borderColor: 'var(--color-border)',
            }}
          >
            <Icon className="w-5 h-5" style={{ color }} />
            <span className="text-sm font-medium">{label}</span>
          </Link>
        ))}
      </div>

      {/* Updated time */}
      {overview?.updated_at && (
        <div className="mt-4 text-xs text-right" style={{ color: 'var(--color-text-muted)' }}>
          数据更新: {new Date(overview.updated_at).toLocaleString('zh-CN')}
        </div>
      )}
    </div>
  );
}
