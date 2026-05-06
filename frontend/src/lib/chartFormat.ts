import type { Bucket } from '@/api/stats'

const dayFmtCache = new Map<string, Intl.DateTimeFormat>()
const monthFmtCache = new Map<string, Intl.DateTimeFormat>()

function dayFmt(locale: string): Intl.DateTimeFormat {
  let f = dayFmtCache.get(locale)
  if (!f) {
    f = new Intl.DateTimeFormat(locale, { month: 'short', day: 'numeric' })
    dayFmtCache.set(locale, f)
  }
  return f
}

function monthFmt(locale: string): Intl.DateTimeFormat {
  let f = monthFmtCache.get(locale)
  if (!f) {
    f = new Intl.DateTimeFormat(locale, { month: 'short', year: 'numeric' })
    monthFmtCache.set(locale, f)
  }
  return f
}

function intlLocale(locale: string): string {
  return locale === 'uk' ? 'uk-UA' : 'en-US'
}

// formatBucketLabel turns a backend bucket key (always YYYY-MM-DD,
// representing the bucket start) into a human label appropriate for the
// bucket size. Months show "Apr 2026"; days/weeks show "Apr 4".
export function formatBucketLabel(key: string, bucket: Bucket, locale: string): string {
  const d = parseISODate(key)
  if (!d) return key
  const lang = intlLocale(locale)
  if (bucket === 'month') return monthFmt(lang).format(d)
  return dayFmt(lang).format(d)
}

// parseISODate handles "2026-04-04" without timezone shifts.
function parseISODate(s: string): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s)
  if (!m) return null
  return new Date(Date.UTC(+m[1], +m[2] - 1, +m[3]))
}

export function formatHourLabel(hour: number): string {
  return `${hour}:00`
}
