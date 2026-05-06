<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import LocaleSwitcher from '@/components/LocaleSwitcher.vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { localizeError } from '@/lib/errorMapping'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { settingsApi } from '@/api/settings'
import { setLocale, isSupportedLocale, readStoredLocale } from '@/i18n'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const settings = useSettingsStore()
const { appName } = storeToRefs(settings)
const { t } = useI18n()

const username = ref('')
const password = ref('')
const error = ref<string | null>(null)
const loading = ref(false)

onMounted(async () => {
  try {
    const info = await settingsApi.publicInfo()
    settings.applyPublicInfo(info)
    if (!readStoredLocale() && isSupportedLocale(info.default_locale)) {
      setLocale(info.default_locale)
    }
  } catch {
    // best-effort; the login page falls back to defaults
  }
})

async function submit() {
  if (!username.value || !password.value) return
  loading.value = true
  error.value = null
  try {
    await auth.login(username.value.trim(), password.value)
    const next = typeof route.query.next === 'string' ? route.query.next : '/'
    router.replace(next || '/')
  } catch (e) {
    error.value = localizeError(e, t)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="min-h-screen flex flex-col bg-background p-4">
    <div class="self-end flex items-center gap-2">
      <LocaleSwitcher />
      <ThemeToggle />
    </div>
    <div class="flex-1 flex items-center justify-center">
      <form
        class="w-full max-w-sm space-y-4 rounded-lg border bg-card p-6 shadow-sm"
        @submit.prevent="submit"
      >
        <div class="space-y-1">
          <h1 class="text-xl font-semibold">{{ t('auth.login') }}</h1>
          <p class="text-sm text-muted-foreground">{{ appName }}</p>
        </div>

        <div class="space-y-2">
          <Label for="username">{{ t('auth.username') }}</Label>
          <Input
            id="username"
            v-model="username"
            autocomplete="username"
            required
            :disabled="loading"
          />
        </div>

        <div class="space-y-2">
          <Label for="password">{{ t('auth.password') }}</Label>
          <Input
            id="password"
            v-model="password"
            type="password"
            autocomplete="current-password"
            required
            :disabled="loading"
          />
        </div>

        <p v-if="error" class="text-sm text-destructive">{{ error }}</p>

        <Button type="submit" :disabled="loading" class="w-full">
          {{ loading ? t('auth.loggingIn') : t('auth.login') }}
        </Button>
      </form>
    </div>
  </main>
</template>
