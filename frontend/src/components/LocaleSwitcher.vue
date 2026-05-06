<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { setLocale, isSupportedLocale, SUPPORTED_LOCALES, type Locale } from '@/i18n'

const { locale } = useI18n()

function pick(next: Locale) {
  setLocale(next)
}

function onChange(e: Event) {
  const v = (e.target as HTMLSelectElement).value
  if (isSupportedLocale(v)) pick(v)
}
</script>

<template>
  <label class="inline-flex items-center text-xs text-muted-foreground gap-1">
    <span class="sr-only">{{ $t('locale.label') }}</span>
    <select
      :value="locale"
      class="h-8 rounded-md border border-input bg-transparent px-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
      :aria-label="$t('locale.label')"
      @change="onChange"
    >
      <option v-for="loc in SUPPORTED_LOCALES" :key="loc" :value="loc">
        {{ loc.toUpperCase() }}
      </option>
    </select>
  </label>
</template>
