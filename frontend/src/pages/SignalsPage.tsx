import { useState, useEffect, useCallback, useMemo } from 'react';
import { signalApi, alertsApi, type SignalEvent, type Anomaly, type Alert } from '@/lib/api';
import { Radio, AlertTriangle, Bell, Calendar, RefreshCw, PieChart, TrendingUp, Activity } from 'lucide-react';
import EmptyState from '@/components/EmptyState';
import ErrorState from '@/components/ErrorState';
import { SkeletonInlineTable } from '@/components/ui/Skeleton';
import EChart from '@/components/charts/EChart';
import type { EChartsOption } from 'echarts';

/* ────────────────────── Mock Data ────────────────────── */

/** 信号分类统计 mock */
const MOCK_SIGNAL_STATS = [
  { name: '买入信号', value: 128 },
  { name: '卖出信号', value: 76 },
  { name: '持有信号', value: 215 },
  { name: '观望信号', value: 53 },
];

/** 信号颜色: 绿买/红卖/蓝持有/灰观望 */
const SIGNAL_COLORS = ['#22c55e', '#ef4444', '#3b82f6', '#64748b'];

/** 生成近30天信号强度趋势 mock 数据 */
function generateMockTrendData() {
  const dates: string[] = [];
  const values: number[] = [];
  const now = new Date();
  let base = 62;
  for (let i = 29; i >= 0; i--) {
    const d = new Date(now);
    d.setDate(d.getDate() - i);
    const mm = String(d.getMonth() + 1).padStart(2, '0');
    const dd = String(d.getDate()).padStart(2, '0');
    dates.push(`${mm}-${dd}`);
    base += (Math.random() - 0.48) * 6;
    base = Math.max(30, Math.min(95, base));
    values.push(Math.round(base * 10) / 10);
  }
  return { dates, values };
}

const MOCK_TREND = generateMockTrendData();

/* ────────────────────── Tabs & helpers ────────────────────── */

const TABS = ['信号日历', '信号历史', '异常检测', '系统告警'] as const;
type Tab = (typeof TABS)[number];

const DAYS_OPTIONS = [1, 3, 5, 10, 30] as const;

const severityStyle = (severity: string) => {
  switch (severity) {
    case 'high':
      return { bg: 'rgba(239,68,68,0.15)', color: 'var(--color-danger)' };
    case 'medium':
      return { bg: 'rgba(234,179,8,0.15)', color: 'var(--color-warning)' };
    case 'low':
      return { bg: 'rgba(34,197,94,0.15)', color: 'var(--color-success)' };
    default:
      return { bg: 'rgba(107,114,128,0.15)', color: 'var(--color-text-muted)' };
  }
};

const severityLabel = (s: string) => (s === 'high' ? '高' : s === 'medium' ? '中' : s === 'low' ? '低' : s);

/* ────────────────────── Chart option builders ────────────────────── */

/** 信号统计饼图 (圆环) */
function buildPieOption(): EChartsOption {
  const total = MOCK_SIGNAL_STATS.reduce((s, d) => s + d.value, 0);
  return {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)',
    },
    legend: {
      orient: 'vertical',
      right: '5%',
      top: 'center',
      textStyle: { color: '#94a3b8', fontSize: 12 },
      itemWidth: 10,
      itemHeight: 10,
      itemGap: 14,
    },
    series: [
      {
        type: 'pie',
        radius: ['48%', '72%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: false,
        padAngle: 2,
        itemStyle: {
          borderRadius: 6,
          borderColor: 'rgba(15,23,42,0.8)',
          borderWidth: 2,
        },
        label: {
          show: true,
          position: 'center',
          formatter: [`{total|${total}}`, '{sub|信号总数}'].join('\n'),
          rich: {
            total: {
              fontSize: 22,
              fontWeight: 700,
              color: '#f1f5f9',
              lineHeight: 30,
            },
            sub: {
              fontSize: 12,
              color: '#64748b',
              lineHeight: 20,
            },
          },
        },
        emphasis: {
          label: { show: true },
          itemStyle: {
            shadowBlur: 20,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0,0,0,0.4)',
          },
        },
        labelLine: { show: false },
        data: MOCK_SIGNAL_STATS.map((d, i) => ({
          name: d.name,
          value: d.value,
          itemStyle: { color: SIGNAL_COLORS[i] },
        })),
      },
    ],
  };
}

