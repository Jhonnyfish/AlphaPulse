import { useState, useEffect, useCallback, useMemo } from 'react';
import { useView } from '@/lib/ViewContext';
import { marketApi, type NewsItem } from '@/lib/api';
import { RefreshCw, ExternalLink, Newspaper, TrendingUp, TrendingDown, Minus } from 'lucide-react';
import EmptyState from '@/components/EmptyState';
import ErrorState from '@/components/ErrorState';
import { SkeletonList } from '@/components/ui/Skeleton';

/* ── Sentiment analysis ──────────────────────────────────── */

type Sentiment = 'positive' | 'negative' | 'neutral';

const POSITIVE_KEYWORDS = ['涨', '利好', '突破', '新高', '上涨', '大涨', '涨停', '暴涨', '增长', '盈利', '翻倍', '飙升', '反弹', '回暖', '看好', '推荐', '买入', '强势', '领涨', '加仓', '增持', '超预期'];
const NEGATIVE_KEYWORDS = ['跌', '利空', '减持', '暴雷', '下跌', '大跌', '跌停', '暴跌', '亏损', '退市', '处罚', '违规', '踩雷', '爆仓', '预警', '看空', '卖出', '弱势', '领跌', '减仓', '低于预期', '风险'];

function inferSentiment(title: string, summary?: string): Sentiment {
  const text = `${title} ${summary || ''}`;
  const posCount = POSITIVE_KEYWORDS.filter((kw) => text.includes(kw)).length;
  const negCount = NEGATIVE_KEYWORDS.filter((kw) => text.includes(kw)).length;
  if (posCount > negCount) return 'positive';
  if (negCount > posCount) return 'negative';
  return 'neutral';
}

const SENTIMENT_CONFIG: Record<Sentiment, { label: string; bg: string; color: string; border: string; icon: typeof TrendingUp }> = {
  positive: {
    label: '利好',
    bg: 'rgba(34, 197, 94, 0.15)',
    color: '#22c55e',
    border: 'rgba(34, 197, 94, 0.3)',
    icon: TrendingUp,
  },
  negative: {
    label: '利空',
    bg: 'rgba(239, 68, 68, 0.15)',
    color: '#ef4444',
    border: 'rgba(239, 68, 68, 0.3)',
    icon: TrendingDown,
  },
  neutral: {
    label: '中性',
    bg: 'rgba(107, 114, 128, 0.15)',
    color: '#6b7280',
    border: 'rgba(107, 114, 128, 0.3)',
    icon: Minus,
  },
};

/* ── Related stock extraction ────────────────────────────── */

interface RelatedStock {
  code: string;
  name: string;
}

// Extract stock codes (6-digit patterns) from text
function extractStockCodes(title: string, summary?: string): RelatedStock[] {
  const text = `${title} ${summary || ''}`;
  const codePattern = /[（(]?(?:SH|SZ|sh|sz)?\.?(\d{6})[)）]?/g;
  const codes: RelatedStock[] = [];
  let match;
  const seen = new Set<string>();
  while ((match = codePattern.exec(text)) !== null) {
    const code = match[1];
    if (!seen.has(code)) {
      seen.add(code);
      codes.push({ code, name: '' });
    }
  }
  return codes;
}

// Well-known stock name mapping for common tickers
const STOCK_NAMES: Record<string, string> = {
  '600519': '贵州茅台', '000858': '五粮液', '601318': '中国平安',
  '600036': '招商银行', '000001': '平安银行', '600900': '长江电力',
  '601012': '隆基绿能', '300750': '宁德时代', '002594': '比亚迪',
  '601899': '紫金矿业', '600276': '恒瑞医药', '000333': '美的集团',
  '002415': '海康威视', '600887': '伊利股份', '601888': '中国中免',
  '300059': '东方财富', '002475': '立讯精密', '603259': '药明康德',
  '600585': '海螺水泥', '002714': '牧原股份', '601398': '工商银行',
  '601288': '农业银行', '600030': '中信证券', '601857': '中国石油',
  '600028': '中国石化', '000002': '万科A', '600009': '上海机场',
  '603288': '海天味业', '300124': '汇川技术', '002032': '苏泊尔',
  '688981': '中芯国际', '688012': '中微公司', '300661': '圣邦股份',
  '600809': '山西汾酒', '000568': '泸州老窖', '002304': '洋河股份',
  '601166': '兴业银行', '600000': '浦发银行', '000725': '京东方A',
};

