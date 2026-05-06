<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  modelValue?: string | number | null
  disabled?: boolean
  required?: boolean
  name?: string
  id?: string
  class?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
}>()

const classes = computed(() =>
  cn(
    'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
    props.class,
  ),
)

function onChange(event: Event) {
  emit('update:modelValue', (event.target as HTMLSelectElement).value)
}
</script>

<template>
  <select
    :id="id"
    :name="name"
    :disabled="disabled"
    :required="required"
    :value="modelValue ?? ''"
    :class="classes"
    @change="onChange"
  >
    <slot />
  </select>
</template>
