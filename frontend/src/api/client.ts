const BASE = '/api'

export class ApiError extends Error {
  status: number
  code: string | null
  body: unknown

  constructor(status: number, message: string, body: unknown, code: string | null = null) {
    super(message)
    this.status = status
    this.body = body
    this.code = code
  }
}

type MaintenanceCallback = () => void
let maintenanceCallback: MaintenanceCallback | null = null

export function setMaintenanceCallback(cb: MaintenanceCallback | null) {
  maintenanceCallback = cb
}

export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    credentials: 'include',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (res.status === 503) {
    maintenanceCallback?.()
  }
  if (!res.ok) {
    let parsed: unknown = null
    let text = ''
    try {
      text = await res.text()
      parsed = text ? JSON.parse(text) : null
    } catch {
      parsed = text
    }
    let message = `${res.status} ${res.statusText}`
    let code: string | null = null
    if (parsed && typeof parsed === 'object') {
      const obj = parsed as { code?: unknown; message?: unknown; error?: unknown }
      if (typeof obj.message === 'string') message = obj.message
      else if (typeof obj.error === 'string') message = obj.error
      if (typeof obj.code === 'string') code = obj.code
    }
    throw new ApiError(res.status, message, parsed, code)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export interface HealthResponse {
  status: string
  version: string
  time: string
  maintenance?: boolean
}

export const healthApi = {
  get: () => request<HealthResponse>('GET', '/health'),
}
