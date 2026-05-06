<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  modelValue?: string | null
  placeholder?: string
  disabled?: boolean
  required?: boolean
  rows?: number
  name?: string
  id?: string
  class?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
}>()

const classes = computed(() =>
  cn(
    'flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
    props.class,
  ),
)

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
}
</script>

<template>
  <textarea
    :id="id"
    :name="name"
    :placeholder="placeholder"
    :disabled="disabled"
    :required="required"
    :rows="rows ?? 3"
    :value="modelValue ?? ''"
    :class="classes"
    @input="onInput"
  />
</template>
