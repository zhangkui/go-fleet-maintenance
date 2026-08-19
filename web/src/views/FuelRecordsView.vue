<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatYuan, formatLitersUnit, formatKm, formatDateTime, toRFC3339 } from '@/utils/format'
import type { FuelRecord, Vehicle } from '@/types'
import Pagination from '@/components/Pagination.vue'
import Modal from '@/components/Modal.vue'

const auth = useAuthStore()

const items = ref<FuelRecord[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const vehicleId = ref('' as number | '')
const abnormalOnly = ref(false)
const loading = ref(false)
const error = ref('')

const vehicles = ref<Vehicle[]>([])
async function loadOptions() {
  try {
    vehicles.value = (await api.listVehicles({ limit: 100 })).items
  } catch { /* 忽略 */ }
}
function vehicleLabel(id: number) {
  const x = vehicles.value.find((v) => v.id === id)
  return x ? `${x.plate_number}（${x.model}）` : `#${id}`
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, unknown> = { limit: limit.value, offset: (page.value - 1) * limit.value }
    if (vehicleId.value !== '' && vehicleId.value !== 0) params.vehicle_id = Number(vehicleId.value)
    if (abnormalOnly.value) params.abnormal = true
    const data = await api.listFuelRecords(params as never)
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
const form = ref({
  vehicle_id: '' as number | '', liters: 0, unit_price: 0, odometer_km: 0, total_cost: 0, recorded_at: '',
})
const saving = ref(false)

async function submitCreate() {
  saving.value = true
  try {
    await api.createFuelRecord({
      vehicle_id: Number(form.value.vehicle_id),
      liters_milli: Math.round(Number(form.value.liters) * 1000),
      unit_price_cents: Math.round(Number(form.value.unit_price) * 100),
      odometer_km: Number(form.value.odometer_km),
      total_cost_cents: Math.round(Number(form.value.total_cost) * 100),
      recorded_at: toRFC3339(form.value.recorded_at || new Date().toISOString().slice(0, 16)),
      idempotency_key: crypto.randomUUID(),
    })
    showCreate.value = false
    form.value = { vehicle_id: '', liters: 0, unit_price: 0, odometer_km: 0, total_cost: 0, recorded_at: '' }
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    saving.value = false
  }
}

onMounted(() => { loadOptions(); load() })
</script>

<template>
  <div>
    <div class="toolbar">
      <select v-model="vehicleId" class="input narrow" @change="onSearch">
        <option :value="''">全部车辆</option>
        <option v-for="v in vehicles" :key="v.id" :value="v.id">{{ v.plate_number }}</option>
      </select>
      <label class="checkbox"><input type="checkbox" v-model="abnormalOnly" @change="onSearch" /> 仅异常</label>
      <button class="btn btn-sm" @click="onSearch">筛选</button>
      <button class="btn btn-sm" @click="load">刷新</button>
      <button v-if="auth.hasPermission('fuel:create')" class="btn btn-primary btn-sm" @click="showCreate = true">+ 录入油耗</button>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="card">
      <table class="table">
        <thead>
          <tr><th>ID</th><th>车辆</th><th>加油量</th><th>单价</th><th>里程</th><th>总价</th><th>异常</th><th>记录时间</th></tr>
        </thead>
        <tbody>
          <tr v-for="r in items" :key="r.id">
            <td>{{ r.id }}</td>
            <td>{{ vehicleLabel(r.vehicle_id) }}</td>
            <td>{{ formatLitersUnit(r.liters_milli) }}</td>
            <td>{{ formatYuan(r.unit_price_cents) }}/L</td>
            <td>{{ formatKm(r.odometer_km) }}</td>
            <td>{{ formatYuan(r.total_cost_cents) }}</td>
            <td><span :class="r.abnormal ? 'tag-red' : 'tag-gray'">{{ r.abnormal ? '异常' : '正常' }}</span></td>
            <td>{{ formatDateTime(r.recorded_at) }}</td>
          </tr>
          <tr v-if="!items.length && !loading"><td colspan="8" class="empty">暂无油耗记录</td></tr>
        </tbody>
      </table>
    </div>

    <Pagination :total="total" :limit="limit" :page="page" @page="(p) => { page = p; load() }" @size="(s) => { limit = s; page = 1; load() }" />

    <Modal :show="showCreate" title="录入油耗记录" wide @close="showCreate = false">
      <div class="form-grid-2">
        <div class="form-group"><label>车辆</label>
          <select v-model="form.vehicle_id" class="input">
            <option :value="''">请选择车辆</option>
            <option v-for="v in vehicles" :key="v.id" :value="v.id">{{ v.plate_number }}（{{ v.model }}）</option>
          </select>
        </div>
        <div class="form-group"><label>加油量(升)</label><input v-model.number="form.liters" type="number" step="0.001" class="input" /></div>
        <div class="form-group"><label>单价(元/升)</label><input v-model.number="form.unit_price" type="number" step="0.01" class="input" /></div>
        <div class="form-group"><label>里程(km)</label><input v-model.number="form.odometer_km" type="number" class="input" /></div>
        <div class="form-group"><label>总价(元)</label><input v-model.number="form.total_cost" type="number" step="0.01" class="input" /></div>
        <div class="form-group"><label>记录时间</label><input v-model="form.recorded_at" type="datetime-local" class="input" /></div>
      </div>
      <p class="hint">总价与单价可只填一项，另一项由系统计算；为兼容后端建议两项都填。</p>
      <template #footer>
        <button class="btn btn-sm" @click="showCreate = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="submitCreate">{{ saving ? '保存中...' : '保存' }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; align-items: center; }
.narrow { width: 160px; }
.checkbox { display: flex; align-items: center; gap: 6px; font-size: 14px; }
.form-grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.tag-red { color: #dc2626; font-weight: 600; }
.tag-gray { color: var(--muted); }
.hint { color: var(--muted); font-size: 12px; }
</style>
