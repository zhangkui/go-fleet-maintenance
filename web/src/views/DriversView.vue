<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatDate, toRFC3339 } from '@/utils/format'
import type { Driver } from '@/types'
import Pagination from '@/components/Pagination.vue'
import Modal from '@/components/Modal.vue'
import StatusBadge from '@/components/StatusBadge.vue'

const router = useRouter()
const auth = useAuthStore()

const items = ref<Driver[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const keyword = ref('')
const statusFilter = ref('')
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, unknown> = { limit: limit.value, offset: (page.value - 1) * limit.value }
    if (keyword.value) params.keyword = keyword.value
    if (statusFilter.value) params.status = statusFilter.value
    const data = await api.listDrivers(params as never)
    items.value = data.items
    total.value = data.total
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

function onSearch() { page.value = 1; load() }

const showCreate = ref(false)
const form = ref<Partial<Driver>>({ name: '', license_number: '', license_class: '', phone: '', status: 'active', license_expiry: undefined })
const formLocal = ref({ license_expiry: '' })
const saving = ref(false)

function openCreate() {
  form.value = { name: '', license_number: '', license_class: '', phone: '', status: 'active' }
  formLocal.value = { license_expiry: '' }
  showCreate.value = true
}

async function submitCreate() {
  saving.value = true
  try {
    const body: Record<string, unknown> = { ...form.value }
    if (formLocal.value.license_expiry) body.license_expiry = toRFC3339(formLocal.value.license_expiry)
    await api.createDriver(body as never)
    showCreate.value = false
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    saving.value = false
  }
}

const statusTarget = ref<Driver | null>(null)
const statusForm = ref({ status: '' })
async function submitStatus() {
  if (!statusTarget.value) return
  try {
    await api.updateDriverStatus(statusTarget.value.id, statusForm.value.status)
    statusTarget.value = null
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

const statusOptions = [
  { value: 'active', label: '活跃' },
  { value: 'suspended', label: '停用' },
  { value: 'resigned', label: '离职' },
]

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <input v-model="keyword" placeholder="搜索姓名/驾驶证号" @keyup.enter="onSearch" class="input" />
      <select v-model="statusFilter" class="input" @change="onSearch">
        <option value="">全部状态</option>
        <option v-for="s in statusOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
      </select>
      <button class="btn btn-sm" @click="onSearch">搜索</button>
      <button class="btn btn-sm" @click="load">刷新</button>
      <button v-if="auth.hasPermission('driver:create')" class="btn btn-primary btn-sm" @click="openCreate">+ 新增司机</button>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="card">
      <table class="table">
        <thead>
          <tr><th>ID</th><th>姓名</th><th>驾驶证号</th><th>准驾类型</th><th>电话</th><th>状态</th><th>驾照到期</th><th>操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="d in items" :key="d.id" class="clickable" @click="router.push(`/drivers/${d.id}`)">
            <td>{{ d.id }}</td>
            <td>{{ d.name }}</td>
            <td class="mono">{{ d.license_number }}</td>
            <td>{{ d.license_class }}</td>
            <td>{{ d.phone }}</td>
            <td><StatusBadge :status="d.status" /></td>
            <td>{{ formatDate(d.license_expiry) }}</td>
            <td class="actions" @click.stop>
              <button v-if="auth.hasPermission('driver:update')" class="btn btn-sm" @click="statusTarget = d; statusForm = { status: '' }">状态</button>
            </td>
          </tr>
          <tr v-if="!items.length && !loading"><td colspan="8" class="empty">暂无司机</td></tr>
        </tbody>
      </table>
    </div>

    <Pagination :total="total" :limit="limit" :page="page" @page="(p) => { page = p; load() }" @size="(s) => { limit = s; page = 1; load() }" />

    <Modal :show="showCreate" title="新增司机" @close="showCreate = false">
      <div class="form-grid-2">
        <div class="form-group"><label>姓名</label><input v-model="form.name" class="input" /></div>
        <div class="form-group"><label>驾驶证号</label><input v-model="form.license_number" class="input" /></div>
        <div class="form-group"><label>准驾类型</label>
          <select v-model="form.license_class" class="input">
            <option value="">请选择</option>
            <option value="A1">A1</option>
            <option value="A2">A2</option>
            <option value="B1">B1</option>
            <option value="B2">B2</option>
            <option value="C1">C1</option>
          </select>
        </div>
        <div class="form-group"><label>电话</label><input v-model="form.phone" class="input" /></div>
        <div class="form-group"><label>驾照到期</label><input v-model="formLocal.license_expiry" type="date" class="input" /></div>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="showCreate = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="submitCreate">{{ saving ? '保存中...' : '保存' }}</button>
      </template>
    </Modal>

    <Modal :show="!!statusTarget" title="变更司机状态" @close="statusTarget = null">
      <p>司机：<strong>{{ statusTarget?.name }}</strong></p>
      <div class="form-group"><label>新状态</label>
        <select v-model="statusForm.status" class="input">
          <option value="" disabled>选择状态</option>
          <option v-for="s in statusOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
        </select>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="statusTarget = null">取消</button>
        <button class="btn btn-primary btn-sm" @click="submitStatus">确认</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; }
.form-grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.actions { display: flex; gap: 6px; }
.mono { font-family: monospace; font-size: 12px; }
.clickable { cursor: pointer; }
.clickable:hover { background: #f9fafb; }
</style>
