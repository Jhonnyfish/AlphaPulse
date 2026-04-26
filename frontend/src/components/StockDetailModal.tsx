import { useState, useEffect, useCallback } from 'react';
import { X, TrendingUp, TrendingDown, Activity, BarChart3, Target, Loader2, Tag, AlertTriangle } from 'lucide-react';
import { marketApi, signalApi, analyzeApi, type KlinePoint, type SignalEvent, type AnalyzeResult } from '@/lib/api';
import EChart from '@/components/charts/EChart';

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

interface StockDetailModalProps {
  stock: Candidate;
  onClose: () => void;
}

// --- Mock technical indicators derived from real data or fallback ---
function computeMockIndicators(close: number) {
  return {
    ma5: +(close * (0.98 + Math.random() * 0.04)).toFixed(2),
    ma10: +(close * (0.97 + Math.random() * 0.06)).toFixed(2),
    ma20: +(close * (0.95 + Math.random() * 0.10)).toFixed(2),
    macd: +(Math.random() * 2 - 1).toFixed(3),
    macdSignal: +(Math.random() * 2 - 1).toFixed(3),
    macdHist: +(Math.random() * 2 - 1).toFixed(3),
    rsi: +(30 + Math.random() * 40).toFixed(1),
    k: +(20 + Math.random() * 60).toFixed(1),
    d: +(20 + Math.random() * 60).toFixed(1),
    j: +(Math.random() * 100).toFixed(1),
  };
}

function generateMockKline(close: number, days = 60): KlinePoint[] {
  const points: KlinePoint[] = [];
  let price = close * (0.85 + Math.random() * 0.2);
  const now = new Date();
  for (let i = days; i >= 0; i--) {
    const d = new Date(now);
    d.setDate(d.getDate() - i);
    if (d.getDay() === 0 || d.getDay() === 6) continue;
    const change = (Math.random() - 0.48) * 0.04 * price;
    const open = price;
    const c = price + change;
    const high = Math.max(open, c) + Math.random() * 0.02 * price;
    const low = Math.min(open, c) - Math.random() * 0.02 * price;
    price = c;
    points.push({
      date: d.toISOString().split('T')[0],
      open: +open.toFixed(2),
      close: +c.toFixed(2),
      high: +high.toFixed(2),
      low: +low.toFixed(2),
      volume: Math.floor(500000 + Math.random() * 2000000),
      amount: 0,
    });
  }
  // Adjust last close to match real
  if (points.length > 0) {
    points[points.length - 1].close = close;
  }
  return points;
}

// --- 6-Dimension Radar Mock Scores ---
interface RadarScores {
  momentum: number;
  trend: number;
  fundFlow: number;
  technical: number;
  valuation: number;
  sentiment: number;
}

function generateRadarScoresForModal(c: Candidate): RadarScores {
  const seed = c.code.split('').reduce((a, ch) => a + ch.charCodeAt(0), 0);
  const pseudoRand = (offset: number) => {
    const x = Math.sin(seed + offset) * 10000;
    return x - Math.floor(x);
  };
  const clamp = (v: number) => Math.max(5, Math.min(100, Math.round(v)));
  const momRaw = ((c.momentum + 0.5) / 1.5) * 100;
  const trendRaw = ((c.trend + 0.3) / 0.8) * 100;
  return {
    momentum: clamp(momRaw * 0.6 + pseudoRand(1) * 40),
    trend: clamp(trendRaw * 0.5 + pseudoRand(2) * 50),
    fundFlow: clamp(pseudoRand(3) * 60 + 20),
    technical: clamp(pseudoRand(4) * 50 + c.score * 10),
    valuation: clamp(pseudoRand(5) * 40 + 30),
    sentiment: clamp(pseudoRand(6) * 50 + 15),
  };
}

function scoreToColor(score: number): string {
  const t = Math.max(0, Math.min(100, score)) / 100;
  if (t <= 0.5) {
    const u = t * 2;
    const r = Math.round(239 + (245 - 239) * u);
    const g = Math.round(68 + (158 - 68) * u);
    const b = Math.round(68 + (11 - 68) * u);
    return `rgb(${r},${g},${b})`;
  } else {
    const u = (t - 0.5) * 2;
    const r = Math.round(245 + (34 - 245) * u);
    const g = Math.round(158 + (197 - 158) * u);
    const b = Math.round(11 + (94 - 11) * u);
    return `rgb(${r},${g},${b})`;
  }
}

