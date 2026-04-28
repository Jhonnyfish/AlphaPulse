import { useState, useCallback, useMemo } from 'react';
import { screenerApi } from '@/lib/api';
import DegradedBanner from '@/components/DegradedBanner';
import EChart from '@/components/charts/EChart';
import EmptyState from '@/components/EmptyState';
import ErrorState from '@/components/ErrorState';
import { Filter, ArrowUpDown, TrendingUp, Search, X, Target, Hash, BarChart3, PieChart as PieChartIcon } from 'lucide-react';

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
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  filters: Record<string, any>;
}

type SortField = 'rank' | 'score' | 'momentum' | 'trend' | 'close' | 'volatility';

export default function ScreenerPage() {
  const [data, setData] = useState<ScreenerResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [degraded, setDegraded] = useState(false);
  const [sortField, setSortField] = useState<SortField>('score');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');

  // Filter inputs
  const [minScore, setMinScore] = useState('60');
  const [minMomentum, setMinMomentum] = useState('');
  const [minTrend, setMinTrend] = useState('');
  const [industry, setIndustry] = useState('');

  // Active filter tags
  const activeFilters = useMemo(() => {
    const tags: { key: string; label: string; value: string; icon: typeof Target }[] = [];
    if (minScore) tags.push({ key: 'minScore', label: '最低评分', value: minScore, icon: Hash });
    if (minMomentum) tags.push({ key: 'minMomentum', label: '最低动量', value: minMomentum, icon: TrendingUp });
    if (minTrend) tags.push({ key: 'minTrend', label: '最低趋势', value: minTrend, icon: TrendingUp });
    if (industry) tags.push({ key: 'industry', label: '行业', value: industry, icon: Filter });
    return tags;
  }, [minScore, minMomentum, minTrend, industry]);

  const removeFilter = (key: string) => {
    switch (key) {
      case 'minScore': setMinScore(''); break;
      case 'minMomentum': setMinMomentum(''); break;
      case 'minTrend': setMinTrend(''); break;
      case 'industry': setIndustry(''); break;
    }
  };

  // Industry distribution pie chart
  const industryPieOption = useMemo(() => {
    if (!data?.results?.length) return null;
    const industryCount: Record<string, number> = {};
    data.results.forEach(r => {
      const ind = r.industry || '未知';
      industryCount[ind] = (industryCount[ind] || 0) + 1;
    });
    const sorted = Object.entries(industryCount).sort((a, b) => b[1] - a[1]);
    const top = sorted.slice(0, 10);
    const others = sorted.slice(10).reduce((sum, [, v]) => sum + v, 0);
    const pieData = [...top];
    if (others > 0) pieData.push(['其他', others]);

    return {
      tooltip: { trigger: 'item', formatter: '{b}: {c}只 ({d}%)' },
      color: ['#3b82f6', '#22c55e', '#ef4444', '#f59e0b', '#8b5cf6', '#ec4899', '#06b6d4', '#f97316', '#14b8a6', '#6366f1', '#94a3b8'],
      series: [{
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: true,
        itemStyle: { borderRadius: 6, borderColor: '#222536', borderWidth: 2 },
        label: { show: true, color: '#94a3b8', fontSize: 11, formatter: '{b}\n{d}%' },
        emphasis: {
          label: { show: true, fontSize: 13, fontWeight: 'bold' },
          itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.5)' }
        },
        data: pieData.map(([name, value]) => ({ name, value }))
      }]
    };
  }, [data]);

  // Score distribution bar chart
  const scoreBarOption = useMemo(() => {
    if (!data?.results?.length) return null;
    const ranges = [
      { label: '0-20', min: 0, max: 20 },
      { label: '20-40', min: 20, max: 40 },
      { label: '40-60', min: 40, max: 60 },
      { label: '60-70', min: 60, max: 70 },
      { label: '70-80', min: 70, max: 80 },
      { label: '80-90', min: 80, max: 90 },
      { label: '90-100', min: 90, max: 100 },
    ];
    const counts = ranges.map(r => data.results.filter(d => d.score >= r.min && d.score < r.max).length);
    // last range includes 100
    counts[counts.length - 1] += data.results.filter(d => d.score === 100).length;

    const colors = ['#ef4444', '#f97316', '#f59e0b', '#eab308', '#22c55e', '#10b981', '#3b82f6'];

    return {
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      grid: { left: '3%', right: '4%', bottom: '3%', top: '10%', containLabel: true },
      xAxis: {
        type: 'category',
        data: ranges.map(r => r.label),
        axisLabel: { color: '#94a3b8', fontSize: 11 },
        axisLine: { lineStyle: { color: '#334155' } },
        axisTick: { show: false },
      },
      yAxis: {
        type: 'value',
        axisLabel: { color: '#94a3b8', fontSize: 11 },
        splitLine: { lineStyle: { color: '#1e293b' } },
        axisLine: { show: false },
      },
      series: [{
        type: 'bar',
        data: counts.map((v, i) => ({ value: v, itemStyle: { color: colors[i], borderRadius: [4, 4, 0, 0] } })),
        barWidth: '60%',
        label: { show: true, position: 'top', color: '#94a3b8', fontSize: 11, formatter: '{c}只' },
      }]
    };
  }, [data]);

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
      setDegraded(!!res.data.degraded);
    } catch {
      setError('选股筛选失败');
      setDegraded(false);
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

      <DegradedBanner
        visible={degraded}
        message="Alpha300 排名服务暂时不可用，筛选功能暂不可用"
      />

      {/* Filter bar */}
      <div
        className="rounded-lg border p-4 space-y-3"
        style={{
          background: 'rgba(34,37,54,0.7)',
          backdropFilter: 'blur(18px)',
          borderColor: 'var(--color-border)',
        }}
      >
        // eslint-disable-next-line react-hooks/static-components
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

      {/* Active filter tags */}
      {activeFilters.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          <Target className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
          <span className="text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>筛选条件:</span>
          {activeFilters.map(({ key, label, value, icon: Icon }) => (
            <button
              key={key}
              onClick={() => removeFilter(key)}
              className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-all hover:opacity-80 animate-fade-in"
              style={{
                background: 'rgba(59,130,246,0.15)',
                color: '#60a5fa',
                border: '1px solid rgba(59,130,246,0.3)',
              }}
              title="点击移除"
            >
              <Icon className="w-3 h-3" />
              {label}: {value}
              <X className="w-3 h-3 opacity-60" />
            </button>
          ))}
        </div>
      )}

      {/* Charts section */}
      {data && data.results.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Industry distribution pie */}
          <div
            className="rounded-lg border p-4"
            style={{
              background: 'rgba(34,37,54,0.7)',
              backdropFilter: 'blur(18px)',
              borderColor: 'var(--color-border)',
            }}
          >
            <div className="flex items-center gap-2 mb-3">
              <PieChartIcon className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
              <h3 className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
                行业分布
              </h3>
            </div>
            <div style={{ height: 260 }}>
              {industryPieOption && (
                <EChart option={industryPieOption} height="100%" />
              )}
            </div>
          </div>

          {/* Score distribution bar */}
          <div
            className="rounded-lg border p-4"
            style={{
              background: 'rgba(34,37,54,0.7)',
              backdropFilter: 'blur(18px)',
              borderColor: 'var(--color-border)',
            }}
          >
            <div className="flex items-center gap-2 mb-3">
              <BarChart3 className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
              <h3 className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
                评分分布
              </h3>
            </div>
            <div style={{ height: 260 }}>
              {scoreBarOption && (
                <EChart option={scoreBarOption} height="100%" />
              )}
            </div>
          </div>
        </div>
      )}

      {/* Error */}
      {error && (
        <ErrorState
          title="筛选失败"
          description={error}
          onRetry={() => { setError(''); doScreen(); }}
        />
      )}

      {/* Loading */}
      {loading && (
        <div className="space-y-3 animate-pulse">
          <div className="grid grid-cols-2 gap-4">
            <div className="rounded-lg border p-4" style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)', height: 300 }} />
            <div className="rounded-lg border p-4" style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)', height: 300 }} />
          </div>
          <div className="rounded-lg border" style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)', height: 400 }} />
        </div>
      )}

      {/* Results table */}
      {data && sorted.length > 0 && (
        <div
          className="rounded-lg border overflow-x-auto"
          style={{
            background: 'rgba(34,37,54,0.7)',
            backdropFilter: 'blur(18px)',
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
                // eslint-disable-next-line react-hooks/static-components
                </th>
                // eslint-disable-next-line react-hooks/static-components
                <SortHeader field="score" label="评分" />
                // eslint-disable-next-line react-hooks/static-components
                <SortHeader field="close" label="现价" />
                // eslint-disable-next-line react-hooks/static-components
                <SortHeader field="momentum" label="动量" />
                // eslint-disable-next-line react-hooks/static-components
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
                      <button
                        onClick={() => {/* Could trigger navigation to kline view */}}
                        className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors"
                        style={{
                          background: 'var(--color-bg-hover)',
                          color: 'var(--color-accent)',
                        }}
                      >
                        <TrendingUp className="w-3 h-3" />
                        K线
                      </button>
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
        <EmptyState
          icon={Filter}
          title="无筛选结果"
          description="尝试调整筛选条件"
        />
      )}

      {/* Initial state */}
      {!data && !loading && !error && (
        <EmptyState
          icon={Search}
          title="选股器"
          description="设置筛选条件后点击筛选开始选股"
        />
      )}
    </div>
  );
}
