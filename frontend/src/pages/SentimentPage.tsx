import { useState, useEffect } from 'react';
import api from '@/lib/api';
import ReactECharts from '@/components/charts/ReactECharts';
import { Heart, RefreshCw, ArrowUpCircle, ArrowDownCircle, Thermometer, TrendingUp } from 'lucide-react';

interface SentimentData {
  ok: boolean;
  fear_greed_index: number;
  fear_greed_label: string;
  up_count: number;
  down_count: number;
  flat_count: number;
  total_count: number;
  limit_up: number;
  limit_down: number;
  volume_today: number;
  volume_avg_5d: number;
  sector_volumes: { name: string; volume: number; change_pct: number }[] | null;
  temperature: number;
  server_time: string;
}

function getSentimentColor(index: number) {
  if (index >= 80) return '#ef4444'; // Extreme greed
  if (index >= 60) return '#f59e0b'; // Greed
  if (index >= 40) return '#64748b'; // Neutral
  if (index >= 20) return '#22d3ee'; // Fear
  return '#22c55e'; // Extreme fear
}

function getSentimentEmoji(index: number) {
  if (index >= 80) return '🔥';
  if (index >= 60) return '😊';
  if (index >= 40) return '😐';
  if (index >= 20) return '😰';
  return '😱';
}

