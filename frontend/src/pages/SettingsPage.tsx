import { useState, useEffect } from 'react';
import api from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { useTheme } from '@/lib/theme';
import {
 Settings,
 Server,
 BookOpen,
 Trash2,
 User,
 RefreshCw,
 ExternalLink,
 CheckCircle,
 XCircle,
 Sun,
 Moon,
 Monitor,
} from 'lucide-react';
import ErrorState from '@/components/ErrorState';

interface SystemInfo {
  version: string;
  go_version: string;
  uptime: string;
  database: string;
  os: string;
  arch: string;
  cpu_count: number;
  goroutines: number;
  memory_alloc: string;
  [key: string]: unknown;
}

export default function SettingsPage() {
  const { user } = useAuth();
  const { theme, setTheme } = useTheme();
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [cacheClearing, setCacheClearing] = useState(false);
  const [cacheMessage, setCacheMessage] = useState<{ ok: boolean; text: string } | null>(null);

  useEffect(() => {
    api
      .get<SystemInfo>('/system/info')
      .then((res) => setSystemInfo(res.data))
      .catch(() => setError('加载系统信息失败'))
      .finally(() => setLoading(false));
  }, []);

  const handleClearCache = async () => {
    setCacheClearing(true);
    setCacheMessage(null);
    try {
      const res = await api.post<{ message?: string }>('/cache/clear');
      setCacheMessage({ ok: true, text: res.data?.message || '缓存已清理' });
    } catch {
      setCacheMessage({ ok: false, text: '清理缓存失败' });
    } finally {
      setCacheClearing(false);
    }
  };

  const infoItems = systemInfo
    ? [
        { label: '版本', value: systemInfo.version },
        { label: 'Go 版本', value: systemInfo.go_version },
        { label: '运行时间', value: systemInfo.uptime },
        { label: '数据库', value: systemInfo.database },
        { label: '操作系统', value: systemInfo.os },
        { label: '架构', value: systemInfo.arch },
        { label: 'CPU 核心数', value: String(systemInfo.cpu_count) },
        { label: 'Goroutines', value: String(systemInfo.goroutines) },
        { label: '内存分配', value: systemInfo.memory_alloc },
      ].filter((item) => item.value && item.value !== 'undefined')
    : [];

  const keyEndpoints = [
    { method: 'GET', path: '/api/market/overview', desc: '市场总览' },
    { method: 'GET', path: '/api/market/quote', desc: '个股行情' },
    { method: 'GET', path: '/api/watchlist', desc: '自选股列表' },
    { method: 'POST', path: '/api/watchlist', desc: '添加自选股' },
    { method: 'GET', path: '/api/sectors', desc: '板块行情' },
    { method: 'GET', path: '/api/auth/verify', desc: '验证令牌' },
  ];

  return (
    <div>
      <div className="flex items-center gap-2 mb-6">
        <Settings className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
        <h1 className="text-xl font-bold">系统设置</h1>
      </div>

      {/* User section */}
      <div
        className="rounded-xl border p-4 mb-4"
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
      >
        <div className="flex items-center gap-2 mb-3">
          <User className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <span className="text-sm font-medium">当前用户</span>
        </div>
        <div className="space-y-1 text-sm">
          <div className="flex items-center justify-between">
            <span style={{ color: 'var(--color-text-muted)' }}>用户名</span>
            <span className="font-mono">{user?.username || '—'}</span>
          </div>
          <div className="flex items-center justify-between">
            <span style={{ color: 'var(--color-text-muted)' }}>角色</span>
            <span className="font-mono">{user?.role || '—'}</span>
          </div>
          <div className="flex items-center justify-between">
            <span style={{ color: 'var(--color-text-muted)' }}>ID</span>
            <span className="font-mono text-xs">{user?.id || '—'}</span>
          </div>
        </div>
      </div>

      {/* Theme */}
      <div
        className="rounded-xl border p-4 mb-4"
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
      >
        <div className="flex items-center gap-2 mb-3">
          <Monitor className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <span className="text-sm font-medium">外观主题</span>
        </div>
        <div className="flex gap-2">
          {([
            { value: 'dark' as const, label: '暗色', icon: Moon },
            { value: 'light' as const, label: '亮色', icon: Sun },
          ]).map(({ value, label, icon: Icon }) => (
            <button
              key={value}
              onClick={() => setTheme(value)}
              className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm transition-all"
              style={{
                background: theme === value
                  ? 'linear-gradient(90deg, rgba(59, 130, 246, 0.2), rgba(59, 130, 246, 0.08))'
                  : 'var(--color-bg-card)',
                border: theme === value
                  ? '1px solid rgba(59, 130, 246, 0.4)'
                  : '1px solid var(--color-border)',
                color: theme === value ? 'var(--color-text-primary)' : 'var(--color-text-secondary)',
              }}
            >
              <Icon className="w-4 h-4" />
              {label}
              {theme === value && <CheckCircle className="w-3.5 h-3.5" style={{ color: 'var(--color-accent)' }} />}
            </button>
          ))}
        </div>
      </div>

      {/* System Info */}
      <div
        className="rounded-xl border p-4 mb-4"
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
      >
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Server className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
            <span className="text-sm font-medium">系统信息</span>
          </div>
          <button
            onClick={() => {
              setLoading(true);
              setError('');
              api
                .get<SystemInfo>('/system/info')
                .then((res) => setSystemInfo(res.data))
                .catch(() => setError('加载系统信息失败'))
                .finally(() => setLoading(false));
            }}
            disabled={loading}
            className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {error && (
          <div className="mb-3">
            <ErrorState
              title="加载失败"
              description={error}
              onRetry={() => {
                setLoading(true);
                setError('');
                api.get<SystemInfo>('/system/info')
                  .then((res) => setSystemInfo(res.data))
                  .catch(() => setError('加载系统信息失败'))
                  .finally(() => setLoading(false));
              }}
            />
          </div>
        )}

        {loading ? (
          <Skeleton rows={4} />
        ) : infoItems.length > 0 ? (
          <div className="space-y-1 text-sm">
            {infoItems.map((item) => (
              <div key={item.label} className="flex items-center justify-between">
                <span style={{ color: 'var(--color-text-muted)' }}>{item.label}</span>
                <span className="font-mono">{item.value}</span>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>暂无系统信息</p>
        )}
      </div>

      {/* API Docs */}
      <div
        className="rounded-xl border p-4 mb-4"
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
      >
        <div className="flex items-center gap-2 mb-3">
          <BookOpen className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <span className="text-sm font-medium">API 文档</span>
        </div>
        <a
          href="/api/docs"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1.5 text-sm px-3 py-1.5 rounded-lg transition-colors hover:bg-[var(--color-bg-hover)]"
          style={{ color: 'var(--color-accent)' }}
        >
          <ExternalLink className="w-3.5 h-3.5" />
          打开 API 文档
        </a>
        <div className="mt-3 space-y-1">
          {keyEndpoints.map((ep) => (
            <div key={ep.path} className="flex items-center gap-2 text-xs">
              <span
                className="px-1.5 py-0.5 rounded font-mono font-medium"
                style={{
                  background: ep.method === 'GET' ? 'rgba(34,197,94,0.15)' : 'rgba(59,130,246,0.15)',
                  color: ep.method === 'GET' ? 'var(--color-success)' : 'var(--color-accent)',
                }}
              >
                {ep.method}
              </span>
              <span className="font-mono" style={{ color: 'var(--color-text-secondary)' }}>
                {ep.path}
              </span>
              <span style={{ color: 'var(--color-text-muted)' }}>- {ep.desc}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Cache Management */}
      <div
        className="rounded-xl border p-4"
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
      >
        <div className="flex items-center gap-2 mb-3">
          <Trash2 className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <span className="text-sm font-medium">缓存管理</span>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={handleClearCache}
            disabled={cacheClearing}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors"
            style={{
              background: 'rgba(239,68,68,0.1)',
              color: 'var(--color-danger)',
              border: '1px solid rgba(239,68,68,0.3)',
            }}
          >
            {cacheClearing ? (
              <RefreshCw className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Trash2 className="w-3.5 h-3.5" />
            )}
            {cacheClearing ? '清理中...' : '清理缓存'}
          </button>
          {cacheMessage && (
            <span
              className="flex items-center gap-1 text-sm"
              style={{ color: cacheMessage.ok ? 'var(--color-success)' : 'var(--color-danger)' }}
            >
              {cacheMessage.ok ? <CheckCircle className="w-3.5 h-3.5" /> : <XCircle className="w-3.5 h-3.5" />}
              {cacheMessage.text}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
