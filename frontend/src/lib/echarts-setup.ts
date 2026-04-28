/**
 * Centralized ECharts setup with tree-shaken imports.
 * Only registers the chart types and components actually used by the app.
 * Import this instead of 'echarts' directly to keep bundle size small.
 */
import * as echarts from 'echarts/core';

// ── Chart types actually used in the app ──
import { LineChart } from 'echarts/charts';
import { BarChart } from 'echarts/charts';
import { PieChart } from 'echarts/charts';
import { ScatterChart } from 'echarts/charts';
import { HeatmapChart } from 'echarts/charts';
import { RadarChart } from 'echarts/charts';
import { TreemapChart } from 'echarts/charts';

// ── Components actually used in the app ──
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent,
  MarkLineComponent,
  VisualMapComponent,
  RadarComponent,
  CalendarComponent,
} from 'echarts/components';

// ── Renderer ──
import { CanvasRenderer } from 'echarts/renderers';

// ── Register everything ──
echarts.use([
  // Charts
  LineChart,
  BarChart,
  PieChart,
  ScatterChart,
  HeatmapChart,
  RadarChart,
  TreemapChart,
  // Components
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent,
  MarkLineComponent,
  VisualMapComponent,
  RadarComponent,
  CalendarComponent,
  // Renderer
  CanvasRenderer,
]);

export default echarts;
export { echarts };

// Re-export types that pages need (type-only, erased at build time)
export type { EChartsOption } from 'echarts';
