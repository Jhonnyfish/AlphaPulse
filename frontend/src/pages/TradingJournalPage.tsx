import EmptyState from '@/components/EmptyState';
import { tradingJournalApi, type TradeRecord, type TradeStats, type TradeCalendarDay } from '@/lib/api';
import { useState, useEffect, useCallback, useMemo } from 'react';
import { Plus, Trash2, TrendingUp, TrendingDown, BookOpen, Calendar, BarChart3, RefreshCw, Award, Frown } from 'lucide-react';
import EChart from '@/components/charts/EChart';
import type { EChartsOption } from 'echarts';
import { SkeletonInlineTable } from '@/components/ui/Skeleton';

interface AddForm {
  code: string;
  name: string;
  direction: 'buy' | 'sell';
  price: string;
  quantity: string;
  trade_date: string;
  strategy: string;
  reason: string;
  emotion: string;
  notes: string;
}

const emptyForm: AddForm = {
  code: '',
  name: '',
  direction: 'buy',
  price: '',
  quantity: '',
  trade_date: new Date().toISOString().slice(0, 10),
  strategy: '',
  reason: '',
  emotion: '',
  notes: '',
};
const WEEKDAYS = ['日', '一', '二', '三', '四', '五', '六'];
/** Deterministic mock trades for the heatmap demo when no API data exists */
function generateMockTrades(): TradeRecord[] {
  const result: TradeRecord[] = [];
  const now = new Date();
  const codes = ['600519', '000858', '601318', '000001', '300750'];
  const names = ['贵州茅台', '五粮液', '中国平安', '平安银行', '宁德时代'];
  const strategies = ['均线突破', 'MACD金叉', '放量突破', '回调买入', '趋势跟踪'];

  for (let i = 0; i < 365; i++) {
    const d = new Date(now);
    d.setDate(d.getDate() - i);
    if (d.getDay() === 0 || d.getDay() === 6) continue;

    const seed = d.getFullYear() * 10000 + (d.getMonth() + 1) * 100 + d.getDate();
    const hash = ((seed * 2654435761) >>> 0) % 100;
    if (hash >= 62) continue;

    const count = (hash % 3) + 1;
    for (let j = 0; j < count; j++) {
      const idx = (hash + j) % 5;
      const pnlSeed = ((seed * (j + 1) * 2246822519) >>> 0) % 1000;
      const pnl = (pnlSeed - 420) * 8;
      const price = 30 + ((hash * 7) % 300);
      const qty = 100 + (hash % 10) * 100;
      const dateStr = [
        d.getFullYear(),
        String(d.getMonth() + 1).padStart(2, '0'),
        String(d.getDate()).padStart(2, '0'),
      ].join('-');

      result.push({
        id: `mock-${i}-${j}`,
        code: codes[idx],
        name: names[idx],
        direction: j % 2 === 0 ? 'buy' : 'sell',
        price,
        quantity: qty,
        amount: qty * price,
        trade_date: dateStr,
        strategy: strategies[idx],
        profit_loss: pnl,
        profit_loss_pct: pnl / (qty * price) * 100,
      });
    }
  }
  return result;
}

