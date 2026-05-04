import { useEffect, useMemo, useRef, useState } from 'react';
import { AlertCircle, Loader2, Search, Target, X } from 'lucide-react';
import { candidatesApi, type CandidateListItem } from '@/lib/api';

interface Alpha300SelectorProps {
  open: boolean;
  onClose: () => void;
  onSelect: (code: string) => void;
}

const CACHE_TTL = 6 * 60 * 60 * 1000; // 6 hours — Alpha300 list refreshes daily
const LS_KEY = 'alpha300_cache';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
let cachedItems: CandidateListItem[] | null = null;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
let cachedAt = 0;
let inflightRequest: Promise<CandidateListItem[]> | null = null;

function loadFromStorage(): CandidateListItem[] | null {
  try {
    const raw = localStorage.getItem(LS_KEY);
    if (!raw) return null;
    const { items, ts } = JSON.parse(raw);
    if (Date.now() - ts > CACHE_TTL) {
      localStorage.removeItem(LS_KEY);
      return null;
    }
    cachedItems = items;
    cachedAt = ts;
    return items;
  } catch {
    return null;
  }
}

function saveToStorage(items: CandidateListItem[]) {
  try {
    localStorage.setItem(LS_KEY, JSON.stringify({ items, ts: Date.now() }));
  } catch { /* quota exceeded, ignore */ }
}

function tierStyle(tier: string) {
  switch (tier) {
    case 'focus':
      return {
        label: '重点',
        background: 'rgba(239,68,68,0.14)',
        color: '#ef4444',
      };
    case 'observe':
      return {
        label: '观察',
        background: 'rgba(245,158,11,0.14)',
        color: '#f59e0b',
      };
    default:
      return {
        label: '默认',
        background: 'rgba(107,114,128,0.14)',
        color: '#9ca3af',
      };
  }
}

async function fetchAlpha300Candidates(force = false) {
  // Always prefer localStorage — only hit API when empty or forced
  if (!force) {
    const stored = loadFromStorage();
    if (stored) return stored;
  }

  if (!force && inflightRequest) {
    return inflightRequest;
  }

  inflightRequest = candidatesApi
    .list({ limit: 300 })
    .then((res) => {
      const items = res.data?.data?.items ?? [];
      cachedItems = items;
      cachedAt = Date.now();
      saveToStorage(items);
      return items;
    })
    .finally(() => {
      inflightRequest = null;
    });

  return inflightRequest;
}