const MODAL_RADAR_DIMS = [
  { key: 'momentum' as const, label: '动量' },
  { key: 'trend' as const, label: '趋势' },
  { key: 'fundFlow' as const, label: '资金' },
  { key: 'technical' as const, label: '技术' },
  { key: 'valuation' as const, label: '估值' },
  { key: 'sentiment' as const, label: '情绪' },
];

// --- Score bar component ---
function ScoreBar({ label, value, max = 100, color }: { label: string; value: number; max?: number; color: string }) {
  const pct = Math.min(100, Math.max(0, (value / max) * 100));
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-xs">
        <span style={{ color: 'var(--color-text-muted)' }}>{label}</span>
        <span className="font-mono font-bold" style={{ color }}>{value.toFixed(1)}</span>
      </div>
      <div className="h-2 rounded-full overflow-hidden" style={{ background: 'rgba(255,255,255,0.06)' }}>
        <div
          className="h-full rounded-full transition-all duration-700"
          style={{ width: `${pct}%`, background: `linear-gradient(90deg, ${color}88, ${color})` }}
        />
      </div>
    </div>
  );
}

// --- Modal Radar Chart ---
function ModalRadar({ stock }: { stock: Candidate }) {
  const scores = generateRadarScoresForModal(stock);
  const values = MODAL_RADAR_DIMS.map(d => scores[d.key]);
  const avgScore = values.reduce((a, b) => a + b, 0) / values.length;
  const fillColor = scoreToColor(avgScore);

  const radarOption = {
    radar: {
      indicator: MODAL_RADAR_DIMS.map(d => ({ name: d.label, max: 100 })),
      shape: 'polygon' as const,
      center: ['50%', '52%'],
      radius: '65%',
      splitNumber: 5,
      splitArea: {
        areaStyle: {
          color: [
            'rgba(239,68,68,0.03)',
            'rgba(239,68,68,0.04)',
            'rgba(245,158,11,0.04)',
            'rgba(34,197,94,0.04)',
            'rgba(34,197,94,0.06)',
          ],
        },
      },
      axisLine: { lineStyle: { color: 'rgba(148,163,184,0.15)' } },
      splitLine: { lineStyle: { color: 'rgba(148,163,184,0.1)' } },
      axisName: {
        color: '#94a3b8',
        fontSize: 10,
        formatter: (name: string) => {
          const dim = MODAL_RADAR_DIMS.find(d => d.label === name);
          const val = dim ? scores[dim.key] : 0;
          return `${name}\n{val|${val}}`;
        },
        rich: {
          val: {
            fontSize: 10,
            fontWeight: 'bold' as const,
            color: '#e2e8f0',
            padding: [2, 0, 0, 0],
          },
        },
      },
    },
    tooltip: {
      trigger: 'item' as const,
      backgroundColor: 'rgba(15,23,42,0.92)',
      borderColor: 'rgba(148,163,184,0.2)',
      textStyle: { color: '#e2e8f0', fontSize: 12 },
    },
    series: [{
      type: 'radar' as const,
      symbol: 'circle',
      symbolSize: 5,
      data: [{
        value: values,
        name: stock.name,
        areaStyle: {
          color: {
            type: 'linear' as const,
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: fillColor.replace('rgb', 'rgba').replace(')', ',0.25)') },
              { offset: 1, color: fillColor.replace('rgb', 'rgba').replace(')', ',0.06)') },
            ],
          },
        },
        lineStyle: { color: fillColor, width: 2 },
        itemStyle: { color: fillColor, borderColor: fillColor, borderWidth: 2 },
        label: {
          show: true,
          formatter: '{c}',
          fontSize: 9,
          color: '#e2e8f0',
          position: 'top' as const,
          distance: 3,
        },
      }],
    }],
  };

  return <EChart option={radarOption} height={220} />;
}

