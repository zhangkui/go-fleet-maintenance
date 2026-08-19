<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'

const auth = useAuthStore()

const form = ref({
  old_password: '',
  new_password: '',
  confirm: '',
})
const error = ref('')
const success = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  success.value = ''
  if (form.value.new_password !== form.value.confirm) {
    error.value = '两次输入的新密码不一致'
    return
  }
  loading.value = true
  try {
    await api.changePassword(form.value.old_password, form.value.new_password)
    success.value = '密码修改成功'
    form.value = { old_password: '', new_password: '', confirm: '' }
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="profile-page">
    <div class="card">
      <h2 class="card-title">个人信息</h2>
      <dl class="info-grid">
        <div><dt>用户名</dt><dd>{{ auth.user?.username }}</dd></div>
        <div><dt>姓名</dt><dd>{{ auth.user?.full_name }}</dd></div>
        <div><dt>邮箱</dt><dd>{{ auth.user?.email }}</dd></div>
        <div><dt>状态</dt><dd><StatusBadge :status="auth.user?.status ?? 'active'" /></dd></div>
        <div><dt>角色</dt><dd>{{ auth.roles.join('、') || '-' }}</dd></div>
      </dl>
    </div>

    <div class="card">
      <h2 class="card-title">修改密码</h2>
      <form @submit.prevent="submit" class="form-grid">
        <div class="form-group">
          <label>当前密码</label>
          <input v-model="form.old_password" type="password" required autocomplete="current-password" />
        </div>
        <div class="form-group">
          <label>新密码</label>
          <input v-model="form.new_password" type="password" required autocomplete="new-password" />
        </div>
        <div class="form-group">
          <label>确认新密码</label>
          <input v-model="form.confirm" type="password" required autocomplete="new-password" />
        </div>
        <div class="form-actions">
          <div v-if="error" class="alert alert-error">{{ error }}</div>
          <div v-if="success" class="alert alert-success">{{ success }}</div>
          <button class="btn btn-primary" type="submit" :disabled="loading">
            {{ loading ? '提交中...' : '修改密码' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.profile-page {
  display: grid;
  gap: 20px;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
}
.info-grid {
  display: grid;
  gap: 14px;
  margin: 0;
}
.info-grid > div {
  display: grid;
  grid-template-columns: 100px 1fr;
  gap: 12px;
  align-items: center;
}
.info-grid dt {
  color: var(--muted);
  font-size: 13px;
}
.info-grid dd {
  margin: 0;
  font-size: 14px;
}
.form-grid {
  display: grid;
  gap: 16px;
  max-width: 420px;
}
.form-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: flex-start;
}
</style>
