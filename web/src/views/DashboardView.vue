<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatYuan, formatDateTime } from '@/utils/format'
import type { FleetSummary, Part } from '@/types'

const auth = useAuthStore()
const summary = ref<FleetSummary | null>(null)
const lowStock = ref<Part[]>([])
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const promises: Promise<void>[] = []
    if (auth.hasPermission('report:read')) {
      promises.push(api.fleetSummary().then((d) => { summary.value = d }))
    }
    if (auth.hasPermission('part:read')) {
      promises.push(api.listLowStock().then((d) => { lowStock.value = d }))
    }
    await Promise.all(promises)
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

interface StatCard {
  key: string
  label: string
  icon: string
  color: string
  bg: string
  get: () => number
}

const stats = computed<StatCard[]>(() => [
  { key: 'total_vehicles', label: '车辆总数', icon: '🚛', color: '#4f46e5', bg: '#eef2ff', get: () => summary.value?.total_vehicles ?? 0 },
  { key: 'active_vehicles', label: '在用车辆', icon: '✅', color: '#16a34a', bg: '#f0fdf4', get: () => summary.value?.active_vehicles ?? 0 },
  { key: 'in_maintenance', label: '维保中', icon: '🔧', color: '#d97706', bg: '#fffbeb', get: () => summary.value?.in_maintenance ?? 0 },
  { key: 'total_drivers', label: '司机总数', icon: '👤', color: '#0284c7', bg: '#f0f9ff', get: () => summary.value?.total_drivers ?? 0 },
  { key: 'active_drivers', label: '在岗司机', icon: '🧑‍💼', color: '#0891b2', bg: '#ecfeff', get: () => summary.value?.active_drivers ?? 0 },
  { key: 'open_trips', label: '进行中任务', icon: '🧭', color: '#7c3aed', bg: '#f5f3ff', get: () => summary.value?.open_trips ?? 0 },
  { key: 'open_orders', label: '未结工单', icon: '📋', color: '#dc2626', bg: '#fef2f2', get: () => summary.value?.open_orders ?? 0 },
  { key: 'low_stock_parts', label: '低库存配件', icon: '📦', color: '#ca8a04', bg: '#fefce8', get: () => summary.value?.low_stock_parts ?? 0 },
  { key: 'expiring_documents', label: '到期证照', icon: '🔔', color: '#be185d', bg: '#fdf2f8', get: () => summary.value?.expiring_documents ?? 0 },
])

onMounted(load)
</script>

<template>
  <div class="dashboard">
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-if="loading" class="loading">加载中...</div>

    <template v-if="auth.hasPermission('report:read') && summary">
      <div class="page-header">
        <h3 class="section-title">车队总览</h3>
        <p class="gen-time">报表生成时间：{{ formatDateTime(summary.generated_at) }}</p>
      </div>
      <div class="stat-grid">
        <div v-for="s in stats" :key="s.key" class="stat-card">
          <div class="stat-icon" :style="{ background: s.bg, color: s.color }">{{ s.icon }}</div>
          <div class="stat-body">
            <div class="stat-value" :style="{ color: s.color }">{{ s.get() }}</div>
            <div class="stat-label">{{ s.label }}</div>
          </div>
        </div>
      </div>
    </template>

    <template v-if="auth.hasPermission('part:read')">
      <h3 class="section-title">低库存配件提醒</h3>
      <div class="card">
        <table v-if="lowStock.length" class="table">
          <thead>
            <tr>
              <th>SKU</th><th>名称</th><th>单位</th><th>当前库存</th><th>补货点</th><th>单价</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in lowStock" :key="p.id">
              <td><code>{{ p.sku }}</code></td>
              <td>{{ p.name }}</td>
              <td>{{ p.unit }}</td>
              <td><span class="tag-red">{{ p.stock_quantity }}</span></td>
              <td>{{ p.reorder_point }}</td>
              <td>{{ formatYuan(p.unit_cost_cents) }}</td>
            </tr>
          </tbody>
        </table>
        <div v-else class="empty">🎉 暂无低库存配件，库存充足</div>
      </div>
    </template>

    <template v-if="!auth.hasPermission('report:read') && !auth.hasPermission('part:read')">
      <div class="empty">您暂无可在仪表盘显示的数据权限。</div>
    </template>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.page-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
.section-title {
  margin: 0 0 4px;
  font-size: 17px;
  font-weight: 700;
  color: var(--text);
}
.stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 16px;
}
.stat-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: var(--shadow);
  transition: all 0.2s;
}
.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}
.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}
.stat-body {
  min-width: 0;
}
.stat-value {
  font-size: 26px;
  font-weight: 800;
  line-height: 1.1;
}
.stat-label {
  color: var(--muted);
  font-size: 13px;
  margin-top: 2px;
}
.gen-time {
  color: var(--muted);
  font-size: 12px;
  margin: 0;
}
</style>
