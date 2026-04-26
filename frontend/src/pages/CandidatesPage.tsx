import { useState, useEffect, useCallback } from 'react';
import { candidatesApi } from '@/lib/api';
import { Target, RefreshCw, ArrowUpDown, TrendingUp, ExternalLink } from 'lucide-react';
import { Link } from 'react-router-dom';

interface Candidate {
  code: string;
  ts_code: string;
  name: string;
  rank: number;
  score: number;
  close: number;
  atr14: number;
  buy_low: number;
  buy_high: number;
  sell_low: number;
  sell_high: number;
  stop_loss: number;
  momentum: number;
  trend: number;
  volatility: number;
  liquidity: number;
  industry: string;
  limit_up_today: boolean;
  limit_up_prev_day: boolean;
  leader_signal: string;
}

interface CandidatesResponse {
  limit: number;
  items: Candidate[];
  tier_counts: Record<string, number>;
  fetched_at: string;
  strategy_id?: string;
}

type SortField = 'rank' | 'score' | 'momentum' | 'trend' | 'close';

export default function CandidatesPage() {
  const [data, setData] = useState<CandidatesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [sortField, setSortField] = useState<SortField>('rank');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await candidatesApi.list();
      setData(res.data);
    } catch {
      setError('加载候选股失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortField(field);
      setSortDir(field === 'rank' ? 'asc' : 'desc');
    }
  };

  const sorted = data?.items
    ? [...data.items].sort((a, b) => {
        const av = a[sortField];
        const bv = b[sortField];
        const cmp = av < bv ? -1 : av > bv ? 1 : 0;
        return sortDir === 'asc' ? cmp : -cmp;
      })
    : [];

  const scoreColor = (score: number) => {
    if (score >= 80) return 'var(--color-success, #22c55e)';
    if (score >= 60) return 'var(--color-warning, #f59e0b)';
    return 'var(--color-danger, #ef4444)';
  };

  const SortHeader = ({
    field,
    label,
  }: {
    field: SortField;
    label: string;
  }) => (
    <th
      className="px-3 py-2.5 text-left text-xs font-medium cursor-pointer select-none hover:opacity-80 transition-opacity"
      style={{ color: 'var(--color-text-muted)' }}
      onClick={() => handleSort(field)}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        {sortField === field && (
          <ArrowUpDown className="w-3 h-3" style={{ color: 'var(--color-accent)' }} />
        )}
      </span>
    </th>
  );

  return (
    <div className="p-4 lg:p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Target className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">AI 候选股</h1>
          {data && (
            <span
              className="text-xs px-2 py-0.5 rounded-full"
              style={{
                background: 'var(--color-bg-hover)',
                color: 'var(--color-text-muted)',
              }}
            >
              {data.items.length} 只 · 更新于 {new Date(data.fetched_at).toLocaleTimeString()}
            </span>
          )}
        </div>
        <button
          onClick={fetchData}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors"
          style={{
            background: 'var(--color-bg-hover)',
            color: 'var(--color-text-secondary)',
          }}
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {/* Tier summary */}
      {data?.tier_counts && Object.keys(data.tier_counts).length > 0 && (
        <div className="flex gap-3 flex-wrap">
          {Object.entries(data.tier_counts).map(([tier, count]) => (
            <span
              key={tier}
              className="text-xs px-2.5 py-1 rounded-lg"
              style={{
                background: 'var(--color-bg-hover)',
                color: 'var(--color-text-secondary)',
              }}
            >
              {tier}: <strong>{count}</strong>
            </span>
          ))}
        </div>
      )}

      {/* Error */}
      {error && (
        <div
          className="px-4 py-3 rounded-lg text-sm"
          style={{ background: 'rgba(239,68,68,0.1)', color: '#ef4444' }}
        >
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && !data && (
        <div
          className="flex items-center justify-center py-20 text-sm"
          style={{ color: 'var(--color-text-muted)' }}
        >
          <RefreshCw className="w-5 h-5 animate-spin mr-2" />
          加载中...
        </div>
      )}

      {/* Table */}
      {data && sorted.length > 0 && (
        <div
          className="rounded-lg border overflow-x-auto"
          style={{
            background: 'var(--color-bg-secondary)',
            borderColor: 'var(--color-border)',
          }}
        >
          <table className="w-full text-sm">
            <thead>
              <tr
                className="border-b"
                style={{
                  borderColor: 'var(--color-border)',
                  background: 'var(--color-bg-hover)',
                }}
              >
                <SortHeader field="rank" label="排名" />
                <th
                  className="px-3 py-2.5 text-left text-xs font-medium"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  代码
                </th>
                <th
                  className="px-3 py-2.5 text-left text-xs font-medium"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  名称
                </th>
                <SortHeader field="score" label="评分" />
                <SortHeader field="close" label="现价" />
                <SortHeader field="momentum" label="动量" />
                <SortHeader field="trend" label="趋势" />
                <th
                  className="px-3 py-2.5 text-left text-xs font-medium"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  买入区间
                </th>
                <th
                  className="px-3 py-2.5 text-left text-xs font-medium"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  止损
                </th>
                <th
                  className="px-3 py-2.5 text-left text-xs font-medium"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  行业
                </th>
                <th
                  className="px-3 py-2.5 text-left text-xs font-medium"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  信号
                </th>
                <th
                  className="px-3 py-2.5 text-center text-xs font-medium"
                  style={{ color: 'var(--color-text-muted)' }}
                >
                  操作
                </th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((c) => (
                <tr
                  key={c.code}
                  className="border-b transition-colors hover:opacity-90"
                  style={{ borderColor: 'var(--color-border)' }}
                >
                  <td className="px-3 py-2.5 font-mono" style={{ color: 'var(--color-text-muted)' }}>
                    #{c.rank}
                  </td>
                  <td className="px-3 py-2.5 font-mono font-medium" style={{ color: 'var(--color-accent)' }}>
                    {c.code}
                  </td>
                  <td className="px-3 py-2.5 font-medium" style={{ color: 'var(--color-text-primary)' }}>
                    {c.name}
                    {c.limit_up_today && (
                      <span className="ml-1 text-xs" style={{ color: '#ef4444' }}>🔥</span>
                    )}
                  </td>
                  <td className="px-3 py-2.5">
                    <span
                      className="font-bold font-mono"
                      style={{ color: scoreColor(c.score) }}
                    >
                      {c.score.toFixed(1)}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 font-mono" style={{ color: 'var(--color-text-primary)' }}>
                    {c.close.toFixed(2)}
                  </td>
                  <td className="px-3 py-2.5 font-mono">
                    <span style={{ color: c.momentum >= 0 ? '#22c55e' : '#ef4444' }}>
                      {c.momentum >= 0 ? '+' : ''}{c.momentum.toFixed(1)}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 font-mono">
                    <span style={{ color: c.trend >= 0 ? '#22c55e' : '#ef4444' }}>
                      {c.trend >= 0 ? '+' : ''}{c.trend.toFixed(1)}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 font-mono text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                    {c.buy_low.toFixed(2)} ~ {c.buy_high.toFixed(2)}
                  </td>
                  <td className="px-3 py-2.5 font-mono text-xs" style={{ color: '#ef4444' }}>
                    {c.stop_loss.toFixed(2)}
                  </td>
                  <td className="px-3 py-2.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                    {c.industry || '-'}
                  </td>
                  <td className="px-3 py-2.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                    {c.leader_signal || '-'}
                  </td>
                  <td className="px-3 py-2.5 text-center">
                    <Link
                      to={`/kline?code=${c.code}`}
                      className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors"
                      style={{
                        background: 'var(--color-bg-hover)',
                        color: 'var(--color-accent)',
                      }}
                    >
                      <TrendingUp className="w-3 h-3" />
                      K线
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Empty state */}
      {data && sorted.length === 0 && (
        <div
          className="flex flex-col items-center justify-center py-20 text-sm"
          style={{ color: 'var(--color-text-muted)' }}
        >
          <Target className="w-10 h-10 mb-3 opacity-40" />
          暂无候选股数据
        </div>
      )}
    </div>
  );
}
