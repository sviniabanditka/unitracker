import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { settingsApi, type SettingsMap } from '@/api/settings'

const DEFAULTS: SettingsMap = {
  backup_interval_hours: '6',
  backup_retention_count: '20',
  app_name: 'Baby Tracker',
  default_locale: 'en',
}

export const useSettingsStore = defineStore('settings', () => {
  const map = ref<SettingsMap>({ ...DEFAULTS })
  const loaded = ref(false)
  const saving = ref(false)
  let inflight: Promise<void> | null = null

  const appName = computed(() => map.value.app_name || DEFAULTS.app_name)
  const defaultLocale = computed(() => map.value.default_locale || DEFAULTS.default_locale)
  const backupIntervalHours = computed(() => Number(map.value.backup_interval_hours) || Number(DEFAULTS.backup_interval_hours))
  const backupRetentionCount = computed(() => Number(map.value.backup_retention_count) || Number(DEFAULTS.backup_retention_count))

  async function fetchAll(force = false) {
    if (loaded.value && !force) return
    if (inflight) return inflight
    inflight = (async () => {
      try {
        map.value = { ...DEFAULTS, ...(await settingsApi.getAll()) }
        loaded.value = true
      } finally {
        inflight = null
      }
    })()
    return inflight
  }

  async function loadPublicInfo() {
    try {
      applyPublicInfo(await settingsApi.publicInfo())
    } catch {
      // best-effort: keep DEFAULTS / previously loaded values
    }
  }

  async function save(changes: SettingsMap) {
    saving.value = true
    try {
      map.value = { ...DEFAULTS, ...(await settingsApi.patch(changes)) }
      loaded.value = true
    } finally {
      saving.value = false
    }
  }

  function applyPublicInfo(info: { app_name: string; default_locale: string }) {
    map.value = { ...map.value, app_name: info.app_name, default_locale: info.default_locale }
  }

  function reset() {
    map.value = { ...DEFAULTS }
    loaded.value = false
  }

  return {
    map,
    loaded,
    saving,
    appName,
    defaultLocale,
    backupIntervalHours,
    backupRetentionCount,
    fetchAll,
    loadPublicInfo,
    save,
    applyPublicInfo,
    reset,
  }
})
