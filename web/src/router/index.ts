import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

// 路由元信息：required 权限码（任一匹配即可，空数组表示仅需登录）。
declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    guest?: boolean
    perms?: string[]
    title?: string
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { guest: true, title: '登录' },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/views/RegisterView.vue'),
    meta: { guest: true, title: '注册' },
  },
  {
    path: '/',
    component: () => import('@/components/AppLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', name: 'dashboard', component: () => import('@/views/DashboardView.vue'), meta: { title: '仪表盘', perms: ['report:read', 'part:read'] } },
      { path: 'profile', name: 'profile', component: () => import('@/views/ProfileView.vue'), meta: { title: '个人资料' } },
      { path: 'users', name: 'users', component: () => import('@/views/UsersView.vue'), meta: { title: '用户管理', perms: ['user:read'] } },
      { path: 'roles', name: 'roles', component: () => import('@/views/RolesView.vue'), meta: { title: '角色与权限', perms: ['role:read'] } },
      { path: 'vehicles', name: 'vehicles', component: () => import('@/views/VehiclesView.vue'), meta: { title: '车辆', perms: ['vehicle:read'] } },
      { path: 'vehicles/:id', name: 'vehicle-detail', component: () => import('@/views/VehicleDetailView.vue'), meta: { title: '车辆详情', perms: ['vehicle:read'] } },
      { path: 'drivers', name: 'drivers', component: () => import('@/views/DriversView.vue'), meta: { title: '司机', perms: ['driver:read'] } },
      { path: 'drivers/:id', name: 'driver-detail', component: () => import('@/views/DriverDetailView.vue'), meta: { title: '司机详情', perms: ['driver:read'] } },
      { path: 'trips', name: 'trips', component: () => import('@/views/TripsView.vue'), meta: { title: '出车任务', perms: ['trip:read'] } },
      { path: 'trips/:id', name: 'trip-detail', component: () => import('@/views/TripDetailView.vue'), meta: { title: '任务详情', perms: ['trip:read'] } },
      { path: 'fuel-records', name: 'fuel-records', component: () => import('@/views/FuelRecordsView.vue'), meta: { title: '油耗记录', perms: ['fuel:read'] } },
      { path: 'maintenance', name: 'maintenance', component: () => import('@/views/MaintenanceView.vue'), meta: { title: '维保', perms: ['maintenance:read'] } },
      { path: 'maintenance/orders/:id', name: 'maintenance-order-detail', component: () => import('@/views/MaintenanceOrderDetailView.vue'), meta: { title: '维保工单详情', perms: ['maintenance:read'] } },
      { path: 'parts', name: 'parts', component: () => import('@/views/PartsView.vue'), meta: { title: '配件', perms: ['part:read'] } },
      { path: 'parts/:id', name: 'part-detail', component: () => import('@/views/PartDetailView.vue'), meta: { title: '配件详情', perms: ['part:read'] } },
      { path: 'reminders', name: 'reminders', component: () => import('@/views/RemindersView.vue'), meta: { title: '到期提醒', perms: ['reminder:read'] } },
      { path: 'reports', name: 'reports', component: () => import('@/views/ReportsView.vue'), meta: { title: '报表', perms: ['report:read'] } },
      { path: 'audit-logs', name: 'audit-logs', component: () => import('@/views/AuditLogsView.vue'), meta: { title: '审计日志', perms: ['audit:read'] } },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.meta.guest) {
    if (auth.isLoggedIn) return { name: 'dashboard' }
    return true
  }
  if (to.meta.requiresAuth) {
    if (!auth.isLoggedIn) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
    // 首次进入时拉取 /api/me 以同步权限。
    if (!auth.meLoaded) {
      try {
        await auth.fetchMe()
      } catch {
        auth.clear()
        return { name: 'login' }
      }
    }
    const perms = to.meta.perms
    if (perms && perms.length) {
      const allowed = perms.some((p) => auth.permissions.includes(p))
      if (!allowed) {
        return { name: 'dashboard' }
      }
    }
  }
  return true
})

export default router