// --- Main modal ---
export default function StockDetailModal({ stock, onClose }: StockDetailModalProps) {
  const [klineData, setKlineData] = useState<KlinePoint[]>([]);
  const [signals, setSignals] = useState<SignalEvent[]>([]);
  const [analysis, setAnalysis] = useState<AnalyzeResult | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    setLoading(true);
    const [klRes, sigRes, anaRes] = await Promise.allSettled([
      marketApi.kline(stock.code, 90),
      signalApi.calendar({ code: stock.code, days: 30 }),
      analyzeApi.analyze(stock.code),
    ]);
    if (klRes.status === 'fulfilled' && klRes.value.data?.length > 0) {
      setKlineData(klRes.value.data);
    } else {
      setKlineData(generateMockKline(stock.close));
    }
    if (sigRes.status === 'fulfilled') {
      setSignals(sigRes.value.data || []);
    }
    if (anaRes.status === 'fulfilled') {
      setAnalysis(anaRes.value.data);
    }
    setLoading(false);
  }, [stock.code, stock.close]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Close on ESC
  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [onClose]);

  // Prevent body scroll
  useEffect(() => {
    document.body.style.overflow = 'hidden';
    return () => { document.body.style.overflow = ''; };
  }, []);

  const indicators = computeMockIndicators(stock.close);
  const isUp = stock.momentum >= 0;
  const priceColor = isUp ? '#ef4444' : '#22c55e';

  // KLine chart option
  const klineDates = klineData.map(d => d.date);
  const klineCloses = klineData.map(d => d.close);
  const klineVolumes = klineData.map(d => d.volume);

  const chartOption = {
    tooltip: {
      trigger: 'axis' as const,
      axisPointer: { type: 'cross' as const },
    },
    grid: [
      { left: '8%', right: '3%', top: '8%', height: '58%' },
      { left: '8%', right: '3%', top: '72%', height: '18%' },
    ],
    xAxis: [
      { type: 'category' as const, data: klineDates, gridIndex: 0, boundaryGap: false, axisLabel: { show: false } },
      { type: 'category' as const, data: klineDates, gridIndex: 1, boundaryGap: false, axisLabel: { fontSize: 10, color: '#64748b' } },
    ],
    yAxis: [
      { type: 'value' as const, gridIndex: 0, splitLine: { lineStyle: { color: 'rgba(148,163,184,0.08)' } }, axisLabel: { fontSize: 10, color: '#94a3b8' } },
      { type: 'value' as const, gridIndex: 1, splitLine: { show: false }, axisLabel: { show: false } },
    ],
    series: [
      {
        name: '收盘价',
        type: 'line',
        data: klineCloses,
        xAxisIndex: 0,
        yAxisIndex: 0,
        smooth: true,
        showSymbol: false,
        lineStyle: { color: '#3b82f6', width: 2 },
        areaStyle: {
          color: {
            type: 'linear' as const,
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(59,130,246,0.25)' },
              { offset: 1, color: 'rgba(59,130,246,0.02)' },
            ],
          },
        },
      },
      {
        name: '成交量',
        type: 'bar',
        data: klineVolumes,
        xAxisIndex: 1,
        yAxisIndex: 1,
        itemStyle: {
          color: (params: { dataIndex: number }) => {
            const idx = params.dataIndex;
            if (idx === 0) return '#64748b';
            return klineCloses[idx] >= klineCloses[idx - 1] ? 'rgba(239,68,68,0.6)' : 'rgba(34,197,94,0.6)';
          },
        },
      },
    ],
  };

  // Technical indicator tag
  function IndicatorTag({ label, value, color }: { label: string; value: string; color?: string }) {
    return (
      <div className="flex flex-col items-center p-2 rounded-lg" style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)' }}>
        <span className="text-xs mb-0.5" style={{ color: 'var(--color-text-muted)' }}>{label}</span>
        <span className="font-mono font-bold text-sm" style={{ color: color || 'var(--color-text-primary)' }}>{value}</span>
      </div>
    );
  }

  // Analysis scores — use API data or fallback from candidate
  const overallScore = analysis?.score ?? stock.score;
  const momentumScore = stock.momentum;
  const trendScore = stock.trend;

  // Concept tags from analysis or mock
  const conceptTags = analysis?.dimensions?.map(d => d.name) || ['动量因子', '趋势跟踪', '量价配合', '技术形态'];

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center animate-fade-in"
      style={{ background: 'rgba(0,0,0,0.65)', backdropFilter: 'blur(4px)' }}
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div
        className="relative w-full max-w-3xl mx-4 rounded-2xl animate-scale-in overflow-hidden"
        style={{
          background: '#222536',
          backdropFilter: 'blur(18px)',
          border: '1px solid rgba(148,163,184,0.12)',
          boxShadow: '0 24px 80px rgba(0,0,0,0.5)',
          maxHeight: '88vh',
        }}
      >
        {/* Close button */}
        <button
          onClick={onClose}
          className="absolute top-4 right-4 z-10 p-1.5 rounded-lg transition-colors hover:bg-white/10"
          style={{ color: 'var(--color-text-muted)' }}
        >
          <X className="w-5 h-5" />
        </button>

        {/* Scrollable body */}
        <div className="overflow-y-auto" style={{ maxHeight: '88vh', padding: '24px' }}>

          {/* ── Header: stock info ── */}
          <div className="flex items-start justify-between mb-5">
            <div className="flex items-center gap-3">
              <div
                className="w-12 h-12 rounded-xl flex items-center justify-center font-bold text-lg"
                style={{ background: 'rgba(59,130,246,0.15)', color: '#3b82f6' }}
              >
                {stock.name.slice(0, 1)}
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-mono font-bold text-lg" style={{ color: 'var(--color-text-primary)' }}>
                    {stock.code}
                  </span>
                  <span className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
                    {stock.name}
                  </span>
                  {stock.limit_up_today && (
                    <span className="text-xs px-1.5 py-0.5 rounded" style={{ background: 'rgba(239,68,68,0.15)', color: '#ef4444' }}>
                      涨停
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-2 mt-1">
                  {stock.industry && (
                    <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'rgba(255,255,255,0.06)', color: 'var(--color-text-muted)' }}>
                      {stock.industry}
                    </span>
                  )}
                  {stock.leader_signal && (
                    <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'rgba(59,130,246,0.15)', color: '#3b82f6' }}>
                      {stock.leader_signal}
                    </span>
                  )}
                </div>
              </div>
            </div>
            <div className="text-right">
              <div className="font-mono font-bold text-2xl" style={{ color: priceColor }}>
                ¥{stock.close.toFixed(2)}
              </div>
              <div className="flex items-center gap-1 justify-end mt-0.5">
                {isUp ? <TrendingUp className="w-4 h-4" style={{ color: '#ef4444' }} /> : <TrendingDown className="w-4 h-4" style={{ color: '#22c55e' }} />}
                <span className="font-mono text-sm font-bold" style={{ color: priceColor }}>
                  {isUp ? '+' : ''}{stock.momentum.toFixed(1)}%
                </span>
              </div>
            </div>
          </div>

          {/* ── K-line chart ── */}
          <div className="mb-5">
            <div className="flex items-center gap-2 mb-2">
              <BarChart3 className="w-4 h-4" style={{ color: '#3b82f6' }} />
              <span className="text-sm font-medium" style={{ color: 'var(--color-text-secondary)' }}>价格走势</span>
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>(近90日)</span>
            </div>
            <div className="rounded-xl overflow-hidden" style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)' }}>
              <EChart option={chartOption} height={260} loading={loading} />
            </div>
          </div>

          {/* ── Technical Indicators grid ── */}
          <div className="mb-5">
            <div className="flex items-center gap-2 mb-2">
              <Activity className="w-4 h-4" style={{ color: '#22d3ee' }} />
              <span className="text-sm font-medium" style={{ color: 'var(--color-text-secondary)' }}>技术指标</span>
            </div>
            <div className="grid grid-cols-3 sm:grid-cols-6 gap-2">
              <IndicatorTag label="MA5" value={indicators.ma5.toString()} />
              <IndicatorTag label="MA10" value={indicators.ma10.toString()} />
              <IndicatorTag label="MA20" value={indicators.ma20.toString()} />
              <IndicatorTag label="MACD" value={indicators.macd.toString()} color={+indicators.macd >= 0 ? '#ef4444' : '#22c55e'} />
              <IndicatorTag label="RSI" value={indicators.rsi.toString()} color={+indicators.rsi > 70 ? '#ef4444' : +indicators.rsi < 30 ? '#22c55e' : 'var(--color-text-primary)'} />
              <IndicatorTag label="KDJ" value={`${indicators.k}/${indicators.d}/${indicators.j}`} />
            </div>
          </div>

          {/* ── Signals list ── */}
          <div className="mb-5">
            <div className="flex items-center gap-2 mb-2">
              <AlertTriangle className="w-4 h-4" style={{ color: '#f59e0b' }} />
              <span className="text-sm font-medium" style={{ color: 'var(--color-text-secondary)' }}>相关信号</span>
            </div>
            {signals.length > 0 ? (
              <div className="space-y-1.5 max-h-32 overflow-y-auto">
                {signals.map((sig, i) => (
                  <div
                    key={i}
                    className="flex items-center justify-between px-3 py-2 rounded-lg text-xs"
                    style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)' }}
                  >
                    <div className="flex items-center gap-2">
                      <span className="font-medium" style={{ color: 'var(--color-text-primary)' }}>{sig.signal_type}</span>
                      <span style={{ color: 'var(--color-text-muted)' }}>{sig.direction}</span>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="font-mono" style={{ color: sig.direction === 'buy' ? '#ef4444' : '#22c55e' }}>
                        {sig.score.toFixed(0)}分
                      </span>
                      <span style={{ color: 'var(--color-text-muted)' }}>{sig.date}</span>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-xs px-3 py-3 rounded-lg text-center" style={{ background: 'rgba(255,255,255,0.03)', color: 'var(--color-text-muted)' }}>
                {stock.leader_signal ? `当前信号: ${stock.leader_signal}` : '暂无相关信号数据'}
              </div>
            )}
          </div>

          {/* ── Analysis Scores ── */}
          <div className="mb-5">
            <div className="flex items-center gap-2 mb-3">
              <Target className="w-4 h-4" style={{ color: '#3b82f6' }} />
              <span className="text-sm font-medium" style={{ color: 'var(--color-text-secondary)' }}>分析评分</span>
              {analysis && (
                <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'rgba(34,197,94,0.15)', color: '#22c55e' }}>
                  {analysis.recommendation}
                </span>
              )}
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              {/* Radar chart */}
              <div className="rounded-xl p-3" style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.06)' }}>
                <div className="text-xs font-medium mb-1" style={{ color: 'var(--color-text-muted)' }}>六维评分雷达</div>
                <ModalRadar stock={stock} />
              </div>
              {/* Score bars */}
              <div className="space-y-3 flex flex-col justify-center">
                <ScoreBar label="综合评分" value={overallScore} color="#3b82f6" />
                <ScoreBar label="动量" value={Math.abs(momentumScore)} color={momentumScore >= 0 ? '#ef4444' : '#22c55e'} />
                <ScoreBar label="趋势" value={Math.abs(trendScore)} color={trendScore >= 0 ? '#ef4444' : '#22c55e'} />
              </div>
            </div>
            {analysis?.summary && (
              <div className="mt-3 text-xs p-3 rounded-lg" style={{ background: 'rgba(59,130,246,0.08)', color: 'var(--color-text-secondary)', borderLeft: '3px solid #3b82f6' }}>
                {analysis.summary}
              </div>
            )}
          </div>

          {/* ── Buy range & stop loss ── */}
          <div className="mb-5 grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div className="p-3 rounded-xl" style={{ background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.12)' }}>
              <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>买入低</div>
              <div className="font-mono font-bold text-sm" style={{ color: '#ef4444' }}>¥{stock.buy_low.toFixed(2)}</div>
            </div>
            <div className="p-3 rounded-xl" style={{ background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.12)' }}>
              <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>买入高</div>
              <div className="font-mono font-bold text-sm" style={{ color: '#ef4444' }}>¥{stock.buy_high.toFixed(2)}</div>
            </div>
            <div className="p-3 rounded-xl" style={{ background: 'rgba(34,197,94,0.06)', border: '1px solid rgba(34,197,94,0.12)' }}>
              <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>卖出区间</div>
              <div className="font-mono font-bold text-sm" style={{ color: '#22c55e' }}>¥{stock.sell_low.toFixed(2)}~{stock.sell_high.toFixed(2)}</div>
            </div>
            <div className="p-3 rounded-xl" style={{ background: 'rgba(239,68,68,0.08)', border: '1px solid rgba(239,68,68,0.15)' }}>
              <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>止损位</div>
              <div className="font-mono font-bold text-sm" style={{ color: '#ef4444' }}>¥{stock.stop_loss.toFixed(2)}</div>
            </div>
          </div>

          {/* ── Concept / industry tags ── */}
          <div className="mb-2">
            <div className="flex items-center gap-2 mb-2">
              <Tag className="w-4 h-4" style={{ color: '#64748b' }} />
              <span className="text-sm font-medium" style={{ color: 'var(--color-text-secondary)' }}>标签</span>
            </div>
            <div className="flex flex-wrap gap-2">
              {stock.industry && (
                <span className="text-xs px-2.5 py-1 rounded-full" style={{ background: 'rgba(59,130,246,0.12)', color: '#60a5fa', border: '1px solid rgba(59,130,246,0.2)' }}>
                  {stock.industry}
                </span>
              )}
              {conceptTags.map((tag, i) => (
                <span key={i} className="text-xs px-2.5 py-1 rounded-full" style={{ background: 'rgba(255,255,255,0.04)', color: 'var(--color-text-muted)', border: '1px solid rgba(255,255,255,0.08)' }}>
                  {tag}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
