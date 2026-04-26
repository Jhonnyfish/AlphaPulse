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
import SettingsPage from '@/pages/SettingsPage';
import PortfolioPage from '@/pages/PortfolioPage';
import TradingJournalPage from '@/pages/TradingJournalPage';
import CandidatesPage from '@/pages/CandidatesPage';
import ScreenerPage from '@/pages/ScreenerPage';
import DragonTigerPage from '@/pages/DragonTigerPage';
import HotConceptsPage from '@/pages/HotConceptsPage';
import StrategiesPage from '@/pages/StrategiesPage';
import SignalsPage from '@/pages/SignalsPage';
import WatchlistAnalysisPage from '@/pages/WatchlistAnalysisPage';

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
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/portfolio" element={<PortfolioPage />} />
        <Route path="/journal" element={<TradingJournalPage />} />
        <Route path="/candidates" element={<CandidatesPage />} />
        <Route path="/screener" element={<ScreenerPage />} />
        <Route path="/dragon-tiger" element={<DragonTigerPage />} />
        <Route path="/hot-concepts" element={<HotConceptsPage />} />
        <Route path="/strategies" element={<StrategiesPage />} />
        <Route path="/signals" element={<SignalsPage />} />
        <Route path="/watchlist-analysis" element={<WatchlistAnalysisPage />} />
      </Route>
      <Route path="*" element={<Navigate to={token ? '/dashboard' : '/login'} replace />} />
    </Routes>
  );
}
