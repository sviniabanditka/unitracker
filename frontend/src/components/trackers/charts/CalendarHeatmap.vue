<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import VChart from 'vue-echarts'
import '@/lib/echarts'
import { useTheme } from '@/lib/useTheme'
import ChartCard from './ChartCard.vue'
import { statsApi, type StatsResponse } from '@/api/stats'

const props = defineProps<{
  trackerId: number
}>()

const { t, locale } = useI18n()
const theme = useTheme()
const data = ref<StatsResponse | null>(null)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const to = new Date()
    const from = new Date(to)
    from.setUTCFullYear(from.getUTCFullYear() - 1)
    data.value = await statsApi.getTracker(props.trackerId, {
      bucket: 'day',
      from: from.toISOString(),
      to: to.toISOString(),
    })
  } catch {
    data.value = null
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(() => props.trackerId, load)

const empty = computed(() => !data.value || data.value.entry_count.every(v => v === 0))

const series = computed(() => {
  if (!data.value) return []
  const out: Array<[string, number]> = []
  for (let i = 0; i < data.value.buckets.length; i++) {
    const v = data.value.entry_count[i] ?? 0
    if (v > 0) out.push([data.value.buckets[i], v])
  }
  return out
})

const range = computed(() => {
  if (!data.value || !data.value.buckets.length) {
    const now = new Date()
    const yr = now.getUTCFullYear()
    return [`${yr - 1}-01-01`, `${yr}-12-31`]
  }
  return [data.value.buckets[0], data.value.buckets[data.value.buckets.length - 1]]
})

const monthAbbr = computed(() => {
  const lang = locale.value === 'uk' ? 'uk-UA' : 'en-US'
  const fmt = new Intl.DateTimeFormat(lang, { month: 'short' })
  return Array.from({ length: 12 }, (_, m) => fmt.format(new Date(Date.UTC(2020, m, 1))))
})

const dayAbbr = computed(() => {
  const lang = locale.value === 'uk' ? 'uk-UA' : 'en-US'
  const fmt = new Intl.DateTimeFormat(lang, { weekday: 'narrow' })
  // ECharts dayLabel.nameMap order: Sun..Sat
  return Array.from({ length: 7 }, (_, d) => fmt.format(new Date(Date.UTC(2024, 0, 7 + d))))
})

const option = computed(() => {
  const max = data.value ? Math.max(1, ...data.value.entry_count) : 1
  return {
    tooltip: {
      formatter: (p: { value: [string, number] }) => `${p.value[0]} · ${p.value[1]}`,
    },
    visualMap: {
      min: 0,
      max,
      orient: 'horizontal',
      left: 'center',
      bottom: 0,
      inRange: { color: ['#e0f2fe', '#0ea5e9', '#0c4a6e'] },
      text: [String(max), '0'],
    },
    calendar: {
      top: 16,
      left: 30,
      right: 8,
      cellSize: ['auto', 14],
      range: range.value,
      itemStyle: { borderWidth: 1, borderColor: 'transparent' },
      yearLabel: { show: false },
      monthLabel: { nameMap: monthAbbr.value },
      dayLabel: { nameMap: dayAbbr.value, firstDay: 1 },
    },
    series: [
      {
        type: 'heatmap',
        coordinateSystem: 'calendar',
        data: series.value,
      },
    ],
  }
})
</script>

<template>
  <ChartCard :title="t('charts.calendarHeatmap')" :empty="empty">
    <VChart :option="option" :theme="theme" autoresize style="height: 200px" />
  </ChartCard>
</template>