export default function SentimentPage() {
  const [data, setData] = useState<SentimentData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = () => {
    setLoading(true);
    setError('');
    api
      .get<SentimentData>('/market/sentiment')
      .then((res) => setData(res.data))
      .catch(() => setError('加载情绪数据失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchData();
  }, []);

  // Gauge chart for fear/greed index
  const getGaugeOption = (index: number) => ({
    series: [{
      type: 'gauge',
      startAngle: 200,
      endAngle: -20,
      min: 0,
      max: 100,
      splitNumber: 5,
      itemStyle: { color: getSentimentColor(index) },
      progress: { show: true, width: 16, roundCap: true },
      pointer: { show: false },
      axisLine: { lineStyle: { width: 16, color: [[0.2, '#22c55e'], [0.4, '#22d3ee'], [0.6, '#64748b'], [0.8, '#f59e0b'], [1, '#ef4444']] } },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: { show: false },
      title: { show: false },
      detail: {
        valueAnimation: true,
        fontSize: 32,
        fontWeight: 'bold',
        color: getSentimentColor(index),
        offsetCenter: [0, '10%'],
        formatter: '{value}',
      },
      data: [{ value: index }],
    }],
  });

  // Bar chart for up/down counts
  const getUpDownOption = (d: SentimentData) => ({
    tooltip: { trigger: 'axis' as const },
    grid: { top: 10, right: 20, bottom: 30, left: 60 },
    xAxis: {
      type: 'category' as const,
      data: ['上涨', '平盘', '下跌', '涨停', '跌停'],
      axisLabel: { color: '#94a3b8' },
    },
    yAxis: { type: 'value' as const, axisLabel: { color: '#94a3b8' }, splitLine: { lineStyle: { color: 'rgba(148,163,184,0.1)' } } },
    series: [{
      type: 'bar',
      data: [
        { value: d.up_count, itemStyle: { color: '#ef4444', borderRadius: [4, 4, 0, 0] } },
        { value: d.flat_count, itemStyle: { color: '#64748b', borderRadius: [4, 4, 0, 0] } },
        { value: d.down_count, itemStyle: { color: '#22c55e', borderRadius: [4, 4, 0, 0] } },
        { value: d.limit_up, itemStyle: { color: '#dc2626', borderRadius: [4, 4, 0, 0] } },
        { value: d.limit_down, itemStyle: { color: '#16a34a', borderRadius: [4, 4, 0, 0] } },
      ],
      barWidth: '50%',
    }],
  });

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Heart className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">市场情绪</h1>
        </div>
        <div className="glass-panel rounded-xl p-8 animate-pulse" style={{ height: '200px' }} />
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Heart className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">市场情绪</h1>
        </div>
        <div className="text-sm px-4 py-3 rounded-lg" style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
          <button onClick={fetchData} className="ml-3 underline">重试</button>
        </div>
      </div>
    );
  }

  if (!data) return null;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Heart className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">市场情绪</h1>
        </div>
        <button onClick={fetchData} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      {/* Fear & Greed Gauge */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        <div className="glass-panel rounded-xl p-6" style={{ borderColor: 'var(--color-border)' }}>
          <div className="text-center mb-2">
            <span className="text-3xl">{getSentimentEmoji(data.fear_greed_index)}</span>
          </div>
          <div className="h-48">
            <ReactECharts option={getGaugeOption(data.fear_greed_index)} style={{ height: '100%' }} />
          </div>
          <div className="text-center mt-2">
            <span className="text-lg font-bold" style={{ color: getSentimentColor(data.fear_greed_index) }}>
              {data.fear_greed_label}
            </span>
            <span className="text-sm ml-2" style={{ color: 'var(--color-text-muted)' }}>
              ({data.fear_greed_index}/100)
            </span>
          </div>
        </div>

        {/* Up/Down Bar Chart */}
        <div className="glass-panel rounded-xl p-6" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
            涨跌统计
          </h3>
          <div className="h-48">
            <ReactECharts option={getUpDownOption(data)} style={{ height: '100%' }} />
          </div>
        </div>
      </div>

      {/* Detail cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <ArrowUpCircle className="w-4 h-4" style={{ color: 'var(--color-danger)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>上涨</span>
          </div>
          <div className="text-xl font-bold font-mono" style={{ color: 'var(--color-danger)' }}>{data.up_count}</div>
        </div>
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <ArrowDownCircle className="w-4 h-4" style={{ color: 'var(--color-success)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>下跌</span>
          </div>
          <div className="text-xl font-bold font-mono" style={{ color: 'var(--color-success)' }}>{data.down_count}</div>
        </div>
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <Thermometer className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>温度</span>
          </div>
          <div className="text-xl font-bold font-mono" style={{ color: getSentimentColor(data.temperature) }}>
            {data.temperature}
          </div>
        </div>
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <TrendingUp className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>涨停</span>
          </div>
          <div className="text-xl font-bold font-mono" style={{ color: 'var(--color-danger)' }}>{data.limit_up}</div>
        </div>
      </div>

      {/* Volume comparison */}
      <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
        <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
          成交量对比
        </h3>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <div style={{ color: 'var(--color-text-muted)' }}>今日成交量</div>
            <div className="font-mono font-medium">{data.volume_today.toLocaleString()}</div>
          </div>
          <div>
            <div style={{ color: 'var(--color-text-muted)' }}>5日均量</div>
            <div className="font-mono font-medium">{data.volume_avg_5d.toLocaleString()}</div>
          </div>
        </div>
        {data.volume_avg_5d > 0 && (
          <div className="mt-2">
            <div className="flex items-center gap-2 text-xs" style={{ color: 'var(--color-text-muted)' }}>
              <span>量比</span>
              <div
                className="flex-1 h-2 rounded-full overflow-hidden"
                style={{ background: 'var(--color-bg-hover)' }}
              >
                <div
                  className="h-full rounded-full transition-all"
                  style={{
                    width: `${Math.min((data.volume_today / data.volume_avg_5d) * 50, 100)}%`,
                    background: data.volume_today > data.volume_avg_5d ? 'var(--color-danger)' : 'var(--color-success)',
                  }}
                />
              </div>
              <span className="font-mono">
                {(data.volume_today / Math.max(data.volume_avg_5d, 1)).toFixed(2)}x
              </span>
            </div>
          </div>
        )}
      </div>

      {/* Timestamp */}
      <div className="text-xs mt-4 text-right" style={{ color: 'var(--color-text-muted)' }}>
        更新时间: {data.server_time ? new Date(data.server_time).toLocaleString('zh-CN') : '—'}
      </div>
    </div>
  );
}
