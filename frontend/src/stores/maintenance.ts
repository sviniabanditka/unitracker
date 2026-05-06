import { defineStore } from 'pinia'
import { ref } from 'vue'

interface HealthResponse {
  status: string
  maintenance?: boolean
  version?: string
  time?: string
}

export const useMaintenanceStore = defineStore('maintenance', () => {
  const active = ref(false)
  const since = ref<number | null>(null)
  let timer: number | null = null
  const recoveryHandlers = new Set<() => void | Promise<void>>()

  function on() {
    if (active.value) return
    active.value = true
    since.value = Date.now()
    startPolling()
  }

  function off() {
    if (!active.value) return
    active.value = false
    since.value = null
    stopPolling()
    runRecoveryHandlers()
  }

  function startPolling() {
    if (timer != null) return
    timer = window.setInterval(check, 3000)
    void check()
  }

  function stopPolling() {
    if (timer != null) {
      window.clearInterval(timer)
      timer = null
    }
  }

  async function check() {
    try {
      const res = await fetch('/api/health', { credentials: 'include' })
      if (res.status === 503) return
      if (!res.ok) return
      const body = (await res.json()) as HealthResponse
      if (!body.maintenance) off()
    } catch {
      // Network errors — keep overlay until /api/health succeeds.
    }
  }

  function onRecovery(handler: () => void | Promise<void>) {
    recoveryHandlers.add(handler)
    return () => recoveryHandlers.delete(handler)
  }

  function runRecoveryHandlers() {
    for (const fn of recoveryHandlers) {
      try {
        const result = fn()
        if (result instanceof Promise) result.catch(() => {})
      } catch {
        // ignore
      }
    }
  }

  return { active, since, on, off, onRecovery }
})
