<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatKm, formatDateTime, toRFC3339 } from '@/utils/format'
import type { Trip, TripHandover, Vehicle, Driver } from '@/types'
import Modal from '@/components/Modal.vue'
import StatusBadge from '@/components/StatusBadge.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const trip = ref<Trip | null>(null)
const handovers = ref<TripHandover[]>([])
const vehicles = ref<Vehicle[]>([])
const drivers = ref<Driver[]>([])
const loading = ref(false)
const error = ref('')
const id = Number(route.params.id)

function vehicleLabel(vid: number) {
  const x = vehicles.value.find((v) => v.id === vid)
  return x ? `${x.plate_number}（${x.model}）` : `#${vid}`
}
function driverLabel(did: number) {
  const x = drivers.value.find((d) => d.id === did)
  return x ? `${x.name}（${x.license_number}）` : `#${did}`
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [t, vs, ds] = await Promise.all([
      api.getTrip(id),
      api.listVehicles({ limit: 100 }).catch(() => ({ items: [] as Vehicle[] })),
      api.listDrivers({ limit: 100 }).catch(() => ({ items: [] as Driver[] })),
    ])
    trip.value = t
    vehicles.value = vs.items
    drivers.value = ds.items
    handovers.value = [] // 后端暂未提供任务交接列表接口时为空
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

const showHandover = ref(false)
const handoverForm = ref({ from_driver_id: '' as number | '', to_driver_id: '' as number | '', handover_at: '', location: '', note: '' })
const saving = ref(false)

async function submitHandover() {
  saving.value = true
  try {
    await api.createHandover(id, {
      from_driver_id: Number(handoverForm.value.from_driver_id),
      to_driver_id: Number(handoverForm.value.to_driver_id),
      handover_at: toRFC3339(handoverForm.value.handover_at),
      location: handoverForm.value.location,
      note: handoverForm.value.note,
    } as never)
    showHandover.value = false
    handoverForm.value = { from_driver_id: '', to_driver_id: '', handover_at: '', location: '', note: '' }
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    saving.value = false
  }
}

const infoRows = computed(() => {
  if (!trip.value) return []
  const t = trip.value
  return [
    { label: '任务ID', value: t.id },
    { label: '车辆', value: vehicleLabel(t.vehicle_id) },
    { label: '司机', value: driverLabel(t.driver_id) },
    { label: '路线', value: t.route },
    { label: '载重', value: formatKm(t.load_kg).replace('km', 'kg') },
    { label: '起始里程', value: formatKm(t.start_odometer_km) },
    { label: '结束里程', value: formatKm(t.end_odometer_km) },
    { label: '完成时间', value: formatDateTime(t.completed_at) },
    { label: '创建时间', value: formatDateTime(t.created_at) },
    { label: '备注', value: t.note || '-' },
  ]
})

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <button class="btn btn-sm" @click="router.push('/trips')">← 返回任务列表</button>
      <span v-if="trip" class="page-subtitle">任务 #{{ trip.id }}</span>
      <StatusBadge v-if="trip" :status="trip.status" />
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-if="loading" class="loading">加载中...</div>

    <template v-if="trip">
      <div class="card">
        <h3 class="card-title">任务信息</h3>
        <dl class="info-grid">
          <div v-for="r in infoRows" :key="r.label">
            <dt>{{ r.label }}</dt><dd>{{ r.value }}</dd>
          </div>
        </dl>
      </div>

      <div class="card">
        <div class="card-head">
          <h3 class="card-title">交接记录</h3>
          <button v-if="auth.hasPermission('trip:update') && trip.status === 'in_progress'" class="btn btn-sm btn-primary" @click="showHandover = true">+ 交接</button>
        </div>
        <table class="table">
          <thead><tr><th>ID</th><th>原司机</th><th>新司机</th><th>交接时间</th><th>地点</th><th>备注</th></tr></thead>
          <tbody>
            <tr v-for="h in handovers" :key="h.id">
              <td>{{ h.id }}</td>
              <td>{{ driverLabel(h.from_driver_id) }}</td>
              <td>{{ driverLabel(h.to_driver_id) }}</td>
              <td>{{ formatDateTime(h.handover_at) }}</td>
              <td>{{ h.location }}</td>
              <td>{{ h.note }}</td>
            </tr>
            <tr v-if="!handovers.length"><td colspan="6" class="empty">暂无交接记录</td></tr>
          </tbody>
        </table>
      </div>
    </template>

    <Modal :show="showHandover" title="新增交接" wide @close="showHandover = false">
      <div class="form-grid-2">
        <div class="form-group"><label>原司机</label>
          <select v-model="handoverForm.from_driver_id" class="input">
            <option :value="''">请选择司机</option>
            <option v-for="d in drivers" :key="d.id" :value="d.id">{{ d.name }}（{{ d.license_number }}）</option>
          </select>
        </div>
        <div class="form-group"><label>新司机</label>
          <select v-model="handoverForm.to_driver_id" class="input">
            <option :value="''">请选择司机</option>
            <option v-for="d in drivers" :key="d.id" :value="d.id">{{ d.name }}（{{ d.license_number }}）</option>
          </select>
        </div>
        <div class="form-group"><label>交接时间</label><input v-model="handoverForm.handover_at" type="datetime-local" class="input" /></div>
        <div class="form-group"><label>地点</label><input v-model="handoverForm.location" class="input" /></div>
        <div class="form-group" style="grid-column: span 2"><label>备注</label><textarea v-model="handoverForm.note" class="input" rows="2"></textarea></div>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="showHandover = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="submitHandover">{{ saving ? '保存中...' : '保存' }}</button>
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
.info-grid dd { margin: 0; }
.form-grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
</style>
