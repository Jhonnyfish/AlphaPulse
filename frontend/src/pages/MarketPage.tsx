import { useCallback, useEffect, useMemo, useState } from 'react';
import type { EChartsOption } from 'echarts';
import {
  marketApi,
  type MarketOverview,
  type Quote,
  type SearchSuggestion,
  type TopMover,
} from '@/lib/api';
import EChart from '@/components/charts/EChart';
import EmptyState from '@/components/EmptyState';
import ErrorState from '@/components/ErrorState';
import StockSearch from '@/components/StockSearch';
import Alpha300Selector from '@/components/Alpha300Selector';
import { Skeleton, SkeletonInlineTable, SkeletonStatCards } from '@/components/ui/Skeleton';
import {
  Activity,
  ArrowDownRight,
  ArrowUpRight,
  BarChart3,
  CandlestickChart,
  Loader2,
  Minus,
  RefreshCw,
  Search,
  TrendingDown,
  TrendingUp,
  Wallet,
} from 'lucide-react';

const INDEX_PRIORITY = ['上证', '深证', '创业'];

function formatAmount(value: number) {
  if (!Number.isFinite(value)) return '--';
  if (Math.abs(value) >= 1e8) return `${(value / 1e8).toFixed(2)}亿`;
  if (Math.abs(value) >= 1e4) return `${(value / 1e4).toFixed(2)}万`;
  return value.toFixed(2);
}

function formatVolume(value: number) {
  if (!Number.isFinite(value)) return '--';
  if (Math.abs(value) >= 1e8) return `${(value / 1e8).toFixed(2)}亿`;
  if (Math.abs(value) >= 1e4) return `${(value / 1e4).toFixed(2)}万`;
  return value.toLocaleString('zh-CN');
}

function formatSigned(value: number, digits = 2) {
  if (!Number.isFinite(value)) return '--';
  return `${value >= 0 ? '+' : ''}${value.toFixed(digits)}`;
}

function changeColor(value: number) {
  if (value > 0) return 'var(--color-danger)';
  if (value < 0) return 'var(--color-success)';
  return 'var(--color-text-secondary)';
}

function changeTextClass(value: number) {
  if (value > 0) return 'text-[var(--color-danger)]';
  if (value < 0) return 'text-[var(--color-success)]';
  return 'text-[var(--color-text-secondary)]';
}

function dedupeMovers(items: TopMover[]) {
  const map = new Map<string, TopMover>();
  items.forEach((item) => {
    const current = map.get(item.code);
    if (!current || Math.abs(item.change_percent) > Math.abs(current.change_percent)) {
      map.set(item.code, item);
    }
  });
  return Array.from(map.values());
}

function pickCoreIndices(overview: MarketOverview | null) {
  const indices = overview?.indices ?? [];
  const ordered = INDEX_PRIORITY.map((keyword) =>
    indices.find((item) => item.name.includes(keyword)),
  ).filter(Boolean);

  const existingCodes = new Set(ordered.map((item) => item!.code));
  const fallback = indices.filter((item) => !existingCodes.has(item.code)).slice(0, 3 - ordered.length);
  return [...ordered, ...fallback];
}

