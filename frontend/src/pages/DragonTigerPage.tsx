import { useState, useEffect, useCallback } from 'react';
import { dragonTigerApi, type DragonTigerItem } from '@/lib/api';
import { Crown, RefreshCw, TrendingUp, TrendingDown, Building2, Calendar, Trophy } from 'lucide-react';
import EmptyState from '@/components/EmptyState';
import ErrorState from '@/components/ErrorState';
import { SkeletonInlineTable } from '@/components/ui/Skeleton';

const DAYS_OPTIONS = [1, 3, 5, 10] as const;

export default function DragonTigerPage() {
  const [items, setItems] = useState<DragonTigerItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [days, setDays] = useState<number>(1);
  const [institutions, setInstitutions] = useState<unknown[]>([]);
  const [instLoading, setInstLoading] = useState(true);

  const fetchData = useCallback(async (d: number) => {
    setLoading(true);
    setError('');
    try {
      const res = await dragonTigerApi.list(d);
      setItems(res.data);
    } catch {
      setError('加载龙虎榜数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchInstitutions = useCallback(async () => {
    setInstLoading(true);
    try {
      const res = await dragonTigerApi.institutions(days);
      const data = res.data;
      setInstitutions(Array.isArray(data) ? data : []);
    } catch {
      // ignore — section will show empty
    } finally {
      setInstLoading(false);
    }
  }, [days]);

  useEffect(() => {
    fetchData(days);
  }, [days, fetchData]);

  useEffect(() => {
    fetchInstitutions();
  }, [fetchInstitutions]);

  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

  const formatAmount = (v: number) => {
    if (Math.abs(v) >= 1e8) return (v / 1e8).toFixed(2) + '亿';
    if (Math.abs(v) >= 1e4) return (v / 1e4).toFixed(2) + '万';
    return v.toFixed(2);
  };

  // Summary
  const totalNet = items.reduce((s, i) => s + i.net_amount, 0);
  const buyCount = items.filter((i) => i.net_amount > 0).length;
  const sellCount = items.filter((i) => i.net_amount < 0).length;

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Crown className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <div>
            <h1 className="text-xl font-bold">龙虎榜</h1>
            <p className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>
              机构与游资买卖数据
            </p>
          </div>
        </div>
        <button
          onClick={() => fetchData(days)}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors"
          style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {/* Day filter */}
      <div className="flex items-center gap-2 mb-4">
        <Calendar className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
        <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>时间范围:</span>
        <div className="flex rounded-lg overflow-hidden border" style={{ borderColor: 'var(--color-border)' }}>
          {DAYS_OPTIONS.map((d) => (
            <button
              key={d}
              onClick={() => setDays(d)}
              className="px-3 py-1.5 text-xs font-medium transition-colors"
              style={{
                background: days === d ? 'var(--color-accent)' : 'var(--color-bg-card)',
                color: days === d ? '#fff' : 'var(--color-text-muted)',
              }}
            >
              {d}天
            </button>
          ))}
        </div>
      </div>

      {/* Summary bar */}
      {items.length > 0 && (
        <div
          className="flex items-center gap-4 mb-4 px-4 py-2.5 rounded-lg text-xs"
          style={{ background: 'var(--color-bg-secondary)' }}
        >
          <span style={{ color: 'var(--color-text-muted)' }}>共 {items.length} 条</span>
          <span style={{ color: 'var(--color-danger)' }}>
            <TrendingUp className="w-3 h-3 inline mr-1" />
            净买入 {buyCount}
          </span>
          <span style={{ color: 'var(--color-success)' }}>
            <TrendingDown className="w-3 h-3 inline mr-1" />
            净卖出 {sellCount}
          </span>
          <span style={{ color: changeColor(totalNet) }}>
            合计净额: {formatAmount(totalNet)}
          </span>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="mb-4">
          <ErrorState
            title="加载失败"
            description={error}
            onRetry={() => { setError(''); fetchData(days); }}
          />
        </div>
      )}

      {/* Loading */}
      {loading && items.length === 0 ? (
        <SkeletonInlineTable rows={8} columns={9} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Trophy}
          title="暂无龙虎榜数据"
          description="非交易日或数据尚未更新"
        />
      ) : (
        <div
          className="rounded-lg border overflow-x-auto mb-6"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <table className="w-full text-sm">
            <thead>
              <tr
                className="border-b"
                style={{ borderColor: 'var(--color-border)', background: 'var(--color-bg-hover)' }}
              >
                <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>代码</th>
                <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>名称</th>
                <th className="px-3 py-2.5 text-right text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>收盘价</th>
                <th className="px-3 py-2.5 text-right text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>涨跌幅</th>
                <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>上榜原因</th>
                <th className="px-3 py-2.5 text-right text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>买入额</th>
                <th className="px-3 py-2.5 text-right text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>卖出额</th>
                <th className="px-3 py-2.5 text-right text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>净额</th>
                <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>日期</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item, idx) => (
                <tr
                  key={`${item.code}-${item.date}-${idx}`}
                  className="border-b transition-colors hover:opacity-90"
                  style={{ borderColor: 'var(--color-border)' }}
                >
                  <td className="px-3 py-2.5 font-mono font-medium" style={{ color: 'var(--color-accent)' }}>
                    {item.code}
                  </td>
                  <td className="px-3 py-2.5 font-medium" style={{ color: 'var(--color-text-primary)' }}>
                    {item.name}
                  </td>
                  <td className="px-3 py-2.5 font-mono text-right" style={{ color: 'var(--color-text-primary)' }}>
                    {item.close_price.toFixed(2)}
                  </td>
                  <td className="px-3 py-2.5 font-mono text-right" style={{ color: changeColor(item.change_pct) }}>
                    {item.change_pct >= 0 ? '+' : ''}{item.change_pct.toFixed(2)}%
                  </td>
                  <td className="px-3 py-2.5 text-xs max-w-[200px] truncate" style={{ color: 'var(--color-text-secondary)' }} title={item.reason}>
                    {item.reason || '-'}
                  </td>
                  <td className="px-3 py-2.5 font-mono text-right text-xs" style={{ color: 'var(--color-danger)' }}>
                    {formatAmount(item.buy_amount)}
                  </td>
                  <td className="px-3 py-2.5 font-mono text-right text-xs" style={{ color: 'var(--color-success)' }}>
                    {formatAmount(item.sell_amount)}
                  </td>
                  <td className="px-3 py-2.5 font-mono text-right font-medium" style={{ color: changeColor(item.net_amount) }}>
                    {formatAmount(item.net_amount)}
                  </td>
                  <td className="px-3 py-2.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                    {item.date}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Institution tracker section */}
      <div className="mt-6">
        <div className="flex items-center gap-2 mb-3">
          <Building2 className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <h2 className="text-sm font-bold">机构动向</h2>
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>({days}天)</span>
        </div>

        {instLoading ? (
          <SkeletonInlineTable rows={4} columns={5} />
        ) : institutions.length === 0 ? (
          <div
            className="text-center py-8 rounded-lg border"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>暂无机构数据</p>
          </div>
        ) : (
          <div
            className="rounded-lg border overflow-x-auto"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <table className="w-full text-sm">
              <thead>
                <tr
                  className="border-b"
                  style={{ borderColor: 'var(--color-border)', background: 'var(--color-bg-hover)' }}
                >
                  {institutions.length > 0 &&
                    Object.keys(institutions[0] as Record<string, unknown>).map((key) => (
                      <th
                        key={key}
                        className="px-3 py-2.5 text-left text-xs font-medium"
                        style={{ color: 'var(--color-text-muted)' }}
                      >
                        {key}
                      </th>
                    ))}
                </tr>
              </thead>
              <tbody>
                {institutions.map((inst, idx) => {
                  const record = inst as Record<string, unknown>;
                  const keys = Object.keys(record);
                  return (
                    <tr
                      key={idx}
                      className="border-b transition-colors hover:opacity-90"
                      style={{ borderColor: 'var(--color-border)' }}
                    >
                      {keys.map((key) => (
                        <td
                          key={key}
                          className="px-3 py-2.5 text-xs"
                          style={{ color: 'var(--color-text-secondary)' }}
                        >
                          {String(record[key] ?? '-')}
                        </td>
                      ))}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
