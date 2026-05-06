<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import VChart from 'vue-echarts'
import '@/lib/echarts'
import { useTheme } from '@/lib/useTheme'
import { formatBucketLabel } from '@/lib/chartFormat'
import { formatDurationMs } from '@/lib/format'
import ChartCard from './ChartCard.vue'
import type { Bucket, NumericStatsField } from '@/api/stats'

const props = defineProps<{
  field: NumericStatsField
  buckets: string[]
  bucket: Bucket
  title: string
}>()

const { t, locale } = useI18n()
const theme = useTheme()

type Agg = 'sum' | 'avg' | 'min' | 'max' | 'count'
const agg = ref<Agg>('sum')

const aggOptions: Agg[] = ['sum', 'avg', 'min', 'max', 'count']

const series = computed<number[]>(() => {
  switch (agg.value) {
    case 'sum':
      return props.field.sum
    case 'avg':
      return props.field.avg
    case 'min':
      return props.field.min
    case 'max':
      return props.field.max
    case 'count':
      return props.field.count
  }
})

const empty = computed(() => series.value.every(v => v === 0))

const labels = computed(() => props.buckets.map(b => formatBucketLabel(b, props.bucket, locale.value)))

function fmtValue(v: number): string {
  if (props.field.type === 'duration') return formatDurationMs(v)
  if (agg.value === 'count') return String(v)
  if (props.field.unit) {
    const rounded = Number.isInteger(v) ? v : Number(v.toFixed(2))
    return `${rounded} ${props.field.unit}`
  }
  return Number.isInteger(v) ? String(v) : v.toFixed(2)
}

const option = computed(() => ({
  grid: { left: 48, right: 12, top: 12, bottom: 32 },
  tooltip: {
    trigger: 'axis',
    formatter: (params: Array<{ axisValueLabel: string; value: number }>) => {
      const p = params[0]
      return `${p.axisValueLabel}<br/>${fmtValue(p.value)}`
    },
  },
  xAxis: {
    type: 'category',
    data: labels.value,
    axisLabel: { interval: 'auto', hideOverlap: true },
  },
  yAxis: {
    type: 'value',
    axisLabel: {
      formatter: (v: number) =>
        props.field.type === 'duration' ? formatDurationMs(v) : String(v),
    },
  },
  series: [
    {
      type: agg.value === 'count' ? 'bar' : 'line',
      data: series.value,
      smooth: true,
      symbol: 'circle',
      itemStyle: { color: '#0ea5e9' },
      areaStyle: agg.value === 'count' ? undefined : { opacity: 0.15 },
    },
  ],
}))
</script>

<template>
  <ChartCard
    :title="title"
    :subtitle="field.unit ? field.unit : undefined"
    :empty="empty"
  >
    <template #actions>
      <select
        v-model="agg"
        class="h-7 rounded-md border border-input bg-transparent px-2 text-xs"
        :aria-label="t('charts.aggregation.sum')"
      >
        <option v-for="o in aggOptions" :key="o" :value="o">
          {{ t(`charts.aggregation.${o}`) }}
        </option>
      </select>
    </template>
    <VChart :option="option" :theme="theme" autoresize style="height: 240px" />
  </ChartCard>
</template>
