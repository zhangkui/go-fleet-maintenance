<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/format'
import type { Reminder, ReminderScanResult } from '@/types'
import Pagination from '@/components/Pagination.vue'
import StatusBadge from '@/components/StatusBadge.vue'

const auth = useAuthStore()

const items = ref<Reminder[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const statusFilter = ref('')
const loading = ref(false)
const error = ref('')

const scanDays = ref(30)
const scanning = ref(false)
const scanResult = ref<ReminderScanResult | null>(null)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, unknown> = { limit: limit.value, offset: (page.value - 1) * limit.value }
    if (statusFilter.value) params.status = statusFilter.value
    const data = await api.listReminders(params as never)
    items.value = data.items
    total.value = data.total
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

async function scan() {
  scanning.value = true
  scanResult.value = null
  try {
    scanResult.value = await api.scanReminders(scanDays.value)
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    scanning.value = false
  }
}

const entityTypeLabels: Record<string, string> = {
  vehicle_insurance: '车辆保险',
  vehicle_inspection: '车辆年检',
  driver_license: '司机驾照',
  maintenance_policy: '维保策略',
}

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <select v-model="statusFilter" class="input narrow" @change="page = 1; load()">
        <option value="">全部状态</option>
        <option value="pending">待处理</option>
        <option value="sent">已发送</option>
        <option value="dismissed">已忽略</option>
      </select>
      <button class="btn btn-sm" @click="load">刷新</button>
      <div class="scan-bar">
        <input v-model.number="scanDays" type="number" class="input narrow" min="1" />
        <span class="hint">天内到期</span>
        <button v-if="auth.hasPermission('reminder:scan')" class="btn btn-primary btn-sm" :disabled="scanning" @click="scan">
          {{ scanning ? '扫描中...' : '扫描到期' }}
        </button>
      </div>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div v-if="scanResult" class="card scan-result">
      <h3 class="card-title">扫描结果</h3>
      <div class="scan-grid">
        <div><span>保险到期</span><strong>{{ scanResult.insurance }}</strong></div>
        <div><span>年检到期</span><strong>{{ scanResult.inspection }}</strong></div>
        <div><span>驾照到期</span><strong>{{ scanResult.license }}</strong></div>
        <div><span>维保到期</span><strong>{{ scanResult.maintenance }}</strong></div>
        <div><span>新增提醒</span><strong>{{ scanResult.created }}</strong></div>
      </div>
    </div>

    <div class="card">
      <table class="table">
        <thead>
          <tr><th>ID</th><th>类型</th><th>实体ID</th><th>到期时间</th><th>消息</th><th>状态</th><th>创建时间</th></tr>
        </thead>
        <tbody>
          <tr v-for="r in items" :key="r.id">
            <td>{{ r.id }}</td>
            <td>{{ entityTypeLabels[r.entity_type] ?? r.entity_type }}</td>
            <td>{{ r.entity_id }}</td>
            <td>{{ formatDateTime(r.due_at) }}</td>
            <td>{{ r.message }}</td>
            <td><StatusBadge :status="r.status" /></td>
            <td>{{ formatDateTime(r.created_at) }}</td>
          </tr>
          <tr v-if="!items.length && !loading"><td colspan="7" class="empty">暂无提醒</td></tr>
        </tbody>
      </table>
    </div>

    <Pagination :total="total" :limit="limit" :page="page" @page="(p) => { page = p; load() }" @size="(s) => { limit = s; page = 1; load() }" />
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; align-items: center; }
.narrow { width: 120px; }
.scan-bar { display: flex; align-items: center; gap: 8px; margin-left: auto; }
.hint { font-size: 13px; color: var(--muted); }
.scan-result { margin-bottom: 16px; }
.scan-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; }
.scan-grid > div { display: flex; flex-direction: column; gap: 4px; padding: 12px; background: var(--bg); border-radius: 8px; }
.scan-grid span { color: var(--muted); font-size: 13px; }
.scan-grid strong { font-size: 20px; color: var(--primary); }
</style>
