import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { useView } from '@/lib/ViewContext';
import {
  createChart,
  CandlestickSeries,
  HistogramSeries,
  LineSeries,
  type IChartApi,
  type ISeriesApi,
} from 'lightweight-charts';
import type { EChartsOption } from 'echarts';
import ReactECharts from 'echarts-for-react';
import { marketApi, type KlinePoint, type Quote, type SearchSuggestion } from '@/lib/api';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import StockSearch from '@/components/StockSearch';
import Alpha300Selector from '@/components/Alpha300Selector';
import ErrorState from '@/components/ErrorState';
import { calcMA, calcMACD, calcKDJ, calcRSI } from '@/lib/indicators';

/* ── Constants ──────────────────────────────────────────────────── */

const PERIODS = [
  { value: 'daily', label: '日K', days: 120 },
  { value: 'weekly', label: '周K', days: 365 },
  { value: 'monthly', label: '月K', days: 730 },
] as const;

/** Main-chart (overlay) indicator ids */
type OverlayIndicator = 'MA';
/** Sub-chart (pane) indicator ids */
type SubIndicator = 'MACD' | 'KDJ' | 'RSI';
type IndicatorId = OverlayIndicator | SubIndicator;

const SUB_INDICATORS: { id: SubIndicator; label: string }[] = [
  { id: 'MACD', label: 'MACD' },
  { id: 'KDJ', label: 'KDJ' },
  { id: 'RSI', label: 'RSI' },
];

// MA line colors
const MA_COLORS: Record<string, string> = {
  MA5: '#f59e0b',   // amber
  MA10: '#3b82f6',  // blue
  MA20: '#a855f7',  // purple
  MA60: '#22d3ee',  // cyan
};

// MACD / KDJ / RSI palette
const CHART_COLORS = {
  dif: '#f59e0b',
  dea: '#3b82f6',
  macdPos: 'rgba(239,68,68,0.7)',
  macdNeg: 'rgba(34,197,94,0.7)',
  kLine: '#f59e0b',
  dLine: '#3b82f6',
  jLine: '#a855f7',
  rsi6: '#f59e0b',
  rsi12: '#3b82f6',
  rsi24: '#a855f7',
};

/* ── Component ──────────────────────────────────────────────────── */

