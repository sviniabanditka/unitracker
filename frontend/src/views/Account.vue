<script setup lang="ts">
import { ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/AppLayout.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { localizeError } from '@/lib/errorMapping'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const { me } = storeToRefs(auth)
const { t } = useI18n()

const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const error = ref<string | null>(null)
const success = ref<string | null>(null)
const loading = ref(false)

async function submit() {
  error.value = null
  success.value = null
  if (newPassword.value.length < 8) {
    error.value = t('validation.minLength', { min: 8 })
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    error.value = t('account.passwordMismatch') === 'account.passwordMismatch'
      ? "New passwords don't match"
      : t('account.passwordMismatch')
    return
  }
  loading.value = true
  try {
    await auth.changePassword(oldPassword.value, newPassword.value)
    success.value = t('auth.passwordChanged')
    oldPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (e) {
    error.value = localizeError(e, t)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AppLayout>
    <div class="container max-w-xl px-4 py-8 space-y-6">
      <header class="space-y-2">
        <h1 class="text-2xl font-semibold">{{ t('account.title') }}</h1>
        <p v-if="me" class="text-sm text-muted-foreground flex items-center gap-2">
          <span class="font-medium text-foreground">{{ me.username }}</span>
          <Badge :variant="me.role === 'admin' ? 'default' : 'secondary'">{{ me.role }}</Badge>
        </p>
      </header>

      <section class="rounded-lg border bg-card p-5 space-y-4">
        <h2 class="font-medium">{{ t('auth.changePassword') }}</h2>
        <form class="space-y-3" @submit.prevent="submit">
          <div class="space-y-2">
            <Label for="old">{{ t('auth.currentPassword') }}</Label>
            <Input id="old" v-model="oldPassword" type="password" autocomplete="current-password" required />
          </div>
          <div class="space-y-2">
            <Label for="new">{{ t('auth.newPassword') }}</Label>
            <Input id="new" v-model="newPassword" type="password" autocomplete="new-password" required />
          </div>
          <div class="space-y-2">
            <Label for="confirm">{{ t('auth.newPassword') }}</Label>
            <Input id="confirm" v-model="confirmPassword" type="password" autocomplete="new-password" required />
          </div>
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
          <p v-if="success" class="text-sm text-emerald-600">{{ success }}</p>
          <Button type="submit" :disabled="loading">
            {{ loading ? t('common.saving') : t('common.save') }}
          </Button>
        </form>
      </section>
    </div>
  </AppLayout>
</template>
