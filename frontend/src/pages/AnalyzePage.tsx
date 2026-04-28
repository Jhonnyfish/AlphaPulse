import { useState, useEffect, useCallback } from 'react';
import { useView } from '@/lib/ViewContext';
import { analyzeApi, type AnalyzeResult } from '@/lib/api';

// Raw API response type (actual backend structure)
interface RawAnalyzeResponse {
  code: string;
  name: string;
  fetched_at: string;
  quote: Record<string, unknown>;
  order_flow: { verdict: string; [k: string]: unknown };
  volume_price: { verdict: string; turnover_level?: string; [k: string]: unknown };
  valuation: { verdict: string; pe_level?: string; pb_level?: string; [k: string]: unknown };
  volatility: { verdict: string; amplitude_level?: string; [k: string]: unknown };
  money_flow: { verdict: string; [k: string]: unknown };
  technical: { verdict: string; [k: string]: unknown };
  sector: { verdict: string; [k: string]: unknown };
  sentiment: { verdict: string; sentiment_score?: number; [k: string]: unknown };
  summary: {
    overall_score: number;
    overall_signal: string;
    strengths: string[];
    risks: string[];
    suggestion: string;
  };
}

function levelToScore(level: string | undefined): number {
  if (!level) return 5;
  const map: Record<string, number> = {
    '偏低': 7, '低': 7, '低波动': 6,
    '正常': 5, '中性': 5, '适中': 5,
    '偏高': 3, '高': 3, '高波动': 4,
    '强势': 8, '偏强': 7, '偏弱': 3, '弱势': 2,
    '大盘股': 6, '中盘股': 5, '小盘股': 4,
  };
  return map[level] ?? 5;
}

function transformResponse(raw: RawAnalyzeResponse): AnalyzeResult {
  const s = raw.summary;
  const strengths = s.strengths ?? [];
  const risks = s.risks ?? [];
  const dims: { name: string; score: number; detail: string }[] = [
    { name: 'order_flow',   score: levelToScore(raw.order_flow.net_direction as string),   detail: raw.order_flow.verdict },
    { name: 'volume_price', score: levelToScore(raw.volume_price.turnover_level),          detail: raw.volume_price.verdict },
    { name: 'valuation',    score: levelToScore(raw.valuation.pe_level),                   detail: raw.valuation.verdict },
    { name: 'volatility',   score: levelToScore(raw.volatility.amplitude_level),           detail: raw.volatility.verdict },
    { name: 'money_flow',   score: levelToScore(raw.money_flow.today_main_direction as string), detail: raw.money_flow.verdict },
    { name: 'technical',    score: levelToScore(raw.technical.rsi_level as string),        detail: raw.technical.verdict },
    { name: 'sector',       score: raw.sector.is_sector_leader ? 7 : 5,                   detail: raw.sector.verdict },
    { name: 'sentiment',    score: Math.max(1, Math.min(10, ((strengths.length - risks.length) * 1.5) + 5)), detail: raw.sentiment.verdict },
  ];

  const parts: string[] = [s.suggestion];
  if (strengths.length) parts.push(`优势: ${strengths.join('、')}`);
  if (risks.length) parts.push(`风险: ${risks.join('、')}`);

  // Detect staleness: if fetched_at date differs from today
  const fetchedDate = raw.fetched_at ? raw.fetched_at.slice(0, 10) : '';
  const today = new Date().toISOString().slice(0, 10);
  const dataStale = fetchedDate !== today && fetchedDate !== '';

  return {
    code: raw.code,
    name: raw.name,
    score: s.overall_score / 10,
    dimensions: dims,
    recommendation: s.overall_signal,
    summary: parts.join('。'),
    fetched_at: raw.fetched_at,
    data_stale: dataStale,
  };
}
import EChart from '@/components/charts/EChart';
import StockSearch from '@/components/StockSearch';
import Alpha300Selector from '@/components/Alpha300Selector';
import { Skeleton, SkeletonCard } from '@/components/ui/Skeleton';
import { Search, TrendingUp, TrendingDown, Star, Activity, BarChart3, Shield, Zap, AlertCircle, RefreshCw } from 'lucide-react';

const DIMENSION_LABELS: Record<string, string> = {
  order_flow: '订单流',
  volume_price: '量价',
  valuation: '估值',
  volatility: '波动率',
  money_flow: '资金流',
  technical: '技术面',
  sector: '板块',
  sentiment: '情绪',
};

const DIMENSION_ICONS: Record<string, React.ElementType> = {
  order_flow: BarChart3,
  volume_price: Activity,
  valuation: Shield,
  volatility: Zap,
  money_flow: TrendingUp,
  technical: TrendingDown,
  sector: Star,
  sentiment: Search,
};

function scoreColor(score: number): string {
  if (score >= 7) return 'var(--color-success)';
  if (score >= 4) return 'var(--color-warning)';
  return 'var(--color-danger)';
}