function GlassCard({
  children,
  className = '',
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={`glass rounded-2xl p-5 ${className}`}>{children}</div>;
}

function SectionHeader({
  icon: Icon,
  title,
  subtitle,
  action,
}: {
  icon: React.ElementType;
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-3 mb-4">
      <div className="flex items-start gap-3">
        <div
          className="w-10 h-10 rounded-xl flex items-center justify-center"
          style={{ background: 'rgba(59,130,246,0.12)', color: 'var(--color-accent)' }}
        >
          <Icon className="w-4.5 h-4.5" />
        </div>
        <div>
          <h2 className="text-sm font-semibold">{title}</h2>
          {subtitle && (
            <p className="text-xs mt-1" style={{ color: 'var(--color-text-muted)' }}>
              {subtitle}
            </p>
          )}
        </div>
      </div>
      {action}
    </div>
  );
}

function StatCard({
  title,
  value,
  secondary,
  trend,
}: {
  title: string;
  value: string;
  secondary: string;
  trend?: number;
}) {
  const TrendIcon = trend == null ? Minus : trend > 0 ? ArrowUpRight : trend < 0 ? ArrowDownRight : Minus;
  return (
    <GlassCard className="min-h-[118px]">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-xs mb-3" style={{ color: 'var(--color-text-muted)' }}>
            {title}
          </p>
          <div className="text-2xl font-bold font-mono">{value}</div>
          <p className="text-xs mt-2" style={{ color: trend == null ? 'var(--color-text-secondary)' : changeColor(trend) }}>
            {secondary}
          </p>
        </div>
        <div
          className="w-10 h-10 rounded-xl flex items-center justify-center"
          style={{
            background:
              trend == null
                ? 'rgba(148,163,184,0.12)'
                : trend > 0
                  ? 'rgba(239,68,68,0.12)'
                  : trend < 0
                    ? 'rgba(34,197,94,0.12)'
                    : 'rgba(148,163,184,0.12)',
            color: trend == null ? 'var(--color-text-secondary)' : changeColor(trend),
          }}
        >
          <TrendIcon className="w-4.5 h-4.5" />
        </div>
      </div>
    </GlassCard>
  );
}

function RankingTable({
  title,
  icon: Icon,
  rows,
  onSelect,
  amountMode = false,
}: {
  title: string;
  icon: React.ElementType;
  rows: TopMover[];
  onSelect: (item: TopMover) => void;
  amountMode?: boolean;
}) {
  return (
    <GlassCard>
      <SectionHeader icon={Icon} title={title} subtitle={`共 ${rows.length} 条`} />
      {rows.length === 0 ? (
        <EmptyState icon={Icon} title="暂无排行数据" description="等待行情数据同步" />
      ) : (
        <div className="overflow-hidden rounded-xl border" style={{ borderColor: 'var(--color-border)' }}>
          <table className="w-full text-sm">
            <thead style={{ background: 'rgba(148,163,184,0.06)' }}>
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>代码</th>
                <th className="px-4 py-3 text-left text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>名称</th>
                <th className="px-4 py-3 text-right text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>
                  {amountMode ? '成交额' : '涨跌幅'}
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((item) => (
                <tr
                  key={`${title}-${item.code}`}
                  className="cursor-pointer transition-colors hover:bg-white/5"
                  onClick={() => onSelect(item)}
                >
                  <td className="px-4 py-3 font-mono text-xs" style={{ borderTop: '1px solid var(--color-border)', color: 'var(--color-accent)' }}>
                    {item.code}
                  </td>
                  <td className="px-4 py-3 text-sm font-medium" style={{ borderTop: '1px solid var(--color-border)' }}>
                    {item.name}
                  </td>
                  <td
                    className={`px-4 py-3 text-right font-mono text-sm ${amountMode ? '' : changeTextClass(item.change_percent)}`}
                    style={{ borderTop: '1px solid var(--color-border)', color: amountMode ? 'var(--color-text-primary)' : undefined }}
                  >
                    {amountMode ? formatAmount(item.amount) : `${formatSigned(item.change_percent)}%`}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </GlassCard>
  );
}

export default function MarketPage() {
  const [overview, setOverview] = useState<MarketOverview | null>(null);
  const [movers, setMovers] = useState<TopMover[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [lastUpdated, setLastUpdated] = useState('');

  const [quote, setQuote] = useState<Quote | null>(null);
  const [quoteLoading, setQuoteLoading] = useState(false);
  const [quoteError, setQuoteError] = useState('');
  const [alpha300Open, setAlpha300Open] = useState(false);

  const fetchPageData = useCallback(async () => {
    setLoading(true);
    setError('');

    const [overviewRes, gainersRes, losersRes] = await Promise.allSettled([
      marketApi.overview(),
      marketApi.topMovers('desc', 120),
      marketApi.topMovers('asc', 120),
    ]);

    let nextOverview: MarketOverview | null = null;
    const nextMovers: TopMover[] = [];
    const failures: string[] = [];

    if (overviewRes.status === 'fulfilled') {
      nextOverview = overviewRes.value.data;
      setOverview(nextOverview);
      setLastUpdated(nextOverview.updated_at ?? '');
    } else {
      failures.push('大盘概览');
    }

    if (gainersRes.status === 'fulfilled') {
      nextMovers.push(...gainersRes.value.data);
    } else {
      failures.push('涨幅排行');
    }

    if (losersRes.status === 'fulfilled') {
      nextMovers.push(...losersRes.value.data);
    } else {
      failures.push('跌幅排行');
    }

    const merged = dedupeMovers(nextMovers);
    if (merged.length > 0) {
      setMovers(merged);
    }

    if (!nextOverview && merged.length === 0) {
      setError('行情概览与排行榜均加载失败');
    } else if (failures.length > 0) {
      setError(`部分数据加载失败：${failures.join('、')}`);
    }

    setLoading(false);
  }, []);

  useEffect(() => {
    void fetchPageData();
  }, [fetchPageData]);

  const handleSelect = useCallback(async (suggestion: SearchSuggestion) => {
    setQuoteLoading(true);
    setQuoteError('');
    try {
      const response = await marketApi.quote(suggestion.code);
      setQuote(response.data);
    } catch (err: unknown) {
      const message =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ??
        '查询股票行情失败';
      setQuoteError(message);
      setQuote(null);
    } finally {
      setQuoteLoading(false);
    }
  }, []);

  const coreIndices = useMemo(() => pickCoreIndices(overview), [overview]);
  const breadthTotal =
    (overview?.advance_count ?? 0) +
    (overview?.decline_count ?? 0) +
    (overview?.flat_count ?? 0);

  const topGainers = useMemo(
    () => [...movers].sort((a, b) => b.change_percent - a.change_percent).slice(0, 10),
    [movers],
  );
  const topLosers = useMemo(
    () => [...movers].sort((a, b) => a.change_percent - b.change_percent).slice(0, 10),
    [movers],
  );
  const amountLeaders = useMemo(
    () => [...movers].sort((a, b) => b.amount - a.amount).slice(0, 10),
    [movers],
  );

  const breadthOption = useMemo<any>(() => ({
    tooltip: { trigger: 'item' },
    legend: { bottom: 0, textStyle: { color: '#94a3b8' } },
    series: [
      {
        type: 'pie',
        radius: ['48%', '72%'],
        center: ['50%', '45%'],
        label: { color: '#cbd5e1', formatter: '{b}\n{d}%' },
        data: [
          { name: '上涨', value: overview?.advance_count ?? 0, itemStyle: { color: '#ef4444' } },
          { name: '下跌', value: overview?.decline_count ?? 0, itemStyle: { color: '#22c55e' } },
          { name: '平盘', value: overview?.flat_count ?? 0, itemStyle: { color: '#64748b' } },
        ],
      },
    ],
  }), [overview]);

  const indexOption = useMemo<any>(() => ({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { left: 60, right: 18, top: 16, bottom: 24 },
    xAxis: {
      type: 'category',
      data: coreIndices.map((item) => item?.name ?? '--'),
    },
    yAxis: {
      type: 'value',
      axisLabel: { formatter: '{value}%' },
    },
    series: [
      {
        type: 'bar',
        barWidth: '42%',
        data: coreIndices.map((item) => ({
          value: item?.change_percent ?? 0,
          itemStyle: {
            color:
              (item?.change_percent ?? 0) > 0
                ? '#ef4444'
                : (item?.change_percent ?? 0) < 0
                  ? '#22c55e'
                  : '#64748b',
            borderRadius: (item?.change_percent ?? 0) >= 0 ? [6, 6, 0, 0] : [0, 0, 6, 6],
          },
        })),
        label: {
          show: true,
          position: 'top',
          formatter: (params: { value: number }) => `${formatSigned(params.value)}%`,
          color: '#cbd5e1',
          fontSize: 11,
        },
      },
    ],
  }), [coreIndices]);

  const quotePct = quote?.change_percent ?? 0;
  const QuoteTrend = quotePct > 0 ? TrendingUp : quotePct < 0 ? TrendingDown : Minus;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <BarChart3 className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
            <h1 className="text-xl font-bold">行情中心</h1>
          </div>
          <p className="text-sm mt-1" style={{ color: 'var(--color-text-muted)' }}>
            指数概览、市场宽度、涨跌排行与成交额强度
          </p>
        </div>
        <div className="flex items-center gap-2">
          {lastUpdated && (
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
              更新于 {lastUpdated}
            </span>
          )}
          <button
            type="button"
            onClick={() => void fetchPageData()}
            disabled={loading}
            className="inline-flex items-center gap-2 px-3 py-2 rounded-xl text-sm transition-colors hover:bg-white/5"
            style={{ border: '1px solid var(--color-border)' }}
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </button>
        </div>
      </div>

      <GlassCard>
        <SectionHeader
          icon={Search}
          title="个股即时报价"
          subtitle="搜索代码或名称，快速查看单只股票的价格与日内区间"
        />
        <div className="grid grid-cols-1 xl:grid-cols-[340px_minmax(0,1fr)] gap-5">
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <StockSearch onSelect={handleSelect} className="flex-1" />
              <button
                type="button"
                onClick={() => setAlpha300Open(true)}
                className="px-3 py-2 rounded-xl text-sm transition-colors hover:bg-white/5"
                style={{ border: '1px solid var(--color-border)' }}
                title="从 Alpha300 选择"
                aria-label="从 Alpha300 选择"
              >
                🎯
              </button>
            </div>

            {quoteLoading && (
              <div className="rounded-xl px-4 py-3 text-sm flex items-center gap-2" style={{ background: 'rgba(59,130,246,0.08)', color: 'var(--color-accent)' }}>
                <Loader2 className="w-4 h-4 animate-spin" />
                正在查询最新行情
              </div>
            )}

            {quoteError && (
              <div className="rounded-xl px-4 py-3 text-sm" style={{ background: 'rgba(239,68,68,0.08)', color: '#f87171' }}>
                {quoteError}
              </div>
            )}

            {!quote && !quoteLoading && !quoteError && (
              <EmptyState
                icon={Search}
                title="搜索一只股票"
                description="支持代码、简称和 Alpha300 快速选择"
              />
            )}
          </div>

          <div>
            {quote ? (
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3">
                <GlassCard className="md:col-span-2 xl:col-span-1">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="text-sm font-semibold">{quote.name}</div>
                      <div className="text-xs mt-1 font-mono" style={{ color: 'var(--color-text-muted)' }}>
                        {quote.code}
                      </div>
                    </div>
                    <div
                      className="w-10 h-10 rounded-xl flex items-center justify-center"
                      style={{
                        background: quotePct > 0 ? 'rgba(239,68,68,0.12)' : quotePct < 0 ? 'rgba(34,197,94,0.12)' : 'rgba(148,163,184,0.12)',
                        color: changeColor(quotePct),
                      }}
                    >
                      <QuoteTrend className="w-4.5 h-4.5" />
                    </div>
                  </div>
                  <div className="text-3xl font-bold font-mono mt-4" style={{ color: changeColor(quotePct) }}>
                    {quote.price.toFixed(2)}
                  </div>
                  <div className="text-sm font-mono mt-2" style={{ color: changeColor(quotePct) }}>
                    {formatSigned(quote.change)} / {formatSigned(quotePct)}%
                  </div>
                </GlassCard>

                {[
                  { label: '日内区间', value: `${quote.low.toFixed(2)} - ${quote.high.toFixed(2)}` },
                  { label: '开盘 / 昨收', value: `${quote.open.toFixed(2)} / ${quote.prev_close.toFixed(2)}` },
                  {
                    label: '振幅',
                    value:
                      quote.prev_close > 0
                        ? `${(((quote.high - quote.low) / quote.prev_close) * 100).toFixed(2)}%`
                        : '--',
                  },
                ].map((item) => (
                  <GlassCard key={item.label}>
                    <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>
                      {item.label}
                    </div>
                    <div className="text-xl font-semibold font-mono mt-4">{item.value}</div>
                    <div className="text-xs mt-2" style={{ color: 'var(--color-text-muted)' }}>
                      更新时间 {quote.updated_at}
                    </div>
                  </GlassCard>
                ))}
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                {Array.from({ length: 3 }).map((_, index) => (
                  <GlassCard key={index}>
                    <Skeleton variant="text" width="40%" height={14} />
                    <Skeleton variant="text" width="65%" height={28} className="mt-4" />
                    <Skeleton variant="text" width="80%" height={12} className="mt-3" />
                  </GlassCard>
                ))}
              </div>
            )}
          </div>
        </div>
      </GlassCard>

      {loading && !overview ? (
        <SkeletonStatCards />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
          {coreIndices.slice(0, 3).map((item) => (
            <StatCard
              key={item?.code ?? item?.name}
              title={item?.name ?? '--'}
              value={item ? item.price.toFixed(2) : '--'}
              secondary={item ? `${formatSigned(item.change)} / ${formatSigned(item.change_percent)}%` : '--'}
              trend={item?.change_percent}
            />
          ))}
          <StatCard
            title="今日涨跌比"
            value={`${overview?.advance_count ?? 0} : ${overview?.decline_count ?? 0}`}
            secondary={`平盘 ${overview?.flat_count ?? 0} 家`}
          />
        </div>
      )}

      {error && !loading && !overview && movers.length === 0 && (
        <ErrorState title="市场数据加载失败" description={error} onRetry={() => void fetchPageData()} />
      )}

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-5">
        <GlassCard>
          <SectionHeader
            icon={Activity}
            title="市场宽度"
            subtitle={breadthTotal > 0 ? `上涨 ${overview?.advance_count ?? 0} / 下跌 ${overview?.decline_count ?? 0} / 平盘 ${overview?.flat_count ?? 0}` : '暂无宽度数据'}
          />
          {breadthTotal > 0 ? (
            <EChart option={breadthOption} height={300} loading={loading} />
          ) : loading ? (
            <div className="h-[300px]">
              <Skeleton variant="rectangular" height="100%" className="rounded-xl" />
            </div>
          ) : (
            <EmptyState icon={Activity} title="暂无宽度数据" description="等待市场概览返回涨跌家数" />
          )}
        </GlassCard>

        <GlassCard>
          <SectionHeader
            icon={CandlestickChart}
            title="指数表现对比"
            subtitle="跟踪核心指数当前涨跌幅"
          />
          {coreIndices.length > 0 ? (
            <EChart option={indexOption} height={300} loading={loading} />
          ) : loading ? (
            <div className="h-[300px]">
              <Skeleton variant="rectangular" height="100%" className="rounded-xl" />
            </div>
          ) : (
            <EmptyState icon={CandlestickChart} title="暂无指数数据" description="市场概览尚未返回指数列表" />
          )}
        </GlassCard>
      </div>

      {error && (overview || movers.length > 0) && (
        <div className="rounded-2xl px-4 py-3 text-sm" style={{ background: 'rgba(245,158,11,0.08)', color: '#fbbf24' }}>
          {error}
        </div>
      )}

      {loading && movers.length === 0 ? (
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-5">
          <GlassCard><SkeletonInlineTable rows={8} columns={3} /></GlassCard>
          <GlassCard><SkeletonInlineTable rows={8} columns={3} /></GlassCard>
          <GlassCard><SkeletonInlineTable rows={8} columns={3} /></GlassCard>
        </div>
      ) : (
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-5">
          <RankingTable
            title="涨幅前十"
            icon={TrendingUp}
            rows={topGainers}
            onSelect={(item) => void handleSelect({ code: item.code, name: item.name })}
          />
          <RankingTable
            title="跌幅前十"
            icon={TrendingDown}
            rows={topLosers}
            onSelect={(item) => void handleSelect({ code: item.code, name: item.name })}
          />
          <RankingTable
            title="成交额前十"
            icon={Wallet}
            rows={amountLeaders}
            amountMode
            onSelect={(item) => void handleSelect({ code: item.code, name: item.name })}
          />
        </div>
      )}

      <Alpha300Selector
        open={alpha300Open}
        onClose={() => setAlpha300Open(false)}
        onSelect={(selectedCode) => {
          void handleSelect({ code: selectedCode, name: selectedCode });
        }}
      />
    </div>
  );
}
