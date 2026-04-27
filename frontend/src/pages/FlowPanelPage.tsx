import { useState, useEffect, useMemo } from 'react'
import { fundFlowApi, type FundFlowItem } from '@/lib/api'
import { useToast } from '@/lib/toast'
import EChart from '@/components/charts/EChart'
import { Skeleton, SkeletonCard, SkeletonTable } from '@/components/ui/Skeleton'
import { ArrowUpRight, ArrowDownRight, Droplets, Filter, Search } from 'lucide-react'
import Alpha300Selector from '@/components/Alpha300Selector'

const YI = 100_000_000

function formatYi(v: number): string {
  return (v / YI).toFixed(2)
}

function colorClass(v: number): string {
  if (v > 0) return 'text-red-400'
  if (v < 0) return 'text-green-400'
  return 'text-gray-400'
}

function MetricCard({ label, value, icon }: { label: string; value: number; icon: React.ReactNode }) {
  const positive = value > 0
  const negative = value < 0
  return (
    <div className="glass-panel p-4 flex flex-col gap-1">
      <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{label}</span>
      <div className="flex items-center gap-2">
        <span className={`text-xl font-bold ${colorClass(value)}`}>
          {positive ? '+' : ''}{formatYi(value)}亿
        </span>
        {positive && <ArrowUpRight className="w-4 h-4 text-red-400" />}
        {negative && <ArrowDownRight className="w-4 h-4 text-green-400" />}
      </div>
      <span style={{ color: 'var(--color-text-muted)' }} className="text-xs">{icon}</span>
    </div>
  )
}

type Tab = 'market' | 'stock'

