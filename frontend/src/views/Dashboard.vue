<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/AppLayout.vue'
import { Button } from '@/components/ui/button'
import TrackerIcon from '@/components/TrackerIcon.vue'
import { dashboardApi, type DashboardResponse } from '@/api/dashboard'
import { localizeError } from '@/lib/errorMapping'
import { useProfilesStore } from '@/stores/profiles'

const { t, locale } = useI18n()
const router = useRouter()

const profiles = useProfilesStore()
const { activeId } = storeToRefs(profiles)

const data = ref<DashboardResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

async function load() {
  if (activeId.value == null) {
    data.value = null
    return
  }
  loading.value = true
  error.value = null
  try {
    data.value = await dashboardApi.get({ profileId: activeId.value })
  } catch (e) {
    error.value = localizeError(e, t)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await profiles.ensure()
  void load()
})
watch(activeId, () => void load())

const dateFmt = computed(
  () =>
    new Intl.DateTimeFormat(locale.value === 'uk' ? 'uk-UA' : 'en-US', {
      hour: '2-digit',
      minute: '2-digit',
    }),
)

function fmtTime(iso: string) {
  try {
    return dateFmt.value.format(new Date(iso))
  } catch {
    return iso
  }
}

function fmtRelative(iso: string) {
  const d = new Date(iso)
  const diffMs = Date.now() - d.getTime()
  const min = Math.round(diffMs / 60000)
  if (min < 1) return t('dashboard.today')
  if (min < 60) return `${min}m`
  const hr = Math.round(min / 60)
  if (hr < 24) return `${hr}h`
  const day = Math.round(hr / 24)
  return `${day}d`
}

function valuePreview(entry: { data: Record<string, unknown> }): string {
  const data = entry.data || {}
  const keys = Object.keys(data)
  for (const k of keys) {
    const v = data[k]
    if (v == null) continue
    if (typeof v === 'string') {
      if (/^\d{4}-\d{2}-\d{2}T/.test(v)) continue
      return v
    }
    if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  }
  return ''
}

const topTrackerCards = computed(() => {
  if (!data.value) return []
  const map = new Map<number, (typeof data.value.last_per_tracker)[number]>()
  for (const item of data.value.last_per_tracker) map.set(item.tracker_id, item)
  return data.value.top_trackers
    .map(id => map.get(id))
    .filter((v): v is NonNullable<typeof v> => Boolean(v))
})

function openTracker(id: number) {
  void router.push(`/trackers/${id}`)
}
</script>

<template>
  <AppLayout>
    <div class="container max-w-3xl lg:max-w-5xl px-4 py-6 space-y-6">
      <header class="flex items-center justify-between gap-3">
        <h1 class="text-2xl font-semibold">{{ t('dashboard.title') }}</h1>
      </header>

      <p v-if="error" class="text-sm text-destructive">{{ error }}</p>

      <section
        v-if="topTrackerCards.length"
        class="rounded-lg border bg-card p-4 space-y-3"
      >
        <h2 class="text-sm font-medium">{{ t('dashboard.quickAdd') }}</h2>
        <div class="flex flex-wrap gap-2">
          <Button
            v-for="card in topTrackerCards"
            :key="card.tracker_id"
            type="button"
            variant="secondary"
            size="sm"
            class="rounded-full inline-flex items-center gap-2"
            @click="openTracker(card.tracker_id)"
          >
            <TrackerIcon :name="card.tracker_icon" :color="card.tracker_color" :size="16" />
            {{ card.tracker_name }}
          </Button>
        </div>
      </section>

      <div class="lg:grid lg:grid-cols-2 lg:gap-6 space-y-6 lg:space-y-0">
      <section class="rounded-lg border bg-card">
        <header class="px-4 py-3 border-b flex items-center justify-between">
          <h2 class="font-medium">{{ t('dashboard.today') }}</h2>
          <span v-if="data" class="text-xs text-muted-foreground">{{ data.date }}</span>
        </header>
        <ul v-if="data && data.today.length" class="divide-y">
          <li
            v-for="item in data.today"
            :key="item.entry.id"
            class="px-4 py-3 flex items-center gap-3 cursor-pointer hover:bg-accent/40"
            @click="openTracker(item.tracker_id)"
          >
            <TrackerIcon
              :name="item.tracker_icon"
              :color="item.tracker_color"
              :size="20"
              badge
            />
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium truncate">{{ item.tracker_name }}</p>
              <p class="text-xs text-muted-foreground truncate">{{ valuePreview(item.entry) }}</p>
            </div>
            <span class="text-xs text-muted-foreground tabular-nums">{{ fmtTime(item.entry.occurred_at) }}</span>
          </li>
        </ul>
        <p v-else-if="!loading" class="px-4 py-6 text-sm text-muted-foreground text-center">
          {{ t('dashboard.noEntriesToday') }}
        </p>
      </section>

      <section v-if="data && data.last_per_tracker.length" class="rounded-lg border bg-card">
        <header class="px-4 py-3 border-b">
          <h2 class="font-medium">{{ t('dashboard.lastPerTracker') }}</h2>
        </header>
        <ul class="divide-y">
          <li
            v-for="item in data.last_per_tracker"
            :key="item.tracker_id"
            class="px-4 py-3 flex items-center gap-3 cursor-pointer hover:bg-accent/40"
            @click="openTracker(item.tracker_id)"
          >
            <TrackerIcon
              :name="item.tracker_icon"
              :color="item.tracker_color"
              :size="20"
              badge
            />
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium truncate">{{ item.tracker_name }}</p>
              <p class="text-xs text-muted-foreground truncate">
                {{ item.entry ? valuePreview(item.entry) : t('dashboard.noEntriesYet') }}
              </p>
            </div>
            <span v-if="item.entry" class="text-xs text-muted-foreground tabular-nums">
              {{ fmtRelative(item.entry.occurred_at) }}
            </span>
          </li>
        </ul>
      </section>
      </div>
    </div>
  </AppLayout>
</template>
