import { request } from './client'
import type { Entry } from './entries'

export type ChangeType = 'create' | 'update' | 'delete' | 'restore'

export interface Revision {
  id: number
  entry_id: number
  data: Record<string, unknown>
  occurred_at: string
  profile_id: number | null
  is_deleted: boolean
  change_type: ChangeType
  changed_by: number | null
  changed_at: string
}

export const revisionsApi = {
  list: (entryId: number) =>
    request<{ revisions: Revision[] }>('GET', `/entries/${entryId}/revisions`).then(r => r.revisions),
  restore: (entryId: number, revisionId: number) =>
    request<{ entry: Entry }>('POST', `/entries/${entryId}/restore/${revisionId}`).then(r => r.entry),
  restoreDeleted: (entryId: number) =>
    request<{ entry: Entry }>('POST', `/entries/${entryId}/restore-deleted`).then(r => r.entry),
}
