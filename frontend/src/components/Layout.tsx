import { NavLink, Outlet } from 'react-router-dom';
import { useAuth } from '@/lib/auth';
import { BarChart3, Star, TrendingUp, Newspaper, LogOut, Activity, CandlestickChart } from 'lucide-react';

const navItems = [
  { to: '/watchlist', label: '自选股', icon: Star },
  { to: '/market', label: '行情', icon: TrendingUp },
  { to: '/kline', label: 'K线', icon: CandlestickChart },
  { to: '/sectors', label: '板块', icon: BarChart3 },
  { to: '/news', label: '资讯', icon: Newspaper },
];

export default function Layout() {
  const { user, logout } = useAuth();

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <aside className="w-56 flex flex-col border-r"
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}>
        {/* Logo */}
        <div className="flex items-center gap-2 px-5 py-4 border-b"
          style={{ borderColor: 'var(--color-border)' }}>
          <Activity className="w-6 h-6" style={{ color: 'var(--color-accent)' }} />
          <span className="text-lg font-bold tracking-tight">AlphaPulse</span>
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

      {/* Main content */}
      <main className="flex-1 overflow-auto p-6">
        <Outlet />
      </main>
    </div>
  );
}
