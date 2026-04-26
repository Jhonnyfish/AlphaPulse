import { useState, useEffect, useMemo, useCallback } from 'react';
import { useAuth } from '@/lib/auth';
import { useKeyboard } from '@/lib/useKeyboard';
import { useTheme } from '@/lib/theme';
import { useView, type ViewName } from '@/lib/ViewContext';
import {
  BarChart3, Star, TrendingUp, Newspaper, LogOut, Activity,
  CandlestickChart, LayoutDashboard, Menu, X, GitCompareArrows,
  Settings, Briefcase, BookOpen, Target, Filter, Crown, Flame,
  Zap, Radio, Grid3X3, Droplets, Search, Sun, Moon, Keyboard,
  Heart, FileText, Monitor, AlertTriangle, Building2, Trophy,
  Gauge, GitBranch, Network, CalendarClock, FlaskConical, CalendarDays,
  Scan, ShieldAlert, Bolt,
} from 'lucide-react';
import CommandPalette from '@/components/CommandPalette';
import TickerTape from '@/components/TickerTape';
import KeyboardHelpPanel from '@/components/KeyboardHelpPanel';

interface NavItem {
  view: ViewName;
  label: string;
  icon: typeof Activity;
  group: string;
}

const navItems: NavItem[] = [
  { view: 'daily-report', label: '每日报告', icon: CalendarClock, group: '工具' },
  { view: 'perf-stats', label: '绩效统计', icon: Gauge, group: '工具' },
  { view: 'multi-trend', label: '多周期趋势', icon: GitBranch, group: '分析' },
  { view: 'correlation', label: '相关性', icon: Network, group: '分析' },
  { view: 'investment-plans', label: '投资计划', icon: Target, group: '交易' },
  { view: 'dashboard', label: '总览', icon: LayoutDashboard, group: '核心' },
  { view: 'watchlist', label: '自选股', icon: Star, group: '核心' },
  { view: 'market', label: '行情', icon: TrendingUp, group: '核心' },
  { view: 'kline', label: 'K线', icon: CandlestickChart, group: '核心' },

  { view: 'analyze', label: '个股分析', icon: Activity, group: '分析' },
  { view: 'sectors', label: '板块', icon: BarChart3, group: '分析' },
  { view: 'compare', label: '对比', icon: GitCompareArrows, group: '分析' },
  { view: 'flow', label: '资金流向', icon: Droplets, group: '分析' },
  { view: 'trends', label: '趋势', icon: TrendingUp, group: '分析' },
  { view: 'breadth', label: '市场广度', icon: Activity, group: '分析' },
  { view: 'sentiment', label: '市场情绪', icon: Heart, group: '分析' },

  { view: 'candidates', label: '候选股', icon: Target, group: '选股' },
  { view: 'screener', label: '选股器', icon: Filter, group: '选股' },
  { view: 'ranking', label: '综合排名', icon: Trophy, group: '选股' },
  { view: 'hot-concepts', label: '热门概念', icon: Flame, group: '选股' },
  { view: 'dragon-tiger', label: '龙虎榜', icon: Crown, group: '选股' },
  { view: 'pattern-scanner', label: '形态扫描', icon: Scan, group: '选股' },

  { view: 'portfolio', label: '持仓', icon: Briefcase, group: '交易' },
  { view: 'journal', label: '交易日志', icon: BookOpen, group: '交易' },
  { view: 'strategies', label: '策略', icon: Zap, group: '交易' },
  { view: 'backtest', label: '策略回测', icon: FlaskConical, group: '交易' },
  { view: 'strategy-eval', label: '策略评估', icon: BarChart3, group: '交易' },
  { view: 'trade-calendar', label: '交易日历', icon: CalendarDays, group: '交易' },
  { view: 'signals', label: '信号', icon: Radio, group: '交易' },
  { view: 'portfolio-risk', label: '组合风险', icon: ShieldAlert, group: '交易' },

  { view: 'watchlist-analysis', label: '自选分析', icon: Grid3X3, group: '工具' },
  { view: 'news', label: '资讯', icon: Newspaper, group: '工具' },
  { view: 'daily-brief', label: '每日简报', icon: FileText, group: '工具' },
  { view: 'institutions', label: '机构动向', icon: Building2, group: '工具' },
  { view: 'anomalies', label: '异常检测', icon: AlertTriangle, group: '工具' },
  { view: 'diag', label: '系统诊断', icon: Monitor, group: '工具' },
  { view: 'settings', label: '设置', icon: Settings, group: '工具' },
  { view: 'quick-actions', label: '快捷操作', icon: Bolt, group: '工具' },
];

