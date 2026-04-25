import { useState } from 'react';
import {
  compareApi,
  type SectorCompareResult,
  type BacktestResult,
} from '@/lib/api';
import {
  Search,
  Play,
  Loader2,
  AlertCircle,
  TrendingUp,
  BarChart3,
} from 'lucide-react';

type Tab = 'sector' | 'backtest';

export default function ComparePage() {
  const [tab, setTab] = useState<Tab>('sector');

  // Sector state
  const [sectorCode, setSectorCode] = useState('');
  const [sectorLoading, setSectorLoading] = useState(false);
  const [sectorError, setSectorError] = useState('');
  const [sectorResult, setSectorResult] = useState<SectorCompareResult | null>(null);

  // Backtest state
  const [btCodes, setBtCodes] = useState('');
  const [btDays, setBtDays] = useState(30);
  const [btLoading, setBtLoading] = useState(false);
  const [btError, setBtError] = useState('');
  const [btResults, setBtResults] = useState<BacktestResult[]>([]);

  const handleSectorSearch = async () => {
    const code = sectorCode.trim();
    if (!code) return;
    setSectorLoading(true);
    setSectorError('');
    setSectorResult(null);
    try {
      const res = await compareApi.sectorCompare(code);
      setSectorResult(res.data);
    } catch {
      setSectorError('查询板块信息失败，请检查股票代码');
    } finally {
      setSectorLoading(false);
    }
  };

  const handleBacktest = async () => {
    const codes = btCodes
      .split(/[,，\s]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (codes.length < 2 || codes.length > 5) {
      setBtError('请输入 2-5 个股票代码');
      return;
    }
    setBtLoading(true);
    setBtError('');
    setBtResults([]);
    try {
      const res = await compareApi.backtestCompare(codes.join(','), btDays);
      setBtResults(res.data);
    } catch {
      setBtError('回测查询失败，请检查股票代码');
    } finally {
      setBtLoading(false);
    }
  };

  const changeColor = (n: number) =>
    n > 0
      ? 'var(--color-danger)'
      : n < 0
        ? 'var(--color-success)'
        : 'var(--color-text-secondary)';

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-bold">对比分析</h1>
          <p
            className="text-xs mt-0.5"
            style={{ color: 'var(--color-text-muted)' }}
          >
            板块排名对比 &amp; 回测策略对比
          </p>
        </div>
      </div>

      {/* Tabs */}
      <div
        className="flex rounded-lg overflow-hidden border mb-6"
        style={{ borderColor: 'var(--color-border)', width: 'fit-content' }}
      >
        <button
          onClick={() => setTab('sector')}
          className="flex items-center gap-1.5 px-4 py-2 text-sm transition-colors"
          style={{
            background:
              tab === 'sector' ? 'var(--color-accent)' : 'var(--color-bg-card)',
            color: tab === 'sector' ? '#fff' : 'var(--color-text-muted)',
          }}
        >
          <BarChart3 className="w-3.5 h-3.5" />
          板块对比
        </button>
        <button
          onClick={() => setTab('backtest')}
          className="flex items-center gap-1.5 px-4 py-2 text-sm transition-colors"
          style={{
            background:
              tab === 'backtest' ? 'var(--color-accent)' : 'var(--color-bg-card)',
            color: tab === 'backtest' ? '#fff' : 'var(--color-text-muted)',
          }}
        >
          <TrendingUp className="w-3.5 h-3.5" />
          回测对比
        </button>
      </div>

      {/* ── Sector Compare ── */}
      {tab === 'sector' && (
        <div>
          <div className="flex items-center gap-2 mb-4">
            <input
              value={sectorCode}
              onChange={(e) => setSectorCode(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSectorSearch()}
              placeholder="输入股票代码，如 600176"
              className="flex-1 max-w-xs px-3 py-2 rounded-lg border text-sm outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
              style={{
                background: 'var(--color-bg-card)',
                borderColor: 'var(--color-border)',
                color: 'var(--color-text-primary)',
              }}
            />
            <button
              onClick={handleSectorSearch}
              disabled={sectorLoading}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm text-white transition-colors disabled:opacity-50"
              style={{ background: 'var(--color-accent)' }}
            >
              {sectorLoading ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Search className="w-3.5 h-3.5" />
              )}
              查询
            </button>
          </div>

          {sectorError && (
            <div
              className="flex items-center gap-2 text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
              style={{
                background: 'rgba(239,68,68,0.1)',
                color: 'var(--color-danger)',
              }}
            >
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              {sectorError}
            </div>
          )}

          {sectorResult && (
            <div>
              {/* Sector info */}
              <div
                className="flex items-center gap-4 mb-4 px-4 py-2.5 rounded-lg text-sm"
                style={{ background: 'var(--color-bg-secondary)' }}
              >
                <span style={{ color: 'var(--color-text-muted)' }}>
                  所属板块:
                  <span
                    className="ml-1 font-medium"
                    style={{ color: 'var(--color-text-primary)' }}
                  >
                    {sectorResult.sector_name}
                  </span>
                </span>
                <span style={{ color: 'var(--color-text-muted)' }}>
                  板块代码:
                  <span className="ml-1 font-mono">{sectorResult.board_code}</span>
                </span>
                <span style={{ color: 'var(--color-text-muted)' }}>
                  排名:
                  <span className="ml-1 font-bold" style={{ color: 'var(--color-accent)' }}>
                    {sectorResult.current_rank} / {sectorResult.total_count}
                  </span>
                </span>
              </div>

              {/* Top 5 table */}
              <div
                className="rounded-lg border overflow-hidden"
                style={{
                  background: 'var(--color-bg-secondary)',
                  borderColor: 'var(--color-border)',
                }}
              >
                <table className="w-full text-sm">
                  <thead>
                    <tr
                      className="text-left text-xs"
                      style={{
                        background: 'var(--color-bg-card)',
                        color: 'var(--color-text-muted)',
                      }}
                    >
                      <th className="px-4 py-2 font-medium">排名</th>
                      <th className="px-4 py-2 font-medium">名称</th>
                      <th className="px-4 py-2 font-medium">代码</th>
                      <th className="px-4 py-2 font-medium text-right">涨跌幅</th>
                      <th className="px-4 py-2 font-medium text-right">PE</th>
                      <th className="px-4 py-2 font-medium text-right">PB</th>
                      <th className="px-4 py-2 font-medium text-right">成交额(万)</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sectorResult.top5.map((m, i) => (
                      <tr
                        key={m.code}
                        className="border-t"
                        style={{
                          borderColor: 'var(--color-border)',
                          background:
                            m.code === sectorResult.code
                              ? 'rgba(var(--color-accent-rgb, 59 130 246), 0.08)'
                              : undefined,
                        }}
                      >
                        <td className="px-4 py-2.5 font-mono text-xs">
                          {i + 1}
                        </td>
                        <td className="px-4 py-2.5">
                          <span className="font-medium">{m.name}</span>
                          {m.code === sectorResult.code && (
                            <span
                              className="ml-2 text-xs px-1.5 py-0.5 rounded"
                              style={{
                                background: 'var(--color-accent)',
                                color: '#fff',
                              }}
                            >
                              当前
                            </span>
                          )}
                        </td>
                        <td
                          className="px-4 py-2.5 font-mono text-xs"
                          style={{ color: 'var(--color-text-muted)' }}
                        >
                          {m.code}
                        </td>
                        <td
                          className="px-4 py-2.5 font-mono text-right"
                          style={{ color: changeColor(m.change_pct) }}
                        >
                          {m.change_pct >= 0 ? '+' : ''}
                          {m.change_pct.toFixed(2)}%
                        </td>
                        <td
                          className="px-4 py-2.5 font-mono text-right"
                          style={{ color: 'var(--color-text-secondary)' }}
                        >
                          {m.pe.toFixed(2)}
                        </td>
                        <td
                          className="px-4 py-2.5 font-mono text-right"
                          style={{ color: 'var(--color-text-secondary)' }}
                        >
                          {m.pb.toFixed(2)}
                        </td>
                        <td
                          className="px-4 py-2.5 font-mono text-right"
                          style={{ color: 'var(--color-text-secondary)' }}
                        >
                          {(m.amount / 10000).toFixed(0)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ── Backtest Compare ── */}
      {tab === 'backtest' && (
        <div>
          <div className="flex flex-wrap items-center gap-2 mb-4">
            <input
              value={btCodes}
              onChange={(e) => setBtCodes(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleBacktest()}
              placeholder="输入股票代码，逗号分隔，如 600176,000001"
              className="flex-1 min-w-[240px] max-w-md px-3 py-2 rounded-lg border text-sm outline-none focus:ring-2 focus:ring-[var(--color-accent)]"
              style={{
                background: 'var(--color-bg-card)',
                borderColor: 'var(--color-border)',
                color: 'var(--color-text-primary)',
              }}
            />
            <select
              value={btDays}
              onChange={(e) => setBtDays(Number(e.target.value))}
              className="px-3 py-2 rounded-lg border text-sm outline-none"
              style={{
                background: 'var(--color-bg-card)',
                borderColor: 'var(--color-border)',
                color: 'var(--color-text-primary)',
              }}
            >
              <option value={30}>30 天</option>
              <option value={60}>60 天</option>
              <option value={90}>90 天</option>
              <option value={180}>180 天</option>
              <option value={365}>365 天</option>
            </select>
            <button
              onClick={handleBacktest}
              disabled={btLoading}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm text-white transition-colors disabled:opacity-50"
              style={{ background: 'var(--color-accent)' }}
            >
              {btLoading ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Play className="w-3.5 h-3.5" />
              )}
              回测
            </button>
          </div>

          {btError && (
            <div
              className="flex items-center gap-2 text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
              style={{
                background: 'rgba(239,68,68,0.1)',
                color: 'var(--color-danger)',
              }}
            >
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              {btError}
            </div>
          )}

          {btResults.length > 0 && (
            <div>
              {/* Comparison table */}
              <div
                className="rounded-lg border overflow-hidden mb-6"
                style={{
                  background: 'var(--color-bg-secondary)',
                  borderColor: 'var(--color-border)',
                }}
              >
                <table className="w-full text-sm">
                  <thead>
                    <tr
                      className="text-left text-xs"
                      style={{
                        background: 'var(--color-bg-card)',
                        color: 'var(--color-text-muted)',
                      }}
                    >
                      <th className="px-4 py-2 font-medium">代码</th>
                      <th className="px-4 py-2 font-medium">名称</th>
                      <th className="px-4 py-2 font-medium text-right">信号数</th>
                      <th className="px-4 py-2 font-medium text-right">胜率</th>
                      <th className="px-4 py-2 font-medium text-right">平均收益</th>
                      <th className="px-4 py-2 font-medium text-right">最大回撤</th>
                    </tr>
                  </thead>
                  <tbody>
                    {btResults.map((r) => (
                      <tr
                        key={r.code}
                        className="border-t"
                        style={{ borderColor: 'var(--color-border)' }}
                      >
                        <td className="px-4 py-2.5 font-mono text-xs">
                          {r.code}
                        </td>
                        <td className="px-4 py-2.5">
                          {r.error ? (
                            <span style={{ color: 'var(--color-danger)' }}>
                              {r.error}
                            </span>
                          ) : (
                            r.name
                          )}
                        </td>
                        <td
                          className="px-4 py-2.5 font-mono text-right"
                          style={{ color: 'var(--color-text-secondary)' }}
                        >
                          {r.signal_count}
                        </td>
                        <td
                          className="px-4 py-2.5 font-mono text-right"
                          style={{
                            color: r.win_rate >= 50
                              ? 'var(--color-danger)'
                              : 'var(--color-success)',
                          }}
                        >
                          {r.win_rate.toFixed(1)}%
                        </td>
                        <td
                          className="px-4 py-2.5 font-mono text-right"
                          style={{ color: changeColor(r.avg_return_pct) }}
                        >
                          {r.avg_return_pct >= 0 ? '+' : ''}
                          {r.avg_return_pct.toFixed(2)}%
                        </td>
                        <td
                          className="px-4 py-2.5 font-mono text-right"
                          style={{ color: 'var(--color-success)' }}
                        >
                          -{r.max_drawdown_pct.toFixed(2)}%
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Equity curve sparklines */}
              <h2
                className="text-sm font-medium mb-3"
                style={{ color: 'var(--color-text-secondary)' }}
              >
                资金曲线
              </h2>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {btResults
                  .filter((r) => !r.error && r.equity_curve.length > 1)
                  .map((r) => (
                    <div
                      key={r.code}
                      className="rounded-lg border p-3"
                      style={{
                        background: 'var(--color-bg-secondary)',
                        borderColor: 'var(--color-border)',
                      }}
                    >
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-sm font-medium">{r.name}</span>
                        <span
                          className="text-xs font-mono"
                          style={{ color: 'var(--color-text-muted)' }}
                        >
                          {r.code}
                        </span>
                      </div>
                      <Sparkline data={r.equity_curve} />
                    </div>
                  ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function Sparkline({ data }: { data: number[] }) {
  if (data.length < 2) return null;

  const w = 280;
  const h = 80;
  const pad = 4;
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;

  const points = data.map((v, i) => {
    const x = pad + (i / (data.length - 1)) * (w - 2 * pad);
    const y = h - pad - ((v - min) / range) * (h - 2 * pad);
    return `${x},${y}`;
  });

  const isUp = data[data.length - 1] >= data[0];
  const strokeColor = isUp
    ? 'var(--color-danger, #ef4444)'
    : 'var(--color-success, #22c55e)';

  // Fill area path
  const firstX = pad;
  const lastX = pad + ((data.length - 1) / (data.length - 1)) * (w - 2 * pad);
  const fillPath = `M${firstX},${h - pad} L${points.join(' L')} L${lastX},${h - pad} Z`;

  return (
    <svg
      viewBox={`0 0 ${w} ${h}`}
      className="w-full"
      style={{ height: h }}
      preserveAspectRatio="none"
    >
      <defs>
        <linearGradient id={`grad-${data[0]}`} x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stopColor={strokeColor} stopOpacity="0.2" />
          <stop offset="100%" stopColor={strokeColor} stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={fillPath} fill={`url(#grad-${data[0]})`} />
      <polyline
        points={points.join(' ')}
        fill="none"
        stroke={strokeColor}
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
    </svg>
  );
}
