import { request } from './client'

export type Bucket = 'day' | 'week' | 'month'

export interface StatsFieldOption {
  value: string
  label: { en: string; uk?: string }
}

interface BaseField {
  key: string
}

export interface NumericStatsField extends BaseField {
  type: 'number' | 'duration'
  unit?: string
  sum: number[]
  avg: number[]
  min: number[]
  max: number[]
  count: number[]
}

export interface BooleanStatsField extends BaseField {
  type: 'boolean'
  true_count: number[]
  false_count: number[]
}

export interface CategoricalStatsField extends BaseField {
  type: 'select' | 'multiselect'
  options: StatsFieldOption[]
  by_value: Record<string, number[]>
}

export type StatsField = NumericStatsField | BooleanStatsField | CategoricalStatsField

export interface StatsResponse {
  tracker_id: number
  from: string
  to: string
  bucket: Bucket
  buckets: string[]
  entry_count: number[]
  hour_histogram: number[]
  fields: StatsField[]
}

export interface StatsParams {
  from?: string
  to?: string
  bucket?: Bucket
}

function buildQuery(p: StatsParams): string {
  const qs = new URLSearchParams()
  if (p.from) qs.set('from', p.from)
  if (p.to) qs.set('to', p.to)
  if (p.bucket) qs.set('bucket', p.bucket)
  const s = qs.toString()
  return s ? `?${s}` : ''
}

export const statsApi = {
  getTracker: (id: number, params: StatsParams = {}) =>
    request<StatsResponse>('GET', `/trackers/${id}/stats${buildQuery(params)}`),
}
