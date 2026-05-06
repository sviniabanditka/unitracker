<script setup lang="ts">
import { computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import FieldEditor from './FieldEditor.vue'
import {
  type FieldDef,
  type Schema,
  defaultFieldByType,
  validateSchemaClient,
} from '@/lib/schema'

const props = defineProps<{
  modelValue: Schema
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: Schema): void
  (e: 'validity-change', errors: string[]): void
}>()

const errors = computed(() => validateSchemaClient(props.modelValue))
watch(errors, v => emit('validity-change', v), { immediate: true })

function emitFields(fields: FieldDef[]) {
  emit('update:modelValue', { ...props.modelValue, fields })
}

function updateField(idx: number, f: FieldDef) {
  const next = props.modelValue.fields.slice()
  next[idx] = f
  emitFields(next)
}

function removeField(idx: number) {
  emitFields(props.modelValue.fields.filter((_, i) => i !== idx))
}

function moveField(idx: number, dir: -1 | 1) {
  const target = idx + dir
  if (target < 0 || target >= props.modelValue.fields.length) return
  const next = props.modelValue.fields.slice()
  const [item] = next.splice(idx, 1)
  next.splice(target, 0, item)
  emitFields(next)
}

function addField() {
  emitFields([...props.modelValue.fields, defaultFieldByType('text')])
}
</script>

<template>
  <div class="space-y-3">
    <FieldEditor
      v-for="(f, idx) in modelValue.fields"
      :key="idx"
      :field="f"
      :index="idx"
      :total="modelValue.fields.length"
      @update="updateField(idx, $event)"
      @remove="removeField(idx)"
      @move="moveField(idx, $event)"
    />
    <Button type="button" variant="outline" @click="addField">+ Add field</Button>
    <ul v-if="errors.length" class="text-sm text-destructive list-disc pl-5">
      <li v-for="(e, i) in errors" :key="i">{{ e }}</li>
    </ul>
  </div>
</template>
