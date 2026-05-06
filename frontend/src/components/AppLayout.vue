<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import ProfileSwitcher from '@/components/ProfileSwitcher.vue'
import LocaleSwitcher from '@/components/LocaleSwitcher.vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { setLocale, isSupportedLocale, readStoredLocale } from '@/i18n'

const auth = useAuthStore()
const { me } = storeToRefs(auth)
const settings = useSettingsStore()
const { appName, defaultLocale } = storeToRefs(settings)
const { t, locale } = useI18n()
const router = useRouter()
const route = useRoute()

const isAdmin = computed(() => me.value?.role === 'admin')

const menuOpen = ref(false)
function closeMenu() {
  menuOpen.value = false
}

watch(() => route.fullPath, closeMenu)

watch(
  menuOpen,
  open => {
    document.body.style.overflow = open ? 'hidden' : ''
  },
  { flush: 'sync' },
)

onBeforeUnmount(() => {
  document.body.style.overflow = ''
})

watch(
  () => me.value?.role,
  role => {
    if (role === 'admin' && !settings.loaded) {
      settings.fetchAll().catch(() => undefined)
    }
  },
  { immediate: false },
)

watch(defaultLocale, (next, prev) => {
  if (next === prev) return
  if (readStoredLocale()) return
  if (isSupportedLocale(next)) setLocale(next)
})

onMounted(() => {
  if (isAdmin.value && !settings.loaded) {
    settings.fetchAll().catch(() => undefined)
  }
})

async function handleLogout() {
  closeMenu()
  await auth.logout()
  router.push('/login')
}

const linkBase = 'text-muted-foreground hover:text-foreground'
const drawerLink =
  'block rounded-md px-3 py-2 text-sm hover:bg-accent text-foreground'

const adminLinks = computed(() => [
  { to: '/admin/users', label: t('nav.users') },
  { to: '/admin/backups', label: t('nav.backups') },
  { to: '/admin/settings', label: t('nav.settings') },
])
</script>

