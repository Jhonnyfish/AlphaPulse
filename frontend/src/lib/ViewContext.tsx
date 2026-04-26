import { createContext, useContext } from 'react';

/** All valid view names matching nav items */
export type ViewName =
  | 'dashboard' | 'watchlist' | 'market' | 'kline'
  | 'analyze' | 'sectors' | 'compare' | 'flow' | 'trends' | 'breadth' | 'sentiment'
  | 'candidates' | 'screener' | 'ranking' | 'hot-concepts' | 'dragon-tiger' | 'pattern-scanner'
  | 'portfolio' | 'journal' | 'strategies' | 'backtest' | 'strategy-eval'
  | 'trade-calendar' | 'signals' | 'portfolio-risk' | 'investment-plans'
  | 'watchlist-analysis' | 'news' | 'daily-brief' | 'institutions' | 'anomalies'
  | 'diag' | 'settings' | 'quick-actions'
  | 'daily-report' | 'perf-stats' | 'multi-trend' | 'correlation';

/** Map from URL-style path to ViewName */
export const pathToView: Record<string, ViewName> = {
  '/dashboard': 'dashboard',
  '/watchlist': 'watchlist',
  '/market': 'market',
  '/kline': 'kline',
  '/analyze': 'analyze',
  '/sectors': 'sectors',
  '/compare': 'compare',
  '/flow': 'flow',
  '/trends': 'trends',
  '/breadth': 'breadth',
  '/sentiment': 'sentiment',
  '/candidates': 'candidates',
  '/screener': 'screener',
  '/ranking': 'ranking',
  '/hot-concepts': 'hot-concepts',
  '/dragon-tiger': 'dragon-tiger',
  '/pattern-scanner': 'pattern-scanner',
  '/portfolio': 'portfolio',
  '/journal': 'journal',
  '/strategies': 'strategies',
  '/backtest': 'backtest',
  '/strategy-eval': 'strategy-eval',
  '/trade-calendar': 'trade-calendar',
  '/signals': 'signals',
  '/portfolio-risk': 'portfolio-risk',
  '/investment-plans': 'investment-plans',
  '/watchlist-analysis': 'watchlist-analysis',
  '/news': 'news',
  '/daily-brief': 'daily-brief',
  '/institutions': 'institutions',
  '/anomalies': 'anomalies',
  '/diag': 'diag',
  '/settings': 'settings',
  '/quick-actions': 'quick-actions',
  '/daily-report': 'daily-report',
  '/perf-stats': 'perf-stats',
  '/multi-trend': 'multi-trend',
  '/correlation': 'correlation',
};

export interface ViewContextValue {
  activeView: ViewName;
  viewParams: Record<string, string>;
  navigate: (view: ViewName, params?: Record<string, string>) => void;
}

export const ViewContext = createContext<ViewContextValue>({
  activeView: 'dashboard',
  viewParams: {},
  navigate: () => {},
});

export function useView(): ViewContextValue {
  return useContext(ViewContext);
}
