import { useState, useEffect, useCallback } from 'react';
import { marketApi, type NewsItem } from '@/lib/api';
import { RefreshCw, ExternalLink, Newspaper } from 'lucide-react';
import EmptyState from '@/components/EmptyState';
import { SkeletonList } from '@/components/ui/Skeleton';

export default function NewsPage() {
  const [news, setNews] = useState<NewsItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

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
        <div
          className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
        >
          {error}
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
          {news.map((item, i) => (
            <a
              key={i}
              href={item.url}
              target="_blank"
              rel="noopener noreferrer"
              className="block rounded-lg border p-4 hover:bg-[var(--color-bg-hover)] transition-colors"
              style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <h3 className="font-medium text-sm mb-1 line-clamp-2">{item.title}</h3>
                  {item.summary && (
                    <p
                      className="text-xs line-clamp-2 mb-2"
                      style={{ color: 'var(--color-text-muted)' }}
                    >
                      {item.summary}
                    </p>
                  )}
                  <div className="flex items-center gap-3 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                    {item.source && <span>{item.source}</span>}
                    <span>{formatTime(item.published_at)}</span>
                  </div>
                </div>
                <ExternalLink className="w-4 h-4 flex-shrink-0 mt-0.5" style={{ color: 'var(--color-text-muted)' }} />
              </div>
            </a>
          ))}
        </div>
      )}
    </div>
  );
}
