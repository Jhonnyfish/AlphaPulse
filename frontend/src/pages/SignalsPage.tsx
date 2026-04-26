import { useState, useEffect, useCallback } from 'react';
import { signalApi, alertsApi, type SignalEvent, type Anomaly, type Alert } from '@/lib/api';
import { Radio, AlertTriangle, Bell, Calendar, RefreshCw } from 'lucide-react';
import EmptyState from '@/components/EmptyState';
import { SkeletonInlineTable } from '@/components/ui/Skeleton';

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

export default function SignalsPage() {
  const [tab, setTab] = useState<Tab>('信号日历');
  const [days, setDays] = useState<number>(5);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [calendar, setCalendar] = useState<SignalEvent[]>([]);
  const [history, setHistory] = useState<SignalEvent[]>([]);
  const [anomalies, setAnomalies] = useState<Anomaly[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);

  const fetchCalendar = useCallback(async (d: number) => {
    setLoading(true);
    setError('');
    try {
      const res = await signalApi.calendar({ days: d });
      setCalendar(res.data ?? []);
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
      setHistory(res.data ?? []);
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
      setAnomalies(res.data ?? []);
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
      setAlerts(res.data ?? []);
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

      {/* Tabs */}
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
        <div
          className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
        >
          {error}
        </div>
      )}

      {/* Content */}
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
