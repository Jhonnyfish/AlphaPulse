import { useState, useEffect, useCallback } from 'react';
import { useView } from '@/lib/ViewContext';
import { portfolioApi, type PortfolioRisk } from '@/lib/api';
import { Shield, AlertTriangle, CheckCircle, TrendingDown, PieChart, BarChart3, RefreshCw, ArrowLeft } from 'lucide-react';

const riskLevelConfig: Record<string, { label: string; color: string; bg: string; icon: typeof Shield }> = {
  low: { label: '低风险', color: 'var(--color-success)', bg: 'rgba(34,197,94,0.15)', icon: CheckCircle },
  medium: { label: '中风险', color: 'var(--color-warning)', bg: 'rgba(234,179,8,0.15)', icon: AlertTriangle },
  high: { label: '高风险', color: 'var(--color-danger)', bg: 'rgba(239,68,68,0.15)', icon: TrendingDown },
};

function getRiskConfig(level: string) {
  return riskLevelConfig[level] || riskLevelConfig.medium;
}

export default function PortfolioRiskPage() {
  const { navigate } = useView();
  const [risk, setRisk] = useState<PortfolioRisk | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await portfolioApi.risk();
      setRisk(res.data.data);
    } catch {
      setError('加载风险数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Loading state
  if (loading) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <button
            onClick={() => navigate('portfolio')}
            className="flex items-center gap-1 text-sm hover:underline"
            style={{ color: 'var(--color-text-muted)' }}
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            返回组合
          </button>
        </div>
        <div className="space-y-4 py-4">
            <SkeletonStatCards count={4} />
            <SkeletonInlineTable rows={5} columns={5} />
          </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <button
            onClick={() => navigate('portfolio')}
            className="flex items-center gap-1 text-sm hover:underline"
            style={{ color: 'var(--color-text-muted)' }}
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            返回组合
          </button>
        </div>
        <div
          className="rounded-xl border p-8 text-center"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <AlertTriangle className="w-8 h-8 mx-auto mb-3" style={{ color: 'var(--color-danger)' }} />
          <p className="text-sm mb-3" style={{ color: 'var(--color-danger)' }}>{error}</p>
          <button
            onClick={fetchData}
            className="px-4 py-2 rounded-lg text-sm transition-colors"
            style={{ background: 'var(--color-accent)', color: '#fff' }}
          >
            重试
          </button>
        </div>
      </div>
    );
  }

  // Empty state — no risk data returned (likely no portfolio)
  if (!risk) {
    return (
      <div>
        <div className="flex items-center gap-2 mb-6">
          <button
            onClick={() => navigate('portfolio')}
            className="flex items-center gap-1 text-sm hover:underline"
            style={{ color: 'var(--color-text-muted)' }}
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            返回组合
          </button>
        </div>
        <div
          className="rounded-xl border p-12 text-center"
          style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
        >
          <Shield className="w-10 h-10 mx-auto mb-3" style={{ color: 'var(--color-text-muted)' }} />
          <p className="text-base font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>
            暂无持仓数据
          </p>
          <p className="text-sm mb-4" style={{ color: 'var(--color-text-muted)' }}>
            请先添加持仓
          </p>
          <button
            onClick={() => navigate('portfolio')}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium transition-colors"
            style={{ background: 'var(--color-accent)', color: '#fff' }}
          >
            前往持仓管理
          </button>
        </div>
      </div>
    );
  }

  const riskCfg = getRiskConfig(risk.risk_level);
  const RiskIcon = riskCfg.icon;
  const concentrationPct = (risk.concentration_risk * 100).toFixed(1);
  const maxPositionPct = (risk.max_single_position_pct * 100).toFixed(1);
  const sectorCount = risk.sector_concentration.length;

  return (
    <div>
      {/* Back link */}
      <div className="flex items-center gap-2 mb-6">
        <button
          onClick={() => navigate('portfolio')}
          className="flex items-center gap-1 text-sm hover:underline"
          style={{ color: 'var(--color-text-muted)' }}
        >
          <ArrowLeft className="w-4 h-4" />
          返回持仓
        </button>
      </div>

      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Shield className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          <h1 className="text-xl font-bold">组合风险分析</h1>
        </div>
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

      {/* Risk Level Card */}
      <div
        className="glass-panel rounded-xl border p-6 mb-6 text-center"
        style={{ borderColor: 'var(--color-border)' }}
      >
        <div className="flex items-center justify-center gap-3 mb-2">
          <RiskIcon className="w-8 h-8" style={{ color: riskCfg.color }} />
          <span
            className="text-2xl font-bold px-5 py-1.5 rounded-full"
            style={{ background: riskCfg.bg, color: riskCfg.color }}
          >
            {riskCfg.label}
          </span>
        </div>
        <p className="text-xs mt-2" style={{ color: 'var(--color-text-muted)' }}>
          综合风险评级
        </p>
      </div>

      {/* Key Metrics Row */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-6">
        {/* Concentration Risk */}
        <div
          className="glass-panel rounded-xl border p-4"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <div className="flex items-center gap-2 mb-3">
            <BarChart3 className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
            <span className="text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
              集中度风险
            </span>
          </div>
          <div className="font-mono text-2xl font-bold mb-2" style={{ color: 'var(--color-text-primary)' }}>
            {concentrationPct}%
          </div>
          <div className="h-2 rounded-full" style={{ background: 'var(--color-bg-primary)' }}>
            <div
              className="h-full rounded-full transition-all"
              style={{
                width: `${Math.min(parseFloat(concentrationPct), 100)}%`,
                background:
                  parseFloat(concentrationPct) > 70
                    ? 'var(--color-danger)'
                    : parseFloat(concentrationPct) > 40
                      ? 'var(--color-warning)'
                      : 'var(--color-success)',
              }}
            />
          </div>
        </div>

        {/* Max Single Position */}
        <div
          className="glass-panel rounded-xl border p-4"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <div className="flex items-center gap-2 mb-3">
            <PieChart className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
            <span className="text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
              最大单只占比
            </span>
          </div>
          <div className="font-mono text-2xl font-bold mb-2" style={{ color: 'var(--color-text-primary)' }}>
            {maxPositionPct}%
          </div>
          <div className="h-2 rounded-full" style={{ background: 'var(--color-bg-primary)' }}>
            <div
              className="h-full rounded-full transition-all"
              style={{
                width: `${Math.min(parseFloat(maxPositionPct), 100)}%`,
                background:
                  parseFloat(maxPositionPct) > 50
                    ? 'var(--color-danger)'
                    : parseFloat(maxPositionPct) > 30
                      ? 'var(--color-warning)'
                      : 'var(--color-success)',
              }}
            />
          </div>
        </div>

        {/* Sector Count */}
        <div
          className="glass-panel rounded-xl border p-4"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <div className="flex items-center gap-2 mb-3">
            <BarChart3 className="w-4 h-4" style={{ color: 'var(--color-success)' }} />
            <span className="text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
              持仓行业数
            </span>
          </div>
          <div className="font-mono text-2xl font-bold mb-2" style={{ color: 'var(--color-text-primary)' }}>
            {sectorCount}
          </div>
          <div className="h-2 rounded-full" style={{ background: 'var(--color-bg-primary)' }}>
            <div
              className="h-full rounded-full transition-all"
              style={{
                width: `${Math.min((sectorCount / 10) * 100, 100)}%`,
                background: sectorCount >= 5 ? 'var(--color-success)' : sectorCount >= 3 ? 'var(--color-warning)' : 'var(--color-danger)',
              }}
            />
          </div>
        </div>
      </div>

      {/* Sector Concentration Table */}
      {risk.sector_concentration.length > 0 && (
        <div
          className="glass-panel rounded-xl border overflow-hidden mb-6"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <div
            className="px-4 py-3 flex items-center justify-between border-b"
            style={{ borderColor: 'var(--color-border)' }}
          >
            <div className="flex items-center gap-2">
              <PieChart className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
              <span className="text-sm font-medium">行业集中度</span>
            </div>
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
              {sectorCount} 个行业
            </span>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr
                  className="text-xs"
                  style={{ color: 'var(--color-text-muted)', borderBottom: '1px solid var(--color-border)' }}
                >
                  <th className="text-left px-4 py-2.5 font-medium">行业</th>
                  <th className="text-left px-4 py-2.5 font-medium">占比分布</th>
                  <th className="text-right px-4 py-2.5 font-medium">占比</th>
                </tr>
              </thead>
              <tbody>
                {risk.sector_concentration
                  .sort((a, b) => b.pct - a.pct)
                  .map((s) => {
                    const pct = (s.pct * 100).toFixed(1);
                    const pctNum = s.pct * 100;
                    return (
                      <tr
                        key={s.sector}
                        className="transition-colors hover:bg-[var(--color-bg-hover)]"
                        style={{ borderBottom: '1px solid var(--color-border)' }}
                      >
                        <td className="px-4 py-2.5 font-medium">{s.sector}</td>
                        <td className="px-4 py-2.5">
                          <div className="h-2 rounded-full max-w-[200px]" style={{ background: 'var(--color-bg-primary)' }}>
                            <div
                              className="h-full rounded-full transition-all"
                              style={{
                                width: `${Math.min(pctNum, 100)}%`,
                                background:
                                  pctNum > 40
                                    ? 'var(--color-danger)'
                                    : pctNum > 25
                                      ? 'var(--color-warning)'
                                      : 'var(--color-accent)',
                              }}
                            />
                          </div>
                        </td>
                        <td className="px-4 py-2.5 text-right font-mono font-medium">
                          {pct}%
                        </td>
                      </tr>
                    );
                  })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Suggestions */}
      {risk.suggestions.length > 0 && (
        <div
          className="glass-panel rounded-xl border p-4"
          style={{ borderColor: 'var(--color-border)' }}
        >
          <div className="flex items-center gap-2 mb-3">
            <AlertTriangle className="w-4 h-4" style={{ color: 'var(--color-warning)' }} />
            <span className="text-sm font-medium">风险建议</span>
          </div>
          <ul className="space-y-2">
            {risk.suggestions.map((s, i) => (
              <li
                key={i}
                className="flex items-start gap-2 text-sm px-3 py-2 rounded-lg"
                style={{ color: 'var(--color-text-secondary)', background: 'var(--color-bg-primary)' }}
              >
                <CheckCircle className="w-4 h-4 mt-0.5 flex-shrink-0" style={{ color: 'var(--color-accent)' }} />
                <span>{s}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
