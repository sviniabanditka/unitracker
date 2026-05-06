<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref, watch } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { storeToRefs } from 'pinia'
import AppLayout from '@/components/AppLayout.vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import DynamicEntryForm from '@/components/trackers/DynamicEntryForm.vue'
import EntryListItem from '@/components/trackers/EntryListItem.vue'
import RevisionsPanel from '@/components/trackers/RevisionsPanel.vue'
import RowActions, { type RowAction } from '@/components/RowActions.vue'
import TrackerIcon from '@/components/TrackerIcon.vue'
import { ApiError } from '@/api/client'
import { useI18n } from 'vue-i18n'

const { t: tx } = useI18n()

const ChartsSection = defineAsyncComponent(
  () => import('@/components/trackers/charts/ChartsSection.vue'),
)
import { trackersApi, type Tracker } from '@/api/trackers'
import type { Entry } from '@/api/entries'
import { useAuthStore } from '@/stores/auth'
import { useProfilesStore } from '@/stores/profiles'
import { useRouter } from 'vue-router'
import { useEntriesStore } from '@/stores/entries'
import type { Schema } from '@/lib/schema'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const profilesStore = useProfilesStore()
const entriesStore = useEntriesStore()
const { activeId } = storeToRefs(profilesStore)

const tracker = ref<Tracker | null>(null)
const loading = ref(true)
const trackerError = ref<string | null>(null)

const fromDate = ref('')
const toDate = ref('')
const showDeleted = ref(false)

const showAddForm = ref(false)
const editingId = ref<number | null>(null)
const formData = ref<Record<string, unknown>>({})
const saving = ref(false)
const formError = ref<string | null>(null)
const formDetails = ref<string[]>([])

const historyEntryId = ref<number | null>(null)

const isAdmin = computed(() => auth.me?.role === 'admin')
const trackerId = computed(() => Number(route.params.id))
const schema = computed<Schema>(() => tracker.value?.schema_json ?? { fields: [] })

const bucket = computed(() => entriesStore.byTracker[trackerId.value])
const entries = computed<Entry[]>(() => bucket.value?.items ?? [])
const listLoading = computed(() => Boolean(bucket.value?.loading))
const hasMore = computed(() => Boolean(bucket.value?.hasMore))

function isoFromDateInput(d: string, endOfDay: boolean): string | null {
  if (!d) return null
  const [y, m, day] = d.split('-').map(Number)
  if (!y || !m || !day) return null
  const date = endOfDay ? new Date(y, m - 1, day, 23, 59, 59) : new Date(y, m - 1, day, 0, 0, 0)
  return date.toISOString()
}

const filters = computed(() => ({
  from: isoFromDateInput(fromDate.value, false),
  to: isoFromDateInput(toDate.value, true),
  includeDeleted: showDeleted.value,
}))

async function loadTracker() {
  loading.value = true
  trackerError.value = null
  try {
    tracker.value = await trackersApi.get(trackerId.value)
  } catch (e) {
    trackerError.value =
      e instanceof ApiError ? e.message : e instanceof Error ? e.message : 'Failed to load tracker'
  } finally {
    loading.value = false
  }
}

async function init() {
  await loadTracker()
  if (tracker.value) {
    await profilesStore.ensure()
    await entriesStore.ensure(trackerId.value, filters.value)
  }
}

function primedDefaults(): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  for (const f of schema.value.fields) {
    if (!f.isPrimaryTime) continue
    if (f.type === 'datetime') {
      out[f.key] = new Date().toISOString()
    } else if (f.type === 'date') {
      out[f.key] = new Date().toISOString().slice(0, 10)
    } else if (f.type === 'time') {
      const d = new Date()
      out[f.key] = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
    }
  }
  return out
}

function openAddForm() {
  editingId.value = null
  formData.value = primedDefaults()
  formError.value = null
  formDetails.value = []
  showAddForm.value = true
  historyEntryId.value = null
}

