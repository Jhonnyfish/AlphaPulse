import { useState } from 'react';
import { AlertTriangle, X } from 'lucide-react';

interface DegradedBannerProps {
  visible: boolean;
  message?: string;
  onDismiss?: () => void;
}

export default function DegradedBanner({ visible, message, onDismiss }: DegradedBannerProps) {
  const [dismissed, setDismissed] = useState(false);

  if (!visible || dismissed) return null;

  const handleDismiss = () => {
    setDismissed(true);
    onDismiss?.();
  };

  return (
    <div
      className="flex items-center gap-3 px-4 py-3 mb-4 rounded-lg border transition-opacity duration-300"
      style={{
        background: 'rgba(245, 158, 11, 0.08)',
        borderColor: 'rgba(245, 158, 11, 0.3)',
      }}
    >
      <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0" />
      <div className="flex-1 min-w-0">
        <span className="text-sm font-medium text-amber-400">数据暂时不可用</span>
        {message && (
          <span className="text-sm text-gray-400 ml-2">— {message}</span>
        )}
      </div>
      <button
        onClick={handleDismiss}
        className="shrink-0 p-1 rounded-md transition-colors hover:bg-white/10"
        aria-label="关闭"
      >
        <X className="w-4 h-4 text-gray-400" />
      </button>
    </div>
  );
}
