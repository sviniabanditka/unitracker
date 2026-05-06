<script setup lang="ts">
import { RouterView } from 'vue-router'
import MaintenanceOverlay from '@/components/MaintenanceOverlay.vue'
import UpdatePrompt from '@/components/UpdatePrompt.vue'
</script>

<template>
  <RouterView />
  <MaintenanceOverlay />
  <UpdatePrompt />
</template>
