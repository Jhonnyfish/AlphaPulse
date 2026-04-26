import { useState, useEffect, useCallback, useMemo } from 'react';
import { candidatesApi } from '@/lib/api';
import {
  Target, RefreshCw, TrendingUp, Star, Eye, Zap,
  ArrowUpDown, ChevronUp, ChevronDown, Flame, Shield,
  BarChart3, Plus, Check
} from 'lucide-react';
import EChart from '@/components/charts/EChart';
import type { EChartsOption } from 'echarts';

/* ─── Types ─── */
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
  leader_label?: string;
  harvest_risk_level: string;
  focus_rank: number;
  focus_score: number;
  recommendation_tier: string;
  focus_reason: string;
  harvest_risk_note: string;
  in_watchlist: boolean;
}

interface CandidatesResponse {
  limit: number;
  items: Candidate[];
  tier_counts: Record<string, number>;
  fetched_at: string;
}

type SortKey = 'rank' | 'score' | 'momentum' | 'trend' | 'volatility' | 'close';
type TabKey = 'focus' | 'observe' | 'all';

/* ─── Helpers ─── */
const TIER_CONFIG: Record<string, { label: string; color: string; bg: string; icon: React.ReactNode }> = {
  focus:  { label: '重点关注', color: '#ef4444', bg: 'rgba(239,68,68,0.12)',  icon: <Flame className="w-3.5 h-3.5" /> },
  observe:{ label: '观察池',   color: '#f59e0b', bg: 'rgba(245,158,11,0.12)', icon: <Eye className="w-3.5 h-3.5" /> },
  watch:  { label: '关注',     color: '#6b7280', bg: 'rgba(107,114,128,0.1)', icon: <Star className="w-3.5 h-3.5" /> },
};

const RISK_CONFIG: Record<string, { label: string; color: string }> = {
  low:    { label: '低', color: '#22c55e' },
  medium: { label: '中', color: '#f59e0b' },
  high:   { label: '高', color: '#ef4444' },
};

function tierOf(c: Candidate) {
  return TIER_CONFIG[c.recommendation_tier] || TIER_CONFIG.watch;
}

function scoreColor(score: number) {
  if (score >= 1.5) return '#ef4444';
  if (score >= 1.0) return '#f59e0b';
  if (score >= 0.5) return '#3b82f6';
  return '#6b7280';
}

function factorBar(value: number, max: number, color: string) {
  const pct = Math.min(100, Math.abs(value) / max * 100);
  return (
    <div className="flex items-center gap-1.5">
      <div className="w-16 h-1.5 rounded-full overflow-hidden" style={{ background: 'rgba(255,255,255,0.06)' }}>
        <div className="h-full rounded-full transition-all" style={{ width: `${pct}%`, background: color }} />
      </div>
      <span className="font-mono text-xs" style={{ color, minWidth: 36 }}>
        {value >= 0 ? '+' : ''}{value.toFixed(2)}
      </span>
    </div>
  );
}

/* ─── Radar Chart ─── */
function FactorRadar({ candidate }: { candidate: Candidate }) {
  const option: EChartsOption = {
    radar: {
      indicator: [
        { name: '动量', max: 1 },
        { name: '趋势', max: 0.5 },
        { name: '波动率', max: 0.1 },
        { name: '流动性', max: 20 },
      ],
      shape: 'polygon',
      splitArea: { areaStyle: { color: ['rgba(59,130,246,0.02)', 'rgba(59,130,246,0.05)'] } },
      axisLine: { lineStyle: { color: 'rgba(148,163,184,0.15)' } },
      splitLine: { lineStyle: { color: 'rgba(148,163,184,0.1)' } },
      axisName: { color: '#94a3b8', fontSize: 11 },
    },
    series: [{
      type: 'radar',
      data: [{
        value: [candidate.momentum, candidate.trend, candidate.volatility, candidate.liquidity],
        name: candidate.name,
        areaStyle: { color: 'rgba(59,130,246,0.15)' },
        lineStyle: { color: '#3b82f6', width: 2 },
        itemStyle: { color: '#3b82f6' },
      }],
    }],
  };
  return <EChart option={option} height={220} />;
}