function openEditForm(e: Entry) {
  if (e.is_deleted) return
  editingId.value = e.id
  formData.value = { ...e.data }
  formError.value = null
  formDetails.value = []
  showAddForm.value = false
  historyEntryId.value = null
}

function closeForm() {
  showAddForm.value = false
  editingId.value = null
  formData.value = {}
  formError.value = null
  formDetails.value = []
}

async function submit() {
  saving.value = true
  formError.value = null
  formDetails.value = []
  try {
    if (editingId.value != null) {
      await entriesStore.update(trackerId.value, editingId.value, {
        data: formData.value,
      })
    } else {
      await entriesStore.create(trackerId.value, {
        data: formData.value,
      })
    }
    closeForm()
  } catch (e) {
    if (e instanceof ApiError) {
      formError.value = e.message
      const body = e.body as { details?: string[] } | null
      if (body && Array.isArray(body.details)) formDetails.value = body.details
    } else if (e instanceof Error) {
      formError.value = e.message
    } else {
      formError.value = 'Save failed'
    }
  } finally {
    saving.value = false
  }
}

async function deleteEntry(e: Entry) {
  if (!confirm('Delete this entry?')) return
  try {
    await entriesStore.remove(trackerId.value, e.id)
  } catch (err) {
    formError.value =
      err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Delete failed'
  }
}

async function restoreDeletedEntry(e: Entry) {
  try {
    await entriesStore.restoreDeleted(trackerId.value, e.id)
  } catch (err) {
    formError.value =
      err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Restore failed'
  }
}

function toggleHistory(e: Entry) {
  historyEntryId.value = historyEntryId.value === e.id ? null : e.id
}

function entryActions(e: Entry): RowAction[] {
  const out: RowAction[] = [
    { label: tx('entries.history'), variant: 'ghost', onClick: () => toggleHistory(e) },
  ]
  if (e.is_deleted) {
    out.push({
      label: tx('entries.restore'),
      variant: 'ghost',
      onClick: () => restoreDeletedEntry(e),
    })
  } else {
    out.push({ label: tx('common.edit'), variant: 'ghost', onClick: () => openEditForm(e) })
    out.push({
      label: tx('common.delete'),
      variant: 'destructive',
      onClick: () => deleteEntry(e),
    })
  }
  return out
}

async function onHistoryRestored(entryId: number) {
  // The panel wrote a restore-revision; reload the list so data/occurred_at/is_deleted reflect the new state.
  try {
    const s = entriesStore.byTracker[trackerId.value]
    if (s) s.items = []
    await entriesStore.ensure(trackerId.value, filters.value)
  } catch (err) {
    formError.value =
      err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Reload failed'
  }
  historyEntryId.value = entryId
}

watch(filters, async () => {
  if (!tracker.value) return
  await entriesStore.ensure(trackerId.value, filters.value)
})

// If active profile changes and the current tracker no longer belongs to it,
// bounce back to /trackers (which is profile-scoped).
watch(activeId, pid => {
  if (tracker.value && pid != null && tracker.value.profile_id !== pid) {
    void router.replace('/trackers')
  }
})

watch(() => route.params.id, init)
onMounted(init)
</script>

