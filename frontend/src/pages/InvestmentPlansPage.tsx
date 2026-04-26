import { useState, useEffect } from 'react';
import api from '@/lib/api';
import { Target, RefreshCw, Plus, Trash2, X, TrendingUp, Shield, DollarSign } from 'lucide-react';
import { useToast } from '@/lib/toast';

interface InvestmentPlan {
  code: string;
  name: string;
  target_price: number;
  stop_loss: number;
  buy_amount: number;
  notes: string;
  created_at: string;
}

interface PlansResponse {
  ok: boolean;
  plans: Record<string, InvestmentPlan>;
}

export default function InvestmentPlansPage() {
  const [plans, setPlans] = useState<InvestmentPlan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ code: '', name: '', target_price: 0, stop_loss: 0, buy_amount: 0, notes: '' });
  const { toast: showToast } = useToast();

  const fetchPlans = () => {
    setLoading(true);
    setError('');
    api.get<PlansResponse>('/investment-plans')
      .then((res) => {
        const plansMap = res.data.plans || {};
        setPlans(Object.values(plansMap));
      })
      .catch(() => setError('加载投资计划失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchPlans(); }, []);

  const handleAdd = () => {
    if (!form.code.trim()) {
      showToast({ type: 'error', title: '请输入股票代码' });
      return;
    }
    api.post('/investment-plans', form)
      .then(() => {
        showToast({ type: 'success', title: '投资计划已创建' });
        setShowForm(false);
        setForm({ code: '', name: '', target_price: 0, stop_loss: 0, buy_amount: 0, notes: '' });
        fetchPlans();
      })
      .catch(() => showToast({ type: 'error', title: '创建失败' }));
  };

  const handleDelete = (code: string) => {
    api.delete(`/investment-plans/${code}`)
      .then(() => {
        showToast({ type: 'success', title: '已删除' });
        fetchPlans();
      })
      .catch(() => showToast({ type: 'error', title: '删除失败' }));
  };

  const formatCurrency = (val: number) => `¥${val.toFixed(2)}`;

  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Target className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">投资计划</h1>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="glass-panel rounded-xl p-4 animate-pulse" style={{ height: '120px' }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <Target className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">投资计划</h1>
        </div>
        <div className="text-sm px-4 py-3 rounded-lg" style={{ background: 'rgba(239,68,68,0.1)', color: 'var(--color-danger)' }}>
          {error}
          <button onClick={fetchPlans} className="ml-3 underline">重试</button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Target className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">投资计划</h1>
          <span className="text-xs px-2 py-0.5 rounded-full font-mono" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-muted)' }}>
            {plans.length} 个
          </span>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={fetchPlans} className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors">
            <RefreshCw className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
          </button>
          <button
            onClick={() => setShowForm(!showForm)}
            className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors"
            style={{ background: 'var(--color-accent)', color: '#fff' }}
          >
            {showForm ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
            {showForm ? '取消' : '新建计划'}
          </button>
        </div>
      </div>

      {/* Add form */}
      {showForm && (
        <div className="glass-panel rounded-xl p-4 mb-6" style={{ borderColor: 'var(--color-accent)' }}>
          <h3 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-secondary)' }}>新建投资计划</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-3">
            <div>
              <label className="text-xs mb-1 block" style={{ color: 'var(--color-text-muted)' }}>股票代码</label>
              <input
                value={form.code}
                onChange={(e) => setForm({ ...form, code: e.target.value })}
                placeholder="如 sh600000"
                className="w-full px-3 py-2 rounded-lg text-sm"
                style={{ background: 'var(--color-bg-hover)', border: '1px solid var(--color-border)', color: 'var(--color-text-primary)' }}
              />
            </div>
            <div>
              <label className="text-xs mb-1 block" style={{ color: 'var(--color-text-muted)' }}>名称</label>
              <input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="股票名称"
                className="w-full px-3 py-2 rounded-lg text-sm"
                style={{ background: 'var(--color-bg-hover)', border: '1px solid var(--color-border)', color: 'var(--color-text-primary)' }}
              />
            </div>
            <div>
              <label className="text-xs mb-1 block" style={{ color: 'var(--color-text-muted)' }}>买入金额</label>
              <input
                type="number"
                value={form.buy_amount || ''}
                onChange={(e) => setForm({ ...form, buy_amount: parseFloat(e.target.value) || 0 })}
                className="w-full px-3 py-2 rounded-lg text-sm"
                style={{ background: 'var(--color-bg-hover)', border: '1px solid var(--color-border)', color: 'var(--color-text-primary)' }}
              />
            </div>
            <div>
              <label className="text-xs mb-1 block" style={{ color: 'var(--color-text-muted)' }}>目标价</label>
              <input
                type="number"
                value={form.target_price || ''}
                onChange={(e) => setForm({ ...form, target_price: parseFloat(e.target.value) || 0 })}
                className="w-full px-3 py-2 rounded-lg text-sm"
                style={{ background: 'var(--color-bg-hover)', border: '1px solid var(--color-border)', color: 'var(--color-text-primary)' }}
              />
            </div>
            <div>
              <label className="text-xs mb-1 block" style={{ color: 'var(--color-text-muted)' }}>止损价</label>
              <input
                type="number"
                value={form.stop_loss || ''}
                onChange={(e) => setForm({ ...form, stop_loss: parseFloat(e.target.value) || 0 })}
                className="w-full px-3 py-2 rounded-lg text-sm"
                style={{ background: 'var(--color-bg-hover)', border: '1px solid var(--color-border)', color: 'var(--color-text-primary)' }}
              />
            </div>
            <div>
              <label className="text-xs mb-1 block" style={{ color: 'var(--color-text-muted)' }}>备注</label>
              <input
                value={form.notes}
                onChange={(e) => setForm({ ...form, notes: e.target.value })}
                placeholder="投资逻辑..."
                className="w-full px-3 py-2 rounded-lg text-sm"
                style={{ background: 'var(--color-bg-hover)', border: '1px solid var(--color-border)', color: 'var(--color-text-primary)' }}
              />
            </div>
          </div>
          <button
            onClick={handleAdd}
            className="px-4 py-2 rounded-lg text-sm font-medium"
            style={{ background: 'var(--color-accent)', color: '#fff' }}
          >
            创建计划
          </button>
        </div>
      )}

      {/* Plans grid */}
      {plans.length === 0 ? (
        <div className="text-center py-16 rounded-xl" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
          <Target className="w-12 h-12 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p style={{ color: 'var(--color-text-muted)' }}>暂无投资计划</p>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>点击「新建计划」开始制定投资策略</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {plans.map((p) => (
            <div key={p.code} className="glass-panel rounded-xl p-4" style={{ borderColor: 'var(--color-border)' }}>
              <div className="flex items-center justify-between mb-3">
                <div>
                  <span className="font-bold">{p.name || p.code}</span>
                  <span className="text-xs ml-2 font-mono" style={{ color: 'var(--color-text-muted)' }}>{p.code}</span>
                </div>
                <button
                  onClick={() => handleDelete(p.code)}
                  className="p-1 rounded-lg hover:bg-[rgba(239,68,68,0.1)] transition-colors"
                >
                  <Trash2 className="w-4 h-4" style={{ color: 'var(--color-danger)' }} />
                </button>
              </div>

              <div className="grid grid-cols-3 gap-2 mb-3">
                <div className="text-center p-2 rounded-lg" style={{ background: 'var(--color-bg-hover)' }}>
                  <TrendingUp className="w-4 h-4 mx-auto mb-1" style={{ color: 'var(--color-danger)' }} />
                  <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>目标价</div>
                  <div className="text-sm font-bold font-mono">{formatCurrency(p.target_price)}</div>
                </div>
                <div className="text-center p-2 rounded-lg" style={{ background: 'var(--color-bg-hover)' }}>
                  <Shield className="w-4 h-4 mx-auto mb-1" style={{ color: 'var(--color-success)' }} />
                  <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>止损价</div>
                  <div className="text-sm font-bold font-mono">{formatCurrency(p.stop_loss)}</div>
                </div>
                <div className="text-center p-2 rounded-lg" style={{ background: 'var(--color-bg-hover)' }}>
                  <DollarSign className="w-4 h-4 mx-auto mb-1" style={{ color: 'var(--color-accent)' }} />
                  <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>金额</div>
                  <div className="text-sm font-bold font-mono">{formatCurrency(p.buy_amount)}</div>
                </div>
              </div>

              {p.notes && (
                <p className="text-xs p-2 rounded-lg" style={{ background: 'var(--color-bg-hover)', color: 'var(--color-text-secondary)' }}>
                  {p.notes}
                </p>
              )}

              <div className="text-xs mt-2" style={{ color: 'var(--color-text-muted)' }}>
                创建于 {new Date(p.created_at).toLocaleDateString('zh-CN')}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
