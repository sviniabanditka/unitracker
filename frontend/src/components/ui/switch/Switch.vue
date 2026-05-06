<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  modelValue?: boolean
  disabled?: boolean
  id?: string
  class?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
}>()

const checked = computed(() => Boolean(props.modelValue))

function toggle() {
  if (props.disabled) return
  emit('update:modelValue', !checked.value)
}

const classes = computed(() =>
  cn(
    'peer inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50',
    checked.value ? 'bg-primary' : 'bg-input',
    props.class,
  ),
)
</script>

<template>
  <button
    :id="id"
    type="button"
    role="switch"
    :aria-checked="checked"
    :disabled="disabled"
    :class="classes"
    @click="toggle"
  >
    <span
      class="pointer-events-none block h-5 w-5 rounded-full bg-background shadow-lg ring-0 transition-transform"
      :class="checked ? 'translate-x-5' : 'translate-x-0'"
    />
  </button>
</template>
