<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/AppLayout.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import SchemaEditor from '@/components/trackers/SchemaEditor.vue'
import DynamicEntryForm from '@/components/trackers/DynamicEntryForm.vue'
import TrackerIconPicker from '@/components/TrackerIconPicker.vue'
import TrackerIcon from '@/components/TrackerIcon.vue'
import TrackerLibraryDialog from '@/components/trackers/TrackerLibraryDialog.vue'
import { ApiError } from '@/api/client'
import { trackersApi, type LibraryTracker } from '@/api/trackers'
import { useTrackersStore } from '@/stores/trackers'
import { useProfilesStore } from '@/stores/profiles'
import { type Schema } from '@/lib/schema'

const { t: tx } = useI18n()
const route = useRoute()
const router = useRouter()
const store = useTrackersStore()
const profilesStore = useProfilesStore()
const { activeId } = storeToRefs(profilesStore)
const libraryOpen = ref(false)

const isEdit = computed(() => Boolean(route.params.id))
const id = computed(() => (isEdit.value ? Number(route.params.id) : null))

const form = reactive({
  name: '',
  icon: '',
  color: '',
  description: '',
})

const schema = ref<Schema>({ fields: [] })
const schemaErrors = ref<string[]>([])
const previewData = ref<Record<string, unknown>>({})

const loading = ref(false)
const saving = ref(false)
const error = ref<string | null>(null)
const warnings = ref<string[]>([])

async function load() {
  if (!isEdit.value || id.value == null) return
  loading.value = true
  try {
    const t = await trackersApi.get(id.value)
    form.name = t.name
    form.icon = t.icon ?? ''
    form.color = t.color ?? ''
    form.description = t.description ?? ''
    schema.value = t.schema_json ?? { fields: [] }
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function submit() {
  error.value = null
  warnings.value = []
  if (schemaErrors.value.length) {
    error.value = 'Fix the schema errors below before saving.'
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      icon: form.icon.trim() || null,
      color: form.color.trim() || null,
      description: form.description.trim() || null,
      schema_json: schema.value,
    }
    if (isEdit.value && id.value != null) {
      const res = await store.update(id.value, payload)
      warnings.value = res.warnings ?? []
      router.push(`/trackers/${id.value}`)
    } else {
      if (activeId.value == null) {
        error.value = tx('profiles.noAccess')
        return
      }
      const created = await store.create(activeId.value, payload)
      router.push(`/trackers/${created.id}`)
    }
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : e instanceof Error ? e.message : 'Save failed'
  } finally {
    saving.value = false
  }
}

function onCloneSelect(src: LibraryTracker) {
  form.name = src.name
  form.icon = src.icon ?? ''
  form.color = src.color ?? ''
  form.description = src.description ?? ''
  // Deep clone schema so user-side edits don't mutate the cached library entry.
  schema.value = JSON.parse(JSON.stringify(src.schema_json ?? { fields: [] }))
}

onMounted(load)
</script>

<template>
  <AppLayout>
    <div class="container max-w-3xl px-4 py-8 space-y-6">
      <header class="flex items-center justify-between gap-2 flex-wrap">
        <h1 class="text-2xl font-semibold">
          {{ isEdit ? 'Edit tracker' : 'New tracker' }}
        </h1>
        <div class="flex gap-2">
          <Button v-if="!isEdit" type="button" variant="outline" @click="libraryOpen = true">
            {{ tx('trackers.cloneFromLibrary') }}
          </Button>
          <Button variant="outline" @click="router.back()">Back</Button>
        </div>
      </header>

      <p v-if="loading" class="text-sm text-muted-foreground">Loading…</p>

      <form v-else class="space-y-6" @submit.prevent="submit">
        <section class="rounded-lg border bg-card p-5 space-y-4">
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="space-y-2">
              <Label for="t-name">Name</Label>
              <Input id="t-name" v-model="form.name" required />
            </div>
            <div class="space-y-2">
              <Label for="t-color">Color</Label>
              <Input id="t-color" v-model="form.color" placeholder="#aabbcc" />
            </div>
          </div>
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <Label>Icon</Label>
              <span class="inline-flex items-center gap-2 text-xs text-muted-foreground">
                <span>Preview:</span>
                <TrackerIcon :name="form.icon || null" :color="form.color || null" :size="20" badge />
              </span>
            </div>
            <TrackerIconPicker
              :model-value="form.icon || null"
              :color="form.color || null"
              @update:model-value="form.icon = $event ?? ''"
            />
          </div>
          <div class="space-y-2">
            <Label for="t-desc">Description</Label>
            <Textarea id="t-desc" v-model="form.description" :rows="2" />
          </div>
        </section>

        <section class="rounded-lg border bg-card p-5 space-y-3">
          <h2 class="font-medium">Schema</h2>
          <SchemaEditor v-model="schema" @validity-change="schemaErrors = $event" />
        </section>

        <section v-if="schema.fields.length" class="rounded-lg border bg-card p-5 space-y-3">
          <h2 class="font-medium">Preview</h2>
          <DynamicEntryForm v-model="previewData" :schema="schema" disabled />
          <pre class="text-xs bg-muted rounded p-3 overflow-auto">{{ previewData }}</pre>
        </section>

        <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
        <ul v-if="warnings.length" class="text-sm text-amber-600 list-disc pl-5">
          <li v-for="(w, i) in warnings" :key="i">{{ w }}</li>
        </ul>

        <div class="flex gap-2">
          <Button type="submit" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</Button>
          <Button type="button" variant="outline" :disabled="saving" @click="router.back()">Cancel</Button>
        </div>
      </form>

      <TrackerLibraryDialog
        v-model:open="libraryOpen"
        @select="onCloneSelect"
      />
    </div>
  </AppLayout>
</template>