export default function Alpha300Selector({
  open,
  onClose,
  onSelect,
}: Alpha300SelectorProps) {
  const [items, setItems] = useState<CandidateListItem[]>([]);
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  const loadItems = async (force = false) => {
    setLoading(true);
    setError('');
    try {
      const nextItems = await fetchAlpha300Candidates(force);
      setItems(nextItems);
    } catch {
      setError('加载 Alpha300 候选失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!open) return;

    setQuery('');
    void loadItems();

    const raf = requestAnimationFrame(() => inputRef.current?.focus());
    return () => cancelAnimationFrame(raf);
  }, [open]);

  useEffect(() => {
    if (!open) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [open, onClose]);

  const filteredItems = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return items;
    return items.filter(
      (item) =>
        item.code.toLowerCase().includes(needle) ||
        item.name.toLowerCase().includes(needle)
    );
  }, [items, query]);

  if (!open) return null;

  return (
    <div
      className="modal-backdrop animate-fade-in"
      style={{ zIndex: 220 }}
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label="Alpha300 快速选股"
    >
      <div
        className="modal-content animate-scale-in w-full h-full sm:h-auto sm:max-h-[80vh] max-w-2xl mx-0 sm:mx-4 rounded-none sm:rounded-2xl overflow-hidden"
        onClick={(event) => event.stopPropagation()}
      >
        <div
          className="flex items-center gap-3 px-4 py-3 border-b"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <div
            className="w-9 h-9 rounded-xl flex items-center justify-center shrink-0"
            style={{
              background: 'rgba(59,130,246,0.14)',
              color: 'var(--color-accent)',
            }}
          >
            <Target className="w-4.5 h-4.5" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="text-sm font-semibold">Alpha300 快速选股</div>
            <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
              搜索代码或名称，点击后回填输入框
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-2 rounded-lg transition-colors hover:bg-[var(--color-bg-hover)]"
            aria-label="关闭 Alpha300 选择器"
          >
            <X className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
          </button>
        </div>

        <div className="p-4 border-b" style={{ borderColor: 'var(--color-border)' }}>
          <div className="relative">
            <Search
              className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4"
              style={{ color: 'var(--color-text-muted)' }}
            />
            <input
              ref={inputRef}
              type="text"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="搜索股票代码或名称..."
              role="combobox"
              aria-expanded={filteredItems.length > 0}
              aria-controls="alpha300-listbox"
              aria-label="搜索股票代码或名称"
              className="w-full pl-9 pr-3 py-2.5 rounded-xl border text-sm outline-none"
              style={{
                background: 'var(--color-bg-card)',
                borderColor: 'var(--color-border)',
                color: 'var(--color-text-primary)',
              }}
            />
          </div>
        </div>

        <div className="px-4 py-2 text-xs flex items-center justify-between" style={{ color: 'var(--color-text-muted)' }}>
          <span>Alpha300 候选池</span>
          <span>{filteredItems.length} 条结果</span>
        </div>

        <div className="px-4 pb-4">
          <div
            className="rounded-2xl border overflow-hidden"
            style={{
              background: 'rgba(15,23,42,0.48)',
              borderColor: 'var(--color-border)',
            }}
          >
            {loading ? (
              <div className="h-[50vh] sm:h-[420px] flex items-center justify-center gap-2 text-sm" style={{ color: 'var(--color-text-muted)' }}>
                <Loader2 className="w-4 h-4 animate-spin" />
                加载 Alpha300 中...
              </div>
            ) : error ? (
              <div className="h-[50vh] sm:h-[420px] flex flex-col items-center justify-center gap-3 px-6 text-center">
                <AlertCircle className="w-5 h-5" style={{ color: 'var(--color-danger)' }} />
                <div className="text-sm" style={{ color: 'var(--color-danger)' }}>{error}</div>
                <button
                  type="button"
                  onClick={() => void loadItems(true)}
                  className="px-3 py-1.5 rounded-lg text-sm"
                  style={{ background: 'var(--color-accent)', color: '#fff' }}
                >
                  重试
                </button>
              </div>
            ) : filteredItems.length === 0 ? (
              <div className="h-[50vh] sm:h-[420px] flex items-center justify-center text-sm" style={{ color: 'var(--color-text-muted)' }}>
                未找到匹配的 Alpha300 标的
              </div>
            ) : (
              <div id="alpha300-listbox" role="listbox" aria-label="Alpha300 候选列表" className="max-h-[50vh] sm:max-h-[420px] overflow-y-auto divide-y" style={{ borderColor: 'var(--color-border)' }}>
                {filteredItems.map((item) => {
                  const tier = tierStyle(item.recommendation_tier);
                  return (
                    <button
                      key={item.code}
                      type="button"
                      role="option"
                      onClick={() => {
                        onSelect(item.code);
                        onClose();
                      }}
                      className="w-full px-4 py-3 flex items-center gap-3 text-left transition-colors hover:bg-[var(--color-bg-hover)]"
                    >
                      <div className="w-10 text-center shrink-0">
                        <div className="text-[11px]" style={{ color: 'var(--color-text-muted)' }}>#{item.rank}</div>
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-mono text-xs px-2 py-0.5 rounded-lg" style={{ background: 'var(--color-bg-secondary)', color: 'var(--color-accent)' }}>
                            {item.code}
                          </span>
                          <span className="text-sm font-medium" style={{ color: 'var(--color-text-primary)' }}>
                            {item.name}
                          </span>
                        </div>
                        <div className="mt-1 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                          评分 {item.score.toFixed(2)}
                        </div>
                      </div>
                      <span
                        className="text-[11px] px-2 py-1 rounded-full shrink-0"
                        style={{ background: tier.background, color: tier.color }}
                      >
                        {tier.label}
                      </span>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
