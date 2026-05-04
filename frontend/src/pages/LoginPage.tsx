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

  // 加载保存的用户名
  useEffect(() => {
    const savedUsername = localStorage.getItem('alphapulse_remembered_user');
    if (savedUsername) {
      setUsername(savedUsername);
      setRememberMe(true);
    }
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    
    try {
      await login(username, password);
      
      // 保存/清除用户名
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

  return (
    <div className="min-h-screen flex items-center justify-center p-4"
      style={{ 
        background: 'var(--color-bg-primary)',
      }}>
      
      {/* 背景装饰 */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        {/* 渐变光晕 */}
        <div className="absolute top-1/4 left-1/4 w-96 h-96 rounded-full opacity-20"
          style={{ background: 'radial-gradient(circle, rgba(59, 130, 246, 0.4), transparent 70%)' }} />
        <div className="absolute bottom-1/4 right-1/4 w-80 h-80 rounded-full opacity-15"
          style={{ background: 'radial-gradient(circle, rgba(34, 211, 238, 0.4), transparent 70%)' }} />
      </div>

      {/* 登录卡片 */}
      <div className="relative w-full max-w-md">
        {/* 发光边框效果 */}
        <div className="absolute -inset-1 rounded-2xl opacity-30 blur-sm"
          style={{ background: 'linear-gradient(135deg, var(--color-accent), var(--color-cyan))' }} />
        
        <div className="relative p-8 rounded-2xl glass-panel animate-scale-in"
          style={{ 
            background: 'rgba(15, 23, 42, 0.85)',
            backdropFilter: 'blur(20px)',
          }}>
          
          {/* Logo 和品牌 */}
          <div className="text-center mb-8">
            <div className="flex items-center justify-center gap-3 mb-4">
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
            <span className="inline-block mt-2 px-3 py-1 rounded-full text-xs"
              style={{ 
                background: 'rgba(59, 130, 246, 0.15)',
                color: 'var(--color-accent)',
                border: '1px solid rgba(59, 130, 246, 0.3)',
              }}>
              v2.0
            </span>
          </div>

          {/* 登录表单 */}
          <form onSubmit={handleSubmit} className="space-y-5">
            {/* 用户名输入框 */}
            <div>
              <label className="block text-sm font-medium mb-2"
                style={{ color: 'var(--color-text-secondary)' }}>
                用户名
              </label>
              <div className="relative">
                <User className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4"
                  style={{ color: 'var(--color-text-muted)' }} />
                <input
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full pl-10 pr-4 py-3 rounded-xl text-sm outline-none transition-all duration-200"
                  style={{
                    background: 'rgba(15, 23, 42, 0.6)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                  onFocus={(e) => {
                    e.target.style.borderColor = 'var(--color-accent)';
                    e.target.style.boxShadow = '0 0 0 3px rgba(59, 130, 246, 0.2)';
                  }}
                  onBlur={(e) => {
                    e.target.style.borderColor = 'var(--color-border)';
                    e.target.style.boxShadow = 'none';
                  }}
                  placeholder="输入用户名"
                  autoFocus
                />
              </div>
            </div>

            {/* 密码输入框 */}
            <div>
              <label className="block text-sm font-medium mb-2"
                style={{ color: 'var(--color-text-secondary)' }}>
                密码
              </label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4"
                  style={{ color: 'var(--color-text-muted)' }} />
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full pl-10 pr-12 py-3 rounded-xl text-sm outline-none transition-all duration-200"
                  style={{
                    background: 'rgba(15, 23, 42, 0.6)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                  onFocus={(e) => {
                    e.target.style.borderColor = 'var(--color-accent)';
                    e.target.style.boxShadow = '0 0 0 3px rgba(59, 130, 246, 0.2)';
                  }}
                  onBlur={(e) => {
                    e.target.style.borderColor = 'var(--color-border)';
                    e.target.style.boxShadow = 'none';
                  }}
                  placeholder="输入密码"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 p-1 rounded-md hover:bg-white/10 transition-colors"
                  style={{ color: 'var(--color-text-muted)' }}>
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
            </div>

            {/* 记住我 & 忘记密码 */}
            <div className="flex items-center justify-between">
              <label className="flex items-center gap-2 cursor-pointer group">
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

            {/* 忘记密码提示 */}
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

            {/* 错误提示 */}
            {error && (
              <div className="text-sm px-4 py-3 rounded-xl flex items-center gap-2 animate-shake"
                style={{ 
                  background: 'rgba(239, 68, 68, 0.1)',
                  border: '1px solid rgba(239, 68, 68, 0.2)',
                  color: 'var(--color-danger)',
                }}>
                <span>⚠</span>
                {error}
              </div>
            )}

            {/* 登录按钮 */}
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 rounded-xl text-sm font-medium text-white transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed hover:shadow-lg active:scale-[0.98]"
              style={{ 
                background: loading 
                  ? 'var(--color-accent)' 
                  : 'linear-gradient(135deg, var(--color-accent), #2563eb)',
                boxShadow: loading ? 'none' : '0 4px 15px rgba(59, 130, 246, 0.4)',
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

          {/* 底部信息 */}
          <div className="mt-8 text-center">
            <p className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
              © 2026 AlphaPulse. All rights reserved.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}