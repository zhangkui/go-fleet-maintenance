<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatKm, formatDateTime, formatDate, toRFC3339 } from '@/utils/format'
import type { Vehicle, VehicleStatusHistory, VehicleLicense, DriverVehicleBinding, Driver } from '@/types'
import Modal from '@/components/Modal.vue'
import StatusBadge from '@/components/StatusBadge.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const vehicle = ref<Vehicle | null>(null)
const history = ref<VehicleStatusHistory[]>([])
const licenses = ref<VehicleLicense[]>([])
const bindings = ref<DriverVehicleBinding[]>([])
const drivers = ref<Driver[]>([])
const loading = ref(false)
const error = ref('')

const id = Number(route.params.id)

function driverLabel(did: number) {
  const d = drivers.value.find((x) => x.id === did)
  return d ? `${d.name}（${d.license_number}）` : `#${did}`
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [v, h, lics, binds] = await Promise.all([
      api.getVehicle(id),
      api.vehicleStatusHistory(id),
      api.listVehicleLicenses(id),
      auth.hasPermission('driver:read') ? api.listBindings(id).catch(() => []) : Promise.resolve([] as DriverVehicleBinding[]),
    ])
    vehicle.value = v
    history.value = h
    licenses.value = lics
    bindings.value = binds
    // 加载司机列表用于绑定表格显示司机名。
    try {
      drivers.value = (await api.listDrivers({ limit: 100 })).items
    } catch { /* 忽略 */ }
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

// 添加证照
const showLicense = ref(false)
const licForm = ref({ kind: 'insurance', number: '', issue_date: '', expiry_date: '' })
const saving = ref(false)

async function submitLicense() {
  saving.value = true
  try {
    const body: Record<string, unknown> = {
      kind: licForm.value.kind,
      number: licForm.value.number,
      expiry_date: toRFC3339(licForm.value.expiry_date),
    }
    if (licForm.value.issue_date) body.issue_date = toRFC3339(licForm.value.issue_date)
    await api.addVehicleLicense(id, body as never)
    showLicense.value = false
    licForm.value = { kind: 'insurance', number: '', issue_date: '', expiry_date: '' }
    licenses.value = await api.listVehicleLicenses(id)
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    saving.value = false
  }
}

const kindOptions = [
  { value: 'insurance', label: '保险' },
  { value: 'inspection', label: '年检' },
  { value: 'road_transport', label: '道路运输证' },
  { value: 'green_book', label: '绿本' },
]

const infoRows: Array<{ key: string; label: string; fmt?: (v: Vehicle) => string }> = [
  { key: 'model', label: '型号' },
  { key: 'vin', label: 'VIN' },
  { key: 'plate_number', label: '车牌号' },
  { key: 'color', label: '颜色' },
  { key: 'engine_no', label: '发动机号' },
  { key: 'odometer_km', label: '里程', fmt: (v: Vehicle) => formatKm(v.odometer_km) },
  { key: 'purchase_date', label: '购置日期', fmt: (v: Vehicle) => formatDate(v.purchase_date) },
  { key: 'insurance_expiry', label: '保险到期', fmt: (v: Vehicle) => formatDate(v.insurance_expiry) },
  { key: 'inspection_expiry', label: '年检到期', fmt: (v: Vehicle) => formatDate(v.inspection_expiry) },
]

function formatRow(row: { key: string; fmt?: (v: Vehicle) => string }, v: Vehicle): string {
  if (row.fmt) return row.fmt(v)
  return String((v as unknown as Record<string, unknown>)[row.key] ?? '')
}

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <button class="btn btn-sm" @click="router.push('/vehicles')">← 返回车辆列表</button>
      <span v-if="vehicle" class="page-subtitle">{{ vehicle.plate_number }} - {{ vehicle.model }}</span>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-if="loading" class="loading">加载中...</div>

    <template v-if="vehicle">
      <div class="card">
        <div class="card-head">
          <h3 class="card-title">车辆信息</h3>
          <StatusBadge :status="vehicle.status" />
        </div>
        <dl class="info-grid">
          <div v-for="row in infoRows" :key="row.key">
            <dt>{{ row.label }}</dt>
            <dd>{{ formatRow(row, vehicle) }}</dd>
          </div>
        </dl>
      </div>

      <div class="grid-2">
        <div class="card">
          <div class="card-head">
            <h3 class="card-title">证照信息</h3>
            <button v-if="auth.hasPermission('vehicle:update')" class="btn btn-sm btn-primary" @click="showLicense = true">+ 添加</button>
          </div>
          <table class="table">
            <thead><tr><th>类型</th><th>编号</th><th>签发日</th><th>到期日</th></tr></thead>
            <tbody>
              <tr v-for="l in licenses" :key="l.id">
                <td>{{ kindOptions.find(k => k.value === l.kind)?.label ?? l.kind }}</td>
                <td>{{ l.number }}</td>
                <td>{{ formatDate(l.issue_date) }}</td>
                <td>{{ formatDate(l.expiry_date) }}</td>
              </tr>
              <tr v-if="!licenses.length"><td colspan="4" class="empty">暂无证照</td></tr>
            </tbody>
          </table>
        </div>

        <div class="card">
          <h3 class="card-title">状态流转历史</h3>
          <table class="table">
            <thead><tr><th>原状态</th><th>新状态</th><th>原因</th><th>操作人</th><th>时间</th></tr></thead>
            <tbody>
              <tr v-for="h in history" :key="h.id">
                <td><StatusBadge :status="h.from_status" /></td>
                <td><StatusBadge :status="h.to_status" /></td>
                <td>{{ h.reason }}</td>
                <td>{{ h.changed_by }}</td>
                <td>{{ formatDateTime(h.changed_at) }}</td>
              </tr>
              <tr v-if="!history.length"><td colspan="5" class="empty">暂无状态流转记录</td></tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card">
        <h3 class="card-title">司机绑定</h3>
        <table class="table">
          <thead><tr><th>绑定ID</th><th>司机</th><th>开始日期</th><th>结束日期</th><th>状态</th></tr></thead>
          <tbody>
            <tr v-for="b in bindings" :key="b.id">
              <td>{{ b.id }}</td>
              <td>{{ driverLabel(b.driver_id) }}</td>
              <td>{{ formatDate(b.start_date) }}</td>
              <td>{{ formatDate(b.end_date) }}</td>
              <td><StatusBadge :status="b.status" /></td>
            </tr>
            <tr v-if="!bindings.length"><td colspan="5" class="empty">暂无司机绑定</td></tr>
          </tbody>
        </table>
      </div>
    </template>

    <Modal :show="showLicense" title="添加证照" @close="showLicense = false">
      <div class="form-group"><label>类型</label>
        <select v-model="licForm.kind" class="input">
          <option v-for="k in kindOptions" :key="k.value" :value="k.value">{{ k.label }}</option>
        </select>
      </div>
      <div class="form-group"><label>编号</label><input v-model="licForm.number" class="input" /></div>
      <div class="form-group"><label>签发日期</label><input v-model="licForm.issue_date" type="date" class="input" /></div>
      <div class="form-group"><label>到期日期</label><input v-model="licForm.expiry_date" type="date" class="input" /></div>
      <template #footer>
        <button class="btn btn-sm" @click="showLicense = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="submitLicense">{{ saving ? '保存中...' : '保存' }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.page-subtitle { font-weight: 600; }
.card-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.info-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 14px; margin: 0; }
.info-grid > div { display: grid; grid-template-columns: 90px 1fr; gap: 8px; }
.info-grid dt { color: var(--muted); font-size: 13px; }
.info-grid dd { margin: 0; font-size: 14px; word-break: break-all; }
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-top: 20px; align-items: start; }
@media (max-width: 900px) { .grid-2 { grid-template-columns: 1fr; } }
</style>
