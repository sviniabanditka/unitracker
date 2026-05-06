import { createI18n } from 'vue-i18n'
import en from './en.json'
import uk from './uk.json'

export type Locale = 'en' | 'uk'
export const SUPPORTED_LOCALES: Locale[] = ['en', 'uk']
export const STORAGE_KEY = 'locale'

export function readStoredLocale(): Locale | null {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === 'en' || v === 'uk') return v
  } catch {
    // ignore
  }
  return null
}

export function persistLocale(locale: Locale) {
  try {
    localStorage.setItem(STORAGE_KEY, locale)
  } catch {
    // ignore
  }
}

export function isSupportedLocale(v: string): v is Locale {
  return v === 'en' || v === 'uk'
}

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: readStoredLocale() ?? 'en',
  fallbackLocale: 'en',
  messages: { en, uk },
})

export function setLocale(locale: Locale) {
  i18n.global.locale.value = locale
  document.documentElement.setAttribute('lang', locale)
  persistLocale(locale)
}
