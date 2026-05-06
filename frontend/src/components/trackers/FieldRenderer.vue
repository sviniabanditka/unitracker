<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Select } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { FieldDef } from '@/lib/schema'
import {
  formatDurationMs,
  isoToLocalDatetime,
  localDatetimeToIso,
  parseDurationToMs,
} from '@/lib/format'
import { resolveLabel } from '@/lib/locale'
import type { Locale } from '@/i18n'

const { locale, t } = useI18n()
const currentLocale = computed<Locale>(() => (locale.value === 'uk' ? 'uk' : 'en'))

const props = defineProps<{
  field: FieldDef
  modelValue?: unknown
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: unknown): void
}>()

function asString(v: unknown): string {
  return typeof v === 'string' ? v : v == null ? '' : String(v)
}

function asNumber(v: unknown): number | null {
  if (v === '' || v == null) return null
  const n = Number(v)
  return Number.isFinite(n) ? n : null
}

function asArray(v: unknown): string[] {
  return Array.isArray(v) ? v.map(String) : []
}

const datetimeValue = computed(() => {
  const s = asString(props.modelValue)
  if (!s) return ''
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(s)) return s
  return isoToLocalDatetime(s)
})

function emitDatetime(local: string) {
  if (!local) {
    emit('update:modelValue', '')
    return
  }
  emit('update:modelValue', localDatetimeToIso(local))
}

const durationText = computed(() => {
  const v = props.modelValue
  if (v == null || v === '') return ''
  if (typeof v === 'number') return formatDurationMs(v)
  if (typeof v === 'string' && /^\d+$/.test(v)) return formatDurationMs(Number(v))
  return asString(v)
})

function emitDuration(text: string) {
  if (text === '') {
    emit('update:modelValue', null)
    return
  }
  const ms = parseDurationToMs(text)
  emit('update:modelValue', ms == null ? text : ms)
}

function onMultiselectChange(event: Event) {
  const select = event.target as HTMLSelectElement
  const selected = Array.from(select.selectedOptions).map(o => o.value)
  emit('update:modelValue', selected)
}
</script>

<template>
  <div class="space-y-1">
    <Input
      v-if="field.type === 'text'"
      :model-value="asString(modelValue)"
      :disabled="disabled"
      :required="field.required"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <Textarea
      v-else-if="field.type === 'longtext'"
      :model-value="asString(modelValue)"
      :disabled="disabled"
      :required="field.required"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <div v-else-if="field.type === 'number'" class="flex items-center gap-2">
      <Input
        type="number"
        :model-value="asString(modelValue)"
        :disabled="disabled"
        :required="field.required"
        :min="field.min"
        :max="field.max"
        @update:model-value="emit('update:modelValue', asNumber($event))"
      />
      <span v-if="field.unit" class="text-sm text-muted-foreground">{{ field.unit }}</span>
    </div>
    <Input
      v-else-if="field.type === 'datetime'"
      type="datetime-local"
      :model-value="datetimeValue"
      :disabled="disabled"
      :required="field.required"
      @update:model-value="emitDatetime($event)"
    />
    <Input
      v-else-if="field.type === 'date'"
      type="date"
      :model-value="asString(modelValue)"
      :disabled="disabled"
      :required="field.required"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <Input
      v-else-if="field.type === 'time'"
      type="time"
      :model-value="asString(modelValue)"
      :disabled="disabled"
      :required="field.required"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <Input
      v-else-if="field.type === 'duration'"
      type="text"
      placeholder="hh:mm:ss or mm:ss"
      :model-value="durationText"
      :disabled="disabled"
      :required="field.required"
      @update:model-value="emitDuration($event)"
    />
    <Input
      v-else-if="field.type === 'color'"
      type="color"
      class="h-10 w-20 cursor-pointer p-1"
      :model-value="asString(modelValue) || '#000000'"
      :disabled="disabled"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <Select
      v-else-if="field.type === 'select'"
      :model-value="asString(modelValue)"
      :disabled="disabled"
      :required="field.required"
      @update:model-value="emit('update:modelValue', $event)"
    >
      <option value="">— {{ t('common.none') }} —</option>
      <option v-for="opt in field.options" :key="opt.value" :value="opt.value">
        {{ resolveLabel(opt.label, currentLocale) }}
      </option>
    </Select>
    <select
      v-else-if="field.type === 'multiselect'"
      multiple
      :disabled="disabled"
      class="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
      :value="asArray(modelValue)"
      @change="onMultiselectChange"
    >
      <option v-for="opt in field.options" :key="opt.value" :value="opt.value">
        {{ resolveLabel(opt.label, currentLocale) }}
      </option>
    </select>
    <label v-else-if="field.type === 'boolean'" class="flex items-center gap-2">
      <Switch
        :model-value="Boolean(modelValue)"
        :disabled="disabled"
        @update:model-value="emit('update:modelValue', $event)"
      />
      <span class="text-sm text-muted-foreground">{{ modelValue ? t('common.yes') : t('common.no') }}</span>
    </label>
    <p v-else class="text-sm text-destructive">Unsupported field type: {{ field.type }}</p>
  </div>
</template>
