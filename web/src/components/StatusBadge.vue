<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const colorClass = computed(() => {
  switch (props.status) {
    case 'active':
    case 'approved':
    case 'completed':
      return 'badge-green'
    case 'in_progress':
    case 'in_maintenance':
      return 'badge-blue'
    case 'scheduled':
    case 'pending':
      return 'badge-gray'
    case 'cancelled':
    case 'disabled':
    case 'resigned':
    case 'retired':
    case 'suspended':
      return 'badge-red'
    case 'paid':
    case 'contested':
      return 'badge-yellow'
    default:
      return 'badge-gray'
  }
})

const labels: Record<string, string> = {
  active: '活跃',
  disabled: '已禁用',
  in_maintenance: '维保中',
  retired: '已报废',
  scheduled: '已排程',
  in_progress: '进行中',
  completed: '已完成',
  cancelled: '已取消',
  suspended: '已停用',
  resigned: '已离职',
  pending: '待处理',
  approved: '已批准',
  paid: '已缴费',
  contested: '申诉中',
  sent: '已发送',
  dismissed: '已忽略',
}

const label = computed(() => labels[props.status] ?? props.status)
</script>

<template>
  <span class="badge" :class="colorClass">{{ label }}</span>
</template>

<style scoped>
.badge {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}
.badge-green {
  background: #dcfce7;
  color: #15803d;
}
.badge-blue {
  background: #dbeafe;
  color: #1d4ed8;
}
.badge-gray {
  background: #f3f4f6;
  color: #6b7280;
}
.badge-red {
  background: #fee2e2;
  color: #b91c1c;
}
.badge-yellow {
  background: #fef9c3;
  color: #a16207;
}
</style>
