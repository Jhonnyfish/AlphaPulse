import { Keyboard, X } from 'lucide-react';

interface Shortcut {
  keys: string;
  description: string;
}

interface KeyboardHelpPanelProps {
  open: boolean;
  onClose: () => void;
  shortcuts?: Shortcut[];
}

const defaultShortcuts: Shortcut[] = [
  { keys: '⌘ K', description: '搜索 / 导航' },
  { keys: 'R', description: '刷新页面' },
  { keys: 'D', description: '回到总览' },
  { keys: 'W', description: '自选股' },
  { keys: '?', description: '显示快捷键帮助' },
  { keys: 'ESC', description: '关闭弹窗 / 侧边栏' },
];

export default function KeyboardHelpPanel({ open, onClose, shortcuts }: KeyboardHelpPanelProps) {
  if (!open) return null;

  const items = shortcuts || defaultShortcuts;

  return (
    <div className="modal-backdrop" onClick={onClose} role="dialog" aria-modal="true" aria-label="快捷键帮助">
      <div
        className="modal-content p-6 w-full max-w-md animate-scale-in"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-2">
            <Keyboard className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
            <h2 className="text-lg font-bold">快捷键</h2>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
            aria-label="关闭"
          >
            <X className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
          </button>
        </div>

        <div className="space-y-2">
          {items.map((s) => (
            <div
              key={s.keys}
              className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
            >
              <span className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
                {s.description}
              </span>
              <kbd
                className="inline-flex items-center px-2 py-1 rounded text-xs font-mono font-medium"
                style={{
                  background: 'var(--color-bg-hover)',
                  border: '1px solid var(--color-border)',
                  color: 'var(--color-text-primary)',
                }}
              >
                {s.keys}
              </kbd>
            </div>
          ))}
        </div>

        <div
          className="mt-4 pt-3 text-xs text-center"
          style={{ color: 'var(--color-text-muted)', borderTop: '1px solid var(--color-border)' }}
        >
          按 <kbd className="px-1 py-0.5 rounded text-[10px]"
            style={{ background: 'var(--color-bg-hover)', border: '1px solid var(--color-border)' }}
          >ESC</kbd> 关闭
        </div>
      </div>
    </div>
  );
}
