import { useState, useEffect, useCallback } from 'react';
import { useView } from '@/lib/ViewContext';
import { analyzeApi, type AnalyzeResult, type ScoreHistoryResponse } from '@/lib/api';

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
  fundamentals?: { verdict: string; score?: number; roe?: number; gross_margin?: number; net_margin?: number; debt_ratio?: number; revenue_growth?: number; net_profit_growth?: number; roe_level?: string; [k: string]: unknown };
  northbound?: { verdict: string; signal?: string; flow_direction?: string; stock_action?: string; [k: string]: unknown };
  margin_detail?: { verdict: string; signal?: string; margin_balance_trend?: string; [k: string]: unknown };
  trend_analysis?: {
    trend_stage: {
      direction: string;
      stage: string;
      strength: number;
      confidence: number;
      signals: string[];
      description: string;
    };
    support_resistance: {
      support1: number;
      support2: number;
      support3: number;
      resistance1: number;
      resistance2: number;
      resistance3: number;
      ma5: number;
      ma10: number;
      ma20: number;
      ma60: number;
      boll_upper: number;
      boll_mid: number;
      boll_lower: number;
      nearest_level: number;
      nearest_type: string;
      distance_pct: number;
    };
    verdict: string;
  };
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
  const dims: { name: string; score: number; detail: string; rawData?: Record<string, unknown> }[] = [
    { name: 'order_flow',   score: levelToScore(raw.order_flow.net_direction as string),   detail: raw.order_flow.verdict, rawData: raw.order_flow as Record<string, unknown> },
    { name: 'volume_price', score: levelToScore(raw.volume_price.turnover_level),          detail: raw.volume_price.verdict, rawData: raw.volume_price as Record<string, unknown> },
    { name: 'valuation',    score: levelToScore(raw.valuation.pe_level),                   detail: raw.valuation.verdict, rawData: raw.valuation as Record<string, unknown> },
    { name: 'volatility',   score: levelToScore(raw.volatility.amplitude_level),           detail: raw.volatility.verdict, rawData: raw.volatility as Record<string, unknown> },
    { name: 'money_flow',   score: levelToScore(raw.money_flow.today_main_direction as string), detail: raw.money_flow.verdict, rawData: raw.money_flow as Record<string, unknown> },
    { name: 'technical',    score: levelToScore(raw.technical.rsi_level as string),        detail: raw.technical.verdict, rawData: raw.technical as Record<string, unknown> },
    { name: 'sector',       score: raw.sector.is_sector_leader ? 7 : 5,                   detail: raw.sector.verdict, rawData: raw.sector as Record<string, unknown> },
    { name: 'sentiment',    score: Math.max(1, Math.min(10, ((strengths.length - risks.length) * 1.5) + 5)), detail: raw.sentiment.verdict, rawData: raw.sentiment as Record<string, unknown> },
    { name: 'fundamentals', score: levelToScore(raw.fundamentals?.roe_level as string),    detail: raw.fundamentals?.verdict ?? '数据不足', rawData: raw.fundamentals as Record<string, unknown> },
    { name: 'northbound',   score: levelToScore(raw.northbound?.flow_direction as string), detail: raw.northbound?.verdict ?? '数据不足', rawData: raw.northbound as Record<string, unknown> },
    { name: 'margin_detail', score: levelToScore(raw.margin_detail?.margin_balance_trend as string), detail: raw.margin_detail?.verdict ?? '数据不足', rawData: raw.margin_detail as Record<string, unknown> },
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
    trend_analysis: raw.trend_analysis,
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
  fundamentals: '基本面',
  northbound: '北向资金',
  margin_detail: '融资融券',
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
  fundamentals: Shield,
  northbound: TrendingUp,
  margin_detail: BarChart3,
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

function formatVolume(vol: number): string {
  if (vol >= 100000000) return `${(vol / 100000000).toFixed(1)}亿`;
  if (vol >= 10000) return `${(vol / 10000).toFixed(1)}万`;
  return vol.toFixed(0);
}

function formatMoney(amount: number): string {
  const abs = Math.abs(amount);
  const sign = amount >= 0 ? '+' : '-';
  if (abs >= 100000000) return `${sign}${(abs / 100000000).toFixed(2)}亿`;
  if (abs >= 10000) return `${sign}${(abs / 10000).toFixed(0)}万`;
  return `${sign}${abs.toFixed(0)}`;
}

function formatMarketValue(mv: number): string {
  if (mv >= 100000000) return `${(mv / 100000000).toFixed(0)}亿`;
  if (mv >= 10000) return `${(mv / 10000).toFixed(0)}万`;
  return mv.toFixed(0);
}

export default function AnalyzePage() {
  const { viewParams, navigate } = useView();
  const code = viewParams.code ?? '';

  const [data, setData] = useState<AnalyzeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [alpha300Open, setAlpha300Open] = useState(false);
  const [scoreHistory, setScoreHistory] = useState<ScoreHistoryResponse | null>(null);

  const fetchAnalysis = useCallback(async (stockCode: string) => {
    setLoading(true);
    setError(null);
    setData(null);
    setScoreHistory(null);
    try {
      const [analysisRes, historyRes] = await Promise.all([
        analyzeApi.analyze(stockCode),
        analyzeApi.scoreHistory(stockCode).catch(() => null),
      ]);
      setData(transformResponse(analysisRes.data as unknown as RawAnalyzeResponse));
      if (historyRes?.data) {
        setScoreHistory(historyRes.data);
      }
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
                <DimensionCard key={dim.name} {...dim} rawData={dim.rawData} />
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

          {/* Trend Analysis */}
          {data.trend_analysis && (
            <TrendAnalysisCard trend={data.trend_analysis} />
          )}

          {/* Score Trend & Comparison */}
          {scoreHistory && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {scoreHistory.trend && (
                <ScoreTrendCard trend={scoreHistory.trend} dimTrends={scoreHistory.dim_trends} />
              )}
              {scoreHistory.comparison && (
                <ComparisonCard comparison={scoreHistory.comparison} />
              )}
            </div>
          )}
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

function DimensionCard({ name, score, detail, rawData }: { name: string; score: number; detail: string; rawData?: Record<string, unknown> }) {
  const Icon = DIMENSION_ICONS[name] ?? BarChart3;
  const label = DIMENSION_LABELS[name] ?? name;

  // Extract key metrics from raw data
  const metrics: { label: string; value: string; color?: string }[] = [];
  if (rawData) {
    if (name === 'order_flow') {
      if (rawData.outer_vol && Number(rawData.outer_vol) > 0) metrics.push({ label: '外盘', value: formatVolume(Number(rawData.outer_vol)) });
      if (rawData.inner_vol && Number(rawData.inner_vol) > 0) metrics.push({ label: '内盘', value: formatVolume(Number(rawData.inner_vol)) });
      if (rawData.outer_ratio && Number(rawData.outer_ratio) > 0) metrics.push({ label: '外盘比', value: `${Number(rawData.outer_ratio).toFixed(1)}%`, color: Number(rawData.outer_ratio) > 55 ? '#ef4444' : undefined });
      if (rawData.net_direction) metrics.push({ label: '方向', value: String(rawData.net_direction), color: rawData.net_direction === '外盘占优' ? '#ef4444' : '#22c55e' });
    } else if (name === 'volume_price') {
      if (rawData.today_change_pct && Number(rawData.today_change_pct) !== 0) metrics.push({ label: '涨跌幅', value: `${Number(rawData.today_change_pct).toFixed(2)}%`, color: Number(rawData.today_change_pct) > 0 ? '#ef4444' : '#22c55e' });
      if (rawData.volume_ratio && Number(rawData.volume_ratio) > 0) metrics.push({ label: '量比', value: Number(rawData.volume_ratio).toFixed(2), color: Number(rawData.volume_ratio) > 1.5 ? '#ef4444' : undefined });
      if (rawData.turnover && Number(rawData.turnover) > 0) metrics.push({ label: '成交额', value: formatMoney(Number(rawData.turnover)) });
      if (rawData.turnover_level) metrics.push({ label: '活跃度', value: String(rawData.turnover_level), color: rawData.turnover_level === '过热' ? '#ef4444' : rawData.turnover_level === '冷清' ? '#22c55e' : undefined });
      if (rawData.price_volume_harmony) metrics.push({ label: '量价', value: String(rawData.price_volume_harmony) });
    } else if (name === 'valuation') {
      if (rawData.pe && Number(rawData.pe) > 0) metrics.push({ label: 'PE', value: Number(rawData.pe).toFixed(1), color: Number(rawData.pe) > 50 ? '#ef4444' : Number(rawData.pe) < 15 ? '#22c55e' : undefined });
      if (rawData.pb && Number(rawData.pb) > 0) metrics.push({ label: 'PB', value: Number(rawData.pb).toFixed(2), color: Number(rawData.pb) > 5 ? '#ef4444' : Number(rawData.pb) < 1 ? '#22c55e' : undefined });
      if (rawData.total_mv && Number(rawData.total_mv) > 0) metrics.push({ label: '市值', value: formatMarketValue(Number(rawData.total_mv)) });
      if (rawData.mv_level) metrics.push({ label: '级别', value: String(rawData.mv_level) });
    } else if (name === 'volatility') {
      if (rawData.amplitude && Number(rawData.amplitude) > 0) metrics.push({ label: '振幅', value: `${Number(rawData.amplitude).toFixed(2)}%`, color: Number(rawData.amplitude) > 5 ? '#ef4444' : undefined });
      if (rawData.distance_to_limit_up && Number(rawData.distance_to_limit_up) > 0) metrics.push({ label: '距涨停', value: `${Number(rawData.distance_to_limit_up).toFixed(1)}%` });
      if (rawData.distance_to_limit_down && Number(rawData.distance_to_limit_down) > 0) metrics.push({ label: '距跌停', value: `${Number(rawData.distance_to_limit_down).toFixed(1)}%` });
    } else if (name === 'money_flow') {
      if (rawData.today_main_net && Number(rawData.today_main_net) !== 0) metrics.push({ label: '主力净流入', value: formatMoney(Number(rawData.today_main_net)), color: Number(rawData.today_main_net) > 0 ? '#ef4444' : '#22c55e' });
      if (rawData.today_huge_net && Number(rawData.today_huge_net) !== 0) metrics.push({ label: '超大单', value: formatMoney(Number(rawData.today_huge_net)), color: Number(rawData.today_huge_net) > 0 ? '#ef4444' : '#22c55e' });
      if (rawData.today_big_net && Number(rawData.today_big_net) !== 0) metrics.push({ label: '大单', value: formatMoney(Number(rawData.today_big_net)), color: Number(rawData.today_big_net) > 0 ? '#ef4444' : '#22c55e' });
      if (rawData.today_main_direction) metrics.push({ label: '主力方向', value: String(rawData.today_main_direction), color: rawData.today_main_direction === '流入' ? '#ef4444' : rawData.today_main_direction === '流出' ? '#22c55e' : undefined });
      if (rawData.main_consecutive_days && Number(rawData.main_consecutive_days) > 0) metrics.push({ label: '连续', value: `${rawData.main_consecutive_days}日${rawData.main_consecutive_direction || ''}` });
      if (rawData.institution_vs_hotmoney) metrics.push({ label: '主导', value: String(rawData.institution_vs_hotmoney) });
      if (rawData.retail_behavior) metrics.push({ label: '散户', value: String(rawData.retail_behavior) });
    } else if (name === 'technical') {
      if (rawData.rsi_14 && Number(rawData.rsi_14) > 0) {
        const rsi = Number(rawData.rsi_14);
        metrics.push({ label: 'RSI', value: rsi.toFixed(1), color: rsi > 70 ? '#ef4444' : rsi < 30 ? '#22c55e' : undefined });
      }
      if (rawData.macd_signal) metrics.push({ label: 'MACD', value: String(rawData.macd_signal), color: rawData.macd_signal === '金叉' ? '#ef4444' : rawData.macd_signal === '死叉' ? '#22c55e' : undefined });
      if (rawData.kdj_signal) metrics.push({ label: 'KDJ', value: String(rawData.kdj_signal) });
      if (rawData.ma_arrangement) metrics.push({ label: '均线', value: String(rawData.ma_arrangement) });
      if (rawData.boll_position) metrics.push({ label: '布林', value: String(rawData.boll_position) });
      if (rawData.weekly_rsi_level) metrics.push({ label: '周RSI', value: String(rawData.weekly_rsi_level), color: rawData.weekly_rsi_level === '超买' ? '#ef4444' : rawData.weekly_rsi_level === '超卖' ? '#22c55e' : undefined });
      if (rawData.period_align) metrics.push({ label: '周期', value: String(rawData.period_align), color: String(rawData.period_align).includes('共振') ? '#ef4444' : undefined });
    } else if (name === 'sector') {
      if (rawData.primary_sector) metrics.push({ label: '板块', value: String(rawData.primary_sector) });
      if (rawData.rel_strength_tag) metrics.push({ label: '相对强度', value: String(rawData.rel_strength_tag), color: rawData.rel_strength_tag === '强于板块' ? '#ef4444' : rawData.rel_strength_tag === '弱于板块' ? '#22c55e' : undefined });
      if (rawData.stock_pct_chg_5d && Number(rawData.stock_pct_chg_5d) !== 0) metrics.push({ label: '个股5日', value: `${Number(rawData.stock_pct_chg_5d).toFixed(2)}%`, color: Number(rawData.stock_pct_chg_5d) > 0 ? '#ef4444' : '#22c55e' });
      if (rawData.sector_pct_chg_5d && Number(rawData.sector_pct_chg_5d) !== 0) metrics.push({ label: '板块5日', value: `${Number(rawData.sector_pct_chg_5d).toFixed(2)}%`, color: Number(rawData.sector_pct_chg_5d) > 0 ? '#ef4444' : '#22c55e' });
    } else if (name === 'sentiment') {
      if (rawData.news_count && Number(rawData.news_count) > 0) metrics.push({ label: '新闻', value: `${rawData.news_count}条` });
      if (rawData.announcement_count && Number(rawData.announcement_count) > 0) metrics.push({ label: '公告', value: `${rawData.announcement_count}条` });
      if (rawData.sentiment_score && Number(rawData.sentiment_score) !== 0) metrics.push({ label: '情绪分', value: Number(rawData.sentiment_score).toFixed(1), color: Number(rawData.sentiment_score) > 0 ? '#ef4444' : '#22c55e' });
      if (rawData.sentiment_label) metrics.push({ label: '情绪', value: String(rawData.sentiment_label) });
    } else if (name === 'fundamentals') {
      if (rawData.roe && Number(rawData.roe) > 0) metrics.push({ label: 'ROE', value: `${Number(rawData.roe).toFixed(1)}%`, color: Number(rawData.roe) > 15 ? '#ef4444' : Number(rawData.roe) < 5 ? '#22c55e' : undefined });
      if (rawData.gross_margin && Number(rawData.gross_margin) > 0) metrics.push({ label: '毛利率', value: `${Number(rawData.gross_margin).toFixed(1)}%` });
      if (rawData.net_margin && Number(rawData.net_margin) > 0) metrics.push({ label: '净利率', value: `${Number(rawData.net_margin).toFixed(1)}%` });
      if (rawData.debt_ratio && Number(rawData.debt_ratio) > 0) metrics.push({ label: '负债率', value: `${Number(rawData.debt_ratio).toFixed(1)}%`, color: Number(rawData.debt_ratio) > 60 ? '#ef4444' : undefined });
      if (rawData.revenue_growth && Number(rawData.revenue_growth) !== 0) metrics.push({ label: '营收增长', value: `${Number(rawData.revenue_growth).toFixed(1)}%`, color: Number(rawData.revenue_growth) > 0 ? '#ef4444' : '#22c55e' });
      if (rawData.net_profit_growth && Number(rawData.net_profit_growth) !== 0) metrics.push({ label: '利润增长', value: `${Number(rawData.net_profit_growth).toFixed(1)}%`, color: Number(rawData.net_profit_growth) > 0 ? '#ef4444' : '#22c55e' });
    } else if (name === 'northbound') {
      if (rawData.latest_net_flow && Number(rawData.latest_net_flow) !== 0) metrics.push({ label: '净流入', value: formatMoney(Number(rawData.latest_net_flow)), color: Number(rawData.latest_net_flow) > 0 ? '#ef4444' : '#22c55e' });
      if (rawData.stock_net_amount && Number(rawData.stock_net_amount) !== 0) metrics.push({ label: '个股净买', value: formatMoney(Number(rawData.stock_net_amount)), color: Number(rawData.stock_net_amount) > 0 ? '#ef4444' : '#22c55e' });
      if (rawData.trend_5d && Number(rawData.trend_5d) !== 0) metrics.push({ label: '5日趋势', value: formatMoney(Number(rawData.trend_5d)), color: Number(rawData.trend_5d) > 0 ? '#ef4444' : '#22c55e' });
      if (rawData.flow_direction) metrics.push({ label: '方向', value: String(rawData.flow_direction), color: rawData.flow_direction === '流入' ? '#ef4444' : '#22c55e' });
      if (rawData.stock_action) metrics.push({ label: '操作', value: String(rawData.stock_action) });
    } else if (name === 'margin_detail') {
      if (rawData.latest_margin_balance && Number(rawData.latest_margin_balance) > 0) metrics.push({ label: '融资余额', value: formatMoney(Number(rawData.latest_margin_balance)) });
      if (rawData.margin_balance_trend) metrics.push({ label: '余额趋势', value: String(rawData.margin_balance_trend), color: rawData.margin_balance_trend === '上升' ? '#ef4444' : rawData.margin_balance_trend === '下降' ? '#22c55e' : undefined });
      if (rawData.margin_buying_trend) metrics.push({ label: '融资买入', value: String(rawData.margin_buying_trend) });
      if (rawData.short_selling_trend) metrics.push({ label: '融券卖出', value: String(rawData.short_selling_trend) });
      if (rawData.signal) metrics.push({ label: '信号', value: String(rawData.signal) });
    }
  }

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
      {/* Key metrics */}
      {metrics.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-2">
          {metrics.map((m, i) => (
            <span key={i} className="text-xs px-1.5 py-0.5 rounded font-mono" style={{ background: 'var(--color-bg-hover)', color: m.color || 'var(--color-text-secondary)' }}>
              {m.label}: {m.value}
            </span>
          ))}
        </div>
      )}
      <p className="text-xs leading-relaxed" style={{ color: 'var(--color-text-secondary)' }}>
        {detail}
      </p>
    </div>
  );
}

function TrendAnalysisCard({ trend }: { trend: NonNullable<AnalyzeResult['trend_analysis']> }) {
  const { trend_stage: stage, support_resistance: sr, verdict } = trend;

  const directionColor = stage.direction === '上升' ? '#ef4444' : stage.direction === '下降' ? '#22c55e' : '#f59e0b';
  const stageColor = stage.stage === '早期' ? '#3b82f6' : stage.stage === '中期' ? '#f59e0b' : stage.stage === '末期' ? '#ef4444' : '#94a3b8';

  return (
    <div className="glass-panel p-5">
      <h3 className="text-sm font-medium mb-4" style={{ color: 'var(--color-text-secondary)' }}>
        趋势分析
      </h3>

      {/* Trend stage */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
        <div className="text-center">
          <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>方向</div>
          <div className="text-lg font-bold" style={{ color: directionColor }}>
            {stage.direction}
          </div>
        </div>
        <div className="text-center">
          <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>阶段</div>
          <div className="text-lg font-bold" style={{ color: stageColor }}>
            {stage.stage}
          </div>
        </div>
        <div className="text-center">
          <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>强度</div>
          <div className="text-lg font-bold" style={{ color: 'var(--color-text-primary)' }}>
            {stage.strength.toFixed(0)}
          </div>
        </div>
        <div className="text-center">
          <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>置信度</div>
          <div className="text-lg font-bold" style={{
            color: stage.confidence >= 70 ? '#22c55e' : stage.confidence >= 40 ? '#f59e0b' : '#ef4444'
          }}>
            {stage.confidence.toFixed(0)}%
          </div>
        </div>
      </div>

      {/* Description */}
      <p className="text-sm mb-4" style={{ color: 'var(--color-text-secondary)' }}>
        {stage.description}
      </p>

      {/* Support/Resistance levels */}
      <div className="grid grid-cols-2 gap-4 mb-4">
        {/* Resistance */}
        <div>
          <div className="text-xs font-medium mb-2" style={{ color: '#ef4444' }}>阻力位</div>
          <div className="space-y-1">
            {sr.resistance1 > 0 && (
              <div className="flex justify-between text-xs">
                <span style={{ color: 'var(--color-text-muted)' }}>R1</span>
                <span className="font-mono" style={{ color: 'var(--color-text-primary)' }}>{sr.resistance1.toFixed(2)}</span>
              </div>
            )}
            {sr.resistance2 > 0 && (
              <div className="flex justify-between text-xs">
                <span style={{ color: 'var(--color-text-muted)' }}>R2</span>
                <span className="font-mono" style={{ color: 'var(--color-text-primary)' }}>{sr.resistance2.toFixed(2)}</span>
              </div>
            )}
            {sr.resistance3 > 0 && (
              <div className="flex justify-between text-xs">
                <span style={{ color: 'var(--color-text-muted)' }}>R3</span>
                <span className="font-mono" style={{ color: 'var(--color-text-primary)' }}>{sr.resistance3.toFixed(2)}</span>
              </div>
            )}
          </div>
        </div>
        {/* Support */}
        <div>
          <div className="text-xs font-medium mb-2" style={{ color: '#22c55e' }}>支撑位</div>
          <div className="space-y-1">
            {sr.support1 > 0 && (
              <div className="flex justify-between text-xs">
                <span style={{ color: 'var(--color-text-muted)' }}>S1</span>
                <span className="font-mono" style={{ color: 'var(--color-text-primary)' }}>{sr.support1.toFixed(2)}</span>
              </div>
            )}
            {sr.support2 > 0 && (
              <div className="flex justify-between text-xs">
                <span style={{ color: 'var(--color-text-muted)' }}>S2</span>
                <span className="font-mono" style={{ color: 'var(--color-text-primary)' }}>{sr.support2.toFixed(2)}</span>
              </div>
            )}
            {sr.support3 > 0 && (
              <div className="flex justify-between text-xs">
                <span style={{ color: 'var(--color-text-muted)' }}>S3</span>
                <span className="font-mono" style={{ color: 'var(--color-text-primary)' }}>{sr.support3.toFixed(2)}</span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Key MAs */}
      <div className="flex flex-wrap gap-3 mb-3">
        <span className="text-xs px-2 py-1 rounded" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}>
          MA5: {sr.ma5.toFixed(2)}
        </span>
        <span className="text-xs px-2 py-1 rounded" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}>
          MA10: {sr.ma10.toFixed(2)}
        </span>
        <span className="text-xs px-2 py-1 rounded" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}>
          MA20: {sr.ma20.toFixed(2)}
        </span>
        <span className="text-xs px-2 py-1 rounded" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}>
          MA60: {sr.ma60.toFixed(2)}
        </span>
      </div>

      {/* Nearest level */}
      {sr.nearest_level > 0 && (
        <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
          距最近{sr.nearest_type}位 <span className="font-mono font-medium" style={{ color: 'var(--color-text-primary)' }}>{sr.nearest_level.toFixed(2)}</span> 约 <span className="font-mono" style={{ color: sr.distance_pct < 3 ? '#ef4444' : 'var(--color-text-secondary)' }}>{sr.distance_pct.toFixed(1)}%</span>
        </div>
      )}

      {/* Verdict */}
      <div className="mt-3 text-sm font-medium" style={{ color: 'var(--color-accent)' }}>
        {verdict}
      </div>
    </div>
  );
}

