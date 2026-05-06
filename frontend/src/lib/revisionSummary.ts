import type { Revision } from '@/api/revisions'
import type { Schema, FieldDef } from './schema'
import { formatFieldValue } from './format'

export interface ChangeBit {
  key: string
  label: string
  before: string
  after: string
}

function fieldByKey(schema: Schema, key: string): FieldDef | null {
  return schema.fields.find(f => f.key === key) ?? null
}

function valueText(field: FieldDef | null, raw: unknown): string {
  if (raw == null || raw === '' || (Array.isArray(raw) && raw.length === 0)) return '∅'
  if (!field) return JSON.stringify(raw)
  return formatFieldValue(field, raw) || JSON.stringify(raw)
}

function shallowEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true
  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return false
    return a.every((v, i) => v === b[i])
  }
  return JSON.stringify(a ?? null) === JSON.stringify(b ?? null)
}

// `revisions` is the full list, ordered DESC by changed_at (newest first).
// `index` is the position of the revision we're summarizing.
export function summarize(revisions: Revision[], index: number, schema: Schema): {
  headline: string
  changes: ChangeBit[]
} {
  const rev = revisions[index]
  if (!rev) return { headline: '', changes: [] }

  if (rev.change_type === 'create') {
    return { headline: 'Created', changes: [] }
  }
  if (rev.change_type === 'delete') {
    return { headline: 'Deleted', changes: [] }
  }

  // The revision *before* this one in chronological order is at index+1
  // (because list is newest-first).
  const prev = revisions[index + 1]
  if (!prev) {
    return {
      headline: rev.change_type === 'restore' ? 'Restored' : 'Updated',
      changes: [],
    }
  }

  const keys = new Set<string>([
    ...Object.keys(prev.data ?? {}),
    ...Object.keys(rev.data ?? {}),
  ])
  const changes: ChangeBit[] = []
  for (const key of keys) {
    const a = prev.data?.[key]
    const b = rev.data?.[key]
    if (shallowEqual(a, b)) continue
    const f = fieldByKey(schema, key)
    changes.push({
      key,
      label: f?.label.en ?? key,
      before: valueText(f, a),
      after: valueText(f, b),
    })
  }

  if (rev.is_deleted !== prev.is_deleted) {
    changes.push({
      key: '__deleted',
      label: 'Status',
      before: prev.is_deleted ? 'deleted' : 'active',
      after: rev.is_deleted ? 'deleted' : 'active',
    })
  }
  if (rev.occurred_at !== prev.occurred_at) {
    changes.push({
      key: '__occurred_at',
      label: 'When',
      before: prev.occurred_at,
      after: rev.occurred_at,
    })
  }

  let headline: string
  if (rev.change_type === 'restore') {
    headline = changes.length ? 'Restored' : 'Restored (no field changes)'
  } else {
    headline = changes.length
      ? `Updated: ${changes.map(c => c.label).join(', ')}`
      : 'Updated (no detectable changes)'
  }

  return { headline, changes }
}
