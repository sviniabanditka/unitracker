import type { FieldDef } from './schema'

const SAME_DAY_TIME = new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' })
const FULL_DT = new Intl.DateTimeFormat(undefined, {
  month: 'short',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
})

export function formatOccurredAt(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const now = new Date()
  const sameDay =
    d.getFullYear() === now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate()
  return sameDay ? SAME_DAY_TIME.format(d) : FULL_DT.format(d)
}

export function formatFieldValue(field: FieldDef, raw: unknown): string {
  if (raw == null || raw === '') return ''
  switch (field.type) {
    case 'datetime':
      return formatOccurredAt(String(raw))
    case 'date':
      return String(raw)
    case 'time':
      return String(raw)
    case 'number': {
      const n = Number(raw)
      if (!Number.isFinite(n)) return String(raw)
      return field.unit ? `${n} ${field.unit}` : String(n)
    }
    case 'duration': {
      const ms = Number(raw)
      if (!Number.isFinite(ms)) return String(raw)
      return formatDurationMs(ms)
    }
    case 'select': {
      const opt = field.options?.find(o => o.value === raw)
      return opt?.label.en ?? String(raw)
    }
    case 'multiselect':
      if (!Array.isArray(raw)) return ''
      return raw
        .map(v => field.options?.find(o => o.value === v)?.label.en ?? String(v))
        .join(', ')
    case 'boolean':
      return raw ? 'yes' : 'no'
    case 'color':
      return String(raw)
    case 'text':
    case 'longtext':
      return String(raw)
    default:
      return String(raw)
  }
}

export function formatDurationMs(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000))
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  if (h > 0) return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  return `${m}:${s.toString().padStart(2, '0')}`
}

// "1:30" or "1:30:45" or "90" (seconds) → milliseconds
export function parseDurationToMs(s: string): number | null {
  if (!s) return null
  const trimmed = s.trim()
  if (/^\d+$/.test(trimmed)) {
    return Number(trimmed) * 1000
  }
  const parts = trimmed.split(':').map(p => p.trim())
  if (parts.some(p => !/^\d+$/.test(p))) return null
  const nums = parts.map(Number)
  let h = 0
  let m = 0
  let sec = 0
  if (nums.length === 2) {
    ;[m, sec] = nums
  } else if (nums.length === 3) {
    ;[h, m, sec] = nums
  } else {
    return null
  }
  return ((h * 60 + m) * 60 + sec) * 1000
}

export function nowDatetimeLocal(): string {
  const d = new Date()
  const tz = d.getTimezoneOffset()
  const local = new Date(d.getTime() - tz * 60_000)
  return local.toISOString().slice(0, 16)
}

export function localDatetimeToIso(local: string): string {
  if (!local) return ''
  const d = new Date(local)
  if (Number.isNaN(d.getTime())) return local
  return d.toISOString()
}

export function isoToLocalDatetime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const tz = d.getTimezoneOffset()
  const local = new Date(d.getTime() - tz * 60_000)
  return local.toISOString().slice(0, 16)
}
