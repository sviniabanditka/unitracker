<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/AppLayout.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import RowActions, { type RowAction } from '@/components/RowActions.vue'
import UserProfilesDialog from '@/components/users/UserProfilesDialog.vue'
import type { Role, User } from '@/api/auth'
import { usersApi } from '@/api/users'
import { localizeError } from '@/lib/errorMapping'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()

const auth = useAuthStore()
const { me } = storeToRefs(auth)

const users = ref<User[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

type Mode = 'idle' | 'create' | 'edit' | 'reset'
const mode = ref<Mode>('idle')
const target = ref<User | null>(null)

const form = reactive({
  username: '',
  password: '',
  role: 'user' as Role,
  newPassword: '',
})

const formError = ref<string | null>(null)
const formLoading = ref(false)

const editTitle = computed(() => {
  switch (mode.value) {
    case 'create':
      return t('users.add')
    case 'edit':
      return `${t('common.edit')} ${target.value?.username ?? ''}`
    case 'reset':
      return `${t('users.resetPassword')} — ${target.value?.username ?? ''}`
    default:
      return ''
  }
})

async function load() {
  loading.value = true
  error.value = null
  try {
    users.value = await usersApi.list()
  } catch (e) {
    error.value = localizeError(e, t)
  } finally {
    loading.value = false
  }
}

function startCreate() {
  mode.value = 'create'
  target.value = null
  form.username = ''
  form.password = ''
  form.role = 'user'
  formError.value = null
}

function startEdit(u: User) {
  mode.value = 'edit'
  target.value = u
  form.username = u.username
  form.role = u.role
  formError.value = null
}

function startReset(u: User) {
  mode.value = 'reset'
  target.value = u
  form.newPassword = ''
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
      if (form.password.length < 8) {
        formError.value = t('validation.minLength', { min: 8 })
        return
      }
      await usersApi.create({
        username: form.username.trim(),
        password: form.password,
        role: form.role,
      })
    } else if (mode.value === 'edit' && target.value) {
      const patch: { username?: string; role?: Role } = {}
      if (form.username.trim() !== target.value.username) patch.username = form.username.trim()
      if (form.role !== target.value.role) patch.role = form.role
      if (Object.keys(patch).length === 0) {
        cancel()
        return
      }
      await usersApi.update(target.value.id, patch)
    } else if (mode.value === 'reset' && target.value) {
      if (form.newPassword.length < 8) {
        formError.value = t('validation.minLength', { min: 8 })
        return
      }
      await usersApi.update(target.value.id, { new_password: form.newPassword })
    }
    await load()
    cancel()
  } catch (e) {
    formError.value = localizeError(e, t)
  } finally {
    formLoading.value = false
  }
}

async function remove(u: User) {
  if (!confirm(`${t('users.delete')}: ${u.username}?`)) return
  try {
    await usersApi.remove(u.id)
    await load()
  } catch (e) {
    error.value = localizeError(e, t)
  }
}

const accessOpen = ref(false)
const accessTarget = ref<User | null>(null)

function startAccess(u: User) {
  accessTarget.value = u
  accessOpen.value = true
}

function rowActionsFor(u: User): RowAction[] {
  const out: RowAction[] = [
    { label: t('common.edit'), onClick: () => startEdit(u) },
    { label: t('users.resetPassword'), onClick: () => startReset(u) },
  ]
  if (u.role !== 'admin') {
    out.push({ label: t('users.manageProfiles'), onClick: () => startAccess(u) })
  }
  out.push({
    label: t('common.delete'),
    variant: 'destructive',
    disabled: me.value?.id === u.id,
    onClick: () => remove(u),
  })
  return out
}

onMounted(load)
</script>

<template>
  <AppLayout>
    <div class="container max-w-3xl lg:max-w-5xl px-4 py-8 space-y-6">
      <header class="flex items-center justify-between">
        <h1 class="text-2xl font-semibold">{{ t('users.title') }}</h1>
        <Button @click="startCreate">{{ t('users.add') }}</Button>
      </header>

      <p v-if="error" class="text-sm text-destructive">{{ error }}</p>

      <section v-if="mode !== 'idle'" class="rounded-lg border bg-card p-5 space-y-4">
        <header class="flex items-center justify-between">
          <h2 class="font-medium">{{ editTitle }}</h2>
          <Button variant="ghost" size="sm" @click="cancel">{{ t('common.close') }}</Button>
        </header>

        <form class="space-y-3" @submit.prevent="submit">
          <template v-if="mode === 'create' || mode === 'edit'">
            <div class="space-y-2">
              <Label for="f-username">{{ t('users.username') }}</Label>
              <Input id="f-username" v-model="form.username" required />
            </div>
            <div class="space-y-2">
              <Label for="f-role">{{ t('users.role') }}</Label>
              <select
                id="f-role"
                v-model="form.role"
                class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="user">{{ t('users.roleUser') }}</option>
                <option value="admin">{{ t('users.roleAdmin') }}</option>
              </select>
            </div>
          </template>

          <div v-if="mode === 'create'" class="space-y-2">
            <Label for="f-password">{{ t('auth.password') }}</Label>
            <Input id="f-password" v-model="form.password" type="password" required autocomplete="new-password" />
          </div>

          <div v-if="mode === 'reset'" class="space-y-2">
            <Label for="f-new-password">{{ t('auth.newPassword') }}</Label>
            <Input
              id="f-new-password"
              v-model="form.newPassword"
              type="password"
              required
              autocomplete="new-password"
            />
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
              <TableHead>{{ t('users.username') }}</TableHead>
              <TableHead>{{ t('users.role') }}</TableHead>
              <TableHead>{{ t('backups.createdAt') }}</TableHead>
              <TableHead class="text-right">{{ t('common.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="5" class="text-center text-muted-foreground py-6">{{ t('common.loading') }}</TableCell>
            </TableRow>
            <TableRow v-else-if="users.length === 0">
              <TableCell colspan="5" class="text-center text-muted-foreground py-6">—</TableCell>
            </TableRow>
            <TableRow v-for="u in users" :key="u.id">
              <TableCell>{{ u.id }}</TableCell>
              <TableCell class="font-medium">{{ u.username }}</TableCell>
              <TableCell>
                <Badge :variant="u.role === 'admin' ? 'default' : 'secondary'">{{ u.role }}</Badge>
              </TableCell>
              <TableCell class="text-muted-foreground">{{ new Date(u.created_at).toLocaleString() }}</TableCell>
              <TableCell class="text-right">
                <RowActions :actions="rowActionsFor(u)" />
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>

      <UserProfilesDialog
        v-model:open="accessOpen"
        :user-id="accessTarget?.id ?? null"
        :username="accessTarget?.username ?? ''"
      />
    </div>
  </AppLayout>
</template>
