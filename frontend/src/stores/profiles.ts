import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { profilesApi, type Profile, type ProfileInput } from '@/api/profiles'

const ACTIVE_KEY = 'tracker:active-profile'
const LEGACY_ACTIVE_KEY = 'baby-tracker:active-child'

export const useProfilesStore = defineStore('profiles', () => {
  const list = ref<Profile[]>([])
  const ready = ref(false)
  const activeId = ref<number | null>(loadActive())
  let inflight: Promise<void> | null = null

  function loadActive(): number | null {
    // One-shot migration of the pre-Phase-10 key so members don't lose their selection.
    const legacy = localStorage.getItem(LEGACY_ACTIVE_KEY)
    if (legacy != null && localStorage.getItem(ACTIVE_KEY) == null) {
      localStorage.setItem(ACTIVE_KEY, legacy)
    }
    if (legacy != null) localStorage.removeItem(LEGACY_ACTIVE_KEY)

    const raw = localStorage.getItem(ACTIVE_KEY)
    if (!raw) return null
    const n = Number(raw)
    return Number.isFinite(n) ? n : null
  }

  function persistActive() {
    if (activeId.value == null) localStorage.removeItem(ACTIVE_KEY)
    else localStorage.setItem(ACTIVE_KEY, String(activeId.value))
  }

  function reconcileActive() {
    if (list.value.length === 0) {
      activeId.value = null
    } else if (!activeId.value || !list.value.some(p => p.id === activeId.value)) {
      activeId.value = list.value[0].id
    }
    persistActive()
  }

  async function ensure() {
    if (ready.value) return
    if (inflight) return inflight
    inflight = (async () => {
      try {
        list.value = await profilesApi.list()
        reconcileActive()
        ready.value = true
      } finally {
        inflight = null
      }
    })()
    return inflight
  }

  async function refresh() {
    list.value = await profilesApi.list()
    reconcileActive()
  }

  async function create(input: ProfileInput) {
    const created = await profilesApi.create(input)
    list.value.push(created)
    if (activeId.value == null) activeId.value = created.id
    persistActive()
    return created
  }

  async function update(id: number, input: Partial<ProfileInput>) {
    const updated = await profilesApi.update(id, input)
    const idx = list.value.findIndex(p => p.id === id)
    if (idx !== -1) list.value[idx] = updated
    return updated
  }

  async function remove(id: number) {
    await profilesApi.remove(id)
    list.value = list.value.filter(p => p.id !== id)
    reconcileActive()
  }

  function setActive(id: number | null) {
    activeId.value = id
    persistActive()
  }

  const active = computed(() => list.value.find(p => p.id === activeId.value) ?? null)

  function reset() {
    list.value = []
    ready.value = false
    activeId.value = null
    localStorage.removeItem(ACTIVE_KEY)
  }

  return { list, ready, activeId, active, ensure, refresh, create, update, remove, setActive, reset }
})
