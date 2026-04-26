import { useState, useEffect, useCallback } from 'react';
import { portfolioApi, type PortfolioPosition, type PortfolioAnalytics, type PortfolioRisk } from '@/lib/api';
import { Plus, Trash2, TrendingUp, TrendingDown, PieChart, Shield, RefreshCw } from 'lucide-react';

interface AddForm {
  code: string;
  quantity: string;
  cost_price: string;
  buy_date: string;
}

const emptyForm: AddForm = { code: '', quantity: '', cost_price: '', buy_date: '' };

export default function PortfolioPage() {
  const [positions, setPositions] = useState<PortfolioPosition[]>([]);
  const [analytics, setAnalytics] = useState<PortfolioAnalytics | null>(null);
  const [risk, setRisk] = useState<PortfolioRisk | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState<AddForm>(emptyForm);
  const [submitting, setSubmitting] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [posRes, anaRes, riskRes] = await Promise.allSettled([
        portfolioApi.list(),
        portfolioApi.analytics(),
        portfolioApi.risk(),
      ]);
      if (posRes.status === 'fulfilled') setPositions(posRes.value.data);
      if (anaRes.status === 'fulfilled') setAnalytics(anaRes.value.data);
      if (riskRes.status === 'fulfilled') setRisk(riskRes.value.data);
    } catch {
      setError('加载数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleAdd = async () => {
    if (!form.code || !form.quantity || !form.cost_price || !form.buy_date) return;
    setSubmitting(true);
    try {
      await portfolioApi.add({
        code: form.code.trim().toUpperCase(),
        quantity: Number(form.quantity),
        cost_price: Number(form.cost_price),
        buy_date: form.buy_date,
      });
      setShowModal(false);
      setForm(emptyForm);
      await fetchData();
    } catch {
      setError('添加持仓失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    setDeletingId(id);
    try {
      await portfolioApi.remove(id);
      await fetchData();
    } catch {
      setError('删除持仓失败');
    } finally {
      setDeletingId(null);
    }
  };

  const pnlColor = (n: number) =>
    n > 0
      ? 'var(--color-danger)'
      : n < 0
        ? 'var(--color-success)'
        : 'var(--color-text-secondary)';

  const formatPct = (n: number) => `${n >= 0 ? '+' : ''}${n.toFixed(2)}%`;
  const formatNum = (n: number) => n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <PieChart className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">投资组合</h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowModal(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors"
            style={{
              background: 'var(--color-accent)',
              color: '#fff',
            }}
          >
            <Plus className="w-3.5 h-3.5" />
            添加持仓
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

      {/* Analytics summary */}
      {analytics && (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
          <div
            className="rounded-xl border p-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>总市值</div>
            <div className="text-lg font-bold font-mono">{formatNum(analytics.total_value)}</div>
          </div>
          <div
            className="rounded-xl border p-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>总成本</div>
            <div className="text-lg font-bold font-mono">{formatNum(analytics.total_cost)}</div>
          </div>
          <div
            className="rounded-xl border p-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>总盈亏</div>
            <div className="text-lg font-bold font-mono" style={{ color: pnlColor(analytics.total_profit_loss) }}>
              {formatNum(analytics.total_profit_loss)}
              <span className="text-sm ml-1">{formatPct(analytics.total_profit_loss_pct)}</span>
            </div>
          </div>
          <div
            className="rounded-xl border p-4"
            style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
          >
            <div className="text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>持仓数量</div>
            <div className="text-lg font-bold font-mono">{analytics.position_count}</div>
          </div>
        </div>
      )}

      {/* Positions table */}
      <div
        className="rounded-xl border mb-6 overflow-hidden"
        style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
      >
        <div className="px-4 py-3 flex items-center justify-between border-b" style={{ borderColor: 'var(--color-border)' }}>
          <span className="text-sm font-medium">持仓明细</span>
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
            {positions.length} 只股票
          </span>
        </div>

        {positions.length === 0 ? (
          <div className="p-8 text-center">
            <p className="text-sm" style={{ color: 'var(--color-text-muted)' }}>
              {loading ? '加载持仓数据...' : '暂无持仓，点击「添加持仓」开始'}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr
                  className="text-xs"
                  style={{ color: 'var(--color-text-muted)', borderBottom: '1px solid var(--color-border)' }}
                >
                  <th className="text-left px-4 py-2.5 font-medium">代码</th>
                  <th className="text-left px-4 py-2.5 font-medium">名称</th>
                  <th className="text-right px-4 py-2.5 font-medium">数量</th>
                  <th className="text-right px-4 py-2.5 font-medium">成本价</th>
                  <th className="text-right px-4 py-2.5 font-medium">现价</th>
                  <th className="text-right px-4 py-2.5 font-medium">市值</th>
                  <th className="text-right px-4 py-2.5 font-medium">盈亏%</th>
                  <th className="text-center px-4 py-2.5 font-medium">操作</th>
                </tr>
              </thead>
              <tbody>
                {positions.map((pos) => (
                  <tr
                    key={pos.id}
                    className="transition-colors hover:bg-[var(--color-bg-hover)]"
                    style={{ borderBottom: '1px solid var(--color-border)' }}
                  >
                    <td className="px-4 py-2.5 font-mono text-xs" style={{ color: 'var(--color-accent)' }}>
                      {pos.code}
                    </td>
                    <td className="px-4 py-2.5">{pos.name}</td>
                    <td className="px-4 py-2.5 text-right font-mono">{pos.quantity}</td>
                    <td className="px-4 py-2.5 text-right font-mono">{pos.cost_price.toFixed(2)}</td>
                    <td className="px-4 py-2.5 text-right font-mono">{pos.current_price.toFixed(2)}</td>
                    <td className="px-4 py-2.5 text-right font-mono">{formatNum(pos.market_value)}</td>
                    <td className="px-4 py-2.5 text-right font-mono font-medium" style={{ color: pnlColor(pos.profit_loss_pct) }}>
                      {formatPct(pos.profit_loss_pct)}
                    </td>
                    <td className="px-4 py-2.5 text-center">
                      <button
                        onClick={() => handleDelete(pos.id)}
                        disabled={deletingId === pos.id}
                        className="p-1.5 rounded-lg hover:bg-[var(--color-bg-hover)] transition-colors disabled:opacity-50"
                        style={{ color: 'var(--color-text-muted)' }}
                        title="删除"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Top gainers / losers */}
      {analytics && (analytics.top_gainers.length > 0 || analytics.top_losers.length > 0) && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 mb-6">
          {analytics.top_gainers.length > 0 && (
            <div
              className="rounded-xl border p-4"
              style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
            >
              <div className="flex items-center gap-2 mb-3">
                <TrendingUp className="w-4 h-4" style={{ color: 'var(--color-danger)' }} />
                <span className="text-sm font-medium">最佳持仓</span>
              </div>
              <div className="space-y-1.5">
                {analytics.top_gainers.slice(0, 5).map((p) => (
                  <div key={p.id} className="flex items-center justify-between px-2 py-1.5 rounded-lg">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="text-xs font-mono" style={{ color: 'var(--color-accent)' }}>{p.code}</span>
                      <span className="text-sm truncate" style={{ color: 'var(--color-text-secondary)' }}>{p.name}</span>
                    </div>
                    <span className="font-mono text-sm font-medium flex-shrink-0" style={{ color: 'var(--color-danger)' }}>
                      {formatPct(p.profit_loss_pct)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
          {analytics.top_losers.length > 0 && (
            <div
              className="rounded-xl border p-4"
              style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
            >
              <div className="flex items-center gap-2 mb-3">
                <TrendingDown className="w-4 h-4" style={{ color: 'var(--color-success)' }} />
                <span className="text-sm font-medium">最差持仓</span>
              </div>
              <div className="space-y-1.5">
                {analytics.top_losers.slice(0, 5).map((p) => (
                  <div key={p.id} className="flex items-center justify-between px-2 py-1.5 rounded-lg">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="text-xs font-mono" style={{ color: 'var(--color-accent)' }}>{p.code}</span>
                      <span className="text-sm truncate" style={{ color: 'var(--color-text-secondary)' }}>{p.name}</span>
                    </div>
                    <span className="font-mono text-sm font-medium flex-shrink-0" style={{ color: 'var(--color-success)' }}>
                      {formatPct(p.profit_loss_pct)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Sector allocation */}
      {analytics && analytics.sector_allocation.length > 0 && (
        <div
          className="rounded-xl border p-4 mb-6"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <div className="flex items-center gap-2 mb-3">
            <PieChart className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
            <span className="text-sm font-medium">行业配置</span>
          </div>
          <div className="space-y-2">
            {analytics.sector_allocation.map((s) => (
              <div key={s.sector}>
                <div className="flex items-center justify-between text-xs mb-1">
                  <span style={{ color: 'var(--color-text-secondary)' }}>{s.sector}</span>
                  <span className="font-mono" style={{ color: 'var(--color-text-muted)' }}>
                    {formatNum(s.value)} ({s.pct.toFixed(1)}%)
                  </span>
                </div>
                <div className="h-1.5 rounded-full" style={{ background: 'var(--color-bg-primary)' }}>
                  <div
                    className="h-full rounded-full transition-all"
                    style={{ width: `${Math.min(s.pct, 100)}%`, background: 'var(--color-accent)' }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Risk analysis */}
      {risk && (
        <div
          className="rounded-xl border p-4"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <div className="flex items-center gap-2 mb-4">
            <Shield className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
            <span className="text-sm font-medium">风险分析</span>
            <span
              className="text-xs px-2 py-0.5 rounded-full font-medium ml-auto"
              style={{
                background: risk.risk_level === 'low'
                  ? 'rgba(34,197,94,0.15)'
                  : risk.risk_level === 'medium'
                    ? 'rgba(234,179,8,0.15)'
                    : 'rgba(239,68,68,0.15)',
                color: risk.risk_level === 'low'
                  ? 'var(--color-success)'
                  : risk.risk_level === 'medium'
                    ? 'var(--color-warning)'
                    : 'var(--color-danger)',
              }}
            >
              {risk.risk_level === 'low' ? '低风险' : risk.risk_level === 'medium' ? '中风险' : '高风险'}
            </span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-4">
            <div className="px-3 py-2 rounded-lg" style={{ background: 'var(--color-bg-primary)' }}>
              <div className="text-xs mb-0.5" style={{ color: 'var(--color-text-muted)' }}>集中度风险</div>
              <div className="font-mono font-medium">{(risk.concentration_risk * 100).toFixed(1)}%</div>
            </div>
            <div className="px-3 py-2 rounded-lg" style={{ background: 'var(--color-bg-primary)' }}>
              <div className="text-xs mb-0.5" style={{ color: 'var(--color-text-muted)' }}>最大单只持仓占比</div>
              <div className="font-mono font-medium">{(risk.max_single_position_pct * 100).toFixed(1)}%</div>
            </div>
          </div>

          {risk.sector_concentration.length > 0 && (
            <div className="mb-4">
              <div className="text-xs font-medium mb-2" style={{ color: 'var(--color-text-muted)' }}>行业集中度</div>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                {risk.sector_concentration.map((s) => (
                  <div key={s.sector} className="px-3 py-2 rounded-lg" style={{ background: 'var(--color-bg-primary)' }}>
                    <div className="text-xs truncate" style={{ color: 'var(--color-text-secondary)' }}>{s.sector}</div>
                    <div className="font-mono text-sm font-medium">{(s.pct * 100).toFixed(1)}%</div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {risk.suggestions.length > 0 && (
            <div>
              <div className="text-xs font-medium mb-2" style={{ color: 'var(--color-text-muted)' }}>建议</div>
              <ul className="space-y-1.5">
                {risk.suggestions.map((s, i) => (
                  <li
                    key={i}
                    className="text-sm px-3 py-1.5 rounded-lg"
                    style={{ color: 'var(--color-text-secondary)', background: 'var(--color-bg-primary)' }}
                  >
                    {s}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {/* Add position modal */}
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
            <h2 className="text-base font-bold mb-4">添加持仓</h2>
            <div className="space-y-3">
              <div>
                <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>股票代码</label>
                <input
                  value={form.code}
                  onChange={(e) => setForm((f) => ({ ...f, code: e.target.value }))}
                  placeholder="如 600519"
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none"
                  style={{
                    background: 'var(--color-bg-primary)',
                    border: '1px solid var(--color-border)',
                    color: 'var(--color-text-primary)',
                  }}
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>数量</label>
                  <input
                    type="number"
                    value={form.quantity}
                    onChange={(e) => setForm((f) => ({ ...f, quantity: e.target.value }))}
                    placeholder="股数"
                    className="w-full px-3 py-2 rounded-lg text-sm outline-none font-mono"
                    style={{
                      background: 'var(--color-bg-primary)',
                      border: '1px solid var(--color-border)',
                      color: 'var(--color-text-primary)',
                    }}
                  />
                </div>
                <div>
                  <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>成本价</label>
                  <input
                    type="number"
                    step="0.01"
                    value={form.cost_price}
                    onChange={(e) => setForm((f) => ({ ...f, cost_price: e.target.value }))}
                    placeholder="元"
                    className="w-full px-3 py-2 rounded-lg text-sm outline-none font-mono"
                    style={{
                      background: 'var(--color-bg-primary)',
                      border: '1px solid var(--color-border)',
                      color: 'var(--color-text-primary)',
                    }}
                  />
                </div>
              </div>
              <div>
                <label className="block text-xs mb-1" style={{ color: 'var(--color-text-muted)' }}>买入日期</label>
                <input
                  type="date"
                  value={form.buy_date}
                  onChange={(e) => setForm((f) => ({ ...f, buy_date: e.target.value }))}
                  className="w-full px-3 py-2 rounded-lg text-sm outline-none"
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
                disabled={submitting || !form.code || !form.quantity || !form.cost_price || !form.buy_date}
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
