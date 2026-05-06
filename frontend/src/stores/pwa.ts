import { defineStore } from 'pinia'
import { ref } from 'vue'

export const usePwaStore = defineStore('pwa', () => {
  const updateAvailable = ref(false)
  let updateFn: ((reload?: boolean) => Promise<void>) | null = null

  function setUpdateHandler(fn: (reload?: boolean) => Promise<void>) {
    updateFn = fn
  }

  function markUpdate() {
    updateAvailable.value = true
  }

  async function reload() {
    if (updateFn) {
      await updateFn(true)
    } else {
      window.location.reload()
    }
  }

  return { updateAvailable, setUpdateHandler, markUpdate, reload }
})
