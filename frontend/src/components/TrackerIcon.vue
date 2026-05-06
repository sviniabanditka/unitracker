<script setup lang="ts">
import { computed } from 'vue'
import { trackerIconComponent } from '@/lib/trackerIcons'

const props = defineProps<{
  name?: string | null
  size?: number | string
  color?: string | null
  // when true, draws a subtle rounded square tinted with the tracker's color
  // behind the icon — used for cards/headers; off by default for inline usage.
  badge?: boolean
}>()

const Icon = computed(() => trackerIconComponent(props.name))
const size = computed(() => Number(props.size ?? 18))
const tint = computed(() => props.color ?? undefined)
</script>

<template>
  <span
    v-if="badge"
    class="inline-flex items-center justify-center rounded-md shrink-0"
    :style="{
      width: `${Number(size) * 1.6}px`,
      height: `${Number(size) * 1.6}px`,
      backgroundColor: tint ? `${tint}1f` : 'hsl(var(--muted))',
      color: tint ?? undefined,
    }"
    aria-hidden="true"
  >
    <component
      :is="Icon"
      v-if="Icon"
      :size="size"
      :stroke-width="1.75"
    />
    <span
      v-else
      class="inline-block rounded-full"
      :style="{
        width: `${Number(size) * 0.5}px`,
        height: `${Number(size) * 0.5}px`,
        backgroundColor: tint ?? 'currentColor',
      }"
    />
  </span>
  <component
    v-else-if="Icon"
    :is="Icon"
    :size="size"
    :color="tint"
    :stroke-width="1.75"
    aria-hidden="true"
  />
  <span
    v-else
    class="inline-block rounded-full shrink-0"
    :style="{
      width: `${Number(size) * 0.5}px`,
      height: `${Number(size) * 0.5}px`,
      backgroundColor: tint ?? 'currentColor',
    }"
    aria-hidden="true"
  />
</template>