<template>
  <div class="min-h-screen flex flex-col bg-background text-foreground">
    <header class="border-b">
      <div class="container max-w-4xl lg:max-w-6xl flex h-14 items-center justify-between gap-4 lg:gap-6 px-4">
        <RouterLink to="/" class="font-semibold whitespace-nowrap truncate">
          {{ appName }}
        </RouterLink>

        <nav class="hidden md:flex items-center gap-3 text-sm min-w-0">
          <RouterLink to="/trackers" :class="linkBase">{{ t('nav.trackers') }}</RouterLink>
          <RouterLink to="/profiles" :class="linkBase">{{ t('nav.profiles') }}</RouterLink>
          <RouterLink to="/account" :class="linkBase">{{ t('nav.account') }}</RouterLink>
          <template v-if="isAdmin">
            <RouterLink to="/admin/users" :class="[linkBase, 'lg:hidden']">{{ t('nav.admin') }}</RouterLink>
            <RouterLink to="/admin/users" :class="[linkBase, 'hidden lg:inline']">{{ t('nav.users') }}</RouterLink>
            <RouterLink to="/admin/backups" :class="[linkBase, 'hidden lg:inline']">{{ t('nav.backups') }}</RouterLink>
            <RouterLink to="/admin/settings" :class="[linkBase, 'hidden lg:inline']">{{ t('nav.settings') }}</RouterLink>
          </template>
        </nav>

        <div class="hidden md:flex items-center gap-2 text-sm">
          <ProfileSwitcher v-if="me" />
          <span v-if="me" class="hidden xl:inline">{{ me.username }}</span>
          <Badge v-if="me" :variant="isAdmin ? 'default' : 'secondary'" class="hidden lg:inline-flex">{{ me.role }}</Badge>
          <LocaleSwitcher />
          <ThemeToggle />
          <Button v-if="me" variant="outline" size="sm" @click="handleLogout">
            {{ t('auth.logout') }}
          </Button>
        </div>

        <Button
          v-if="me"
          type="button"
          variant="ghost"
          size="sm"
          class="md:hidden px-2"
          :aria-label="menuOpen ? t('nav.closeMenu') : t('nav.openMenu')"
          :aria-expanded="menuOpen"
          @click="menuOpen = !menuOpen"
        >
          <span aria-hidden="true" class="text-xl leading-none">{{ menuOpen ? '✕' : '☰' }}</span>
        </Button>
      </div>
    </header>

    <Teleport to="body">
      <Transition name="fade">
        <div
          v-if="menuOpen"
          class="md:hidden fixed inset-0 z-40 bg-background/60 backdrop-blur-sm"
          @click="closeMenu"
        />
      </Transition>

      <aside
        class="md:hidden fixed top-0 right-0 z-50 h-full w-80 max-w-[85%] bg-card border-l shadow-xl flex flex-col transition-transform duration-200 ease-out"
        :class="menuOpen ? 'translate-x-0' : 'translate-x-full pointer-events-none'"
        :aria-hidden="!menuOpen"
        :inert="!menuOpen"
      >
        <header class="border-b px-4 h-14 flex items-center justify-between gap-3">
          <span class="font-semibold truncate">{{ appName }}</span>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="px-2"
            :aria-label="t('nav.closeMenu')"
            @click="closeMenu"
          >
            <span aria-hidden="true" class="text-xl leading-none">✕</span>
          </Button>
        </header>

        <nav class="flex-1 overflow-y-auto p-3 space-y-0.5">
          <RouterLink to="/" :class="drawerLink" active-class="bg-accent">
            {{ t('nav.home') }}
          </RouterLink>
          <RouterLink to="/trackers" :class="drawerLink" active-class="bg-accent">
            {{ t('nav.trackers') }}
          </RouterLink>
          <RouterLink to="/profiles" :class="drawerLink" active-class="bg-accent">
            {{ t('nav.profiles') }}
          </RouterLink>
          <RouterLink to="/account" :class="drawerLink" active-class="bg-accent">
            {{ t('nav.account') }}
          </RouterLink>

          <template v-if="isAdmin">
            <div class="pt-3 pb-1 px-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              {{ t('nav.admin') }}
            </div>
            <RouterLink to="/admin/users" :class="drawerLink" active-class="bg-accent">
              {{ t('nav.users') }}
            </RouterLink>
            <RouterLink to="/admin/backups" :class="drawerLink" active-class="bg-accent">
              {{ t('nav.backups') }}
            </RouterLink>
            <RouterLink to="/admin/settings" :class="drawerLink" active-class="bg-accent">
              {{ t('nav.settings') }}
            </RouterLink>
          </template>
        </nav>

        <footer class="border-t p-3 space-y-3">
          <ProfileSwitcher v-if="me" />
          <div v-if="me" class="flex items-center justify-between gap-2 px-1 text-sm">
            <span class="truncate">{{ me.username }}</span>
            <Badge :variant="isAdmin ? 'default' : 'secondary'">{{ me.role }}</Badge>
          </div>
          <div class="flex items-center justify-between gap-2 px-1">
            <LocaleSwitcher />
            <ThemeToggle />
          </div>
          <Button
            v-if="me"
            type="button"
            variant="outline"
            class="w-full"
            @click="handleLogout"
          >
            {{ t('auth.logout') }}
          </Button>
        </footer>
      </aside>
    </Teleport>

    <nav
      v-if="isAdmin && route.path.startsWith('/admin')"
      class="lg:hidden border-b bg-card/40"
      :aria-label="t('nav.admin')"
    >
      <div class="container max-w-4xl flex gap-1 px-4 overflow-x-auto text-sm">
        <RouterLink
          v-for="link in adminLinks"
          :key="link.to"
          :to="link.to"
          class="px-3 py-2 whitespace-nowrap text-muted-foreground hover:text-foreground border-b-2 border-transparent"
          active-class="text-foreground border-primary"
        >
          {{ link.label }}
        </RouterLink>
      </div>
    </nav>

    <main class="flex-1" :data-locale="locale">
      <slot />
    </main>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 200ms ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
