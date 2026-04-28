import { useState, useEffect } from 'react';
import api from '@/lib/api';
import ReactECharts from '@/components/charts/ReactECharts';
import { Activity, RefreshCw, ArrowUpCircle, ArrowDownCircle, MinusCircle } from 'lucide-react';

interface BreadthData {
  advancing: number;
  declining: number;
  flat: number;
  limit_up: number;
  limit_down: number;
  ad_ratio: number;
  breadth_thrust: number;
  limit_ratio: number;
  total: number;
  volume_stats: {
    up_volume: number;
    down_volume: number;
    flat_volume: number;
  };
  distribution: number[] | null;
  timestamp: string;
}

interface BreadthResponse {
  ok: boolean;
  data: BreadthData;
  cached: boolean;
}

export default function BreadthPage() {
  const [data, setData] = useState<BreadthData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = () => {
    setLoading(true);
    setError('');
    api
      .get<BreadthResponse>('/market/breadth')
      .then((res) => setData(res.data.data))
      .catch(() => setError('加载市场广度数据失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchData();
  }, []);

  const getADRatioInterpretation = (ratio: number) => {
    if (ratio >= 2) return { text: '极度强势', color: 'var(--color-danger)' };
    if (ratio >= 1.5) return { text: '强势', color: 'var(--color-danger)' };
    if (ratio >= 1) return { text: '偏多', color: 'var(--color-warning)' };
    if (ratio >= 0.5) return { text: '偏空', color: 'var(--color-success)' };
    return { text: '弱势', color: 'var(--color-success)' };
  };

  // Pie chart for adv/dec/flat
  const getPieOption = (d: BreadthData) => ({
    tooltip: { trigger: 'item' as const, formatter: '{b}: {c} ({d}%)' },
    series: [{
      type: 'pie',
      radius: ['50%', '75%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 4, borderColor: 'var(--color-bg-primary)', borderWidth: 2 },
      label: { show: false },
      emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
      data: [
        { value: d.advancing, name: '上涨', itemStyle: { color: '#ef4444' } },
        { value: d.declining, name: '下跌', itemStyle: { color: '#22c55e' } },
        { value: d.flat, name: '平盘', itemStyle: { color: '#64748b' } },
      ].filter(item => item.value > 0),
    }],
  });

  // Bar chart for limit up/down
  const getLimitOption = (d: BreadthData) => ({
    tooltip: { trigger: 'axis' as const },
    grid: { top: 10, right: 20, bottom: 30, left: 60 },
    xAxis: { type: 'category' as const, data: ['涨停', '跌停'], axisLabel: { color: '#94a3b8' } },
    yAxis: { type: 'value' as const, axisLabel: { color: '#94a3b8' }, splitLine: { lineStyle: { color: 'rgba(148,163,184,0.1)' } } },
    series: [{
      type: 'bar',
      data: [
        { value: d.limit_up, itemStyle: { color: '#ef4444', borderRadius: [4, 4, 0, 0] } },
        { value: d.limit_down, itemStyle: { color: '#22c55e', borderRadius: [4, 4, 0, 0] } },
      ],
      barWidth: '40%',
    }],
  });

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Activity className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">市场广度</h1>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '100px' }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Activity className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">市场广度</h1>
        </div>
        <div className="text-sm px-4 py-3 rounded-lg" style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
          <button onClick={fetchData} className="ml-3 underline">重试</button>
        </div>
      </div>
    );
  }

  if (!data) return null;

  const adInterp = getADRatioInterpretation(data.ad_ratio);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Activity className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">市场广度</h1>
        </div>
        <button
          onClick={fetchData}
          className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <ArrowUpCircle className="w-4 h-4" style={{ color: 'var(--color-danger)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>上涨</span>
          </div>
          <div className="text-2xl font-bold font-mono" style={{ color: 'var(--color-danger)' }}>
            {data.advancing}
          </div>
        </div>
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <ArrowDownCircle className="w-4 h-4" style={{ color: 'var(--color-success)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>下跌</span>
          </div>
          <div className="text-2xl font-bold font-mono" style={{ color: 'var(--color-success)' }}>
            {data.declining}
          </div>
        </div>
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <MinusCircle className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>涨跌比</span>
          </div>
          <div className="text-2xl font-bold font-mono" style={{ color: adInterp.color }}>
            {data.ad_ratio.toFixed(2)}
          </div>
          <div className="text-xs mt-1" style={{ color: adInterp.color }}>{adInterp.text}</div>
        </div>
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <Activity className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>总量</span>
          </div>
          <div className="text-2xl font-bold font-mono" style={{ color: 'var(--color-text-primary)' }}>
            {data.total}
          </div>
        </div>
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
            涨跌分布
          </h3>
          <div className="h-48">
            <ReactECharts option={getPieOption(data)} style={{ height: '100%' }} />
          </div>
        </div>
        <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
            涨停 / 跌停
          </h3>
          <div className="h-48">
            <ReactECharts option={getLimitOption(data)} style={{ height: '100%' }} />
          </div>
        </div>
      </div>

      {/* Breadth indicators */}
      <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
        <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
          广度指标
        </h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <div>
            <div style={{ color: 'var(--color-text-muted)' }}>AD Ratio</div>
            <div className="font-mono font-medium">{data.ad_ratio.toFixed(3)}</div>
          </div>
          <div>
            <div style={{ color: 'var(--color-text-muted)' }}>Breadth Thrust</div>
            <div className="font-mono font-medium">{data.breadth_thrust.toFixed(3)}</div>
          </div>
          <div>
            <div style={{ color: 'var(--color-text-muted)' }}>Limit Ratio</div>
            <div className="font-mono font-medium">{data.limit_ratio.toFixed(3)}</div>
          </div>
          <div>
            <div style={{ color: 'var(--color-text-muted)' }}>平盘</div>
            <div className="font-mono font-medium">{data.flat}</div>
          </div>
        </div>
      </div>

      {/* Timestamp */}
      <div className="text-xs mt-4 text-right" style={{ color: 'var(--color-text-muted)' }}>
        更新时间: {data.timestamp ? new Date(data.timestamp).toLocaleString('zh-CN') : '—'}
      </div>
    </div>
  );
}
