<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { useI18n } from 'vue-i18n'

type Theme = 'light' | 'dark'
const theme = ref<Theme>('light')
const { t } = useI18n()

function readTheme(): Theme {
  try {
    const v = localStorage.getItem('theme')
    if (v === 'dark' || v === 'light') return v
  } catch {
    // ignore
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function applyTheme(next: Theme) {
  theme.value = next
  if (next === 'dark') document.documentElement.classList.add('dark')
  else document.documentElement.classList.remove('dark')
  try {
    localStorage.setItem('theme', next)
  } catch {
    // ignore
  }
}

onMounted(() => {
  theme.value = readTheme()
})

const icon = computed(() => (theme.value === 'dark' ? '☀️' : '🌙'))
const aria = computed(() => (theme.value === 'dark' ? t('theme.toggleLight') : t('theme.toggleDark')))

function toggle() {
  applyTheme(theme.value === 'dark' ? 'light' : 'dark')
}
</script>

<template>
  <Button
    type="button"
    variant="ghost"
    size="sm"
    :aria-label="aria"
    :title="aria"
    class="px-2"
    @click="toggle"
  >
    <span aria-hidden="true">{{ icon }}</span>
  </Button>
</template>
