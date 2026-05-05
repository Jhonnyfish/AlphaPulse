/**
 * Drop-in replacement for `import ReactECharts from 'echarts-for-react'`.
 * Uses EChartsReactCore (no full echarts import) with our tree-shaken echarts instance.
 */
import EChartsReactCore from 'echarts-for-react/esm/core';
import { echarts } from '@/lib/echarts-setup';

import type { EChartsReactProps } from 'echarts-for-react/esm/types';

export default function ReactECharts(props: EChartsReactProps) {
  return <EChartsReactCore {...props} echarts={echarts} />;
}
