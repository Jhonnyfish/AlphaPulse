// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { type LucideIcon } from 'lucide-react';

interface EmptyStateProps {
  icon: React.ElementType;
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
}

export default function EmptyState({ icon: Icon, title, description, actionLabel, onAction }: EmptyStateProps) {
  return (
    <div
      className="flex flex-col items-center justify-center py-16 rounded-xl animate-fade-in"
      role="status"
      style={{
        background: 'rgba(15, 23, 42, 0.4)',
        backdropFilter: 'blur(18px)',
        border: '1px solid rgba(148, 163, 184, 0.08)',
      }}
    >
      <Icon className="w-12 h-12 mb-4 opacity-30" style={{ color: 'var(--color-text-muted)' }} />
      <p className="text-sm font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>
        {title}
      </p>
      <p className="text-xs mb-4" style={{ color: 'var(--color-text-muted)' }}>
        {description}
      </p>
      {actionLabel && onAction && (
        <button
          onClick={onAction}
          className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-medium transition-colors hover:opacity-90"
          style={{ background: 'var(--color-accent)', color: '#fff' }}
        >
          {actionLabel}
        </button>
      )}
    </div>
  );
}
