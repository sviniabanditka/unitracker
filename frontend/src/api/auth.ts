import { request } from './client'

export type Role = 'admin' | 'user'

export interface User {
  id: number
  username: string
  role: Role
  created_at: string
  updated_at: string
}

interface UserResponse {
  user: User
}

export const authApi = {
  me: () => request<UserResponse>('GET', '/auth/me').then(r => r.user),
  login: (username: string, password: string) =>
    request<UserResponse>('POST', '/auth/login', { username, password }).then(r => r.user),
  logout: () => request<void>('POST', '/auth/logout'),
  changePassword: (oldPassword: string, newPassword: string) =>
    request<void>('POST', '/auth/change-password', {
      old_password: oldPassword,
      new_password: newPassword,
    }),
}
