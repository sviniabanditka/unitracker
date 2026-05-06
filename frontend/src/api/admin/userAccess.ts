import { request } from '@/api/client'

export const userAccessApi = {
  list: (userId: number) =>
    request<{ profile_ids: number[] }>('GET', `/admin/users/${userId}/profiles`).then(
      r => r.profile_ids,
    ),
  replace: (userId: number, profileIds: number[]) =>
    request<{ profile_ids: number[] }>(
      'PUT',
      `/admin/users/${userId}/profiles`,
      { profile_ids: profileIds },
    ).then(r => r.profile_ids),
}
