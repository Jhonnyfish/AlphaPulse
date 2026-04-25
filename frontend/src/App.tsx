import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from '@/lib/auth';
import Layout from '@/components/Layout';
import LoginPage from '@/pages/LoginPage';
import DashboardPage from '@/pages/DashboardPage';
import WatchlistPage from '@/pages/WatchlistPage';
import MarketPage from '@/pages/MarketPage';
import KlinePage from '@/pages/KlinePage';
import SectorsPage from '@/pages/SectorsPage';
import ComparePage from '@/pages/ComparePage';
import NewsPage from '@/pages/NewsPage';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { token, loading } = useAuth();

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center"
        style={{ background: 'var(--color-bg-primary)' }}>
        <span style={{ color: 'var(--color-text-muted)' }}>加载中...</span>
      </div>
    );
  }

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

export default function App() {
  const { token, loading } = useAuth();

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center"
        style={{ background: 'var(--color-bg-primary)' }}>
        <span style={{ color: 'var(--color-text-muted)' }}>加载中...</span>
      </div>
    );
  }

  return (
    <Routes>
      <Route
        path="/login"
        element={token ? <Navigate to="/dashboard" replace /> : <LoginPage />}
      />
      <Route
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/watchlist" element={<WatchlistPage />} />
        <Route path="/market" element={<MarketPage />} />
        <Route path="/kline" element={<KlinePage />} />
        <Route path="/sectors" element={<SectorsPage />} />
        <Route path="/compare" element={<ComparePage />} />
        <Route path="/news" element={<NewsPage />} />
      </Route>
      <Route path="*" element={<Navigate to={token ? '/dashboard' : '/login'} replace />} />
    </Routes>
  );
}
