import type { EChartsOption } from 'echarts'
import ReactECharts from 'echarts-for-react'
import { Loader2 } from 'lucide-react'

const darkThemeDefaults = {
  backgroundColor: 'transparent' as const,
  textStyle: { color: '#e2e8f0' },
  title: { textStyle: { color: '#e2e8f0' } },
  legend: { textStyle: { color: '#94a3b8' } },
  tooltip: {
    backgroundColor: 'rgba(15,23,42,0.9)',
    borderColor: 'rgba(148,163,184,0.2)',
    textStyle: { color: '#e2e8f0' },
  },
}

const axisStyle = {
  axisLine: { lineStyle: { color: '#334155' } },
  axisTick: { lineStyle: { color: '#334155' } },
  axisLabel: { color: '#94a3b8' },
  splitLine: { lineStyle: { color: 'rgba(51,65,85,0.5)' } },
}

function mergeAxisDefaults(option: EChartsOption): EChartsOption {
  const result = { ...option }
  if (result.xAxis && typeof result.xAxis === 'object') {
    result.xAxis = { ...axisStyle, ...(result.xAxis as object) }
  }
  if (result.yAxis && typeof result.yAxis === 'object') {
    result.yAxis = { ...axisStyle, ...(result.yAxis as object) }
  }
  return result
}

interface EChartProps {
  option: EChartsOption
  height?: string | number
  loading?: boolean
  className?: string
}

export default function EChart({ option, height = 400, loading, className }: EChartProps) {
  const mergedOption = mergeAxisDefaults({ ...darkThemeDefaults, ...option })

  return (
    <div className={`relative ${className ?? ''}`} style={{ height }}>
      {loading && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-[rgba(15,23,42,0.6)] rounded-lg">
          <Loader2 className="h-8 w-8 animate-spin text-accent" />
        </div>
      )}
      <ReactECharts
        option={mergedOption}
        style={{ height: '100%', width: '100%' }}
        opts={{ renderer: 'canvas' }}
        notMerge
      />
    </div>
  )
}