/* ─── Trade Plan Card ─── */
function TradePlan({ c }: { c: Candidate }) {
  const riskColor = RISK_CONFIG[c.harvest_risk_level]?.color || '#6b7280';
  return (
    <div className="rounded-lg p-4 space-y-3" style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)' }}>
      <div className="text-sm font-bold mb-2" style={{ color: 'var(--color-text-primary)' }}>📐 交易计划</div>
      <div className="grid grid-cols-2 gap-3 text-xs">
        <div>
          <div className="mb-1" style={{ color: 'var(--color-text-muted)' }}>买入区间</div>
          <div className="font-mono font-bold" style={{ color: '#22c55e' }}>
            ¥{c.buy_low.toFixed(2)} ~ ¥{c.buy_high.toFixed(2)}
          </div>
        </div>
        <div>
          <div className="mb-1" style={{ color: 'var(--color-text-muted)' }}>止损价</div>
          <div className="font-mono font-bold" style={{ color: '#ef4444' }}>
            ¥{c.stop_loss.toFixed(2)}
          </div>
        </div>
        <div>
          <div className="mb-1" style={{ color: 'var(--color-text-muted)' }}>止盈区间</div>
          <div className="font-mono font-bold" style={{ color: '#3b82f6' }}>
            ¥{c.sell_low.toFixed(2)} ~ ¥{c.sell_high.toFixed(2)}
          </div>
        </div>
        <div>
          <div className="mb-1" style={{ color: 'var(--color-text-muted)' }}>ATR14</div>
          <div className="font-mono font-bold" style={{ color: 'var(--color-text-primary)' }}>
            ¥{c.atr14.toFixed(2)}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-3 pt-2" style={{ borderTop: '1px solid rgba(255,255,255,0.06)' }}>
        <span className="text-xs px-2 py-0.5 rounded" style={{ background: `${riskColor}15`, color: riskColor }}>
          <Shield className="w-3 h-3 inline mr-1" />
          收割风险: {RISK_CONFIG[c.harvest_risk_level]?.label || c.harvest_risk_level}
        </span>
        {c.leader_signal && c.leader_signal !== 'none' && (
          <span className="text-xs px-2 py-0.5 rounded" style={{ background: 'rgba(239,68,68,0.12)', color: '#ef4444' }}>
            🔥 {c.leader_label || c.leader_signal}
          </span>
        )}
      </div>
      {c.focus_reason && (
        <div className="text-xs leading-relaxed" style={{ color: 'var(--color-text-muted)' }}>
          💡 {c.focus_reason}
        </div>
      )}
      {c.harvest_risk_note && (
        <div className="text-xs leading-relaxed" style={{ color: 'var(--color-text-muted)' }}>
          ⚠️ {c.harvest_risk_note}
        </div>
      )}
    </div>
  );
}

