import { useState, useEffect, useCallback } from 'react';
import { useView } from '@/lib/ViewContext';
import { analyzeApi, type AnalyzeResult } from '@/lib/api';
import EChart from '@/components/charts/EChart';
import StockSearch from '@/components/StockSearch';
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

  const fetchAnalysis = useCallback(async (stockCode: string) => {
    setLoading(true);
    setError(null);
    setData(null);
    try {
      const res = await analyzeApi.analyze(stockCode);
      setData(res.data);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '分析失败，请稍后重试';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (code) {
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
          <StockSearch onSelect={handleSelect} autoFocus className="w-full" />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4 max-w-7xl mx-auto">
      {/* Search bar */}
      <div className="flex items-center gap-4">
        <StockSearch onSelect={handleSelect} className="w-72" placeholder="切换股票..." />
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
