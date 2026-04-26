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
