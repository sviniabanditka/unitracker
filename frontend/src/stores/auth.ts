import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ApiError } from '@/api/client'
import { authApi, type User } from '@/api/auth'
import { useProfilesStore } from './profiles'
import { useEntriesStore } from './entries'
import { useTrackersStore } from './trackers'
import { useSettingsStore } from './settings'

export const useAuthStore = defineStore('auth', () => {
  const me = ref<User | null>(null)
  const ready = ref(false)
  let inflight: Promise<void> | null = null

  async function ensure() {
    if (ready.value) return
    if (inflight) return inflight
    inflight = (async () => {
      try {
        me.value = await authApi.me()
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          me.value = null
        } else {
          throw err
        }
      } finally {
        ready.value = true
        inflight = null
      }
    })()
    return inflight
  }

  async function login(username: string, password: string) {
    me.value = await authApi.login(username, password)
    ready.value = true
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      me.value = null
      useProfilesStore().reset()
      useTrackersStore().reset()
      useEntriesStore().reset()
      useSettingsStore().reset()
    }
  }

  async function changePassword(oldPassword: string, newPassword: string) {
    await authApi.changePassword(oldPassword, newPassword)
  }

  function isAdmin() {
    return me.value?.role === 'admin'
  }

  return { me, ready, ensure, login, logout, changePassword, isAdmin }
})
