<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatYuan } from '@/utils/format'
import type { Part } from '@/types'
import Pagination from '@/components/Pagination.vue'
import Modal from '@/components/Modal.vue'

const router = useRouter()
const auth = useAuthStore()

const items = ref<Part[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const keyword = ref('')
const lowStockOnly = ref(false)
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    if (lowStockOnly.value) {
      const data = await api.listLowStock()
      items.value = data
      total.value = data.length
    } else {
      const params: Record<string, unknown> = { limit: limit.value, offset: (page.value - 1) * limit.value }
      if (keyword.value) params.keyword = keyword.value
      const data = await api.listParts(params as never)
      items.value = data.items
      total.value = data.total
    }
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

function onSearch() { page.value = 1; load() }

// 创建
const showCreate = ref(false)
const form = ref({ sku: '', name: '', unit: '个', stock_quantity: 0, reorder_point: 0, unit_cost: 0 })
const saving = ref(false)
async function submitCreate() {
  saving.value = true
  try {
    await api.createPart({
      sku: form.value.sku,
      name: form.value.name,
      unit: form.value.unit,
      stock_quantity: Number(form.value.stock_quantity),
      reorder_point: Number(form.value.reorder_point),
      unit_cost_cents: Math.round(Number(form.value.unit_cost) * 100),
    } as never)
    showCreate.value = false
    form.value = { sku: '', name: '', unit: '个', stock_quantity: 0, reorder_point: 0, unit_cost: 0 }
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    saving.value = false
  }
}

// 调整库存
const adjustTarget = ref<Part | null>(null)
const adjustForm = ref({ change: 0, reason: 'adjustment', note: '' })
const savingAdjust = ref(false)
async function submitAdjust() {
  if (!adjustTarget.value) return
  savingAdjust.value = true
  try {
    await api.adjustStock(adjustTarget.value.id, {
      change: Number(adjustForm.value.change),
      reason: adjustForm.value.reason,
      note: adjustForm.value.note,
    })
    adjustTarget.value = null
    adjustForm.value = { change: 0, reason: 'adjustment', note: '' }
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    savingAdjust.value = false
  }
}

const reasonOptions = [
  { value: 'purchase', label: '采购入库' },
  { value: 'adjustment', label: '盘点调整' },
  { value: 'return', label: '退货' },
  { value: 'order_consumption', label: '工单消耗' },
]

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <input v-model="keyword" placeholder="搜索SKU/名称" @keyup.enter="onSearch" class="input" />
      <label class="checkbox"><input type="checkbox" v-model="lowStockOnly" @change="onSearch" /> 仅低库存</label>
      <button class="btn btn-sm" @click="onSearch">筛选</button>
      <button class="btn btn-sm" @click="load">刷新</button>
      <button v-if="auth.hasPermission('part:create')" class="btn btn-primary btn-sm" @click="showCreate = true">+ 新增配件</button>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="card">
      <table class="table">
        <thead>
          <tr><th>ID</th><th>SKU</th><th>名称</th><th>单位</th><th>库存</th><th>补货点</th><th>单价</th><th>状态</th><th>操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="p in items" :key="p.id" class="clickable" @click="router.push(`/parts/${p.id}`)">
            <td>{{ p.id }}</td>
            <td class="mono">{{ p.sku }}</td>
            <td>{{ p.name }}</td>
            <td>{{ p.unit }}</td>
            <td :class="{ 'text-danger': p.stock_quantity <= p.reorder_point }">{{ p.stock_quantity }}</td>
            <td>{{ p.reorder_point }}</td>
            <td>{{ formatYuan(p.unit_cost_cents) }}</td>
            <td>
              <span v-if="p.stock_quantity <= p.reorder_point" class="tag-red">低库存</span>
              <span v-else class="tag-green">正常</span>
            </td>
            <td class="actions" @click.stop>
              <button v-if="auth.hasPermission('part:update')" class="btn btn-sm" @click="adjustTarget = p; adjustForm = { change: 0, reason: 'adjustment', note: '' }">调整库存</button>
            </td>
          </tr>
          <tr v-if="!items.length && !loading"><td colspan="9" class="empty">暂无配件</td></tr>
        </tbody>
      </table>
    </div>

    <Pagination v-if="!lowStockOnly" :total="total" :limit="limit" :page="page" @page="(p) => { page = p; load() }" @size="(s) => { limit = s; page = 1; load() }" />

    <Modal :show="showCreate" title="新增配件" @close="showCreate = false">
      <div class="form-grid-2">
        <div class="form-group"><label>SKU</label><input v-model="form.sku" class="input" /></div>
        <div class="form-group"><label>名称</label><input v-model="form.name" class="input" /></div>
        <div class="form-group"><label>单位</label><input v-model="form.unit" class="input" /></div>
        <div class="form-group"><label>初始库存</label><input v-model.number="form.stock_quantity" type="number" class="input" /></div>
        <div class="form-group"><label>补货点</label><input v-model.number="form.reorder_point" type="number" class="input" /></div>
        <div class="form-group"><label>单价(元)</label><input v-model.number="form.unit_cost" type="number" step="0.01" class="input" /></div>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="showCreate = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="saving" @click="submitCreate">{{ saving ? '保存中...' : '保存' }}</button>
      </template>
    </Modal>

    <Modal :show="!!adjustTarget" title="调整库存" @close="adjustTarget = null">
      <p>配件：<strong>{{ adjustTarget?.name }} ({{ adjustTarget?.sku }})</strong></p>
      <p>当前库存：{{ adjustTarget?.stock_quantity }}（补货点 {{ adjustTarget?.reorder_point }}）</p>
      <div class="form-group"><label>变动数量(正数入库/负数出库)</label><input v-model.number="adjustForm.change" type="number" class="input" /></div>
      <div class="form-group"><label>原因</label>
        <select v-model="adjustForm.reason" class="input">
          <option v-for="r in reasonOptions" :key="r.value" :value="r.value">{{ r.label }}</option>
        </select>
      </div>
      <div class="form-group"><label>备注</label><textarea v-model="adjustForm.note" class="input" rows="2"></textarea></div>
      <template #footer>
        <button class="btn btn-sm" @click="adjustTarget = null">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="savingAdjust" @click="submitAdjust">{{ savingAdjust ? '保存中...' : '确认调整' }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; align-items: center; }
.checkbox { display: flex; align-items: center; gap: 6px; font-size: 14px; }
.actions { display: flex; gap: 6px; }
.mono { font-family: monospace; font-size: 12px; }
.clickable { cursor: pointer; }
.clickable:hover { background: #f9fafb; }
.text-danger { color: #dc2626; font-weight: 600; }
.tag-red { color: #dc2626; font-weight: 600; font-size: 12px; }
.tag-green { color: #15803d; font-weight: 600; font-size: 12px; }
.form-grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
</style>
