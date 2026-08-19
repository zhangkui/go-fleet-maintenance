<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { formatYuan, formatDateTime } from '@/utils/format'
import type { Part, PartStockMovement } from '@/types'
import Pagination from '@/components/Pagination.vue'

const route = useRoute()
const router = useRouter()

const part = ref<Part | null>(null)
const movements = ref<PartStockMovement[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const loading = ref(false)
const error = ref('')
const id = Number(route.params.id)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, unknown> = { limit: limit.value, offset: (page.value - 1) * limit.value }
    const [p, m] = await Promise.all([
      api.listParts({ keyword: '' }).then((d) => d.items.find((x) => x.id === id) ?? null).catch(() => null),
      api.listStockMovements(id, params as never),
    ])
    part.value = p
    movements.value = m.items
    total.value = m.total
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

const reasonLabels: Record<string, string> = {
  purchase: '采购入库',
  adjustment: '盘点调整',
  return: '退货',
  order_consumption: '工单消耗',
}

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <button class="btn btn-sm" @click="router.push('/parts')">← 返回配件列表</button>
      <span v-if="part" class="page-subtitle">{{ part.name }} ({{ part.sku }})</span>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-if="loading" class="loading">加载中...</div>

    <template v-if="part">
      <div class="card">
        <h3 class="card-title">配件信息</h3>
        <dl class="info-grid">
          <div><dt>SKU</dt><dd>{{ part.sku }}</dd></div>
          <div><dt>名称</dt><dd>{{ part.name }}</dd></div>
          <div><dt>单位</dt><dd>{{ part.unit }}</dd></div>
          <div><dt>库存</dt><dd :class="{ 'text-danger': part.stock_quantity <= part.reorder_point }">{{ part.stock_quantity }}</dd></div>
          <div><dt>补货点</dt><dd>{{ part.reorder_point }}</dd></div>
          <div><dt>单价</dt><dd>{{ formatYuan(part.unit_cost_cents) }}</dd></div>
        </dl>
      </div>

      <div class="card">
        <h3 class="card-title">库存变动流水</h3>
        <table class="table">
          <thead><tr><th>ID</th><th>变动数量</th><th>原因</th><th>关联工单</th><th>变动后余额</th><th>操作人</th><th>时间</th></tr></thead>
          <tbody>
            <tr v-for="m in movements" :key="m.id">
              <td>{{ m.id }}</td>
              <td :class="m.change_quantity >= 0 ? 'text-green' : 'text-red'">{{ m.change_quantity > 0 ? '+' : '' }}{{ m.change_quantity }}</td>
              <td>{{ reasonLabels[m.reason] ?? m.reason }}</td>
              <td>{{ m.ref_order_id ?? '-' }}</td>
              <td>{{ m.balance_after }}</td>
              <td>{{ m.created_by }}</td>
              <td>{{ formatDateTime(m.created_at) }}</td>
            </tr>
            <tr v-if="!movements.length"><td colspan="7" class="empty">暂无变动记录</td></tr>
          </tbody>
        </table>
      </div>
      <Pagination :total="total" :limit="limit" :page="page" @page="(p) => { page = p; load() }" @size="(s) => { limit = s; page = 1; load() }" />
    </template>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.page-subtitle { font-weight: 600; }
.info-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 14px; margin: 0; }
.info-grid > div { display: grid; grid-template-columns: 80px 1fr; gap: 8px; }
.info-grid dt { color: var(--muted); font-size: 13px; }
.info-grid dd { margin: 0; }
.text-green { color: #15803d; font-weight: 600; }
.text-red { color: #dc2626; font-weight: 600; }
.text-danger { color: #dc2626; font-weight: 600; }
</style>
