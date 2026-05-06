<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useMaintenanceStore } from '@/stores/maintenance'

const maintenance = useMaintenanceStore()
const { active } = storeToRefs(maintenance)
const { t } = useI18n()
</script>

<template>
  <Transition name="fade">
    <div
      v-if="active"
      class="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
      role="status"
      aria-live="polite"
    >
      <div class="rounded-lg border bg-card text-card-foreground shadow-lg p-6 max-w-sm text-center space-y-2">
        <h2 class="text-lg font-semibold">{{ t('maintenance.title') }}</h2>
        <p class="text-sm text-muted-foreground">{{ t('maintenance.description') }}</p>
        <div class="pt-2 flex items-center justify-center">
          <span class="inline-block h-2 w-2 rounded-full bg-primary animate-pulse" />
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 200ms ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
