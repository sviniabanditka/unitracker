<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import VChart from 'vue-echarts'
import '@/lib/echarts'
import { useTheme } from '@/lib/useTheme'
import { formatBucketLabel } from '@/lib/chartFormat'
import ChartCard from './ChartCard.vue'
import type { Bucket, BooleanStatsField } from '@/api/stats'

const props = defineProps<{
  field: BooleanStatsField
  buckets: string[]
  bucket: Bucket
  title: string
}>()

const { t, locale } = useI18n()
const theme = useTheme()

const empty = computed(
  () =>
    props.field.true_count.every(v => v === 0) &&
    props.field.false_count.every(v => v === 0),
)

const labels = computed(() => props.buckets.map(b => formatBucketLabel(b, props.bucket, locale.value)))

const option = computed(() => ({
  grid: { left: 40, right: 12, top: 32, bottom: 32 },
  tooltip: { trigger: 'axis' },
  legend: { top: 0 },
  xAxis: { type: 'category', data: labels.value, axisLabel: { hideOverlap: true } },
  yAxis: { type: 'value', minInterval: 1 },
  series: [
    {
      name: t('charts.boolean.true'),
      type: 'bar',
      stack: 'b',
      data: props.field.true_count,
      itemStyle: { color: '#0ea5e9' },
    },
    {
      name: t('charts.boolean.false'),
      type: 'bar',
      stack: 'b',
      data: props.field.false_count,
      itemStyle: { color: '#94a3b8' },
    },
  ],
}))
</script>

<template>
  <ChartCard :title="title" :empty="empty">
    <VChart :option="option" :theme="theme" autoresize style="height: 240px" />
  </ChartCard>
</template>
