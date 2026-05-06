import { request } from './client'

export interface Profile {
  id: number
  name: string
  avatar_url?: string | null
  description?: string | null
  created_at: string
  updated_at: string
}

export interface ProfileInput {
  name: string
  avatar_url?: string | null
  description?: string | null
}

export const profilesApi = {
  list: () => request<{ profiles: Profile[] }>('GET', '/profiles').then(r => r.profiles),
  create: (input: ProfileInput) =>
    request<{ profile: Profile }>('POST', '/profiles', input).then(r => r.profile),
  update: (id: number, input: Partial<ProfileInput>) =>
    request<{ profile: Profile }>('PATCH', `/profiles/${id}`, input).then(r => r.profile),
  remove: (id: number) => request<void>('DELETE', `/profiles/${id}`),
}
