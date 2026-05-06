<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import TrackerIcon from '@/components/TrackerIcon.vue'
import type { LibraryTracker } from '@/api/trackers'
import { useTrackersStore } from '@/stores/trackers'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (e: 'update:open', v: boolean): void
  (e: 'select', t: LibraryTracker): void
}>()

const { t } = useI18n()
const store = useTrackersStore()
const { library, libraryReady } = storeToRefs(store)

const search = ref('')
const loading = ref(false)
const error = ref<string | null>(null)

async function load() {
  if (libraryReady.value) return
  loading.value = true
  error.value = null
  try {
    await store.loadLibrary()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

watch(() => props.open, async open => {
  if (open) {
    search.value = ''
    await load()
  }
})

onMounted(() => {
  if (props.open) void load()
})

function close() {
  emit('update:open', false)
}

function pick(t: LibraryTracker) {
  emit('select', t)
  close()
}

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return library.value
  return library.value.filter(item => {
    return (
      item.name.toLowerCase().includes(q) ||
      (item.profile_name ?? '').toLowerCase().includes(q)
    )
  })
})

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') close()
}
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="open"
        class="fixed inset-0 z-40 bg-background/60 backdrop-blur-sm"
        @click="close"
      />
    </Transition>
    <div
      v-if="open"
      class="fixed inset-x-4 top-12 z-50 mx-auto max-w-xl rounded-lg border bg-card shadow-xl flex flex-col max-h-[80vh]"
      role="dialog"
      :aria-label="t('trackers.libraryDialogTitle')"
      @keydown="onKey"
    >
      <header class="flex items-center justify-between gap-3 px-4 py-3 border-b">
        <h2 class="font-medium">{{ t('trackers.libraryDialogTitle') }}</h2>
        <Button variant="ghost" size="sm" class="px-2" @click="close">
          <span aria-hidden="true" class="text-xl leading-none">✕</span>
        </Button>
      </header>
      <div class="px-4 py-3 border-b">
        <Input
          v-model="search"
          autofocus
          :placeholder="t('trackers.searchLibrary')"
          aria-label="search"
        />
      </div>
      <div class="flex-1 overflow-y-auto">
        <p v-if="loading" class="px-4 py-6 text-sm text-muted-foreground text-center">
          {{ t('common.loading') }}
        </p>
        <p v-else-if="error" class="px-4 py-6 text-sm text-destructive text-center">{{ error }}</p>
        <p v-else-if="filtered.length === 0" class="px-4 py-6 text-sm text-muted-foreground text-center">
          {{ t('trackers.noLibraryItems') }}
        </p>
        <ul v-else class="divide-y">
          <li v-for="item in filtered" :key="item.id">
            <button
              type="button"
              class="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-accent/40"
              @click="pick(item)"
            >
              <TrackerIcon :name="item.icon" :color="item.color" :size="20" badge />
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium truncate">{{ item.name }}</p>
                <p class="text-xs text-muted-foreground truncate">{{ item.profile_name }}</p>
              </div>
              <span class="text-xs text-muted-foreground">
                {{ item.schema_json.fields?.length ?? 0 }} {{ t('trackers.fields').toLowerCase() }}
              </span>
            </button>
          </li>
        </ul>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 200ms ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
