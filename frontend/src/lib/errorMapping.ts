import type { Composer } from 'vue-i18n'
import { ApiError } from '@/api/client'

// errorBody is the unified envelope: { code, message, fields? }.
type ErrorBody = { code?: string; message?: string; error?: string; fields?: Record<string, string> } | null | undefined

export function pickErrorCode(err: unknown): string | null {
  if (err instanceof ApiError) {
    const body = err.body as ErrorBody
    if (body && typeof body === 'object' && typeof body.code === 'string') {
      return body.code
    }
  }
  return null
}

// localizeError returns a user-facing string for the given error using the
// stable backend code → i18n key map. Falls back to the error message.
export function localizeError(err: unknown, t: Composer['t']): string {
  const code = pickErrorCode(err)
  if (code) {
    const key = `errors.${code}`
    const v = t(key)
    if (v && v !== key) return v
  }
  if (err instanceof ApiError) {
    const body = err.body as ErrorBody
    if (body?.message) return body.message
    return err.message
  }
  if (err instanceof Error) return err.message
  return String(err ?? t('errors.unknown'))
}

export function fieldErrors(err: unknown): Record<string, string> | null {
  if (!(err instanceof ApiError)) return null
  const body = err.body as ErrorBody
  if (body && body.fields && typeof body.fields === 'object') {
    return body.fields
  }
  return null
}
