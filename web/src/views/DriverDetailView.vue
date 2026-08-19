<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime, formatDate, formatYuan, toRFC3339 } from '@/utils/format'
import type { Driver, DriverSchedule, DriverViolation } from '@/types'
import Modal from '@/components/Modal.vue'
import StatusBadge from '@/components/StatusBadge.vue'

const violationTypes: { value: string; label: string }[] = [
  { value: 'speeding', label: '超速' },
  { value: 'overload', label: '超载' },
  { value: 'illegal_parking', label: '违停' },
  { value: 'red_light', label: '闯红灯' },
  { value: 'other', label: '其他' },
]
function violationTypeLabel(t: string) {
  return violationTypes.find((x) => x.value === t)?.label ?? t
}

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const driver = ref<Driver | null>(null)
const schedules = ref<DriverSchedule[]>([])
const violations = ref<DriverViolation[]>([])
const loading = ref(false)
const error = ref('')
const id = Number(route.params.id)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [d, schs, viols] = await Promise.all([
      api.getDriver(id),
      api.listSchedules(id),
      api.listViolations(id),
    ])
    driver.value = d
    schedules.value = schs
    violations.value = viols
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

// 排班
const showSchedule = ref(false)
const schForm = ref({ shift_date: '', shift_type: 'morning', note: '' })
const savingSch = ref(false)
async function submitSchedule() {
  savingSch.value = true
  try {
    await api.createSchedule({
      driver_id: id,
      shift_date: toRFC3339(schForm.value.shift_date),
      shift_type: schForm.value.shift_type,
      note: schForm.value.note,
    } as never)
    showSchedule.value = false
    schForm.value = { shift_date: '', shift_type: 'morning', note: '' }
    schedules.value = await api.listSchedules(id)
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    savingSch.value = false
  }
}

// 违章
const showViolation = ref(false)
const violForm = ref({ occurred_at: '', violation_type: '', points: 0, fine_cents: 0, description: '', status: 'pending' })
const savingViol = ref(false)
async function submitViolation() {
  savingViol.value = true
  try {
    await api.createViolation(id, {
      occurred_at: toRFC3339(violForm.value.occurred_at),
      violation_type: violForm.value.violation_type,
      points: Number(violForm.value.points),
      fine_cents: Number(violForm.value.fine_cents) * 100,
      description: violForm.value.description,
      status: violForm.value.status,
    } as never)
    showViolation.value = false
    violForm.value = { occurred_at: '', violation_type: '', points: 0, fine_cents: 0, description: '', status: 'pending' }
    violations.value = await api.listViolations(id)
  } catch (e) {
    alert(extractErrorMessage(e))
  } finally {
    savingViol.value = false
  }
}

const shiftOptions = [
  { value: 'morning', label: '早班' },
  { value: 'evening', label: '晚班' },
  { value: 'night', label: '夜班' },
  { value: 'off', label: '休息' },
]

