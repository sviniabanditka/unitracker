import { request } from './client'

export type SnapshotType = 'auto' | 'manual' | 'pre-restore'

export interface Snapshot {
  id: number
  filename: string
  size_bytes: number
  type: SnapshotType
  note?: string | null
  created_by?: number | null
  created_at: string
}

export interface ListResponse {
  items: Snapshot[]
}

export const snapshotsApi = {
  list: (params: { type?: SnapshotType; limit?: number } = {}) => {
    const qs = new URLSearchParams()
    if (params.type) qs.set('type', params.type)
    if (params.limit) qs.set('limit', String(params.limit))
    const tail = qs.toString()
    return request<ListResponse>('GET', `/admin/snapshots${tail ? `?${tail}` : ''}`).then(r => r.items)
  },
  create: (note?: string) =>
    request<{ snapshot: Snapshot }>('POST', '/admin/snapshots', note ? { note } : {}).then(
      r => r.snapshot,
    ),
  restore: (id: number) =>
    request<{ status: string }>('POST', `/admin/snapshots/${id}/restore`),
  remove: (id: number) => request<void>('DELETE', `/admin/snapshots/${id}`),
  downloadUrl: (id: number) => `/api/admin/snapshots/${id}/download`,
}
