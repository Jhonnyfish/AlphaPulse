import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from '@/lib/auth';
import Layout from '@/components/Layout';
import LoginPage from '@/pages/LoginPage';
import WatchlistPage from '@/pages/WatchlistPage';
import MarketPage from '@/pages/MarketPage';

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
        element={token ? <Navigate to="/watchlist" replace /> : <LoginPage />}
      />
      <Route
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
        <Route path="/watchlist" element={<WatchlistPage />} />
        <Route path="/market" element={<MarketPage />} />
        <Route path="/sectors" element={<PlaceholderPage title="板块" />} />
        <Route path="/news" element={<PlaceholderPage title="资讯" />} />
      </Route>
      <Route path="*" element={<Navigate to={token ? '/watchlist' : '/login'} replace />} />
    </Routes>
  );
}

function PlaceholderPage({ title }: { title: string }) {
  return (
    <div>
      <h1 className="text-xl font-bold mb-4">{title}</h1>
      <p style={{ color: 'var(--color-text-muted)' }}>即将上线...</p>
    </div>
  );
}
