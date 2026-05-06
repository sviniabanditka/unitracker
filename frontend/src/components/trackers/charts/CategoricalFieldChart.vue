<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import VChart from 'vue-echarts'
import '@/lib/echarts'
import { useTheme } from '@/lib/useTheme'
import { formatBucketLabel } from '@/lib/chartFormat'
import { resolveLabel } from '@/lib/locale'
import type { Locale } from '@/i18n'
import ChartCard from './ChartCard.vue'
import type { Bucket, CategoricalStatsField } from '@/api/stats'

const props = defineProps<{
  field: CategoricalStatsField
  buckets: string[]
  bucket: Bucket
  title: string
}>()

const { locale } = useI18n()
const theme = useTheme()

const currentLocale = computed<Locale>(() => (locale.value === 'uk' ? 'uk' : 'en'))

const palette = ['#0ea5e9', '#10b981', '#f59e0b', '#ef4444', '#a855f7', '#ec4899', '#14b8a6', '#f97316']

const empty = computed(() =>
  props.field.options.every(opt => (props.field.by_value[opt.value] ?? []).every(v => v === 0)),
)

const labels = computed(() => props.buckets.map(b => formatBucketLabel(b, props.bucket, locale.value)))

const series = computed(() =>
  props.field.options.map((opt, i) => ({
    name: resolveLabel(opt.label, currentLocale.value) || opt.value,
    type: 'bar',
    stack: 'cat',
    data: props.field.by_value[opt.value] ?? [],
    itemStyle: { color: palette[i % palette.length] },
  })),
)

const option = computed(() => ({
  grid: { left: 40, right: 12, top: 32, bottom: 32 },
  tooltip: { trigger: 'axis' },
  legend: { top: 0, type: 'scroll' },
  xAxis: { type: 'category', data: labels.value, axisLabel: { hideOverlap: true } },
  yAxis: { type: 'value', minInterval: 1 },
  series: series.value,
}))
</script>

<template>
  <ChartCard :title="title" :empty="empty">
    <VChart :option="option" :theme="theme" autoresize style="height: 260px" />
  </ChartCard>
</template>
