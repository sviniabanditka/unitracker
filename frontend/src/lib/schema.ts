export type FieldType =
  | 'datetime'
  | 'date'
  | 'time'
  | 'duration'
  | 'number'
  | 'text'
  | 'longtext'
  | 'select'
  | 'multiselect'
  | 'boolean'
  | 'color'

export const FIELD_TYPES: FieldType[] = [
  'datetime',
  'date',
  'time',
  'duration',
  'number',
  'text',
  'longtext',
  'select',
  'multiselect',
  'boolean',
  'color',
]

export const TIME_FIELD_TYPES: FieldType[] = ['datetime', 'date', 'time']

export interface FieldOption {
  value: string
  label: { en: string; uk?: string }
}

export interface FieldDef {
  key: string
  label: { en: string; uk?: string }
  type: FieldType
  required?: boolean
  isPrimaryTime?: boolean
  unit?: string
  min?: number
  max?: number
  options?: FieldOption[]
}

export interface Schema {
  fields: FieldDef[]
}

export const KEY_REGEX = /^[a-z][a-z0-9_]*$/

export function defaultFieldByType(type: FieldType): FieldDef {
  const base: FieldDef = {
    key: '',
    label: { en: '' },
    type,
  }
  if (type === 'select' || type === 'multiselect') {
    base.options = [{ value: '', label: { en: '' } }]
  }
  return base
}

export function validateSchemaClient(schema: Schema): string[] {
  const errors: string[] = []
  if (!schema.fields.length) {
    errors.push('At least one field is required')
    return errors
  }
  const seenKeys = new Set<string>()
  let primary = 0
  schema.fields.forEach((f, idx) => {
    const where = `Field #${idx + 1}${f.key ? ` (${f.key})` : ''}`
    if (!KEY_REGEX.test(f.key)) {
      errors.push(`${where}: key must match [a-z][a-z0-9_]*`)
    }
    if (seenKeys.has(f.key)) {
      errors.push(`${where}: duplicate key "${f.key}"`)
    } else if (f.key) {
      seenKeys.add(f.key)
    }
    if (!f.label?.en) {
      errors.push(`${where}: label.en is required`)
    }
    if (!FIELD_TYPES.includes(f.type)) {
      errors.push(`${where}: unknown type "${f.type}"`)
    }
    if (f.isPrimaryTime) {
      if (!TIME_FIELD_TYPES.includes(f.type)) {
        errors.push(`${where}: isPrimaryTime requires datetime|date|time`)
      }
      primary++
    }
    if (f.type === 'select' || f.type === 'multiselect') {
      if (!f.options?.length) {
        errors.push(`${where}: ${f.type} requires at least one option`)
      } else {
        const seenVals = new Set<string>()
        f.options.forEach((opt, i) => {
          if (!opt.value) errors.push(`${where}: option #${i + 1} value is required`)
          if (seenVals.has(opt.value)) errors.push(`${where}: duplicate option "${opt.value}"`)
          else if (opt.value) seenVals.add(opt.value)
          if (!opt.label?.en) errors.push(`${where}: option #${i + 1} label.en is required`)
        })
      }
    }
  })
  if (primary > 1) errors.push('At most one field may have isPrimaryTime')
  return errors
}

export function emptySchema(): Schema {
  return { fields: [] }
}