// Group nav items by category
function groupNavItems(items: NavItem[]): [string, NavItem[]][] {
  const groups: Record<string, NavItem[]> = {};
  for (const item of items) {
    if (!groups[item.group]) groups[item.group] = [];
    groups[item.group].push(item);
  }
  return Object.entries(groups);
}

interface LayoutProps {
  children: React.ReactNode;
}

export default function Layout({ children }: LayoutProps) {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const { activeView, navigate } = useView();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [keyboardHelpOpen, setKeyboardHelpOpen] = useState(false);

  // Close sidebar on Escape key
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setSidebarOpen(false);
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, []);

  // Close sidebar on view change (mobile)
  useEffect(() => {
    setSidebarOpen(false);
  }, [activeView]);

  // Global keyboard shortcuts
  const keyBindings = useMemo(() => [
    {
      key: 'k',
      ctrl: true,
      meta: true,
      handler: () => setCommandPaletteOpen(prev => !prev),
      description: '搜索 / 导航',
    },
    {
      key: 'r',
      handler: () => window.location.reload(),
      description: '刷新页面',
    },
    {
      key: 'd',
      handler: () => navigate('dashboard'),
      description: '回到总览',
    },
    {
      key: 'w',
      handler: () => navigate('watchlist'),
      description: '自选股',
    },
    {
      key: '?',
      shift: true,
      handler: () => setKeyboardHelpOpen(prev => !prev),
      description: '快捷键帮助',
    },
  ], [navigate]);

  useKeyboard(keyBindings);

  const grouped = groupNavItems(navItems);

  return (
    <div className="flex h-screen relative">
      {/* Command Palette */}
      <CommandPalette
        open={commandPaletteOpen}
        onClose={() => setCommandPaletteOpen(false)}
      />

      {/* Keyboard Help */}
      <KeyboardHelpPanel
        open={keyboardHelpOpen}
        onClose={() => setKeyboardHelpOpen(false)}
      />

      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 lg:hidden animate-fade-in"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`
          fixed inset-y-0 left-0 z-50 w-56 flex flex-col border-r transform transition-transform duration-200 ease-in-out
          lg:relative lg:translate-x-0
          ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'}
        `}
        style={{
          background: 'var(--glass-bg)',
          backdropFilter: 'blur(18px)',
          WebkitBackdropFilter: 'blur(18px)',
          borderColor: 'var(--color-border)',
        }}
      >
        {/* Logo */}
        <div className="flex items-center justify-between px-5 py-4 border-b"
          style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2.5">
            <div className="relative">
              <Activity className="w-6 h-6" style={{ color: 'var(--color-accent)' }} />
              <div
                className="absolute inset-0 rounded-full"
                style={{
                  background: 'rgba(59, 130, 246, 0.3)',
                  filter: 'blur(8px)',
                }}
              />
            </div>
            <span className="text-lg font-bold tracking-tight text-gradient">AlphaPulse</span>
          </div>
          <button
            onClick={() => setSidebarOpen(false)}
            className="p-1 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors lg:hidden"
          >
            <X className="w-5 h-5" style={{ color: 'var(--color-text-muted)' }} />
          </button>
        </div>

        {/* Search trigger */}
        <div className="px-3 pt-3">
          <button
            onClick={() => setCommandPaletteOpen(true)}
            className="flex items-center gap-2.5 w-full px-3 py-2 rounded-lg text-sm transition-colors hover:bg-[var(--color-bg-hover)]"
            style={{
              color: 'var(--color-text-muted)',
              border: '1px solid var(--color-border)',
            }}
          >
            <Search className="w-4 h-4" />
            <span className="flex-1 text-left">搜索...</span>
            <kbd
              className="hidden sm:inline-flex items-center px-1.5 py-0.5 rounded text-[10px]"
              style={{
                background: 'var(--color-bg-hover)',
                border: '1px solid var(--color-border)',
              }}
            >
              ⌘K
            </kbd>
          </button>
        </div>

        {/* Nav */}
        <nav className="flex-1 py-2 px-2 overflow-y-auto">
          {grouped.map(([group, items]) => (
            <div key={group} className="mb-2">
              <div
                className="px-3 py-1.5 text-[10px] font-semibold uppercase tracking-widest"
                style={{ color: 'var(--color-text-muted)' }}
              >
                {group}
              </div>
              {items.map(({ view, label, icon: Icon }) => {
                const isActive = activeView === view;
                return (
                  <button
                    key={view}
                    onClick={() => navigate(view)}
                    className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-all duration-150 w-full text-left ${
                      isActive
                        ? 'font-medium'
                        : 'hover:bg-[var(--color-bg-hover)]'
                    }`}
                    style={{
                      background: isActive
                        ? 'linear-gradient(90deg, rgba(59, 130, 246, 0.15), rgba(59, 130, 246, 0.05))'
                        : undefined,
                      color: isActive ? 'var(--color-text-primary)' : 'var(--color-text-secondary)',
                      borderLeft: isActive ? '2px solid var(--color-accent)' : '2px solid transparent',
                    }}
                  >
                    <Icon className="w-4 h-4" style={{ opacity: 0.85 }} />
                    {label}
                  </button>
                );
              })}
            </div>
          ))}
        </nav>

        {/* User section */}
        <div className="px-4 py-3 border-t flex items-center justify-between"
          style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2.5">
            <div
              className="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold"
              style={{
                background: 'linear-gradient(135deg, var(--color-accent), var(--color-cyan))',
                color: '#fff',
              }}
            >
              {user?.username?.[0]?.toUpperCase() || 'U'}
            </div>
            <span className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
              {user?.username}
            </span>
          </div>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setKeyboardHelpOpen(true)}
              className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
              title="快捷键帮助"
            >
              <Keyboard className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
            </button>
            <button
              onClick={toggleTheme}
              className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
              title={theme === 'dark' ? '切换到亮色模式' : '切换到暗色模式'}
            >
              {theme === 'dark' ? (
                <Sun className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
              ) : (
                <Moon className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
              )}
            </button>
            <button
              onClick={logout}
              className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
              title="退出登录"
            >
              <LogOut className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
            </button>
          </div>
        </div>
      </aside>

      {/* Main content area */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Mobile top bar */}
        <header
          className="flex items-center gap-3 px-4 py-3 border-b lg:hidden"
          style={{
            background: 'var(--glass-bg)',
            backdropFilter: 'blur(18px)',
            WebkitBackdropFilter: 'blur(18px)',
            borderColor: 'var(--color-border)',
          }}
        >
          <button
            onClick={() => setSidebarOpen(true)}
            className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
          >
            <Menu className="w-5 h-5" style={{ color: 'var(--color-text-primary)' }} />
          </button>
          <div className="flex items-center gap-2">
            <Activity className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
            <span className="font-bold text-gradient">AlphaPulse</span>
          </div>
          <div className="ml-auto flex items-center gap-1">
            <button
              onClick={toggleTheme}
              className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
              title={theme === 'dark' ? '切换到亮色模式' : '切换到暗色模式'}
            >
              {theme === 'dark' ? (
                <Sun className="w-5 h-5" style={{ color: 'var(--color-text-muted)' }} />
              ) : (
                <Moon className="w-5 h-5" style={{ color: 'var(--color-text-muted)' }} />
              )}
            </button>
            <button
              onClick={() => setCommandPaletteOpen(true)}
              className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
            >
              <Search className="w-5 h-5" style={{ color: 'var(--color-text-muted)' }} />
            </button>
          </div>
        </header>

        {/* Ticker Tape */}
        <TickerTape />

        {/* Content */}
        <main className="flex-1 overflow-auto p-4 sm:p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
