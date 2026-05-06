<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/AppLayout.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import RowActions, { type RowAction } from '@/components/RowActions.vue'
import type { Profile } from '@/api/profiles'
import { localizeError } from '@/lib/errorMapping'
import { useProfilesStore } from '@/stores/profiles'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()

const profilesStore = useProfilesStore()
const auth = useAuthStore()
const { list, ready } = storeToRefs(profilesStore)

const loading = ref(false)
const error = ref<string | null>(null)

type Mode = 'idle' | 'create' | 'edit'
const mode = ref<Mode>('idle')
const target = ref<Profile | null>(null)

const form = reactive({
  name: '',
  description: '',
  avatar_url: '',
})
const formError = ref<string | null>(null)
const formLoading = ref(false)

const isAdmin = computed(() => auth.me?.role === 'admin')

const editTitle = computed(() =>
  mode.value === 'create' ? t('profiles.add') : `${t('common.edit')} ${target.value?.name ?? ''}`,
)

async function load() {
  loading.value = true
  error.value = null
  try {
    await profilesStore.refresh()
  } catch (e) {
    error.value = localizeError(e, t)
  } finally {
    loading.value = false
  }
}

function startCreate() {
  mode.value = 'create'
  target.value = null
  form.name = ''
  form.description = ''
  form.avatar_url = ''
  formError.value = null
}

function startEdit(p: Profile) {
  mode.value = 'edit'
  target.value = p
  form.name = p.name
  form.description = p.description ?? ''
  form.avatar_url = p.avatar_url ?? ''
  formError.value = null
}

function cancel() {
  mode.value = 'idle'
  target.value = null
  formError.value = null
}

async function submit() {
  formError.value = null
  formLoading.value = true
  try {
    if (mode.value === 'create') {
      await profilesStore.create({
        name: form.name.trim(),
        description: form.description || null,
        avatar_url: form.avatar_url || null,
      })
    } else if (mode.value === 'edit' && target.value) {
      const patch: Record<string, unknown> = {}
      if (form.name.trim() !== target.value.name) patch.name = form.name.trim()
      if ((form.description || null) !== (target.value.description ?? null)) patch.description = form.description || null
      if ((form.avatar_url || null) !== (target.value.avatar_url ?? null)) patch.avatar_url = form.avatar_url || null
      if (Object.keys(patch).length) await profilesStore.update(target.value.id, patch)
    }
    cancel()
  } catch (e) {
    formError.value = localizeError(e, t)
  } finally {
    formLoading.value = false
  }
}

async function remove(p: Profile) {
  if (!confirm(`${t('profiles.delete')}: ${p.name}?`)) return
  try {
    await profilesStore.remove(p.id)
  } catch (e) {
    error.value = localizeError(e, t)
  }
}

function rowActionsFor(p: Profile): RowAction[] {
  if (!isAdmin.value) return []
  return [
    { label: t('common.edit'), onClick: () => startEdit(p) },
    {
      label: t('common.delete'),
      variant: 'destructive',
      onClick: () => remove(p),
    },
  ]
}

onMounted(async () => {
  if (!ready.value) await profilesStore.ensure()
})
</script>

<template>
  <AppLayout>
    <div class="container max-w-3xl lg:max-w-5xl px-4 py-8 space-y-6">
      <header class="flex items-center justify-between">
        <h1 class="text-2xl font-semibold">{{ t('profiles.title') }}</h1>
        <Button v-if="isAdmin" @click="startCreate">{{ t('profiles.add') }}</Button>
      </header>

      <p v-if="error" class="text-sm text-destructive">{{ error }}</p>

      <section v-if="mode !== 'idle'" class="rounded-lg border bg-card p-5 space-y-4">
        <header class="flex items-center justify-between">
          <h2 class="font-medium">{{ editTitle }}</h2>
          <Button variant="ghost" size="sm" @click="cancel">{{ t('common.close') }}</Button>
        </header>
        <form class="space-y-3" @submit.prevent="submit">
          <div class="space-y-2">
            <Label for="p-name">{{ t('profiles.name') }}</Label>
            <Input id="p-name" v-model="form.name" required />
          </div>
          <div class="space-y-2">
            <Label for="p-desc">{{ t('profiles.descriptionLabel') }}</Label>
            <Textarea id="p-desc" v-model="form.description" :rows="2" />
          </div>
          <div class="space-y-2">
            <Label for="p-avatar">{{ t('profiles.avatarUrl') }}</Label>
            <Input id="p-avatar" v-model="form.avatar_url" placeholder="https://…" />
          </div>
          <p v-if="formError" class="text-sm text-destructive">{{ formError }}</p>
          <div class="flex gap-2">
            <Button type="submit" :disabled="formLoading">
              {{ formLoading ? t('common.saving') : t('common.save') }}
            </Button>
            <Button type="button" variant="outline" :disabled="formLoading" @click="cancel">{{ t('common.cancel') }}</Button>
          </div>
        </form>
      </section>

      <div class="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>{{ t('profiles.name') }}</TableHead>
              <TableHead>{{ t('profiles.descriptionLabel') }}</TableHead>
              <TableHead class="text-right">{{ t('common.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="4" class="text-center text-muted-foreground py-6">{{ t('common.loading') }}</TableCell>
            </TableRow>
            <TableRow v-else-if="list.length === 0">
              <TableCell colspan="4" class="text-center text-muted-foreground py-6">{{ t('profiles.noProfiles') }}</TableCell>
            </TableRow>
            <TableRow v-for="p in list" :key="p.id">
              <TableCell>{{ p.id }}</TableCell>
              <TableCell class="font-medium">{{ p.name }}</TableCell>
              <TableCell class="text-muted-foreground line-clamp-1">{{ p.description ?? '—' }}</TableCell>
              <TableCell class="text-right">
                <RowActions :actions="rowActionsFor(p)" />
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </div>
  </AppLayout>
</template>
