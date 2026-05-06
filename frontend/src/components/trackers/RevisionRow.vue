<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import type { Revision } from '@/api/revisions'
import type { Schema } from '@/lib/schema'
import type { ChangeBit } from '@/lib/revisionSummary'

const props = defineProps<{
  revision: Revision
  headline: string
  changes: ChangeBit[]
  schema: Schema
  isLatest: boolean
  restoring?: boolean
}>()

const emit = defineEmits<{
  (e: 'restore', revisionId: number): void
}>()

const time = computed(() => {
  const d = new Date(props.revision.changed_at)
  return Number.isNaN(d.getTime()) ? props.revision.changed_at : d.toLocaleString()
})

const variant = computed(() => {
  switch (props.revision.change_type) {
    case 'create':
      return 'default'
    case 'update':
      return 'secondary'
    case 'delete':
      return 'destructive'
    case 'restore':
      return 'outline'
    default:
      return 'secondary'
  }
})

function onRestore() {
  emit('restore', props.revision.id)
}
</script>

<template>
  <div class="rounded-md border bg-background p-3 space-y-2">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <div class="flex items-center gap-2">
        <Badge :variant="variant">{{ revision.change_type }}</Badge>
        <span class="text-xs text-muted-foreground">{{ time }}</span>
        <span v-if="revision.changed_by != null" class="text-xs text-muted-foreground">
          · user #{{ revision.changed_by }}
        </span>
      </div>
      <Button
        v-if="!isLatest"
        type="button"
        size="sm"
        variant="outline"
        :disabled="restoring"
        @click="onRestore"
      >
        {{ restoring ? 'Restoring…' : 'Restore this version' }}
      </Button>
    </div>
    <div class="text-sm">{{ headline }}</div>
    <ul v-if="changes.length" class="text-xs text-muted-foreground space-y-0.5">
      <li v-for="c in changes" :key="c.key">
        <span class="font-medium">{{ c.label }}:</span>
        <span class="line-through opacity-70 mx-1">{{ c.before }}</span>
        →
        <span class="ml-1">{{ c.after }}</span>
      </li>
    </ul>
  </div>
</template>
