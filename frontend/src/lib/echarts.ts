// Centralized tree-shaking entry for ECharts. Importing this module ensures
// only the chart types and helpers we actually render are bundled.
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, LineChart, HeatmapChart } from 'echarts/charts'
import {
  CalendarComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  TitleComponent,
  VisualMapComponent,
  DataZoomComponent,
} from 'echarts/components'

use([
  CanvasRenderer,
  BarChart,
  LineChart,
  HeatmapChart,
  CalendarComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  TitleComponent,
  VisualMapComponent,
  DataZoomComponent,
])
