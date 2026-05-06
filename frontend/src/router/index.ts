import {
  createRouter,
  createWebHistory,
  type RouteLocationNormalized,
  type RouteRecordRaw,
} from 'vue-router'
import { useAuthStore } from '@/stores/auth'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
    publicOnly?: boolean
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/Login.vue'),
    meta: { publicOnly: true },
  },
  {
    path: '/',
    name: 'home',
    component: () => import('@/views/Dashboard.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/account',
    name: 'account',
    component: () => import('@/views/Account.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/profiles',
    name: 'profiles',
    component: () => import('@/views/ProfilesAdmin.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/trackers',
    name: 'trackers',
    component: () => import('@/views/TrackerList.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/trackers/new',
    name: 'tracker-new',
    component: () => import('@/views/TrackerForm.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
  },
  {
    path: '/trackers/:id/edit',
    name: 'tracker-edit',
    component: () => import('@/views/TrackerForm.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
  },
  {
    path: '/trackers/:id',
    name: 'tracker-view',
    component: () => import('@/views/TrackerView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/admin/users',
    name: 'admin-users',
    component: () => import('@/views/admin/UsersAdmin.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
  },
  {
    path: '/admin/backups',
    name: 'admin-backups',
    component: () => import('@/views/admin/BackupsAdmin.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
  },
  {
    path: '/admin/settings',
    name: 'admin-settings',
    component: () => import('@/views/admin/SettingsAdmin.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/',
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to: RouteLocationNormalized) => {
  const auth = useAuthStore()
  await auth.ensure()

  if (to.meta.publicOnly && auth.me) {
    return { path: '/' }
  }
  if (to.meta.requiresAuth && !auth.me) {
    return { path: '/login', query: to.fullPath !== '/' ? { next: to.fullPath } : undefined }
  }
  if (to.meta.requiresAdmin && auth.me?.role !== 'admin') {
    return { path: '/' }
  }
  return true
})
