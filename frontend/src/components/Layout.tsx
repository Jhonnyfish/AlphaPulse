import { useState, useEffect } from 'react';
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import { useAuth } from '@/lib/auth';
import { BarChart3, Star, TrendingUp, Newspaper, LogOut, Activity, CandlestickChart, LayoutDashboard, Menu, X, GitCompareArrows } from 'lucide-react';

const navItems = [
  { to: '/dashboard', label: '总览', icon: LayoutDashboard },
  { to: '/watchlist', label: '自选股', icon: Star },
  { to: '/market', label: '行情', icon: TrendingUp },
  { to: '/kline', label: 'K线', icon: CandlestickChart },
  { to: '/sectors', label: '板块', icon: BarChart3 },
  { to: '/compare', label: '对比', icon: GitCompareArrows },
  { to: '/news', label: '资讯', icon: Newspaper },
];

export default function Layout() {
  const { user, logout } = useAuth();
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(false);

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

  return (
    <div className="flex h-screen relative">
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 lg:hidden"
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
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
      >
        {/* Logo */}
        <div className="flex items-center justify-between px-5 py-4 border-b"
          style={{ borderColor: 'var(--color-border)' }}>
          <div className="flex items-center gap-2">
            <Activity className="w-6 h-6" style={{ color: 'var(--color-accent)' }} />
            <span className="text-lg font-bold tracking-tight">AlphaPulse</span>
          </div>
          <button
            onClick={() => setSidebarOpen(false)}
            className="p-1 rounded hover:bg-[var(--color-bg-hover)] transition-colors lg:hidden"
          >
            <X className="w-5 h-5" style={{ color: 'var(--color-text-muted)' }} />
          </button>
        </div>

        {/* Nav */}
        <nav className="flex-1 py-3 px-2 space-y-1">
          {navItems.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-colors ${
                  isActive
                    ? 'font-medium'
                    : 'hover:bg-[var(--color-bg-hover)]'
                }`
              }
              style={({ isActive }) => ({
                background: isActive ? 'var(--color-bg-hover)' : undefined,
                color: isActive ? 'var(--color-text-primary)' : 'var(--color-text-secondary)',
              })}
            >
              <Icon className="w-4 h-4" />
              {label}
            </NavLink>
          ))}
        </nav>

        {/* User */}
        <div className="px-4 py-3 border-t flex items-center justify-between"
          style={{ borderColor: 'var(--color-border)' }}>
          <span className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
            {user?.username}
          </span>
          <button
            onClick={logout}
            className="p-1.5 rounded hover:bg-[var(--color-bg-hover)] transition-colors"
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
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <button
            onClick={() => setSidebarOpen(true)}
            className="p-1.5 rounded hover:bg-[var(--color-bg-hover)] transition-colors"
          >
            <Menu className="w-5 h-5" style={{ color: 'var(--color-text-primary)' }} />
          </button>
          <div className="flex items-center gap-2">
            <Activity className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
            <span className="font-bold">AlphaPulse</span>
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
