<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/format'
import type { User, Role } from '@/types'
import Pagination from '@/components/Pagination.vue'
import Modal from '@/components/Modal.vue'
import StatusBadge from '@/components/StatusBadge.vue'

const auth = useAuthStore()

const items = ref<User[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const keyword = ref('')
const loading = ref(false)
const error = ref('')

const rolesMap = ref<Map<number, Role>>(new Map())

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, unknown> = { limit: limit.value, offset: (page.value - 1) * limit.value }
    if (keyword.value) params.keyword = keyword.value
    const data = await api.listUsers(params as never)
    items.value = data.items
    total.value = data.total
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

const userRoles = ref<Record<number, Role[]>>({})
async function loadUserRoles(userId: number) {
  try {
    const roles = await api.userRoles(userId)
    userRoles.value[userId] = roles
  } catch {
    userRoles.value[userId] = []
  }
}

async function toggleUser(u: User) {
  const enable = u.status !== 'active'
  if (!confirm(`确认${enable ? '启用' : '禁用'}用户 ${u.username}？`)) return
  try {
    await api.toggleUser(u.id, enable)
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

// 重置密码
const resetTarget = ref<User | null>(null)
const newPassword = ref('')
async function submitReset() {
  if (!resetTarget.value) return
  try {
    await api.resetPassword(resetTarget.value.id, newPassword.value)
    alert('密码已重置')
    resetTarget.value = null
    newPassword.value = ''
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

// 角色分配
const roleTarget = ref<User | null>(null)
const allRoles = ref<Role[]>([])
const selectedRoleId = ref<number | null>(null)
const targetRoles = ref<Role[]>([])

async function openRoles(u: User) {
  roleTarget.value = u
  if (allRoles.value.length === 0) {
    allRoles.value = await api.listRoles()
  }
  targetRoles.value = await api.userRoles(u.id)
  selectedRoleId.value = null
}

async function assignRole() {
  if (!roleTarget.value || !selectedRoleId.value) return
  try {
    await api.assignRole(roleTarget.value.id, selectedRoleId.value)
    targetRoles.value = await api.userRoles(roleTarget.value.id)
    selectedRoleId.value = null
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

async function revokeRole(roleId: number) {
  if (!roleTarget.value) return
  if (!confirm('确认移除该角色？')) return
  try {
    await api.revokeRole(roleTarget.value.id, roleId)
    targetRoles.value = await api.userRoles(roleTarget.value.id)
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

function onSearch() {
  page.value = 1
  load()
}

onMounted(async () => {
  await load()
  if (auth.hasPermission('role:read')) {
    const roles = await api.listRoles()
    rolesMap.value = new Map(roles.map((r) => [r.id, r]))
    items.value.forEach((u) => loadUserRoles(u.id))
  }
})
</script>

<template>
  <div>
    <div class="toolbar">
      <input v-model="keyword" placeholder="搜索用户名/姓名" @keyup.enter="onSearch" class="input" />
      <button class="btn btn-sm" @click="onSearch">搜索</button>
      <button class="btn btn-sm" @click="load">刷新</button>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="card">
      <table class="table">
        <thead>
          <tr>
            <th>ID</th><th>用户名</th><th>姓名</th><th>邮箱</th><th>状态</th>
            <th>角色</th><th>最近登录</th><th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in items" :key="u.id">
            <td>{{ u.id }}</td>
            <td>{{ u.username }}</td>
            <td>{{ u.full_name }}</td>
            <td>{{ u.email }}</td>
            <td><StatusBadge :status="u.status" /></td>
            <td>
              <span v-if="userRoles[u.id]?.length">
                {{ userRoles[u.id].map(r => r.code).join('、') }}
              </span>
              <span v-else class="text-muted">-</span>
            </td>
            <td>{{ formatDateTime(u.last_login_at) }}</td>
            <td class="actions">
              <template v-if="auth.hasPermission('user:manage')">
                <button class="btn btn-sm" @click="toggleUser(u)">
                  {{ u.status === 'active' ? '禁用' : '启用' }}
                </button>
                <button class="btn btn-sm" @click="resetTarget = u">重置密码</button>
              </template>
              <template v-if="auth.hasPermission('role:manage')">
                <button class="btn btn-sm" @click="openRoles(u)">角色</button>
              </template>
            </td>
          </tr>
          <tr v-if="!items.length && !loading">
            <td colspan="8" class="empty">暂无数据</td>
          </tr>
        </tbody>
      </table>
    </div>

    <Pagination :total="total" :limit="limit" :page="page" @page="(p) => { page = p; load() }" @size="(s) => { limit = s; page = 1; load() }" />

    <Modal :show="!!resetTarget" title="重置密码" @close="resetTarget = null">
      <p>为用户 <strong>{{ resetTarget?.username }}</strong> 重置密码：</p>
      <div class="form-group">
        <label>新密码</label>
        <input v-model="newPassword" type="password" required />
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="resetTarget = null">取消</button>
        <button class="btn btn-primary btn-sm" @click="submitReset" :disabled="!newPassword">确认重置</button>
      </template>
    </Modal>

    <Modal :show="!!roleTarget" title="分配角色" @close="roleTarget = null">
      <p>用户 <strong>{{ roleTarget?.username }}</strong> 的角色：</p>
      <div class="role-list">
        <div v-for="r in targetRoles" :key="r.id" class="role-item">
          <span>{{ r.name }} ({{ r.code }})</span>
          <button class="btn btn-sm btn-danger-outline" @click="revokeRole(r.id)">移除</button>
        </div>
        <div v-if="!targetRoles.length" class="text-muted">暂无角色</div>
      </div>
      <div class="form-group" style="margin-top: 16px">
        <label>分配新角色</label>
        <select v-model="selectedRoleId" class="input">
          <option :value="null" disabled>选择角色</option>
          <option v-for="r in allRoles" :key="r.id" :value="r.id">{{ r.name }} ({{ r.code }})</option>
        </select>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="roleTarget = null">关闭</button>
        <button class="btn btn-primary btn-sm" @click="assignRole" :disabled="!selectedRoleId">分配</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.role-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.role-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: var(--bg);
  border-radius: 8px;
}
.text-muted {
  color: var(--muted);
}
</style>
