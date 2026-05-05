import { AlertTriangle, RefreshCw } from 'lucide-react';

interface ErrorStateProps {
  title?: string;
  description?: string;
  onRetry?: () => void;
  retryText?: string;
}

export default function ErrorState({ title = '加载失败', description, onRetry, retryText = '重试' }: ErrorStateProps) {
  return (
    <div
      className="flex flex-col items-center justify-center py-16 rounded-xl animate-fade-in"
      role="alert"
      style={{
        background: 'rgba(15, 23, 42, 0.4)',
        backdropFilter: 'blur(18px)',
        border: '1px solid rgba(239, 68, 68, 0.15)',
      }}
    >
      <AlertTriangle className="w-12 h-12 mb-4" style={{ color: '#ef4444', opacity: 0.6 }} />
      <p className="text-sm font-medium mb-1" style={{ color: '#f87171' }}>
        {title}
      </p>
      {description && (
        <p className="text-xs mb-4 max-w-xs text-center" style={{ color: 'var(--color-text-muted)' }}>
          {description}
        </p>
      )}
      {onRetry && (
        <button
          onClick={onRetry}
          className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-medium transition-all hover:opacity-90"
          style={{ background: 'var(--color-accent)', color: '#fff' }}
        >
          <RefreshCw className="w-3.5 h-3.5" />
          {retryText}
        </button>
      )}
    </div>
  );
}
