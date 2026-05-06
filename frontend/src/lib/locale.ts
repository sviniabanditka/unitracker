import type { Locale } from '@/i18n'

// resolveLabel reads a label that can be either a plain string or a per-locale
// dictionary like {en: "...", uk: "..."}.
export function resolveLabel(
  label: string | Record<string, string> | null | undefined,
  locale: Locale,
): string {
  if (!label) return ''
  if (typeof label === 'string') return label
  if (label[locale]) return label[locale]
  if (label.en) return label.en
  const first = Object.values(label).find(v => typeof v === 'string' && v !== '')
  return first ?? ''
}
