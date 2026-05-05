import { clsx } from 'clsx'

interface SkeletonProps {
  className?: string
  variant?: 'text' | 'circular' | 'rectangular'
  width?: string | number
  height?: string | number
}

export function Skeleton({ className, variant = 'rectangular', width, height }: SkeletonProps) {
  const style = { width, height }

  return (
    <div
      className={clsx(
        'skeleton-shimmer bg-[rgba(148,163,184,0.1)]',
        variant === 'text' && 'h-4 rounded-sm',
        variant === 'circular' && 'rounded-full',
        variant === 'rectangular' && 'rounded-md',
        className,
      )}
      style={style}
    />
  )
}

export function SkeletonCard({ className }: { className?: string }) {
  return (
    <div className={clsx('glass-panel p-5 space-y-3', className)} style={{ height: 200 }}>
      <Skeleton variant="text" width="60%" height={20} />
      <Skeleton variant="text" width="100%" />
      <Skeleton variant="text" width="85%" />
      <div className="flex gap-3 pt-2">
        <Skeleton variant="rectangular" width={80} height={32} className="rounded-lg" />
        <Skeleton variant="rectangular" width={80} height={32} className="rounded-lg" />
      </div>
    </div>
  )
}

/** Compact card skeleton for grid layouts (height ~120px) */
export function SkeletonGridCard({ className }: { className?: string }) {
  return (
    <div className={clsx('glass-panel p-4 space-y-2', className)} style={{ height: 120 }}>
      <Skeleton variant="text" width="50%" height={16} />
      <Skeleton variant="text" width="80%" height={24} />
      <Skeleton variant="text" width="65%" height={12} />
    </div>
  )
}

interface SkeletonTableProps {
  rows?: number
  columns?: number
  className?: string
}

export function SkeletonTable({ rows = 5, columns = 4, className }: SkeletonTableProps) {
  return (
    <div className={clsx('space-y-3', className)}>
      <div className="flex gap-4 pb-2 border-b border-[var(--color-border)]">
        {Array.from({ length: columns }).map((_, i) => (
          <Skeleton key={i} variant="text" width={`${100 / columns}%`} height={14} />
        ))}
      </div>
      {Array.from({ length: rows }).map((_, row) => (
        <div key={row} className="flex gap-4">
          {Array.from({ length: columns }).map((_, col) => (
            <Skeleton key={col} variant="text" width={`${100 / columns}%`} height={12} />
          ))}
        </div>
      ))}
    </div>
  )
}

/** List row skeleton — for list/card-based loading (news, cards, etc.) */
interface SkeletonListProps {
  rows?: number
  className?: string
}

export function SkeletonList({ rows = 5, className }: SkeletonListProps) {
  return (
    <div className={clsx('space-y-3', className)}>
      {Array.from({ length: rows }).map((_, i) => (
        <div
          key={i}
          className="rounded-lg border p-4 space-y-2"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <Skeleton variant="text" width={`${55 + (i * 7) % 30}%`} height={16} />
          <Skeleton variant="text" width="100%" height={12} />
          <div className="flex items-center gap-3">
            <Skeleton variant="text" width={60} height={10} />
            <Skeleton variant="text" width={40} height={10} />
          </div>
        </div>
      ))}
    </div>
  )
}

/** Inline table skeleton that renders inside a table container (thead + tbody) */
interface SkeletonInlineTableProps {
  rows?: number
  columns?: number
  className?: string
}

export function SkeletonInlineTable({ rows = 5, columns = 6, className }: SkeletonInlineTableProps) {
  return (
    <div className={clsx('overflow-x-auto', className)}>
      <table className="w-full text-sm">
        <thead>
          <tr style={{ borderBottom: '1px solid var(--color-border)' }}>
            {Array.from({ length: columns }).map((_, i) => (
              <th key={i} className="px-4 py-2.5 text-left">
                <Skeleton variant="text" width={`${40 + (i * 11) % 40}%`} height={12} />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: rows }).map((_, row) => (
            <tr key={row} style={{ borderBottom: '1px solid var(--color-border)' }}>
              {Array.from({ length: columns }).map((_, col) => (
                <td key={col} className="px-4 py-2.5">
                  <Skeleton variant="text" width={`${30 + Math.random() * 50}%`} height={10} />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/** Calendar grid skeleton */
export function SkeletonCalendar({ className }: { className?: string }) {
  return (
    <div className={clsx('space-y-2', className)}>
      <div className="grid grid-cols-7 gap-1 mb-1">
        {Array.from({ length: 7 }).map((_, i) => (
          <div key={i} className="text-center">
            <Skeleton variant="text" width="60%" height={12} className="mx-auto" />
          </div>
        ))}
      </div>
      {Array.from({ length: 5 }).map((_, week) => (
        <div key={week} className="grid grid-cols-7 gap-1">
          {Array.from({ length: 7 }).map((_, day) => (
            <Skeleton key={day} variant="rectangular" height={48} className="rounded" />
          ))}
        </div>
      ))}
    </div>
  )
}

/** Summary stats row skeleton (4 cards in a row) */
export function SkeletonStatCards({ count = 4, className }: { count?: number; className?: string }) {
  return (
    <div className={clsx('grid grid-cols-2 md:grid-cols-4 gap-4', className)}>
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="glass-panel rounded-xl p-4 space-y-2" style={{ height: 100 }}>
          <Skeleton variant="text" width="50%" height={12} />
          <Skeleton variant="text" width="70%" height={24} />
        </div>
      ))}
    </div>
  )
}
