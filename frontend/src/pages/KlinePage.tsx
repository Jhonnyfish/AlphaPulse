import { useState, useEffect, useRef, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { createChart, CandlestickSeries, HistogramSeries, type IChartApi } from 'lightweight-charts';
import { marketApi, type KlinePoint, type Quote } from '@/lib/api';
import { Search, TrendingUp, TrendingDown, Minus } from 'lucide-react';

const PERIODS = [
  { value: 'daily', label: '日K', days: 120 },
  { value: 'weekly', label: '周K', days: 365 },
  { value: 'monthly', label: '月K', days: 730 },
] as const;

export default function KlinePage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [code, setCode] = useState(searchParams.get('code') || '');
  const [inputCode, setInputCode] = useState(searchParams.get('code') || '');
  const [period, setPeriod] = useState('daily');
  const [quote, setQuote] = useState<Quote | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);

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
      setQuote(quoteRes.data);

      if (!chartContainerRef.current) return;

      // Clear old chart
      if (chartRef.current) {
        chartRef.current.remove();
        chartRef.current = null;
      }

      const chart = createChart(chartContainerRef.current, {
        width: chartContainerRef.current.clientWidth,
        height: 500,
        layout: {
          background: { color: '#1a1d27' },
          textColor: '#8b8fa3',
        },
        grid: {
          vertLines: { color: '#2a2d3e' },
          horzLines: { color: '#2a2d3e' },
        },
        crosshair: {
          mode: 0,
        },
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

      const candleData = points.map((p) => ({
        time: p.date as string,
        open: p.open,
        high: p.high,
        low: p.low,
        close: p.close,
      }));
      candleSeries.setData(candleData);

      // Volume series
      const volumeSeries = chart.addSeries(HistogramSeries, {
        priceFormat: { type: 'volume' },
        priceScaleId: 'volume',
      });

      chart.priceScale('volume').applyOptions({
        scaleMargins: { top: 0.8, bottom: 0 },
      });

      const volumeData = points.map((p) => ({
        time: p.date as string,
        value: p.volume,
        color: p.close >= p.open ? 'rgba(239,68,68,0.4)' : 'rgba(34,197,94,0.4)',
      }));
      volumeSeries.setData(volumeData);

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

  // Load when code or period changes
  useEffect(() => {
    if (code) {
      loadKline(code, period);
    }
    return () => {
      if (chartRef.current) {
        chartRef.current.remove();
        chartRef.current = null;
      }
    };
  }, [code, period, loadKline]);

  // Resize handler
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

  const handleSearch = () => {
    const c = inputCode.trim();
    if (!c) return;
    setCode(c);
    setSearchParams({ code: c });
  };

  const pct = quote?.change_percent ?? 0;
  const color =
    pct > 0 ? 'var(--color-danger)' : pct < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';
  const TrendIcon = pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Minus;

  return (
    <div>
      <h1 className="text-xl font-bold mb-6">K线图</h1>

      {/* Search bar */}
      <div className="flex gap-2 mb-4">
        <div className="relative flex-1 max-w-xs">
          <Search
            className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4"
            style={{ color: 'var(--color-text-muted)' }}
          />
          <input
            type="text"
            value={inputCode}
            onChange={(e) => setInputCode(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            placeholder="输入股票代码，如 600519"
            className="w-full pl-9 pr-3 py-2 rounded-lg border text-sm outline-none"
            style={{
              background: 'var(--color-bg-card)',
              borderColor: 'var(--color-border)',
              color: 'var(--color-text-primary)',
            }}
          />
        </div>
        <button
          onClick={handleSearch}
          disabled={loading || !inputCode.trim()}
          className="px-4 py-2 rounded-lg text-sm font-medium text-white transition-colors disabled:opacity-50"
          style={{ background: 'var(--color-accent)' }}
        >
          查看
        </button>
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
        <div
          className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
        >
          {error}
        </div>
      )}

      {/* Chart container */}
      {loading && (
        <div className="text-center py-20" style={{ color: 'var(--color-text-muted)' }}>
          加载K线数据...
        </div>
      )}

      <div
        ref={chartContainerRef}
        className="rounded-lg overflow-hidden"
        style={{
          border: code ? '1px solid var(--color-border)' : 'none',
          display: code && !loading ? 'block' : 'none',
        }}
      />

      {/* Empty state */}
      {!code && !loading && (
        <div
          className="text-center py-16 rounded-lg border"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <p style={{ color: 'var(--color-text-muted)' }}>输入股票代码查看K线图</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            支持日K、周K、月K线，红涨绿跌
          </p>
        </div>
      )}
    </div>
  );
}