const violStatusOptions = [
  { value: 'pending', label: '待处理' },
  { value: 'paid', label: '已缴费' },
  { value: 'contested', label: '申诉中' },
]

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <button class="btn btn-sm" @click="router.push('/drivers')">← 返回司机列表</button>
      <span v-if="driver" class="page-subtitle">{{ driver.name }}</span>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>
    <div v-if="loading" class="loading">加载中...</div>

    <template v-if="driver">
      <div class="card">
        <div class="card-head">
          <h3 class="card-title">司机信息</h3>
          <StatusBadge :status="driver.status" />
        </div>
        <dl class="info-grid">
          <div><dt>姓名</dt><dd>{{ driver.name }}</dd></div>
          <div><dt>驾驶证号</dt><dd>{{ driver.license_number }}</dd></div>
          <div><dt>准驾类型</dt><dd>{{ driver.license_class }}</dd></div>
          <div><dt>电话</dt><dd>{{ driver.phone }}</dd></div>
          <div><dt>驾照到期</dt><dd>{{ formatDate(driver.license_expiry) }}</dd></div>
        </dl>
      </div>

      <div class="grid-2">
        <div class="card">
          <div class="card-head">
            <h3 class="card-title">排班记录</h3>
            <button v-if="auth.hasPermission('driver:update')" class="btn btn-sm btn-primary" @click="showSchedule = true">+ 排班</button>
          </div>
          <table class="table">
            <thead><tr><th>日期</th><th>班次</th><th>备注</th></tr></thead>
            <tbody>
              <tr v-for="s in schedules" :key="s.id">
                <td>{{ formatDate(s.shift_date) }}</td>
                <td>{{ shiftOptions.find(o => o.value === s.shift_type)?.label ?? s.shift_type }}</td>
                <td>{{ s.note }}</td>
              </tr>
              <tr v-if="!schedules.length"><td colspan="3" class="empty">暂无排班</td></tr>
            </tbody>
          </table>
        </div>

        <div class="card">
          <div class="card-head">
            <h3 class="card-title">违章记录</h3>
            <button v-if="auth.hasPermission('driver:update')" class="btn btn-sm btn-primary" @click="showViolation = true">+ 录入</button>
          </div>
          <table class="table">
            <thead><tr><th>时间</th><th>类型</th><th>扣分</th><th>罚款</th><th>状态</th><th>描述</th></tr></thead>
            <tbody>
              <tr v-for="v in violations" :key="v.id">
                <td>{{ formatDateTime(v.occurred_at) }}</td>
                <td>{{ violationTypeLabel(v.violation_type) }}</td>
                <td>{{ v.points }}</td>
                <td>{{ formatYuan(v.fine_cents) }}</td>
                <td><StatusBadge :status="v.status" /></td>
                <td>{{ v.description }}</td>
              </tr>
              <tr v-if="!violations.length"><td colspan="6" class="empty">暂无违章记录</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <Modal :show="showSchedule" title="新增排班" @close="showSchedule = false">
      <div class="form-group"><label>排班日期</label><input v-model="schForm.shift_date" type="date" class="input" /></div>
      <div class="form-group"><label>班次</label>
        <select v-model="schForm.shift_type" class="input">
          <option v-for="o in shiftOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
        </select>
      </div>
      <div class="form-group"><label>备注</label><textarea v-model="schForm.note" class="input" rows="2"></textarea></div>
      <template #footer>
        <button class="btn btn-sm" @click="showSchedule = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="savingSch" @click="submitSchedule">{{ savingSch ? '保存中...' : '保存' }}</button>
      </template>
    </Modal>

    <Modal :show="showViolation" title="录入违章" wide @close="showViolation = false">
      <div class="form-grid-2">
        <div class="form-group"><label>发生时间</label><input v-model="violForm.occurred_at" type="datetime-local" class="input" /></div>
        <div class="form-group"><label>违章类型</label>
          <select v-model="violForm.violation_type" class="input">
            <option value="">请选择</option>
            <option v-for="t in violationTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
          </select>
        </div>
        <div class="form-group"><label>扣分</label><input v-model.number="violForm.points" type="number" class="input" /></div>
        <div class="form-group"><label>罚款金额(元)</label><input v-model.number="violForm.fine_cents" type="number" class="input" /></div>
        <div class="form-group"><label>状态</label>
          <select v-model="violForm.status" class="input">
            <option v-for="o in violStatusOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>
        </div>
        <div class="form-group" style="grid-column: span 2"><label>描述</label><textarea v-model="violForm.description" class="input" rows="2"></textarea></div>
      </div>
      <template #footer>
        <button class="btn btn-sm" @click="showViolation = false">取消</button>
        <button class="btn btn-primary btn-sm" :disabled="savingViol" @click="submitViolation">{{ savingViol ? '保存中...' : '保存' }}</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.page-subtitle { font-weight: 600; }
.card-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.info-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 14px; margin: 0; }
.info-grid > div { display: grid; grid-template-columns: 80px 1fr; gap: 8px; }
.info-grid dt { color: var(--muted); font-size: 13px; }
.info-grid dd { margin: 0; }
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-top: 20px; align-items: start; }
.form-grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
@media (max-width: 900px) { .grid-2 { grid-template-columns: 1fr; } }
</style>
