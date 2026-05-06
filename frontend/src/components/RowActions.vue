<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Settings } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'

export interface RowAction {
  label: string
  onClick?: () => void | Promise<void>
  href?: string
  variant?: 'default' | 'outline' | 'destructive' | 'ghost'
  disabled?: boolean
  hidden?: boolean
}

const props = defineProps<{ actions: RowAction[] }>()
const { t } = useI18n()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
const triggerEl = ref<HTMLElement | null>(null)
const menuEl = ref<HTMLElement | null>(null)
const menuStyle = ref<{ top: string; right: string }>({ top: '0', right: '0' })

const visible = computed(() => props.actions.filter(a => !a.hidden))

function close() {
  open.value = false
}

function reposition() {
  const tEl = triggerEl.value
  if (!tEl) return
  const r = tEl.getBoundingClientRect()
  menuStyle.value = {
    top: `${Math.round(r.bottom + 4)}px`,
    right: `${Math.round(window.innerWidth - r.right)}px`,
  }
}

function onDocMouseDown(e: MouseEvent) {
  if (!open.value) return
  const target = e.target as Node | null
  if (!target) return
  if (root.value?.contains(target)) return
  if (menuEl.value?.contains(target)) return
  close()
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && open.value) close()
}

onMounted(() => {
  document.addEventListener('mousedown', onDocMouseDown)
  document.addEventListener('keydown', onKey)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocMouseDown)
  document.removeEventListener('keydown', onKey)
  window.removeEventListener('resize', reposition)
  window.removeEventListener('scroll', reposition, true)
})

watch(open, on => {
  if (on) {
    nextTick(() => {
      reposition()
      window.addEventListener('resize', reposition)
      window.addEventListener('scroll', reposition, true)
    })
  } else {
    window.removeEventListener('resize', reposition)
    window.removeEventListener('scroll', reposition, true)
  }
})

function fire(a: RowAction) {
  if (a.disabled) return
  close()
  if (a.onClick) void a.onClick()
}
</script>

<template>
  <div ref="root" class="inline-flex items-center">
    <div class="hidden lg:flex gap-2">
      <template v-for="(a, i) in visible" :key="i">
        <a v-if="a.href" :href="a.href">
          <Button
            type="button"
            size="sm"
            :variant="a.variant ?? 'outline'"
            :disabled="a.disabled"
          >{{ a.label }}</Button>
        </a>
        <Button
          v-else
          type="button"
          size="sm"
          :variant="a.variant ?? 'outline'"
          :disabled="a.disabled"
          @click="fire(a)"
        >{{ a.label }}</Button>
      </template>
    </div>

    <button
      ref="triggerEl"
      type="button"
      class="lg:hidden inline-flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground hover:bg-accent disabled:opacity-50"
      :aria-label="t('common.actions')"
      :aria-expanded="open"
      :aria-haspopup="true"
      :disabled="visible.length === 0"
      @click.stop="open = !open"
    >
      <Settings class="h-4 w-4" aria-hidden="true" />
    </button>

    <Teleport to="body">
      <div
        v-if="open"
        ref="menuEl"
        :style="menuStyle"
        class="lg:hidden fixed z-50 min-w-[10rem] rounded-md border bg-card text-card-foreground shadow-lg py-1"
        role="menu"
      >
        <template v-for="(a, i) in visible" :key="i">
          <a
            v-if="a.href"
            :href="a.href"
            role="menuitem"
            class="block px-3 py-2 text-sm hover:bg-accent"
            :class="a.disabled ? 'opacity-50 pointer-events-none' : ''"
            @click="close"
          >{{ a.label }}</a>
          <button
            v-else
            type="button"
            role="menuitem"
            class="block w-full text-left px-3 py-2 text-sm hover:bg-accent disabled:opacity-50"
            :class="a.variant === 'destructive' ? 'text-destructive' : ''"
            :disabled="a.disabled"
            @click="fire(a)"
          >{{ a.label }}</button>
        </template>
      </div>
    </Teleport>
  </div>
</template>
