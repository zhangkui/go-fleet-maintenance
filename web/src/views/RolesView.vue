<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { Role, Permission } from '@/types'
import Modal from '@/components/Modal.vue'

const auth = useAuthStore()

const roles = ref<Role[]>([])
const permissions = ref<Permission[]>([])
const selectedRoleId = ref<number | null>(null)
const rolePerms = ref<Permission[]>([])
const loading = ref(false)
const error = ref('')
const permInput = ref<number | null>(null)

// 创建角色
const showCreate = ref(false)
const createForm = ref({ name: '', code: '', description: '' })
const saving = ref(false)
async function submitCreate() {
  saving.value = true
  try {
    await api.createRole({
      name: createForm.value.name,
      code: createForm.value.code,
      description: createForm.value.description,
    })
    showCreate.value = false
    createForm.value = { name: '', code: '', description: '' }
    await loadRoles()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    saving.value = false
  }
}

async function loadRoles() {
  try {
    roles.value = await api.listRoles()
    if (roles.value.length && selectedRoleId.value === null) {
      selectedRoleId.value = roles.value[0].id
    }
  } catch (e) {
    error.value = extractErrorMessage(e)
  }
}

async function loadPerms() {
  try {
    permissions.value = await api.listPermissions()
  } catch (e) {
    error.value = extractErrorMessage(e)
  }
}

async function loadRolePerms(id: number) {
  try {
    rolePerms.value = await api.rolePermissions(id)
  } catch (e) {
    error.value = extractErrorMessage(e)
  }
}

watch(selectedRoleId, (v) => {
  if (v !== null) loadRolePerms(v)
})

async function grant() {
  if (!selectedRoleId.value || !permInput.value) return
  try {
    await api.grantPermission(selectedRoleId.value, permInput.value)
    await loadRolePerms(selectedRoleId.value)
    permInput.value = null
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

onMounted(() => {
  loadRoles()
  loadPerms()
})
</script>

<template>
  <div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div class="roles-grid">
      <div class="card">
        <div class="card-head">
          <h3 class="card-title">角色列表</h3>
          <button v-if="auth.hasPermission('role:manage')" class="btn btn-primary btn-sm" @click="showCreate = true">+ 新建角色</button>
        </div>
        <div class="role-select">
          <div
            v-for="r in roles"
            :key="r.id"
            class="role-row"
            :class="{ active: selectedRoleId === r.id }"
            @click="selectedRoleId = r.id"
          >
            <div class="role-row-name">{{ r.name }}</div>
            <div class="role-row-code">{{ r.code }}</div>
            <div class="role-row-desc">{{ r.description }}</div>
          </div>
        </div>
      </div>

      <div class="card">
        <h3 class="card-title">角色权限</h3>
        <div v-if="loading" class="loading">加载中...</div>
        <table v-else class="table">
          <thead>
            <tr><th>权限码</th><th>名称</th><th>资源</th><th>动作</th></tr>
          </thead>
          <tbody>
            <tr v-for="p in rolePerms" :key="p.id">
              <td><code>{{ p.code }}</code></td>
              <td>{{ p.name }}</td>
              <td>{{ p.resource }}</td>
              <td>{{ p.action }}</td>
            </tr>
            <tr v-if="!rolePerms.length"><td colspan="4" class="empty">该角色暂无权限</td></tr>
          </tbody>
        </table>
        <div v-if="auth.hasPermission('role:manage')" class="grant-bar">
          <select v-model="permInput" class="input">
            <option :value="null" disabled>选择权限授予</option>
            <option v-for="p in permissions" :key="p.id" :value="p.id">{{ p.code }} ({{ p.name }})</option>
          </select>
          <button class="btn btn-primary btn-sm" :disabled="!permInput || !selectedRoleId" @click="grant">授予权限</button>
        </div>
      </div>
    </div>

    <div class="card" style="margin-top: 20px">
      <h3 class="card-title">全部权限定义</h3>
      <table class="table">
        <thead>
          <tr><th>权限码</th><th>名称</th><th>资源</th><th>动作</th></tr>
        </thead>
        <tbody>
          <tr v-for="p in permissions" :key="p.id">
            <td><code>{{ p.code }}</code></td>
            <td>{{ p.name }}</td>
            <td>{{ p.resource }}</td>
            <td>{{ p.action }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <Modal :show="showCreate" title="新建角色" @close="showCreate = false">
      <div class="form-group"><label>角色名称</label><input v-model="createForm.name" class="input" placeholder="如：车队队长" /></div>
      <div class="form-group"><label>角色编码</label><input v-model="createForm.code" class="input" placeholder="如：fleet_leader（英文）" /></div>
      <div class="form-group"><label>描述</label><input v-model="createForm.description" class="input" placeholder="角色职责说明" /></div>
      <template #footer>
        <button class="btn btn-sm" @click="showCreate = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="submitCreate">{{ saving ? '保存中...' : '创建' }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.roles-grid {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 20px;
  align-items: start;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.role-select {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.role-row {
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}
.role-row:hover {
  border-color: var(--primary);
}
.role-row.active {
  border-color: var(--primary);
  background: #eff6ff;
}
.role-row-name {
  font-weight: 600;
}
.role-row-code {
  font-size: 12px;
  color: var(--muted);
}
.role-row-desc {
  font-size: 12px;
  color: var(--muted);
  margin-top: 4px;
}
.grant-bar {
  display: flex;
  gap: 8px;
  margin-top: 16px;
  align-items: center;
}
@media (max-width: 768px) {
  .roles-grid {
    grid-template-columns: 1fr;
  }
}
</style>
