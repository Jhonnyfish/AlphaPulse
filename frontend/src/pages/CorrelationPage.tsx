import { useState, useEffect } from 'react';
import api from '@/lib/api';
import ReactECharts from '@/components/charts/ReactECharts';
import { Network, RefreshCw } from 'lucide-react';

interface CorrelationData {
  ok: boolean;
  codes: string[];
  names: string[];
  matrix: number[][];
  message?: string;
  cached: boolean;
}

export default function CorrelationPage() {
  const [data, setData] = useState<CorrelationData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = () => {
    setLoading(true);
    setError('');
    api.get<CorrelationData>('/correlation')
      .then((res) => setData(res.data))
      .catch(() => setError('加载相关性数据失败'))
      .finally(() => setLoading(false));
  };

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { fetchData(); }, []);

  // Heatmap chart
  const heatmapChart = data?.matrix && data.matrix.length > 0 ? {
    tooltip: {
      position: 'top' as const,
      formatter: (p: { data: [number, number, number] }) => {
        const [x, y, val] = p.data;
        const nameI = data.names[y] || data.codes[y];
        const nameJ = data.names[x] || data.codes[x];
        return `${nameI} ↔ ${nameJ}<br/>相关系数: ${val.toFixed(3)}`;
      },
    },
    grid: { left: '12%', right: '8%', bottom: '15%', top: '5%' },
    xAxis: {
      type: 'category' as const,
      data: data.codes,
      axisLabel: { rotate: 45, fontSize: 10, color: '#94a3b8' },
      splitArea: { show: true, areaStyle: { color: ['rgba(30,41,59,0.5)', 'rgba(15,23,42,0.5)'] } },
    },
    yAxis: {
      type: 'category' as const,
      data: data.codes,
      axisLabel: { fontSize: 10, color: '#94a3b8' },
      splitArea: { show: true, areaStyle: { color: ['rgba(30,41,59,0.5)', 'rgba(15,23,42,0.5)'] } },
    },
    visualMap: {
      min: -1,
      max: 1,
      calculable: true,
      orient: 'horizontal' as const,
      left: 'center' as const,
      bottom: '0%',
      inRange: {
        color: ['#22c55e', '#3b82f6', '#6b7280', '#f59e0b', '#ef4444'],
      },
      textStyle: { color: '#94a3b8' },
    },
    series: [{
      type: 'heatmap' as const,
      data: data.matrix.flatMap((row, y) =>
        row.map((val, x) => [x, y, +val.toFixed(3)])
      ),
      label: { show: data.codes.length <= 10, fontSize: 10, color: '#e2e8f0' },
      emphasis: { itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.5)' } },
    }],
  } : null;

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Network className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">相关性分析</h1>
        </div>
        <div className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '400px' }} />
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Network className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">相关性分析</h1>
        </div>
        <div className="text-sm px-4 py-3 rounded-lg" style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
          <button onClick={fetchData} className="ml-3 underline">重试</button>
        </div>
      </div>
    );
  }

  const isEmpty = !data?.codes?.length || data.codes.length < 2;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Network className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">相关性分析</h1>
          {data?.cached && (
            <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>缓存</span>
          )}
        </div>
        <button onClick={fetchData} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      {isEmpty ? (
        <div className="text-center py-16 rounded-xl" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
          <Network className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>{data?.message || '需要至少2只自选股才能计算相关性'}</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>请先添加自选股后再查看</p>
        </div>
      ) : (
        <>
          {/* Heatmap */}
          {heatmapChart && (
            <div className="glass-panel rounded-xl p-4 mb-6" style={{ borderColor: 'var(--color-border)' }}>
              <h3 className="text-sm font-medium mb-2" style={{ color: 'var(--color-text-secondary)' }}>
                自选股相关性热力图 ({data!.codes.length} × {data!.codes.length})
              </h3>
              <ReactECharts option={heatmapChart} style={{ height: Math.max(400, data!.codes.length * 40 + 100) + 'px' }} />
            </div>
          )}

          {/* Legend */}
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>相关性说明</h3>
            <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
              {[
                { range: '0.7 ~ 1.0', label: '强正相关', color: '#ef4444' },
                { range: '0.4 ~ 0.7', label: '中正相关', color: '#f59e0b' },
                { range: '-0.4 ~ 0.4', label: '低相关', color: '#6b7280' },
                { range: '-0.7 ~ -0.4', label: '中负相关', color: '#3b82f6' },
                { range: '-1.0 ~ -0.7', label: '强负相关', color: '#22c55e' },
              ].map((item) => (
                <div key={item.range} className="flex items-center gap-2 p-2 rounded-lg" style={{ background: 'var(--color-bg-hover)' }}>
                  <div className="w-3 h-3 rounded-full" style={{ background: item.color }} />
                  <div>
                    <div className="text-xs font-medium">{item.label}</div>
                    <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{item.range}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
