import { useState, useEffect, useMemo } from 'react';
import { tradingJournalApi, type TradeCalendarDay } from '@/lib/api';
import {
  CalendarDays,
  ChevronLeft,
  ChevronRight,
  RefreshCw,
} from 'lucide-react';
import { Skeleton } from '@/components/ui/Skeleton';

const WEEKDAYS = ['一', '二', '三', '四', '五', '六', '日'];
const MONTHS = ['一月', '二月', '三月', '四月', '五月', '六月', '七月', '八月', '九月', '十月', '十一月', '十二月'];

export default function TradeCalendarPage() {
  const [year, setYear] = useState(new Date().getFullYear());
  const [month, setMonth] = useState(new Date().getMonth() + 1);
  const [data, setData] = useState<TradeCalendarDay[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = (y: number, m: number) => {
    setLoading(true);
    setError('');
    tradingJournalApi.calendar({ year: y, month: m })
      .then((res) => setData(Array.isArray(res.data.data) ? res.data.data : Array.isArray(res.data) ? res.data : []))
      .catch(() => setError('加载交易日历失败'))
      .finally(() => setLoading(false));
  };

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { fetchData(year, month); }, [year, month]);

  const navigate = (delta: number) => {
    let newMonth = month + delta;
    let newYear = year;
    if (newMonth > 12) { newMonth = 1; newYear++; }
    if (newMonth < 1) { newMonth = 12; newYear--; }
    setMonth(newMonth);
    setYear(newYear);
  };

  const jumpToToday = () => {
    const now = new Date();
    setYear(now.getFullYear());
    setMonth(now.getMonth() + 1);
  };

  // Build calendar grid
  const calendarDays = useMemo(() => {
    const firstDay = new Date(year, month - 1, 1);
    const lastDay = new Date(year, month, 0);
    const daysInMonth = lastDay.getDate();
    // Monday=0, Sunday=6
    let startWeekday = firstDay.getDay() - 1;
    if (startWeekday < 0) startWeekday = 6;

    const days: { date: string; day: number; isCurrentMonth: boolean; trade?: TradeCalendarDay }[] = [];

    // Previous month padding
    const prevMonthLast = new Date(year, month - 1, 0).getDate();
    for (let i = startWeekday - 1; i >= 0; i--) {
      days.push({ date: '', day: prevMonthLast - i, isCurrentMonth: false });
    }

    // Current month
    for (let d = 1; d <= daysInMonth; d++) {
      const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
      const trade = data.find((t) => t.date === dateStr);
      days.push({ date: dateStr, day: d, isCurrentMonth: true, trade });
    }

    // Next month padding
    const remaining = 7 - (days.length % 7);
    if (remaining < 7) {
      for (let i = 1; i <= remaining; i++) {
        days.push({ date: '', day: i, isCurrentMonth: false });
      }
    }

    return days;
  }, [year, month, data]);

  // Summary stats
  const stats = useMemo(() => {
    const tradingDays = data.filter((d) => d.trade_count > 0);
    const totalTrades = data.reduce((s, d) => s + d.trade_count, 0);
    const totalPnl = data.reduce((s, d) => s + d.profit_loss, 0);
    const winDays = tradingDays.filter((d) => d.profit_loss > 0).length;
    const lossDays = tradingDays.filter((d) => d.profit_loss < 0).length;
    const bestDay = tradingDays.reduce((best, d) => (!best || d.profit_loss > best.profit_loss) ? d : best, null as TradeCalendarDay | null);
    const worstDay = tradingDays.reduce((worst, d) => (!worst || d.profit_loss < worst.profit_loss) ? d : worst, null as TradeCalendarDay | null);
    return { tradingDays: tradingDays.length, totalTrades, totalPnl, winDays, lossDays, bestDay, worstDay };
  }, [data]);

  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-muted)';

  // Heat intensity based on P&L
  const getHeatColor = (pnl: number, maxAbs: number) => {
    if (pnl === 0 || maxAbs === 0) return 'transparent';
    const intensity = Math.min(Math.abs(pnl) / maxAbs, 1);
    const alpha = 0.08 + intensity * 0.27; // 0.08 to 0.35
    return pnl > 0 ? `rgba(239,68,68,${alpha})` : `rgba(34,197,94,${alpha})`;
  };

  const maxAbsPnl = useMemo(() => {
    return Math.max(...data.map((d) => Math.abs(d.profit_loss)), 1);
  }, [data]);

  const isToday = (dateStr: string) => {
    const today = new Date();
    const todayStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;
    return dateStr === todayStr;
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <CalendarDays className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">交易日历</h1>
        </div>
        <button onClick={() => fetchData(year, month)} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      <p className="text-sm mb-5" style={{ color: 'var(--color-text-secondary)' }}>
        按日历热力图展示每日交易盈亏
      </p>

      {/* Navigation */}
      <div className="glass-panel rounded-xl p-4 mb-6">
        <div className="flex items-center justify-center gap-3 mb-5">
          <button
            onClick={() => navigate(-1)}
            className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
          >
            <ChevronLeft className="w-4 h-4" style={{ color: 'var(--color-text-secondary)' }} />
          </button>
          <span className="text-base font-medium min-w-[120px] text-center">
            {year}年 {MONTHS[month - 1]}
          </span>
          <button
            onClick={() => navigate(1)}
            className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
          >
            <ChevronRight className="w-4 h-4" style={{ color: 'var(--color-text-secondary)' }} />
          </button>
          <button
            onClick={jumpToToday}
            className="px-3 py-1 rounded-lg text-xs font-medium hover:bg-[var(--color-bg-hover)] transition-colors"
            style={{ border: '1px solid var(--color-border)', color: 'var(--color-text-secondary)' }}
          >
            今天
          </button>
        </div>

        {/* Legend */}
        <div className="hidden sm:flex items-center justify-center gap-2 mb-4 text-xs" style={{ color: 'var(--color-text-muted)' }}>
          <span>盈利</span>
          {[0.35, 0.2, 0.08].map((a) => (
            <div key={a} className="w-4 h-4 rounded" style={{ background: `rgba(220,38,38,${a})` }} />
          ))}
          <div className="w-4 h-4 rounded" style={{ background: 'var(--color-bg-hover)' }} />
          {[0.08, 0.2, 0.35].map((a) => (
            <div key={a} className="w-4 h-4 rounded" style={{ background: `rgba(34,197,94,${a})` }} />
          ))}
          <span>亏损</span>
        </div>

        {loading ? (
          <div className="space-y-4">
            <Skeleton /><Skeleton /><Skeleton /><Skeleton />
            <Skeleton />
          </div>
        ) : error ? (
          <div className="text-center py-10">
            <p className="text-sm" style={{ color: 'var(--color-danger)' }}>{error}</p>
          </div>
        ) : (
          <>
            {/* Weekday headers */}
            <div className="grid grid-cols-7 gap-1 mb-1">
              {WEEKDAYS.map((w) => (
                <div key={w} className="text-center text-xs font-medium py-2" style={{ color: 'var(--color-text-muted)' }}>
                  {w}
                </div>
              ))}
            </div>

            {/* Calendar grid */}
            <div className="grid grid-cols-7 gap-1">
              {calendarDays.map((d, i) => {
                const hasTrade = d.trade && d.trade.trade_count > 0;
                const bg = d.isCurrentMonth && hasTrade
                  ? getHeatColor(d.trade!.profit_loss, maxAbsPnl)
                  : 'transparent';
                const today = d.isCurrentMonth && d.date && isToday(d.date);

                return (
                  <div
                    key={i}
                    className={`
                      relative rounded-lg p-2 min-h-[60px] text-center transition-colors
                      ${d.isCurrentMonth ? '' : 'opacity-30'}
                      ${hasTrade ? 'cursor-default' : ''}
                    `}
                    style={{
                      background: bg,
                      border: today ? '1px solid var(--color-accent)' : '1px solid transparent',
                    }}
                  >
                    <div
                      className="text-xs font-medium"
                      style={{
                        color: today ? 'var(--color-accent)' : 'var(--color-text-secondary)',
                      }}
                    >
                      {d.day}
                    </div>
                    {hasTrade && (
                      <div className="mt-1">
                        <div className="text-[10px] font-mono" style={{ color: changeColor(d.trade!.profit_loss) }}>
                          {d.trade!.profit_loss >= 0 ? '+' : ''}{d.trade!.profit_loss.toFixed(0)}
                        </div>
                        <div className="text-[9px]" style={{ color: 'var(--color-text-muted)' }}>
                          {d.trade!.trade_count}笔
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </>
        )}
      </div>

      {/* Summary */}
      {!loading && !error && stats.tradingDays > 0 && (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
          <SummaryCard label="交易天数" value={String(stats.tradingDays)} />
          <SummaryCard label="总交易笔数" value={String(stats.totalTrades)} />
          <SummaryCard
            label="总盈亏"
            value={`${stats.totalPnl >= 0 ? '+' : ''}${stats.totalPnl.toFixed(0)}`}
            color={changeColor(stats.totalPnl)}
          />
          <SummaryCard label="盈利天数" value={String(stats.winDays)} color="var(--color-danger)" />
          <SummaryCard label="亏损天数" value={String(stats.lossDays)} color="var(--color-success)" />
          <SummaryCard
            label="最佳日"
            value={stats.bestDay ? `+${stats.bestDay.profit_loss.toFixed(0)}` : '-'}
            color="var(--color-danger)"
          />
        </div>
      )}

      {/* Empty state */}
      {!loading && !error && data.filter((d) => d.trade_count > 0).length === 0 && (
        <div className="text-center py-12 rounded-xl" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
          <CalendarDays className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>本月暂无交易记录</p>
          <p className="text-xs mt-1" style={{ color: 'var(--color-text-muted)' }}>请先在交易日志中添加交易记录</p>
        </div>
      )}
    </div>
  );
}

function SummaryCard({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div className="glass-panel rounded-xl p-3 text-center">
      <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>{label}</div>
      <div className="text-sm font-bold font-mono" style={{ color: color || 'var(--color-text-primary)' }}>{value}</div>
    </div>
  );
}
