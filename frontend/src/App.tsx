import { useState, useMemo } from 'react';
import { useAuth } from '@/lib/auth';
import { ViewContext, type ViewName, type ViewContextValue } from '@/lib/ViewContext';
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
import PatternScannerPage from '@/pages/PatternScannerPage';
import PortfolioRiskPage from '@/pages/PortfolioRiskPage';
import QuickActionsPage from '@/pages/QuickActionsPage';

export default function App() {
  const { token, loading } = useAuth();
  const [activeView, setActiveView] = useState<ViewName>('dashboard');
  const [viewParams, setViewParams] = useState<Record<string, string>>({});

  const ctxValue = useMemo<ViewContextValue>(() => ({
    activeView,
    viewParams,
    navigate: (view: ViewName, params?: Record<string, string>) => {
      setActiveView(view);
      setViewParams(params ?? {});
    },
  }), [activeView, viewParams]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center"
        style={{ background: 'var(--color-bg-primary)' }}>
        <span style={{ color: 'var(--color-text-muted)' }}>加载中...</span>
      </div>
    );
  }

  if (!token) {
    return <LoginPage />;
  }

  return (
    <ViewContext.Provider value={ctxValue}>
      <Layout>
        {activeView === 'dashboard' && <DashboardPage />}
        {activeView === 'watchlist' && <WatchlistPage />}
        {activeView === 'market' && <MarketPage />}
        {activeView === 'kline' && <KlinePage />}
        {activeView === 'analyze' && <AnalyzePage />}
        {activeView === 'sectors' && <SectorsPage />}
        {activeView === 'compare' && <ComparePage />}
        {activeView === 'news' && <NewsPage />}
        {activeView === 'settings' && <SettingsPage />}
        {activeView === 'portfolio' && <PortfolioPage />}
        {activeView === 'journal' && <TradingJournalPage />}
        {activeView === 'candidates' && <CandidatesPage />}
        {activeView === 'screener' && <ScreenerPage />}
        {activeView === 'dragon-tiger' && <DragonTigerPage />}
        {activeView === 'hot-concepts' && <HotConceptsPage />}
        {activeView === 'strategies' && <StrategiesPage />}
        {activeView === 'signals' && <SignalsPage />}
        {activeView === 'watchlist-analysis' && <WatchlistAnalysisPage />}
        {activeView === 'flow' && <FlowPanelPage />}
        {activeView === 'trends' && <TrendsPage />}
        {activeView === 'breadth' && <BreadthPage />}
        {activeView === 'sentiment' && <SentimentPage />}
        {activeView === 'daily-brief' && <DailyBriefPage />}
        {activeView === 'diag' && <DiagPage />}
        {activeView === 'anomalies' && <AnomalyPage />}
        {activeView === 'institutions' && <InstitutionPage />}
        {activeView === 'ranking' && <RankingPage />}
        {activeView === 'daily-report' && <DailyReportPage />}
        {activeView === 'perf-stats' && <PerfStatsPage />}
        {activeView === 'multi-trend' && <MultiTrendPage />}
        {activeView === 'correlation' && <CorrelationPage />}
        {activeView === 'investment-plans' && <InvestmentPlansPage />}
        {activeView === 'backtest' && <BacktestPage />}
        {activeView === 'strategy-eval' && <StrategyEvalPage />}
        {activeView === 'trade-calendar' && <TradeCalendarPage />}
        {activeView === 'pattern-scanner' && <PatternScannerPage />}
        {activeView === 'portfolio-risk' && <PortfolioRiskPage />}
        {activeView === 'quick-actions' && <QuickActionsPage />}
      </Layout>
    </ViewContext.Provider>
  );
}
