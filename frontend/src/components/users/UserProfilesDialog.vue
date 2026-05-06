<script setup lang="ts">
import { ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import TrackerIcon from '@/components/TrackerIcon.vue'
import { userAccessApi } from '@/api/admin/userAccess'
import { useProfilesStore } from '@/stores/profiles'

const props = defineProps<{
  open: boolean
  userId: number | null
  username: string
}>()

const emit = defineEmits<{
  (e: 'update:open', v: boolean): void
  (e: 'saved'): void
}>()

const { t } = useI18n()
const profilesStore = useProfilesStore()
const { list } = storeToRefs(profilesStore)

const checked = ref<Set<number>>(new Set())
const loading = ref(false)
const saving = ref(false)
const error = ref<string | null>(null)

async function load() {
  if (!props.userId) return
  loading.value = true
  error.value = null
  try {
    await profilesStore.ensure()
    const ids = await userAccessApi.list(props.userId)
    checked.value = new Set(ids)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.open, props.userId],
  ([open]) => {
    if (open) void load()
  },
  { immediate: true },
)

function toggle(id: number, on: boolean) {
  const next = new Set(checked.value)
  if (on) next.add(id)
  else next.delete(id)
  checked.value = next
}

async function save() {
  if (!props.userId) return
  saving.value = true
  error.value = null
  try {
    await userAccessApi.replace(props.userId, [...checked.value])
    emit('saved')
    close()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    saving.value = false
  }
}

function close() {
  emit('update:open', false)
}

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
      :aria-label="t('users.manageProfiles')"
      @keydown="onKey"
    >
      <header class="flex items-center justify-between gap-3 px-4 py-3 border-b">
        <h2 class="font-medium">{{ t('users.manageProfilesTitle', { username }) }}</h2>
        <Button variant="ghost" size="sm" class="px-2" @click="close">
          <span aria-hidden="true" class="text-xl leading-none">✕</span>
        </Button>
      </header>
      <div class="flex-1 overflow-y-auto">
        <p v-if="loading" class="px-4 py-6 text-sm text-muted-foreground text-center">
          {{ t('common.loading') }}
        </p>
        <p v-else-if="list.length === 0" class="px-4 py-6 text-sm text-muted-foreground text-center">
          {{ t('profiles.noProfiles') }}
        </p>
        <ul v-else class="divide-y">
          <li v-for="p in list" :key="p.id" class="px-4 py-3 flex items-center gap-3">
            <Checkbox
              :id="`upa-${p.id}`"
              :model-value="checked.has(p.id)"
              @update:model-value="toggle(p.id, $event)"
            />
            <label :for="`upa-${p.id}`" class="flex-1 flex items-center gap-2 cursor-pointer min-w-0">
              <TrackerIcon :name="null" :color="null" :size="20" badge />
              <div class="min-w-0">
                <p class="text-sm font-medium truncate">{{ p.name }}</p>
                <p v-if="p.description" class="text-xs text-muted-foreground truncate">
                  {{ p.description }}
                </p>
              </div>
            </label>
          </li>
        </ul>
      </div>
      <p v-if="error" class="px-4 py-2 text-sm text-destructive">{{ error }}</p>
      <footer class="border-t px-4 py-3 flex justify-end gap-2">
        <Button type="button" variant="outline" :disabled="saving" @click="close">
          {{ t('common.cancel') }}
        </Button>
        <Button type="button" :disabled="saving || loading" @click="save">
          {{ saving ? t('common.saving') : t('common.save') }}
        </Button>
      </footer>
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
