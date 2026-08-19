<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { extractErrorMessage } from '@/api/client'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const username = ref('admin')
const password = ref('Admin123!')
const showPassword = ref(false)
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value, password.value)
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
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
          <svg viewBox="0 0 24 24" width="44" height="44" fill="none" stroke="#fff" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M1 3h15v13H1z"/><path d="M16 8h4l3 3v5h-7V8z"/><circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/>
          </svg>
        </div>
        <h1>车队维保管理系统</h1>
        <p class="auth-sub">车队车辆与维保调度平台</p>
      </div>

      <form @submit.prevent="submit" class="auth-form">
        <div class="field">
          <span class="field-icon">👤</span>
          <input v-model="username" type="text" required autocomplete="username" placeholder="用户名" />
        </div>
        <div class="field">
          <span class="field-icon">🔒</span>
          <input v-model="password" :type="showPassword ? 'text' : 'password'" required autocomplete="current-password" placeholder="密码" />
          <button type="button" class="toggle-pwd" @click="showPassword = !showPassword">{{ showPassword ? '🙈' : '👁️' }}</button>
        </div>
        <transition name="fade">
          <div v-if="error" class="error-msg">⚠️ {{ error }}</div>
        </transition>
        <button class="submit-btn" type="submit" :disabled="loading">
          <span v-if="loading" class="spinner"></span>
          {{ loading ? '登录中...' : '登 录' }}
        </button>
      </form>

      <div class="auth-footer">
        还没有账户？<RouterLink to="/register" class="link">立即注册 →</RouterLink>
      </div>

      <div class="demo-notice">
        <div class="demo-title">⚠️ 本地验收默认管理员</div>
        <div class="demo-creds">
          <span>用户名</span><code>admin</code>
          <span>密码</span><code>Admin123!</code>
        </div>
        <div class="demo-warn">该凭据仅用于本地验收，生产部署后请立即修改。</div>
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
  width: 600px;
  height: 600px;
  background: radial-gradient(circle, rgba(79,70,229,0.18) 0%, transparent 65%);
  top: -200px;
  right: -150px;
  border-radius: 50%;
}
.auth-page::after {
  content: '';
  position: absolute;
  width: 500px;
  height: 500px;
  background: radial-gradient(circle, rgba(2,132,199,0.14) 0%, transparent 65%);
  bottom: -150px;
  left: -150px;
  border-radius: 50%;
}

.auth-card {
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(20px);
  border-radius: 20px;
  padding: 42px 38px 34px;
  width: 100%;
  max-width: 420px;
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.45);
  position: relative;
  z-index: 1;
}

.auth-header {
  text-align: center;
  margin-bottom: 32px;
}
.auth-logo {
  width: 72px;
  height: 72px;
  margin: 0 auto 14px;
  background: linear-gradient(135deg, var(--primary) 0%, #7c3aed 100%);
  border-radius: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 24px rgba(79, 70, 229, 0.4);
}
.auth-header h1 {
  font-size: 22px;
  margin: 0 0 5px;
  font-weight: 800;
  color: var(--text);
}
.auth-sub {
  color: var(--muted);
  font-size: 13px;
  margin: 0;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.field {
  position: relative;
  display: flex;
  align-items: center;
}
.field-icon {
  position: absolute;
  left: 14px;
  font-size: 16px;
  opacity: 0.5;
  pointer-events: none;
}
.field input {
  width: 100%;
  padding: 11px 14px 11px 42px;
  border: 1.5px solid var(--border);
  border-radius: 10px;
  font-size: 14px;
  background: #f8fafc;
  color: var(--text);
  outline: none;
  transition: all 0.18s;
}
.field input:focus {
  border-color: var(--primary);
  background: #fff;
  box-shadow: 0 0 0 4px rgba(79, 70, 229, 0.1);
}
.toggle-pwd {
  position: absolute;
  right: 10px;
  background: none;
  border: none;
  font-size: 16px;
  cursor: pointer;
  padding: 4px;
}

.error-msg {
  background: var(--danger-light);
  border: 1px solid #fecaca;
  color: #b91c1c;
  padding: 9px 14px;
  border-radius: 9px;
  font-size: 13px;
}

.submit-btn {
  padding: 11px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--primary) 0%, #7c3aed 100%);
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  box-shadow: 0 4px 14px rgba(79, 70, 229, 0.35);
}
.submit-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(79, 70, 229, 0.45);
}
.submit-btn:active:not(:disabled) {
  transform: translateY(0);
}
.submit-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255,255,255,0.35);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.fade-enter-active, .fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

.auth-footer {
  text-align: center;
  margin-top: 20px;
  font-size: 13px;
  color: var(--muted);
}
.link {
  color: var(--primary);
  font-weight: 600;
  text-decoration: none;
}
.link:hover { text-decoration: underline; }

.demo-notice {
  margin-top: 22px;
  padding: 14px 16px;
  background: var(--warning-light);
  border: 1px solid #fde68a;
  border-radius: 12px;
  font-size: 13px;
}
.demo-title {
  font-weight: 700;
  color: #92400e;
  margin-bottom: 9px;
}
.demo-creds {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  color: #78350f;
  margin-bottom: 8px;
  font-size: 13px;
}
.demo-creds span {
  font-weight: 600;
  margin-left: 4px;
}
.demo-creds span:first-child { margin-left: 0; }
.demo-creds code {
  background: #fef3c7;
  padding: 2px 8px;
  border-radius: 5px;
  font-family: 'SF Mono', monospace;
  font-weight: 600;
}
.demo-warn {
  color: #b45309;
  font-size: 12px;
  line-height: 1.5;
}
</style>
