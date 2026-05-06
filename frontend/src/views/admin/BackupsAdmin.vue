<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/AppLayout.vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import RowActions, { type RowAction } from '@/components/RowActions.vue'
import { snapshotsApi, type Snapshot, type SnapshotType } from '@/api/snapshots'
import { localizeError } from '@/lib/errorMapping'

const { t } = useI18n()

const items = ref<Snapshot[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

const creating = ref(false)
const createNote = ref('')
const showCreate = ref(false)

const restoreTarget = ref<Snapshot | null>(null)
const restoreUnderstood = ref(false)
const restoring = ref(false)

const deleteTarget = ref<Snapshot | null>(null)
const deleting = ref(false)

const sortedItems = computed(() => items.value.slice().sort((a, b) => b.created_at.localeCompare(a.created_at)))

async function load() {
  loading.value = true
  error.value = null
  try {
    items.value = await snapshotsApi.list({ limit: 200 })
  } catch (e) {
    error.value = localizeError(e, t)
  } finally {
    loading.value = false
  }
}

async function submitCreate() {
  creating.value = true
  error.value = null
  try {
    const note = createNote.value.trim()
    await snapshotsApi.create(note || undefined)
    createNote.value = ''
    showCreate.value = false
    await load()
  } catch (e) {
    error.value = localizeError(e, t)
  } finally {
    creating.value = false
  }
}

function startRestore(snap: Snapshot) {
  restoreTarget.value = snap
  restoreUnderstood.value = false
}

async function submitRestore() {
  if (!restoreTarget.value) return
  restoring.value = true
  error.value = null
  try {
    await snapshotsApi.restore(restoreTarget.value.id)
    restoreTarget.value = null
    restoreUnderstood.value = false
    await load()
  } catch (e) {
    error.value = localizeError(e, t)
  } finally {
    restoring.value = false
  }
}

function startDelete(snap: Snapshot) {
  deleteTarget.value = snap
}

async function submitDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  error.value = null
  try {
    await snapshotsApi.remove(deleteTarget.value.id)
    deleteTarget.value = null
    await load()
  } catch (e) {
    error.value = localizeError(e, t)
  } finally {
    deleting.value = false
  }
}

function badgeVariant(t: SnapshotType): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (t) {
    case 'manual':
      return 'default'
    case 'auto':
      return 'secondary'
    case 'pre-restore':
      return 'outline'
    default:
      return 'secondary'
  }
}

