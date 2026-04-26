import { useState, useEffect } from 'react';
import api from '@/lib/api';
import { FileText, RefreshCw, TrendingUp, TrendingDown, Star, AlertTriangle } from 'lucide-react';

interface BriefStock {
  code: string;
  name: string;
  price: number;
  change_pct: number;
}

interface DailyBriefData {
  ok: boolean;
  market: {
    indices: { code: string; name: string; price: number; change_pct: number }[] | null;
    breadth: { advancing: number; declining: number; flat: number } | null;
  };
  sectors: { name: string; change_pct: number }[] | null;
  watchlist: {
    count: number;
    top: BriefStock[];
    bottom: BriefStock[];
  };
}

export default function DailyBriefPage() {
  const [data, setData] = useState<DailyBriefData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = () => {
    setLoading(true);
    setError('');
    api
      .get<DailyBriefData>('/daily-brief')
      .then((res) => setData(res.data))
      .catch(() => setError('加载每日简报失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchData();
  }, []);

  const formatChange = (val: number) => {
    const prefix = val > 0 ? '+' : '';
    return `${prefix}${val.toFixed(2)}%`;
  };

  const getChangeColor = (val: number) => {
    if (val > 0) return 'var(--color-danger)';
    if (val < 0) return 'var(--color-success)';
    return 'var(--color-text-muted)';
  };

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <FileText className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">每日简报</h1>
        </div>
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '80px' }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <FileText className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">每日简报</h1>
        </div>
        <div className="text-sm px-4 py-3 rounded-lg" style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
          <button onClick={fetchData} className="ml-3 underline">重试</button>
        </div>
      </div>
    );
  }

  if (!data) return null;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <FileText className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">每日简报</h1>
        </div>
        <button onClick={fetchData} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} style={{ color: 'var(--color-text-muted)' }} />
        </button>
      </div>

      {/* Market indices */}
      {data.market?.indices && data.market.indices.length > 0 && (
        <div className="glass-panel rounded-xl p-4 mb-4" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>大盘指数</h3>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
            {data.market.indices.map((idx) => (
              <div key={idx.code} className="flex items-center justify-between p-2 rounded-lg" style={{ background: 'var(--color-bg-hover)' }}>
                <span className="text-sm">{idx.name}</span>
                <div className="text-right">
                  <div className="font-mono text-sm">{idx.price.toFixed(2)}</div>
                  <div className="font-mono text-xs" style={{ color: getChangeColor(idx.change_pct) }}>
                    {formatChange(idx.change_pct)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Breadth */}
      {data.market?.breadth && (
        <div className="glass-panel rounded-xl p-4 mb-4" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>涨跌统计</h3>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <TrendingUp className="w-4 h-4" style={{ color: 'var(--color-danger)' }} />
              <span className="font-mono" style={{ color: 'var(--color-danger)' }}>{data.market.breadth.advancing}</span>
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>上涨</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="font-mono" style={{ color: 'var(--color-text-muted)' }}>{data.market.breadth.flat}</span>
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>平盘</span>
            </div>
            <div className="flex items-center gap-2">
              <TrendingDown className="w-4 h-4" style={{ color: 'var(--color-success)' }} />
              <span className="font-mono" style={{ color: 'var(--color-success)' }}>{data.market.breadth.declining}</span>
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>下跌</span>
            </div>
          </div>
        </div>
      )}

      {/* Watchlist */}
      {data.watchlist && data.watchlist.count > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          {/* Top performers */}
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <div className="flex items-center gap-2 mb-3">
              <Star className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
              <h3 className="text-sm font-medium" style={{ color: 'var(--color-text-secondary)' }}>自选强势</h3>
            </div>
            <div className="space-y-2">
              {data.watchlist.top.map((s) => (
                <div key={s.code} className="flex items-center justify-between p-2 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
                  <div>
                    <span className="text-sm">{s.name}</span>
                    <span className="text-xs ml-2" style={{ color: 'var(--color-text-muted)' }}>{s.code}</span>
                  </div>
                  <div className="text-right">
                    <span className="font-mono text-sm">{s.price.toFixed(2)}</span>
                    <span className="font-mono text-xs ml-2" style={{ color: getChangeColor(s.change_pct) }}>
                      {formatChange(s.change_pct)}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Bottom performers */}
          <div className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
            <div className="flex items-center gap-2 mb-3">
              <AlertTriangle className="w-4 h-4" style={{ color: 'var(--color-danger)' }} />
              <h3 className="text-sm font-medium" style={{ color: 'var(--color-text-secondary)' }}>自选弱势</h3>
            </div>
            <div className="space-y-2">
              {data.watchlist.bottom.map((s) => (
                <div key={s.code} className="flex items-center justify-between p-2 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
                  <div>
                    <span className="text-sm">{s.name}</span>
                    <span className="text-xs ml-2" style={{ color: 'var(--color-text-muted)' }}>{s.code}</span>
                  </div>
                  <div className="text-right">
                    <span className="font-mono text-sm">{s.price.toFixed(2)}</span>
                    <span className="font-mono text-xs ml-2" style={{ color: getChangeColor(s.change_pct) }}>
                      {formatChange(s.change_pct)}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Empty state */}
      {!data.watchlist?.count && !data.market?.indices?.length && (
        <div
          className="text-center py-16 rounded-xl"
          style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}
        >
          <FileText className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>暂无简报数据</p>
        </div>
      )}
    </div>
  );
}
