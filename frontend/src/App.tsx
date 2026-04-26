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
import AnalyzePage from '@/pages/AnalyzePage';
import FlowPanelPage from '@/pages/FlowPanelPage';
import TrendsPage from '@/pages/TrendsPage';
import BreadthPage from '@/pages/BreadthPage';
import SentimentPage from '@/pages/SentimentPage';
import DailyBriefPage from '@/pages/DailyBriefPage';
import DiagPage from '@/pages/DiagPage';
import AnomalyPage from '@/pages/AnomalyPage';
import InstitutionPage from '@/pages/InstitutionPage';
import RankingPage from '@/pages/RankingPage';
import DailyReportPage from '@/pages/DailyReportPage';
import PerfStatsPage from '@/pages/PerfStatsPage';
import MultiTrendPage from '@/pages/MultiTrendPage';
import CorrelationPage from '@/pages/CorrelationPage';
import InvestmentPlansPage from '@/pages/InvestmentPlansPage';
import BacktestPage from '@/pages/BacktestPage';
import StrategyEvalPage from '@/pages/StrategyEvalPage';
import TradeCalendarPage from '@/pages/TradeCalendarPage';

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
        <Route path="/analyze" element={<AnalyzePage />} />
        <Route path="/flow" element={<FlowPanelPage />} />
        <Route path="/trends" element={<TrendsPage />} />
        <Route path="/breadth" element={<BreadthPage />} />
        <Route path="/sentiment" element={<SentimentPage />} />
        <Route path="/daily-brief" element={<DailyBriefPage />} />
        <Route path="/diag" element={<DiagPage />} />
        <Route path="/anomalies" element={<AnomalyPage />} />
        <Route path="/institutions" element={<InstitutionPage />} />
        <Route path="/ranking" element={<RankingPage />} />
        <Route path="/daily-report" element={<DailyReportPage />} />
        <Route path="/perf-stats" element={<PerfStatsPage />} />
        <Route path="/multi-trend" element={<MultiTrendPage />} />
        <Route path="/correlation" element={<CorrelationPage />} />
        <Route path="/investment-plans" element={<InvestmentPlansPage />} />
        <Route path="/backtest" element={<BacktestPage />} />
        <Route path="/strategy-eval" element={<StrategyEvalPage />} />
        <Route path="/trade-calendar" element={<TradeCalendarPage />} />
      </Route>
      <Route path="*" element={<Navigate to={token ? '/dashboard' : '/login'} replace />} />
    </Routes>
  );
}
