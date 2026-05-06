<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import VChart from 'vue-echarts'
import '@/lib/echarts'
import { useTheme } from '@/lib/useTheme'
import { formatHourLabel } from '@/lib/chartFormat'
import ChartCard from './ChartCard.vue'

const props = defineProps<{ values: number[] }>()

const { t } = useI18n()
const theme = useTheme()

const empty = computed(() => props.values.every(v => v === 0))

const labels = computed(() => Array.from({ length: 24 }, (_, h) => formatHourLabel(h)))

const option = computed(() => ({
  grid: { left: 40, right: 12, top: 12, bottom: 32 },
  tooltip: { trigger: 'axis' },
  xAxis: {
    type: 'category',
    data: labels.value,
    axisLabel: { interval: 1, hideOverlap: true },
  },
  yAxis: { type: 'value', minInterval: 1 },
  series: [{ type: 'bar', data: props.values, itemStyle: { color: '#10b981' } }],
}))
</script>

<template>
  <ChartCard :title="t('charts.hourOfDay')" :empty="empty">
    <VChart :option="option" :theme="theme" autoresize style="height: 240px" />
  </ChartCard>
</template>