function scoreLabel(score: number): string {
  if (score >= 8) return '强势';
  if (score >= 7) return '偏强';
  if (score >= 5) return '中性';
  if (score >= 4) return '偏弱';
  return '弱势';
}

export default function AnalyzePage() {
  const { viewParams, navigate } = useView();
  const code = viewParams.code ?? '';

  const [data, setData] = useState<AnalyzeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [alpha300Open, setAlpha300Open] = useState(false);

  const fetchAnalysis = useCallback(async (stockCode: string) => {
    setLoading(true);
    setError(null);
    setData(null);
    try {
      const res = await analyzeApi.analyze(stockCode);
      setData(transformResponse(res.data as unknown as RawAnalyzeResponse));
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '分析失败，请稍后重试';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (code) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      fetchAnalysis(code);
    } else {
      setData(null);
      setError(null);
      setLoading(false);
    }
  }, [code, fetchAnalysis]);

  const handleSelect = (suggestion: { code: string }) => {
    navigate('analyze', { code: suggestion.code });
  };

  if (!code) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-6">
        <div className="glass-panel p-10 flex flex-col items-center gap-6 max-w-md w-full">
          <Activity className="w-12 h-12" style={{ color: 'var(--color-accent)' }} />
          <h2 className="text-xl font-semibold">个股深度分析</h2>
          <p className="text-sm text-center" style={{ color: 'var(--color-text-secondary)' }}>
            输入股票代码或名称，获取多维度综合评分与雷达图分析
          </p>
          <div className="flex items-center gap-2 w-full">
            <StockSearch onSelect={handleSelect} autoFocus className="flex-1" />
            <button
              type="button"
              onClick={() => setAlpha300Open(true)}
              className="px-3 py-2 rounded-lg text-sm shrink-0 transition-colors hover:bg-[var(--color-bg-hover)]"
              style={{
                background: 'var(--color-bg-card)',
                border: '1px solid var(--color-border)',
                color: 'var(--color-text-secondary)',
              }}
              title="从 Alpha300 选择"
              aria-label="从 Alpha300 选择"
            >
              🎯
            </button>
          </div>
        </div>
        <Alpha300Selector
          open={alpha300Open}
          onClose={() => setAlpha300Open(false)}
          onSelect={(selectedCode) => navigate('analyze', { code: selectedCode })}
        />
      </div>
    );
  }

  return (
    <div className="space-y-4 max-w-7xl mx-auto">
      {/* Search bar */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2 w-72">
          <StockSearch onSelect={handleSelect} className="flex-1" placeholder="切换股票..." />
          <button
            type="button"
            onClick={() => setAlpha300Open(true)}
            className="px-3 py-2 rounded-lg text-sm shrink-0 transition-colors hover:bg-[var(--color-bg-hover)]"
            style={{
              background: 'var(--color-bg-card)',
              border: '1px solid var(--color-border)',
              color: 'var(--color-text-secondary)',
            }}
            title="从 Alpha300 选择"
            aria-label="从 Alpha300 选择"
          >
            🎯
          </button>
        </div>
      </div>

      {error && (
        <div className="glass-panel p-6 flex items-center gap-4">
          <AlertCircle className="w-5 h-5 shrink-0" style={{ color: 'var(--color-danger)' }} />
          <span className="text-sm flex-1" style={{ color: 'var(--color-danger)' }}>{error}</span>
          <button
            onClick={() => fetchAnalysis(code)}
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors"
            style={{ background: 'var(--color-accent)', color: '#fff' }}
          >
            <RefreshCw className="w-3.5 h-3.5" />
            重试
          </button>
        </div>
      )}

      {loading && (
        <>
          <div className="glass-panel p-5 flex items-center gap-4">
            <Skeleton variant="circular" width={48} height={48} />
            <div className="flex-1 space-y-2">
              <Skeleton variant="text" width="30%" height={24} />
              <Skeleton variant="text" width="50%" height={16} />
            </div>
            <Skeleton variant="rectangular" width={80} height={64} className="rounded-lg" />
          </div>
          <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">
            <SkeletonCard className="lg:col-span-3" />
            <div className="lg:col-span-2 space-y-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <SkeletonCard key={i} />
              ))}
            </div>
          </div>
        </>
      )}

      {data && !loading && (
        <>
          {/* Staleness warning */}
          {data.data_stale && data.fetched_at && (
            <div
              className="glass-panel px-4 py-2.5 flex items-center gap-2 text-xs"
              style={{ background: 'rgba(245,158,11,0.08)', borderColor: 'rgba(245,158,11,0.2)' }}
            >
              <AlertCircle className="w-3.5 h-3.5 shrink-0" style={{ color: '#f59e0b' }} />
              <span style={{ color: '#f59e0b' }}>
                数据截至 {data.fetched_at.slice(0, 10)}，非今日实时数据，仅供参考
              </span>
            </div>
          )}

          {/* Header */}
          <div className="glass-panel p-5 flex items-center gap-4">
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-1">
                <h1 className="text-xl font-bold">{data.name}</h1>
                <span
                  className="font-mono text-xs px-2 py-0.5 rounded"
                  style={{ background: 'var(--color-bg-secondary)', color: 'var(--color-accent)' }}
                >
                  {data.code}
                </span>
              </div>
              <p className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
                综合评分 · {scoreLabel(data.score)}
                {data.fetched_at && (
                  <span className="ml-2 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                    数据 {data.fetched_at.slice(0, 10)}
                  </span>
                )}
              </p>
            </div>
            <div className="text-right">
              <div className="text-4xl font-bold" style={{ color: scoreColor(data.score) }}>
                {data.score.toFixed(1)}
              </div>
              <div className="text-xs mt-1" style={{ color: 'var(--color-text-muted)' }}>
                / 10
              </div>
            </div>
          </div>

          {/* Main content: radar + breakdown */}
          <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">
            {/* Radar chart */}
            <div className="lg:col-span-3 glass-panel p-4">
              <h3 className="text-sm font-medium mb-2" style={{ color: 'var(--color-text-secondary)' }}>
                多维雷达图
              </h3>
              <RadarChart dimensions={data.dimensions} />
            </div>

            {/* Dimension breakdown */}
            <div className="lg:col-span-2 space-y-3">
              {data.dimensions.map((dim) => (
                <DimensionCard key={dim.name} {...dim} />
              ))}
            </div>
          </div>

          {/* Recommendation */}
          <div className="glass-panel p-5">
            <h3 className="text-sm font-medium mb-2" style={{ color: 'var(--color-text-secondary)' }}>
              分析建议
            </h3>
            <div
              className="inline-block text-xs font-semibold px-2 py-1 rounded mb-3"
              style={{
                background: `${scoreColor(data.score)}20`,
                color: scoreColor(data.score),
              }}
            >
              {data.recommendation}
            </div>
            <p className="text-sm leading-relaxed" style={{ color: 'var(--color-text-primary)' }}>
              {data.summary}
            </p>
          </div>
        </>
      )}

      <Alpha300Selector
        open={alpha300Open}
        onClose={() => setAlpha300Open(false)}
        onSelect={(selectedCode) => navigate('analyze', { code: selectedCode })}
      />
    </div>
  );
}

