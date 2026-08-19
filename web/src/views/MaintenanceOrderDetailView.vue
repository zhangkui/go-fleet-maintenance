<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatYuan, formatDateTime, toRFC3339 } from '@/utils/format'
import type { MaintenanceOrder, MaintenanceOrderPart, Vehicle, Part } from '@/types'
import Modal from '@/components/Modal.vue'
import StatusBadge from '@/components/StatusBadge.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const order = ref<MaintenanceOrder | null>(null)
const parts = ref<MaintenanceOrderPart[]>([])
const vehicles = ref<Vehicle[]>([])
const partList = ref<Part[]>([])
const loading = ref(false)
const error = ref('')
const id = Number(route.params.id)

function vehicleLabel(vid: number) {
  const x = vehicles.value.find((v) => v.id === vid)
  return x ? `${x.plate_number}（${x.model}）` : `#${vid}`
}
function partLabel(pid: number) {
  const x = partList.value.find((p) => p.id === pid)
  return x ? `${x.name}（${x.sku}）` : `#${pid}`
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [data, vs, ps] = await Promise.all([
      api.getOrder(id),
      api.listVehicles({ limit: 100 }).catch(() => ({ items: [] as Vehicle[] })),
      api.listParts({ limit: 100 }).catch(() => ({ items: [] as Part[] })),
    ])
    order.value = data.order
    parts.value = data.parts ?? []
    vehicles.value = vs.items
    partList.value = ps.items
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

// 状态流转
const showStatus = ref(false)
const statusForm = ref({ status: '' })
async function submitStatus() {
  try {
    await api.transitionOrder(id, statusForm.value.status)
    showStatus.value = false
    statusForm.value = { status: '' }
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

// 完成
const showComplete = ref(false)
const completeForm = ref({ downtime_end: '', note: '' })
async function submitComplete() {
  try {
    await api.completeOrder(id, {
      downtime_end: toRFC3339(completeForm.value.downtime_end || new Date().toISOString().slice(0, 16)),
      note: completeForm.value.note,
    })
    showComplete.value = false
    completeForm.value = { downtime_end: '', note: '' }
    await load()
  } catch (e) {
    alert(extractErrorMessage(e))
  }
}

const statusOptions = [
  { value: 'pending', label: '待审批' },
  { value: 'approved', label: '已批准' },
  { value: 'in_progress', label: '进行中' },
  { value: 'completed', label: '已完成' },
  { value: 'cancelled', label: '已取消' },
]

const kindLabels: Record<string, string> = { periodic: '定期保养', inspection: '检查', repair: '维修' }

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <button class="btn btn-sm" @click="router.push('/maintenance')">← 返回维保列表</button>
      <span v-if="order" class="page-subtitle">工单 #{{ order.id }}</span>
      <StatusBadge v-if="order" :status="order.status" />
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-if="loading" class="loading">加载中...</div>

    <template v-if="order">
      <div class="card">
        <div class="card-head">
          <h3 class="card-title">工单信息</h3>
          <div class="action-bar">
            <template v-if="auth.hasPermission('maintenance:approve')">
              <button v-if="order.status === 'pending' || order.status === 'approved'" class="btn btn-sm" @click="showStatus = true; statusForm = { status: '' }">状态流转</button>
            </template>
            <template v-if="auth.hasPermission('maintenance:update')">
              <button v-if="order.status === 'in_progress'" class="btn btn-primary btn-sm" @click="showComplete = true">完成工单</button>
            </template>
          </div>
        </div>
        <dl class="info-grid">
          <div><dt>车辆</dt><dd>{{ vehicleLabel(order.vehicle_id) }}</dd></div>
          <div><dt>策略ID</dt><dd>{{ order.policy_id || '-' }}</dd></div>
          <div><dt>类型</dt><dd>{{ kindLabels[order.kind] ?? order.kind }}</dd></div>
          <div><dt>标题</dt><dd>{{ order.title }}</dd></div>
          <div><dt>停机开始</dt><dd>{{ formatDateTime(order.downtime_start) }}</dd></div>
          <div><dt>停机结束</dt><dd>{{ formatDateTime(order.downtime_end) }}</dd></div>
          <div><dt>配件费</dt><dd>{{ formatYuan(order.parts_cost_cents) }}</dd></div>
          <div><dt>工时费</dt><dd>{{ formatYuan(order.labor_cost_cents) }}</dd></div>
          <div><dt>总费用</dt><dd>{{ formatYuan(order.total_cost_cents) }}</dd></div>
          <div><dt>完成时间</dt><dd>{{ formatDateTime(order.completed_at) }}</dd></div>
          <div><dt>创建人</dt><dd>{{ order.created_by }}</dd></div>
          <div><dt>创建时间</dt><dd>{{ formatDateTime(order.created_at) }}</dd></div>
        </dl>
      </div>

      <div class="card">
        <h3 class="card-title">配件明细</h3>
        <table class="table">
          <thead><tr><th>ID</th><th>配件</th><th>数量</th><th>单价</th><th>小计</th></tr></thead>
          <tbody>
            <tr v-for="p in parts" :key="p.id">
              <td>{{ p.id }}</td>
              <td>{{ partLabel(p.part_id) }}</td>
              <td>{{ p.quantity }}</td>
              <td>{{ formatYuan(p.unit_cost_cents) }}</td>
              <td>{{ formatYuan(p.line_total_cents) }}</td>
            </tr>
            <tr v-if="!parts.length"><td colspan="5" class="empty">暂无配件明细</td></tr>
          </tbody>
        </table>
      </div>
    </template>

    <Modal :show="showStatus" title="工单状态流转" @close="showStatus = false">
      <p>当前状态：<StatusBadge :status="order?.status ?? ''" /></p>
      <div class="form-group"><label>目标状态</label>
        <select v-model="statusForm.status" class="input">
          <option value="" disabled>选择状态</option>
          <option v-for="s in statusOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
        </select>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="showStatus = false">取消</button>
        <button class="btn btn-primary btn-sm" @click="submitStatus">确认流转</button>
      </template>
    </Modal>

    <Modal :show="showComplete" title="完成工单" @close="showComplete = false">
      <div class="form-group"><label>停机结束时间</label><input v-model="completeForm.downtime_end" type="datetime-local" class="input" /></div>
      <div class="form-group"><label>备注</label><textarea v-model="completeForm.note" class="input" rows="3"></textarea></div>
      <template #footer>
        <button class="btn btn-sm" @click="showComplete = false">取消</button>
        <button class="btn btn-primary btn-sm" @click="submitComplete">确认完成</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.page-subtitle { font-weight: 600; }
.card-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; gap: 12px; flex-wrap: wrap; }
.action-bar { display: flex; gap: 8px; }
.info-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 14px; margin: 0; }
.info-grid > div { display: grid; grid-template-columns: 90px 1fr; gap: 8px; }
.info-grid dt { color: var(--muted); font-size: 13px; }
.info-grid dd { margin: 0; }
</style>
