import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import client from '@/api/client'
import type { AuthTokens, MeResponse, User } from '@/types'

const STORAGE_KEY = 'fleet_auth'

interface StoredAuth {
  access_token: string
  refresh_token: string
  expires_at: string
  user: User
  permissions: string[]
}

function loadStored(): StoredAuth | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    return JSON.parse(raw) as StoredAuth
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const stored = loadStored()

  const accessToken = ref<string>(stored?.access_token ?? '')
  const refreshToken = ref<string>(stored?.refresh_token ?? '')
  const expiresAt = ref<string>(stored?.expires_at ?? '')
  const user = ref<User | null>(stored?.user ?? null)
  const permissions = ref<string[]>(stored?.permissions ?? [])
  // /api/me 返回的角色码集合。
  const roles = ref<string[]>([])
  const meLoaded = ref(false)

  const isLoggedIn = computed(() => !!accessToken.value)
  const displayName = computed(() => user.value?.full_name || user.value?.username || '')

  function persist(t: AuthTokens) {
    accessToken.value = t.access_token
    refreshToken.value = t.refresh_token
    expiresAt.value = t.expires_at
    user.value = t.user
    permissions.value = t.permissions
    roles.value = []
    meLoaded.value = false
    const data: StoredAuth = {
      access_token: t.access_token,
      refresh_token: t.refresh_token,
      expires_at: t.expires_at,
      user: t.user,
      permissions: t.permissions,
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
  }

  function persistMe(m: MeResponse) {
    if (user.value) {
      user.value = { ...user.value, id: m.id, username: m.username, email: m.email, full_name: m.full_name, status: m.status }
    } else {
      user.value = {
        id: m.id, username: m.username, email: m.email, full_name: m.full_name, status: m.status,
        created_at: '', updated_at: '',
      }
    }
    permissions.value = m.permissions
    roles.value = m.roles
    meLoaded.value = true
    // 更新本地存储中的 user 与 permissions。
    const existing = loadStored()
    if (existing) {
      existing.user = user.value
      existing.permissions = m.permissions
      localStorage.setItem(STORAGE_KEY, JSON.stringify(existing))
    }
  }

  function clear() {
    accessToken.value = ''
    refreshToken.value = ''
    expiresAt.value = ''
    user.value = null
    permissions.value = []
    roles.value = []
    meLoaded.value = false
    localStorage.removeItem(STORAGE_KEY)
  }

  function hasPermission(code: string): boolean {
    return permissions.value.includes(code)
  }

  function hasAny(codes: string[]): boolean {
    return codes.some((c) => permissions.value.includes(c))
  }

  async function login(username: string, password: string) {
    const { data } = await client.post<AuthTokens>('/api/auth/login', { username, password })
    persist(data)
    return data
  }

  async function refresh(): Promise<string> {
    if (!refreshToken.value) throw new Error('no refresh token')
    const { data } = await client.post<AuthTokens>(
      '/api/auth/refresh',
      { refresh_token: refreshToken.value },
      { _refresh: true } as never,
    )
    persist(data)
    return data.access_token
  }

  async function logout() {
    try {
      await client.post('/api/auth/logout', { refresh_token: refreshToken.value })
    } catch {
      // 忽略登出失败。
    }
    clear()
  }

  async function fetchMe() {
    const { data } = await client.get<MeResponse>('/api/me')
    persistMe(data)
    return data
  }

  return {
    accessToken,
    refreshToken,
    expiresAt,
    user,
    permissions,
    roles,
    meLoaded,
    isLoggedIn,
    displayName,
    persist,
    persistMe,
    clear,
    hasPermission,
    hasAny,
    login,
    refresh,
    logout,
    fetchMe,
  }
})
