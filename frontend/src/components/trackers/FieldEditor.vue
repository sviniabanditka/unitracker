<script setup lang="ts">
import { computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { type FieldDef, type FieldType, FIELD_TYPES, TIME_FIELD_TYPES, defaultFieldByType } from '@/lib/schema'

const props = defineProps<{
  field: FieldDef
  index: number
  total: number
}>()

const emit = defineEmits<{
  (e: 'update', f: FieldDef): void
  (e: 'remove'): void
  (e: 'move', dir: -1 | 1): void
}>()

const isTimeType = computed(() => TIME_FIELD_TYPES.includes(props.field.type))
const isNumber = computed(() => props.field.type === 'number')
const hasOptions = computed(() => props.field.type === 'select' || props.field.type === 'multiselect')

function patch(p: Partial<FieldDef>) {
  emit('update', { ...props.field, ...p })
}

function onTypeChange(t: string) {
  const next: FieldDef = {
    ...defaultFieldByType(t as FieldType),
    key: props.field.key,
    label: props.field.label,
    required: props.field.required,
  }
  // Drop incompatible flags.
  if (!TIME_FIELD_TYPES.includes(next.type)) next.isPrimaryTime = false
  emit('update', next)
}

function addOption() {
  const opts = [...(props.field.options ?? []), { value: '', label: { en: '' } }]
  patch({ options: opts })
}

function updateOption(i: number, p: Partial<{ value: string; labelEn: string }>) {
  const opts = (props.field.options ?? []).map((o, j) =>
    j === i
      ? {
          value: p.value !== undefined ? p.value : o.value,
          label: { ...o.label, en: p.labelEn !== undefined ? p.labelEn : o.label.en },
        }
      : o,
  )
  patch({ options: opts })
}

function removeOption(i: number) {
  const opts = (props.field.options ?? []).filter((_, j) => j !== i)
  patch({ options: opts })
}
</script>

<template>
  <div class="rounded-lg border bg-card p-4 space-y-3">
    <header class="flex items-center justify-between gap-2">
      <span class="text-sm font-medium">
        Field #{{ index + 1 }}
        <span v-if="field.key" class="text-muted-foreground">— {{ field.key }}</span>
      </span>
      <div class="flex items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :disabled="index === 0"
          @click="emit('move', -1)"
        >↑</Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :disabled="index === total - 1"
          @click="emit('move', 1)"
        >↓</Button>
        <Button type="button" variant="destructive" size="sm" @click="emit('remove')">Remove</Button>
      </div>
    </header>

    <div class="grid gap-3 sm:grid-cols-2">
      <div class="space-y-2">
        <Label :for="`fk-${index}`">Key</Label>
        <Input
          :id="`fk-${index}`"
          :model-value="field.key"
          placeholder="e.g. amount"
          @update:model-value="patch({ key: String($event) })"
        />
      </div>
      <div class="space-y-2">
        <Label :for="`ft-${index}`">Type</Label>
        <Select :id="`ft-${index}`" :model-value="field.type" @update:model-value="onTypeChange">
          <option v-for="t in FIELD_TYPES" :key="t" :value="t">{{ t }}</option>
        </Select>
      </div>
      <div class="space-y-2">
        <Label :for="`fle-${index}`">Label (EN)</Label>
        <Input
          :id="`fle-${index}`"
          :model-value="field.label.en"
          @update:model-value="patch({ label: { ...field.label, en: String($event) } })"
        />
      </div>
      <div class="space-y-2">
        <Label :for="`flu-${index}`">Label (UK)</Label>
        <Input
          :id="`flu-${index}`"
          :model-value="field.label.uk ?? ''"
          @update:model-value="patch({ label: { ...field.label, uk: String($event) || undefined } })"
        />
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-4">
      <label class="flex items-center gap-2 text-sm">
        <Switch
          :model-value="!!field.required"
          @update:model-value="patch({ required: $event })"
        />
        Required
      </label>
      <label v-if="isTimeType" class="flex items-center gap-2 text-sm">
        <Switch
          :model-value="!!field.isPrimaryTime"
          @update:model-value="patch({ isPrimaryTime: $event })"
        />
        Primary time
      </label>
    </div>

    <div v-if="isNumber" class="grid gap-3 sm:grid-cols-3">
      <div class="space-y-2">
        <Label :for="`fmin-${index}`">Min</Label>
        <Input
          :id="`fmin-${index}`"
          type="number"
          :model-value="field.min ?? ''"
          @update:model-value="patch({ min: $event === '' || $event === null ? undefined : Number($event) })"
        />
      </div>
      <div class="space-y-2">
        <Label :for="`fmax-${index}`">Max</Label>
        <Input
          :id="`fmax-${index}`"
          type="number"
          :model-value="field.max ?? ''"
          @update:model-value="patch({ max: $event === '' || $event === null ? undefined : Number($event) })"
        />
      </div>
      <div class="space-y-2">
        <Label :for="`funit-${index}`">Unit</Label>
        <Input
          :id="`funit-${index}`"
          :model-value="field.unit ?? ''"
          placeholder="ml, g, …"
          @update:model-value="patch({ unit: String($event) || undefined })"
        />
      </div>
    </div>

    <div v-if="hasOptions" class="space-y-2">
      <Label>Options</Label>
      <div v-for="(opt, i) in field.options" :key="i" class="flex items-center gap-2">
        <Input
          class="flex-1"
          placeholder="value"
          :model-value="opt.value"
          @update:model-value="updateOption(i, { value: String($event) })"
        />
        <Input
          class="flex-1"
          placeholder="label (EN)"
          :model-value="opt.label.en"
          @update:model-value="updateOption(i, { labelEn: String($event) })"
        />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :disabled="(field.options?.length ?? 0) <= 1"
          @click="removeOption(i)"
        >Remove</Button>
      </div>
      <Button type="button" variant="outline" size="sm" @click="addOption">+ Add option</Button>
    </div>
  </div>
</template>
