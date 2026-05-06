<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Ban } from 'lucide-vue-next'
import { TRACKER_ICONS, TRACKER_ICON_KEYS } from '@/lib/trackerIcons'

const props = defineProps<{
  modelValue?: string | null
  color?: string | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: string | null): void
}>()

const { t } = useI18n()

const selected = computed(() => (props.modelValue ?? '').toLowerCase().trim())

function pick(key: string | null) {
  emit('update:modelValue', key)
}
</script>

<template>
  <div class="space-y-2">
    <div
      class="grid gap-1 grid-cols-8 sm:grid-cols-10 lg:grid-cols-12"
      role="radiogroup"
      :aria-label="t('trackers.icon')"
    >
      <button
        type="button"
        class="aspect-square rounded-md border flex items-center justify-center text-muted-foreground hover:bg-accent"
        :class="selected === '' ? 'border-primary ring-2 ring-primary/30' : 'border-transparent'"
        :aria-checked="selected === ''"
        :aria-label="t('common.none')"
        role="radio"
        @click="pick(null)"
      >
        <Ban class="h-4 w-4" aria-hidden="true" />
      </button>
      <button
        v-for="key in TRACKER_ICON_KEYS"
        :key="key"
        type="button"
        class="aspect-square rounded-md border flex items-center justify-center hover:bg-accent"
        :class="
          selected === key
            ? 'border-primary ring-2 ring-primary/30'
            : 'border-transparent'
        "
        :title="key"
        :aria-checked="selected === key"
        :aria-label="key"
        role="radio"
        :style="selected === key && color ? { color } : undefined"
        @click="pick(key)"
      >
        <component :is="TRACKER_ICONS[key]" :size="18" :stroke-width="1.75" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>