export default function TradingJournalPage() {
  const [trades, setTrades] = useState<TradeRecord[]>([]);
  const [stats, setStats] = useState<TradeStats | null>(null);
  const [calendarDays, setCalendarDays] = useState<TradeCalendarDay[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState<AddForm>(emptyForm);
  const [submitting, setSubmitting] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const now = new Date();
  const [calYear, setCalYear] = useState(now.getFullYear());
  const [calMonth, setCalMonth] = useState(now.getMonth() + 1);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [tradesRes, statsRes, calRes] = await Promise.allSettled([
        tradingJournalApi.list(),
        tradingJournalApi.stats(),
        tradingJournalApi.calendar({ year: calYear, month: calMonth }),
      ]);
      if (tradesRes.status === 'fulfilled') setTrades(tradesRes.value.data);
      if (statsRes.status === 'fulfilled') setStats(statsRes.value.data);
      if (calRes.status === 'fulfilled') setCalendarDays(calRes.value.data);
    } catch {
      setError('加载数据失败');
    } finally {
      setLoading(false);
    }
  }, [calYear, calMonth]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleAdd = async () => {
    if (!form.code || !form.price || !form.quantity || !form.trade_date) return;
    setSubmitting(true);
    try {
      const price = Number(form.price);
      const quantity = Number(form.quantity);
      const parts: string[] = [];
      if (form.reason) parts.push(`原因: ${form.reason}`);
      if (form.emotion) parts.push(`情绪: ${form.emotion}`);
      if (form.notes) parts.push(form.notes);

      await tradingJournalApi.create({
        code: form.code.trim().toUpperCase(),
        name: form.name.trim(),
        direction: form.direction,
        price,
        quantity,
        amount: price * quantity,
        trade_date: form.trade_date,
        profit_loss: 0,
        profit_loss_pct: 0,
        strategy: form.strategy.trim(),
        notes: parts.length > 0 ? parts.join(' | ') : undefined,
      });
      setShowModal(false);
      setForm(emptyForm);
      await fetchData();
    } catch {
      setError('添加交易失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    setDeletingId(id);
    try {
      await tradingJournalApi.remove(id);
      await fetchData();
    } catch {
      setError('删除交易失败');
    } finally {
      setDeletingId(null);
    }
  };

  const pnlColor = (n: number | undefined) => {
    const v = n ?? 0;
    return v > 0
      ? 'var(--color-danger)'
      : v < 0
        ? 'var(--color-success)'
        : 'var(--color-text-secondary)';
  };

  const formatPct = (n: number | undefined) => { const v = n ?? 0; return `${v >= 0 ? '+' : ''}${v.toFixed(2)}%`; };
  const formatNum = (n: number | undefined) => {
    const v = n ?? 0;
    return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  };

  // Calendar helpers
  const firstDay = new Date(calYear, calMonth - 1, 1).getDay();
  const daysInMonth = new Date(calYear, calMonth, 0).getDate();
  const calMap = new Map(calendarDays.map((d) => [d.date, d]));

  const prevMonth = () => {
    if (calMonth === 1) { setCalMonth(12); setCalYear((y) => y - 1); }
    else setCalMonth((m) => m - 1);
  };
  const nextMonth = () => {
    if (calMonth === 12) { setCalMonth(1); setCalYear((y) => y + 1); }
    else setCalMonth((m) => m + 1);
  };

  const calendarCells: (number | null)[] = [];
  for (let i = 0; i < firstDay; i++) calendarCells.push(null);
  for (let d = 1; d <= daysInMonth; d++) calendarCells.push(d);

  // ── ECharts Calendar Heatmap (12-month rolling P&L) ────────────
  const heatmapData = useMemo(() => {
    const dailyMap: Record<string, { pnl: number; pnlPct: number[]; count: number }> = {};
    const src = trades.length > 0 ? trades : generateMockTrades();

    src.forEach((t) => {
      const date = t.trade_date;
      if (!dailyMap[date]) dailyMap[date] = { pnl: 0, pnlPct: [], count: 0 };
      dailyMap[date].pnl += t.profit_loss ?? 0;
      if (t.profit_loss_pct != null) dailyMap[date].pnlPct.push(t.profit_loss_pct);
      dailyMap[date].count += 1;
    });

    return { dailyMap, isMock: trades.length === 0 };
  }, [trades]);

  const heatmapOption = useMemo<EChartsOption>(() => {
    const { dailyMap } = heatmapData;
    const data: [string, number][] = [];
    const vals: number[] = [];
    for (const [date, info] of Object.entries(dailyMap)) {
      const rounded = Math.round(info.pnl * 100) / 100;
      data.push([date, rounded]);
      vals.push(rounded);
    }

    const minV = vals.length ? Math.min(...vals) : -1;
    const maxV = vals.length ? Math.max(...vals) : 1;
    const absMax = Math.max(Math.abs(minV), Math.abs(maxV), 1);

    const endDate = new Date();
    const startDate = new Date(endDate);
    startDate.setFullYear(startDate.getFullYear() - 1);
    const fmt = (d: Date) =>
      `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
    const range: [string, string] = [fmt(startDate), fmt(endDate)];

    return {
      tooltip: {
        backgroundColor: 'rgba(15,23,42,0.95)',
        borderColor: 'rgba(148,163,184,0.2)',
        textStyle: { color: '#e2e8f0', fontSize: 12 },
        formatter(params: any) {
          if (!params.data) return '';
          const [date, value] = params.data as [string, number];
          const info = dailyMap[date];
          if (!info) return date;
          const clr = value > 0 ? '#ef4444' : value < 0 ? '#22c55e' : '#94a3b8';
          const sign = value > 0 ? '+' : '';
          const avgPct =
            info.pnlPct.length > 0
              ? info.pnlPct.reduce((a: number, b: number) => a + b, 0) / info.pnlPct.length
              : 0;
          return [
            `<div style="font-weight:600;margin-bottom:4px">${date}</div>`,
            `<div>交易: <b>${info.count}</b> 笔</div>`,
            `<div>盈亏: <span style="color:${clr};font-weight:600">${sign}¥${value.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span></div>`,
            `<div>均收益: <span style="color:${clr}">${sign}${avgPct.toFixed(2)}%</span></div>`,
          ].join('');
        },
      },
      visualMap: {
        min: -absMax,
        max: absMax,
        calculable: true,
        orient: 'horizontal',
        left: 'center',
        bottom: 0,
        itemWidth: 12,
        itemHeight: 100,
        text: [`+${Math.round(absMax).toLocaleString()}`, `-${Math.round(absMax).toLocaleString()}`],
        textStyle: { color: '#94a3b8', fontSize: 10 },
        inRange: {
          color: ['#22c55e', '#4ade80', '#86efac', '#2a2d3a', '#fca5a5', '#f87171', '#ef4444'],
        },
        outOfRange: { color: '#1a1d2e' },
      },
      calendar: {
        top: 28,
        left: 40,
        right: 30,
        bottom: 50,
        cellSize: ['auto', 14],
        range,
        splitLine: { show: true, lineStyle: { color: '#334155', width: 1.5 } },
        itemStyle: {
          borderWidth: 0.5,
          borderColor: 'rgba(30,41,59,0.8)',
          color: '#1a1d2e',
        },
        yearLabel: { show: true, color: '#94a3b8', fontSize: 12 },
        monthLabel: { color: '#94a3b8', fontSize: 11 },
        dayLabel: {
          firstDay: 1,
          nameMap: ['日', '一', '二', '三', '四', '五', '六'],
          color: '#64748b',
          fontSize: 10,
        },
      },
      series: [
        {
          type: 'heatmap',
          coordinateSystem: 'calendar',
          calendarIndex: 0,
          data,
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowColor: 'rgba(0, 0, 0, 0.5)',
              borderColor: '#3b82f6',
              borderWidth: 1,
            },
          },
        },
      ],
    } as EChartsOption;
  }, [heatmapData]);

  // ── Profit Distribution Histogram ──────────────────────────
  const histogramOption = useMemo<EChartsOption>(() => {
    const src = trades.length > 0 ? trades : generateMockTrades();

    const binDefs: { min: number; max: number; label: string }[] = [];
    for (let i = -6; i < 6; i++) {
      binDefs.push({ min: i, max: i + 1, label: `${i}%~${i + 1}%` });
    }

    const counts = new Array(binDefs.length).fill(0);
    src.forEach((t) => {
      const pct = t.profit_loss_pct ?? 0;
      for (let b = 0; b < binDefs.length; b++) {
        if (pct >= binDefs[b].min && pct < binDefs[b].max) {
          counts[b]++;
          break;
        }
      }
    });

    const barData = binDefs.map((bin, i) => ({
      value: counts[i],
      itemStyle: {
        color:
          bin.max <= 0
            ? { type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#ef4444' }, { offset: 1, color: '#dc2626' }] }
            : bin.min >= 0
              ? { type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#22c55e' }, { offset: 1, color: '#16a34a' }] }
              : { type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#ef4444' }, { offset: 1, color: '#dc2626' }] },
        borderRadius: [3, 3, 0, 0],
      },
    }));

    return {
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(15,23,42,0.95)',
        borderColor: 'rgba(148,163,184,0.2)',
        textStyle: { color: '#e2e8f0', fontSize: 12 },
        formatter(params: any) {
          const p = params[0];
          return `<div style="font-weight:600">${p.name}</div><div>交易次数: <b>${p.value}</b></div>`;
        },
      },
      grid: { top: 20, right: 20, bottom: 40, left: 50 },
      xAxis: {
        type: 'category',
        data: binDefs.map((b) => b.label),
        axisLabel: { color: '#94a3b8', fontSize: 10, rotate: 30 },
        axisLine: { lineStyle: { color: '#334155' } },
      },
      yAxis: {
        type: 'value',
        axisLabel: { color: '#94a3b8', fontSize: 10 },
        splitLine: { lineStyle: { color: 'rgba(51,65,85,0.5)' } },
      },
      series: [
        {
          type: 'bar',
          data: barData,
          barWidth: '60%',
        },
      ],
    } as EChartsOption;
  }, [trades]);

  // ── Monthly Statistics ──────────────────────────────────────
  const monthlyStats = useMemo(() => {
    const src = trades.length > 0 ? trades : generateMockTrades();
    const monthMap: Record<string, { count: number; totalPnl: number; wins: number; pnlPcts: number[] }> = {};

    src.forEach((t) => {
      const month = t.trade_date.slice(0, 7);
      if (!monthMap[month]) monthMap[month] = { count: 0, totalPnl: 0, wins: 0, pnlPcts: [] };
      monthMap[month].count++;
      monthMap[month].totalPnl += t.profit_loss ?? 0;
      if ((t.profit_loss ?? 0) > 0) monthMap[month].wins++;
      if (t.profit_loss_pct != null) monthMap[month].pnlPcts.push(t.profit_loss_pct);
    });

    return Object.entries(monthMap)
      .sort(([a], [b]) => a.localeCompare(b))
      .slice(-12)
      .map(([month, data]) => ({
        month,
        count: data.count,
        totalPnl: data.totalPnl,
        winRate: data.count > 0 ? (data.wins / data.count) * 100 : 0,
        avgReturn:
          data.pnlPcts.length > 0
            ? data.pnlPcts.reduce((a, b) => a + b, 0) / data.pnlPcts.length
            : 0,
      }));
  }, [trades]);

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <BookOpen className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">交易日志</h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowModal(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors"
            style={{ background: 'var(--color-accent)', color: '#fff' }}
          >
            <Plus className="w-3.5 h-3.5" />
            添加交易
          </button>
          <button
            onClick={fetchData}
            disabled={loading}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm hover:bg-[var(--color-bg-hover)] transition-colors"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </button>
        </div>
      </div>

      {error && (
        <div
          className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
        >
          {error}
        </div>
      )}

      {/* Statistics cards */}
      {stats && (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-6">
          <div
            className="rounded-xl border p-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <div className="flex items-center gap-1.5 text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>
              <BarChart3 className="w-3 h-3" />
              总交易
            </div>
            <div className="text-lg font-bold font-mono">{stats.total_trades}</div>
          </div>
          <div
            className="rounded-xl border p-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <div className="flex items-center gap-1.5 text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>
              <Award className="w-3 h-3" />
              胜率
            </div>
            <div className="text-lg font-bold font-mono" style={{ color: pnlColor(stats.win_rate * 2 - 1) }}>
              {(stats.win_rate * 100).toFixed(1)}%
            </div>
          </div>
          <div
            className="rounded-xl border p-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <div className="flex items-center gap-1.5 text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>
              <TrendingUp className="w-3 h-3" style={{ color: 'var(--color-danger)' }} />
              平均盈利%
            </div>
            <div className="text-lg font-bold font-mono" style={{ color: 'var(--color-danger)' }}>
              {formatPct(stats.avg_profit_pct)}
            </div>
          </div>
          <div
            className="rounded-xl border p-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <div className="flex items-center gap-1.5 text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>
              <TrendingDown className="w-3 h-3" style={{ color: 'var(--color-success)' }} />
              平均亏损%
            </div>
            <div className="text-lg font-bold font-mono" style={{ color: 'var(--color-success)' }}>
              {formatPct(stats.avg_loss_pct)}
            </div>
          </div>
          <div
            className="rounded-xl border p-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>总盈亏</div>
            <div className="text-lg font-bold font-mono" style={{ color: pnlColor(stats.total_profit_loss) }}>
              {formatNum(stats.total_profit_loss)}
            </div>
          </div>
          <div
            className="rounded-xl border p-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>盈亏比</div>
            <div className="text-lg font-bold font-mono">{stats.profit_factor.toFixed(2)}</div>
          </div>
        </div>
      )}

      {/* ── ECharts Calendar Heatmap ─────────────────────────── */}
      <div
        className="rounded-xl border p-4 mb-6"
        style={{
          background: 'var(--color-bg-secondary)',
          borderColor: 'var(--color-border)',
          backdropFilter: 'blur(12px)',
        }}
      >
        <div className="flex items-center gap-1.5 mb-1">
          <Calendar className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <span className="text-sm font-medium">交易热力图</span>
          <span className="text-xs ml-1" style={{ color: 'var(--color-text-muted)' }}>
            — 近12个月每日盈亏
          </span>
          {heatmapData.isMock && (
            <span
              className="text-[10px] ml-auto px-2 py-0.5 rounded-full"
              style={{ background: 'rgba(59,130,246,0.15)', color: 'var(--color-accent)' }}
            >
              示例数据
            </span>
          )}
        </div>
        <EChart option={heatmapOption} height={230} loading={loading} />
        <div className="flex items-center justify-center gap-4 mt-1">
          <span className="flex items-center gap-1.5 text-[10px]" style={{ color: 'var(--color-text-muted)' }}>
            <span className="inline-block w-2.5 h-2.5 rounded-sm" style={{ background: '#22c55e' }} />
            亏损
          </span>
          <span className="flex items-center gap-1.5 text-[10px]" style={{ color: 'var(--color-text-muted)' }}>
            <span className="inline-block w-2.5 h-2.5 rounded-sm" style={{ background: '#2a2d3a' }} />
            无交易
          </span>
          <span className="flex items-center gap-1.5 text-[10px]" style={{ color: 'var(--color-text-muted)' }}>
            <span className="inline-block w-2.5 h-2.5 rounded-sm" style={{ background: '#ef4444' }} />
            盈利
          </span>
        </div>
      </div>

      {/* ── Profit Distribution Histogram ────────────────────── */}
      <div
        className="rounded-xl border p-4 mb-6"
        style={{
          background: 'var(--color-bg-secondary)',
          borderColor: 'var(--color-border)',
          backdropFilter: 'blur(12px)',
        }}
      >
        <div className="flex items-center gap-1.5 mb-1">
          <BarChart3 className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <span className="text-sm font-medium">收益分布</span>
          <span className="text-xs ml-1" style={{ color: 'var(--color-text-muted)' }}>
            — 收益率区间分布
          </span>
          {heatmapData.isMock && (
            <span
              className="text-[10px] ml-auto px-2 py-0.5 rounded-full"
              style={{ background: 'rgba(59,130,246,0.15)', color: 'var(--color-accent)' }}
            >
              示例数据
            </span>
          )}
        </div>
        <EChart option={histogramOption} height={240} loading={loading} />
        <div className="flex items-center justify-center gap-4 mt-1">
          <span className="flex items-center gap-1.5 text-[10px]" style={{ color: 'var(--color-text-muted)' }}>
            <span className="inline-block w-2.5 h-2.5 rounded-sm" style={{ background: '#ef4444' }} />
            负收益
          </span>
          <span className="flex items-center gap-1.5 text-[10px]" style={{ color: 'var(--color-text-muted)' }}>
            <span className="inline-block w-2.5 h-2.5 rounded-sm" style={{ background: '#22c55e' }} />
            正收益
          </span>
        </div>
      </div>

      {/* ── Monthly Statistics ──────────────────────────────── */}
      <div
        className="rounded-xl border p-4 mb-6"
        style={{
          background: 'var(--color-bg-secondary)',
          borderColor: 'var(--color-border)',
          backdropFilter: 'blur(12px)',
        }}
      >
        <div className="flex items-center gap-1.5 mb-4">
          <Calendar className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <span className="text-sm font-medium">月度统计</span>
          {heatmapData.isMock && (
            <span
              className="text-[10px] ml-auto px-2 py-0.5 rounded-full"
              style={{ background: 'rgba(59,130,246,0.15)', color: 'var(--color-accent)' }}
            >
              示例数据
            </span>
          )}
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
          {monthlyStats.map((ms) => (
            <div
              key={ms.month}
              className="rounded-lg border p-3 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg"
              style={{
                background: 'rgba(15, 23, 42, 0.4)',
                borderColor: 'var(--color-border)',
                backdropFilter: 'blur(8px)',
              }}
            >
              <div className="text-sm font-semibold mb-2 font-mono" style={{ color: 'var(--color-accent)' }}>
                {ms.month}
              </div>
              <div className="space-y-1.5 text-xs">
                <div className="flex justify-between items-center">
                  <span style={{ color: 'var(--color-text-muted)' }}>交易次数</span>
                  <span className="font-mono font-medium">{ms.count}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span style={{ color: 'var(--color-text-muted)' }}>总盈亏</span>
                  <span className="font-mono font-medium" style={{ color: pnlColor(ms.totalPnl) }}>
                    {formatNum(ms.totalPnl)}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span style={{ color: 'var(--color-text-muted)' }}>胜率</span>
                  <span
                    className="font-mono font-medium"
                    style={{
                      color:
                        ms.winRate > 50
                          ? 'var(--color-success)'
                          : ms.winRate < 50
                            ? 'var(--color-danger)'
                            : 'var(--color-text-secondary)',
                    }}
                  >
                    {ms.winRate.toFixed(1)}%
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span style={{ color: 'var(--color-text-muted)' }}>平均收益率</span>
                  <span className="font-mono font-medium" style={{ color: pnlColor(ms.avgReturn) }}>
                    {formatPct(ms.avgReturn)}
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        {/* Calendar */}
        <div
          className="rounded-xl border p-4 lg:col-span-1"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-1.5">
              <Calendar className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
              <span className="text-sm font-medium">交易日历</span>
            </div>
            <div className="flex items-center gap-1">
              <button
                onClick={prevMonth}
                className="px-2 py-0.5 rounded text-xs hover:bg-[var(--color-bg-hover)] transition-colors"
                style={{ color: 'var(--color-text-muted)' }}
              >
                &lt;
              </button>
              <span className="text-xs font-mono" style={{ color: 'var(--color-text-secondary)' }}>
                {calYear}-{String(calMonth).padStart(2, '0')}
              </span>
              <button
                onClick={nextMonth}
                className="px-2 py-0.5 rounded text-xs hover:bg-[var(--color-bg-hover)] transition-colors"
                style={{ color: 'var(--color-text-muted)' }}
              >
                &gt;
              </button>
            </div>
          </div>

          {/* Weekday headers */}
          <div className="grid grid-cols-7 gap-1 mb-1">
            {WEEKDAYS.map((d) => (
              <div
                key={d}
                className="text-center text-[10px] py-1"
                style={{ color: 'var(--color-text-muted)' }}
              >
                {d}
              </div>
            ))}
          </div>

          {/* Day cells */}
          <div className="grid grid-cols-7 gap-1">
            {calendarCells.map((day, i) => {
              if (day === null) return <div key={`e${i}`} />;
              const dateStr = `${calYear}-${String(calMonth).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
              const calDay = calMap.get(dateStr);
              const isToday =
                calYear === now.getFullYear() &&
                calMonth === now.getMonth() + 1 &&
                day === now.getDate();

              let cellBg = 'transparent';
              let cellColor = 'var(--color-text-secondary)';
              if (calDay) {
                if (calDay.profit_loss > 0) {
                  cellBg = 'rgba(239,68,68,0.15)';
                  cellColor = 'var(--color-danger)';
                } else if (calDay.profit_loss < 0) {
                  cellBg = 'rgba(34,197,94,0.15)';
                  cellColor = 'var(--color-success)';
                } else {
                  cellBg = 'rgba(59,130,246,0.1)';
                  cellColor = 'var(--color-accent)';
                }
              }

              return (
                <div
                  key={day}
                  className="text-center py-1.5 rounded-md text-xs relative"
                  style={{
                    background: cellBg,
                    color: cellColor,
                    fontWeight: isToday ? 700 : calDay ? 500 : 400,
                    outline: isToday ? '1px solid var(--color-accent)' : undefined,
                  }}
                  title={calDay ? `${dateStr} | 交易${calDay.trade_count}笔 | 盈亏 ${formatNum(calDay.profit_loss)}` : dateStr}
                >
                  {day}
                  {calDay && calDay.trade_count > 0 && (
                    <div className="text-[8px] leading-none mt-0.5">{calDay.trade_count}笔</div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* Best / worst trade cards */}
        {stats && (stats.best_trade || stats.worst_trade) && (
          <div className="lg:col-span-2 grid grid-cols-1 sm:grid-cols-2 gap-3">
            {stats.best_trade && (
              <div
                className="rounded-xl border p-4"
                style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
              >
                <div className="flex items-center gap-2 mb-3">
                  <TrendingUp className="w-4 h-4" style={{ color: 'var(--color-danger)' }} />
                  <span className="text-sm font-medium">最佳交易</span>
                </div>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-mono" style={{ color: 'var(--color-accent)' }}>
                        {stats.best_trade.code}
                      </span>
                      <span className="text-sm">{stats.best_trade.name}</span>
                    </div>
                    <span className="font-mono text-sm font-bold" style={{ color: 'var(--color-danger)' }}>
                      {formatPct(stats.best_trade.profit_loss_pct)}
                    </span>
                  </div>
                  <div className="text-xs space-y-1" style={{ color: 'var(--color-text-muted)' }}>
                    <div className="flex justify-between">
                      <span>方向</span>
                      <span style={{ color: 'var(--color-text-secondary)' }}>
                        {stats.best_trade.direction === 'buy' ? '买入' : '卖出'}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span>盈亏</span>
                      <span className="font-mono" style={{ color: 'var(--color-danger)' }}>
                        {formatNum(stats.best_trade.profit_loss)}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span>策略</span>
                      <span style={{ color: 'var(--color-text-secondary)' }}>
                        {stats.best_trade.strategy || '-'}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span>日期</span>
                      <span className="font-mono" style={{ color: 'var(--color-text-secondary)' }}>
                        {stats.best_trade.trade_date}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            )}
            {stats.worst_trade && (
              <div
                className="rounded-xl border p-4"
                style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
              >
                <div className="flex items-center gap-2 mb-3">
                  <Frown className="w-4 h-4" style={{ color: 'var(--color-success)' }} />
                  <span className="text-sm font-medium">最差交易</span>
                </div>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-mono" style={{ color: 'var(--color-accent)' }}>
                        {stats.worst_trade.code}
                      </span>
                      <span className="text-sm">{stats.worst_trade.name}</span>
                    </div>
                    <span className="font-mono text-sm font-bold" style={{ color: 'var(--color-success)' }}>
                      {formatPct(stats.worst_trade.profit_loss_pct)}
                    </span>
                  </div>
                  <div className="text-xs space-y-1" style={{ color: 'var(--color-text-muted)' }}>
                    <div className="flex justify-between">
                      <span>方向</span>
                      <span style={{ color: 'var(--color-text-secondary)' }}>
                        {stats.worst_trade.direction === 'buy' ? '买入' : '卖出'}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span>盈亏</span>
                      <span className="font-mono" style={{ color: 'var(--color-success)' }}>
                        {formatNum(stats.worst_trade.profit_loss)}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span>策略</span>
                      <span style={{ color: 'var(--color-text-secondary)' }}>
                        {stats.worst_trade.strategy || '-'}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span>日期</span>
                      <span className="font-mono" style={{ color: 'var(--color-text-secondary)' }}>
                        {stats.worst_trade.trade_date}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Trades table */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
      >
        <div className="px-4 py-3 flex items-center justify-between border-b" style={{ borderColor: 'var(--color-border)' }}>
          <span className="text-sm font-medium">交易记录</span>
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
            {trades.length} 条记录
          </span>
        </div>

        {trades.length === 0 ? (
          loading ? (
            <div className="p-8"><SkeletonInlineTable rows={5} columns={7} /></div>
          ) : (
            <EmptyState
              icon={BookOpen}
              title="暂无交易记录"
              description="记录您的每一笔交易，追踪投资表现"
              actionLabel="添加交易"
              onAction={() => setShowModal(true)}
            />
          )
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr
                  className="text-xs"
                  style={{ color: 'var(--color-text-muted)', borderBottom: '1px solid var(--color-border)' }}
                >
                  <th className="text-left px-4 py-2.5 font-medium">代码</th>
                  <th className="text-left px-4 py-2.5 font-medium">名称</th>
                  <th className="text-center px-4 py-2.5 font-medium">方向</th>
                  <th className="text-right px-4 py-2.5 font-medium">价格</th>
                  <th className="text-right px-4 py-2.5 font-medium">数量</th>
                  <th className="text-right px-4 py-2.5 font-medium">金额</th>
                  <th className="text-left px-4 py-2.5 font-medium">日期</th>
                  <th className="text-right px-4 py-2.5 font-medium">盈亏</th>
                  <th className="text-right px-4 py-2.5 font-medium">盈亏%</th>
                  <th className="text-left px-4 py-2.5 font-medium">策略</th>
                  <th className="text-left px-4 py-2.5 font-medium">备注</th>
                  <th className="text-center px-4 py-2.5 font-medium">操作</th>
                </tr>
              </thead>
              <tbody>
                {trades.map((t) => (
                  <tr
                    key={t.id}
                    className="transition-colors hover:bg-[var(--color-bg-hover)]"
                    style={{ borderBottom: '1px solid var(--color-border)' }}
                  >
                    <td className="px-4 py-2.5 font-mono text-xs" style={{ color: 'var(--color-accent)' }}>
                      {t.code}
                    </td>
                    <td className="px-4 py-2.5">{t.name}</td>
                    <td className="px-4 py-2.5 text-center">
                      <span
                        className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
                        style={{
                          background: t.direction === 'buy'
                            ? 'rgba(239,68,68,0.15)'
                            : 'rgba(34,197,94,0.15)',
                          color: t.direction === 'buy'
                            ? 'var(--color-danger)'
                            : 'var(--color-success)',
                        }}
                      >
                        {t.direction === 'buy' ? (
                          <><TrendingUp className="w-3 h-3" /> 买入</>
                        ) : (
                          <><TrendingDown className="w-3 h-3" /> 卖出</>
                        )}
                      </span>
                    </td>
                    <td className="px-4 py-2.5 text-right font-mono">{t.price.toFixed(2)}</td>
                    <td className="px-4 py-2.5 text-right font-mono">{t.quantity}</td>
                    <td className="px-4 py-2.5 text-right font-mono">{formatNum(t.amount)}</td>
                    <td className="px-4 py-2.5 font-mono text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                      {t.trade_date}
                    </td>
                    <td className="px-4 py-2.5 text-right font-mono font-medium" style={{ color: pnlColor(t.profit_loss) }}>
                      {formatNum(t.profit_loss)}
                    </td>
                    <td className="px-4 py-2.5 text-right font-mono font-medium" style={{ color: pnlColor(t.profit_loss_pct) }}>
                      {formatPct(t.profit_loss_pct)}
                    </td>
                    <td className="px-4 py-2.5 text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                      {t.strategy || '-'}
                    </td>
                    <td className="px-4 py-2.5 text-xs max-w-[120px] truncate" style={{ color: 'var(--color-text-muted)' }}>
                      {t.notes || '-'}
                    </td>
                    <td className="px-4 py-2.5 text-center">
                      <button
                        onClick={() => handleDelete(t.id)}
                        disabled={deletingId === t.id}
                        className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors disabled:opacity-50"
                        style={{ color: 'var(--color-text-muted)' }}
                        title="删除"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Add trade modal */}
      {showModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center"
          style={{ background: 'rgba(0,0,0,0.6)' }}
          onClick={(e) => { if (e.target === e.currentTarget) setShowModal(false); }}
        >
          <div
            className="rounded-xl border p-6 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <h2 className="text-base font-bold mb-4">添加交易</h2>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>股票代码 *</label>
                  <input
                    value={form.code}
                    onChange={(e) => setForm((f) => ({ ...f, code: e.target.value }))}
                    placeholder="如 600519"
                    className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                    style={{
                      background: 'var(--color-bg-primary)',
                      border: '1px solid var(--color-border)',
                      color: 'var(--color-text-primary)',
                    }}
                  />
                </div>
                <div>
                  <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>股票名称</label>
                  <input
                    value={form.name}
                    onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                    placeholder="如 贵州茅台"
                    className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                    style={{
                      background: 'var(--color-bg-primary)',
                      border: '1px solid var(--color-border)',
                      color: 'var(--color-text-primary)',
                    }}
                  />
                </div>
              </div>

              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>方向 *</label>
                  <select
                    value={form.direction}
                    onChange={(e) => setForm((f) => ({ ...f, direction: e.target.value as 'buy' | 'sell' }))}
                    className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                    style={{
                      background: 'var(--color-bg-primary)',
                      border: '1px solid var(--color-border)',
                      color: 'var(--color-text-primary)',
                    }}
                  >
                    <option value="buy">买入</option>
                    <option value="sell">卖出</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>价格 *</label>
                  <input
                    type="number"
                    step="0.01"
                    value={form.price}
                    onChange={(e) => setForm((f) => ({ ...f, price: e.target.value }))}
                    placeholder="元"
                    className="w-full px-3 py-2 rounded-lg text-sm outline-none font-mono"
                    style={{
                      background: 'var(--color-bg-primary)',
                      border: '1px solid var(--color-border)',
                      color: 'var(--color-text-primary)',
                    }}
                  />
                </div>
                <div>
                  <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>数量 *</label>
                  <input
                    type="number"
                    value={form.quantity}
                    onChange={(e) => setForm((f) => ({ ...f, quantity: e.target.value }))}
                    placeholder="股"
                    className="w-full px-3 py-2 rounded-lg text-sm outline-none font-mono"
                    style={{
                      background: 'var(--color-bg-primary)',
                      border: '1px solid var(--color-border)',
                      color: 'var(--color-text-primary)',
                    }}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>交易日期 *</label>
                  <input
                    type="date"
                    value={form.trade_date}
                    onChange={(e) => setForm((f) => ({ ...f, trade_date: e.target.value }))}
                    className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                    style={{
                      background: 'var(--color-bg-primary)',
                      border: '1px solid var(--color-border)',
                      color: 'var(--color-text-primary)',
                    }}
                  />
                </div>
                <div>
                  <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>策略</label>
                  <input
                    value={form.strategy}
                    onChange={(e) => setForm((f) => ({ ...f, strategy: e.target.value }))}
                    placeholder="如 均线突破"
                    className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                    style={{
                      background: 'var(--color-bg-primary)',
                      border: '1px solid var(--color-border)',
                      color: 'var(--color-text-primary)',
                    }}
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>交易原因</label>
                <input
                  value={form.reason}
                  onChange={(e) => setForm((f) => ({ ...f, reason: e.target.value }))}
                  placeholder="为什么做这笔交易"
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                  style={{
                    background: 'var(--color-bg-primary)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                />
              </div>

              <div>
                <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>情绪状态</label>
                <input
                  value={form.emotion}
                  onChange={(e) => setForm((f) => ({ ...f, emotion: e.target.value }))}
                  placeholder="如 冷静、贪婪、恐慌"
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                  style={{
                    background: 'var(--color-bg-primary)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                />
              </div>

              <div>
                <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>备注</label>
                <textarea
                  value={form.notes}
                  onChange={(e) => setForm((f) => ({ ...f, notes: e.target.value }))}
                  placeholder="其他备注..."
                  rows={2}
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none resize-none"
                  style={{
                    background: 'var(--color-bg-primary)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                />
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 mt-5">
              <button
                onClick={() => { setShowModal(false); setForm(emptyForm); }}
                className="px-4 py-2 rounded-lg text-sm transition-colors hover:bg-[var(--color-bg-hover)]"
                style={{ color: 'var(--color-text-secondary)' }}
              >
                取消
              </button>
              <button
                onClick={handleAdd}
                disabled={submitting || !form.code || !form.price || !form.quantity || !form.trade_date}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50"
                style={{ background: 'var(--color-accent)', color: '#fff' }}
              >
                {submitting ? '提交中...' : '确认添加'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
