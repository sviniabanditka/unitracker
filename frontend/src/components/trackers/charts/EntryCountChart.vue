<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import VChart from 'vue-echarts'
import '@/lib/echarts'
import { useTheme } from '@/lib/useTheme'
import { formatBucketLabel } from '@/lib/chartFormat'
import ChartCard from './ChartCard.vue'
import type { Bucket } from '@/api/stats'

const props = defineProps<{
  buckets: string[]
  values: number[]
  bucket: Bucket
}>()

const { t, locale } = useI18n()
const theme = useTheme()

const empty = computed(() => props.values.every(v => v === 0))

const labels = computed(() => props.buckets.map(b => formatBucketLabel(b, props.bucket, locale.value)))

const option = computed(() => ({
  grid: { left: 40, right: 12, top: 12, bottom: 32 },
  tooltip: { trigger: 'axis' },
  xAxis: {
    type: 'category',
    data: labels.value,
    axisLabel: { interval: 'auto', hideOverlap: true },
  },
  yAxis: { type: 'value', minInterval: 1 },
  series: [
    {
      type: 'bar',
      data: props.values,
      itemStyle: { color: '#0ea5e9' },
    },
  ],
}))
</script>

<template>
  <ChartCard :title="t('charts.entryCount', { bucket: t(`charts.bucket.${bucket}`).toLowerCase() })" :empty="empty">
    <VChart :option="option" :theme="theme" autoresize style="height: 240px" />
  </ChartCard>
</template>
