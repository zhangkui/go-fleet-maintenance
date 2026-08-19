<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatYuan, formatDateTime, formatDate, toRFC3339 } from '@/utils/format'
import type { MaintenanceOrder, MaintenancePolicy, Vehicle } from '@/types'
import Pagination from '@/components/Pagination.vue'
import Modal from '@/components/Modal.vue'
import StatusBadge from '@/components/StatusBadge.vue'

const router = useRouter()
const auth = useAuthStore()

// 车辆下拉选项
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

const tab = ref<'orders' | 'policies'>('orders')

// 工单
const orders = ref<MaintenanceOrder[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const statusFilter = ref('')
const loading = ref(false)
const error = ref('')

async function loadOrders() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, unknown> = { limit: limit.value, offset: (page.value - 1) * limit.value }
    if (statusFilter.value) params.status = statusFilter.value
    const data = await api.listOrders(params as never)
    orders.value = data.items
    total.value = data.total
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

// 策略
const policies = ref<MaintenancePolicy[]>([])
async function loadPolicies() {
  try {
    policies.value = await api.listPolicies()
  } catch (e) {
    error.value = extractErrorMessage(e)
  }
}

async function load() {
  if (tab.value === 'orders') await loadOrders()
  else await loadPolicies()
}

function onTab(t: 'orders' | 'policies') {
  tab.value = t
  page.value = 1
  load()
}

// 触发到期
const triggering = ref(false)
async function triggerDue() {
  if (!confirm('确认扫描并生成到期维保工单？')) return
  triggering.value = true
  try {
    const res = await api.triggerDue()
    alert(`已生成 ${res.created} 个工单`)
    await loadOrders()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    triggering.value = false
  }
}

// 创建工单
const showOrder = ref(false)
const orderForm = ref({ vehicle_id: '' as number | '', policy_id: '' as number | '', kind: 'periodic', title: '', labor_cost: 0, parts: '' as string })
const savingOrder = ref(false)
// 选中车辆后，可选项为该车辆的维保策略
const policyOptions = computed(() => {
  if (orderForm.value.vehicle_id === '' || orderForm.value.vehicle_id === 0) return []
  const vid = Number(orderForm.value.vehicle_id)
  // 当前已加载的策略全部列出（后端按 vehicle_id 过滤）
  return policies.value.filter((p) => p.vehicle_id === vid)
})
async function submitOrder() {
  savingOrder.value = true
  try {
    const body: Record<string, unknown> = {
      vehicle_id: Number(orderForm.value.vehicle_id),
      kind: orderForm.value.kind,
      title: orderForm.value.title,
      labor_cost_cents: Math.round(Number(orderForm.value.labor_cost) * 100),
      idempotency_key: crypto.randomUUID(),
    }
    if (orderForm.value.policy_id !== '' && orderForm.value.policy_id !== 0) body.policy_id = Number(orderForm.value.policy_id)
    await api.createOrder(body as never)
    showOrder.value = false
    orderForm.value = { vehicle_id: '', policy_id: '', kind: 'periodic', title: '', labor_cost: 0, parts: '' }
    await loadOrders()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    savingOrder.value = false
  }
}

// 创建策略
const showPolicy = ref(false)
const policyForm = ref({
  vehicle_id: '' as number | '', name: '', kind: 'periodic', interval_km: 0, interval_days: 0,
  last_service_km: 0, last_service_at: '', enabled: true,
})
const savingPolicy = ref(false)
async function submitPolicy() {
  savingPolicy.value = true
  try {
    const body: Record<string, unknown> = {
      vehicle_id: Number(policyForm.value.vehicle_id),
      name: policyForm.value.name,
      kind: policyForm.value.kind,
      interval_km: Number(policyForm.value.interval_km),
      interval_days: Number(policyForm.value.interval_days),
      last_service_km: Number(policyForm.value.last_service_km),
      last_service_at: toRFC3339(policyForm.value.last_service_at || new Date().toISOString().slice(0, 16)),
      enabled: policyForm.value.enabled,
    }
    await api.createPolicy(body as never)
    showPolicy.value = false
    policyForm.value = { vehicle_id: '', name: '', kind: 'periodic', interval_km: 0, interval_days: 0, last_service_km: 0, last_service_at: '', enabled: true }
    await loadPolicies()
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    savingPolicy.value = false
  }
}

const statusOptions = [
  { value: 'pending', label: '待审批' },
  { value: 'approved', label: '已批准' },
  { value: 'in_progress', label: '进行中' },
  { value: 'completed', label: '已完成' },
  { value: 'cancelled', label: '已取消' },
]

const kindOptions = [
  { value: 'periodic', label: '定期保养' },
  { value: 'inspection', label: '检查' },
  { value: 'repair', label: '维修' },
]

onMounted(() => { loadOptions(); loadOrders() })

// 打开新建工单时，若策略未加载则先加载该车辆策略
watch(() => showOrder.value, (v) => {
  if (v && policies.value.length === 0) loadPolicies()
})
</script>

<template>
  <div>
    <div class="tabs">
      <button class="tab" :class="{ active: tab === 'orders' }" @click="onTab('orders')">维保工单</button>
      <button class="tab" :class="{ active: tab === 'policies' }" @click="onTab('policies')">维保策略</button>
      <div class="tab-actions">
        <template v-if="tab === 'orders'">
          <select v-model="statusFilter" class="input narrow" @change="page = 1; loadOrders()">
            <option value="">全部状态</option>
            <option v-for="s in statusOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
          </select>
          <button v-if="auth.hasPermission('maintenance:trigger')" class="btn btn-sm" :disabled="triggering" @click="triggerDue">
            {{ triggering ? '触发中...' : '触发到期' }}
          </button>
          <button v-if="auth.hasPermission('maintenance:create')" class="btn btn-primary btn-sm" @click="showOrder = true">+ 新建工单</button>
        </template>
        <template v-else>
          <button v-if="auth.hasPermission('maintenance:create')" class="btn btn-primary btn-sm" @click="showPolicy = true">+ 新建策略</button>
        </template>
        <button class="btn btn-sm" @click="load">刷新</button>
      </div>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <template v-if="tab === 'orders'">
      <div class="card">
        <table class="table">
          <thead>
            <tr><th>ID</th><th>车辆</th><th>标题</th><th>类型</th><th>状态</th><th>配件费</th><th>工时费</th><th>总费用</th><th>创建时间</th></tr>
          </thead>
          <tbody>
            <tr v-for="o in orders" :key="o.id" class="clickable" @click="router.push(`/maintenance/orders/${o.id}`)">
              <td>{{ o.id }}</td>
              <td>{{ vehicleLabel(o.vehicle_id) }}</td>
              <td>{{ o.title }}</td>
              <td>{{ kindOptions.find(k => k.value === o.kind)?.label ?? o.kind }}</td>
              <td><StatusBadge :status="o.status" /></td>
              <td>{{ formatYuan(o.parts_cost_cents) }}</td>
              <td>{{ formatYuan(o.labor_cost_cents) }}</td>
              <td>{{ formatYuan(o.total_cost_cents) }}</td>
              <td>{{ formatDateTime(o.created_at) }}</td>
            </tr>
            <tr v-if="!orders.length && !loading"><td colspan="9" class="empty">暂无工单</td></tr>
          </tbody>
        </table>
      </div>
      <Pagination :total="total" :limit="limit" :page="page" @page="(p) => { page = p; loadOrders() }" @size="(s) => { limit = s; page = 1; loadOrders() }" />
    </template>

    <template v-else>
      <div class="card">
        <table class="table">
          <thead>
            <tr><th>ID</th><th>车辆</th><th>名称</th><th>类型</th><th>间隔里程</th><th>间隔天数</th><th>下次到期里程</th><th>下次到期日</th><th>启用</th></tr>
          </thead>
          <tbody>
            <tr v-for="p in policies" :key="p.id">
              <td>{{ p.id }}</td>
              <td>{{ vehicleLabel(p.vehicle_id) }}</td>
              <td>{{ p.name }}</td>
              <td>{{ kindOptions.find(k => k.value === p.kind)?.label ?? p.kind }}</td>
              <td>{{ p.interval_km }} km</td>
              <td>{{ p.interval_days }} 天</td>
              <td>{{ p.next_due_km }} km</td>
              <td>{{ formatDate(p.next_due_at) }}</td>
              <td><span :class="p.enabled ? 'tag-green' : 'tag-gray'">{{ p.enabled ? '是' : '否' }}</span></td>
            </tr>
            <tr v-if="!policies.length"><td colspan="9" class="empty">暂无策略</td></tr>
          </tbody>
        </table>
      </div>
    </template>

    <Modal :show="showOrder" title="新建维保工单" wide @close="showOrder = false">
      <div class="form-grid-2">
        <div class="form-group"><label>车辆</label>
          <select v-model="orderForm.vehicle_id" class="input">
            <option :value="''">请选择车辆</option>
            <option v-for="v in vehicles" :key="v.id" :value="v.id">{{ v.plate_number }}（{{ v.model }}）</option>
          </select>
        </div>
        <div class="form-group"><label>关联策略(可选)</label>
          <select v-model="orderForm.policy_id" class="input">
            <option :value="''">无</option>
            <option v-for="p in policyOptions" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
        </div>
        <div class="form-group"><label>类型</label>
          <select v-model="orderForm.kind" class="input">
            <option v-for="k in kindOptions" :key="k.value" :value="k.value">{{ k.label }}</option>
          </select>
        </div>
        <div class="form-group"><label>工时费(元)</label><input v-model.number="orderForm.labor_cost" type="number" step="0.01" class="input" /></div>
        <div class="form-group" style="grid-column: span 2"><label>标题</label><input v-model="orderForm.title" class="input" /></div>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="showOrder = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="savingOrder" @click="submitOrder">{{ savingOrder ? '保存中...' : '保存' }}</button>
      </template>
    </Modal>

    <Modal :show="showPolicy" title="新建维保策略" wide @close="showPolicy = false">
      <div class="form-grid-2">
        <div class="form-group"><label>车辆</label>
          <select v-model="policyForm.vehicle_id" class="input">
            <option :value="''">请选择车辆</option>
            <option v-for="v in vehicles" :key="v.id" :value="v.id">{{ v.plate_number }}（{{ v.model }}）</option>
          </select>
        </div>
        <div class="form-group"><label>名称</label><input v-model="policyForm.name" class="input" /></div>
        <div class="form-group"><label>类型</label>
          <select v-model="policyForm.kind" class="input">
            <option v-for="k in kindOptions" :key="k.value" :value="k.value">{{ k.label }}</option>
          </select>
        </div>
        <div class="form-group"><label>间隔里程(km)</label><input v-model.number="policyForm.interval_km" type="number" class="input" /></div>
        <div class="form-group"><label>间隔天数</label><input v-model.number="policyForm.interval_days" type="number" class="input" /></div>
        <div class="form-group"><label>上次保养里程</label><input v-model.number="policyForm.last_service_km" type="number" class="input" /></div>
        <div class="form-group"><label>上次保养时间</label><input v-model="policyForm.last_service_at" type="datetime-local" class="input" /></div>
        <div class="form-group"><label>启用</label><input v-model="policyForm.enabled" type="checkbox" /></div>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="showPolicy = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="savingPolicy" @click="submitPolicy">{{ savingPolicy ? '保存中...' : '保存' }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.tabs { display: flex; align-items: center; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.tab { padding: 8px 16px; border: 1px solid var(--border); background: var(--surface); border-radius: 8px; cursor: pointer; font-size: 14px; }
.tab.active { background: var(--primary); color: #fff; border-color: var(--primary); }
.tab-actions { display: flex; gap: 8px; margin-left: auto; align-items: center; flex-wrap: wrap; }
.narrow { width: 130px; }
.clickable { cursor: pointer; }
.clickable:hover { background: #f9fafb; }
.form-grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.tag-green { color: #15803d; font-weight: 600; }
.tag-gray { color: var(--muted); }
</style>