function getStockName(code: string): string {
  return STOCK_NAMES[code] || code;
}

// Generate mock related stocks for news that don't have explicit codes
function getMockRelatedStocks(title: string, summary?: string, existingCode?: string): RelatedStock[] {
  const text = `${title} ${summary || ''}`;
  const stocks: RelatedStock[] = [];

  // If the news item already has a code, include it
  if (existingCode) {
    const code = existingCode.replace(/[^0-9]/g, '');
    if (code.length === 6) {
      stocks.push({ code, name: getStockName(code) });
    }
  }

  // Keyword-based mock association
  const sectorKeywords: Record<string, RelatedStock[]> = {
    '半导体': [{ code: '688981', name: '中芯国际' }, { code: '300661', name: '圣邦股份' }],
    '芯片': [{ code: '688981', name: '中芯国际' }, { code: '688012', name: '中微公司' }],
    '新能源': [{ code: '300750', name: '宁德时代' }, { code: '601012', name: '隆基绿能' }],
    '锂电': [{ code: '300750', name: '宁德时代' }],
    '白酒': [{ code: '600519', name: '贵州茅台' }, { code: '000858', name: '五粮液' }],
    '酒': [{ code: '600519', name: '贵州茅台' }, { code: '600809', name: '山西汾酒' }],
    '银行': [{ code: '601398', name: '工商银行' }, { code: '600036', name: '招商银行' }],
    '券商': [{ code: '600030', name: '中信证券' }, { code: '300059', name: '东方财富' }],
    '证券': [{ code: '600030', name: '中信证券' }],
    '汽车': [{ code: '002594', name: '比亚迪' }],
    '新能源车': [{ code: '002594', name: '比亚迪' }, { code: '300750', name: '宁德时代' }],
    '医药': [{ code: '600276', name: '恒瑞医药' }, { code: '603259', name: '药明康德' }],
    '创新药': [{ code: '600276', name: '恒瑞医药' }],
    '地产': [{ code: '000002', name: '万科A' }],
    '房地产': [{ code: '000002', name: '万科A' }],
    '光伏': [{ code: '601012', name: '隆基绿能' }],
    '消费': [{ code: '600519', name: '贵州茅台' }, { code: '000333', name: '美的集团' }],
    'AI': [{ code: '002415', name: '海康威视' }],
    '人工智能': [{ code: '002415', name: '海康威视' }],
    '算力': [{ code: '002415', name: '海康威视' }],
    '石油': [{ code: '601857', name: '中国石油' }],
    '保险': [{ code: '601318', name: '中国平安' }],
    '平安': [{ code: '601318', name: '中国平安' }],
    '中芯': [{ code: '688981', name: '中芯国际' }],
    '宁德': [{ code: '300750', name: '宁德时代' }],
    '茅台': [{ code: '600519', name: '贵州茅台' }],
    '比亚迪': [{ code: '002594', name: '比亚迪' }],
  };

  const seenCodes = new Set(stocks.map((s) => s.code));
  for (const [keyword, related] of Object.entries(sectorKeywords)) {
    if (text.includes(keyword)) {
      for (const s of related) {
        if (!seenCodes.has(s.code)) {
          seenCodes.add(s.code);
          stocks.push(s);
          if (stocks.length >= 3) break;
        }
      }
      if (stocks.length >= 3) break;
    }
  }

  return stocks.slice(0, 3);
}

/* ── Sentiment Badge Component ───────────────────────────── */

function SentimentBadge({ sentiment }: { sentiment: Sentiment }) {
  const cfg = SENTIMENT_CONFIG[sentiment];
  const Icon = cfg.icon;
  return (
    <span
      className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium flex-shrink-0"
      style={{
        background: cfg.bg,
        color: cfg.color,
        border: `1px solid ${cfg.border}`,
      }}
    >
      <Icon className="w-3 h-3" />
      {cfg.label}
    </span>
  );
}

/* ── Stock Pill Component ────────────────────────────────── */

function StockPill({ code, name, onClick }: RelatedStock & { onClick: () => void }) {
  return (
    <button
      onClick={(e) => {
        e.stopPropagation();
        e.preventDefault();
        onClick();
      }}
      className="inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors hover:brightness-125 flex-shrink-0"
      style={{
        background: 'rgba(59, 130, 246, 0.12)',
        color: '#60a5fa',
        border: '1px solid rgba(59, 130, 246, 0.25)',
      }}
      title={`${name || code} (${code})`}
    >
      <span style={{ color: '#94a3b8', fontSize: '10px' }}>⊙</span>
      {name || code}
      {name && (
        <span style={{ color: '#64748b', fontSize: '10px' }}>{code}</span>
      )}
    </button>
  );
}

