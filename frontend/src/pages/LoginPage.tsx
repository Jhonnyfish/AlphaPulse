import { useState, type FormEvent } from 'react';
import { useAuth } from '@/lib/auth';
import { Activity } from 'lucide-react';

export default function LoginPage() {
  const { login } = useAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(username, password);
      // Auth state change will trigger re-render and redirect
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        '登录失败，请检查用户名和密码';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center"
      style={{ background: 'var(--color-bg-primary)' }}>
      <div className="w-full max-w-sm p-8 rounded-xl border"
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}>
        {/* Logo */}
        <div className="flex items-center justify-center gap-2 mb-8">
          <Activity className="w-8 h-8" style={{ color: 'var(--color-accent)' }} />
          <span className="text-2xl font-bold">AlphaPulse</span>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm mb-1.5"
              style={{ color: 'var(--color-text-secondary)' }}>用户名</label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none focus:ring-2"
              style={{
                background: 'var(--color-bg-card)',
                borderColor: 'var(--color-border)',
                color: 'var(--color-text-primary)',
                // @ts-expect-error CSS custom property
                '--tw-ring-color': 'var(--color-accent)',
              }}
              placeholder="输入用户名"
              autoFocus
            />
          </div>
          <div>
            <label className="block text-sm mb-1.5"
              style={{ color: 'var(--color-text-secondary)' }}>密码</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none focus:ring-2"
              style={{
                background: 'var(--color-bg-card)',
                borderColor: 'var(--color-border)',
                color: 'var(--color-text-primary)',
                // @ts-expect-error CSS custom property
                '--tw-ring-color': 'var(--color-accent)',
              }}
              placeholder="输入密码"
            />
          </div>

          {error && (
            <div className="text-sm px-3 py-2 rounded-lg"
              style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2.5 rounded-lg text-sm font-medium text-white transition-colors disabled:opacity-50"
            style={{ background: 'var(--color-accent)' }}
          >
            {loading ? '登录中...' : '登录'}
          </button>
        </form>
      </div>
    </div>
  );
}