function formatSize(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`
}

function formatDate(s: string): string {
  const d = new Date(s)
  return Number.isNaN(d.getTime()) ? s : d.toLocaleString()
}

function rowActionsFor(snap: Snapshot): RowAction[] {
  return [
    { label: t('backups.download'), href: snapshotsApi.downloadUrl(snap.id) },
    { label: t('backups.restore'), onClick: () => startRestore(snap) },
    { label: t('common.delete'), variant: 'destructive', onClick: () => startDelete(snap) },
  ]
}

onMounted(load)
</script>

<template>
  <AppLayout>
    <div class="container max-w-5xl px-4 py-8 space-y-6">
      <header class="flex items-center justify-between gap-2 flex-wrap">
        <h1 class="text-2xl font-semibold">{{ t('backups.title') }}</h1>
        <Button v-if="!showCreate" @click="showCreate = true">{{ t('backups.create') }}</Button>
      </header>

      <p v-if="error" class="text-sm text-destructive">{{ error }}</p>

      <section
        v-if="showCreate"
        class="rounded-lg border bg-card p-5 space-y-3"
      >
        <h2 class="text-sm font-medium">{{ t('backups.create') }}</h2>
        <div class="space-y-2">
          <Label for="snap-note">{{ t('backups.note') }}</Label>
          <Textarea
            id="snap-note"
            v-model="createNote"
            :rows="2"
          />
        </div>
        <div class="flex gap-2">
          <Button :disabled="creating" @click="submitCreate">
            {{ creating ? t('backups.creating') : t('backups.create') }}
          </Button>
          <Button
            type="button"
            variant="outline"
            :disabled="creating"
            @click="showCreate = false; createNote = ''"
          >
            {{ t('common.cancel') }}
          </Button>
        </div>
      </section>

      <div class="rounded-lg border bg-card overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('backups.createdAt') }}</TableHead>
              <TableHead>{{ t('backups.type') }}</TableHead>
              <TableHead>—</TableHead>
              <TableHead>{{ t('backups.size') }}</TableHead>
              <TableHead>—</TableHead>
              <TableHead>{{ t('backups.note') }}</TableHead>
              <TableHead class="text-right">{{ t('common.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="7" class="text-center text-muted-foreground py-6">{{ t('common.loading') }}</TableCell>
            </TableRow>
            <TableRow v-else-if="!sortedItems.length">
              <TableCell colspan="7" class="text-center text-muted-foreground py-6">—</TableCell>
            </TableRow>
            <TableRow v-for="snap in sortedItems" :key="snap.id">
              <TableCell class="whitespace-nowrap text-muted-foreground">
                {{ formatDate(snap.created_at) }}
              </TableCell>
              <TableCell>
                <Badge :variant="badgeVariant(snap.type)">{{ snap.type }}</Badge>
              </TableCell>
              <TableCell class="font-mono text-xs">{{ snap.filename }}</TableCell>
              <TableCell class="whitespace-nowrap">{{ formatSize(snap.size_bytes) }}</TableCell>
              <TableCell class="text-muted-foreground">
                {{ snap.created_by != null ? `#${snap.created_by}` : '—' }}
              </TableCell>
              <TableCell class="text-muted-foreground max-w-xs truncate">
                {{ snap.note ?? '' }}
              </TableCell>
              <TableCell class="text-right whitespace-nowrap">
                <RowActions :actions="rowActionsFor(snap)" />
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>

      <div
        v-if="restoreTarget"
        class="fixed inset-0 z-40 flex items-center justify-center bg-background/70 backdrop-blur-sm p-4"
      >
        <div class="rounded-lg border bg-card text-card-foreground shadow-lg p-6 max-w-md w-full space-y-4">
          <h2 class="text-lg font-semibold">{{ t('backups.restore') }}?</h2>
          <p class="text-sm text-muted-foreground font-mono break-all">{{ restoreTarget.filename }}</p>
          <label class="flex items-center gap-2 text-sm">
            <Checkbox v-model="restoreUnderstood" />
            <span>{{ t('common.confirm') }}</span>
          </label>
          <div class="flex justify-end gap-2">
            <Button
              variant="outline"
              :disabled="restoring"
              @click="restoreTarget = null; restoreUnderstood = false"
            >
              {{ t('common.cancel') }}
            </Button>
            <Button
              variant="destructive"
              :disabled="!restoreUnderstood || restoring"
              @click="submitRestore"
            >
              {{ restoring ? t('backups.restoring') : t('backups.restore') }}
            </Button>
          </div>
        </div>
      </div>

      <div
        v-if="deleteTarget"
        class="fixed inset-0 z-40 flex items-center justify-center bg-background/70 backdrop-blur-sm p-4"
      >
        <div class="rounded-lg border bg-card text-card-foreground shadow-lg p-6 max-w-md w-full space-y-4">
          <h2 class="text-lg font-semibold">{{ t('common.delete') }}?</h2>
          <p class="text-sm text-muted-foreground font-mono break-all">{{ deleteTarget.filename }}</p>
          <div class="flex justify-end gap-2">
            <Button variant="outline" :disabled="deleting" @click="deleteTarget = null">{{ t('common.cancel') }}</Button>
            <Button variant="destructive" :disabled="deleting" @click="submitDelete">
              {{ deleting ? t('common.saving') : t('common.delete') }}
            </Button>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>
