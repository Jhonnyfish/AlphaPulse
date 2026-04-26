import { useState, useEffect, useCallback } from 'react';
import { strategiesApi, type Strategy } from '@/lib/api';
import { Zap, Plus, Trash2, Power, PowerOff, RefreshCw } from 'lucide-react';
import { SkeletonList } from '@/components/ui/Skeleton';

interface AddForm {
  name: string;
  description: string;
  rules: string;
}

const emptyForm: AddForm = { name: '', description: '', rules: '{}' };

export default function StrategiesPage() {
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState<AddForm>(emptyForm);
  const [submitting, setSubmitting] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await strategiesApi.list();
      setStrategies(res.data ?? []);
    } catch {
      setError('加载策略列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleAdd = async () => {
    if (!form.name.trim()) return;
    setSubmitting(true);
    try {
      let rules: Record<string, unknown> = {};
      try {
        rules = JSON.parse(form.rules);
      } catch {
        setError('规则 JSON 格式不正确');
        setSubmitting(false);
        return;
      }
      await strategiesApi.create({ name: form.name.trim(), description: form.description.trim(), rules });
      setShowModal(false);
      setForm(emptyForm);
      await fetchData();
    } catch {
      setError('添加策略失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggle = async (strategy: Strategy) => {
    setTogglingId(strategy.id);
    try {
      if (strategy.active) {
        await strategiesApi.deactivate(strategy.id);
      } else {
        await strategiesApi.activate(strategy.id);
      }
      await fetchData();
    } catch {
      setError(strategy.active ? '停用策略失败' : '激活策略失败');
    } finally {
      setTogglingId(null);
    }
  };

  const handleDelete = async (id: string) => {
    setDeletingId(id);
    try {
      await strategiesApi.remove(id);
      setConfirmDeleteId(null);
      await fetchData();
    } catch {
      setError('删除策略失败');
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Zap className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">策略管理</h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowModal(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors"
            style={{ background: 'var(--color-accent)', color: '#fff' }}
          >
            <Plus className="w-3.5 h-3.5" />
            添加策略
          </button>
          <button
            onClick={fetchData}
            disabled={loading}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm hover:bg-[var(--color-bg-hover)] transition-colors"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </button>
        </div>
      </div>

      {error && (
        <div
          className="text-sm px-3 py-2 rounded-lg mb-4 max-w-md"
          style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}
        >
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && strategies.length === 0 ? (
        <SkeletonList rows={5} />
      ) : strategies.length === 0 ? (
        <div
          className="text-center py-16 rounded-lg border"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <Zap className="w-10 h-10 mx-auto mb-3 opacity-40" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>暂无策略，点击「添加策略」开始</p>
        </div>
      ) : (
        <div
          className="rounded-xl border overflow-hidden"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <div className="px-4 py-3 flex items-center justify-between border-b" style={{ borderColor: 'var(--color-border)' }}>
            <span className="text-sm font-medium">策略列表</span>
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
              {strategies.length} 条策略
            </span>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs" style={{ color: 'var(--color-text-muted)', borderBottom: '1px solid var(--color-border)' }}>
                  <th className="text-left px-4 py-2.5 font-medium">名称</th>
                  <th className="text-left px-4 py-2.5 font-medium">描述</th>
                  <th className="text-center px-4 py-2.5 font-medium">状态</th>
                  <th className="text-left px-4 py-2.5 font-medium">创建时间</th>
                  <th className="text-center px-4 py-2.5 font-medium">操作</th>
                </tr>
              </thead>
              <tbody>
                {strategies.map((s) => (
                  <tr
                    key={s.id}
                    className="transition-colors hover:bg-[var(--color-bg-hover)]"
                    style={{ borderBottom: '1px solid var(--color-border)' }}
                  >
                    <td className="px-4 py-2.5 font-medium" style={{ color: 'var(--color-text-primary)' }}>
                      {s.name}
                    </td>
                    <td className="px-4 py-2.5 text-xs max-w-[200px] truncate" style={{ color: 'var(--color-text-secondary)' }} title={s.description}>
                      {s.description || '-'}
                    </td>
                    <td className="px-4 py-2.5 text-center">
                      <span
                        className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full font-medium"
                        style={{
                          background: s.active ? 'rgba(34,197,94,0.15)' : 'rgba(107,114,128,0.15)',
                          color: s.active ? 'var(--color-success)' : 'var(--color-text-muted)',
                        }}
                      >
                        {s.active ? '已激活' : '已停用'}
                      </span>
                    </td>
                    <td className="px-4 py-2.5 text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {s.created_at?.slice(0, 10)}
                    </td>
                    <td className="px-4 py-2.5 text-center">
                      <div className="flex items-center justify-center gap-1">
                        <button
                          onClick={() => handleToggle(s)}
                          disabled={togglingId === s.id}
                          className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors disabled:opacity-50"
                          style={{ color: s.active ? 'var(--color-success)' : 'var(--color-text-muted)' }}
                          title={s.active ? '停用' : '激活'}
                        >
                          {s.active ? <PowerOff className="w-3.5 h-3.5" /> : <Power className="w-3.5 h-3.5" />}
                        </button>
                        {confirmDeleteId === s.id ? (
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleDelete(s.id)}
                              disabled={deletingId === s.id}
                              className="px-2 py-1 rounded text-xs font-medium transition-colors"
                              style={{ background: 'rgba(239,68,68,0.15)', color: 'var(--color-danger)' }}
                            >
                              确认
                            </button>
                            <button
                              onClick={() => setConfirmDeleteId(null)}
                              className="px-2 py-1 rounded text-xs transition-colors"
                              style={{ color: 'var(--color-text-muted)' }}
                            >
                              取消
                            </button>
                          </div>
                        ) : (
                          <button
                            onClick={() => setConfirmDeleteId(s.id)}
                            className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors"
                            style={{ color: 'var(--color-text-muted)' }}
                            title="删除"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Add strategy modal */}
      {showModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center"
          style={{ background: 'rgba(0,0,0,0.6)' }}
          onClick={(e) => { if (e.target === e.currentTarget) setShowModal(false); }}
        >
          <div
            className="rounded-xl border p-6 w-full max-w-md mx-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <h2 className="text-base font-bold mb-4">添加策略</h2>
            <div className="space-y-3">
              <div>
                <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>策略名称</label>
                <input
                  value={form.name}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                  placeholder="输入策略名称"
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                  style={{
                    background: 'var(--color-bg-primary)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                />
              </div>
              <div>
                <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>描述</label>
                <input
                  value={form.description}
                  onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                  placeholder="策略描述"
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                  style={{
                    background: 'var(--color-bg-primary)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                />
              </div>
              <div>
                <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>规则 (JSON)</label>
                <textarea
                  value={form.rules}
                  onChange={(e) => setForm((f) => ({ ...f, rules: e.target.value }))}
                  placeholder='{"key": "value"}'
                  rows={5}
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none font-mono resize-y"
                  style={{
                    background: 'var(--color-bg-primary)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                />
              </div>
            </div>
            <div className="flex items-center justify-end gap-2 mt-5">
              <button
                onClick={() => { setShowModal(false); setForm(emptyForm); }}
                className="px-4 py-2 rounded-lg text-sm transition-colors hover:bg-[var(--color-bg-hover)]"
                style={{ color: 'var(--color-text-secondary)' }}
              >
                取消
              </button>
              <button
                onClick={handleAdd}
                disabled={submitting || !form.name.trim()}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50"
                style={{ background: 'var(--color-accent)', color: '#fff' }}
              >
                {submitting ? '提交中...' : '确认添加'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
