<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Label } from '@/components/ui/label'
import FieldRenderer from './FieldRenderer.vue'
import type { Schema } from '@/lib/schema'
import { resolveLabel } from '@/lib/locale'
import type { Locale } from '@/i18n'

const props = defineProps<{
  schema: Schema
  modelValue?: Record<string, unknown>
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: Record<string, unknown>): void
}>()

const { locale } = useI18n()
const currentLocale = computed<Locale>(() => (locale.value === 'uk' ? 'uk' : 'en'))

const data = computed(() => props.modelValue ?? {})
const fields = computed(() => props.schema.fields)

function setField(key: string, value: unknown) {
  emit('update:modelValue', { ...data.value, [key]: value })
}
</script>

<template>
  <div class="space-y-4">
    <p v-if="!fields.length" class="text-sm text-muted-foreground italic">—</p>
    <div v-for="f in fields" :key="f.key" class="space-y-2">
      <Label :for="`f-${f.key}`">
        {{ resolveLabel(f.label, currentLocale) }}
        <span v-if="f.required" class="text-destructive">*</span>
        <span v-if="f.unit" class="ml-1 text-xs text-muted-foreground">({{ f.unit }})</span>
      </Label>
      <FieldRenderer
        :field="f"
        :model-value="data[f.key]"
        :disabled="disabled"
        @update:model-value="setField(f.key, $event)"
      />
    </div>
  </div>
</template>
