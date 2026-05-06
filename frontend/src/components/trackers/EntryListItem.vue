<script setup lang="ts">
import { computed } from 'vue'
import type { Entry } from '@/api/entries'
import type { Schema } from '@/lib/schema'
import { formatFieldValue, formatOccurredAt } from '@/lib/format'

const props = defineProps<{
  entry: Entry
  schema: Schema
}>()

const subtitleParts = computed(() => {
  const out: string[] = []
  for (const f of props.schema.fields) {
    if (f.isPrimaryTime) continue
    const raw = props.entry.data[f.key]
    if (raw == null || raw === '' || (Array.isArray(raw) && raw.length === 0)) continue
    const formatted = formatFieldValue(f, raw)
    if (!formatted) continue
    out.push(formatted)
    if (out.length >= 2) break
  }
  return out
})
</script>

<template>
  <div class="flex items-center justify-between gap-3 rounded-md border bg-card px-4 py-3 hover:bg-accent/50">
    <div class="min-w-0 flex-1 space-y-0.5">
      <div class="text-sm font-medium">{{ formatOccurredAt(entry.occurred_at) }}</div>
      <div v-if="subtitleParts.length" class="truncate text-xs text-muted-foreground">
        {{ subtitleParts.join(' · ') }}
      </div>
    </div>
    <slot name="actions" />
  </div>
</template>
