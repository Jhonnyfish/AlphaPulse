import { useState, useCallback } from 'react';
import { screenerApi } from '@/lib/api';
import { Filter, RefreshCw, ArrowUpDown, TrendingUp, Search, X } from 'lucide-react';
import { Link } from 'react-router-dom';

interface ScreenerResult {
  code: string;
  name: string;
  rank: number;
  score: number;
  close: number;
  momentum: number;
  trend: number;
  volatility: number;
  liquidity: number;
  industry: string;
  recommendation_tier: string;
  leader_signal: string;
  harvest_risk_level: string;
  focus_reason: string;
  in_watchlist: boolean;
}

interface ScreenerResponse {
  ok: boolean;
  count: number;
  total_candidates: number;
  filtered: number;
  results: ScreenerResult[];
  filters: Record<string, any>;
}

type SortField = 'rank' | 'score' | 'momentum' | 'trend' | 'close' | 'volatility';

export default function ScreenerPage() {
  const [data, setData] = useState<ScreenerResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [sortField, setSortField] = useState<SortField>('score');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');

  // Filter inputs
  const [minScore, setMinScore] = useState('60');
  const [minMomentum, setMinMomentum] = useState('');
  const [minTrend, setMinTrend] = useState('');
  const [industry, setIndustry] = useState('');

  const doScreen = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const params: Record<string, string | number> = {};
      if (minScore) params.min_score = Number(minScore);
      if (minMomentum) params.min_momentum = Number(minMomentum);
      if (minTrend) params.min_trend = Number(minTrend);
      if (industry) params.industry = industry;
      const res = await screenerApi.screen(params);
      setData(res.data);
    } catch {
      setError('选股筛选失败');
    } finally {
      setLoading(false);
    }
  }, [minScore, minMomentum, minTrend, industry]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortField(field);
      setSortDir(field === 'rank' ? 'asc' : 'desc');
    }
  };

  const sorted = data?.results
    ? [...data.results].sort((a, b) => {
        const av = a[sortField];
        const bv = b[sortField];
        const cmp = av < bv ? -1 : av > bv ? 1 : 0;
        return sortDir === 'asc' ? cmp : -cmp;
      })
    : [];

  const tierColor = (tier: string) => {
    switch (tier?.toLowerCase()) {
      case 'strong_buy':
      case 'strong buy':
        return { bg: 'rgba(34,197,94,0.15)', text: '#22c55e' };
      case 'buy':
        return { bg: 'rgba(34,197,94,0.1)', text: '#4ade80' };
      case 'hold':
        return { bg: 'rgba(245,158,11,0.1)', text: '#f59e0b' };
      case 'sell':
      case 'strong_sell':
      case 'strong sell':
        return { bg: 'rgba(239,68,68,0.1)', text: '#ef4444' };
      default:
        return { bg: 'var(--color-bg-hover)', text: 'var(--color-text-muted)' };
    }
  };

  const scoreColor = (score: number) => {
    if (score >= 80) return '#22c55e';
    if (score >= 60) return '#f59e0b';
    return '#ef4444';
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

  const InputField = ({
    label,
    value,
    onChange,
    placeholder,
    type = 'text',
  }: {
    label: string;
    value: string;
    onChange: (v: string) => void;
    placeholder: string;
    type?: string;
  }) => (
    <div className="flex flex-col gap-1">
      <label className="text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
        {label}
      </label>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="px-3 py-1.5 rounded-lg text-sm border outline-none transition-colors"
        style={{
          background: 'var(--color-bg-primary)',
          borderColor: 'var(--color-border)',
          color: 'var(--color-text-primary)',
        }}
      />
    </div>
  );

  return (
    <div className="p-4 lg:p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Filter className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">选股器</h1>
          {data && (
            <span
              className="text-xs px-2 py-0.5 rounded-full"
              style={{
                background: 'var(--color-bg-hover)',
                color: 'var(--color-text-muted)',
              }}
            >
              {data.count} / {data.total_candidates} 只
            </span>
          )}
        </div>
      </div>

      {/* Filter bar */}
      <div
        className="rounded-lg border p-4 space-y-3"
        style={{
          background: 'var(--color-bg-secondary)',
          borderColor: 'var(--color-border)',
        }}
      >
        <div className="grid grid-cols-2 lg:grid-cols-5 gap-3">
          <InputField
            label="最低评分"
            value={minScore}
            onChange={setMinScore}
            placeholder="60"
            type="number"
          />
          <InputField
            label="最低动量"
            value={minMomentum}
            onChange={setMinMomentum}
            placeholder="不限"
            type="number"
          />
          <InputField
            label="最低趋势"
            value={minTrend}
            onChange={setMinTrend}
            placeholder="不限"
            type="number"
          />
          <InputField
            label="行业"
            value={industry}
            onChange={setIndustry}
            placeholder="如：半导体"
          />
          <div className="flex items-end gap-2">
            <button
              onClick={doScreen}
              disabled={loading}
              className="flex items-center gap-1.5 px-4 py-1.5 rounded-lg text-sm font-medium transition-colors"
              style={{
                background: 'var(--color-accent)',
                color: '#fff',
              }}
            >
              <Search className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              筛选
            </button>
            {data && (
              <button
                onClick={() => {
                  setData(null);
                  setMinScore('60');
                  setMinMomentum('');
                  setMinTrend('');
                  setIndustry('');
                }}
                className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm transition-colors"
                style={{
                  background: 'var(--color-bg-hover)',
                  color: 'var(--color-text-muted)',
                }}
              >
                <X className="w-4 h-4" />
                重置
              </button>
            )}
          </div>
        </div>
      </div>

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
      {loading && (
        <div className="py-4">
          <SkeletonInlineTable rows={8} columns={6} />
        </div>
      )}

      {/* Results table */}
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
                <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                  代码
                </th>
                <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                  名称
                </th>
                <SortHeader field="score" label="评分" />
                <SortHeader field="close" label="现价" />
                <SortHeader field="momentum" label="动量" />
                <SortHeader field="trend" label="趋势" />
                <SortHeader field="volatility" label="波动" />
                <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                  行业
                </th>
                <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                  评级
                </th>
                <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                  关注原因
                </th>
                <th className="px-3 py-2.5 text-center text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                  操作
                </th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((s) => {
                const tc = tierColor(s.recommendation_tier);
                return (
                  <tr
                    key={s.code}
                    className="border-b transition-colors hover:opacity-90"
                    style={{ borderColor: 'var(--color-border)' }}
                  >
                    <td className="px-3 py-2.5 font-mono" style={{ color: 'var(--color-text-muted)' }}>
                      #{s.rank}
                    </td>
                    <td className="px-3 py-2.5 font-mono font-medium" style={{ color: 'var(--color-accent)' }}>
                      {s.code}
                      {s.in_watchlist && (
                        <span className="ml-1 text-xs" title="已在自选股">⭐</span>
                      )}
                    </td>
                    <td className="px-3 py-2.5 font-medium" style={{ color: 'var(--color-text-primary)' }}>
                      {s.name}
                    </td>
                    <td className="px-3 py-2.5">
                      <span className="font-bold font-mono" style={{ color: scoreColor(s.score) }}>
                        {s.score.toFixed(1)}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 font-mono" style={{ color: 'var(--color-text-primary)' }}>
                      {s.close.toFixed(2)}
                    </td>
                    <td className="px-3 py-2.5 font-mono">
                      <span style={{ color: s.momentum >= 0 ? '#22c55e' : '#ef4444' }}>
                        {s.momentum >= 0 ? '+' : ''}{s.momentum.toFixed(1)}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 font-mono">
                      <span style={{ color: s.trend >= 0 ? '#22c55e' : '#ef4444' }}>
                        {s.trend >= 0 ? '+' : ''}{s.trend.toFixed(1)}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 font-mono text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {s.volatility.toFixed(1)}
                    </td>
                    <td className="px-3 py-2.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {s.industry || '-'}
                    </td>
                    <td className="px-3 py-2.5">
                      {s.recommendation_tier && (
                        <span
                          className="text-xs px-2 py-0.5 rounded-full font-medium"
                          style={{ background: tc.bg, color: tc.text }}
                        >
                          {s.recommendation_tier}
                        </span>
                      )}
                    </td>
                    <td
                      className="px-3 py-2.5 text-xs max-w-40 truncate"
                      style={{ color: 'var(--color-text-muted)' }}
                      title={s.focus_reason}
                    >
                      {s.focus_reason || '-'}
                    </td>
                    <td className="px-3 py-2.5 text-center">
                      <Link
                        to={`/kline?code=${s.code}`}
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
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Empty state */}
      {!loading && data && sorted.length === 0 && (
        <div
          className="flex flex-col items-center justify-center py-20 text-sm"
          style={{ color: 'var(--color-text-muted)' }}
        >
          <Filter className="w-10 h-10 mb-3 opacity-40" />
          没有符合条件的股票，请调整筛选参数
        </div>
      )}

      {/* Initial state */}
      {!data && !loading && (
        <div
          className="flex flex-col items-center justify-center py-20 text-sm"
          style={{ color: 'var(--color-text-muted)' }}
        >
          <Search className="w-10 h-10 mb-3 opacity-40" />
          设置筛选条件后点击"筛选"开始选股
        </div>
      )}
    </div>
  );
}
