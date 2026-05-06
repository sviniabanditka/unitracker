import { defineStore } from 'pinia'
import { ref } from 'vue'
import { entriesApi, type CreateEntryInput, type Entry, type UpdateEntryInput } from '@/api/entries'
import { revisionsApi } from '@/api/revisions'

interface Filters {
  from: string | null
  to: string | null
  includeDeleted: boolean
}

interface TrackerState {
  items: Entry[]
  cursor: string
  hasMore: boolean
  loading: boolean
  filters: Filters
}

function defaultState(): TrackerState {
  return {
    items: [],
    cursor: '',
    hasMore: false,
    loading: false,
    filters: { from: null, to: null, includeDeleted: false },
  }
}

function sameFilters(a: Filters, b: Filters) {
  return (
    a.from === b.from &&
    a.to === b.to &&
    a.includeDeleted === b.includeDeleted
  )
}

export const useEntriesStore = defineStore('entries', () => {
  const byTracker = ref<Record<number, TrackerState>>({})

  function ensureBucket(trackerId: number): TrackerState {
    let s = byTracker.value[trackerId]
    if (!s) {
      s = defaultState()
      byTracker.value[trackerId] = s
    }
    return s
  }

  async function loadFirst(trackerId: number, filters: Filters): Promise<void> {
    const s = ensureBucket(trackerId)
    s.filters = { ...filters }
    s.loading = true
    try {
      const res = await entriesApi.list(trackerId, {
        from: filters.from ?? undefined,
        to: filters.to ?? undefined,
        includeDeleted: filters.includeDeleted,
      })
      s.items = res.items
      s.cursor = res.next_cursor
      s.hasMore = Boolean(res.next_cursor)
    } finally {
      s.loading = false
    }
  }

  async function loadMore(trackerId: number): Promise<void> {
    const s = ensureBucket(trackerId)
    if (!s.hasMore || s.loading || !s.cursor) return
    s.loading = true
    try {
      const res = await entriesApi.list(trackerId, {
        from: s.filters.from ?? undefined,
        to: s.filters.to ?? undefined,
        includeDeleted: s.filters.includeDeleted,
        cursor: s.cursor,
      })
      s.items = s.items.concat(res.items)
      s.cursor = res.next_cursor
      s.hasMore = Boolean(res.next_cursor)
    } finally {
      s.loading = false
    }
  }

  async function ensure(trackerId: number, filters: Filters): Promise<void> {
    const s = ensureBucket(trackerId)
    if (s.items.length === 0 || !sameFilters(s.filters, filters)) {
      await loadFirst(trackerId, filters)
    }
  }

  async function create(trackerId: number, input: CreateEntryInput): Promise<Entry> {
    const s = ensureBucket(trackerId)
    const created = await entriesApi.create(trackerId, input)
    insertSorted(s.items, created)
    return created
  }

  async function update(trackerId: number, id: number, input: UpdateEntryInput): Promise<Entry> {
    const s = ensureBucket(trackerId)
    const updated = await entriesApi.update(id, input)
    s.items = s.items.filter(e => e.id !== id)
    insertSorted(s.items, updated)
    return updated
  }

  async function remove(trackerId: number, id: number): Promise<void> {
    const s = ensureBucket(trackerId)
    await entriesApi.remove(id)
    if (s.filters.includeDeleted) {
      // Mark in place — keep visible in deleted-included list.
      const idx = s.items.findIndex(e => e.id === id)
      if (idx >= 0) {
        s.items[idx] = { ...s.items[idx], is_deleted: true }
      }
    } else {
      s.items = s.items.filter(e => e.id !== id)
    }
  }

  async function restoreRevision(trackerId: number, entryId: number, revisionId: number): Promise<Entry> {
    const s = ensureBucket(trackerId)
    const restored = await revisionsApi.restore(entryId, revisionId)
    s.items = s.items.filter(e => e.id !== entryId)
    if (!restored.is_deleted || s.filters.includeDeleted) {
      insertSorted(s.items, restored)
    }
    return restored
  }

  async function restoreDeleted(trackerId: number, entryId: number): Promise<Entry> {
    const s = ensureBucket(trackerId)
    const restored = await revisionsApi.restoreDeleted(entryId)
    s.items = s.items.filter(e => e.id !== entryId)
    insertSorted(s.items, restored)
    return restored
  }

  function reset() {
    byTracker.value = {}
  }

  return {
    byTracker,
    ensure,
    loadMore,
    create,
    update,
    remove,
    restoreRevision,
    restoreDeleted,
    reset,
  }
})

function insertSorted(items: Entry[], e: Entry) {
  // descending by occurred_at, then by id
  let i = 0
  while (i < items.length) {
    const cur = items[i]
    if (cur.occurred_at < e.occurred_at) break
    if (cur.occurred_at === e.occurred_at && cur.id < e.id) break
    i++
  }
  items.splice(i, 0, e)
}
