<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ApiError } from '@/api/client'
import { revisionsApi, type Revision } from '@/api/revisions'
import type { Schema } from '@/lib/schema'
import { summarize } from '@/lib/revisionSummary'
import RevisionRow from './RevisionRow.vue'

const props = defineProps<{
  entryId: number
  schema: Schema
}>()

const emit = defineEmits<{
  (e: 'restored'): void
}>()

const revisions = ref<Revision[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const restoringId = ref<number | null>(null)

const summaries = computed(() =>
  revisions.value.map((_, i) => summarize(revisions.value, i, props.schema)),
)

async function load() {
  loading.value = true
  error.value = null
  try {
    revisions.value = await revisionsApi.list(props.entryId)
  } catch (e) {
    error.value =
      e instanceof ApiError ? e.message : e instanceof Error ? e.message : 'Failed to load history'
  } finally {
    loading.value = false
  }
}

async function onRestore(revisionId: number) {
  if (!confirm('Restore the entry to this version? A new revision will be added to the history.'))
    return
  restoringId.value = revisionId
  try {
    await revisionsApi.restore(props.entryId, revisionId)
    emit('restored')
    await load()
  } catch (e) {
    error.value =
      e instanceof ApiError ? e.message : e instanceof Error ? e.message : 'Restore failed'
  } finally {
    restoringId.value = null
  }
}

onMounted(load)
watch(() => props.entryId, load)

defineExpose({ reload: load })
</script>

<template>
  <div class="rounded-md border bg-card p-4 space-y-3">
    <div class="flex items-center justify-between">
      <h4 class="text-sm font-medium">History</h4>
      <span v-if="revisions.length" class="text-xs text-muted-foreground">{{ revisions.length }} revisions</span>
    </div>
    <p v-if="loading" class="text-sm text-muted-foreground">Loading…</p>
    <p v-else-if="error" class="text-sm text-destructive">{{ error }}</p>
    <div v-else-if="!revisions.length" class="text-sm text-muted-foreground italic">No revisions.</div>
    <div v-else class="space-y-2">
      <RevisionRow
        v-for="(rev, i) in revisions"
        :key="rev.id"
        :revision="rev"
        :headline="summaries[i].headline"
        :changes="summaries[i].changes"
        :schema="schema"
        :is-latest="i === 0"
        :restoring="restoringId === rev.id"
        @restore="onRestore"
      />
    </div>
  </div>
</template>