/** 信号强度趋势折线图 */
function buildTrendOption(): EChartsOption {
  return {
    tooltip: {
      trigger: 'axis',
      formatter: (params: unknown) => {
        const list = params as Array<{ name: string; value: number; marker: string }>;
        if (!list.length) return '';
        const p = list[0];
        return `${p.name}<br/>${p.marker} 信号强度: <b>${p.value}</b>`;
      },
    },
    grid: { left: '4%', right: '4%', top: '12%', bottom: '10%', containLabel: true },
    xAxis: {
      type: 'category',
      data: MOCK_TREND.dates,
      boundaryGap: false,
      axisLabel: {
        color: '#64748b',
        fontSize: 11,
        interval: 4,
      },
    },
    yAxis: {
      type: 'value',
      min: 20,
      max: 100,
      splitNumber: 4,
      axisLabel: { color: '#64748b', fontSize: 11 },
    },
    series: [
      {
        name: '信号强度',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        showSymbol: false,
        lineStyle: { width: 2.5, color: '#3b82f6' },
        itemStyle: { color: '#3b82f6' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(59,130,246,0.35)' },
              { offset: 0.6, color: 'rgba(59,130,246,0.08)' },
              { offset: 1, color: 'rgba(59,130,246,0)' },
            ],
          },
        },
        data: MOCK_TREND.values,
      },
    ],
  };
}

/* ────────────────────── Glass Card wrapper ────────────────────── */

function GlassCard({
  children,
  className = '',
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`bg-[#222536]/80 backdrop-blur-xl border border-white/10 rounded-2xl ${className}`}
    >
      {children}
    </div>
  );
}

/* ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   SignalsPage
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ */

