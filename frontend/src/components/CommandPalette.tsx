import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { useView, type ViewName } from '@/lib/ViewContext';
import {
  Search, LayoutDashboard, Star, TrendingUp, CandlestickChart,
  Activity, BarChart3, GitCompareArrows, Droplets, Target, Filter,
  Flame, Crown, Briefcase, BookOpen, Zap, Radio, Grid3X3,
  Newspaper, Settings, ArrowRight,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

/* ── Page commands ────────────────────────────────────── */
interface PageCommand {
  id: string;
  type: 'page';
  label: string;
  keywords: string[];
  view: ViewName;
  icon: LucideIcon;
}

const pages: PageCommand[] = [
  { id: 'p-dashboard', type: 'page', label: '总览', keywords: ['dashboard', '总览', '首页'], view: 'dashboard', icon: LayoutDashboard },
  { id: 'p-watchlist', type: 'page', label: '自选股', keywords: ['watchlist', '自选', '自选股'], view: 'watchlist', icon: Star },
  { id: 'p-market', type: 'page', label: '行情', keywords: ['market', '行情', '大盘'], view: 'market', icon: TrendingUp },
  { id: 'p-kline', type: 'page', label: 'K线', keywords: ['kline', 'k线', 'k线图', 'candlestick'], view: 'kline', icon: CandlestickChart },
  { id: 'p-analyze', type: 'page', label: '个股分析', keywords: ['analyze', '分析', '个股', '深度分析'], view: 'analyze', icon: Activity },
  { id: 'p-sectors', type: 'page', label: '板块', keywords: ['sectors', '板块', '行业'], view: 'sectors', icon: BarChart3 },
  { id: 'p-compare', type: 'page', label: '对比', keywords: ['compare', '对比', '比较'], view: 'compare', icon: GitCompareArrows },
  { id: 'p-flow', type: 'page', label: '资金流向', keywords: ['flow', '资金', '流向', '主力'], view: 'flow', icon: Droplets },
  { id: 'p-candidates', type: 'page', label: '候选股', keywords: ['candidates', '候选', '候选股'], view: 'candidates', icon: Target },
  { id: 'p-screener', type: 'page', label: '选股器', keywords: ['screener', '选股', '筛选'], view: 'screener', icon: Filter },
  { id: 'p-concepts', type: 'page', label: '热门概念', keywords: ['concepts', '概念', '热门'], view: 'hot-concepts', icon: Flame },
  { id: 'p-dragon', type: 'page', label: '龙虎榜', keywords: ['dragon', '龙虎', '龙虎榜'], view: 'dragon-tiger', icon: Crown },
  { id: 'p-portfolio', type: 'page', label: '持仓', keywords: ['portfolio', '持仓', '仓位'], view: 'portfolio', icon: Briefcase },
  { id: 'p-journal', type: 'page', label: '交易日志', keywords: ['journal', '日志', '交易'], view: 'journal', icon: BookOpen },
  { id: 'p-strategies', type: 'page', label: '策略', keywords: ['strategies', '策略'], view: 'strategies', icon: Zap },
  { id: 'p-signals', type: 'page', label: '信号', keywords: ['signals', '信号', '买卖'], view: 'signals', icon: Radio },
  { id: 'p-watchlist-analysis', type: 'page', label: '自选分析', keywords: ['watchlist analysis', '自选分析'], view: 'watchlist-analysis', icon: Grid3X3 },
  { id: 'p-news', type: 'page', label: '资讯', keywords: ['news', '资讯', '新闻'], view: 'news', icon: Newspaper },
  { id: 'p-settings', type: 'page', label: '设置', keywords: ['settings', '设置', '配置'], view: 'settings', icon: Settings },
];

type Command = PageCommand;

/* ── Component ────────────────────────────────────────── */
interface CommandPaletteProps {
  open: boolean;
  onClose: () => void;
}

export default function CommandPalette({ open, onClose }: CommandPaletteProps) {
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const { navigate } = useView();

  // Filter commands
  const filtered = useMemo(() => {
    if (!query.trim()) return pages;
    const q = query.toLowerCase();
    return pages.filter(
      p =>
        p.label.toLowerCase().includes(q) ||
        p.keywords.some(k => k.includes(q))
    );
  }, [query]);

  // Reset state when opening
  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setQuery('');
      setSelectedIndex(0);
      // Delay focus to next frame so the element is mounted
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [open]);

  // Keep selected item in view
  useEffect(() => {
    const list = listRef.current;
    if (!list) return;
    const item = list.children[selectedIndex] as HTMLElement | undefined;
    if (item) {
      item.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex]);

  // Reset selected index when filter changes
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setSelectedIndex(0);
  }, [query]);

  const executeCommand = useCallback(
    (cmd: Command) => {
      if (cmd.type === 'page') {
        navigate(cmd.view);
      }
      onClose();
    },
    [navigate, onClose]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          setSelectedIndex(i => Math.min(i + 1, filtered.length - 1));
          break;
        case 'ArrowUp':
          e.preventDefault();
          setSelectedIndex(i => Math.max(i - 1, 0));
          break;
        case 'Enter':
          e.preventDefault();
          if (filtered[selectedIndex]) {
            executeCommand(filtered[selectedIndex]);
          }
          break;
        case 'Escape':
          e.preventDefault();
          onClose();
          break;
      }
    },
    [filtered, selectedIndex, executeCommand, onClose]
  );

  if (!open) return null;

  return (
    <div
      className="modal-backdrop animate-fade-in"
      onClick={onClose}
      style={{ zIndex: 200 }}
      role="dialog"
      aria-modal="true"
      aria-label="搜索与导航"
    >
      <div
        className="modal-content animate-scale-in w-full max-w-lg mx-4"
        onClick={e => e.stopPropagation()}
        onKeyDown={handleKeyDown}
      >
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b" style={{ borderColor: 'var(--color-border)' }}>
          <Search className="w-5 h-5 shrink-0" style={{ color: 'var(--color-text-muted)' }} />
          <input
            ref={inputRef}
            type="text"
            placeholder="搜索页面或输入股票代码..."
            value={query}
            onChange={e => setQuery(e.target.value)}
            className="flex-1 bg-transparent outline-none text-sm"
            style={{ color: 'var(--color-text-primary)' }}
            role="combobox"
            aria-expanded={filtered.length > 0}
            aria-controls="command-palette-listbox"
            aria-activedescendant={filtered[selectedIndex] ? `cmd-${filtered[selectedIndex].id}` : undefined}
            aria-label="搜索页面或输入股票代码"
          />
          <kbd
            className="hidden sm:inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium"
            style={{
              background: 'var(--color-bg-hover)',
              color: 'var(--color-text-muted)',
              border: '1px solid var(--color-border)',
            }}
          >
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div ref={listRef} id="command-palette-listbox" role="listbox" aria-label="搜索结果" className="max-h-[360px] overflow-y-auto py-2">
          {filtered.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm" style={{ color: 'var(--color-text-muted)' }}>
              未找到匹配结果
            </div>
          ) : (
            filtered.map((cmd, index) => {
              const Icon = cmd.icon;
              const isActive = index === selectedIndex;
              return (
                <button
                  key={cmd.id}
                  id={`cmd-${cmd.id}`}
                  role="option"
                  aria-selected={isActive}
                  className={`flex items-center gap-3 w-full px-4 py-2.5 text-left transition-colors text-sm ${
                    isActive ? '' : ''
                  }`}
                  style={{
                    background: isActive ? 'var(--color-bg-hover)' : 'transparent',
                    color: isActive ? 'var(--color-text-primary)' : 'var(--color-text-secondary)',
                  }}
                  onClick={() => executeCommand(cmd)}
                  onMouseEnter={() => setSelectedIndex(index)}
                >
                  <Icon className="w-4 h-4 shrink-0" style={{ opacity: 0.7 }} />
                  <span className="flex-1">{cmd.label}</span>
                  {cmd.type === 'page' && (
                    <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {cmd.view}
                    </span>
                  )}
                  {isActive && (
                    <ArrowRight className="w-3.5 h-3.5 shrink-0" style={{ color: 'var(--color-accent)' }} />
                  )}
                </button>
              );
            })
          )}
        </div>

        {/* Footer */}
        <div
          className="flex items-center gap-4 px-4 py-2.5 border-t text-[11px]"
          style={{ borderColor: 'var(--color-border)', color: 'var(--color-text-muted)' }}
        >
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded" style={{ background: 'var(--color-bg-hover)', border: '1px solid var(--color-border)' }}>↑↓</kbd>
            导航
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded" style={{ background: 'var(--color-bg-hover)', border: '1px solid var(--color-border)' }}>↵</kbd>
            选择
          </span>
          <span className="flex items-center gap-1">
            <kbd className="px-1 py-0.5 rounded" style={{ background: 'var(--color-bg-hover)', border: '1px solid var(--color-border)' }}>ESC</kbd>
            关闭
          </span>
        </div>
      </div>
    </div>
  );
}