/* ── Enriched news item ──────────────────────────────────── */

interface EnrichedNewsItem extends NewsItem {
  sentiment: Sentiment;
  relatedStocks: RelatedStock[];
}

/* ── Main Page ───────────────────────────────────────────── */

export default function NewsPage() {
  const [news, setNews] = useState<NewsItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { navigate } = useView();

  const fetchNews = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await marketApi.news();
      setNews(res.data);
    } catch {
      setError('加载资讯失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchNews();
  }, [fetchNews]);

  // Enrich news with sentiment and related stocks
  const enrichedNews: EnrichedNewsItem[] = useMemo(() => {
    return news.map((item) => {
      // Use API sentiment if available, otherwise infer
      const sentiment: Sentiment =
        (item as unknown as Record<string, unknown>).sentiment === 'positive'
          ? 'positive'
          : (item as unknown as Record<string, unknown>).sentiment === 'negative'
            ? 'negative'
            : (item as unknown as Record<string, unknown>).sentiment === 'neutral'
              ? 'neutral'
              : inferSentiment(item.title, item.summary);

      // Extract explicit stock codes from text, or generate mock ones
      let relatedStocks = extractStockCodes(item.title, item.summary);
      if (relatedStocks.length === 0) {
        relatedStocks = getMockRelatedStocks(item.title, item.summary, item.code);
      } else {
        // Fill in names for extracted codes
        relatedStocks = relatedStocks.map((s) => ({
          ...s,
          name: getStockName(s.code),
        }));
      }

      return {
        ...item,
        sentiment,
        relatedStocks,
      };
    });
  }, [news]);

  const formatTime = (iso: string) => {
    try {
      const d = new Date(iso);
      return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
    } catch {
      return iso;
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-bold">市场资讯</h1>
        <button
          onClick={fetchNews}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm hover:bg-[var(--color-bg-hover)] transition-colors"
          style={{ color: 'var(--color-text-secondary)' }}
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {error && (
        <div className="mb-4">
          <ErrorState
            title="加载失败"
            description={error}
            onRetry={() => { setError(''); fetchNews(); }}
          />
        </div>
      )}

      {loading && news.length === 0 ? (
        <SkeletonList rows={6} />
      ) : news.length === 0 ? (
        <EmptyState
          icon={Newspaper}
          title="暂无新闻"
          description="暂无最新资讯，请稍后刷新"
        />
      ) : (
        <div className="space-y-3">
          {enrichedNews.map((item, i) => (
            <div
              key={i}
              className="rounded-lg border p-4 hover:bg-[var(--color-bg-hover)] transition-colors"
              style={{
                background: 'var(--color-bg-secondary)',
                borderColor: 'var(--color-border)',
                backdropFilter: 'blur(12px)',
              }}
            >
              <a
                href={item.url}
                target="_blank"
                rel="noopener noreferrer"
                className="block"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex-1 min-w-0">
                    {/* Title row with sentiment badge */}
                    <div className="flex items-start gap-2 mb-1">
                      <h3 className="font-medium text-sm line-clamp-2 flex-1">{item.title}</h3>
                      <SentimentBadge sentiment={item.sentiment} />
                    </div>

                    {/* Summary */}
                    {item.summary && (
                      <p
                        className="text-xs line-clamp-2 mb-2"
                        style={{ color: 'var(--color-text-muted)' }}
                      >
                        {item.summary}
                      </p>
                    )}

                    {/* Related stocks */}
                    {item.relatedStocks.length > 0 && (
                      <div className="flex items-center gap-1.5 flex-wrap mb-2">
                        <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
                          关联:
                        </span>
                        {item.relatedStocks.map((stock) => (
                          <StockPill
                            key={stock.code}
                            code={stock.code}
                            name={stock.name}
                            onClick={() => navigate('analyze', { code: stock.code })}
                          />
                        ))}
                      </div>
                    )}

                    {/* Meta info */}
                    <div className="flex items-center gap-3 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {item.source && <span>{item.source}</span>}
                      <span>{formatTime(item.published_at)}</span>
                    </div>
                  </div>
                  <ExternalLink className="w-4 h-4 flex-shrink-0 mt-0.5" style={{ color: 'var(--color-text-muted)' }} />
                </div>
              </a>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
