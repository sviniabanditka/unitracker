import { request } from './client'
import type { Entry } from './entries'

export interface DashboardTrackerMeta {
  tracker_id: number
  tracker_name: string
  tracker_icon?: string | null
  tracker_color?: string | null
}

export interface DashboardTodayItem extends DashboardTrackerMeta {
  entry: Entry
}

export interface DashboardLastItem extends DashboardTrackerMeta {
  entry: Entry | null
}

export interface DashboardResponse {
  date: string
  today: DashboardTodayItem[]
  last_per_tracker: DashboardLastItem[]
  top_trackers: number[]
}

export const dashboardApi = {
  get: (params: { profileId: number; date?: string }) => {
    const qs = new URLSearchParams()
    qs.set('profile_id', String(params.profileId))
    if (params.date) qs.set('date', params.date)
    return request<DashboardResponse>('GET', `/dashboard?${qs.toString()}`)
  },
}
