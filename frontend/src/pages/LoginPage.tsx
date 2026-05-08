import { useState, useEffect, type FormEvent } from 'react';
import { useAuth } from '@/lib/auth';
import { Activity, Eye, EyeOff, Lock, User } from 'lucide-react';

export default function LoginPage() {
  const { login } = useAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [rememberMe, setRememberMe] = useState(false);
  const [showForgotHint, setShowForgotHint] = useState(false);
  const [focusField, setFocusField] = useState<'user' | 'pass' | null>(null);

  useEffect(() => {
    const saved = localStorage.getItem('alphapulse_remembered_user');
    if (saved) {
      setUsername(saved);
      setRememberMe(true);
    }
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(username, password);
      if (rememberMe) {
        localStorage.setItem('alphapulse_remembered_user', username);
      } else {
        localStorage.removeItem('alphapulse_remembered_user');
      }
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        '登录失败，请检查用户名和密码';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  const iconColor = (field: 'user' | 'pass') =>
    focusField === field ? 'var(--color-accent)' : 'var(--color-text-muted)';

  return (
    <div className="min-h-screen flex items-center justify-center p-4"
      style={{ background: 'var(--color-bg-primary)' }}>

      {/* Background decoration */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 rounded-full opacity-25"
          style={{ background: 'radial-gradient(circle, rgba(59, 130, 246, 0.5), transparent 70%)' }} />
        <div className="absolute bottom-1/3 right-1/4 w-80 h-80 rounded-full opacity-20"
          style={{ background: 'radial-gradient(circle, rgba(34, 211, 238, 0.4), transparent 70%)' }} />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] rounded-full opacity-10"
          style={{ background: 'radial-gradient(circle, rgba(99, 102, 241, 0.3), transparent 60%)' }} />
      </div>

      {/* Login card */}
      <div className="relative w-full max-w-md">
        {/* Glow border */}
        <div className="absolute -inset-px rounded-2xl opacity-40 blur-sm"
          style={{ background: 'linear-gradient(135deg, var(--color-accent), var(--color-cyan))' }} />

        <div className="relative p-10 rounded-2xl glass-panel animate-scale-in"
          style={{
            background: 'rgba(15, 23, 42, 0.88)',
            backdropFilter: 'blur(24px)',
            border: '1px solid rgba(148, 163, 184, 0.12)',
          }}>

          {/* Logo */}
          <div className="text-center mb-10">
            <div className="flex items-center justify-center gap-3 mb-3">
              <div className="relative">
                <div className="absolute inset-0 rounded-full opacity-50 pulse-ring"
                  style={{ background: 'var(--color-accent)' }} />
                <Activity className="relative w-10 h-10" style={{ color: 'var(--color-accent)' }} />
              </div>
              <h1 className="text-3xl font-bold text-gradient">AlphaPulse</h1>
            </div>
            <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
              智能量化分析系统
            </p>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-5">

            {/* Username */}
            <div>
              <label className="block text-xs font-medium mb-2 tracking-wide uppercase"
                style={{ color: 'var(--color-text-muted)' }}>
                用户名
              </label>
              <div className="relative">
                <User className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 transition-colors duration-200"
                  style={{ color: iconColor('user') }} />
                <input
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  onFocus={() => setFocusField('user')}
                  onBlur={() => setFocusField(null)}
                  placeholder="输入用户名"
                  autoFocus
                  className="login-input w-full pl-11 pr-4 py-3.5 rounded-xl text-sm outline-none transition-all duration-200"
                  style={{
                    background: 'rgba(15, 23, 42, 0.5)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                />
              </div>
            </div>

            {/* Password */}
            <div>
              <label className="block text-xs font-medium mb-2 tracking-wide uppercase"
                style={{ color: 'var(--color-text-muted)' }}>
                密码
              </label>
              <div className="relative">
                <Lock className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 transition-colors duration-200"
                  style={{ color: iconColor('pass') }} />
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  onFocus={() => setFocusField('pass')}
                  onBlur={() => setFocusField(null)}
                  placeholder="输入密码"
                  className="login-input w-full pl-11 pr-12 py-3.5 rounded-xl text-sm outline-none transition-all duration-200"
                  style={{
                    background: 'rgba(15, 23, 42, 0.5)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 p-1.5 rounded-lg hover:bg-white/10 transition-colors"
                  style={{ color: 'var(--color-text-muted)' }}>
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
            </div>

            {/* Remember me & Forgot password */}
            <div className="flex items-center justify-between">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={rememberMe}
                  onChange={(e) => setRememberMe(e.target.checked)}
                  className="w-4 h-4 rounded border-2 accent-[var(--color-accent)] cursor-pointer"
                  style={{
                    background: 'var(--color-bg-card)',
                    borderColor: 'var(--color-border)',
                  }}
                />
                <span className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
                  记住我
                </span>
              </label>

              <button
                type="button"
                onClick={() => setShowForgotHint(!showForgotHint)}
                className="text-sm hover:underline transition-colors"
                style={{ color: 'var(--color-accent)' }}>
                忘记密码？
              </button>
            </div>

            {/* Forgot password hint */}
            {showForgotHint && (
              <div className="text-sm px-4 py-3 rounded-xl animate-fade-in"
                style={{
                  background: 'rgba(59, 130, 246, 0.1)',
                  border: '1px solid rgba(59, 130, 246, 0.2)',
                  color: 'var(--color-text-secondary)',
                }}>
                请联系系统管理员重置密码
              </div>
            )}

            {/* Error message */}
            {error && (
              <div className="text-sm px-4 py-3 rounded-xl flex items-center gap-2 animate-fade-in"
                style={{
                  background: 'rgba(239, 68, 68, 0.1)',
                  border: '1px solid rgba(239, 68, 68, 0.2)',
                  color: 'var(--color-danger)',
                }}>
                <span>⚠</span>
                {error}
              </div>
            )}

            {/* Submit button */}
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3.5 rounded-xl text-sm font-medium text-white transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed hover:shadow-lg active:scale-[0.98] cursor-pointer"
              style={{
                background: loading
                  ? 'var(--color-accent)'
                  : 'linear-gradient(135deg, var(--color-accent), #2563eb)',
                boxShadow: loading ? 'none' : '0 4px 20px rgba(59, 130, 246, 0.35)',
              }}>
              {loading ? (
                <span className="flex items-center justify-center gap-2">
                  <svg className="w-4 h-4 animate-spin" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  登录中...
                </span>
              ) : '登 录'}
            </button>
          </form>

          {/* Footer */}
          <div className="mt-10 text-center">
            <p className="text-xs" style={{ color: 'var(--color-text-muted)', opacity: 0.6 }}>
              © 2026 AlphaPulse. All rights reserved.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
