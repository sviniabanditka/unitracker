<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { usePwaStore } from '@/stores/pwa'

const pwa = usePwaStore()
const { updateAvailable } = storeToRefs(pwa)
const { t } = useI18n()
</script>

<template>
  <Transition name="fade">
    <div
      v-if="updateAvailable"
      class="fixed bottom-20 md:bottom-4 right-4 z-40 max-w-sm rounded-lg border bg-card text-card-foreground shadow-lg p-3 flex items-center gap-3"
      role="status"
    >
      <span class="text-sm">{{ t('pwa.updateAvailable') }}</span>
      <Button size="sm" @click="pwa.reload()">{{ t('pwa.reload') }}</Button>
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
