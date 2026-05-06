<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  title: string
  subtitle?: string
  empty?: boolean
}>()

const { t } = useI18n()
</script>

<template>
  <section class="rounded-lg border bg-card p-4 space-y-3">
    <header class="flex items-baseline justify-between gap-2">
      <h3 class="font-medium">{{ title }}</h3>
      <p v-if="subtitle" class="text-xs text-muted-foreground truncate">{{ subtitle }}</p>
      <slot name="actions" />
    </header>
    <div class="relative">
      <p
        v-if="empty"
        class="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground"
      >
        {{ t('charts.noData') }}
      </p>
      <div :class="empty ? 'opacity-30 pointer-events-none' : ''">
        <slot />
      </div>
    </div>
  </section>
</template>
