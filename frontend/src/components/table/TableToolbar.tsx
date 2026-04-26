import { Search, X } from 'lucide-react';

export interface FilterOption {
  key: string;
  label: string;
}

interface TableToolbarProps {
  searchQuery: string;
  onSearchChange: (q: string) => void;
  searchPlaceholder?: string;
  filterValue?: string;
  onFilterChange?: (f: string) => void;
  filterOptions?: FilterOption[];
  totalCount?: number;
  children?: React.ReactNode;
}

export default function TableToolbar({
  searchQuery,
  onSearchChange,
  searchPlaceholder = '搜索...',
  filterValue,
  onFilterChange,
  filterOptions,
  totalCount,
  children,
}: TableToolbarProps) {
  return (
    <div
      className="flex flex-wrap items-center gap-3 mb-3"
    >
      {/* Search input */}
      <div className="relative flex-shrink-0" style={{ minWidth: '220px' }}>
        <Search
          className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4"
          style={{ color: 'var(--color-text-muted)' }}
        />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder={searchPlaceholder}
          className="w-full pl-9 pr-8 py-2 rounded-lg border text-sm outline-none transition-colors"
          style={{
            background: '#1a1d27',
            borderColor: searchQuery
              ? 'var(--color-accent)'
              : 'var(--color-border)',
            color: 'var(--color-text-primary)',
          }}
        />
        {searchQuery && (
          <button
            onClick={() => onSearchChange('')}
            className="absolute right-3 top-1/2 -translate-y-1/2"
          >
            <X
              className="w-3.5 h-3.5"
              style={{ color: 'var(--color-text-muted)' }}
            />
          </button>
        )}
      </div>

      {/* Filter buttons */}
      {filterOptions && onFilterChange && (
        <div className="flex items-center gap-1.5 flex-wrap">
          {filterOptions.map((opt) => {
            const isActive = (filterValue ?? 'all') === opt.key;
            return (
              <button
                key={opt.key}
                onClick={() => onFilterChange(opt.key)}
                className="px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
                style={{
                  background: isActive
                    ? 'rgba(59, 130, 246, 0.15)'
                    : 'rgba(148, 163, 184, 0.06)',
                  color: isActive
                    ? 'var(--color-accent)'
                    : 'var(--color-text-muted)',
                  border: `1px solid ${
                    isActive
                      ? 'rgba(59, 130, 246, 0.3)'
                      : 'rgba(148, 163, 184, 0.1)'
                  }`,
                }}
              >
                {opt.label}
              </button>
            );
          })}
        </div>
      )}

      {/* Spacer */}
      <div className="flex-1" />

      {/* Total count */}
      {totalCount != null && (
        <span
          className="text-xs flex-shrink-0"
          style={{ color: 'var(--color-text-muted)' }}
        >
          共 {totalCount} 条
        </span>
      )}

      {/* Extra actions slot */}
      {children}
    </div>
  );
}
