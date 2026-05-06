<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/AppLayout.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { fieldErrors as extractFieldErrors, localizeError } from '@/lib/errorMapping'
import { useSettingsStore } from '@/stores/settings'
import type { SettingsMap } from '@/api/settings'

const { t } = useI18n()

const settings = useSettingsStore()

const form = reactive({
  backup_interval_hours: '6',
  backup_retention_count: '20',
  app_name: 'Baby Tracker',
  default_locale: 'en',
})

const initial = reactive<SettingsMap>({ ...form })

const loading = ref(true)
const error = ref<string | null>(null)
const fieldErrors = ref<Record<string, string>>({})
const successAt = ref<number | null>(null)

const isDirty = computed(() =>
  (Object.keys(form) as Array<keyof typeof form>).some(k => form[k] !== initial[k]),
)
const changed = computed<SettingsMap>(() => {
  const out: SettingsMap = {}
  for (const k of Object.keys(form) as Array<keyof typeof form>) {
    if (form[k] !== initial[k]) out[k] = form[k]
  }
  return out
})

function applyMap(m: SettingsMap) {
  form.backup_interval_hours = m.backup_interval_hours ?? '6'
  form.backup_retention_count = m.backup_retention_count ?? '20'
  form.app_name = m.app_name ?? 'Baby Tracker'
  form.default_locale = m.default_locale ?? 'en'
  Object.assign(initial, { ...form })
}

async function load() {
  loading.value = true
  error.value = null
  try {
    await settings.fetchAll(true)
    applyMap(settings.map)
  } catch (e) {
    error.value = localizeError(e, t)
  } finally {
    loading.value = false
  }
}

async function save() {
  error.value = null
  fieldErrors.value = {}
  successAt.value = null
  try {
    await settings.save(changed.value)
    applyMap(settings.map)
    successAt.value = Date.now()
  } catch (e) {
    const fields = extractFieldErrors(e)
    if (fields) fieldErrors.value = fields
    error.value = localizeError(e, t)
  }
}

function discard() {
  applyMap(initial)
  fieldErrors.value = {}
  error.value = null
}

onMounted(load)
</script>

<template>
  <AppLayout>
    <div class="container max-w-3xl px-4 py-8 space-y-6">
      <header class="flex items-center justify-between gap-3 flex-wrap">
        <h1 class="text-2xl font-semibold">{{ t('settings.title') }}</h1>
        <div class="flex items-center gap-2">
          <Button
            v-if="isDirty"
            type="button"
            variant="outline"
            :disabled="settings.saving"
            @click="discard"
          >
            {{ t('settings.discardChanges') }}
          </Button>
          <Button
            type="button"
            :disabled="!isDirty || settings.saving"
            @click="save"
          >
            {{ settings.saving ? t('common.saving') : t('settings.saveChanges') }}
          </Button>
        </div>
      </header>

      <p v-if="loading" class="text-sm text-muted-foreground">{{ t('common.loading') }}</p>
      <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
      <p
        v-if="successAt && !error"
        class="text-sm text-muted-foreground"
      >
        {{ t('common.save') }} ✓
      </p>

      <section v-if="!loading" class="rounded-lg border bg-card p-5 space-y-5">
        <header>
          <h2 class="font-medium">{{ t('settings.sectionBackups') }}</h2>
        </header>

        <div class="space-y-2">
          <Label for="backup-interval">{{ t('settings.backupIntervalHours') }}</Label>
          <Input
            id="backup-interval"
            v-model="form.backup_interval_hours"
            type="number"
            min="0.05"
            step="0.5"
            class="max-w-[12rem]"
          />
          <p
            v-if="fieldErrors.backup_interval_hours"
            class="text-xs text-destructive"
          >{{ fieldErrors.backup_interval_hours }}</p>
          <p v-else class="text-xs text-muted-foreground">{{ t('settings.backupIntervalHoursHelp') }}</p>
        </div>

        <div class="space-y-2">
          <Label for="backup-retention">{{ t('settings.backupRetentionCount') }}</Label>
          <Input
            id="backup-retention"
            v-model="form.backup_retention_count"
            type="number"
            min="1"
            step="1"
            class="max-w-[12rem]"
          />
          <p
            v-if="fieldErrors.backup_retention_count"
            class="text-xs text-destructive"
          >{{ fieldErrors.backup_retention_count }}</p>
          <p v-else class="text-xs text-muted-foreground">{{ t('settings.backupRetentionCountHelp') }}</p>
        </div>
      </section>

      <section v-if="!loading" class="rounded-lg border bg-card p-5 space-y-5">
        <header>
          <h2 class="font-medium">{{ t('settings.sectionGeneral') }}</h2>
        </header>

        <div class="space-y-2">
          <Label for="app-name">{{ t('settings.appName') }}</Label>
          <Input
            id="app-name"
            v-model="form.app_name"
            maxlength="64"
            class="max-w-md"
          />
          <p
            v-if="fieldErrors.app_name"
            class="text-xs text-destructive"
          >{{ fieldErrors.app_name }}</p>
          <p v-else class="text-xs text-muted-foreground">{{ t('settings.appNameHelp') }}</p>
        </div>

        <div class="space-y-2">
          <Label for="default-locale">{{ t('settings.defaultLocale') }}</Label>
          <Select id="default-locale" v-model="form.default_locale" class="max-w-[12rem]">
            <option value="en">{{ t('locale.en') }}</option>
            <option value="uk">{{ t('locale.uk') }}</option>
          </Select>
          <p
            v-if="fieldErrors.default_locale"
            class="text-xs text-destructive"
          >{{ fieldErrors.default_locale }}</p>
          <p v-else class="text-xs text-muted-foreground">{{ t('settings.defaultLocaleHelp') }}</p>
        </div>
      </section>
    </div>
  </AppLayout>
</template>
