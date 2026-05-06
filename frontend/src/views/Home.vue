<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppLayout from '@/components/AppLayout.vue'
import { Button } from '@/components/ui/button'
import { healthApi, type HealthResponse } from '@/api/client'

const health = ref<HealthResponse | null>(null)
const error = ref<string | null>(null)
const loading = ref(false)

async function check() {
  loading.value = true
  error.value = null
  try {
    health.value = await healthApi.get()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

onMounted(check)
</script>

<template>
  <AppLayout>
    <div class="container max-w-xl px-4 py-8 space-y-6">
      <header class="space-y-2">
        <h1 class="text-2xl font-semibold">Dashboard</h1>
        <p class="text-sm text-muted-foreground">Phase 2 — Auth & Users.</p>
      </header>

      <section class="space-y-3 rounded-md border bg-card p-4">
        <h2 class="font-medium">API health</h2>
        <pre v-if="health" class="text-sm bg-muted rounded p-3 overflow-auto">{{ health }}</pre>
        <p v-else-if="loading" class="text-sm text-muted-foreground">Checking…</p>
        <p v-else-if="error" class="text-sm text-destructive">{{ error }}</p>
        <Button :disabled="loading" @click="check">Re-check</Button>
      </section>
    </div>
  </AppLayout>
</template>
