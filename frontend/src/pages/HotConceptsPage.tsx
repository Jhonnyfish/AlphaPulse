import { useState, useEffect, useCallback } from 'react';
import { hotConceptsApi, type HotConcept } from '@/lib/api';
import EmptyState from '@/components/EmptyState';
import ErrorState from '@/components/ErrorState';
import { Flame, RefreshCw, TrendingUp, TrendingDown, ChevronDown, ChevronUp, Star, Link2 } from 'lucide-react';
import { SkeletonGridCard, SkeletonList, SkeletonInlineTable } from '@/components/ui/Skeleton';

interface ConceptStock {
  code: string;
  name: string;
  change_pct: number;
  price: number;
}

interface WatchlistOverlap {
  code: string;
  name: string;
  concepts: string[];
}

export default function HotConceptsPage() {
  const [concepts, setConcepts] = useState<HotConcept[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [expandedCode, setExpandedCode] = useState<string | null>(null);
  const [expandedStocks, setExpandedStocks] = useState<ConceptStock[]>([]);
  const [expandingLoading, setExpandingLoading] = useState(false);
  const [overlap, setOverlap] = useState<WatchlistOverlap[]>([]);
  const [overlapLoading, setOverlapLoading] = useState(true);

  const fetchConcepts = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await hotConceptsApi.list();
      setConcepts(res.data);
    } catch {
      setError('加载热门概念失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchOverlap = useCallback(async () => {
    setOverlapLoading(true);
    try {
      const res = await hotConceptsApi.watchlistOverlap();
      const data = res.data;
      setOverlap(Array.isArray(data) ? data : []);
    } catch {
      // ignore
    } finally {
      setOverlapLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConcepts();
    fetchOverlap();
  }, [fetchConcepts, fetchOverlap]);

  const toggleExpand = useCallback(async (code: string) => {
    if (expandedCode === code) {
      setExpandedCode(null);
      setExpandedStocks([]);
      return;
    }
    setExpandedCode(code);
    setExpandingLoading(true);
    try {
      const res = await hotConceptsApi.stocks(code);
      const data = res.data;
      setExpandedStocks(Array.isArray(data) ? data : []);
    } catch {
      setExpandedStocks([]);
    } finally {
      setExpandingLoading(false);
    }
  }, [expandedCode]);

  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

  const sorted = [...concepts].sort((a, b) => b.change_pct - a.change_pct);

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Flame className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <div>
            <h1 className="text-xl font-bold">热门概念</h1>
            <p className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>
              当前市场热门概念板块
            </p>
          </div>
        </div>
        <button
          onClick={() => { fetchConcepts(); fetchOverlap(); }}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors"
          style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {/* Error */}
      {error && (
        <div className="mb-4">
          <ErrorState
            title="加载失败"
            description={error}
            onRetry={() => { setError(''); fetchConcepts(); fetchOverlap(); }}
          />
        </div>
      )}

      {/* Loading */}
      {loading && concepts.length === 0 ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <SkeletonGridCard key={i} />
          ))}
        </div>
      ) : concepts.length === 0 ? (
        <EmptyState
          icon={Flame}
          title="暂无热门概念"
          description="暂无概念热点数据"
        />
      ) : (
        /* Concept card grid */
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 mb-6">
          {sorted.map((concept) => {
            const isExpanded = expandedCode === concept.code;
            const pct = concept.change_pct;
            const Icon = pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Flame;

            return (
              <div
                key={concept.code}
                className="rounded-lg border overflow-hidden transition-all"
                style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
              >
                {/* Card header */}
                <button
                  onClick={() => toggleExpand(concept.code)}
                  className="w-full text-left p-4 transition-colors hover:bg-[var(--color-bg-hover)]"
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Icon className="w-4 h-4" style={{ color: changeColor(pct) }} />
                      <span className="font-medium text-sm">{concept.name}</span>
                    </div>
                    {isExpanded ? (
                      <ChevronUp className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
                    ) : (
                      <ChevronDown className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
                    )}
                  </div>
                  <div className="flex items-center justify-between">
                    <span
                      className="font-mono text-sm font-bold"
                      style={{ color: changeColor(pct) }}
                    >
                      {pct >= 0 ? '+' : ''}{pct.toFixed(2)}%
                    </span>
                    <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {concept.stock_count} 只股票
                    </span>
                  </div>

                  {/* Leader info */}
                  {concept.leader_name && (
                    <div
                      className="mt-2 flex items-center gap-1.5 text-xs rounded-md px-2 py-1"
                      style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}
                    >
                      <Star className="w-3 h-3" style={{ color: 'var(--color-accent)' }} />
                      <span>龙头: {concept.leader_name}</span>
                      <span className="font-mono" style={{ color: changeColor(concept.leader_change_pct) }}>
                        {concept.leader_change_pct >= 0 ? '+' : ''}{concept.leader_change_pct.toFixed(2)}%
                      </span>
                    </div>
                  )}
                </button>

                {/* Expanded stocks */}
                {isExpanded && (
                  <div className="border-t" style={{ borderColor: 'var(--color-border)' }}>
                    {expandingLoading ? (
                      <div className="p-4">
                        <SkeletonInlineTable rows={4} columns={4} />
                      </div>
                    ) : expandedStocks.length === 0 ? (
                      <div className="text-center py-6 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                        暂无成分股数据
                      </div>
                    ) : (
                      <div className="max-h-[240px] overflow-y-auto">
                        {expandedStocks.map((stock, idx) => (
                          <div
                            key={`${stock.code}-${idx}`}
                            className="flex items-center justify-between px-4 py-2 border-b last:border-b-0 transition-colors hover:bg-[var(--color-bg-hover)]"
                            style={{ borderColor: 'var(--color-border)' }}
                          >
                            <div className="flex items-center gap-2 min-w-0">
                              <span className="text-xs font-mono font-medium" style={{ color: 'var(--color-accent)' }}>
                                {stock.code}
                              </span>
                              <span className="text-xs truncate" style={{ color: 'var(--color-text-primary)' }}>
                                {stock.name}
                              </span>
                            </div>
                            <div className="flex items-center gap-3 flex-shrink-0">
                              {stock.price > 0 && (
                                <span className="text-xs font-mono" style={{ color: 'var(--color-text-secondary)' }}>
                                  {stock.price.toFixed(2)}
                                </span>
                              )}
                              <span
                                className="text-xs font-mono font-medium"
                                style={{ color: changeColor(stock.change_pct) }}
                              >
                                {stock.change_pct >= 0 ? '+' : ''}{stock.change_pct.toFixed(2)}%
                              </span>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Watchlist-concept overlap section */}
      <div className="mt-6">
        <div className="flex items-center gap-2 mb-3">
          <Link2 className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <h2 className="text-sm font-bold">自选股 × 热门概念</h2>
        </div>

        {overlapLoading ? (
          <SkeletonList rows={3} />
        ) : overlap.length === 0 ? (
          <div
            className="text-center py-8 rounded-lg border"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
              暂无自选股与热门概念的交集数据
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {overlap.map((item, idx) => (
              <div
                key={`${item.code}-${idx}`}
                className="rounded-lg border p-3 transition-colors hover:bg-[var(--color-bg-hover)]"
                style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
              >
                <div className="flex items-center gap-2 mb-2">
                  <span className="font-mono text-xs font-medium" style={{ color: 'var(--color-accent)' }}>
                    {item.code}
                  </span>
                  <span className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
                    {item.name}
                  </span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {item.concepts.map((c) => (
                    <span
                      key={c}
                      className="text-xs px-2 py-0.5 rounded-full"
                      style={{ background: 'var(--color-bg-hover)', color: 'var(--color-accent)' }}
                    >
                      {c}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
