import { useState, useEffect, useCallback } from 'react';
import { candidatesApi } from '@/lib/api';
import { Target, RefreshCw, TrendingUp, Star, Eye, Zap, ChevronDown, ChevronUp } from 'lucide-react';
import StockDetailModal from '@/components/StockDetailModal';

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

interface TierConfig {
  key: string;
  label: string;
  icon: React.ReactNode;
  color: string;
  borderColor: string;
  bgGlow: string;
  minScore: number;
  maxScore: number;
}

const TIERS: TierConfig[] = [
  {
    key: 'core',
    label: '核心',
    icon: <Star className="w-4 h-4" />,
    color: '#3b82f6',
    borderColor: 'rgba(59, 130, 246, 0.5)',
    bgGlow: 'rgba(59, 130, 246, 0.08)',
    minScore: 80,
    maxScore: 100,
  },
  {
    key: 'watch',
    label: '关注',
    icon: <Zap className="w-4 h-4" />,
    color: '#f59e0b',
    borderColor: 'rgba(245, 158, 11, 0.3)',
    bgGlow: 'rgba(245, 158, 11, 0.05)',
    minScore: 60,
    maxScore: 79.99,
  },
  {
    key: 'observe',
    label: '观察',
    icon: <Eye className="w-4 h-4" />,
    color: '#6b7280',
    borderColor: 'rgba(107, 114, 128, 0.2)',
    bgGlow: 'transparent',
    minScore: 0,
    maxScore: 59.99,
  },
];

function getCandidateTier(score: number): TierConfig {
  if (score >= 80) return TIERS[0];
  if (score >= 60) return TIERS[1];
  return TIERS[2];
}