function ScoreTrendCard({ trend, dimTrends }: {
  trend: NonNullable<ScoreHistoryResponse['trend']>;
  dimTrends?: ScoreHistoryResponse['dim_trends'];
}) {
  const trendColor = (change: number) => change > 0 ? '#ef4444' : change < 0 ? '#22c55e' : 'var(--color-text-muted)';
  const trendArrow = (change: number) => change > 0 ? '↑' : change < 0 ? '↓' : '→';
  const trendLabel = (t: string) => t === 'rising' ? '上升' : t === 'falling' ? '下降' : '稳定';

  return (
    <div className="glass-panel p-5">
      <h3 className="text-sm font-medium mb-4" style={{ color: 'var(--color-text-secondary)' }}>
        评分趋势
      </h3>

      {/* Score changes */}
      <div className="grid grid-cols-3 gap-4 mb-4">
        <div className="text-center">
          <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>当前</div>
          <div className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
            {trend.current.toFixed(0)}
          </div>
        </div>
        <div className="text-center">
          <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>7日变化</div>
          <div className="text-lg font-bold" style={{ color: trendColor(trend.change_7d) }}>
            {trendArrow(trend.change_7d)} {Math.abs(trend.change_7d).toFixed(1)}
          </div>
          <div className="text-xs" style={{ color: trendColor(trend.change_7d) }}>
            {trendLabel(trend.trend_7d)}
          </div>
        </div>
        <div className="text-center">
          <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>30日变化</div>
          <div className="text-lg font-bold" style={{ color: trendColor(trend.change_30d) }}>
            {trendArrow(trend.change_30d)} {Math.abs(trend.change_30d).toFixed(1)}
          </div>
          <div className="text-xs" style={{ color: trendColor(trend.change_30d) }}>
            {trendLabel(trend.trend_30d)}
          </div>
        </div>
      </div>

      {/* Dimension trends */}
      {dimTrends && Object.keys(dimTrends).length > 0 && (
        <div>
          <div className="text-xs font-medium mb-2" style={{ color: 'var(--color-text-muted)' }}>维度变化(7日)</div>
          <div className="flex flex-wrap gap-2">
            {Object.entries(dimTrends).map(([dim, dt]) => (
              <span key={dim} className="text-xs px-2 py-1 rounded" style={{
                background: dt.change_7d > 5 ? 'rgba(239,68,68,0.1)' : dt.change_7d < -5 ? 'rgba(34,197,94,0.1)' : 'var(--color-bg-hover)',
                color: trendColor(dt.change_7d),
              }}>
                {DIMENSION_LABELS[dim] ?? dim}: {dt.change_7d > 0 ? '+' : ''}{dt.change_7d.toFixed(1)}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function ComparisonCard({ comparison }: { comparison: NonNullable<ScoreHistoryResponse['comparison']> }) {
  const relColor = (v: number) => v > 0 ? '#ef4444' : v < 0 ? '#22c55e' : 'var(--color-text-muted)';

  return (
    <div className="glass-panel p-5">
      <h3 className="text-sm font-medium mb-4" style={{ color: 'var(--color-text-secondary)' }}>
        对比分析
      </h3>

      {/* Score comparison */}
      <div className="grid grid-cols-2 gap-4 mb-4">
        <div>
          <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>个股评分</div>
          <div className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
            {comparison.stock_score.toFixed(0)}
          </div>
        </div>
        <div>
          <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>
            {comparison.sector_name || '板块'}平均
          </div>
          <div className="text-2xl font-bold" style={{ color: 'var(--color-text-secondary)' }}>
            {comparison.sector_avg_score.toFixed(0)}
          </div>
        </div>
      </div>

      {/* Price change comparison */}
      <div className="space-y-3">
        <div className="flex justify-between items-center">
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>5日涨跌</span>
          <div className="flex items-center gap-3">
            <span className="text-xs font-mono" style={{ color: comparison.stock_change_5d > 0 ? '#ef4444' : '#22c55e' }}>
              {comparison.stock_change_5d > 0 ? '+' : ''}{comparison.stock_change_5d.toFixed(2)}%
            </span>
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>vs</span>
            <span className="text-xs font-mono" style={{ color: 'var(--color-text-secondary)' }}>
              沪深300 {comparison.market_change_5d > 0 ? '+' : ''}{comparison.market_change_5d.toFixed(2)}%
            </span>
          </div>
        </div>

        <div className="flex justify-between items-center">
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>20日涨跌</span>
          <div className="flex items-center gap-3">
            <span className="text-xs font-mono" style={{ color: comparison.stock_change_20d > 0 ? '#ef4444' : '#22c55e' }}>
              {comparison.stock_change_20d > 0 ? '+' : ''}{comparison.stock_change_20d.toFixed(2)}%
            </span>
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>vs</span>
            <span className="text-xs font-mono" style={{ color: 'var(--color-text-secondary)' }}>
              沪深300 {comparison.market_change_20d > 0 ? '+' : ''}{comparison.market_change_20d.toFixed(2)}%
            </span>
          </div>
        </div>
      </div>

      {/* Relative strength */}
      <div className="mt-4 pt-3" style={{ borderTop: '1px solid var(--color-border)' }}>
        <div className="flex justify-between">
          <div className="text-center flex-1">
            <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>vs板块</div>
            <div className="text-sm font-bold" style={{ color: relColor(comparison.vs_sector) }}>
              {comparison.vs_sector > 0 ? '+' : ''}{comparison.vs_sector.toFixed(2)}%
            </div>
          </div>
          <div className="text-center flex-1">
            <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>vs大盘</div>
            <div className="text-sm font-bold" style={{ color: relColor(comparison.vs_market) }}>
              {comparison.vs_market > 0 ? '+' : ''}{comparison.vs_market.toFixed(2)}%
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