export default function FlowPanelPage() {
  const { toast } = useToast()
  const [data, setData] = useState<FundFlowItem[]>([])
  const [loading, setLoading] = useState(true)
  const [tab, setTab] = useState<Tab>('market')
  const [search, setSearch] = useState('')
  const [stockCode, setStockCode] = useState('')
  const [stockData, setStockData] = useState<FundFlowItem[]>([])
  const [stockLoading, setStockLoading] = useState(false)
  const [alpha300Open, setAlpha300Open] = useState(false)

  useEffect(() => {
    fundFlowApi.flow()
      .then(res => setData(res.data))
      .catch(err => toast({ type: 'error', title: '加载失败', message: err.message }))
      .finally(() => setLoading(false))
  }, [])

  const filteredData = useMemo(() => {
    if (!search) return data
    const q = search.toLowerCase()
    return data.filter(d => d.code.toLowerCase().includes(q) || d.name.toLowerCase().includes(q))
  }, [data, search])

  const sortedData = useMemo(
    () => [...filteredData].sort((a, b) => b.main_net_inflow - a.main_net_inflow),
    [filteredData]
  )

  const summary = useMemo(() => {
    const sum = (arr: FundFlowItem[], key: keyof Pick<FundFlowItem, 'main_net_inflow' | 'super_large_net' | 'large_net' | 'medium_net' | 'small_net'>) =>
      arr.reduce((s, d) => s + d[key], 0)
    return {
      main: sum(data, 'main_net_inflow'),
      superLarge: sum(data, 'super_large_net'),
      large: sum(data, 'large_net'),
      smallMedium: sum(data, 'medium_net') + sum(data, 'small_net'),
    }
  }, [data])

  const chartOption = useMemo(() => {
    const top20 = [...data].sort((a, b) => b.main_net_inflow - a.main_net_inflow).slice(0, 20)
    const names = top20.map(d => d.name).reverse()
    const values = top20.map(d => d.main_net_inflow).reverse()
    return {
      tooltip: {
        trigger: 'axis' as const,
        formatter: (params: unknown) => {
          const p = (params as { name: string; value: number }[])[0]
          return `${p.name}<br/>主力净流入: ${formatYi(p.value)}亿`
        },
      },
      grid: { left: 100, right: 30, top: 10, bottom: 20 },
      xAxis: {
        type: 'value' as const,
        axisLabel: {
          formatter: (v: number) => formatYi(v),
        },
      },
      yAxis: {
        type: 'category' as const,
        data: names,
      },
      series: [{
        type: 'bar' as const,
        data: values.map(v => ({
          value: v,
          itemStyle: { color: v >= 0 ? '#ef4444' : '#22c55e' },
        })),
      }],
    }
  }, [data])

  function handleStockSearch() {
    if (!stockCode.trim()) return
    setStockLoading(true)
    fundFlowApi.flow({ code: stockCode.trim() })
      .then(res => setStockData(res.data))
      .catch(err => toast({ type: 'error', title: '查询失败', message: err.message }))
      .finally(() => setStockLoading(false))
  }

  const pieOption = useMemo(() => {
    if (stockData.length === 0) return {}
    const s = stockData[0]
    const items = [
      { name: '超大单', value: Math.abs(s.super_large_net) },
      { name: '大单', value: Math.abs(s.large_net) },
      { name: '中小单', value: Math.abs(s.medium_net) + Math.abs(s.small_net) },
    ].filter(d => d.value > 0)
    return {
      tooltip: { trigger: 'item' as const, formatter: '{b}: {c} ({d}%)' },
      legend: { bottom: 0, textStyle: { color: '#94a3b8' } },
      series: [{
        type: 'pie' as const,
        radius: ['40%', '70%'],
        data: items,
        label: { color: '#e2e8f0' },
        itemStyle: {
          color: (_params: { dataIndex: number }) => {
            const colors = ['#ef4444', '#f97316', '#22c55e']
            return colors[_params.dataIndex % colors.length]
          },
        },
      }],
    }
  }, [stockData])

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => <SkeletonCard key={i} />)}
        </div>
        <SkeletonCard />
        <SkeletonTable rows={10} columns={7} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold flex items-center gap-2">
          <Droplets className="w-5 h-5" style={{ color: 'var(--color-accent)' }} />
          资金流向
        </h1>
        <div className="flex gap-2">
          <button
            onClick={() => setTab('market')}
            className={`px-4 py-1.5 rounded-lg text-sm transition-colors ${
              tab === 'market' ? 'bg-[var(--color-accent)] text-white' : 'glass-panel'
            }`}
          >
            市场总览
          </button>
          <button
            onClick={() => setTab('stock')}
            className={`px-4 py-1.5 rounded-lg text-sm transition-colors ${
              tab === 'stock' ? 'bg-[var(--color-accent)] text-white' : 'glass-panel'
            }`}
          >
            个股查询
          </button>
        </div>
      </div>

      {tab === 'market' && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <MetricCard label="主力净流入" value={summary.main} icon={<Droplets className="w-3 h-3 inline" />} />
            <MetricCard label="超大单" value={summary.superLarge} icon={<ArrowUpRight className="w-3 h-3 inline" />} />
            <MetricCard label="大单" value={summary.large} icon={<ArrowUpRight className="w-3 h-3 inline" />} />
            <MetricCard label="中小单" value={summary.smallMedium} icon={<ArrowDownRight className="w-3 h-3 inline" />} />
          </div>

          <div className="glass-panel p-4">
            <h2 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-muted)' }}>
              主力净流入 TOP 20
            </h2>
            <EChart option={chartOption} height={500} />
          </div>

          <div className="glass-panel p-4 space-y-4">
            <div className="flex items-center gap-3">
              <Filter className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
              <div className="relative flex-1 max-w-sm">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
                <input
                  type="text"
                  value={search}
                  onChange={e => setSearch(e.target.value)}
                  placeholder="搜索代码或名称..."
                  className="w-full pl-9 pr-3 py-2 rounded-lg text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] focus:outline-none focus:border-[var(--color-accent)]"
                />
              </div>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b" style={{ borderColor: 'var(--color-border)' }}>
                    <th className="text-left py-2 px-3 font-medium" style={{ color: 'var(--color-text-muted)' }}>代码</th>
                    <th className="text-left py-2 px-3 font-medium" style={{ color: 'var(--color-text-muted)' }}>名称</th>
                    <th className="text-right py-2 px-3 font-medium" style={{ color: 'var(--color-text-muted)' }}>主力净流入</th>
                    <th className="text-right py-2 px-3 font-medium" style={{ color: 'var(--color-text-muted)' }}>超大单</th>
                    <th className="text-right py-2 px-3 font-medium" style={{ color: 'var(--color-text-muted)' }}>大单</th>
                    <th className="text-right py-2 px-3 font-medium" style={{ color: 'var(--color-text-muted)' }}>中小单</th>
                    <th className="text-right py-2 px-3 font-medium" style={{ color: 'var(--color-text-muted)' }}>涨跌幅</th>
                  </tr>
                </thead>
                <tbody>
                  {sortedData.slice(0, 50).map(d => (
                    <tr key={d.code} className="border-b" style={{ borderColor: 'var(--color-border)' }}>
                      <td className="py-2 px-3 font-mono">{d.code}</td>
                      <td className="py-2 px-3">{d.name}</td>
                      <td className={`text-right py-2 px-3 ${colorClass(d.main_net_inflow)}`}>
                        {formatYi(d.main_net_inflow)}
                      </td>
                      <td className={`text-right py-2 px-3 ${colorClass(d.super_large_net)}`}>
                        {formatYi(d.super_large_net)}
                      </td>
                      <td className={`text-right py-2 px-3 ${colorClass(d.large_net)}`}>
                        {formatYi(d.large_net)}
                      </td>
                      <td className={`text-right py-2 px-3 ${colorClass(d.medium_net + d.small_net)}`}>
                        {formatYi(d.medium_net + d.small_net)}
                      </td>
                      <td className={`text-right py-2 px-3 ${colorClass(d.change_pct)}`}>
                        {d.change_pct > 0 ? '+' : ''}{d.change_pct.toFixed(2)}%
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}

      {tab === 'stock' && (
        <>
          <div className="glass-panel p-4 flex items-center gap-3">
            <Search className="w-4 h-4" style={{ color: 'var(--color-text-muted)' }} />
            <input
              type="text"
              value={stockCode}
              onChange={e => setStockCode(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleStockSearch()}
              placeholder="输入股票代码，如 600519"
              className="flex-1 max-w-xs px-3 py-2 rounded-lg text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] focus:outline-none focus:border-[var(--color-accent)]"
            />
            <button
              onClick={handleStockSearch}
              className="px-4 py-2 rounded-lg text-sm bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity"
            >
              查询
            </button>
            <button
              type="button"
              onClick={() => setAlpha300Open(true)}
              className="px-2.5 py-2 rounded-lg text-sm shrink-0 transition-colors hover:bg-[var(--color-bg-hover)]"
              style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', color: 'var(--color-text-secondary)' }}
              title="从 Alpha300 选择"
            >
              🎯
            </button>
          </div>

          {stockLoading && <SkeletonCard />}

          {!stockLoading && stockData.length > 0 && (
            <>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {stockData.map(s => (
                  <div key={s.code} className="glass-panel p-4 space-y-3">
                    <div>
                      <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{s.code}</div>
                      <div className="font-bold">{s.name}</div>
                    </div>
                    <div className="grid grid-cols-2 gap-2 text-sm">
                      <div>
                        <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>主力净流入</div>
                        <div className={colorClass(s.main_net_inflow)}>{formatYi(s.main_net_inflow)}亿</div>
                      </div>
                      <div>
                        <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>涨跌幅</div>
                        <div className={colorClass(s.change_pct)}>{s.change_pct > 0 ? '+' : ''}{s.change_pct.toFixed(2)}%</div>
                      </div>
                      <div>
                        <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>超大单</div>
                        <div className={colorClass(s.super_large_net)}>{formatYi(s.super_large_net)}亿</div>
                      </div>
                      <div>
                        <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>大单</div>
                        <div className={colorClass(s.large_net)}>{formatYi(s.large_net)}亿</div>
                      </div>
                      <div>
                        <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>中单</div>
                        <div className={colorClass(s.medium_net)}>{formatYi(s.medium_net)}亿</div>
                      </div>
                      <div>
                        <div className="text-xs" style={{ color: 'var(--color-text-muted)' }}>小单</div>
                        <div className={colorClass(s.small_net)}>{formatYi(s.small_net)}亿</div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
              <div className="glass-panel p-4">
                <h2 className="text-sm font-medium mb-3" style={{ color: 'var(--color-text-muted)' }}>
                  资金构成
                </h2>
                <EChart option={pieOption} height={350} />
              </div>
            </>
          )}

          {!stockLoading && stockData.length === 0 && stockCode && (
            <div className="glass-panel p-10 text-center" style={{ color: 'var(--color-text-muted)' }}>
              输入股票代码后点击查询
            </div>
          )}
        </>
      )}

      <Alpha300Selector
        open={alpha300Open}
        onClose={() => setAlpha300Open(false)}
        onSelect={(code) => setStockCode(code)}
      />
    </div>
  )
}
