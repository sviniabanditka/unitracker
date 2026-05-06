<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Select } from '@/components/ui/select'
import { useProfilesStore } from '@/stores/profiles'

const store = useProfilesStore()
const { list, activeId, ready } = storeToRefs(store)
const { t } = useI18n()

onMounted(() => {
  if (!ready.value) void store.ensure()
})

const empty = computed(() => ready.value && list.value.length === 0)
const single = computed(() => list.value.length === 1)

function onChange(v: string) {
  const id = Number(v)
  store.setActive(Number.isFinite(id) ? id : null)
}
</script>

<template>
  <div class="text-sm">
    <RouterLink
      v-if="empty"
      to="/profiles"
      class="text-muted-foreground hover:text-foreground"
    >
      + {{ t('profiles.add') }}
    </RouterLink>
    <span v-else-if="single" class="text-foreground">
      {{ list[0].name }}
    </span>
    <Select
      v-else-if="list.length > 1"
      :model-value="activeId == null ? '' : String(activeId)"
      class="h-9 w-32 lg:w-40"
      :aria-label="t('profiles.switcher')"
      @update:model-value="onChange"
    >
      <option v-for="p in list" :key="p.id" :value="p.id">{{ p.name }}</option>
    </Select>
  </div>
</template>
