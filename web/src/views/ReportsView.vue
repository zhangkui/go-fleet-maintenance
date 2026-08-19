<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import {
  formatYuan, formatKm, formatLitersUnit, formatPct, formatNum,
  toRFC3339,
} from '@/utils/format'
import type { VehicleUtilization, FuelEfficiencyReport, MaintenanceCostReport } from '@/types'
import Pagination from '@/components/Pagination.vue'

const tab = ref<'utilization' | 'fuel' | 'cost'>('utilization')

const loading = ref(false)
const error = ref('')
const items = ref<(VehicleUtilization | FuelEfficiencyReport | MaintenanceCostReport)[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)

const fromLocal = ref('')
const toLocal = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, unknown> = { limit: limit.value, offset: (page.value - 1) * limit.value }
    if (fromLocal.value) params.from = toRFC3339(fromLocal.value)
    if (toLocal.value) params.to = toRFC3339(toLocal.value)
    let data: { items: typeof items.value; total: number; limit: number; page: number }
    if (tab.value === 'utilization') {
      data = await api.vehicleUtilization(params as never)
    } else if (tab.value === 'fuel') {
      data = await api.fuelEfficiency(params as never)
    } else {
      data = await api.maintenanceCost(params as never)
    }
    items.value = data.items
    total.value = data.total
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

function onTab(t: 'utilization' | 'fuel' | 'cost') {
  tab.value = t
  page.value = 1
  items.value = []
  load()
}

function onSearch() { page.value = 1; load() }

onMounted(load)
</script>

<template>
  <div>
    <div class="tabs">
      <button class="tab" :class="{ active: tab === 'utilization' }" @click="onTab('utilization')">车辆利用率</button>
      <button class="tab" :class="{ active: tab === 'fuel' }" @click="onTab('fuel')">油耗效率</button>
      <button class="tab" :class="{ active: tab === 'cost' }" @click="onTab('cost')">维保成本</button>
    </div>

    <div class="toolbar">
      <label>起始日期 <input v-model="fromLocal" type="date" class="input" /></label>
      <label>截止日期 <input v-model="toLocal" type="date" class="input" /></label>
      <button class="btn btn-sm" @click="onSearch">查询</button>
      <button class="btn btn-sm" @click="load">刷新</button>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="card">
      <table class="table">
        <thead>
          <template v-if="tab === 'utilization'">
            <tr><th>车辆ID</th><th>车牌</th><th>型号</th><th>任务数</th><th>总里程</th><th>总加油</th><th>利用率</th></tr>
          </template>
          <template v-else-if="tab === 'fuel'">
            <tr><th>车辆ID</th><th>车牌</th><th>总油量</th><th>总里程</th><th>总费用</th><th>百公里油耗</th><th>每公里成本</th><th>异常记录</th></tr>
          </template>
          <template v-else>
            <tr><th>车辆ID</th><th>车牌</th><th>工单数</th><th>配件费</th><th>工时费</th><th>总费用</th><th>停机时长</th></tr>
          </template>
        </thead>
        <tbody>
          <template v-if="tab === 'utilization'">
            <tr v-for="(r, i) in items as VehicleUtilization[]" :key="i">
              <td>{{ r.vehicle_id }}</td><td>{{ r.plate_number }}</td><td>{{ r.model }}</td>
              <td>{{ formatNum(r.trip_count) }}</td>
              <td>{{ formatKm(r.total_distance_km) }}</td>
              <td>{{ formatLitersUnit(r.total_fuel_liters) }}</td>
              <td>{{ formatPct(r.utilization_pct) }}</td>
            </tr>
          </template>
          <template v-else-if="tab === 'fuel'">
            <tr v-for="(r, i) in items as FuelEfficiencyReport[]" :key="i">
              <td>{{ r.vehicle_id }}</td><td>{{ r.plate_number }}</td>
              <td>{{ formatLitersUnit(r.total_liters_milli) }}</td>
              <td>{{ formatKm(r.total_distance_km) }}</td>
              <td>{{ formatYuan(r.total_cost_cents) }}</td>
              <td>{{ r.liters_per_100km.toFixed(2) }} L/100km</td>
              <td>{{ formatYuan(r.cost_per_km_cents) }}/km</td>
              <td><span :class="r.abnormal_records > 0 ? 'tag-red' : ''">{{ r.abnormal_records }}</span></td>
            </tr>
          </template>
          <template v-else>
            <tr v-for="(r, i) in items as MaintenanceCostReport[]" :key="i">
              <td>{{ r.vehicle_id }}</td><td>{{ r.plate_number }}</td>
              <td>{{ formatNum(r.order_count) }}</td>
              <td>{{ formatYuan(r.parts_cost_cents) }}</td>
              <td>{{ formatYuan(r.labor_cost_cents) }}</td>
              <td>{{ formatYuan(r.total_cost_cents) }}</td>
              <td>{{ r.downtime_hours }} 小时</td>
            </tr>
          </template>
          <tr v-if="!items.length && !loading"><td :colspan="tab === 'fuel' ? 8 : 7" class="empty">暂无数据</td></tr>
        </tbody>
      </table>
    </div>

    <Pagination :total="total" :limit="limit" :page="page" @page="(p) => { page = p; load() }" @size="(s) => { limit = s; page = 1; load() }" />
  </div>
</template>

<style scoped>
.tabs { display: flex; gap: 8px; margin-bottom: 16px; }
.tab { padding: 8px 16px; border: 1px solid var(--border); background: var(--surface); border-radius: 8px; cursor: pointer; font-size: 14px; }
.tab.active { background: var(--primary); color: #fff; border-color: var(--primary); }
.toolbar { display: flex; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; align-items: center; }
.toolbar label { display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--muted); }
.tag-red { color: #dc2626; font-weight: 600; }
</style>
