export default function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <div className="flex flex-col items-center gap-4">
        <div
          className="w-10 h-10 border-4 border-t-transparent rounded-full animate-spin"
          style={{
            borderColor: 'var(--color-border)',
            borderTopColor: 'transparent',
            animationDuration: '0.8s',
          }}
        />
        <span
          className="text-sm"
          style={{ color: 'var(--color-text-muted)' }}
        >
          加载中...
        </span>
      </div>
    </div>
  );
}
