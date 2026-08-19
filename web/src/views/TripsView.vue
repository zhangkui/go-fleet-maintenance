<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatKm, formatDateTime, toRFC3339 } from '@/utils/format'
import type { Trip, Vehicle, Driver } from '@/types'
import Pagination from '@/components/Pagination.vue'
import Modal from '@/components/Modal.vue'
import StatusBadge from '@/components/StatusBadge.vue'

const router = useRouter()
const auth = useAuthStore()

const items = ref<Trip[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const statusFilter = ref('')
const loading = ref(false)
const error = ref('')

// 车辆/司机下拉选项
const vehicles = ref<Vehicle[]>([])
const drivers = ref<Driver[]>([])
async function loadOptions() {
  try {
    const [v, d] = await Promise.all([
      api.listVehicles({ limit: 100 }),
      api.listDrivers({ limit: 100 }),
    ])
    vehicles.value = v.items
    drivers.value = d.items
  } catch { /* 忽略选项加载失败 */ }
}
function vehicleLabel(id: number) {
  const x = vehicles.value.find((v) => v.id === id)
  return x ? `${x.plate_number}（${x.model}）` : `#${id}`
}
function driverLabel(id: number) {
  const x = drivers.value.find((d) => d.id === id)
  return x ? `${x.name}（${x.license_number}）` : `#${id}`
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, unknown> = { limit: limit.value, offset: (page.value - 1) * limit.value }
    if (statusFilter.value) params.status = statusFilter.value
    const data = await api.listTrips(params as never)
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
const form = ref({ vehicle_id: 0 as number | '', driver_id: 0 as number | '', route: '', load_kg: 0, start_odometer_km: 0, note: '' })
const saving = ref(false)

async function submitCreate() {
  saving.value = true
  try {
    await api.createTrip({
      vehicle_id: Number(form.value.vehicle_id),
      driver_id: Number(form.value.driver_id),
      route: form.value.route,
      load_kg: Number(form.value.load_kg),
      start_odometer_km: Number(form.value.start_odometer_km),
      idempotency_key: crypto.randomUUID(),
      note: form.value.note,
    } as never)
    showCreate.value = false
    form.value = { vehicle_id: '', driver_id: '', route: '', load_kg: 0, start_odometer_km: 0, note: '' }
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    saving.value = false
  }
}

async function action(t: Trip, kind: 'start' | 'cancel') {
  if (!confirm(`确认${kind === 'start' ? '开始' : '取消'}该任务？`)) return
  try {
    if (kind === 'start') await api.startTrip(t.id)
    else await api.cancelTrip(t.id)
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

// 完成
const completeTarget = ref<Trip | null>(null)
const completeForm = ref({ end_odometer_km: 0, note: '' })
async function submitComplete() {
  if (!completeTarget.value) return
  try {
    await api.completeTrip(completeTarget.value.id, {
      end_odometer_km: Number(completeForm.value.end_odometer_km),
      completed_at: toRFC3339(new Date().toISOString().slice(0, 16)),
      note: completeForm.value.note,
    })
    completeTarget.value = null
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

const statusOptions = [
  { value: 'scheduled', label: '已排程' },
  { value: 'in_progress', label: '进行中' },
  { value: 'completed', label: '已完成' },
  { value: 'cancelled', label: '已取消' },
]

onMounted(() => { loadOptions(); load() })
</script>

<template>
  <div>
    <div class="toolbar">
      <select v-model="statusFilter" class="input" @change="onSearch">
        <option value="">全部状态</option>
        <option v-for="s in statusOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
      </select>
      <button class="btn btn-sm" @click="load">刷新</button>
      <button v-if="auth.hasPermission('trip:create')" class="btn btn-primary btn-sm" @click="showCreate = true">+ 新建任务</button>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="card">
      <table class="table">
        <thead>
          <tr><th>ID</th><th>车辆</th><th>司机</th><th>路线</th><th>状态</th><th>起始里程</th><th>结束里程</th><th>创建时间</th><th>操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="t in items" :key="t.id" class="clickable" @click="router.push(`/trips/${t.id}`)">
            <td>{{ t.id }}</td>
            <td>{{ vehicleLabel(t.vehicle_id) }}</td>
            <td>{{ driverLabel(t.driver_id) }}</td>
            <td>{{ t.route }}</td>
            <td><StatusBadge :status="t.status" /></td>
            <td>{{ formatKm(t.start_odometer_km) }}</td>
            <td>{{ formatKm(t.end_odometer_km) }}</td>
            <td>{{ formatDateTime(t.created_at) }}</td>
            <td class="actions" @click.stop>
              <button v-if="auth.hasPermission('trip:update') && t.status === 'scheduled'" class="btn btn-sm" @click="action(t, 'start')">开始</button>
              <button v-if="auth.hasPermission('trip:update') && t.status === 'in_progress'" class="btn btn-sm" @click="completeTarget = t; completeForm = { end_odometer_km: 0, note: '' }">完成</button>
              <button v-if="auth.hasPermission('trip:update') && (t.status === 'scheduled' || t.status === 'in_progress')" class="btn btn-sm btn-danger-outline" @click="action(t, 'cancel')">取消</button>
            </td>
          </tr>
          <tr v-if="!items.length && !loading"><td colspan="9" class="empty">暂无任务</td></tr>
        </tbody>
      </table>
    </div>

    <Pagination :total="total" :limit="limit" :page="page" @page="(p) => { page = p; load() }" @size="(s) => { limit = s; page = 1; load() }" />

    <Modal :show="showCreate" title="新建出车任务" wide @close="showCreate = false">
      <div class="form-grid-2">
        <div class="form-group"><label>车辆</label>
          <select v-model="form.vehicle_id" class="input">
            <option :value="''">请选择车辆</option>
            <option v-for="v in vehicles" :key="v.id" :value="v.id">{{ v.plate_number }}（{{ v.model }}）</option>
          </select>
        </div>
        <div class="form-group"><label>司机</label>
          <select v-model="form.driver_id" class="input">
            <option :value="''">请选择司机</option>
            <option v-for="d in drivers" :key="d.id" :value="d.id">{{ d.name }}（{{ d.license_number }}）</option>
          </select>
        </div>
        <div class="form-group"><label>路线</label><input v-model="form.route" class="input" /></div>
        <div class="form-group"><label>载重(kg)</label><input v-model.number="form.load_kg" type="number" class="input" /></div>
        <div class="form-group"><label>起始里程(km)</label><input v-model.number="form.start_odometer_km" type="number" class="input" /></div>
        <div class="form-group" style="grid-column: span 2"><label>备注</label><textarea v-model="form.note" class="input" rows="2"></textarea></div>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="showCreate = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="submitCreate">{{ saving ? '保存中...' : '创建' }}</button>
      </template>
    </Modal>

    <Modal :show="!!completeTarget" title="完成任务" @close="completeTarget = null">
      <p>任务 #{{ completeTarget?.id }}（{{ completeTarget ? vehicleLabel(completeTarget.vehicle_id) : '' }}）</p>
      <div class="form-group"><label>结束里程(km)</label><input v-model.number="completeForm.end_odometer_km" type="number" class="input" /></div>
      <div class="form-group"><label>备注</label><textarea v-model="completeForm.note" class="input" rows="2"></textarea></div>
      <template #footer>
        <button class="btn btn-sm" @click="completeTarget = null">取消</button>
        <button class="btn btn-primary btn-sm" @click="submitComplete">确认完成</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; }
.actions { display: flex; gap: 6px; }
.clickable { cursor: pointer; }
.clickable:hover { background: #f9fafb; }
.form-grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
</style>
