import { useState, useEffect, useCallback, useMemo } from 'react';
import { marketApi, type Sector } from '@/lib/api';
import EmptyState from '@/components/EmptyState';
import ErrorState from '@/components/ErrorState';
import { RefreshCw, TrendingUp, TrendingDown, Minus, LayoutGrid, List, BarChart3 } from 'lucide-react';
import { SkeletonGridCard } from '@/components/ui/Skeleton';
import EChart from '@/components/charts/EChart';

// ─── Mock sector fund flow data ───
interface SectorFundFlow {
  name: string;
  netInflow: number; // 亿元
}

const MOCK_SECTOR_FUND_FLOW: SectorFundFlow[] = [
  { name: '半导体', netInflow: 42.56 },
  { name: '人工智能', netInflow: 38.72 },
  { name: '新能源汽车', netInflow: 25.18 },
  { name: '光伏', netInflow: 19.34 },
  { name: '医药生物', netInflow: 15.67 },
  { name: '军工', netInflow: 12.89 },
  { name: '消费电子', netInflow: 8.45 },
  { name: '白酒', netInflow: 6.23 },
  { name: '银行', netInflow: 3.91 },
  { name: '房地产', netInflow: 1.56 },
  { name: '证券', netInflow: -2.34 },
  { name: '钢铁', netInflow: -5.67 },
  { name: '煤炭', netInflow: -8.12 },
  { name: '传媒', netInflow: -11.45 },
  { name: '教育', netInflow: -14.23 },
  { name: '旅游', netInflow: -16.78 },
  { name: '农牧饲渔', netInflow: -19.56 },
  { name: '纺织服装', netInflow: -22.34 },
  { name: '环保', netInflow: -25.67 },
  { name: '公用事业', netInflow: -28.91 },
];

// ─── Color helpers ───

// Interpolate color based on change percent
function heatColor(pct: number): string {
  // Clamp to [-5, 5] range for color intensity
  const clamped = Math.max(-5, Math.min(5, pct));
  const intensity = Math.abs(clamped) / 5; // 0..1

  if (pct > 0) {
    // Red spectrum (Chinese market: red = up)
    const r = Math.round(180 + 75 * intensity);
    const g = Math.round(60 - 30 * intensity);
    const b = Math.round(50 - 20 * intensity);
    return `rgb(${r}, ${g}, ${b})`;
  } else if (pct < 0) {
    // Green spectrum (Chinese market: green = down)
    const r = Math.round(40 - 15 * intensity);
    const g = Math.round(150 + 80 * intensity);
    const b = Math.round(70 + 30 * intensity);
    return `rgb(${r}, ${g}, ${b})`;
  }
  return 'var(--color-bg-card)';
}

function heatTextColor(pct: number): string {
  const absPct = Math.abs(pct);
  return absPct > 1.5 ? '#fff' : 'var(--color-text-primary)';
}

// ─── Fund flow chart option builder ───
function buildFundFlowChartOption(data: SectorFundFlow[]) {
  // Sort by netInflow descending, take top 15
  const sorted = [...data].sort((a, b) => b.netInflow - a.netInflow).slice(0, 15);

  // Reverse for horizontal bar chart so highest is at top
  const reversed = [...sorted].reverse();

  const names = reversed.map((d) => d.name);
  const values = reversed.map((d) => d.netInflow);
  const colors = reversed.map((d) => (d.netInflow >= 0 ? '#ef4444' : '#22c55e'));

  return {
    tooltip: {
      trigger: 'axis' as const,
      axisPointer: { type: 'shadow' as const },
      formatter: (params: { name: string; value: number }[]) => {
        const item = params[0];
        if (!item) return '';
        const sign = item.value >= 0 ? '+' : '';
        const color = item.value >= 0 ? '#ef4444' : '#22c55e';
        return `<div style="font-size:13px">
          <div style="font-weight:600;margin-bottom:4px">${item.name}</div>
          <div>净流入：<span style="color:${color};font-weight:600">${sign}${item.value.toFixed(2)} 亿元</span></div>
        </div>`;
      },
    },
    grid: {
      left: 100,
      right: 40,
      top: 16,
      bottom: 30,
    },
    xAxis: {
      type: 'value' as const,
      name: '金额（亿元）',
      nameTextStyle: {
        color: '#94a3b8',
        fontSize: 11,
      },
      axisLabel: {
        color: '#94a3b8',
        formatter: (v: number) => v.toFixed(0),
      },
      splitLine: {
        lineStyle: { color: 'rgba(51,65,85,0.4)' },
      },
    },
    yAxis: {
      type: 'category' as const,
      data: names,
      axisLabel: {
        color: '#e2e8f0',
        fontSize: 12,
        width: 80,
        overflow: 'truncate',
      },
      axisLine: { lineStyle: { color: '#334155' } },
    },
    series: [
      {
        type: 'bar',
        data: values.map((v, i) => ({
          value: v,
          itemStyle: {
            color: colors[i],
            borderRadius: v >= 0 ? [0, 4, 4, 0] : [4, 0, 0, 4],
          },
        })),
        barWidth: '55%',
        label: {
          show: true,
          position: 'right' as const,
          formatter: (p: { value: number }) => {
            const v = p.value;
            return (v >= 0 ? '+' : '') + v.toFixed(2);
          },
          fontSize: 11,
          color: '#94a3b8',
        },
      },
    ],
  };
}