export default function KlinePage() {
  const { viewParams, navigate } = useView();
  const [code, setCode] = useState(viewParams.code || '');
  const [period, setPeriod] = useState('daily');
  const [quote, setQuote] = useState<Quote | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Indicator toggles
  const [showMA, setShowMA] = useState(false);
  const [activeSub, setActiveSub] = useState<SubIndicator | null>(null);
  const [alpha300Open, setAlpha300Open] = useState(false);

  // Data cache for indicators (raw kline points)
  const klinePointsRef = useRef<KlinePoint[]>([]);

  // Lightweight-charts refs
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const maSeriesRef = useRef<Record<string, ISeriesApi<'Line'>>>({});

  /* ── Load Kline data ─────────────────────────────────────────── */

  const loadKline = useCallback(async (stockCode: string, p: string) => {
    if (!stockCode.trim()) return;
    setLoading(true);
    setError('');

    try {
      const [klineRes, quoteRes] = await Promise.all([
        marketApi.kline(stockCode, PERIODS.find((pp) => pp.value === p)?.days ?? 120),
        marketApi.quote(stockCode),
      ]);

      const points: KlinePoint[] = klineRes.data;
      klinePointsRef.current = points;
      setQuote(quoteRes.data);

      if (!chartContainerRef.current) return;

      // Clear old chart
      if (chartRef.current) {
        chartRef.current.remove();
        chartRef.current = null;
      }
      maSeriesRef.current = {};

      const chart = createChart(chartContainerRef.current, {
        width: chartContainerRef.current.clientWidth,
        height: 420,
        layout: {
          background: { color: '#1a1d27' },
          textColor: '#8b8fa3',
        },
        grid: {
          vertLines: { color: '#2a2d3e' },
          horzLines: { color: '#2a2d3e' },
        },
        crosshair: { mode: 0 },
        timeScale: {
          timeVisible: false,
          borderColor: '#2a2d3e',
        },
        rightPriceScale: {
          borderColor: '#2a2d3e',
        },
      });
      chartRef.current = chart;

      // Candlestick series
      const candleSeries = chart.addSeries(CandlestickSeries, {
        upColor: '#ef4444',
        downColor: '#22c55e',
        borderUpColor: '#ef4444',
        borderDownColor: '#22c55e',
        wickUpColor: '#ef4444',
        wickDownColor: '#22c55e',
      });
      candleSeries.setData(
        points.map((pt) => ({
          time: pt.date as string,
          open: pt.open,
          high: pt.high,
          low: pt.low,
          close: pt.close,
        })),
      );

      // Volume series
      const volumeSeries = chart.addSeries(HistogramSeries, {
        priceFormat: { type: 'volume' },
        priceScaleId: 'volume',
      });
      chart.priceScale('volume').applyOptions({
        scaleMargins: { top: 0.8, bottom: 0 },
      });
      volumeSeries.setData(
        points.map((pt) => ({
          time: pt.date as string,
          value: pt.volume,
          color: pt.close >= pt.open ? 'rgba(239,68,68,0.4)' : 'rgba(34,197,94,0.4)',
        })),
      );

      chart.timeScale().fitContent();
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        '加载K线数据失败';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  /* ── Sync MA overlays when showMA changes ────────────────────── */

  useEffect(() => {
    const chart = chartRef.current;
    const points = klinePointsRef.current;
    if (!chart || !points.length) return;

    if (showMA) {
      const closes = points.map((p) => p.close);
      const periods = [5, 10, 20, 60];

      periods.forEach((per) => {
        const key = `MA${per}`;
        if (!maSeriesRef.current[key]) {
          const series = chart.addSeries(LineSeries, {
            color: MA_COLORS[key],
            lineWidth: 1,
            priceLineVisible: false,
            lastValueVisible: false,
            crosshairMarkerVisible: false,
          });
          const maData = calcMA(closes, per);
          series.setData(
            points
              .map((pt, i) => ({
                time: pt.date as string,
                value: maData[i] ?? undefined,
              }))
              .filter((d) => d.value != null),
          );
          maSeriesRef.current[key] = series;
        }
      });
    } else {
      // Remove all MA series
      Object.values(maSeriesRef.current).forEach((series) => {
        chart.removeSeries(series);
      });
      maSeriesRef.current = {};
    }
  }, [showMA]);

  /* ── Eagerly rebuild MA on data change if showMA is on ──────── */

  useEffect(() => {
    // After loadKline, if showMA is already on, re-trigger MA overlay
    if (showMA && klinePointsRef.current.length > 0 && chartRef.current) {
      // Force re-render by toggling off then on
      setShowMA(false);
      // Use micro-task to toggle back
      const t = setTimeout(() => setShowMA(true), 0);
      return () => clearTimeout(t);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [klinePointsRef.current.length]);

  /* ── Sub-chart ECharts options ───────────────────────────────── */

  const subChartOption = useMemo<EChartsOption | null>(() => {
    if (!activeSub) return null;

    const points = klinePointsRef.current;
    if (!points.length) return null;

    const dates = points.map((p) => p.date);
    const closes = points.map((p) => p.close);
    const highs = points.map((p) => p.high);
    const lows = points.map((p) => p.low);

    const gridConfig = {
      left: 60,
      right: 20,
      top: 24,
      bottom: 28,
    };

    const baseAxis = {
      type: 'category' as const,
      data: dates,
      axisLine: { lineStyle: { color: '#2a2d3e' } },
      axisTick: { show: false },
      axisLabel: { show: false },
      splitLine: { show: false },
    };

    const yAxisBase = {
      type: 'value' as const,
      splitNumber: 3,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: '#64748b', fontSize: 10 },
      splitLine: { lineStyle: { color: '#2a2d3e' } },
    };

    if (activeSub === 'MACD') {
      const { dif, dea, macd } = calcMACD(closes);
      return {
        animation: false,
        tooltip: {
          trigger: 'axis',
          backgroundColor: 'rgba(15,23,42,0.92)',
          borderColor: 'rgba(148,163,184,0.15)',
          textStyle: { color: '#e2e8f0', fontSize: 11 },
          axisPointer: { type: 'cross', lineStyle: { color: '#475569' } },
        },
        grid: gridConfig,
        xAxis: baseAxis,
        yAxis: yAxisBase,
        series: [
          {
            name: 'DIF',
            type: 'line',
            data: dif,
            lineStyle: { width: 1, color: CHART_COLORS.dif },
            itemStyle: { color: CHART_COLORS.dif },
            symbol: 'none',
          },
          {
            name: 'DEA',
            type: 'line',
            data: dea,
            lineStyle: { width: 1, color: CHART_COLORS.dea },
            itemStyle: { color: CHART_COLORS.dea },
            symbol: 'none',
          },
          {
            name: 'MACD',
            type: 'bar',
            data: macd,
            itemStyle: {
              color: (params: { value: number }) =>
                params.value >= 0 ? CHART_COLORS.macdPos : CHART_COLORS.macdNeg,
            },
          },
        ],
      };
    }

    if (activeSub === 'KDJ') {
      const { k, d, j } = calcKDJ(highs, lows, closes);
      return {
        animation: false,
        tooltip: {
          trigger: 'axis',
          backgroundColor: 'rgba(15,23,42,0.92)',
          borderColor: 'rgba(148,163,184,0.15)',
          textStyle: { color: '#e2e8f0', fontSize: 11 },
          axisPointer: { type: 'cross', lineStyle: { color: '#475569' } },
        },
        grid: gridConfig,
        xAxis: baseAxis,
        yAxis: { ...yAxisBase, min: 0, max: 100 },
        series: [
          {
            name: 'K',
            type: 'line',
            data: k,
            lineStyle: { width: 1, color: CHART_COLORS.kLine },
            itemStyle: { color: CHART_COLORS.kLine },
            symbol: 'none',
          },
          {
            name: 'D',
            type: 'line',
            data: d,
            lineStyle: { width: 1, color: CHART_COLORS.dLine },
            itemStyle: { color: CHART_COLORS.dLine },
            symbol: 'none',
          },
          {
            name: 'J',
            type: 'line',
            data: j,
            lineStyle: { width: 1, color: CHART_COLORS.jLine },
            itemStyle: { color: CHART_COLORS.jLine },
            symbol: 'none',
          },
        ],
      };
    }

    if (activeSub === 'RSI') {
      const rsi6 = calcRSI(closes, 6);
      const rsi12 = calcRSI(closes, 12);
      const rsi24 = calcRSI(closes, 24);
      return {
        animation: false,
        tooltip: {
          trigger: 'axis',
          backgroundColor: 'rgba(15,23,42,0.92)',
          borderColor: 'rgba(148,163,184,0.15)',
          textStyle: { color: '#e2e8f0', fontSize: 11 },
          axisPointer: { type: 'cross', lineStyle: { color: '#475569' } },
        },
        grid: gridConfig,
        xAxis: baseAxis,
        yAxis: { ...yAxisBase, min: 0, max: 100 },
        series: [
          {
            name: 'RSI6',
            type: 'line',
            data: rsi6,
            lineStyle: { width: 1, color: CHART_COLORS.rsi6 },
            itemStyle: { color: CHART_COLORS.rsi6 },
            symbol: 'none',
          },
          {
            name: 'RSI12',
            type: 'line',
            data: rsi12,
            lineStyle: { width: 1, color: CHART_COLORS.rsi12 },
            itemStyle: { color: CHART_COLORS.rsi12 },
            symbol: 'none',
          },
          {
            name: 'RSI24',
            type: 'line',
            data: rsi24,
            lineStyle: { width: 1, color: CHART_COLORS.rsi24 },
            itemStyle: { color: CHART_COLORS.rsi24 },
            symbol: 'none',
          },
          // Overbought / oversold reference lines
          {
            name: '',
            type: 'line',
            data: closes.map(() => 70),
            lineStyle: { width: 1, color: 'rgba(239,68,68,0.3)', type: 'dashed' },
            itemStyle: { color: 'transparent' },
            symbol: 'none',
            silent: true,
          },
          {
            name: '',
            type: 'line',
            data: closes.map(() => 30),
            lineStyle: { width: 1, color: 'rgba(34,197,94,0.3)', type: 'dashed' },
            itemStyle: { color: 'transparent' },
            symbol: 'none',
            silent: true,
          },
        ],
      };
    }

    return null;
  }, [activeSub, klinePointsRef.current.length]); // eslint-disable-line react-hooks/exhaustive-deps

  /* ── Sub-chart legend (inline above chart) ───────────────────── */

  const subChartLegend = useMemo(() => {
    if (!activeSub) return null;
    if (activeSub === 'MACD') {
      return [
        { label: 'DIF', color: CHART_COLORS.dif },
        { label: 'DEA', color: CHART_COLORS.dea },
        { label: 'MACD', color: CHART_COLORS.macdPos },
      ];
    }
    if (activeSub === 'KDJ') {
      return [
        { label: 'K', color: CHART_COLORS.kLine },
        { label: 'D', color: CHART_COLORS.dLine },
        { label: 'J', color: CHART_COLORS.jLine },
      ];
    }
    if (activeSub === 'RSI') {
      return [
        { label: 'RSI6', color: CHART_COLORS.rsi6 },
        { label: 'RSI12', color: CHART_COLORS.rsi12 },
        { label: 'RSI24', color: CHART_COLORS.rsi24 },
      ];
    }
    return null;
  }, [activeSub]);

  /* ── Load when code or period changes ────────────────────────── */

  useEffect(() => {
    if (code) {
      loadKline(code, period);
    }
    return () => {
      if (chartRef.current) {
        chartRef.current.remove();
        chartRef.current = null;
      }
      maSeriesRef.current = {};
    };
  }, [code, period, loadKline]);

  /* ── Resize handler ──────────────────────────────────────────── */

  useEffect(() => {
    const handleResize = () => {
      if (chartRef.current && chartContainerRef.current) {
        chartRef.current.applyOptions({
          width: chartContainerRef.current.clientWidth,
        });
      }
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  /* ── Handlers ────────────────────────────────────────────────── */

  const handleSearch = (suggestion: SearchSuggestion) => {
    setCode(suggestion.code);
    navigate('kline', { code: suggestion.code });
  };

  const toggleSubIndicator = (id: SubIndicator) => {
    setActiveSub((prev) => (prev === id ? null : id));
  };

  /* ── Derived values ──────────────────────────────────────────── */

  const pct = quote?.change_percent ?? 0;
  const color =
    pct > 0 ? 'var(--color-danger)' : pct < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';
  const TrendIcon = pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Minus;

  const chartHasData = code && !loading && klinePointsRef.current.length > 0;

  /* ── Render ──────────────────────────────────────────────────── */

  return (
    <div>
      <h1 className="text-xl font-bold mb-6">K线图</h1>

      {/* Search with autocomplete */}
      <div className="flex gap-2 mb-4">
        <div className="flex-1 max-w-sm flex items-center gap-2">
          <StockSearch
            onSelect={handleSearch}
            placeholder="搜索股票代码或名称..."
          />
          <button
            type="button"
            onClick={() => setAlpha300Open(true)}
            className="px-2.5 py-2 rounded-lg text-sm shrink-0 transition-colors hover:bg-[var(--color-bg-hover)]"
            style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', color: 'var(--color-text-secondary)' }}
            title="从 Alpha300 选择"
          >
            🎯
          </button>
        </div>
      </div>

      {/* Period selector */}
      {code && (
        <div className="flex gap-1 mb-4">
          {PERIODS.map((p) => (
            <button
              key={p.value}
              onClick={() => setPeriod(p.value)}
              className="px-3 py-1.5 rounded-lg text-sm transition-colors"
              style={{
                background: period === p.value ? 'var(--color-accent)' : 'var(--color-bg-card)',
                color: period === p.value ? '#fff' : 'var(--color-text-secondary)',
              }}
            >
              {p.label}
            </button>
          ))}
        </div>
      )}

      {/* Technical indicator toggle bar */}
      {code && (
        <div className="flex items-center gap-2 mb-3">
          {/* MA toggle — overlay indicator */}
          <IndicatorButton
            label="MA"
            active={showMA}
            onClick={() => setShowMA((v) => !v)}
          />

          <span
            className="mx-1 h-4 w-px"
            style={{ background: 'var(--color-border)' }}
          />

          {/* Sub-chart indicator toggles */}
          {SUB_INDICATORS.map((ind) => (
            <IndicatorButton
              key={ind.id}
              label={ind.label}
              active={activeSub === ind.id}
              onClick={() => toggleSubIndicator(ind.id)}
            />
          ))}

          {/* Inline legend for sub-chart */}
          {subChartLegend && (
            <div className="flex items-center gap-3 ml-3">
              {subChartLegend.map((item) => (
                <span
                  key={item.label}
                  className="flex items-center gap-1 text-[11px]"
                  style={{ color: '#8b8fa3' }}
                >
                  <span
                    className="inline-block w-3 h-px"
                    style={{ background: item.color }}
                  />
                  {item.label}
                </span>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Quote summary */}
      {quote && (
        <div
          className="flex items-center gap-4 mb-4 px-4 py-3 rounded-lg"
          style={{ background: 'var(--color-bg-secondary)' }}
        >
          <span className="font-medium">{quote.name}</span>
          <span className="font-mono text-sm" style={{ color: 'var(--color-text-muted)' }}>
            {quote.code}
          </span>
          <span className="font-mono font-bold text-lg" style={{ color }}>
            {quote.price.toFixed(2)}
          </span>
          <span className="flex items-center gap-1 font-mono text-sm" style={{ color }}>
            <TrendIcon className="w-4 h-4" />
            {pct >= 0 ? '+' : ''}
            {pct.toFixed(2)}%
          </span>
        </div>
      )}

      {error && (
        <div className="mb-4">
          <ErrorState
            title="加载失败"
            description={error}
            onRetry={() => { setError(''); if (code) loadKline(code, period); }}
          />
        </div>
      )}

      {/* Loading state */}
      {loading && (
        <div className="text-center py-20" style={{ color: 'var(--color-text-muted)' }}>
          加载K线数据...
        </div>
      )}

      {/* Main K-line chart (lightweight-charts) */}
      <div
        ref={chartContainerRef}
        className="rounded-lg overflow-hidden"
        style={{
          border: chartHasData ? '1px solid var(--color-border)' : 'none',
          display: chartHasData ? 'block' : 'none',
        }}
      />

      {/* Sub-chart indicator (ECharts) */}
      {activeSub && subChartOption && chartHasData && (
        <div
          className="rounded-lg overflow-hidden mt-1 animate-fade-in"
          style={{
            border: '1px solid var(--color-border)',
            borderTop: 'none',
            borderTopLeftRadius: 0,
            borderTopRightRadius: 0,
          }}
        >
          <ReactECharts
            option={subChartOption}
            style={{ height: 150, width: '100%' }}
            opts={{ renderer: 'canvas' }}
            notMerge
          />
        </div>
      )}

      {/* Empty state */}
      {!code && !loading && (
        <div
          className="text-center py-16 rounded-lg border"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <p style={{ color: 'var(--color-text-muted)' }}>搜索股票查看K线图</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            支持日K、周K、月K线，MA/MACD/KDJ/RSI 技术指标
          </p>
        </div>
      )}
      <Alpha300Selector
        open={alpha300Open}
        onClose={() => setAlpha300Open(false)}
        onSelect={(selectedCode) => {
          setCode(selectedCode);
          handleSearch({ code: selectedCode, name: selectedCode });
        }}
      />
    </div>
  );
}

/* ── Glass-morphism indicator toggle button ────────────────────── */

function IndicatorButton({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="px-3 py-1 rounded-md text-xs font-medium transition-all duration-200"
      style={{
        background: active
          ? 'var(--color-accent)'
          : 'rgba(255,255,255,0.06)',
        backdropFilter: 'blur(12px)',
        WebkitBackdropFilter: 'blur(12px)',
        color: active ? '#fff' : '#8b8fa3',
        border: active
          ? '1px solid transparent'
          : '1px solid rgba(148,163,184,0.12)',
        boxShadow: active
          ? '0 0 12px rgba(59,130,246,0.25)'
          : 'none',
      }}
    >
      {label}
    </button>
  );
}