/* ─── Main Component ─── */
export default function CandidatesPage() {
  const [data, setData] = useState<CandidatesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<TabKey>('focus');
  const [sortKey, setSortKey] = useState<SortKey>('rank');
  const [sortAsc, setSortAsc] = useState(true);
  const [selected, setSelected] = useState<Candidate | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await candidatesApi.list({ limit: 50 });
      setData(res.data);
    } catch {
      setError('加载 Alpha300 排行榜失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const items = data?.items || [];

  /* Filtered by tab */
  const filtered = useMemo(() => {
    if (tab === 'all') return items;
    return items.filter(c => c.recommendation_tier === tab);
  }, [items, tab]);

  /* Sorted */
  const sorted = useMemo(() => {
    const copy = [...filtered];
    copy.sort((a, b) => {
      const av = a[sortKey], bv = b[sortKey];
      if (typeof av === 'number' && typeof bv === 'number') {
        return sortAsc ? av - bv : bv - av;
      }
      return 0;
    });
    return copy;
  }, [filtered, sortKey, sortAsc]);

  function toggleSort(key: SortKey) {
    if (sortKey === key) setSortAsc(!sortAsc);
    else { setSortKey(key); setSortAsc(key === 'rank'); }
  }

  function SortHeader({ label, k }: { label: string; k: SortKey }) {
    const active = sortKey === k;
    return (
      <th
        className="px-3 py-2.5 text-left text-xs font-medium cursor-pointer select-none transition-colors hover:text-white"
        style={{ color: active ? '#3b82f6' : 'var(--color-text-muted)' }}
        onClick={() => toggleSort(k)}
      >
        <span className="flex items-center gap-1">
          {label}
          {active ? (
            sortAsc ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />
          ) : (
            <ArrowUpDown className="w-3 h-3 opacity-30" />
          )}
        </span>
      </th>
    );
  }

  /* Stats */
  const focusCount = items.filter(c => c.recommendation_tier === 'focus').length;
  const observeCount = items.filter(c => c.recommendation_tier === 'observe').length;
  const leaderCount = items.filter(c => c.leader_signal && c.leader_signal !== 'none').length;
  const avgScore = items.length ? (items.reduce((s, c) => s + c.score, 0) / items.length).toFixed(2) : '0';

  return (
    <div className="p-4 lg:p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <BarChart3 className="w-5 h-5" style={{ color: '#3b82f6' }} />
          <h1 className="text-xl font-bold">Alpha300 排行榜</h1>
          {data && (
            <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>
              {items.length} 只 · {data.fetched_at ? new Date(data.fetched_at).toLocaleString() : ''}
            </span>
          )}
        </div>
        <button onClick={fetchData} disabled={loading} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}>
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        {[
          { label: '重点关注', value: focusCount, icon: <Flame className="w-4 h-4" />, color: '#ef4444' },
          { label: '观察池', value: observeCount, icon: <Eye className="w-4 h-4" />, color: '#f59e0b' },
          { label: '龙头信号', value: leaderCount, icon: <Zap className="w-4 h-4" />, color: '#3b82f6' },
          { label: '平均评分', value: avgScore, icon: <Target className="w-4 h-4" />, color: '#8b5cf6' },
        ].map(s => (
          <div key={s.label} className="rounded-xl p-3 transition-all" style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)', backdropFilter: 'blur(18px)' }}>
            <div className="flex items-center gap-2 mb-1">
              <span style={{ color: s.color }}>{s.icon}</span>
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{s.label}</span>
            </div>
            <div className="text-2xl font-bold font-mono" style={{ color: s.color }}>{s.value}</div>
          </div>
        ))}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 p-1 rounded-lg" style={{ background: 'rgba(255,255,255,0.03)' }}>
        {([['focus', '🔥 重点关注'], ['observe', '👀 观察池'], ['all', '📋 全部排行']] as [TabKey, string][]).map(([k, label]) => (
          <button key={k} onClick={() => setTab(k)} className="flex-1 px-3 py-2 rounded-lg text-sm font-medium transition-all" style={{
            background: tab === k ? 'rgba(59,130,246,0.15)' : 'transparent',
            color: tab === k ? '#3b82f6' : 'var(--color-text-muted)',
            border: tab === k ? '1px solid rgba(59,130,246,0.3)' : '1px solid transparent',
          }}>
            {label}
            <span className="ml-1.5 font-mono text-xs">
              {k === 'all' ? items.length : items.filter(c => c.recommendation_tier === k).length}
            </span>
          </button>
        ))}
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-xl p-6 text-center" style={{ background: 'rgba(239,68,68,0.05)', border: '1px solid rgba(239,68,68,0.2)' }}>
          <div className="text-sm" style={{ color: '#ef4444' }}>{error}</div>
          <button onClick={fetchData} className="mt-2 text-xs px-3 py-1 rounded" style={{ background: 'rgba(239,68,68,0.1)', color: '#ef4444' }}>重试</button>
        </div>
      )}

      {/* Loading */}
      {loading && !data && (
        <div className="space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-12 rounded-lg animate-pulse" style={{ background: 'var(--color-bg-secondary)' }} />
          ))}
        </div>
      )}

      {/* Table */}
      {sorted.length > 0 && (
        <div className="rounded-xl overflow-hidden" style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)', backdropFilter: 'blur(18px)' }}>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr style={{ borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
                  <SortHeader label="排名" k="rank" />
                  <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>代码 / 名称</th>
                  <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>行业</th>
                  <SortHeader label="评分" k="score" />
                  <SortHeader label="动量" k="momentum" />
                  <SortHeader label="趋势" k="trend" />
                  <SortHeader label="波动率" k="volatility" />
                  <SortHeader label="收盘价" k="close" />
                  <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>买入区间</th>
                  <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>止盈区间</th>
                  <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>止损</th>
                  <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>层级</th>
                  <th className="px-3 py-2.5 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>信号</th>
                  <th className="px-3 py-2.5 text-center text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {sorted.map(c => {
                  const tier = tierOf(c);
                  const isSelected = selected?.code === c.code;
                  return (
                    <tr key={c.code} onClick={() => setSelected(isSelected ? null : c)} className="cursor-pointer transition-colors" style={{
                      borderBottom: '1px solid rgba(255,255,255,0.04)',
                      background: isSelected ? 'rgba(59,130,246,0.08)' : 'transparent',
                    }}
                      onMouseEnter={e => { if (!isSelected) e.currentTarget.style.background = 'rgba(255,255,255,0.02)'; }}
                      onMouseLeave={e => { if (!isSelected) e.currentTarget.style.background = 'transparent'; }}
                    >
                      <td className="px-3 py-2.5">
                        <span className="font-mono font-bold text-sm" style={{ color: scoreColor(c.score) }}>
                          #{c.rank}
                        </span>
                      </td>
                      <td className="px-3 py-2.5">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm font-bold" style={{ color: 'var(--color-text-primary)' }}>{c.code}</span>
                          <span className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>{c.name}</span>
                          {c.limit_up_today && <span className="text-xs px-1 rounded" style={{ background: 'rgba(239,68,68,0.15)', color: '#ef4444' }}>涨停</span>}
                          {c.limit_up_prev_day && <span className="text-xs px-1 rounded" style={{ background: 'rgba(245,158,11,0.15)', color: '#f59e0b' }}>昨涨停</span>}
                        </div>
                      </td>
                      <td className="px-3 py-2.5 text-xs max-w-32 truncate" style={{ color: 'var(--color-text-muted)' }}>
                        {c.industry?.replace(/^C\d+/, '') || '-'}
                      </td>
                      <td className="px-3 py-2.5">
                        <div className="flex items-center gap-2">
                          <div className="w-12 h-1.5 rounded-full overflow-hidden" style={{ background: 'rgba(255,255,255,0.06)' }}>
                            <div className="h-full rounded-full" style={{ width: `${Math.min(100, c.score / 2 * 100)}%`, background: scoreColor(c.score) }} />
                          </div>
                          <span className="font-mono text-xs font-bold" style={{ color: scoreColor(c.score) }}>{c.score.toFixed(2)}</span>
                        </div>
                      </td>
                      <td className="px-3 py-2.5">{factorBar(c.momentum, 1, c.momentum >= 0 ? '#ef4444' : '#22c55e')}</td>
                      <td className="px-3 py-2.5">{factorBar(c.trend, 0.5, c.trend >= 0 ? '#ef4444' : '#22c55e')}</td>
                      <td className="px-3 py-2.5">{factorBar(c.volatility, 0.1, '#8b5cf6')}</td>
                      <td className="px-3 py-2.5 font-mono text-sm" style={{ color: 'var(--color-text-primary)' }}>¥{c.close.toFixed(2)}</td>
                      <td className="px-3 py-2.5 font-mono text-xs" style={{ color: '#22c55e' }}>
                        {c.buy_low.toFixed(2)}~{c.buy_high.toFixed(2)}
                      </td>
                      <td className="px-3 py-2.5 font-mono text-xs" style={{ color: '#3b82f6' }}>
                        {c.sell_low.toFixed(2)}~{c.sell_high.toFixed(2)}
                      </td>
                      <td className="px-3 py-2.5 font-mono text-xs" style={{ color: '#ef4444' }}>
                        {c.stop_loss.toFixed(2)}
                      </td>
                      <td className="px-3 py-2.5">
                        <span className="text-xs px-2 py-0.5 rounded-full font-medium" style={{ background: tier.bg, color: tier.color }}>
                          {tier.icon} {tier.label}
                        </span>
                      </td>
                      <td className="px-3 py-2.5">
                        {c.leader_signal && c.leader_signal !== 'none' ? (
                          <span className="text-xs px-1.5 py-0.5 rounded" style={{ background: 'rgba(239,68,68,0.12)', color: '#ef4444' }}>
                            {c.leader_label || c.leader_signal}
                          </span>
                        ) : (
                          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>-</span>
                        )}
                      </td>
                      <td className="px-3 py-2.5 text-center">
                        <button
                          onClick={e => { e.stopPropagation(); /* TODO: add to watchlist */ }}
                          className="p-1.5 rounded-lg transition-colors"
                          style={{ background: c.in_watchlist ? 'rgba(34,197,94,0.1)' : 'rgba(255,255,255,0.03)', color: c.in_watchlist ? '#22c55e' : 'var(--color-text-muted)' }}
                          title={c.in_watchlist ? '已在自选' : '加入自选'}
                        >
                          {c.in_watchlist ? <Check className="w-3.5 h-3.5" /> : <Plus className="w-3.5 h-3.5" />}
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Detail Panel — shown when a row is selected */}
      {selected && (
        <div className="rounded-xl p-4 space-y-4" style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(59,130,246,0.2)', backdropFilter: 'blur(18px)' }}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <span className="font-mono font-bold text-lg" style={{ color: scoreColor(selected.score) }}>#{selected.rank}</span>
              <span className="font-bold text-lg" style={{ color: 'var(--color-text-primary)' }}>{selected.name}</span>
              <span className="font-mono text-sm" style={{ color: 'var(--color-text-muted)' }}>{selected.code}</span>
              <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: tierOf(selected).bg, color: tierOf(selected).color }}>
                {tierOf(selected).label}
              </span>
            </div>
            <button onClick={() => setSelected(null)} className="text-xs px-2 py-1 rounded" style={{ background: 'rgba(255,255,255,0.05)', color: 'var(--color-text-muted)' }}>
              收起详情
            </button>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* Radar */}
            <div className="rounded-lg p-3" style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)' }}>
              <div className="text-sm font-bold mb-2" style={{ color: 'var(--color-text-primary)' }}>📊 因子雷达</div>
              <FactorRadar candidate={selected} />
            </div>

            {/* Trade Plan */}
            <TradePlan c={selected} />
          </div>
        </div>
      )}

      {/* Empty */}
      {!loading && sorted.length === 0 && (
        <div className="rounded-xl p-8 text-center" style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)' }}>
          <Target className="w-8 h-8 mx-auto mb-2" style={{ color: 'var(--color-text-muted)' }} />
          <div className="text-sm" style={{ color: 'var(--color-text-muted)' }}>暂无数据</div>
        </div>
      )}
    </div>
  );
}