<template>
  <AppLayout>
    <div class="container max-w-3xl px-4 py-8 space-y-6">
      <header class="flex items-center justify-between gap-2">
        <h1 class="text-2xl font-semibold flex items-center gap-3 min-w-0">
          <TrackerIcon
            v-if="tracker"
            :name="tracker.icon"
            :color="tracker.color"
            :size="28"
            badge
          />
          <span class="truncate">{{ tracker?.name ?? '—' }}</span>
          <Badge v-if="tracker?.is_archived" variant="outline">{{ $t('trackers.archived') }}</Badge>
        </h1>
        <RouterLink v-if="isAdmin && tracker" :to="`/trackers/${tracker.id}/edit`">
          <Button variant="outline" size="sm">{{ $t('common.edit') }}</Button>
        </RouterLink>
      </header>

      <p v-if="loading" class="text-sm text-muted-foreground">{{ $t('common.loading') }}</p>
      <p v-else-if="trackerError" class="text-sm text-destructive">{{ trackerError }}</p>

      <template v-else-if="tracker">
        <p v-if="tracker.description" class="text-sm text-muted-foreground">{{ tracker.description }}</p>

        <section class="rounded-lg border bg-card p-5 space-y-3">
          <div class="flex flex-wrap items-end gap-3">
            <div class="space-y-1">
              <Label for="filter-from">{{ $t('entries.from') }}</Label>
              <Input id="filter-from" type="date" v-model="fromDate" />
            </div>
            <div class="space-y-1">
              <Label for="filter-to">{{ $t('entries.to') }}</Label>
              <Input id="filter-to" type="date" v-model="toDate" />
            </div>
            <Button
              v-if="fromDate || toDate"
              type="button"
              variant="outline"
              size="sm"
              @click="fromDate = ''; toDate = ''"
            >
              {{ $t('common.cancel') }}
            </Button>
            <div class="flex items-center gap-2 ml-2">
              <Switch id="show-deleted" v-model="showDeleted" />
              <Label for="show-deleted" class="cursor-pointer">{{ $t('entries.showDeleted') }}</Label>
            </div>
            <div class="ml-auto">
              <Button
                v-if="!showAddForm && editingId == null && !tracker.is_archived"
                type="button"
                @click="openAddForm"
              >
                + {{ $t('entries.add') }}
              </Button>
            </div>
          </div>

          <div
            v-if="showAddForm || editingId != null"
            class="rounded-md border bg-background p-4 space-y-3"
          >
            <h3 class="text-sm font-medium">{{ editingId != null ? $t('entries.edit') : $t('entries.add') }}</h3>
            <DynamicEntryForm v-model="formData" :schema="schema" />
            <p v-if="formError" class="text-sm text-destructive">{{ formError }}</p>
            <ul v-if="formDetails.length" class="text-sm text-destructive list-disc pl-5">
              <li v-for="(d, i) in formDetails" :key="i">{{ d }}</li>
            </ul>
            <div class="flex gap-2">
              <Button type="button" :disabled="saving" @click="submit">
                {{ saving ? $t('common.saving') : $t('common.save') }}
              </Button>
              <Button type="button" variant="outline" :disabled="saving" @click="closeForm">
                {{ $t('common.cancel') }}
              </Button>
            </div>
          </div>
        </section>

        <ChartsSection
          v-if="tracker"
          :tracker="tracker"
          :schema="schema"
        />

        <section class="space-y-2">
          <h2 class="text-sm font-medium text-muted-foreground">
            {{ $t('entries.add') === 'Add entry' ? 'Entries' : 'Записи' }}
            <span v-if="entries.length" class="text-muted-foreground/70">({{ entries.length }})</span>
          </h2>
          <p v-if="!entries.length && !listLoading" class="text-sm text-muted-foreground italic">
            {{ $t('entries.noEntries') }}
          </p>

          <template v-for="e in entries" :key="e.id">
            <EntryListItem
              :entry="e"
              :schema="schema"
              :class="e.is_deleted ? 'opacity-60' : ''"
            >
              <template #actions>
                <RowActions :actions="entryActions(e)" />
              </template>
            </EntryListItem>
            <RevisionsPanel
              v-if="historyEntryId === e.id"
              :entry-id="e.id"
              :schema="schema"
              @restored="onHistoryRestored(e.id)"
            />
          </template>

          <div v-if="hasMore" class="pt-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              :disabled="listLoading"
              @click="entriesStore.loadMore(trackerId)"
            >
              {{ listLoading ? 'Loading…' : 'Load more' }}
            </Button>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>
