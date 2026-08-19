<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatKm, formatDate, toRFC3339 } from '@/utils/format'
import type { Vehicle } from '@/types'
import Pagination from '@/components/Pagination.vue'
import Modal from '@/components/Modal.vue'
import StatusBadge from '@/components/StatusBadge.vue'

const router = useRouter()
const auth = useAuthStore()

const items = ref<Vehicle[]>([])
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
    const data = await api.listVehicles(params as never)
    items.value = data.items
    total.value = data.total
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

function onSearch() { page.value = 1; load() }

// 创建车辆
const showCreate = ref(false)
const form = ref<Partial<Vehicle>>({
  model: '', vin: '', plate_number: '', status: 'active', odometer_km: 0,
  color: '', engine_no: '', purchase_date: undefined, insurance_expiry: undefined, inspection_expiry: undefined,
})
const formLocal = ref({ purchase_date: '', insurance_expiry: '', inspection_expiry: '' })
const saving = ref(false)

function openCreate() {
  form.value = { model: '', vin: '', plate_number: '', status: 'active', odometer_km: 0, color: '', engine_no: '' }
  formLocal.value = { purchase_date: '', insurance_expiry: '', inspection_expiry: '' }
  showCreate.value = true
}

async function submitCreate() {
  saving.value = true
  try {
    const body: Record<string, unknown> = { ...form.value }
    if (formLocal.value.purchase_date) body.purchase_date = toRFC3339(formLocal.value.purchase_date)
    if (formLocal.value.insurance_expiry) body.insurance_expiry = toRFC3339(formLocal.value.insurance_expiry)
    if (formLocal.value.inspection_expiry) body.inspection_expiry = toRFC3339(formLocal.value.inspection_expiry)
    await api.createVehicle(body as never)
    showCreate.value = false
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    saving.value = false
  }
}

// 状态变更
const statusTarget = ref<Vehicle | null>(null)
const statusForm = ref({ status: '', reason: '' })
async function submitStatus() {
  if (!statusTarget.value) return
  try {
    await api.changeVehicleStatus(statusTarget.value.id, statusForm.value.status, statusForm.value.reason)
    statusTarget.value = null
    statusForm.value = { status: '', reason: '' }
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

// 里程更新
const mileageTarget = ref<Vehicle | null>(null)
const mileageForm = ref({ odometer_km: 0 })
async function submitMileage() {
  if (!mileageTarget.value) return
  try {
    await api.updateVehicleMileage(mileageTarget.value.id, mileageForm.value.odometer_km)
    mileageTarget.value = null
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

function openMileage(v: Vehicle) {
  mileageTarget.value = v
  mileageForm.value.odometer_km = v.odometer_km
}

const statusOptions = [
  { value: 'active', label: '在用' },
  { value: 'in_maintenance', label: '维保中' },
  { value: 'retired', label: '报废' },
]

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <input v-model="keyword" placeholder="搜索 VIN/车牌/型号" @keyup.enter="onSearch" class="input" />
      <select v-model="statusFilter" class="input" @change="onSearch">
        <option value="">全部状态</option>
        <option v-for="s in statusOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
      </select>
      <button class="btn btn-sm" @click="onSearch">搜索</button>
      <button class="btn btn-sm" @click="load">刷新</button>
      <button v-if="auth.hasPermission('vehicle:create')" class="btn btn-primary btn-sm" @click="openCreate">+ 新增车辆</button>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="card">
      <table class="table">
        <thead>
          <tr>
            <th>ID</th><th>型号</th><th>VIN</th><th>车牌号</th><th>状态</th>
            <th>里程</th><th>保险到期</th><th>年检到期</th><th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="v in items" :key="v.id" class="clickable" @click="router.push(`/vehicles/${v.id}`)">
            <td>{{ v.id }}</td>
            <td>{{ v.model }}</td>
            <td class="mono">{{ v.vin }}</td>
            <td>{{ v.plate_number }}</td>
            <td><StatusBadge :status="v.status" /></td>
            <td>{{ formatKm(v.odometer_km) }}</td>
            <td>{{ formatDate(v.insurance_expiry) }}</td>
            <td>{{ formatDate(v.inspection_expiry) }}</td>
            <td class="actions" @click.stop>
              <button v-if="auth.hasPermission('vehicle:status')" class="btn btn-sm" @click="statusTarget = v; statusForm = { status: '', reason: '' }">状态</button>
              <button v-if="auth.hasPermission('vehicle:update')" class="btn btn-sm" @click="openMileage(v)">里程</button>
            </td>
          </tr>
          <tr v-if="!items.length && !loading"><td colspan="9" class="empty">暂无车辆</td></tr>
        </tbody>
      </table>
    </div>

    <Pagination :total="total" :limit="limit" :page="page" @page="(p) => { page = p; load() }" @size="(s) => { limit = s; page = 1; load() }" />

    <Modal :show="showCreate" title="新增车辆" wide @close="showCreate = false">
      <div class="form-grid-2">
        <div class="form-group"><label>型号</label><input v-model="form.model" class="input" /></div>
        <div class="form-group"><label>VIN</label><input v-model="form.vin" class="input" /></div>
        <div class="form-group"><label>车牌号</label><input v-model="form.plate_number" class="input" /></div>
        <div class="form-group"><label>颜色</label><input v-model="form.color" class="input" /></div>
        <div class="form-group"><label>发动机号</label><input v-model="form.engine_no" class="input" /></div>
        <div class="form-group"><label>初始里程(km)</label><input v-model.number="form.odometer_km" type="number" class="input" /></div>
        <div class="form-group"><label>购置日期</label><input v-model="formLocal.purchase_date" type="date" class="input" /></div>
        <div class="form-group"><label>保险到期</label><input v-model="formLocal.insurance_expiry" type="date" class="input" /></div>
        <div class="form-group"><label>年检到期</label><input v-model="formLocal.inspection_expiry" type="date" class="input" /></div>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="showCreate = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="submitCreate">{{ saving ? '保存中...' : '保存' }}</button>
      </template>
    </Modal>

    <Modal :show="!!statusTarget" title="变更车辆状态" @close="statusTarget = null">
      <p>车辆：<strong>{{ statusTarget?.plate_number }}</strong> 当前状态：<StatusBadge :status="statusTarget?.status ?? ''" /></p>
      <div class="form-group"><label>新状态</label>
        <select v-model="statusForm.status" class="input">
          <option value="" disabled>选择状态</option>
          <option v-for="s in statusOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
        </select>
      </div>
      <div class="form-group"><label>原因</label><textarea v-model="statusForm.reason" class="input" rows="3"></textarea></div>
      <template #footer>
        <button class="btn btn-sm" @click="statusTarget = null">取消</button>
        <button class="btn btn-primary btn-sm" @click="submitStatus">确认变更</button>
      </template>
    </Modal>

    <Modal :show="!!mileageTarget" title="更新里程" @close="mileageTarget = null">
      <p>车辆：<strong>{{ mileageTarget?.plate_number }}</strong> 当前里程：{{ formatKm(mileageTarget?.odometer_km) }}</p>
      <div class="form-group"><label>新里程(km)</label><input v-model.number="mileageForm.odometer_km" type="number" class="input" /></div>
      <template #footer>
        <button class="btn btn-sm" @click="mileageTarget = null">取消</button>
        <button class="btn btn-primary btn-sm" @click="submitMileage">确认更新</button>
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
