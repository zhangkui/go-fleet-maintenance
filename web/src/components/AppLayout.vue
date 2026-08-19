<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute, RouterLink, RouterView } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const sidebarOpen = ref(false)

interface NavItem {
  to: string
  label: string
  icon: string
  perms: string[]
}

const navGroups: { title: string; items: NavItem[] }[] = [
  {
    title: '总览',
    items: [
      { to: '/', label: '仪表盘', icon: '📊', perms: ['report:read'] },
      { to: '/reminders', label: '到期提醒', icon: '🔔', perms: ['reminder:read'] },
      { to: '/reports', label: '报表', icon: '📈', perms: ['report:read'] },
    ],
  },
  {
    title: '运营',
    items: [
      { to: '/vehicles', label: '车辆', icon: '🚛', perms: ['vehicle:read'] },
      { to: '/drivers', label: '司机', icon: '👤', perms: ['driver:read'] },
      { to: '/trips', label: '出车任务', icon: '🧭', perms: ['trip:read'] },
      { to: '/fuel-records', label: '油耗记录', icon: '⛽', perms: ['fuel:read'] },
    ],
  },
  {
    title: '维保',
    items: [
      { to: '/maintenance', label: '维保管理', icon: '🔧', perms: ['maintenance:read'] },
      { to: '/parts', label: '配件库存', icon: '📦', perms: ['part:read'] },
    ],
  },
  {
    title: '系统',
    items: [
      { to: '/users', label: '用户管理', icon: '👥', perms: ['user:read'] },
      { to: '/roles', label: '角色权限', icon: '🛡️', perms: ['role:read'] },
      { to: '/audit-logs', label: '审计日志', icon: '📜', perms: ['audit:read'] },
    ],
  },
]

const visibleGroups = computed(() =>
  navGroups
    .map((g) => ({ ...g, items: g.items.filter((i) => i.perms.some((p) => auth.hasPermission(p)) || i.perms.length === 0) }))
    .filter((g) => g.items.length > 0),
)

const pageTitle = computed(() => route.meta.title as string | undefined ?? '')

async function handleLogout() {
  await auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="layout">
    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="brand">
        <span class="brand-icon">🚚</span>
        <div class="brand-text">
          <div class="brand-name">车队维保</div>
          <div class="brand-sub">管理系统</div>
        </div>
      </div>
      <nav class="nav">
        <template v-for="g in visibleGroups" :key="g.title">
          <div class="nav-group-title">{{ g.title }}</div>
          <RouterLink
            v-for="item in g.items"
            :key="item.to"
            :to="item.to"
            class="nav-item"
            @click="sidebarOpen = false"
          >
            <span class="nav-icon">{{ item.icon }}</span>
            <span>{{ item.label }}</span>
          </RouterLink>
        </template>
      </nav>
    </aside>

    <div class="main">
      <header class="topbar">
        <button class="menu-btn" @click="sidebarOpen = !sidebarOpen">☰</button>
        <h1 class="page-title">{{ pageTitle }}</h1>
        <div class="topbar-right">
          <span class="user-name">{{ auth.displayName }}</span>
          <RouterLink to="/profile" class="btn btn-sm">个人资料</RouterLink>
          <button class="btn btn-sm btn-danger-outline" @click="handleLogout">退出</button>
        </div>
      </header>
      <main class="content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  min-height: 100vh;
}
.sidebar {
  width: 230px;
  background: linear-gradient(180deg, #1e293b 0%, #0f172a 100%);
  color: #cbd5e1;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  position: sticky;
  top: 0;
  height: 100vh;
  overflow-y: auto;
  box-shadow: 2px 0 12px rgba(0,0,0,0.15);
}
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 20px;
  border-bottom: 1px solid #334155;
}
.brand-icon {
  font-size: 26px;
}
.brand-name {
  font-size: 17px;
  font-weight: 700;
  color: #fff;
  line-height: 1.2;
}
.brand-sub {
  font-size: 11px;
  color: #64748b;
  margin-top: 2px;
}
.nav {
  padding: 12px 10px;
  flex: 1;
}
.nav-group-title {
  padding: 18px 14px 8px;
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.6px;
  color: #475569;
  font-weight: 700;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 10px 14px;
  border-radius: 9px;
  color: #cbd5e1;
  text-decoration: none;
  font-size: 14px;
  margin-bottom: 2px;
  transition: all 0.15s;
  position: relative;
}
.nav-item:hover {
  background: rgba(255,255,255,0.08);
  color: #fff;
  text-decoration: none;
}
.nav-item.router-link-exact-active {
  background: var(--primary);
  color: #fff;
  box-shadow: 0 2px 8px rgba(79,70,229,0.4);
}
.nav-icon {
  font-size: 17px;
  width: 22px;
  text-align: center;
}
.main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg);
}
.topbar {
  height: 60px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 24px;
  gap: 14px;
  position: sticky;
  top: 0;
  z-index: 100;
  box-shadow: var(--shadow-sm);
}
.page-title {
  font-size: 18px;
  font-weight: 700;
  margin: 0;
  flex: 1;
  color: var(--text);
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}
.user-name {
  font-size: 13px;
  color: var(--muted);
  font-weight: 500;
}
.menu-btn {
  display: none;
  background: none;
  border: none;
  font-size: 22px;
  cursor: pointer;
  color: var(--text);
}
.content {
  padding: 26px;
  max-width: 1440px;
  width: 100%;
}
@media (max-width: 900px) {
  .sidebar {
    position: fixed;
    z-index: 200;
    transform: translateX(-100%);
    transition: transform 0.25s;
  }
  .sidebar.open {
    transform: translateX(0);
  }
  .menu-btn {
    display: block;
  }
}
</style>
