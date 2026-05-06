import { request } from './client'

export type SettingsMap = Record<string, string>

export interface PublicInfo {
  app_name: string
  default_locale: string
}

export const settingsApi = {
  getAll: () => request<SettingsMap>('GET', '/admin/settings'),
  patch: (changes: SettingsMap) => request<SettingsMap>('PATCH', '/admin/settings', changes),
  publicInfo: () => request<PublicInfo>('GET', '/public/info'),
}
