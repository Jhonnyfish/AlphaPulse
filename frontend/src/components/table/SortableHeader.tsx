import { ChevronUp, ChevronDown, ChevronsUpDown } from 'lucide-react';
import type { SortDirection } from '@/hooks/useTableSort';

interface SortableHeaderProps {
  label: string;
  sortKey: string;
  currentKey: string;
  direction: SortDirection;
  onSort: (key: string) => void;
  align?: 'left' | 'right' | 'center';
  className?: string;
}

export default function SortableHeader({
  label,
  sortKey,
  currentKey,
  direction,
  onSort,
  align = 'left',
  className = '',
}: SortableHeaderProps) {
  const isActive = currentKey === sortKey && direction != null;

  const Icon =
    currentKey === sortKey && direction === 'asc'
      ? ChevronUp
      : currentKey === sortKey && direction === 'desc'
        ? ChevronDown
        : ChevronsUpDown;

  return (
    <th
      className={`select-none cursor-pointer transition-colors whitespace-nowrap ${className}`}
      style={{
        textAlign: align,
        color: isActive ? 'var(--color-accent)' : 'var(--color-text-muted)',
        padding: '10px 14px',
        fontSize: '12px',
        fontWeight: 600,
        letterSpacing: '0.05em',
        borderBottom: '1px solid var(--color-border)',
      }}
      onClick={() => onSort(sortKey)}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        <Icon
          className="w-3.5 h-3.5 flex-shrink-0"
          style={{
            opacity: isActive ? 1 : 0.3,
            transition: 'opacity 0.15s',
          }}
        />
      </span>
    </th>
  );
}