export default function SignalsPage() {
  const [tab, setTab] = useState<Tab>('信号日历');
  const [days, setDays] = useState<number>(5);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [calendar, setCalendar] = useState<SignalEvent[]>([]);
  const [history, setHistory] = useState<SignalEvent[]>([]);
  const [anomalies, setAnomalies] = useState<Anomaly[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);

  const pieOption = useMemo(() => buildPieOption(), []);
  const trendOption = useMemo(() => buildTrendOption(), []);

  const fetchCalendar = useCallback(async (d: number) => {
    setLoading(true);
    setError('');
    try {
      const res = await signalApi.calendar({ days: d });
      const data = res.data;
      setCalendar(Array.isArray(data) ? data : []);
    } catch {
      setError('加载信号日历失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchHistory = useCallback(async (d: number) => {
    setLoading(true);
    setError('');
    try {
      const res = await signalApi.history({ days: d });
      setHistory((res.data?.items ?? []) as unknown as SignalEvent[]);
    } catch {
      setError('加载信号历史失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchAnomalies = useCallback(async (d: number) => {
    setLoading(true);
    setError('');
    try {
      const res = await signalApi.anomalies(d);
      const data = res.data;
      setAnomalies(Array.isArray(data) ? data : (data as any)?.anomalies ?? []);
    } catch {
      setError('加载异常检测数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchAlerts = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await alertsApi.list();
      const data = res.data;
      setAlerts(Array.isArray(data) ? data : (data as any)?.alerts ?? []);
    } catch {
      setError('加载系统告警失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (tab === '信号日历') fetchCalendar(days);
    else if (tab === '信号历史') fetchHistory(days);
    else if (tab === '异常检测') fetchAnomalies(days);
    else fetchAlerts();
  }, [tab, days, fetchCalendar, fetchHistory, fetchAnomalies, fetchAlerts]);

  const handleRefresh = () => {
    if (tab === '信号日历') fetchCalendar(days);
    else if (tab === '信号历史') fetchHistory(days);
    else if (tab === '异常检测') fetchAnomalies(days);
    else fetchAlerts();
  };

  const tabIcon = (t: Tab) => {
    switch (t) {
      case '信号日历': return <Calendar className="w-3.5 h-3.5" />;
      case '信号历史': return <Radio className="w-3.5 h-3.5" />;
      case '异常检测': return <AlertTriangle className="w-3.5 h-3.5" />;
      case '系统告警': return <Bell className="w-3.5 h-3.5" />;
    }
  };

  /* ── signal list summary badges ── */
  const signalSummaryBadges = useMemo(() => {
    const total = MOCK_SIGNAL_STATS.reduce((s, d) => s + d.value, 0);
    return MOCK_SIGNAL_STATS.map((d, i) => ({
      label: d.name,
      count: d.value,
      pct: ((d.value / total) * 100).toFixed(1),
      color: SIGNAL_COLORS[i],
    }));
  }, []);

  const renderSignalTable = (items: SignalEvent[]) => (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-xs" style={{ color: 'var(--color-text-muted)', borderBottom: '1px solid var(--color-border)' }}>
            <th className="text-left px-4 py-2.5 font-medium">代码</th>
            <th className="text-left px-4 py-2.5 font-medium">名称</th>
            <th className="text-left px-4 py-2.5 font-medium">信号类型</th>
            <th className="text-left px-4 py-2.5 font-medium">方向</th>
            <th className="text-right px-4 py-2.5 font-medium">分数</th>
            <th className="text-left px-4 py-2.5 font-medium">日期</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item, idx) => (
            <tr
              key={`${item.code}-${item.date}-${idx}`}
              className="transition-colors hover:bg-[var(--color-bg-hover)]"
              style={{ borderBottom: '1px solid var(--color-border)' }}
            >
              <td className="px-4 py-2.5 font-mono text-xs" style={{ color: 'var(--color-accent)' }}>
                {item.code}
              </td>
              <td className="px-4 py-2.5" style={{ color: 'var(--color-text-primary)' }}>
                {item.name}
              </td>
              <td className="px-4 py-2.5 text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                {item.signal_type}
              </td>
              <td className="px-4 py-2.5 text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                {item.direction}
              </td>
              <td className="px-4 py-2.5 text-right font-mono font-medium" style={{ color: 'var(--color-text-primary)' }}>
                {item.score}
              </td>
              <td className="px-4 py-2.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                {item.date}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Radio className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">信号中心</h1>
        </div>
        <button
          onClick={handleRefresh}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm hover:bg-[var(--color-bg-hover)] transition-colors"
          style={{ color: 'var(--color-text-secondary)' }}
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {/* ═══════════════ Charts Area ═══════════════ */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
        {/* ── 信号统计饼图 ── */}
        <GlassCard className="p-4">
          <div className="flex items-center gap-2 mb-3">
            <PieChart className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
            <span className="text-sm font-semibold" style={{ color: 'var(--color-text-primary)' }}>
              信号统计
            </span>
          </div>
          <EChart option={pieOption} height={240} />
          {/* Mini legend badges */}
          <div className="flex flex-wrap gap-2 mt-2 justify-center">
            {signalSummaryBadges.map((b) => (
              <span
                key={b.label}
                className="inline-flex items-center gap-1.5 text-xs px-2 py-1 rounded-full font-medium"
                style={{
                  background: `${b.color}18`,
                  color: b.color,
                  border: `1px solid ${b.color}30`,
                }}
              >
                <span
                  className="inline-block w-2 h-2 rounded-full"
                  style={{ background: b.color }}
                />
                {b.label}
                <span className="opacity-70">{b.count}</span>
              </span>
            ))}
          </div>
        </GlassCard>

        {/* ── 信号强度趋势 ── */}
        <GlassCard className="p-4">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <TrendingUp className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
              <span className="text-sm font-semibold" style={{ color: 'var(--color-text-primary)' }}>
                信号强度趋势
              </span>
            </div>
            <div className="flex items-center gap-1.5">
              <Activity className="w-3.5 h-3.5" style={{ color: 'var(--color-text-muted)' }} />
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
                近30天均值
              </span>
            </div>
          </div>
          <EChart option={trendOption} height={260} />
          {/* Trend summary */}
          <div className="flex items-center justify-between mt-2 px-1">
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
              最低 {Math.min(...MOCK_TREND.values).toFixed(1)} · 最高 {Math.max(...MOCK_TREND.values).toFixed(1)}
            </span>
            <span
              className="text-xs font-medium"
              style={{
                color:
                  MOCK_TREND.values[MOCK_TREND.values.length - 1] >= MOCK_TREND.values[0]
                    ? '#22c55e'
                    : '#ef4444',
              }}
            >
              {MOCK_TREND.values[MOCK_TREND.values.length - 1] >= MOCK_TREND.values[0] ? '↑' : '↓'}{' '}
              {Math.abs(
                MOCK_TREND.values[MOCK_TREND.values.length - 1] - MOCK_TREND.values[0]
              ).toFixed(1)}
            </span>
          </div>
        </GlassCard>
      </div>

      {/* ═══════════════ Tabs ═══════════════ */}
      <div className="flex items-center gap-1 mb-4 overflow-x-auto">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors whitespace-nowrap"
            style={{
              background: tab === t ? 'var(--color-bg-hover)' : 'transparent',
              color: tab === t ? 'var(--color-text-primary)' : 'var(--color-text-secondary)',
            }}
          >
            {tabIcon(t)}
            {t}
          </button>
        ))}
      </div>

      {/* Days filter */}
      {tab !== '系统告警' && (
        <div className="flex items-center gap-2 mb-4">
          <Calendar className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>时间范围:</span>
          <div className="flex rounded-lg overflow-hidden border" style={{ borderColor: 'var(--color-border)' }}>
            {DAYS_OPTIONS.map((d) => (
              <button
                key={d}
                onClick={() => setDays(d)}
                className="px-3 py-1.5 text-xs font-medium transition-colors"
                style={{
                  background: days === d ? 'var(--color-accent)' : 'var(--color-bg-card)',
                  color: days === d ? '#fff' : 'var(--color-text-muted)',
                }}
              >
                {d}天
              </button>
            ))}
          </div>
        </div>
      )}

      {error && (
        <div className="mb-4">
          <ErrorState
            title="加载失败"
            description={error}
            onRetry={() => { setError(''); handleRefresh(); }}
          />
        </div>
      )}

      {/* ═══════════════ Content (Signal List) ═══════════════ */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
      >
        <div className="px-4 py-3 flex items-center justify-between border-b" style={{ borderColor: 'var(--color-border)' }}>
          <span className="text-sm font-medium">{tab}</span>
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
            {tab === '信号日历' && `${calendar.length} 条`}
            {tab === '信号历史' && `${history.length} 条`}
            {tab === '异常检测' && `${anomalies.length} 条`}
            {tab === '系统告警' && `${alerts.length} 条`}
          </span>
        </div>

        {loading ? (
          <div className="p-4">
            <SkeletonInlineTable rows={6} columns={6} />
          </div>
        ) : tab === '信号日历' ? (
          calendar.length === 0 ? (
            <EmptyState
              icon={Bell}
              title="暂无交易信号"
              description="系统正在监控市场，有新信号时会及时通知"
            />
          ) : renderSignalTable(calendar)
        ) : tab === '信号历史' ? (
          history.length === 0 ? (
            <EmptyState
              icon={Bell}
              title="暂无交易信号"
              description="系统正在监控市场，有新信号时会及时通知"
            />
          ) : renderSignalTable(history)
        ) : tab === '异常检测' ? (
          anomalies.length === 0 ? (
            <EmptyState
              icon={Bell}
              title="暂无交易信号"
              description="系统正在监控市场，有新信号时会及时通知"
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-xs" style={{ color: 'var(--color-text-muted)', borderBottom: '1px solid var(--color-border)' }}>
                    <th className="text-left px-4 py-2.5 font-medium">代码</th>
                    <th className="text-left px-4 py-2.5 font-medium">名称</th>
                    <th className="text-left px-4 py-2.5 font-medium">异常类型</th>
                    <th className="text-left px-4 py-2.5 font-medium">描述</th>
                    <th className="text-center px-4 py-2.5 font-medium">严重程度</th>
                    <th className="text-left px-4 py-2.5 font-medium">日期</th>
                  </tr>
                </thead>
                <tbody>
                  {anomalies.map((item, idx) => {
                    const ss = severityStyle(item.severity);
                    return (
                      <tr
                        key={`${item.code}-${item.date}-${idx}`}
                        className="transition-colors hover:bg-[var(--color-bg-hover)]"
                        style={{ borderBottom: '1px solid var(--color-border)' }}
                      >
                        <td className="px-4 py-2.5 font-mono text-xs" style={{ color: 'var(--color-accent)' }}>
                          {item.code}
                        </td>
                        <td className="px-4 py-2.5" style={{ color: 'var(--color-text-primary)' }}>
                          {item.name}
                        </td>
                        <td className="px-4 py-2.5 text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                          {item.anomaly_type}
                        </td>
                        <td className="px-4 py-2.5 text-xs max-w-[200px] truncate" style={{ color: 'var(--color-text-secondary)' }} title={item.description}>
                          {item.description}
                        </td>
                        <td className="px-4 py-2.5 text-center">
                          <span
                            className="inline-block text-xs px-2 py-0.5 rounded-full font-medium"
                            style={{ background: ss.bg, color: ss.color }}
                          >
                            {severityLabel(item.severity)}
                          </span>
                        </td>
                        <td className="px-4 py-2.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                          {item.date}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )
        ) : alerts.length === 0 ? (
          <EmptyState
            icon={Bell}
            title="暂无交易信号"
            description="系统正在监控市场，有新信号时会及时通知"
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs" style={{ color: 'var(--color-text-muted)', borderBottom: '1px solid var(--color-border)' }}>
                  <th className="text-left px-4 py-2.5 font-medium">类型</th>
                  <th className="text-left px-4 py-2.5 font-medium">消息</th>
                  <th className="text-center px-4 py-2.5 font-medium">严重程度</th>
                  <th className="text-left px-4 py-2.5 font-medium">时间</th>
                  <th className="text-center px-4 py-2.5 font-medium">已读</th>
                </tr>
              </thead>
              <tbody>
                {alerts.map((item) => {
                  const ss = severityStyle(item.severity);
                  return (
                    <tr
                      key={item.id}
                      className="transition-colors hover:bg-[var(--color-bg-hover)]"
                      style={{ borderBottom: '1px solid var(--color-border)' }}
                    >
                      <td className="px-4 py-2.5 text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                        {item.type}
                      </td>
                      <td className="px-4 py-2.5 text-xs max-w-[300px] truncate" style={{ color: 'var(--color-text-primary)' }} title={item.message}>
                        {item.message}
                      </td>
                      <td className="px-4 py-2.5 text-center">
                        <span
                          className="inline-block text-xs px-2 py-0.5 rounded-full font-medium"
                          style={{ background: ss.bg, color: ss.color }}
                        >
                          {severityLabel(item.severity)}
                        </span>
                      </td>
                      <td className="px-4 py-2.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                        {item.created_at?.slice(0, 19).replace('T', ' ')}
                      </td>
                      <td className="px-4 py-2.5 text-center">
                        <span
                          className="inline-block text-xs px-2 py-0.5 rounded-full font-medium"
                          style={{
                            background: item.read ? 'rgba(34,197,94,0.15)' : 'rgba(234,179,8,0.15)',
                            color: item.read ? 'var(--color-success)' : 'var(--color-warning)',
                          }}
                        >
                          {item.read ? '已读' : '未读'}
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
