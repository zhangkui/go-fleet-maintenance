<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'
import { formatDateTime } from '@/utils/format'
import type { AuditLog } from '@/types'
import Pagination from '@/components/Pagination.vue'

const items = ref<AuditLog[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const keyword = ref('')
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, unknown> = { limit: limit.value, offset: (page.value - 1) * limit.value }
    if (keyword.value) params.keyword = keyword.value
    const data = await api.listAuditLogs(params as never)
    items.value = data.items
    total.value = data.total
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}

function onSearch() { page.value = 1; load() }

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <input v-model="keyword" placeholder="搜索操作/资源/操作人" @keyup.enter="onSearch" class="input" />
      <button class="btn btn-sm" @click="onSearch">搜索</button>
      <button class="btn btn-sm" @click="load">刷新</button>
    </div>
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div class="card">
      <table class="table">
        <thead>
          <tr><th>ID</th><th>操作人</th><th>动作</th><th>资源类型</th><th>资源ID</th><th>详情</th><th>IP</th><th>时间</th></tr>
        </thead>
        <tbody>
          <tr v-for="l in items" :key="l.id">
            <td>{{ l.id }}</td>
            <td>{{ l.actor_name }} <span class="text-muted">(#{{ l.actor_user_id }})</span></td>
            <td><code>{{ l.action }}</code></td>
            <td>{{ l.resource_type }}</td>
            <td>{{ l.resource_id }}</td>
            <td class="detail">{{ l.detail }}</td>
            <td>{{ l.ip }}</td>
            <td>{{ formatDateTime(l.created_at) }}</td>
          </tr>
          <tr v-if="!items.length && !loading"><td colspan="8" class="empty">暂无审计日志</td></tr>
        </tbody>
      </table>
    </div>

    <Pagination :total="total" :limit="limit" :page="page" @page="(p) => { page = p; load() }" @size="(s) => { limit = s; page = 1; load() }" />
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; }
.text-muted { color: var(--muted); font-size: 12px; }
.detail { max-width: 320px; word-break: break-all; }
</style>
