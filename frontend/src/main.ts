import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { registerSW } from 'virtual:pwa-register'
import App from './App.vue'
import { router } from './router'
import './style.css'
import { setMaintenanceCallback } from '@/api/client'
import { useMaintenanceStore } from '@/stores/maintenance'
import { usePwaStore } from '@/stores/pwa'
import { i18n, readStoredLocale } from '@/i18n'

const initialTheme = (() => {
  try {
    const stored = localStorage.getItem('theme')
    if (stored === 'dark' || stored === 'light') return stored
  } catch {
    // ignore
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
})()
if (initialTheme === 'dark') document.documentElement.classList.add('dark')

const initialLocale = readStoredLocale()
if (initialLocale) document.documentElement.setAttribute('lang', initialLocale)

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(i18n)

const maintenance = useMaintenanceStore()
setMaintenanceCallback(() => maintenance.on())

const pwa = usePwaStore()
const updateSW = registerSW({
  onNeedRefresh() {
    pwa.markUpdate()
  },
  onOfflineReady() {
    // optional: could surface a toast; skip for now to avoid noise
  },
})
pwa.setUpdateHandler(updateSW)

app.mount('#app')
