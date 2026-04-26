import { useState, useEffect, useMemo, useCallback } from 'react';
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '@/lib/auth';
import { useKeyboard } from '@/lib/useKeyboard';
import {
  BarChart3, Star, TrendingUp, Newspaper, LogOut, Activity,
  CandlestickChart, LayoutDashboard, Menu, X, GitCompareArrows,
  Settings, Briefcase, BookOpen, Target, Filter, Crown, Flame,
  Zap, Radio, Grid3X3, Droplets, Search,
} from 'lucide-react';
import CommandPalette from '@/components/CommandPalette';

interface NavItem {
  to: string;
  label: string;
  icon: typeof Activity;
  group: string;
}

const navItems: NavItem[] = [
  { to: '/dashboard', label: '总览', icon: LayoutDashboard, group: '核心' },
  { to: '/watchlist', label: '自选股', icon: Star, group: '核心' },
  { to: '/market', label: '行情', icon: TrendingUp, group: '核心' },
  { to: '/kline', label: 'K线', icon: CandlestickChart, group: '核心' },

  { to: '/analyze', label: '个股分析', icon: Activity, group: '分析' },
  { to: '/sectors', label: '板块', icon: BarChart3, group: '分析' },
  { to: '/compare', label: '对比', icon: GitCompareArrows, group: '分析' },
  { to: '/flow', label: '资金流向', icon: Droplets, group: '分析' },

  { to: '/candidates', label: '候选股', icon: Target, group: '选股' },
  { to: '/screener', label: '选股器', icon: Filter, group: '选股' },
  { to: '/hot-concepts', label: '热门概念', icon: Flame, group: '选股' },
  { to: '/dragon-tiger', label: '龙虎榜', icon: Crown, group: '选股' },

  { to: '/portfolio', label: '持仓', icon: Briefcase, group: '交易' },
  { to: '/journal', label: '交易日志', icon: BookOpen, group: '交易' },
  { to: '/strategies', label: '策略', icon: Zap, group: '交易' },
  { to: '/signals', label: '信号', icon: Radio, group: '交易' },

  { to: '/watchlist-analysis', label: '自选分析', icon: Grid3X3, group: '工具' },
  { to: '/news', label: '资讯', icon: Newspaper, group: '工具' },
  { to: '/settings', label: '设置', icon: Settings, group: '工具' },
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

export default function Layout() {
  const { user, logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);

  // Close sidebar on route change (mobile)
  useEffect(() => {
    setSidebarOpen(false);
  }, [location.pathname]);

  // Close sidebar on Escape key
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setSidebarOpen(false);
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, []);

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
      handler: () => navigate('/dashboard'),
      description: '回到总览',
    },
    {
      key: 'w',
      handler: () => navigate('/watchlist'),
      description: '自选股',
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
              {items.map(({ to, label, icon: Icon }) => (
                <NavLink
                  key={to}
                  to={to}
                  className={({ isActive }) =>
                    `flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-all duration-150 ${
                      isActive
                        ? 'font-medium'
                        : 'hover:bg-[var(--color-bg-hover)]'
                    }`
                  }
                  style={({ isActive }) => ({
                    background: isActive
                      ? 'linear-gradient(90deg, rgba(59, 130, 246, 0.15), rgba(59, 130, 246, 0.05))'
                      : undefined,
                    color: isActive ? 'var(--color-text-primary)' : 'var(--color-text-secondary)',
                    borderLeft: isActive ? '2px solid var(--color-accent)' : '2px solid transparent',
                  })}
                >
                  <Icon className="w-4 h-4" style={{ opacity: 0.85 }} />
                  {label}
                </NavLink>
              ))}
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
          <button
            onClick={logout}
            className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
            title="退出登录"
          >
            <LogOut className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
          </button>
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
          <div className="ml-auto">
            <button
              onClick={() => setCommandPaletteOpen(true)}
              className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
            >
              <Search className="w-5 h-5" style={{ color: 'var(--color-text-muted)' }} />
            </button>
          </div>
        </header>

        {/* Content */}
        <main className="flex-1 overflow-auto p-4 sm:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
