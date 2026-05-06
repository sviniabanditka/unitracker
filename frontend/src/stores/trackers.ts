import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { trackersApi, type LibraryTracker, type Tracker, type TrackerInput } from '@/api/trackers'
import { useProfilesStore } from '@/stores/profiles'

export const useTrackersStore = defineStore('trackers', () => {
  const byProfile = ref<Record<number, Tracker[]>>({})
  const includeArchived = ref(false)
  const readyProfile = ref<Record<number, boolean>>({})
  const library = ref<LibraryTracker[]>([])
  const libraryReady = ref(false)
  const inflightProfile = new Map<number, Promise<void>>()
  let libraryInflight: Promise<void> | null = null

  const profilesStore = useProfilesStore()

  // The "current" list reflects the active profile.
  const list = computed<Tracker[]>(() => {
    const pid = profilesStore.activeId
    if (pid == null) return []
    return byProfile.value[pid] ?? []
  })

  async function ensure(profileId: number | null = profilesStore.activeId) {
    if (profileId == null) return
    if (readyProfile.value[profileId]) return
    const inflight = inflightProfile.get(profileId)
    if (inflight) return inflight
    const p = (async () => {
      try {
        byProfile.value[profileId] = await trackersApi.list(profileId, includeArchived.value)
        readyProfile.value[profileId] = true
      } finally {
        inflightProfile.delete(profileId)
      }
    })()
    inflightProfile.set(profileId, p)
    return p
  }

  async function refresh(profileId: number | null = profilesStore.activeId) {
    if (profileId == null) return
    byProfile.value[profileId] = await trackersApi.list(profileId, includeArchived.value)
    readyProfile.value[profileId] = true
  }

  async function setIncludeArchived(v: boolean) {
    includeArchived.value = v
    // Invalidate per-profile caches so next access reflects the new flag.
    readyProfile.value = {}
    if (profilesStore.activeId != null) await refresh(profilesStore.activeId)
  }

  async function get(id: number) {
    return trackersApi.get(id)
  }

  async function loadLibrary(force = false) {
    if (libraryReady.value && !force) return
    if (libraryInflight) return libraryInflight
    libraryInflight = (async () => {
      try {
        library.value = await trackersApi.library()
        libraryReady.value = true
      } finally {
        libraryInflight = null
      }
    })()
    return libraryInflight
  }

  function invalidateLibrary() {
    libraryReady.value = false
    library.value = []
  }

  async function create(profileId: number, input: TrackerInput) {
    const t = await trackersApi.create(profileId, input)
    const cur = byProfile.value[profileId] ?? []
    byProfile.value[profileId] = [...cur, t]
    invalidateLibrary()
    return t
  }

  async function update(id: number, input: Partial<TrackerInput>) {
    const res = await trackersApi.update(id, input)
    const pid = res.tracker.profile_id
    const cur = byProfile.value[pid] ?? []
    const idx = cur.findIndex(t => t.id === id)
    if (idx !== -1) {
      const next = [...cur]
      next[idx] = res.tracker
      byProfile.value[pid] = next
    }
    invalidateLibrary()
    return res
  }

  async function archive(id: number, archived: boolean) {
    const t = await trackersApi.archive(id, archived)
    const pid = t.profile_id
    const cur = byProfile.value[pid] ?? []
    if (!includeArchived.value && archived) {
      byProfile.value[pid] = cur.filter(x => x.id !== id)
    } else {
      const idx = cur.findIndex(x => x.id === id)
      if (idx !== -1) {
        const next = [...cur]
        next[idx] = t
        byProfile.value[pid] = next
      }
    }
    invalidateLibrary()
    return t
  }

  async function remove(id: number) {
    await trackersApi.remove(id)
    for (const pid of Object.keys(byProfile.value)) {
      const list = byProfile.value[Number(pid)]
      byProfile.value[Number(pid)] = list.filter(t => t.id !== id)
    }
    invalidateLibrary()
  }

  function reset() {
    byProfile.value = {}
    readyProfile.value = {}
    library.value = []
    libraryReady.value = false
  }

  return {
    list,
    byProfile,
    library,
    libraryReady,
    includeArchived,
    ensure,
    refresh,
    setIncludeArchived,
    get,
    create,
    update,
    archive,
    remove,
    loadLibrary,
    invalidateLibrary,
    reset,
  }
})
