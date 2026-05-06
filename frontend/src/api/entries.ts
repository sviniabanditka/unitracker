import { request } from './client'

export interface Entry {
  id: number
  tracker_id: number
  profile_id: number | null
  data: Record<string, unknown>
  occurred_at: string
  created_by: number | null
  is_deleted: boolean
  created_at: string
  updated_at: string
}

export interface EntryListResponse {
  items: Entry[]
  next_cursor: string
}

export interface EntryListParams {
  from?: string
  to?: string
  limit?: number
  cursor?: string
  includeDeleted?: boolean
}

export interface CreateEntryInput {
  data: Record<string, unknown>
}

export interface UpdateEntryInput {
  data?: Record<string, unknown>
}

export interface ValidationFailure {
  error: string
  details: string[]
}

function buildQuery(params: EntryListParams): string {
  const qs = new URLSearchParams()
  if (params.from) qs.set('from', params.from)
  if (params.to) qs.set('to', params.to)
  if (params.limit) qs.set('limit', String(params.limit))
  if (params.cursor) qs.set('cursor', params.cursor)
  if (params.includeDeleted) qs.set('include_deleted', 'true')
  const s = qs.toString()
  return s ? `?${s}` : ''
}

export const entriesApi = {
  list: (trackerId: number, params: EntryListParams = {}) =>
    request<EntryListResponse>('GET', `/trackers/${trackerId}/entries${buildQuery(params)}`),
  get: (id: number) =>
    request<{ entry: Entry }>('GET', `/entries/${id}`).then(r => r.entry),
  create: (trackerId: number, input: CreateEntryInput) =>
    request<{ entry: Entry }>('POST', `/trackers/${trackerId}/entries`, input).then(r => r.entry),
  update: (id: number, input: UpdateEntryInput) =>
    request<{ entry: Entry }>('PATCH', `/entries/${id}`, input).then(r => r.entry),
  remove: (id: number) => request<void>('DELETE', `/entries/${id}`),
}
