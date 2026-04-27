import React, { Suspense } from 'react';
import { useState, useMemo } from 'react';
import { useAuth } from '@/lib/auth';
import { ViewContext, type ViewName, type ViewContextValue } from '@/lib/ViewContext';
import Layout from '@/components/Layout';
import LoginPage from '@/pages/LoginPage';
import ErrorBoundary from '@/components/ErrorBoundary';
import AnimatedView from '@/components/AnimatedView';
import LoadingSpinner from '@/components/LoadingSpinner';

// Lazy-loaded page components — each becomes a separate chunk
const DashboardPage = React.lazy(() => import('@/pages/DashboardPage'));
const WatchlistPage = React.lazy(() => import('@/pages/WatchlistPage'));
const MarketPage = React.lazy(() => import('@/pages/MarketPage'));
const KlinePage = React.lazy(() => import('@/pages/KlinePage'));
const SectorsPage = React.lazy(() => import('@/pages/SectorsPage'));
const ComparePage = React.lazy(() => import('@/pages/ComparePage'));
const NewsPage = React.lazy(() => import('@/pages/NewsPage'));
const SettingsPage = React.lazy(() => import('@/pages/SettingsPage'));
const PortfolioPage = React.lazy(() => import('@/pages/PortfolioPage'));
const TradingJournalPage = React.lazy(() => import('@/pages/TradingJournalPage'));
const CandidatesPage = React.lazy(() => import('@/pages/CandidatesPage'));
const ScreenerPage = React.lazy(() => import('@/pages/ScreenerPage'));
const DragonTigerPage = React.lazy(() => import('@/pages/DragonTigerPage'));
const HotConceptsPage = React.lazy(() => import('@/pages/HotConceptsPage'));
const StrategiesPage = React.lazy(() => import('@/pages/StrategiesPage'));
const SignalsPage = React.lazy(() => import('@/pages/SignalsPage'));
const WatchlistAnalysisPage = React.lazy(() => import('@/pages/WatchlistAnalysisPage'));
const AnalyzePage = React.lazy(() => import('@/pages/AnalyzePage'));
const FlowPanelPage = React.lazy(() => import('@/pages/FlowPanelPage'));
const TrendsPage = React.lazy(() => import('@/pages/TrendsPage'));
const BreadthPage = React.lazy(() => import('@/pages/BreadthPage'));
const SentimentPage = React.lazy(() => import('@/pages/SentimentPage'));
const DailyBriefPage = React.lazy(() => import('@/pages/DailyBriefPage'));
const DiagPage = React.lazy(() => import('@/pages/DiagPage'));
const AnomalyPage = React.lazy(() => import('@/pages/AnomalyPage'));
const InstitutionPage = React.lazy(() => import('@/pages/InstitutionPage'));
const RankingPage = React.lazy(() => import('@/pages/RankingPage'));
const DailyReportPage = React.lazy(() => import('@/pages/DailyReportPage'));
const PerfStatsPage = React.lazy(() => import('@/pages/PerfStatsPage'));
const MultiTrendPage = React.lazy(() => import('@/pages/MultiTrendPage'));
const CorrelationPage = React.lazy(() => import('@/pages/CorrelationPage'));
const InvestmentPlansPage = React.lazy(() => import('@/pages/InvestmentPlansPage'));
const BacktestPage = React.lazy(() => import('@/pages/BacktestPage'));
const StrategyEvalPage = React.lazy(() => import('@/pages/StrategyEvalPage'));
const TradeCalendarPage = React.lazy(() => import('@/pages/TradeCalendarPage'));
const PatternScannerPage = React.lazy(() => import('@/pages/PatternScannerPage'));
const PortfolioRiskPage = React.lazy(() => import('@/pages/PortfolioRiskPage'));
const QuickActionsPage = React.lazy(() => import('@/pages/QuickActionsPage'));

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
        <ErrorBoundary key={activeView}>
        <AnimatedView key={activeView}>
          <Suspense fallback={<LoadingSpinner />}>
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
          </Suspense>
        </AnimatedView>
        </ErrorBoundary>
      </Layout>
    </ViewContext.Provider>
  );
}