function RadarChart({ dimensions }: { dimensions: AnalyzeResult['dimensions'] }) {
  const indicator = dimensions.map((d) => ({
    name: DIMENSION_LABELS[d.name] ?? d.name,
    max: 10,
  }));

  const values = dimensions.map((d) => d.score);

  const option = {
    tooltip: {
      trigger: 'item' as const,
      formatter: (params: { name?: string; value?: number[] }) => {
        if (!params.value) return '';
        return dimensions
          .map((d, i) => `${DIMENSION_LABELS[d.name] ?? d.name}: ${params.value![i].toFixed(1)}`)
          .join('<br/>');
      },
    },
    radar: {
      indicator,
      shape: 'polygon' as const,
      radius: '70%',
      axisName: { color: '#94a3b8', fontSize: 12 },
      splitArea: { areaStyle: { color: ['rgba(15,23,42,0.2)', 'rgba(15,23,42,0.4)'] } },
      splitLine: { lineStyle: { color: 'rgba(51,65,85,0.5)' } },
      axisLine: { lineStyle: { color: 'rgba(51,65,85,0.5)' } },
    },
    series: [
      {
        type: 'radar',
        data: [
          {
            value: values,
            name: '评分',
            areaStyle: {
              color: {
                type: 'radial',
                x: 0.5,
                y: 0.5,
                r: 0.5,
                colorStops: [
                  { offset: 0, color: 'rgba(59,130,246,0.35)' },
                  { offset: 1, color: 'rgba(59,130,246,0.05)' },
                ],
              },
            },
            lineStyle: { color: '#3b82f6', width: 2 },
            itemStyle: { color: '#3b82f6' },
          },
        ],
      },
    ],
  };

  return <EChart option={option} height={380} />;
}

function DimensionCard({ name, score, detail }: { name: string; score: number; detail: string }) {
  const Icon = DIMENSION_ICONS[name] ?? BarChart3;
  const label = DIMENSION_LABELS[name] ?? name;

  return (
    <div className="glass-panel p-4">
      <div className="flex items-center gap-3 mb-2">
        <Icon className="w-4 h-4 shrink-0" style={{ color: scoreColor(score) }} />
        <span className="text-sm font-medium flex-1">{label}</span>
        <span className="text-sm font-bold" style={{ color: scoreColor(score) }}>
          {score.toFixed(1)}
        </span>
      </div>
      {/* Score bar */}
      <div
        className="h-1.5 rounded-full overflow-hidden mb-2"
        style={{ background: 'var(--color-bg-secondary)' }}
      >
        <div
          className="h-full rounded-full transition-all duration-500"
          style={{
            width: `${(score / 10) * 100}%`,
            background: scoreColor(score),
          }}
        />
      </div>
      <p className="text-xs leading-relaxed" style={{ color: 'var(--color-text-secondary)' }}>
        {detail}
      </p>
    </div>
  );
}
