import { request } from './client'
import type { Schema } from '@/lib/schema'

export interface Tracker {
  id: number
  profile_id: number
  name: string
  icon?: string | null
  color?: string | null
  description?: string | null
  schema_json: Schema
  is_archived: boolean
  created_at: string
  updated_at: string
}

export interface LibraryTracker extends Tracker {
  profile_name: string
}

export interface TrackerInput {
  name: string
  icon?: string | null
  color?: string | null
  description?: string | null
  schema_json: Schema
}

interface UpdateResponse {
  tracker: Tracker
  warnings: string[]
}

export const trackersApi = {
  list: (profileId: number, includeArchived = false) =>
    request<{ trackers: Tracker[] }>(
      'GET',
      `/profiles/${profileId}/trackers${includeArchived ? '?include_archived=true' : ''}`,
    ).then(r => r.trackers),
  library: () =>
    request<{ trackers: LibraryTracker[] }>('GET', '/trackers/library').then(r => r.trackers),
  get: (id: number) => request<{ tracker: Tracker }>('GET', `/trackers/${id}`).then(r => r.tracker),
  create: (profileId: number, input: TrackerInput) =>
    request<{ tracker: Tracker }>('POST', `/profiles/${profileId}/trackers`, input).then(r => r.tracker),
  update: (id: number, input: Partial<TrackerInput>) =>
    request<UpdateResponse>('PATCH', `/trackers/${id}`, input),
  archive: (id: number, archived: boolean) =>
    request<{ tracker: Tracker }>('POST', `/trackers/${id}/archive`, { is_archived: archived }).then(
      r => r.tracker,
    ),
  remove: (id: number) => request<void>('DELETE', `/trackers/${id}`),
}
