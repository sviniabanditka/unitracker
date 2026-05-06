<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown } from 'lucide-vue-next'
import { localizeError } from '@/lib/errorMapping'
import { resolveLabel } from '@/lib/locale'
import type { Locale } from '@/i18n'
import { statsApi, type Bucket, type StatsResponse } from '@/api/stats'
import type { Tracker } from '@/api/trackers'
import type { Schema } from '@/lib/schema'
import EntryCountChart from './EntryCountChart.vue'
import HourHistogramChart from './HourHistogramChart.vue'
import CalendarHeatmap from './CalendarHeatmap.vue'
import NumericFieldChart from './NumericFieldChart.vue'
import BooleanFieldChart from './BooleanFieldChart.vue'
import CategoricalFieldChart from './CategoricalFieldChart.vue'

const props = defineProps<{
  tracker: Tracker
  schema: Schema
}>()

const { t, locale } = useI18n()
const currentLocale = computed<Locale>(() => (locale.value === 'uk' ? 'uk' : 'en'))

type Range = '7d' | '30d' | '90d' | '1y'
const ranges: Range[] = ['7d', '30d', '90d', '1y']
const buckets: Bucket[] = ['day', 'week', 'month']

const expanded = ref(false)
const bucket = ref<Bucket>('day')
const range = ref<Range>('30d')

const loading = ref(false)
const error = ref<string | null>(null)
const data = ref<StatsResponse | null>(null)

function rangeToFromTo(r: Range): { from: string; to: string } {
  const to = new Date()
  const from = new Date(to)
  switch (r) {
    case '7d':
      from.setUTCDate(from.getUTCDate() - 7)
      break
    case '30d':
      from.setUTCDate(from.getUTCDate() - 30)
      break
    case '90d':
      from.setUTCDate(from.getUTCDate() - 90)
      break
    case '1y':
      from.setUTCFullYear(from.getUTCFullYear() - 1)
      break
  }
  return { from: from.toISOString(), to: to.toISOString() }
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const { from, to } = rangeToFromTo(range.value)
    data.value = await statsApi.getTracker(props.tracker.id, {
      from,
      to,
      bucket: bucket.value,
    })
  } catch (e) {
    error.value = localizeError(e, t)
    data.value = null
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.tracker.id, bucket.value, range.value],
  () => {
    if (expanded.value) load()
  },
)

watch(expanded, open => {
  if (open) load()
})

function toggle() {
  expanded.value = !expanded.value
}

function fieldTitle(key: string): string {
  const def = props.schema.fields.find(f => f.key === key)
  if (!def) return key
  return resolveLabel(def.label, currentLocale.value) || def.key
}
</script>

<template>
  <section class="rounded-lg border bg-card">
    <button
      type="button"
      class="w-full flex items-center justify-between gap-3 px-4 py-3 text-left hover:bg-accent/40 rounded-lg"
      :aria-expanded="expanded"
      aria-controls="charts-body"
      @click="toggle"
    >
      <span class="text-base font-semibold">{{ t('charts.title') }}</span>
      <ChevronDown
        class="h-4 w-4 text-muted-foreground transition-transform duration-200"
        :class="expanded ? 'rotate-180' : ''"
        aria-hidden="true"
      />
    </button>

    <div v-if="expanded" id="charts-body" class="border-t px-4 py-4 space-y-4">
      <header class="flex flex-wrap items-end gap-3">
        <label class="flex flex-col text-xs text-muted-foreground gap-1">
          <span>{{ t('charts.bucket.label') }}</span>
          <select
            v-model="bucket"
            class="h-9 rounded-md border border-input bg-transparent px-2 text-sm"
          >
            <option v-for="b in buckets" :key="b" :value="b">
              {{ t(`charts.bucket.${b}`) }}
            </option>
          </select>
        </label>
        <label class="flex flex-col text-xs text-muted-foreground gap-1">
          <span>{{ t('charts.range.label') }}</span>
          <select
            v-model="range"
            class="h-9 rounded-md border border-input bg-transparent px-2 text-sm"
          >
            <option v-for="r in ranges" :key="r" :value="r">
              {{ t(`charts.range.${r}`) }}
            </option>
          </select>
        </label>
      </header>

      <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
      <p v-if="loading && !data" class="text-sm text-muted-foreground">{{ t('common.loading') }}</p>

      <div v-if="data" class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <EntryCountChart
          :buckets="data.buckets"
          :values="data.entry_count"
          :bucket="data.bucket"
        />
        <HourHistogramChart :values="data.hour_histogram" />
        <CalendarHeatmap
          :tracker-id="tracker.id"
          class="md:col-span-2 lg:col-span-3"
        />
        <template v-for="field in data.fields" :key="field.key">
          <NumericFieldChart
            v-if="field.type === 'number' || field.type === 'duration'"
            :field="field"
            :buckets="data.buckets"
            :bucket="data.bucket"
            :title="fieldTitle(field.key)"
          />
          <BooleanFieldChart
            v-else-if="field.type === 'boolean'"
            :field="field"
            :buckets="data.buckets"
            :bucket="data.bucket"
            :title="fieldTitle(field.key)"
          />
          <CategoricalFieldChart
            v-else-if="field.type === 'select' || field.type === 'multiselect'"
            :field="field"
            :buckets="data.buckets"
            :bucket="data.bucket"
            :title="fieldTitle(field.key)"
          />
        </template>
      </div>
    </div>
  </section>
</template>
