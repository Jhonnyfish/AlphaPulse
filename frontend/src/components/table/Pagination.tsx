import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react';

interface PaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  totalItems?: number;
  pageSize?: number;
}

export default function Pagination({
  page,
  totalPages,
  onPageChange,
  totalItems,
  pageSize,
}: PaginationProps) {
  if (totalPages <= 1) return null;

  /** Generate page numbers to display */
  const getPageNumbers = (): (number | 'ellipsis')[] => {
    const pages: (number | 'ellipsis')[] = [];
    const maxVisible = 7;

    if (totalPages <= maxVisible) {
      for (let i = 1; i <= totalPages; i++) pages.push(i);
      return pages;
    }

    // Always show first page
    pages.push(1);

    if (page > 3) pages.push('ellipsis');

    // Pages around current
    const start = Math.max(2, page - 1);
    const end = Math.min(totalPages - 1, page + 1);
    for (let i = start; i <= end; i++) {
      if (!pages.includes(i)) pages.push(i);
    }

    if (page < totalPages - 2 && !pages.includes('ellipsis')) {
      pages.push('ellipsis');
    }

    // Always show last page
    if (!pages.includes(totalPages)) pages.push(totalPages);

    return pages;
  };

  const btnStyle = (active: boolean, disabled: boolean) =>
    ({
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      minWidth: '32px',
      height: '32px',
      padding: '0 8px',
      borderRadius: '8px',
      fontSize: '13px',
      fontWeight: active ? 600 : 400,
      fontFamily: "'Inter', monospace",
      cursor: disabled ? 'not-allowed' : 'pointer',
      opacity: disabled ? 0.3 : 1,
      transition: 'all 0.15s',
      background: active
        ? 'var(--color-accent)'
        : 'rgba(148, 163, 184, 0.06)',
      color: active ? '#fff' : 'var(--color-text-secondary)',
      border: active
        ? '1px solid var(--color-accent)'
        : '1px solid rgba(148, 163, 184, 0.1)',
    }) as React.CSSProperties;

  return (
    <div className="flex items-center justify-between mt-3 gap-4">
      {/* Left: info */}
      <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
        {totalItems != null && pageSize != null && (
          <span>
            第 {(page - 1) * pageSize + 1}–
            {Math.min(page * pageSize, totalItems)} 条，共 {totalItems} 条
          </span>
        )}
      </div>

      {/* Right: page buttons */}
      <div className="flex items-center gap-1">
        {/* First */}
        <button
          style={btnStyle(false, page === 1)}
          disabled={page === 1}
          onClick={() => onPageChange(1)}
          title="第一页"
        >
          <ChevronsLeft className="w-4 h-4" />
        </button>
        {/* Prev */}
        <button
          style={btnStyle(false, page === 1)}
          disabled={page === 1}
          onClick={() => onPageChange(page - 1)}
          title="上一页"
        >
          <ChevronLeft className="w-4 h-4" />
        </button>

        {/* Page numbers */}
        {getPageNumbers().map((p, i) =>
          p === 'ellipsis' ? (
            <span
              key={`dots-${i}`}
              className="px-1 text-xs"
              style={{ color: 'var(--color-text-muted)' }}
            >
              ···
            </span>
          ) : (
            <button
              key={p}
              style={btnStyle(p === page, false)}
              onClick={() => onPageChange(p)}
            >
              {p}
            </button>
          )
        )}

        {/* Next */}
        <button
          style={btnStyle(false, page === totalPages)}
          disabled={page === totalPages}
          onClick={() => onPageChange(page + 1)}
          title="下一页"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
        {/* Last */}
        <button
          style={btnStyle(false, page === totalPages)}
          disabled={page === totalPages}
          onClick={() => onPageChange(totalPages)}
          title="最后一页"
        >
          <ChevronsRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}
