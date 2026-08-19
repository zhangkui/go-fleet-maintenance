<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  total: number
  limit: number
  page: number
}>()

const emit = defineEmits<{ (e: 'page', p: number): void; (e: 'size', s: number): void }>()

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.limit)))
const from = computed(() => (props.page - 1) * props.limit + 1)
const to = computed(() => Math.min(props.total, props.page * props.limit))

const sizes = [10, 20, 50, 100]

function go(p: number) {
  if (p < 1 || p > totalPages.value) return
  emit('page', p)
}
</script>

<template>
  <div class="pagination">
    <div class="pagination-info">
      共 {{ total }} 条，第 {{ from }}-{{ to }} 条 / 共 {{ totalPages }} 页
    </div>
    <div class="pagination-controls">
      <select :value="limit" class="size-select" @change="emit('size', Number(($event.target as HTMLSelectElement).value))">
        <option v-for="s in sizes" :key="s" :value="s">每页 {{ s }}</option>
      </select>
      <button class="btn btn-sm" :disabled="page <= 1" @click="go(page - 1)">上一页</button>
      <span class="page-no">{{ page }} / {{ totalPages }}</span>
      <button class="btn btn-sm" :disabled="page >= totalPages" @click="go(page + 1)">下一页</button>
      <button class="btn btn-sm" :disabled="page <= 1" @click="go(1)">首页</button>
      <button class="btn btn-sm" :disabled="page >= totalPages" @click="go(totalPages)">末页</button>
    </div>
  </div>
</template>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  padding: 12px 0;
}
.pagination-info {
  color: var(--muted);
  font-size: 13px;
}
.pagination-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}
.size-select {
  padding: 4px 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface);
  color: var(--text);
  font-size: 13px;
}
.page-no {
  font-size: 13px;
  color: var(--muted);
  padding: 0 4px;
}
</style>
