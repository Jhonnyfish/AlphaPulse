import { useState, useMemo } from 'react';
import {
  compareApi,
  type BacktestResult,
  type BacktestTrade,
} from '@/lib/api';
import Alpha300Selector from '@/components/Alpha300Selector';
import ReactECharts from 'echarts-for-react';
import {
  FlaskConical,
  Play,
  Loader2,
  AlertCircle,
  TrendingUp,
  TrendingDown,
  BarChart3,
  ArrowUpDown,
  Trophy,
  Target,
  Shield,
  Hash,
} from 'lucide-react';
import { SkeletonStatCards, SkeletonList } from '@/components/ui/Skeleton';

type SortKey = 'signal_date' | 'return_pct' | 'holding_days' | 'score';
type SortDir = 'asc' | 'desc';

export default function BacktestPage() {
  const [codes, setCodes] = useState('');
  const [alpha300Open, setAlpha300Open] = useState(false);
  const [days, setDays] = useState(60);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [results, setResults] = useState<BacktestResult[]>([]);
  const [sortKey, setSortKey] = useState<SortKey>('signal_date');
  const [sortDir, setSortDir] = useState<SortDir>('desc');

  const handleRun = async () => {
    const codeList = codes
      .split(/[,，\s]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (codeList.length === 0) {
      setError('请输入至少1个股票代码');
      return;
    }
    if (codeList.length > 5) {
      setError('最多支持5个股票代码');
      return;
    }
    setLoading(true);
    setError('');
    setResults([]);
    try {
      const res = await compareApi.backtestCompare(codeList.join(','), days);
      setResults(res.data);
    } catch {
      setError('回测请求失败，请检查股票代码是否正确');
    } finally {
      setLoading(false);
    }
  };

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(key);
      setSortDir('desc');
    }
  };

  const changeColor = (n: number) =>
    n > 0
      ? 'var(--color-danger)'
      : n < 0
        ? 'var(--color-success)'
        : 'var(--color-text-secondary)';

  // Collect all trades from all results and sort
  const allTrades = useMemo(() => {
    const trades: (BacktestTrade & { code: string; name: string })[] = [];
    for (const r of results) {
      if (r.error) continue;
      for (const t of r.trades) {
        trades.push({ ...t, code: r.code, name: r.name });
      }
    }
    trades.sort((a, b) => {
      const av = a[sortKey];
      const bv = b[sortKey];
      const mul = sortDir === 'asc' ? 1 : -1;
      if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * mul;
      return String(av).localeCompare(String(bv)) * mul;
    });
    return trades;
  }, [results, sortKey, sortDir]);

  // Build ECharts option for equity curves
  const equityChartOption = useMemo(() => {
    const validResults = results.filter((r) => !r.error && r.equity_curve.length > 1);
    if (validResults.length === 0) return null;

    const colors = ['#3b82f6', '#f59e0b', '#ef4444', '#22c55e', '#a855f7'];

    return {
      tooltip: {
        trigger: 'axis' as const,
        backgroundColor: 'rgba(15,23,42,0.9)',
        borderColor: 'rgba(59,130,246,0.3)',
        textStyle: { color: '#e2e8f0', fontSize: 12 },
      },
      legend: {
        data: validResults.map((r) => r.name || r.code),
        textStyle: { color: '#94a3b8', fontSize: 11 },
        top: 0,
      },
      grid: { left: '3%', right: '4%', bottom: '3%', top: '40px', containLabel: true },
      xAxis: {
        type: 'category' as const,
        data: Array.from({ length: Math.max(...validResults.map((r) => r.equity_curve.length)) }, (_, i) => `T${i}`),
        axisLabel: { color: '#64748b', fontSize: 10 },
        axisLine: { lineStyle: { color: '#334155' } },
      },
      yAxis: {
        type: 'value' as const,
        axisLabel: { color: '#64748b', fontSize: 10, formatter: '{value}' },
        splitLine: { lineStyle: { color: 'rgba(51,65,85,0.5)' } },
      },
      series: validResults.map((r, i) => ({
        name: r.name || r.code,
        type: 'line' as const,
        data: r.equity_curve,
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 2, color: colors[i % colors.length] },
        areaStyle: {
          color: {
            type: 'linear' as const,
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: `${colors[i % colors.length]}33` },
              { offset: 1, color: `${colors[i % colors.length]}05` },
            ],
          },
        },
      })),
    };
  }, [results]);

  // Stats summary
  const stats = useMemo(() => {
    const valid = results.filter((r) => !r.error);
    if (valid.length === 0) return null;
    const avgWinRate = valid.reduce((s, r) => s + r.win_rate, 0) / valid.length;
    const avgReturn = valid.reduce((s, r) => s + r.avg_return_pct, 0) / valid.length;
    const maxDrawdown = Math.max(...valid.map((r) => r.max_drawdown_pct));
    const totalSignals = valid.reduce((s, r) => s + r.signal_count, 0);
    return { avgWinRate, avgReturn, maxDrawdown, totalSignals, count: valid.length };
  }, [results]);

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <FlaskConical className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">策略回测</h1>
          <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>
            8维评分策略
          </span>
        </div>
      </div>

      <p className="text-sm mb-5" style={{ color: 'var(--color-text-secondary)' }}>
        基于8维评分策略的历史回测模拟，自动标记买卖信号，分析胜率与收益
      </p>

      {/* Input form */}
      <div className="glass-panel rounded-xl p-4 mb-6">
        <div className="flex flex-wrap items-end gap-3">
          <div className="flex-1 min-w-[200px]">
            <label className="block text-xs font-medium mb-1.5" style={{ color: 'var(--color-text-secondary)' }}>
              股票代码（逗号分隔，1-5只）
            </label>
            <input
              value={codes}
              onChange={(e) => setCodes(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleRun()}
              placeholder="如 600519,000001,601318"
              className="w-full px-3 py-2 rounded-lg border text-sm outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
              style={{
                background: 'var(--color-bg-card)',
                borderColor: 'var(--color-border)',
                color: 'var(--color-text-primary)',
              }}
            />
          </div>
          <button type="button" onClick={() => setAlpha300Open(true)}
            className="px-2.5 py-2 rounded-lg text-sm shrink-0 transition-colors hover:bg-[var(--color-bg-hover)]"
            style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', color: 'var(--color-text-secondary)' }}
            title="从 Alpha300 选择">🎯</button>
          <div>
            <label className="block text-xs font-medium mb-1.5" style={{ color: 'var(--color-text-secondary)' }}>
              回测天数
            </label>
            <select
              value={days}
              onChange={(e) => setDays(Number(e.target.value))}
              className="px-3 py-2 rounded-lg border text-sm outline-none"
              style={{
                background: 'var(--color-bg-card)',
                borderColor: 'var(--color-border)',
                color: 'var(--color-text-primary)',
              }}
            >
              <option value={30}>30天</option>
              <option value={60}>60天</option>
              <option value={90}>90天</option>
              <option value={120}>120天</option>
              <option value={180}>180天</option>
              <option value={240}>240天</option>
            </select>
          </div>
          <button
            onClick={handleRun}
            disabled={loading}
            className="flex items-center gap-1.5 px-5 py-2 rounded-lg text-sm font-medium text-white transition-colors disabled:opacity-50"
            style={{ background: 'var(--color-accent)' }}
          >
            {loading ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Play className="w-4 h-4" />
            )}
            开始回测
          </button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div
          className="flex items-center gap-2 text-sm px-4 py-3 rounded-lg mb-6"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
        >
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="space-y-4 py-4">
          <SkeletonStatCards count={4} />
          <SkeletonList rows={5} />
        </div>
      )}

      {/* Results */}
      {!loading && results.length > 0 && (
        <>
          {/* Summary stats */}
          {stats && (
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
              <StatCard
                icon={<Trophy className="w-4 h-4" />}
                label="平均胜率"
                value={`${stats.avgWinRate.toFixed(1)}%`}
                color={stats.avgWinRate >= 50 ? 'var(--color-danger)' : 'var(--color-success)'}
              />
              <StatCard
                icon={<TrendingUp className="w-4 h-4" />}
                label="平均收益"
                value={`${stats.avgReturn >= 0 ? '+' : ''}${stats.avgReturn.toFixed(2)}%`}
                color={changeColor(stats.avgReturn)}
              />
              <StatCard
                icon={<Shield className="w-4 h-4" />}
                label="最大回撤"
                value={`-${stats.maxDrawdown.toFixed(2)}%`}
                color="var(--color-success)"
              />
              <StatCard
                icon={<Hash className="w-4 h-4" />}
                label="总信号数"
                value={String(stats.totalSignals)}
                color="var(--color-accent)"
              />
            </div>
          )}

          {/* Per-stock comparison table */}
          {results.length > 1 && (
            <div className="glass-panel rounded-xl overflow-hidden mb-6">
              <div className="px-4 py-3 font-medium text-sm" style={{ color: 'var(--color-text-secondary)', borderBottom: '1px solid var(--color-border)' }}>
                📊 回测对比表
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr style={{ background: 'var(--color-bg-card)', color: 'var(--color-text-muted)' }}>
                      <th className="px-4 py-2.5 text-left text-xs font-medium">股票</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium">信号数</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium">胜率</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium">平均收益</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium">最大回撤</th>
                      <th className="px-4 py-2.5 text-right text-xs font-medium">评级</th>
                    </tr>
                  </thead>
                  <tbody>
                    {results.map((r) => (
                      <tr key={r.code} className="border-t" style={{ borderColor: 'var(--color-border)' }}>
                        <td className="px-4 py-2.5">
                          <div className="font-medium">{r.error ? '-' : r.name}</div>
                          <div className="text-xs font-mono" style={{ color: 'var(--color-text-muted)' }}>{r.code}</div>
                        </td>
                        <td className="px-4 py-2.5 text-right font-mono" style={{ color: 'var(--color-text-secondary)' }}>
                          {r.error ? '-' : r.signal_count}
                        </td>
                        <td className="px-4 py-2.5 text-right font-mono" style={{ color: r.error ? 'var(--color-text-muted)' : changeColor(r.win_rate - 50) }}>
                          {r.error ? '-' : `${r.win_rate.toFixed(1)}%`}
                        </td>
                        <td className="px-4 py-2.5 text-right font-mono" style={{ color: r.error ? 'var(--color-text-muted)' : changeColor(r.avg_return_pct) }}>
                          {r.error ? '-' : `${r.avg_return_pct >= 0 ? '+' : ''}${r.avg_return_pct.toFixed(2)}%`}
                        </td>
                        <td className="px-4 py-2.5 text-right font-mono" style={{ color: 'var(--color-success)' }}>
                          {r.error ? '-' : `-${r.max_drawdown_pct.toFixed(2)}%`}
                        </td>
                        <td className="px-4 py-2.5 text-right">
                          {r.error ? (
                            <span className="text-xs" style={{ color: 'var(--color-danger)' }}>{r.error}</span>
                          ) : (
                            <RatingBadge winRate={r.win_rate} avgReturn={r.avg_return_pct} />
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Equity curve chart */}
          {equityChartOption && (
            <div className="glass-panel rounded-xl p-4 mb-6">
              <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>
                📈 资金曲线
              </h3>
              <ReactECharts option={equityChartOption} style={{ height: '320px' }} />
            </div>
          )}

          {/* Individual stock details */}
          {results.filter((r) => !r.error).map((r) => (
            <StockBacktestDetail key={r.code} result={r} changeColor={changeColor} />
          ))}

          {/* All trades table */}
          {allTrades.length > 0 && (
            <div className="glass-panel rounded-xl overflow-hidden mt-6">
              <div className="px-4 py-3 font-medium text-sm flex items-center justify-between" style={{ color: 'var(--color-text-secondary)', borderBottom: '1px solid var(--color-border)' }}>
                <span>📋 全部交易记录（{allTrades.length}笔）</span>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr style={{ background: 'var(--color-bg-card)', color: 'var(--color-text-muted)' }}>
                      <SortableTh label="日期" sortKey="signal_date" currentSort={sortKey} sortDir={sortDir} onToggle={toggleSort} />
                      <th className="px-3 py-2 text-left text-xs font-medium">股票</th>
                      <th className="px-3 py-2 text-right text-xs font-medium">买入价</th>
                      <th className="px-3 py-2 text-right text-xs font-medium">卖出价</th>
                      <SortableTh label="持有天数" sortKey="holding_days" currentSort={sortKey} sortDir={sortDir} onToggle={toggleSort} align="right" />
                      <SortableTh label="评分" sortKey="score" currentSort={sortKey} sortDir={sortDir} onToggle={toggleSort} align="right" />
                      <SortableTh label="收益率" sortKey="return_pct" currentSort={sortKey} sortDir={sortDir} onToggle={toggleSort} align="right" />
                    </tr>
                  </thead>
                  <tbody>
                    {allTrades.slice(0, 100).map((t, i) => (
                      <tr key={`${t.code}-${t.signal_date}-${i}`} className="border-t hover:bg-[var(--color-bg-hover)] transition-colors" style={{ borderColor: 'var(--color-border)' }}>
                        <td className="px-3 py-2 font-mono text-xs">{t.signal_date}</td>
                        <td className="px-3 py-2">
                          <span className="text-xs">{t.name}</span>
                          <span className="text-xs font-mono ml-1" style={{ color: 'var(--color-text-muted)' }}>{t.code}</span>
                        </td>
                        <td className="px-3 py-2 text-right font-mono text-xs">{t.buy_price.toFixed(2)}</td>
                        <td className="px-3 py-2 text-right font-mono text-xs">{t.sell_price.toFixed(2)}</td>
                        <td className="px-3 py-2 text-right font-mono text-xs" style={{ color: 'var(--color-text-secondary)' }}>{t.holding_days}天</td>
                        <td className="px-3 py-2 text-right font-mono text-xs" style={{ color: 'var(--color-accent)' }}>{t.score.toFixed(0)}</td>
                        <td className="px-3 py-2 text-right font-mono text-xs" style={{ color: changeColor(t.return_pct) }}>
                          {t.return_pct >= 0 ? '+' : ''}{t.return_pct.toFixed(2)}%
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {allTrades.length > 100 && (
                  <div className="text-center py-2 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                    仅显示前100条记录
                  </div>
                )}
              </div>
            </div>
          )}
        </>
      )}

      {/* Empty state */}
      {!loading && !error && results.length === 0 && (
        <div className="text-center py-20 rounded-xl" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
          <FlaskConical className="w-14 h-14 mx-auto mb-4" style={{ color: 'var(--color-text-muted)' }} />
          <p className="text-lg font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>输入股票代码开始策略回测</p>
          <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>支持1-5只股票，基于8维评分策略的买卖信号模拟</p>
        </div>
      )}
    </div>
  );
}

/* ─── Sub-components ─── */

function StatCard({
  icon,
  label,
  value,
  color,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  color: string;
}) {
  return (
    <div
      className="glass-panel rounded-xl p-4"
    >
      <div className="flex items-center gap-2 mb-2">
        <div style={{ color }}>{icon}</div>
        <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{label}</span>
      </div>
      <div className="text-xl font-bold font-mono" style={{ color }}>{value}</div>
    </div>
  );
}

function RatingBadge({ winRate, avgReturn }: { winRate: number; avgReturn: number }) {
  let label: string;
  let color: string;
  if (winRate >= 60 && avgReturn > 2) {
    label = '优秀';
    color = '#f59e0b';
  } else if (winRate >= 50 && avgReturn > 0) {
    label = '良好';
    color = '#3b82f6';
  } else if (winRate >= 40) {
    label = '一般';
    color = '#6b7280';
  } else {
    label = '较差';
    color = '#ef4444';
  }
  return (
    <span
      className="text-xs px-2 py-0.5 rounded-full font-medium"
      style={{ background: `${color}20`, color }}
    >
      {label}
    </span>
  );
}

function SortableTh({
  label,
  sortKey: key,
  currentSort,
  sortDir,
  onToggle,
  align = 'left',
}: {
  label: string;
  sortKey: SortKey;
  currentSort: SortKey;
  sortDir: SortDir;
  onToggle: (key: SortKey) => void;
  align?: 'left' | 'right';
}) {
  const active = currentSort === key;
  return (
    <th
      className={`px-3 py-2 text-xs font-medium cursor-pointer select-none hover:text-[var(--color-text-primary)] transition-colors ${align === 'right' ? 'text-right' : 'text-left'}`}
      onClick={() => onToggle(key)}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        <ArrowUpDown className="w-3 h-3" style={{ opacity: active ? 1 : 0.3 }} />
      </span>
    </th>
  );
}

function StockBacktestDetail({
  result,
  changeColor,
}: {
  result: BacktestResult;
  changeColor: (n: number) => string;
}) {
  if (result.error) return null;

  // Build per-stock equity chart
  const chartOption = result.equity_curve.length > 1
    ? {
        tooltip: {
          trigger: 'axis' as const,
          backgroundColor: 'rgba(15,23,42,0.9)',
          borderColor: 'rgba(59,130,246,0.3)',
          textStyle: { color: '#e2e8f0', fontSize: 12 },
        },
        grid: { left: '3%', right: '4%', bottom: '3%', top: '10px', containLabel: true },
        xAxis: {
          type: 'category' as const,
          data: result.equity_curve.map((_, i) => `T${i}`),
          axisLabel: { color: '#64748b', fontSize: 10 },
          axisLine: { lineStyle: { color: '#334155' } },
        },
        yAxis: {
          type: 'value' as const,
          axisLabel: { color: '#64748b', fontSize: 10 },
          splitLine: { lineStyle: { color: 'rgba(51,65,85,0.5)' } },
        },
        series: [{
          type: 'line' as const,
          data: result.equity_curve,
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#3b82f6' },
          areaStyle: {
            color: {
              type: 'linear' as const,
              x: 0, y: 0, x2: 0, y2: 1,
              colorStops: [
                { offset: 0, color: 'rgba(59,130,246,0.2)' },
                { offset: 1, color: 'rgba(59,130,246,0.02)' },
              ],
            },
          },
        }],
      }
    : null;

  return (
    <div className="glass-panel rounded-xl p-4 mb-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <BarChart3 className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <span className="font-medium">{result.name}</span>
          <span className="text-xs font-mono" style={{ color: 'var(--color-text-muted)' }}>{result.code}</span>
        </div>
        <div className="flex items-center gap-4 text-xs">
          <span style={{ color: 'var(--color-text-muted)' }}>
            胜率 <span className="font-mono font-medium" style={{ color: changeColor(result.win_rate - 50) }}>{result.win_rate.toFixed(1)}%</span>
          </span>
          <span style={{ color: 'var(--color-text-muted)' }}>
            均收 <span className="font-mono font-medium" style={{ color: changeColor(result.avg_return_pct) }}>
              {result.avg_return_pct >= 0 ? '+' : ''}{result.avg_return_pct.toFixed(2)}%
            </span>
          </span>
          <span style={{ color: 'var(--color-text-muted)' }}>
            信号 <span className="font-mono font-medium" style={{ color: 'var(--color-accent)' }}>{result.signal_count}</span>
          </span>
        </div>
      </div>

      {chartOption && (
        <ReactECharts option={chartOption} style={{ height: '200px' }} />
      )}

      {/* Trades for this stock */}
      {result.trades.length > 0 && (
        <div className="mt-3">
          <div className="text-xs font-medium mb-2" style={{ color: 'var(--color-text-muted)' }}>
            交易明细（{result.trades.length}笔）
          </div>
          <div className="overflow-x-auto max-h-60 overflow-y-auto rounded-lg" style={{ border: '1px solid var(--color-border)' }}>
            <table className="w-full text-xs">
              <thead className="sticky top-0">
                <tr style={{ background: 'var(--color-bg-card)', color: 'var(--color-text-muted)' }}>
                  <th className="px-3 py-1.5 text-left font-medium">买入日期</th>
                  <th className="px-3 py-1.5 text-left font-medium">卖出日期</th>
                  <th className="px-3 py-1.5 text-right font-medium">买入价</th>
                  <th className="px-3 py-1.5 text-right font-medium">卖出价</th>
                  <th className="px-3 py-1.5 text-right font-medium">天数</th>
                  <th className="px-3 py-1.5 text-right font-medium">评分</th>
                  <th className="px-3 py-1.5 text-right font-medium">收益</th>
                </tr>
              </thead>
              <tbody>
                {result.trades.map((t, i) => (
                  <tr key={i} className="border-t hover:bg-[var(--color-bg-hover)] transition-colors" style={{ borderColor: 'var(--color-border)' }}>
                    <td className="px-3 py-1.5 font-mono">{t.signal_date}</td>
                    <td className="px-3 py-1.5 font-mono">{t.sell_date}</td>
                    <td className="px-3 py-1.5 text-right font-mono">{t.buy_price.toFixed(2)}</td>
                    <td className="px-3 py-1.5 text-right font-mono">{t.sell_price.toFixed(2)}</td>
                    <td className="px-3 py-1.5 text-right font-mono" style={{ color: 'var(--color-text-secondary)' }}>{t.holding_days}</td>
                    <td className="px-3 py-1.5 text-right font-mono" style={{ color: 'var(--color-accent)' }}>{t.score.toFixed(0)}</td>
                    <td className="px-3 py-1.5 text-right font-mono" style={{ color: changeColor(t.return_pct) }}>
                      {t.return_pct >= 0 ? '+' : ''}{t.return_pct.toFixed(2)}%
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
      <Alpha300Selector
        open={alpha300Open}
        onClose={() => setAlpha300Open(false)}
        onSelect={(code) => setCodes((prev) => prev ? prev + ',' + code : code)}
      />
    </div>
  );
}
