<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { api } from '@/api'
import { extractErrorMessage } from '@/api/client'

const router = useRouter()

const form = ref({ username: '', password: '', email: '', full_name: '' })
const showPassword = ref(false)
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await api.register(form.value)
    router.push({ name: 'login', query: { registered: '1' } })
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-header">
        <div class="auth-logo">
          <svg viewBox="0 0 24 24" width="42" height="42" fill="none" stroke="#fff" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M19 8v6"/><path d="M22 11h-6"/>
          </svg>
        </div>
        <h1>创建账户</h1>
        <p class="auth-sub">注册新用户</p>
      </div>
      <form @submit.prevent="submit" class="auth-form">
        <div class="field">
          <span class="field-icon">👤</span>
          <input v-model="form.username" type="text" required autocomplete="username" placeholder="用户名" />
        </div>
        <div class="field">
          <span class="field-icon">🔒</span>
          <input v-model="form.password" :type="showPassword ? 'text' : 'password'" required autocomplete="new-password" placeholder="密码（至少 8 位）" />
          <button type="button" class="toggle-pwd" @click="showPassword = !showPassword">{{ showPassword ? '🙈' : '👁️' }}</button>
        </div>
        <div class="field">
          <span class="field-icon">✉️</span>
          <input v-model="form.email" type="email" required autocomplete="email" placeholder="邮箱" />
        </div>
        <div class="field">
          <span class="field-icon">📝</span>
          <input v-model="form.full_name" type="text" required autocomplete="name" placeholder="姓名" />
        </div>
        <transition name="fade">
          <div v-if="error" class="error-msg">⚠️ {{ error }}</div>
        </transition>
        <button class="submit-btn" type="submit" :disabled="loading">
          <span v-if="loading" class="spinner"></span>
          {{ loading ? '注册中...' : '注 册' }}
        </button>
      </form>
      <div class="auth-footer">
        已有账户？<RouterLink to="/login" class="link">返回登录 →</RouterLink>
      </div>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1e1b4b 0%, #0f172a 60%, #1e293b 100%);
  padding: 20px;
  position: relative;
  overflow: hidden;
}
.auth-page::before {
  content: '';
  position: absolute;
  width: 600px; height: 600px;
  background: radial-gradient(circle, rgba(79,70,229,0.18) 0%, transparent 65%);
  top: -200px; right: -150px;
  border-radius: 50%;
}
.auth-page::after {
  content: '';
  position: absolute;
  width: 500px; height: 500px;
  background: radial-gradient(circle, rgba(2,132,199,0.14) 0%, transparent 65%);
  bottom: -150px; left: -150px;
  border-radius: 50%;
}
.auth-card {
  background: rgba(255,255,255,0.98);
  border-radius: 20px;
  padding: 42px 38px 32px;
  width: 100%;
  max-width: 420px;
  box-shadow: 0 30px 80px rgba(0,0,0,0.45);
  position: relative;
  z-index: 1;
}
.auth-header { text-align: center; margin-bottom: 30px; }
.auth-logo {
  width: 68px; height: 68px;
  margin: 0 auto 14px;
  background: linear-gradient(135deg, var(--primary) 0%, #7c3aed 100%);
  border-radius: 18px;
  display: flex; align-items: center; justify-content: center;
  box-shadow: 0 8px 24px rgba(79,70,229,0.4);
}
.auth-header h1 { font-size: 22px; margin: 0 0 5px; font-weight: 800; color: var(--text); }
.auth-sub { color: var(--muted); font-size: 13px; margin: 0; }
.auth-form { display: flex; flex-direction: column; gap: 14px; }
.field { position: relative; display: flex; align-items: center; }
.field-icon { position: absolute; left: 14px; font-size: 16px; opacity: 0.5; pointer-events: none; }
.field input {
  width: 100%;
  padding: 11px 14px 11px 42px;
  border: 1.5px solid var(--border);
  border-radius: 10px;
  font-size: 14px;
  background: #f8fafc;
  outline: none;
  transition: all 0.18s;
}
.field input:focus { border-color: var(--primary); background: #fff; box-shadow: 0 0 0 4px rgba(79,70,229,0.1); }
.toggle-pwd { position: absolute; right: 10px; background: none; border: none; font-size: 16px; cursor: pointer; padding: 4px; }
.error-msg { background: var(--danger-light); border: 1px solid #fecaca; color: #b91c1c; padding: 9px 14px; border-radius: 9px; font-size: 13px; }
.submit-btn {
  padding: 11px; border: none; border-radius: 10px;
  background: linear-gradient(135deg, var(--primary) 0%, #7c3aed 100%);
  color: #fff; font-size: 15px; font-weight: 600; cursor: pointer;
  transition: all 0.2s; display: flex; align-items: center; justify-content: center; gap: 8px;
  box-shadow: 0 4px 14px rgba(79,70,229,0.35);
}
.submit-btn:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 6px 20px rgba(79,70,229,0.45); }
.submit-btn:active:not(:disabled) { transform: translateY(0); }
.submit-btn:disabled { opacity: 0.7; cursor: not-allowed; }
.spinner { width: 16px; height: 16px; border: 2px solid rgba(255,255,255,0.35); border-top-color: #fff; border-radius: 50%; animation: spin 0.7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
.auth-footer { text-align: center; margin-top: 20px; font-size: 13px; color: var(--muted); }
.link { color: var(--primary); font-weight: 600; text-decoration: none; }
.link:hover { text-decoration: underline; }
</style>
