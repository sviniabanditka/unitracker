<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { RouterLink, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/AppLayout.vue'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import RowActions, { type RowAction } from '@/components/RowActions.vue'
import TrackerIcon from '@/components/TrackerIcon.vue'
import { localizeError } from '@/lib/errorMapping'
import type { Tracker } from '@/api/trackers'
import { useTrackersStore } from '@/stores/trackers'
import { useProfilesStore } from '@/stores/profiles'
import { useAuthStore } from '@/stores/auth'

const { t: tx } = useI18n()
const router = useRouter()

const store = useTrackersStore()
const profiles = useProfilesStore()
const auth = useAuthStore()
const { list, includeArchived } = storeToRefs(store)
const { activeId } = storeToRefs(profiles)

const loading = ref(false)
const error = ref<string | null>(null)

const isAdmin = computed(() => auth.me?.role === 'admin')

async function load() {
  loading.value = true
  error.value = null
  try {
    await profiles.ensure()
    if (activeId.value != null) {
      await store.refresh(activeId.value)
    }
  } catch (e) {
    error.value = localizeError(e, tx)
  } finally {
    loading.value = false
  }
}

async function toggleArchived(v: boolean) {
  await store.setIncludeArchived(v)
}

async function archive(id: number, archived: boolean) {
  try {
    await store.archive(id, archived)
  } catch (e) {
    error.value = localizeError(e, tx)
  }
}

async function remove(id: number, name: string) {
  if (!confirm(`${tx('trackers.deleteTracker')}: ${name}?`)) return
  try {
    await store.remove(id)
  } catch (e) {
    error.value = localizeError(e, tx)
  }
}

function rowActionsFor(t: Tracker): RowAction[] {
  return [
    { label: tx('common.edit'), onClick: () => void router.push(`/trackers/${t.id}/edit`) },
    {
      label: t.is_archived ? tx('trackers.unarchive') : tx('trackers.archive'),
      onClick: () => archive(t.id, !t.is_archived),
    },
    {
      label: tx('common.delete'),
      variant: 'destructive',
      onClick: () => remove(t.id, t.name),
    },
  ]
}

onMounted(load)
watch(activeId, () => void load())
</script>

<template>
  <AppLayout>
    <div class="container max-w-4xl lg:max-w-6xl px-4 py-8 space-y-6">
      <header class="flex items-center justify-between gap-4 flex-wrap">
        <h1 class="text-2xl font-semibold">{{ tx('trackers.title') }}</h1>
        <div class="flex items-center gap-4">
          <label class="flex items-center gap-2 text-sm text-muted-foreground">
            <Switch
              :model-value="includeArchived"
              @update:model-value="toggleArchived($event)"
            />
            {{ tx('trackers.archived') }}
          </label>
          <RouterLink v-if="isAdmin && activeId != null" to="/trackers/new">
            <Button>+ {{ tx('trackers.new') }}</Button>
          </RouterLink>
        </div>
      </header>

      <p v-if="error" class="text-sm text-destructive">{{ error }}</p>

      <p v-if="loading" class="text-sm text-muted-foreground">{{ tx('common.loading') }}</p>

      <div
        v-else-if="activeId == null"
        class="rounded-lg border bg-card p-6 text-center space-y-3"
      >
        <p class="text-sm text-muted-foreground">
          {{ isAdmin ? tx('trackers.noActiveProfileAdmin') : tx('profiles.noAccess') }}
        </p>
        <RouterLink v-if="isAdmin" to="/profiles">
          <Button variant="outline">{{ tx('profiles.add') }}</Button>
        </RouterLink>
      </div>

      <p v-else-if="!list.length" class="text-sm text-muted-foreground">
        {{ tx('trackers.noTrackers') }}
        <RouterLink v-if="isAdmin" to="/trackers/new" class="underline">{{ tx('trackers.new') }}</RouterLink>
      </p>

      <ul v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <li v-for="t in list" :key="t.id" class="rounded-lg border bg-card p-4 space-y-2">
          <div class="flex items-start justify-between gap-2">
            <RouterLink
              :to="`/trackers/${t.id}`"
              class="flex items-center gap-2 font-medium hover:underline min-w-0"
            >
              <TrackerIcon :name="t.icon" :color="t.color" :size="22" badge />
              <span class="truncate">{{ t.name }}</span>
            </RouterLink>
            <Badge v-if="t.is_archived" variant="outline">{{ tx('trackers.archived') }}</Badge>
          </div>
          <p v-if="t.description" class="text-sm text-muted-foreground line-clamp-2">{{ t.description }}</p>
          <p class="text-xs text-muted-foreground">
            {{ t.schema_json.fields?.length ?? 0 }} {{ tx('trackers.fields').toLowerCase() }}
          </p>
          <div v-if="isAdmin" class="flex justify-end pt-2">
            <RowActions :actions="rowActionsFor(t)" />
          </div>
        </li>
      </ul>
    </div>
  </AppLayout>
</template>