function CandidateCard({ candidate, onSelect }: { candidate: Candidate; onSelect: (c: Candidate) => void }) {
  const tier = getCandidateTier(candidate.score);
  const momentumColor = candidate.momentum >= 0 ? '#ef4444' : '#22c55e'; // 红涨绿跌
  const trendColor = candidate.trend >= 0 ? '#ef4444' : '#22c55e';

  return (
    <div
      className="rounded-lg p-3 transition-all duration-200 hover:scale-[1.01] cursor-pointer group"
      onClick={() => onSelect(candidate)}
      style={{
        background: 'rgba(255,255,255,0.03)',
        border: '1px solid rgba(255,255,255,0.06)',
      }}
    >
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <span className="font-mono font-bold text-sm" style={{ color: tier.color }}>
            {candidate.code}
          </span>
          <span className="font-medium text-sm" style={{ color: 'var(--color-text-primary)' }}>
            {candidate.name}
          </span>
          {candidate.limit_up_today && (
            <span className="text-xs px-1.5 py-0.5 rounded" style={{ background: 'rgba(239,68,68,0.15)', color: '#ef4444' }}>
              涨停
            </span>
          )}
          {candidate.leader_signal && (
            <span className="text-xs px-1.5 py-0.5 rounded" style={{ background: 'rgba(59,130,246,0.15)', color: '#3b82f6' }}>
              {candidate.leader_signal}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3">
          <span className="font-mono font-bold text-sm" style={{ color: 'var(--color-text-primary)' }}>
            ¥{candidate.close.toFixed(2)}
          </span>
          <span
            className="font-mono text-xs font-bold px-2 py-0.5 rounded-full"
            style={{
              background: `${tier.color}20`,
              color: tier.color,
            }}
          >
            {candidate.score.toFixed(0)}分
          </span>
        </div>
      </div>

      <div className="flex items-center gap-4 text-xs" style={{ color: 'var(--color-text-muted)' }}>
        <span>
          动量 <span style={{ color: momentumColor }} className="font-mono">
            {candidate.momentum >= 0 ? '+' : ''}{candidate.momentum.toFixed(1)}
          </span>
        </span>
        <span>
          趋势 <span style={{ color: trendColor }} className="font-mono">
            {candidate.trend >= 0 ? '+' : ''}{candidate.trend.toFixed(1)}
          </span>
        </span>
        <span>
          买入 <span className="font-mono" style={{ color: 'var(--color-text-secondary)' }}>
            {candidate.buy_low.toFixed(2)}~{candidate.buy_high.toFixed(2)}
          </span>
        </span>
        <span>
          止损 <span className="font-mono" style={{ color: '#ef4444' }}>
            {candidate.stop_loss.toFixed(2)}
          </span>
        </span>
        {candidate.industry && (
          <span className="opacity-60">{candidate.industry}</span>
        )}
        <span className="ml-auto opacity-0 group-hover:opacity-60 transition-opacity flex items-center gap-0.5">
          <TrendingUp className="w-3 h-3" /> 查看K线
        </span>
      </div>
    </div>
  );
}

function TierSection({
  tier,
  candidates,
  onSelect,
  defaultOpen = true,
}: {
  tier: TierConfig;
  candidates: Candidate[];
  onSelect: (c: Candidate) => void;
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);

  if (candidates.length === 0) return null;

  return (
    <div
      className="rounded-xl overflow-hidden transition-all"
      style={{
        background: tier.bgGlow,
        border: `1px solid ${tier.borderColor}`,
        backdropFilter: 'blur(18px)',
      }}
    >
      {/* Tier Header */}
      <button
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between px-4 py-3 transition-colors hover:bg-white/5"
      >
        <div className="flex items-center gap-2">
          <span style={{ color: tier.color }}>{tier.icon}</span>
          <span className="font-bold text-sm" style={{ color: tier.color }}>
            {tier.label}
          </span>
          <span
            className="text-xs px-2 py-0.5 rounded-full font-mono font-bold"
            style={{ background: `${tier.color}20`, color: tier.color }}
          >
            {candidates.length}
          </span>
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
            评分 {tier.minScore}–{tier.maxScore >= 100 ? '100' : Math.floor(tier.maxScore)}
          </span>
        </div>
        {open ? (
          <ChevronUp className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
        ) : (
          <ChevronDown className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
        )}
      </button>

      {/* Tier Content */}
      {open && (
        <div className="px-3 pb-3 space-y-2">
          {candidates.map((c) => (
            <CandidateCard key={c.code} candidate={c} onSelect={onSelect} />
          ))}
        </div>
      )}
    </div>
  );
}

export default function CandidatesPage() {
  const [data, setData] = useState<CandidatesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedStock, setSelectedStock] = useState<Candidate | null>(null);

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

  // Group candidates by tier
  const grouped = TIERS.map((tier) => ({
    tier,
    candidates: (data?.items || [])
      .filter((c) => c.score >= tier.minScore && c.score <= tier.maxScore)
      .sort((a, b) => b.score - a.score),
  }));

  const totalCandidates = data?.items?.length || 0;

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
              {totalCandidates} 只 · 更新于 {new Date(data.fetched_at).toLocaleTimeString()}
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

      {/* Tier summary badges */}
      {data && totalCandidates > 0 && (
        <div className="flex gap-3 flex-wrap">
          {grouped.map(({ tier, candidates }) => (
            <div
              key={tier.key}
              className="flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-lg"
              style={{
                background: `${tier.color}15`,
                color: tier.color,
                border: `1px solid ${tier.borderColor}`,
              }}
            >
              {tier.icon}
              <span className="font-medium">{tier.label}</span>
              <span className="font-bold font-mono">{candidates.length}</span>
            </div>
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
        <div className="space-y-4">
          {TIERS.map((tier) => (
            <div
              key={tier.key}
              className="rounded-xl p-4 animate-pulse"
              style={{ background: 'var(--color-bg-secondary)', border: `1px solid ${tier.borderColor}` }}
            >
              <div className="flex items-center gap-2 mb-3">
                <div className="w-4 h-4 rounded" style={{ background: 'var(--color-bg-hover)' }} />
                <div className="w-12 h-4 rounded" style={{ background: 'var(--color-bg-hover)' }} />
                <div className="w-8 h-5 rounded-full" style={{ background: 'var(--color-bg-hover)' }} />
              </div>
              <div className="space-y-2">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="h-14 rounded-lg" style={{ background: 'var(--color-bg-hover)' }} />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Tier Sections */}
      {data && totalCandidates > 0 && (
        <div className="space-y-4">
          {grouped.map(({ tier, candidates }) => (
            <TierSection
              key={tier.key}
              tier={tier}
              candidates={candidates}
              onSelect={setSelectedStock}
              defaultOpen={tier.key !== 'observe'}
            />
          ))}
        </div>
      )}

      {/* Empty state */}
      {data && totalCandidates === 0 && (
        <div
          className="flex flex-col items-center justify-center py-20 text-sm"
          style={{ color: 'var(--color-text-muted)' }}
        >
          <Target className="w-10 h-10 mb-3 opacity-40" />
          暂无候选股数据
        </div>
      )}

      {/* Stock Detail Modal */}
      {selectedStock && (
        <StockDetailModal
          stock={selectedStock}
          onClose={() => setSelectedStock(null)}
        />
      )}
    </div>
  );
}
