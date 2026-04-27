import { useState, useEffect, useMemo } from 'react';
import api from '@/lib/api';
import ReactECharts from 'echarts-for-react';
import {
  RefreshCw,
  BarChart3,
  Zap,
} from 'lucide-react';
import { Skeleton } from '@/components/ui/Skeleton';

interface StrategyItem {
  label: string;
  trade_count: number;
  win_rate: string;
  total_pnl: number;
  avg_pnl: number;
  max_win: number;
  max_loss: number;
  profit_factor: number;
}

interface StrategyEvalData {
  ok: boolean;
  overall: {
    total_pnl: number;
    win_rate: string;
    total_trades: number;
  };
  strategies: StrategyItem[];
}

export default function StrategyEvalPage() {
  const [data, setData] = useState<StrategyEvalData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = () => {
    setLoading(true);
    setError('');
    api.get<StrategyEvalData>('/trade-strategy-eval')
      .then((res) => setData(res.data))
      .catch(() => setError('加载策略评估数据失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchData(); }, []);

  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

  // Bar chart comparing strategies
  const chartOption = useMemo(() => {
    if (!data?.strategies?.length) return null;
    const strategies = data.strategies;
    return {
      tooltip: {
        trigger: 'axis' as const,
        backgroundColor: 'rgba(15,23,42,0.9)',
        borderColor: 'rgba(59,130,246,0.3)',
        textStyle: { color: '#e2e8f0', fontSize: 12 },
      },
      legend: {
        data: ['总盈亏', '胜率'],
        textStyle: { color: '#94a3b8', fontSize: 11 },
        top: 0,
      },
      grid: { left: '3%', right: '4%', bottom: '3%', top: '40px', containLabel: true },
      xAxis: {
        type: 'category' as const,
        data: strategies.map((s) => s.label),
        axisLabel: { color: '#94a3b8', fontSize: 10, rotate: strategies.length > 5 ? 30 : 0 },
        axisLine: { lineStyle: { color: '#334155' } },
      },
      yAxis: [
        {
          type: 'value' as const,
          name: '盈亏',
          nameTextStyle: { color: '#64748b', fontSize: 10 },
          axisLabel: { color: '#64748b', fontSize: 10 },
          splitLine: { lineStyle: { color: 'rgba(51,65,85,0.5)' } },
        },
        {
          type: 'value' as const,
          name: '胜率%',
          nameTextStyle: { color: '#64748b', fontSize: 10 },
          axisLabel: { color: '#64748b', fontSize: 10, formatter: '{value}%' },
          splitLine: { show: false },
          max: 100,
        },
      ],
      series: [
        {
          name: '总盈亏',
          type: 'bar' as const,
          data: strategies.map((s) => ({
            value: s.total_pnl,
            itemStyle: { color: s.total_pnl >= 0 ? '#ef4444' : '#22c55e' },
          })),
          barMaxWidth: 40,
        },
        {
          name: '胜率',
          type: 'line' as const,
          yAxisIndex: 1,
          data: strategies.map((s) => parseFloat(s.win_rate)),
          symbol: 'circle',
          symbolSize: 8,
          lineStyle: { color: '#f59e0b', width: 2 },
          itemStyle: { color: '#f59e0b' },
        },
      ],
    };
  }, [data]);

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Zap className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">策略评估</h1>
        </div>
        <div className="space-y-4 py-4">
            <Skeleton count={3} />
            <Skeleton rows={5} />
          </div>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Zap className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">策略评估</h1>
        </div>
        <div className="text-sm px-4 py-3 rounded-lg" style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
          <button onClick={fetchData} className="ml-3 underline">重试</button>
        </div>
      </div>
    );
  }

  const isEmpty = !data?.ok || !data.strategies?.length;
  const strategies = data?.strategies || [];
  const overall = data?.overall;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Zap className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">策略评估</h1>
          <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>
            按策略标签分组统计
          </span>
        </div>
        <button onClick={fetchData} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      <p className="text-sm mb-5" style={{ color: 'var(--color-text-secondary)' }}>
        按策略标签分组统计胜率、收益、回撤，对比策略效果
      </p>

      {isEmpty ? (
        <div className="text-center py-20 rounded-xl" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
          <BarChart3 className="w-14 h-14 mx-auto mb-4" style={{ color: 'var(--color-text-muted)' }} />
          <p className="text-lg font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>暂无策略数据</p>
          <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>请先在交易日志中添加策略标签</p>
        </div>
      ) : (
        <>
          {/* Overall summary cards */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
            <div className="glass-panel rounded-xl p-4 text-center">
              <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>策略数量</div>
              <div className="text-2xl font-bold" style={{ color: 'var(--color-accent)' }}>{strategies.length}</div>
            </div>
            <div className="glass-panel rounded-xl p-4 text-center">
              <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>总盈亏</div>
              <div className="text-2xl font-bold" style={{ color: changeColor(overall?.total_pnl || 0) }}>
                {(overall?.total_pnl || 0).toLocaleString()}
              </div>
            </div>
            <div className="glass-panel rounded-xl p-4 text-center">
              <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>总胜率</div>
              <div className="text-2xl font-bold" style={{ color: '#f59e0b' }}>{overall?.win_rate || '0'}%</div>
            </div>
            <div className="glass-panel rounded-xl p-4 text-center">
              <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>总交易次数</div>
              <div className="text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>{overall?.total_trades || 0}</div>
            </div>
          </div>

          {/* Strategy comparison chart */}
          {chartOption && (
            <div className="glass-panel rounded-xl p-4 mb-6">
              <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
                📊 策略收益对比
              </h3>
              <ReactECharts option={chartOption} style={{ height: '300px' }} />
            </div>
          )}

          {/* Per-strategy detail cards */}
          <div className="space-y-4">
            {strategies.map((s) => {
              const pnlColor = changeColor(s.total_pnl);
              const pnlSign = s.total_pnl >= 0 ? '+' : '';
              return (
                <div key={s.label} className="glass-panel rounded-xl p-5">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <span className="text-lg">🏷️</span>
                      <span className="text-sm font-bold">{s.label}</span>
                      <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{s.trade_count}笔交易</span>
                    </div>
                    <span className="text-sm font-bold font-mono" style={{ color: pnlColor }}>
                      {pnlSign}{s.total_pnl.toLocaleString()}
                    </span>
                  </div>
                  <div className="grid grid-cols-2 sm:grid-cols-6 gap-3">
                    <MetricItem label="胜率" value={`${s.win_rate}%`} color="#f59e0b" />
                    <MetricItem
                      label="平均收益"
                      value={`${s.avg_pnl >= 0 ? '+' : ''}${s.avg_pnl.toLocaleString()}`}
                      color={changeColor(s.avg_pnl)}
                    />
                    <MetricItem
                      label="最大盈利"
                      value={`+${s.max_win.toLocaleString()}`}
                      color="var(--color-danger)"
                    />
                    <MetricItem
                      label="最大亏损"
                      value={`${s.max_loss.toLocaleString()}`}
                      color="var(--color-success)"
                    />
                    <MetricItem
                      label="盈亏比"
                      value={s.profit_factor.toFixed(2)}
                      color={s.profit_factor >= 1 ? 'var(--color-accent)' : 'var(--color-danger)'}
                    />
                    <MetricItem
                      label="评级"
                      value={getRating(s.win_rate, s.profit_factor)}
                      color={getRatingColor(s.win_rate, s.profit_factor)}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </>
      )}
    </div>
  );
}

function MetricItem({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div className="text-center p-2 rounded-lg" style={{ background: 'var(--color-bg-hover)' }}>
      <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>{label}</div>
      <div className="text-sm font-bold font-mono" style={{ color }}>{value}</div>
    </div>
  );
}

function getRating(winRate: string, profitFactor: number): string {
  const wr = parseFloat(winRate);
  if (wr >= 60 && profitFactor >= 2) return '优秀';
  if (wr >= 50 && profitFactor >= 1.5) return '良好';
  if (wr >= 40 && profitFactor >= 1) return '一般';
  return '较差';
}

function getRatingColor(winRate: string, profitFactor: number): string {
  const rating = getRating(winRate, profitFactor);
  if (rating === '优秀') return '#f59e0b';
  if (rating === '良好') return '#3b82f6';
  if (rating === '一般') return '#6b7280';
  return '#ef4444';
}
