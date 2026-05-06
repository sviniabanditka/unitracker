import { request } from './client'
import type { Role, User } from './auth'

export const usersApi = {
  list: () => request<{ users: User[] }>('GET', '/admin/users').then(r => r.users),
  create: (input: { username: string; password: string; role: Role }) =>
    request<{ user: User }>('POST', '/admin/users', input).then(r => r.user),
  update: (id: number, input: { username?: string; role?: Role; new_password?: string }) =>
    request<{ user: User }>('PATCH', `/admin/users/${id}`, input).then(r => r.user),
  remove: (id: number) => request<void>('DELETE', `/admin/users/${id}`),
}