export default function SectorsPage() {
  const [sectors, setSectors] = useState<Sector[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [viewMode, setViewMode] = useState<'heatmap' | 'list'>('heatmap');

  const fetchSectors = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await marketApi.sectors();
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const d = res.data as any;
      setSectors(Array.isArray(d) ? d : Array.isArray(d.sectors) ? d.sectors : []);
    } catch {
      setError('加载板块数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSectors();
  }, [fetchSectors]);

  // Sort by change_percent descending
  const sorted = [...sectors].sort((a, b) => b.change_percent - a.change_percent);

  // Summary stats
  const upCount = sectors.filter((s) => s.change_percent > 0).length;
  const downCount = sectors.filter((s) => s.change_percent < 0).length;
  const flatCount = sectors.filter((s) => s.change_percent === 0).length;
  const maxGainer = sorted[0];
  const maxLoser = sorted[sorted.length - 1];

  const changeColor = (n: number) =>
    n > 0 ? 'var(--color-danger)' : n < 0 ? 'var(--color-success)' : 'var(--color-text-secondary)';

  // Fund flow chart option — memoized
  const fundFlowChartOption = useMemo(() => buildFundFlowChartOption(MOCK_SECTOR_FUND_FLOW), []);

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-bold">板块行情</h1>
          <p className="text-xs mt-0.5" style={{ color: 'var(--color-text-muted)' }}>
            颜色深浅反映涨跌幅度，红色上涨，绿色下跌
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* View toggle */}
          <div className="flex rounded-lg overflow-hidden border" style={{ borderColor: 'var(--color-border)' }}>
            <button
              onClick={() => setViewMode('heatmap')}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm transition-colors"
              style={{
                background: viewMode === 'heatmap' ? 'var(--color-accent)' : 'var(--color-bg-card)',
                color: viewMode === 'heatmap' ? '#fff' : 'var(--color-text-muted)',
              }}
            >
              <LayoutGrid className="w-3.5 h-3.5" />
              热力图
            </button>
            <button
              onClick={() => setViewMode('list')}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm transition-colors"
              style={{
                background: viewMode === 'list' ? 'var(--color-accent)' : 'var(--color-bg-card)',
                color: viewMode === 'list' ? '#fff' : 'var(--color-text-muted)',
              }}
            >
              <List className="w-3.5 h-3.5" />
              列表
            </button>
          </div>
          <button
            onClick={fetchSectors}
            disabled={loading}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm hover:bg-[var(--color-bg-hover)] transition-colors"
            style={{ color: 'var(--color-text-secondary)' }}
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </button>
        </div>
      </div>

      {/* ─── Fund Flow Bar Chart Card ─── */}
      <div className="mb-4 bg-[#222536]/80 backdrop-blur-lg rounded-xl border border-white/10 p-5">
        <div className="flex items-center gap-2 mb-4">
          <BarChart3 className="w-4 h-4" style={{ color: 'var(--color-accent)' }} />
          <h2 className="text-sm font-semibold">板块资金流向</h2>
          <span className="text-[10px] px-1.5 py-0.5 rounded" style={{ background: 'rgba(59,130,246,0.15)', color: 'var(--color-accent)' }}>
            Top 15 净流入/流出排名
          </span>
        </div>
        <EChart option={fundFlowChartOption} height={400} />
      </div>

      {/* Summary bar */}
      {sectors.length > 0 && (
        <div
          className="flex items-center gap-4 mb-4 px-4 py-2.5 rounded-lg text-xs"
          style={{ background: 'var(--color-bg-secondary)' }}
        >
          <span style={{ color: 'var(--color-text-muted)' }}>共 {sectors.length} 个板块</span>
          <span style={{ color: 'var(--color-danger)' }}>
            <TrendingUp className="w-3 h-3 inline mr-1" />
            上涨 {upCount}
          </span>
          <span style={{ color: 'var(--color-success)' }}>
            <TrendingDown className="w-3 h-3 inline mr-1" />
            下跌 {downCount}
          </span>
          {flatCount > 0 && (
            <span style={{ color: 'var(--color-text-muted)' }}>
              <Minus className="w-3 h-3 inline mr-1" />
              平盘 {flatCount}
            </span>
          )}
          {maxGainer && (
            <span style={{ color: 'var(--color-danger)' }}>
              领涨: {maxGainer.name} +{maxGainer.change_percent.toFixed(2)}%
            </span>
          )}
          {maxLoser && maxLoser.change_percent < 0 && (
            <span style={{ color: 'var(--color-success)' }}>
              领跌: {maxLoser.name} {maxLoser.change_percent.toFixed(2)}%
            </span>
          )}
        </div>
      )}

      {error && (
        <div className="mb-4">
          <ErrorState
            title="加载失败"
            description={error}
            onRetry={() => { setError(''); fetchSectors(); }}
          />
        </div>
      )}

      {loading && sectors.length === 0 ? (
        <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 gap-1">
          {Array.from({ length: 18 }).map((_, i) => (
            <SkeletonGridCard key={i} />
          ))}
        </div>
      ) : sorted.length === 0 ? (
        <EmptyState
          icon={LayoutGrid}
          title="暂无板块数据"
          description="板块数据加载中"
        />
      ) : viewMode === 'heatmap' ? (
        /* ── Heatmap / Treemap view ── */
        <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 gap-1">
          {sorted.map((sector) => {
            const pct = sector.change_percent;
            const bg = heatColor(pct);
            const fg = heatTextColor(pct);
            const absPct = Math.abs(pct);

            return (
              <div
                key={sector.code}
                className="rounded-md p-2.5 flex flex-col justify-between cursor-default transition-transform hover:scale-[1.03] hover:z-10 min-h-[72px]"
                style={{ background: bg }}
                title={`${sector.name} (${sector.code})\n价格: ${sector.price.toFixed(2)}\n涨跌幅: ${pct >= 0 ? '+' : ''}${pct.toFixed(2)}%`}
              >
                <div className="text-xs font-medium truncate" style={{ color: fg }}>
                  {sector.name}
                </div>
                <div className="mt-auto">
                  <div className="text-sm font-bold font-mono" style={{ color: fg }}>
                    {pct >= 0 ? '+' : ''}
                    {pct.toFixed(2)}%
                  </div>
                  {absPct >= 2 && (
                    <div className="text-[10px] font-mono mt-0.5" style={{ color: fg, opacity: 0.7 }}>
                      {sector.price.toFixed(2)}
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        /* ── List / Table view ── */
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {sorted.map((sector) => {
            const pct = sector.change_percent;
            const Icon = pct > 0 ? TrendingUp : pct < 0 ? TrendingDown : Minus;
            const bg = heatColor(pct);

            return (
              <div
                key={sector.code}
                className="rounded-lg border p-3 flex items-center gap-3 hover:bg-[var(--color-bg-hover)] transition-colors"
                style={{ background: 'var(--color-bg-secondary)', borderColor: 'var(--color-border)' }}
              >
                {/* Color indicator bar */}
                <div
                  className="w-1 self-stretch rounded-full flex-shrink-0"
                  style={{ background: bg }}
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <span className="font-medium text-sm truncate">{sector.name}</span>
                    <span className="text-xs font-mono ml-2 flex-shrink-0" style={{ color: 'var(--color-text-muted)' }}>
                      {sector.code}
                    </span>
                  </div>
                  <div className="flex items-center justify-between mt-1">
                    <span className="font-mono text-sm" style={{ color: 'var(--color-text-secondary)' }}>
                      {sector.price.toFixed(2)}
                    </span>
                    <span className="flex items-center gap-1 font-mono text-sm font-medium" style={{ color: changeColor(pct) }}>
                      <Icon className="w-3.5 h-3.5" />
                      {pct >= 0 ? '+' : ''}
                      {pct.toFixed(2)}%
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Color legend */}
      {viewMode === 'heatmap' && sectors.length > 0 && (
        <div className="flex items-center justify-center gap-2 mt-4">
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>跌</span>
          <div className="flex h-2 rounded-full overflow-hidden" style={{ width: 200 }}>
            {[-5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5].map((v) => (
              <div
                key={v}
                className="flex-1"
                style={{ background: heatColor(v) }}
              />
            ))}
          </div>
          <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>涨</span>
        </div>
      )}
    </div>
  );
}
